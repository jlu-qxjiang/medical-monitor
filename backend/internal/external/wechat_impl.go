package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// RealWechatService 是 WechatService 的真实实现，对接微信官方 API。
// 用于生产环境，替代 MockWechatService。
type RealWechatService struct {
	appID     string
	appSecret string
	client    *http.Client

	// access_token 缓存（微信 access_token 有效期 2 小时）
	mu          sync.Mutex
	accessToken string
	tokenExpire time.Time
}

// NewRealWechatService 创建真实微信服务实现。
func NewRealWechatService(appID, appSecret string) *RealWechatService {
	return &RealWechatService{
		appID:     appID,
		appSecret: appSecret,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// code2SessionResp 微信 jscode2session 接口响应。
type code2SessionResp struct {
	Openid     string `json:"openid"`
	SessionKey string `json:"session_key"`
	Unionid    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// Code2Session 调用微信 jscode2session 接口，用临时 code 换取 openid。
func (s *RealWechatService) Code2Session(ctx context.Context, code string) (*WechatSession, error) {
	if code == "" {
		return nil, fmt.Errorf("code不能为空")
	}

	// 构造请求 URL
	apiURL := "https://api.weixin.qq.com/sns/jscode2session"
	params := url.Values{}
	params.Set("appid", s.appID)
	params.Set("secret", s.appSecret)
	params.Set("js_code", code)
	params.Set("grant_type", "authorization_code")

	fullURL := apiURL + "?" + params.Encode()
	//通过url构造请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	//通过构造好的请求去调用微信接口
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用微信接口失败: %w", err)
	}
	//延迟关闭的原因是：下边的读取数据需要在关闭连接之前完成，否则会导致读取不到数据
	defer resp.Body.Close()
	//读取返回响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result code2SessionResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("微信返回错误 %d: %s", result.ErrCode, result.ErrMsg)
	}

	return &WechatSession{
		Openid:     result.Openid,
		SessionKey: result.SessionKey,
		Unionid:    result.Unionid,
	}, nil
}

// tokenResp 微信 access_token 接口响应。
type tokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// getAccessToken 获取并缓存微信 access_token（有效期 2 小时，提前 5 分钟刷新）。
func (s *RealWechatService) getAccessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 缓存未过期直接返回
	if s.accessToken != "" && time.Now().Before(s.tokenExpire) {
		return s.accessToken, nil
	}

	apiURL := "https://api.weixin.qq.com/cgi-bin/token"
	params := url.Values{}
	params.Set("grant_type", "client_credential")
	params.Set("appid", s.appID)
	params.Set("secret", s.appSecret)

	resp, err := s.client.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("获取 access_token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result tokenResp
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析 access_token 失败: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("微信返回错误 %d: %s", result.ErrCode, result.ErrMsg)
	}

	s.accessToken = result.AccessToken
	// 提前 5 分钟过期，避免边界问题
	s.tokenExpire = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	return s.accessToken, nil
}

// SendSubscribeMessage 发送微信订阅消息。
// 需要先获取 access_token，再调用订阅消息接口。
func (s *RealWechatService) SendSubscribeMessage(ctx context.Context, openid, templateID string, data map[string]map[string]string, page string) error {
	token, err := s.getAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s", token)

	payload := map[string]interface{}{
		"touser":      openid,
		"template_id": templateID,
		"data":        data,
	}
	if page != "" {
		payload["page"] = page
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送订阅消息失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	json.Unmarshal(respBody, &result)
	if result.ErrCode != 0 {
		return fmt.Errorf("订阅消息发送失败 %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}
