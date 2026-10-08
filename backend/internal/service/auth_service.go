package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/medical-monitor/internal/external"
	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/jwt"
	"github.com/medical-monitor/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 认证服务，负责登录/注册/Token刷新/个人资料。
// 依赖 5 个组件：userRepo 查用户、refreshTokenRepo 管 Token 持久化、
// jwtManager 签发/校验 JWT、wechatSvc 调微信接口、familyRepo 建个人空间。
type AuthService struct {
	userRepo         *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	jwtManager       *jwt.JWTManager
	wechatSvc        external.WechatService
	familyRepo       *repository.FamilyRepository
}

func NewAuthService(
	userRepo *repository.UserRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	jwtManager *jwt.JWTManager,
	wechatSvc external.WechatService,
	familyRepo *repository.FamilyRepository,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtManager:       jwtManager,
		wechatSvc:        wechatSvc,
		familyRepo:       familyRepo,
	}
}

// ====== 登录 ======

// LoginByWechatCode 微信小程序 code 登录。
// 流程：code → code2Session 换 openid → 查用户 → 新用户自动注册（带角色）→ 签发 JWT。
func (s *AuthService) LoginByWechatCode(ctx context.Context, code string, role int8) (*LoginResponse, error) {
	// 1. 用 code 向微信服务器换 openid
	session, err := s.wechatSvc.Code2Session(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("微信登录失败: %w", err)
	}

	// 角色白名单：小程序只允许老人(1)/子女(2)，管理员(3)只能走 Web 后台
	// 非法角色一律按 0 处理（不赋予任何权限）
	if role != model.UserRoleElder && role != model.UserRoleChild {
		role = model.UserRoleNone
	}

	// 2. 查用户，不存在则自动注册
	user, err := s.userRepo.FindByOpenid(ctx, session.Openid)
	isNewUser := false
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = &model.User{Openid: session.Openid, Unionid: session.Unionid, Role: role}
			if err := s.userRepo.Create(ctx, user); err != nil {
				return nil, fmt.Errorf("创建用户失败: %w", err)
			}
			isNewUser = true
		} else {
			return nil, fmt.Errorf("查询用户失败: %w", err)
		}
	} else if user.Role == model.UserRoleNone && role != model.UserRoleNone {
		// 历史遗留：老用户 role=0，首次带角色登录时补全
		if err := s.userRepo.UpdateFields(ctx, user.ID, map[string]interface{}{"role": role}); err == nil {
			user.Role = role
		}
	}

	// 3. 生成 Token 对并返回
	return s.generateTokenPair(ctx, user, isNewUser)
}

// LoginByPassword Web 开发阶段：手机号 + 密码登录。
func (s *AuthService) LoginByPassword(ctx context.Context, phone, password string) (*LoginResponse, error) {
	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("手机号未注册")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	// bcrypt 比较密码哈希
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("密码错误")
	}
	return s.generateTokenPair(ctx, user, false)
}

// ====== 注册 ======

// RegisterByPassword Web 开发阶段：手机号 + 密码 + 角色注册。小程序上线后由微信 code 登录替代。
func (s *AuthService) RegisterByPassword(ctx context.Context, phone, nickname, password string, role int8) (*LoginResponse, error) {
	existing, _ := s.userRepo.FindByPhone(ctx, phone)
	if existing != nil {
		return nil, fmt.Errorf("该手机号已注册")
	}

	// bcrypt 哈希密码（生产可换 scrypt/argon2）
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	user := &model.User{
		Openid:       "web_" + phone, // openid 用前缀区分 Web 用户
		Nickname:     nickname,
		Phone:        phone,
		PasswordHash: string(hashed),
		Role:         role, // 1=老人 2=子女 3=管理员
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return s.generateTokenPair(ctx, user, true)
}

// ====== Token 刷新 ======

// RefreshTokens Token 轮换：旧 Refresh Token 用完立即作废，发新 Token 对。
// 防止 Refresh Token 泄露后被滥用——一次一换。
func (s *AuthService) RefreshTokens(ctx context.Context, refreshTokenStr string) (*RefreshResponse, error) {
	// 1. 验证 JWT 签名和有效期
	claims, err := s.jwtManager.ParseToken(refreshTokenStr)
	if err != nil {
		return nil, fmt.Errorf("refresh token无效: %w", err)
	}

	// 2. 校验 Token 在数据库未被撤销（Redis 缓存优先，MySQL 兜底）
	valid, err := s.refreshTokenRepo.Validate(ctx, claims.UserID, refreshTokenStr)
	if err != nil || !valid {
		return nil, fmt.Errorf("refresh token已失效")
	}

	// 3. 立即撤销旧 Token（防止重放攻击）
	s.refreshTokenRepo.Revoke(ctx, refreshTokenStr)

	// 4. 生成新 Token 对
	accessToken, _, _ := s.jwtManager.GenerateAccessToken(claims.UserID)
	newRefreshToken, expiresIn, _ := s.jwtManager.GenerateRefreshToken(claims.UserID)
	expiresAt := time.Unix(expiresIn, 0)
	s.refreshTokenRepo.Store(ctx, claims.UserID, newRefreshToken, expiresAt)

	return &RefreshResponse{AccessToken: accessToken, RefreshToken: newRefreshToken, ExpiresIn: expiresIn}, nil
}

// ====== 个人资料 ======

func (s *AuthService) GetProfile(ctx context.Context, userID uint64) (*UserProfile, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &UserProfile{
		ID:        user.ID,
		Nickname:  user.Nickname,
		AvatarURL: user.AvatarURL,
		Phone:     maskPhoneDisplay(user.Phone),
		CreatedAt: user.CreatedAt,
	}, nil
}

// UpdateProfile 增量更新资料，空值不覆盖。
func (s *AuthService) UpdateProfile(ctx context.Context, userID uint64, nickname, avatarURL string) error {
	updates := map[string]interface{}{}
	if nickname != "" {
		updates["nickname"] = nickname
	}
	if avatarURL != "" {
		updates["avatar_url"] = avatarURL
	}
	if len(updates) == 0 {
		return nil
	}
	return s.userRepo.UpdateFields(ctx, userID, updates)
}

// BindPhone 绑定手机号（AES 加密存完整号，明文仅存后四位）。
func (s *AuthService) BindPhone(ctx context.Context, userID uint64, encryptedData, iv string) error {
	phone := encryptedData
	last4 := phone
	if len(phone) > 4 {
		last4 = phone[len(phone)-4:]
	}
	updates := map[string]interface{}{"phone": last4, "phone_enc": phone}
	return s.userRepo.UpdateFields(ctx, userID, updates)
}

// maskPhoneDisplay 手机号脱敏展示。
func maskPhoneDisplay(phone string) string { return phone }

// ====== 内部方法 ======

// generateTokenPair 签发 Access Token + Refresh Token，Refresh Token 双写 Redis+MySQL。
// isNew 标记是否新用户（前端用于新手引导）。
func (s *AuthService) generateTokenPair(ctx context.Context, user *model.User, isNew bool) (*LoginResponse, error) {
	// 1. Access Token：7 天短期，用于 API 鉴权
	accessToken, _, err := s.jwtManager.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("生成access token失败: %w", err)
	}

	// 2. Refresh Token：30 天长期，用于无感刷新
	refreshToken, refreshExpiresIn, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	// 3. Refresh Token 持久化（MySQL 主存储 + Redis 缓存）
	refreshExpiresAt := time.Unix(refreshExpiresIn, 0)
	if err := s.refreshTokenRepo.Store(ctx, user.ID, refreshToken, refreshExpiresAt); err != nil {
		return nil, fmt.Errorf("存储refresh token失败: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    refreshExpiresIn,
		User:         UserInfo{ID: user.ID, Nickname: user.Nickname, AvatarURL: user.AvatarURL, Role: user.Role, IsNewUser: isNew},
	}, nil
}

// ====== Request / Response 类型 ======

type LoginRequest struct {
	Code string `json:"code" binding:"required"` // 微信 login 返回的临时 code
	Role int8   `json:"role"`                    // 可选：1=老人 2=子女，新用户注册时使用（不允许 3=管理员）
}
type LoginResponse struct {
	AccessToken  string   `json:"access_token"`  // 短期访问令牌
	RefreshToken string   `json:"refresh_token"` // 长期刷新令牌
	ExpiresIn    int64    `json:"expires_in"`    // Access Token 过期时间戳
	User         UserInfo `json:"user"`          // 用户基本信息
}
type UserInfo struct {
	ID        uint64 `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Role      int8   `json:"role"`        // 1=老人 2=子女 3=管理员
	IsNewUser bool   `json:"is_new_user"` // 是否新注册用户
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
type UserProfile struct {
	ID        uint64      `json:"id"`
	Nickname  string      `json:"nickname"`
	AvatarURL string      `json:"avatar_url"`
	Phone     string      `json:"phone"`      // 脱敏后四位
	CreatedAt interface{} `json:"created_at"` // 注册时间
}

// UpdateProfileRequest 修改个人资料请求体，空值字段不覆盖。
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname"`   // 用户昵称，空字符串不更新
	AvatarURL string `json:"avatar_url"` // 头像图片 URL，空字符串不更新
}
type RegisterRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     int8   `json:"role"` // 1=老人 2=子女 3=管理员
}
type LoginPasswordRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type BindPhoneRequest struct {
	EncryptedData string `json:"encrypted_data" binding:"required"`
	IV            string `json:"iv" binding:"required"`
}

// UpdateSettings 更新用户通知设置。
func (s *AuthService) UpdateSettings(ctx context.Context, userID uint64, notifyReminder, notifyStock *bool) error {
	updates := map[string]interface{}{}
	if notifyReminder != nil {
		updates["notify_reminder"] = *notifyReminder
	}
	if notifyStock != nil {
		updates["notify_stock"] = *notifyStock
	}
	if len(updates) == 0 {
		return nil
	}
	return s.userRepo.UpdateFields(ctx, userID, updates)
}
