// Package main is the application entry point for "按时吃药" backend service.
// It wires up the full dependency chain: config → logger → database → repos →
// services → handlers → middleware → routes → scheduled tasks → HTTP server.
//
// Architecture: Modular Monolith with 3-layer separation (Handler → Service → Repository).
// See docs/architecture.md for the full system design.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/medical-monitor/config"
	"github.com/medical-monitor/internal/database"
	"github.com/medical-monitor/internal/external"
	"github.com/medical-monitor/internal/handler"
	"github.com/medical-monitor/internal/logger"
	"github.com/medical-monitor/internal/middleware"
	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/jwt"
	"github.com/medical-monitor/internal/repository"
	"github.com/medical-monitor/internal/scheduler"
	"github.com/medical-monitor/internal/service"
)

// main is the single entry point for the entire backend service.
// Startup order matters: config must load first, then logger, then database,
// because every subsequent component depends on the database connection.
func main() {
	// ---- 1. 配置加载（viper，支持环境变量覆盖）----
	// Must be first — all other components read from cfg.
	cfg, err := config.Load("")

	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// ---- 2. 日志初始化（zap 结构化日志）----
	// Production: JSON format to stdout; Debug: console format.
	logger := logger.New(cfg)
	defer logger.Sync()
	//记录「后端服务启动了」，并附带上版本号和运行模式。
	logger.Info("starting medication reminder backend",
		zap.String("version", "1.0.0"),
		zap.String("mode", cfg.Server.Mode),
	)

	// ---- 3. 数据库连接（MySQL 8.0 via GORM）----
	// Connection pool settings from config: MaxIdleConns / MaxOpenConns.
	db, err := database.New(cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	logger.Info("database connected successfully")

	// ---- 3.5 Redis 连接（可选，Token 缓存加速）----
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		logger.Warn("redis not available, using mysql for tokens", zap.Error(err))
		rdb = nil
	} else {
		logger.Info("redis connected")
	}

	// ---- 4. JWT 管理器（Access Token 7天 + Refresh Token 30天）----
	// TTLs are configurable via environment variables.
	jwtManager := jwt.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.GetAccessTokenTTL(),
		cfg.JWT.GetRefreshTokenTTL(),
	)

	// ---- 5. 外部服务（根据配置切换 Mock / 真实实现）----
	var wechatSvc external.WechatService
	if cfg.Wechat.Mode == "real" {
		if cfg.Wechat.AppID == "" || cfg.Wechat.AppSecret == "" {
			logger.Fatal("wechat.mode=real 但 app_id/app_secret 未配置")
		}
		wechatSvc = external.NewRealWechatService(cfg.Wechat.AppID, cfg.Wechat.AppSecret)
		logger.Info("wechat service: real")
	} else {
		wechatSvc = external.NewMockWechatService()
		logger.Info("wechat service: mock")
	}
	smsSvc := external.NewMockSMSService() // 腾讯云短信（P1 兜底）

	// ---- 6. Repository 数据访问层（数据访问，每个 repo 封装一张或一组表）----
	userRepo := repository.NewUserRepository(db)
	familyRepo := repository.NewFamilyRepository(db)
	medicineRepo := repository.NewMedicineRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)
	reminderRepo := repository.NewReminderRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db, rdb) // MySQL持久化 + Redis缓存

	// ---- 7. Service 层（业务逻辑，纯 Go，不依赖 HTTP）----
	// AuthService 认证服务，负责登录/注册/Token刷新/个人资料。
	authSvc := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, wechatSvc, familyRepo)

	//FamilyService 家庭组业务逻辑：创建、加入、退出、成员角色管理、邀请码刷新。
	familySvc := service.NewFamilyService(familyRepo, userRepo, auditRepo)

	// MedicineService 负责药品管理的业务逻辑，包括：
	//   - 药品的增删改查（CRUD）
	//   - 药品库存（余量）的初始化与管理
	//   - 药品与用药计划的关联校验
	medicineSvc := service.NewMedicineService(medicineRepo, auditRepo)

	// 用药计划服务（ScheduleService）—— 处理家庭 + 个人两套用药计划的业务逻辑，由计划生成提醒。
	scheduleSvc := service.NewScheduleService(scheduleRepo, medicineRepo, familyRepo, reminderRepo, auditRepo)

	// ReminderService 负责用药提醒的确认、跳过、未服用标记和撤销等交互操作。
	reminderSvc := service.NewReminderService(reminderRepo, medicineRepo, auditRepo)

	// DashboardService 负责看板数据的聚合计算，为前端提供今日服药状态和家庭概览。
	dashboardSvc := service.NewDashboardService(reminderRepo, familyRepo, userRepo)

	// NotificationService 负责多通道通知的分发。
	notifSvc := service.NewNotificationService(wechatSvc, smsSvc, logger)

	// ---- 8. 中间件（按 Gin 中间件链顺序注册）----
	authMW := middleware.NewAuthMiddleware(jwtManager)

	// FamilyMiddleware: 校验当前用户是否属于目标家庭组。
	// 通过闭包注入 IsMember 查询，避免 middleware 层直接依赖 repository。
	familyMW := middleware.NewFamilyMiddleware(func(userID, familyID uint64) (string, bool) {
		isMember, role, err := familyRepo.IsMember(context.TODO(), familyID, userID)
		if err != nil {
			return "", false
		}
		return role, isMember
	})

	// RoleMiddleware: 禁止老人（elder 角色）执行管理操作（药品增删改等）。
	nonElderRoleMW := middleware.NewRoleMiddleware(model.RoleOwner, model.RoleAdmin, model.RoleMember)

	// AdminMiddleware: 仅管理员（users.role == 3）可访问后台接口。
	adminMW := middleware.NewAdminMiddleware(func(userID uint64) bool {
		user, err := userRepo.FindByID(context.TODO(), userID)
		if err != nil {
			return false
		}
		return user.Role == model.UserRoleAdmin
	})

	// RateLimitMiddleware: 双层限流（每 IP 50 req/s + 全局 3000 req/s）。
	rateLimitMW := middleware.NewRateLimitMiddleware(50, 100, 3000)

	// RecoveryMiddleware: 捕获 panic，返回 500 而非让进程崩溃。
	recoveryMW := middleware.NewRecoveryMiddleware(logger)

	// LoggerMiddleware: 记录每个请求的方法、路径、耗时、状态码。
	loggerMW := middleware.NewLoggerMiddleware(logger)

	// ---- 9. Handler 层（HTTP 参数校验 → Service 调用 → 统一响应）----
	// AuthHandler 认证：微信登录、密码登录、注册、Token 刷新。
	authH := handler.NewAuthHandler(authSvc)
	// UserHandler 用户资料：查看/更新个人资料。
	userH := handler.NewUserHandler(authSvc)
	// FamilyHandler 家庭组：创建、加入、退出、成员角色管理、邀请码刷新。
	familyH := handler.NewFamilyHandler(familySvc)
	// MedicineHandler 药品管理：家庭 + 个人药品的增删改查。
	medicineH := handler.NewMedicineHandler(medicineSvc)
	// ScheduleHandler 用药计划：家庭 + 个人计划的增删改查、日历。
	scheduleH := handler.NewScheduleHandler(scheduleSvc)
	// ReminderHandler 提醒交互：确认服药、跳过、标记未服用、撤销。
	reminderH := handler.NewReminderHandler(reminderSvc)
	// DashboardHandler 看板：今日服药状态、家庭概览聚合。
	dashboardH := handler.NewDashboardHandler(dashboardSvc)
	// StockHandler 库存：家庭 + 个人药品余量查询、手动校准。
	stockH := handler.NewStockHandler(medicineRepo, auditRepo)
	// AlertHandler 告警：告警记录列表查询。
	alertH := handler.NewAlertHandler(reminderRepo)
	// AdminHandler 后台：管理员统计、用户/家庭/提醒管理。
	adminH := handler.NewAdminHandler(userRepo, familyRepo, reminderRepo)

	// ---- 10. Gin 引擎设置 ----
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// 信任本机 Nginx 代理，从 X-Forwarded-For 获取用户真实 IP（限流依赖）
	_ = r.SetTrustedProxies([]string{"127.0.0.1"})

	// ---- 静态文件服务（Web 前端页面）----
	r.Static("/static", "./web/static")

	// ---- 根路由重定向到管理员登录页 ----
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/static/login.html")
	})

	// ---- 全局中间件（每个请求都会经过）----
	r.Use(recoveryMW.Handler())        // 1. 最外层：panic 恢复
	r.Use(loggerMW.Handler())          // 2. 请求日志
	r.Use(middleware.CORSMiddleware()) // 3. 跨域支持
	r.Use(rateLimitMW.Handler())        // 4. API 限流（双层：全局 + 每 IP）

	// ---- 健康检查（无需鉴权）----
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "朝夕小叮"})
	})

	// ================================================================
	// API v1 路由注册（28 个端点，严格匹配 docs/api-design.md）
	// ================================================================
	v1 := r.Group("/api/v1")

	// --- Auth 模块（无需鉴权）---
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authH.Login)                  // 微信 code 登录 → JWT
		auth.POST("/login-password", authH.LoginPassword) // Web 开发阶段密码登录
		auth.POST("/register", authH.Register)            // Web 开发阶段手机号注册
		auth.POST("/refresh", authH.Refresh)              // Refresh Token 换新 Token 对
	}

	// --- User 模块（需 JWT 鉴权）---
	user := v1.Group("/user")
	user.Use(authMW.Handler())
	{
		user.GET("/profile", userH.GetProfile)      // 查看个人资料
		user.PUT("/profile", userH.UpdateProfile)   // 修改昵称/头像
		user.PUT("/phone", userH.BindPhone)         // 绑定手机号
		user.PUT("/settings", userH.UpdateSettings) // 提醒设置
	}

	// --- Family + Medicine + Schedule + Stock 模块（需 JWT + 家庭组成员鉴权）---
	families := v1.Group("/families")
	families.Use(authMW.Handler())
	{
		families.POST("", familyH.Create)          // 创建家庭组（生成6位邀请码）
		families.GET("", familyH.List)             // 我的家庭组列表
		families.POST("/join", familyH.JoinByCode) // 通过邀请码加入（无需已是成员）

		// 需要家庭组成员身份的路由
		familyWithID := families.Group("/:fid")
		familyWithID.Use(familyMW.Handler())
		{
			familyWithID.GET("", familyH.Detail)                                                    // 家庭组详情
			familyWithID.POST("/join", familyH.Join)                                                // 通过邀请码加入
			familyWithID.POST("/refresh-invite", familyH.RefreshInvite)                             // 刷新邀请码
			familyWithID.PUT("/members/:uid/role", nonElderRoleMW.Handler(), familyH.SetMemberRole) // 修改成员角色
			familyWithID.DELETE("/members/:uid", nonElderRoleMW.Handler(), familyH.RemoveMember)    // 移除成员
			familyWithID.DELETE("/leave", familyH.Leave)                                            // 退出家庭组
		}

		// Medicine 路由（需家庭组成员 + 非老人角色才能增删改）
		medicines := families.Group("/:fid/medicines")
		medicines.Use(familyMW.Handler())
		{
			medicines.GET("", medicineH.List)                                    // 药品列表（可搜索）
			medicines.GET("/:id", medicineH.Detail)                              // 药品详情
			medicines.POST("", nonElderRoleMW.Handler(), medicineH.Create)       // 添加药品（子女）
			medicines.PUT("/:id", nonElderRoleMW.Handler(), medicineH.Update)    // 编辑药品
			medicines.DELETE("/:id", nonElderRoleMW.Handler(), medicineH.Delete) // 删除药品（关联计划时拒绝）
		}

		// Schedule 路由（需家庭组成员 + 非老人角色）
		schedules := families.Group("/:fid/schedules")
		schedules.Use(familyMW.Handler())
		{
			schedules.GET("/calendar", scheduleH.Calendar)                       // 日历视图
			schedules.GET("", scheduleH.List)                                    // 用药计划列表
			schedules.POST("", nonElderRoleMW.Handler(), scheduleH.Create)       // 创建用药计划（时间槽+药品明细）
			schedules.PUT("/:id", nonElderRoleMW.Handler(), scheduleH.Update)    // 编辑用药计划
			schedules.DELETE("/:id", nonElderRoleMW.Handler(), scheduleH.Delete) // 删除用药计划
		}

		// Stock 路由（需家庭组成员）
		stocks := families.Group("/:fid/stocks")
		stocks.Use(familyMW.Handler())
		{
			stocks.GET("", stockH.List)                                 // 药品余量列表
			stocks.PUT("/:id", nonElderRoleMW.Handler(), stockH.Update) // 手动校准余量
		}
	}

	// --- Personal Medicine 模块（需 JWT）---
	personal := v1.Group("/personal")
	personal.Use(authMW.Handler())
	{
		personal.GET("/medicines", medicineH.ListPersonal)
		personal.GET("/medicines/:id", medicineH.DetailPersonal)
		personal.POST("/medicines", medicineH.CreatePersonal)
		personal.PUT("/medicines/:id", medicineH.UpdatePersonal)
		personal.PUT("/medicines/:id/stock", stockH.UpdatePersonalStock) // 手动校准个人药品余量
		personal.DELETE("/medicines/:id", medicineH.DeletePersonal)
		personal.GET("/schedules/calendar", scheduleH.CalendarPersonal)
		personal.GET("/schedules", scheduleH.ListPersonal)
		personal.POST("/schedules", scheduleH.CreatePersonal)
		personal.PUT("/schedules/:id", scheduleH.UpdatePersonal)
		personal.DELETE("/schedules/:id", scheduleH.DeletePersonal)
	}

	// --- Admin 模块（需 JWT + 管理员角色）---
	admin := v1.Group("/admin")
	admin.Use(authMW.Handler(), adminMW.Handler())
	{
		admin.GET("/stats", adminH.Stats)
		admin.GET("/users", adminH.Users)
	}

	// --- Reminder 模块（需 JWT 鉴权，核心交互）---
	reminders := v1.Group("/reminders")
	reminders.Use(authMW.Handler())
	{
		reminders.GET("/today", reminderH.GetToday)       // 今日用药提醒列表（老人端首页）
		reminders.POST("/:id/confirm", reminderH.Confirm) // 确认已服用（含余量扣减）
		reminders.POST("/:id/miss", reminderH.MarkMiss)   // 标记未服用（含原因）
		reminders.POST("/:id/skip", reminderH.Skip)       // 跳过（如身体不适）
		reminders.POST("/:id/revoke", reminderH.Revoke)   // 撤销确认（5分钟窗口内）
	}

	// --- Dashboard 模块（需 JWT 鉴权）---
	dashboard := v1.Group("/dashboard")
	dashboard.Use(authMW.Handler())
	{
		dashboard.GET("/today", dashboardH.GetToday)       // 今日用药看板（子女端首页）
		dashboard.GET("/overview", dashboardH.GetOverview) // 家庭成员总体服药概览
	}

	// --- Alert 模块（需 JWT 鉴权）---
	alerts := v1.Group("/alerts")
	alerts.Use(authMW.Handler())
	{
		alerts.GET("", alertH.List) // 告警列表（超时未服药的提醒）
	}

	// ================================================================
	// 定时任务（robfig/cron v3，秒级精度）
	// 单实例部署无需分布式锁；多实例扩展需加 Redis SETNX 锁
	// ================================================================

	// 提醒调度器：每 60 秒扫描一次用药计划，匹配当前时间窗口（±1分钟）生成今日提醒并推送
	reminderScheduler := scheduler.NewReminderScheduler(
		scheduleRepo, reminderRepo, medicineRepo, notifSvc, familyRepo, userRepo, logger,
	)

	// 告警调度器：每 60 秒检测超时提醒，按 30/60/90 分钟三级升级（子女通知 → 短信 → 语音）
	alertScheduler := scheduler.NewAlertScheduler(
		reminderRepo, familyRepo, userRepo, notifSvc, logger,
	)

	cronScheduler := cron.New(cron.WithSeconds())

	_, err = cronScheduler.AddFunc("@every 60s", reminderScheduler.Run)
	if err != nil {
		logger.Fatal("failed to register reminder scheduler", zap.Error(err))
	}

	_, err = cronScheduler.AddFunc("@every 60s", alertScheduler.Run)
	if err != nil {
		logger.Fatal("failed to register alert scheduler", zap.Error(err))
	}

	// 过期计划自动停用：每天凌晨 2 点检查 end_date < 今天的计划，设为停用
	cronScheduler.AddFunc("0 2 * * *", func() {
		if n, err := scheduleRepo.DeactivateExpired(context.Background()); err == nil && n > 0 {
			logger.Info("deactivated expired schedules", zap.Int64("count", n))
		}
	})

	// Token 过期清理：每 6 小时清理一次 MySQL 中过期的 Refresh Token
	cronScheduler.AddFunc("0 */6 * * *", func() {
		if n, err := refreshTokenRepo.CleanExpired(context.Background()); err == nil && n > 0 {
			logger.Info("cleaned expired tokens", zap.Int64("count", n))
		}
	})

	cronScheduler.Start()
	logger.Info("scheduled tasks started (reminder + alert + token cleanup)")

	// ---- 启动 HTTP Server ----
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	logger.Info("server starting", zap.String("addr", addr))

	// 优雅关闭：Ctrl+C → 停止 cron → 立即退出
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("shutting down...")
		cronScheduler.Stop()
		os.Exit(0)
	}()

	if err := r.Run(addr); err != nil {
		logger.Fatal("failed to start server", zap.Error(err))
	}
}
