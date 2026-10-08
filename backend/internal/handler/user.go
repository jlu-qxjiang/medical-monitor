package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/middleware"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/service"
)

// UserHandler 处理用户个人资料相关的 HTTP 端点：
//   - GET  /api/v1/user/profile —— 获取用户资料
//   - PUT  /api/v1/user/profile —— 更新用户资料
//   - PUT  /api/v1/user/phone   —— 绑定手机号
//
// 所有端点都需要 JWT 认证，userID 从中间件注入的上下文中获取。
type UserHandler struct {
	authService *service.AuthService // 用户资料相关的业务逻辑（由 AuthService 承担）
}

// NewUserHandler 创建 UserHandler 实例。
func NewUserHandler(authService *service.AuthService) *UserHandler {
	return &UserHandler{authService: authService}
}

// GetProfile 处理 GET /api/v1/user/profile —— 获取当前用户的个人资料。
// 返回昵称、头像、脱敏手机号和注册时间。
// 手机号仅返回后4位，完整号码通过加密存储保护隐私。
func (h *UserHandler) GetProfile(c *gin.Context) {
	// 从 Gin 上下文中获取由 JWT 中间件注入的 userID
	userID := middleware.MustGetUserID(c)

	profile, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, profile)
}

// UpdateProfile 处理 PUT /api/v1/user/profile —— 更新用户资料。
// 支持增量更新：仅更新传入的非空字段。
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.MustGetUserID(c)

	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "参数错误")
		return
	}

	if err := h.authService.UpdateProfile(c.Request.Context(), userID, req.Nickname, req.AvatarURL); err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.SuccessMessage(c, "更新成功")
}

// BindPhone 处理 PUT /api/v1/user/phone —— 绑定手机号。
// 接收微信 getPhoneNumber 组件返回的加密数据，
// 在生产环境中需要配合 session_key 解密后存储。
//
// 当前 MVP 阶段为简化实现，将加密数据直接占位存储。
// TODO: 接入微信数据解密 API，解密后存储真实手机号。
func (h *UserHandler) BindPhone(c *gin.Context) {
	userID := middleware.MustGetUserID(c)

	var req service.BindPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "参数错误")
		return
	}

	if err := h.authService.BindPhone(c.Request.Context(), userID, req.EncryptedData, req.IV); err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.SuccessMessage(c, "手机号绑定成功")
}

// UpdateSettings 处理 PUT /api/v1/user/settings
func (h *UserHandler) UpdateSettings(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	var req struct {
		NotifyReminder *bool `json:"notify_reminder"`
		NotifyStock    *bool `json:"notify_stock"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "参数错误")
		return
	}
	if err := h.authService.UpdateSettings(c.Request.Context(), userID, req.NotifyReminder, req.NotifyStock); err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}
	response.SuccessMessage(c, "已保存")
}
