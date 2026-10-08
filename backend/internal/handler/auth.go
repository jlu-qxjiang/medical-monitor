// Package handler 是 HTTP 请求处理层（Controller 层），负责：
//   - 解析 HTTP 请求参数（路径参数、查询参数、请求体）
//   - 调用 service 层完成业务逻辑
//   - 统一使用 response 包返回 JSON 格式的响应
//
// handler 层不包含业务逻辑，仅做参数校验、类型转换和错误码映射。
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/service"
)

// AuthHandler 处理认证相关的 HTTP 端点：
//   - POST /api/v1/auth/login  —— 微信 code 登录
//   - POST /api/v1/auth/refresh —— Token 刷新
//
// 这两个端点不在 JWT 中间件的保护范围内（skipPaths），
// 用户在未登录状态下即可访问。
type AuthHandler struct {
	authService *service.AuthService // 认证业务逻辑
}

// NewAuthHandler 创建 AuthHandler 实例。
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login 处理 POST /api/v1/auth/login —— 微信小程序 code 登录。
//
// 请求流程：
//  1. 从请求体解析微信临时授权码（code）
//  2. 调用 AuthService.LoginByWechatCode 完成登录
//  3. 返回 Access Token + Refresh Token + 用户信息
//
// 错误场景：
//   - 缺少 code 参数 → 400 InvalidParam
//   - 微信 code 无效 → 400 ThirdPartyError
//   - 数据库写入失败 → 500 InternalError
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	// ShouldBindJSON 使用 JSON tag 和 binding tag 进行请求体解析和校验
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请输入微信授权码")
		return
	}

	result, err := h.authService.LoginByWechatCode(c.Request.Context(), req.Code, req.Role)
	if err != nil {
		// 微信 API 返回的错误统一归为第三方服务错误
		response.Error(c, errcode.ThirdPartyError, err.Error())
		return
	}

	response.Success(c, result)
}

// Refresh 处理 POST /api/v1/auth/refresh —— 刷新 Token 对。
//
// 使用 Token 轮换机制：旧 Refresh Token 在刷新后立即失效，
// 返回新的 Access Token + Refresh Token 对。
// 这样可以有效防止 Refresh Token 泄露后被滥用。
//
// 请求流程：
//  1. 解析请求体中的 refresh_token
//  2. 调用 AuthService.RefreshTokens 进行校验和轮换
//  3. 返回新的 Token 对
//
// 错误场景：
//   - 缺少 refresh_token → 400 InvalidParam
//   - Token 过期/无效/已撤销 → 401 TokenInvalid
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req service.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请提供refresh_token")
		return
	}

	result, err := h.authService.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, errcode.TokenInvalid, err.Error())
		return
	}

	response.Success(c, result)
}

// Register 处理 POST /api/v1/auth/register —— Web 开发阶段手机号注册。
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请填写手机号、昵称和密码")
		return
	}
	result, err := h.authService.RegisterByPassword(c.Request.Context(), req.Phone, req.Nickname, req.Password, req.Role)
	if err != nil {
		response.Error(c, errcode.Conflict, err.Error())
		return
	}
	response.Success(c, result)
}

// LoginPassword 处理 POST /api/v1/auth/login-password —— Web 开发阶段密码登录。
func (h *AuthHandler) LoginPassword(c *gin.Context) {
	var req service.LoginPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请填写手机号和密码")
		return
	}
	result, err := h.authService.LoginByPassword(c.Request.Context(), req.Phone, req.Password)
	if err != nil {
		response.Error(c, errcode.TokenInvalid, err.Error())
		return
	}
	response.Success(c, result)
}
