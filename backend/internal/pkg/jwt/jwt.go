// Package jwt 提供 JWT Token 生成与校验。
// 使用 golang-jwt v5 + HMAC-SHA256 签名算法。
// Access Token 7 天有效期，Refresh Token 30 天有效期。
// Token 区分 subject: "access" 用于 API 鉴权，"refresh" 仅用于 Token 刷新。
package jwt

import (
	"errors"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

var (
	// ErrTokenExpired Token 已过期。
	ErrTokenExpired = errors.New("token过期")
	// ErrTokenInvalid Token 格式或签名无效。
	ErrTokenInvalid = errors.New("token无效")
	// ErrTokenNotValidYet Token 尚未到生效时间。
	ErrTokenNotValidYet = errors.New("token尚未生效")
)

// Claims 自定义 JWT 载荷，在标准 RegisteredClaims 基础上加入 user_id。
type Claims struct {
	UserID uint64 `json:"user_id"`
	jwtv5.RegisteredClaims
}

// JWTManager 管理 Token 的生成与解析，持有签名密钥和有效期配置。
type JWTManager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTManager 创建 JWT 管理器，传入密钥和两个 Token 的有效期。
func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// GenerateAccessToken 生成短期 Access Token（subject="access"）。
func (m *JWTManager) GenerateAccessToken(userID uint64) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTokenTTL)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwtv5.RegisteredClaims{
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(expiresAt),
			Subject:   "access",
		},
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, err

	}
	return tokenStr, expiresAt.Unix(), nil
}

// GenerateRefreshToken 生成长期 Refresh Token（subject="refresh"），用于换取新的 Token 对。
func (m *JWTManager) GenerateRefreshToken(userID uint64) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshTokenTTL)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwtv5.RegisteredClaims{
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(expiresAt),
			Subject:   "refresh",
		},
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, err
	}
	return tokenStr, expiresAt.Unix(), nil
}

// ParseToken 解析并校验 JWT Token 字符串，成功返回 Claims。
func (m *JWTManager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwtv5.ParseWithClaims(tokenStr, &Claims{}, func(token *jwtv5.Token) (interface{}, error) {
		// 校验签名算法必须是 HMAC 系列，防止 alg=none 攻击
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwtv5.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwtv5.ErrTokenNotValidYet) {
			return nil, ErrTokenNotValidYet
		}
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// RefreshAccessToken 校验 Refresh Token 后生成全新的 Token 对。
// 注意：此方法仅做密码学校验，不涉及持久化存储的验证（由 AuthService 调用方处理）。
func (m *JWTManager) RefreshAccessToken(refreshTokenStr string) (string, string, int64, error) {
	claims, err := m.ParseToken(refreshTokenStr)
	if err != nil {
		return "", "", 0, err
	}
	// 只有在 subject="refresh" 时才允许换取新 Token
	if claims.Subject != "refresh" {
		return "", "", 0, ErrTokenInvalid
	}
	accessToken, _, err := m.GenerateAccessToken(claims.UserID)
	if err != nil {
		return "", "", 0, err
	}
	newRefreshToken, expiresIn, err := m.GenerateRefreshToken(claims.UserID)
	if err != nil {
		return "", "", 0, err
	}
	return accessToken, newRefreshToken, expiresIn, nil
}
