// Package middleware 提供 Gin 框架的中间件组件，包括认证、鉴权、日志、限流和异常恢复。
// 中间件按固定顺序在路由中注册：Recovery → Logger → RateLimit → CORS → Auth → Family → Role。
package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/jwt"
	"github.com/medical-monitor/internal/pkg/response"
)

// AuthMiddleware 负责 JWT 认证中间件，从 HTTP Authorization 头中提取并校验 Access Token。
//
// 认证流程：
//  1. 检查请求路径是否在跳过列表中（如 /auth/login、/auth/refresh、/health）
//  2. 从 Authorization 头提取 Bearer Token
//  3. 解析并验证 JWT 签名和有效期
//  4. 校验 Token 类型为 "access"（拒绝使用 Refresh Token 访问 API）
//  5. 将解析出的 user_id 注入 Gin Context，供下游使用
//
// 使用方式：
//
//	router.Use(authMiddleware.Handler())
type AuthMiddleware struct {
	jwtManager *jwt.JWTManager     // JWT 签发和校验管理器
	skipPaths  map[string]bool     // 不需要认证的路径白名单
}

// NewAuthMiddleware 创建 AuthMiddleware 实例。
// 默认白名单包含登录、刷新 Token 和健康检查接口。
func NewAuthMiddleware(jwtManager *jwt.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
		skipPaths: map[string]bool{
			"/api/v1/auth/login":   true, // 登录接口无需认证
			"/api/v1/auth/refresh": true, // 刷新 Token 接口无需认证（自身持有 Refresh Token）
			"/health":              true, // K8s 健康检查探针
		},
	}
}

// Handler 返回 Gin 中间件处理函数。
// 认证失败时直接返回 401 并中止请求链。
func (a *AuthMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查请求路径是否在白名单中，是则跳过认证
		if a.skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 从 Authorization 头提取 Bearer Token
		// 格式：Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, errcode.TokenInvalid, "缺少Authorization头")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, errcode.TokenInvalid, "Authorization格式错误，需要 Bearer <token>")
			c.Abort()
			return
		}

		tokenStr := parts[1]

		// 解析并验证 JWT Token
		claims, err := a.jwtManager.ParseToken(tokenStr)
		if err != nil {
			response.Error(c, errcode.TokenInvalid, err.Error())
			c.Abort()
			return
		}

		// 安全策略：仅 Access Token 可访问 API 端点
		// Refresh Token 只能用于 /auth/refresh 接口
		if claims.Subject != "access" {
			response.Error(c, errcode.TokenInvalid, "请使用access token访问")
			c.Abort()
			return
		}

		// 将 user_id 注入 Context，供后续 handler/service 使用
		// 使用 c.Set 而非 context.WithValue 是因为 Gin Context 是请求级别的
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

// GetUserID 从 Gin Context 中安全地提取 user_id。
// 如果 user_id 不存在则返回 0（通常因为 AuthMiddleware 未执行或未正确设置）。
func GetUserID(c *gin.Context) uint64 {
	if uid, exists := c.Get("user_id"); exists {
		return uid.(uint64)
	}
	return 0
}

// MustGetUserID 从 Gin Context 中提取 user_id，若不存在则 panic。
// 仅在能保证 AuthMiddleware 已执行并设置了 user_id 的 handler 中使用。
// panic 会被 RecoveryMiddleware 捕获并转为 500 响应。
func MustGetUserID(c *gin.Context) uint64 {
	return c.MustGet("user_id").(uint64)
}

// FamilyMiddleware 负责家庭组成员身份校验中间件。
// 从 URL 路径参数中提取家庭组 ID（优先 :fid，其次 :id），
// 调用注入的 CheckFunc 校验当前用户是否属于该家庭组。
//
// CheckFunc 采用依赖注入模式，避免中间件层直接依赖 Repository 层，
// 从而解决循环依赖问题（Repository 可能引用 model 中定义的类型）。
//
// 使用方式：
//
//	familyMiddleware := NewFamilyMiddleware(familyRepo.CheckMembership)
//	router.Use(familyMiddleware.Handler())
type FamilyMiddleware struct {
	// CheckFunc 被调用来验证家庭组成员身份。
	// 参数：(userID, familyID)
	// 返回值：(member_role, is_member_bool)
	// 通过依赖注入避免中间件层与 Repository 层的循环依赖。
	CheckFunc func(userID, familyID uint64) (string, bool)
}

// NewFamilyMiddleware 创建 FamilyMiddleware 实例。
// checkFunc 通常由 FamilyRepository.CheckMembership 方法实现。
func NewFamilyMiddleware(checkFunc func(userID, familyID uint64) (string, bool)) *FamilyMiddleware {
	return &FamilyMiddleware{CheckFunc: checkFunc}
}

// Handler 返回 Gin 中间件处理函数。
// 从 URL 的 :fid 或 :id 路径参数中提取 familyID，
// 调用 CheckFunc 验证当前 userID 是否为该家庭组成员。
// 验证通过后将 member_role 注入 Context。
func (f *FamilyMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := MustGetUserID(c)

		// 提取家庭组 ID：优先使用 :fid 参数（family 作用域路由），其次 :id
		familyIDStr := c.Param("fid")
		if familyIDStr == "" {
			familyIDStr = c.Param("id")
		}
		// 如果路径中没有 family 参数，直接放行（非 family 作用域路由）
		if familyIDStr == "" {
			c.Next()
			return
		}

		// 解析 familyID
		familyID, err := strconv.ParseUint(familyIDStr, 10, 64)
		if err != nil {
			response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
			c.Abort()
			return
		}

		// 校验家庭成员身份
		if f.CheckFunc != nil {
			role, isMember := f.CheckFunc(userID, familyID)
			if !isMember {
				response.Error(c, errcode.NotFamilyMember, "")
				c.Abort()
				return
			}
			// 将角色信息注入 Context，供后续 RoleMiddleware 使用
			c.Set("member_role", role)
		}

		c.Next()
	}
}

// GetMemberRole 从 Gin Context 中提取当前用户在家庭组中的角色。
// 返回空字符串表示角色未设置（FamilyMiddleware 未执行或请求不在 family 作用域内）。
func GetMemberRole(c *gin.Context) string {
	if role, exists := c.Get("member_role"); exists {
		return role.(string)
	}
	return ""
}

// RoleMiddleware 负责基于角色的访问控制（RBAC）。
// 在 FamilyMiddleware 之后执行，检查当前用户的角色是否在允许列表中。
//
// 典型用法：限制管理操作仅 owner 和 admin 可执行：
//
//	roleMiddleware := NewRoleMiddleware(model.RoleOwner, model.RoleAdmin)
//	router.PUT("/families/:fid/members/:uid/role", familyMiddleware.Handler(), roleMiddleware.Handler(), handler.SetMemberRole)
type RoleMiddleware struct {
	// AllowedRoles 允许访问的角色列表，如 ["owner", "admin"]
	AllowedRoles []string
}

// NewRoleMiddleware 创建 RoleMiddleware 实例，指定允许的角色列表。
// 如果 allowedRoles 为空，则所有角色均可通过（仅做标记不做限制）。
func NewRoleMiddleware(allowedRoles ...string) *RoleMiddleware {
	return &RoleMiddleware{AllowedRoles: allowedRoles}
}

// Handler 返回 Gin 中间件处理函数。
// 检查当前用户的 member_role 是否在 AllowedRoles 列表中。
// 如果角色未设置（非 family 作用域路由），直接放行。
// 如果角色不在允许列表中，返回 403 Forbidden。
func (r *RoleMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetMemberRole(c)
		// 没有角色信息表示不在 family 作用域，放行
		if role == "" {
			c.Next()
			return
		}

		// 构建允许角色集合用于快速查找
		allowed := make(map[string]bool)
		for _, r := range r.AllowedRoles {
			allowed[r] = true
		}
		// 如果有白名单且当前角色不在白名单中，拒绝访问
		if len(allowed) > 0 && !allowed[role] {
			response.Error(c, errcode.Forbidden, "老人角色无权执行此操作")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminMiddleware 校验当前用户是否为管理员（users.role == 3）。
// 用于保护 /admin 管理后台接口，防止普通用户越权访问。
// CheckFunc 通过依赖注入传入，避免中间件层直接依赖 Repository。
type AdminMiddleware struct {
	CheckFunc func(userID uint64) bool
}

// NewAdminMiddleware 创建 AdminMiddleware 实例。
func NewAdminMiddleware(checkFunc func(userID uint64) bool) *AdminMiddleware {
	return &AdminMiddleware{CheckFunc: checkFunc}
}

// Handler 返回 Gin 中间件处理函数。
// 非管理员返回 403 Forbidden。
func (a *AdminMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := MustGetUserID(c)
		if a.CheckFunc != nil && !a.CheckFunc(userID) {
			response.Error(c, errcode.Forbidden, "无管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

// CORSMiddleware 处理跨域资源共享（CORS）相关的 HTTP 头。
// 允许来自任何源的跨域请求（开发阶段配置，生产环境应限制为小程序域名）。
// 对 OPTIONS 预检请求直接返回 204 No Content。
//
// CORS 头说明：
//   - Access-Control-Allow-Origin: 允许的来源域名（* 表示所有）
//   - Access-Control-Allow-Methods: 允许的 HTTP 方法
//   - Access-Control-Allow-Headers: 允许的请求头
//   - Access-Control-Max-Age: 预检请求缓存时间（秒）
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400") // 24小时缓存

		// OPTIONS 预检请求直接返回，不进入后续处理链
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
