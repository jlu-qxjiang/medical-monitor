// Package external 定义外部服务集成的接口及其 Mock 实现。
// 每个外部服务（微信）都遵循"接口 + 实现 + Mock"的设计模式：
//   - 接口（Interface）：定义业务所需的抽象操作
//   - 生产实现：对接真实第三方 API（独立文件或同一文件中）
//   - Mock 实现：用于开发和测试环境，模拟真实接口行为
//
// 这种模式使得上层服务（Service 层）通过依赖注入使用接口，
// 可以在不改变业务代码的情况下切换生产环境和开发环境。
package external

import (
	"context"
	"fmt"
)

// WechatSession 表示微信 code2Session 接口的返回结果。
// 对应微信官方文档：https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html
//
// 字段说明：
//   - openid: 用户在此小程序下的唯一标识（不同小程序 openid 不同）
//   - session_key: 会话密钥，用于解密微信加密数据（手机号等）
//   - unionid: 用户在开放平台下的唯一标识（同一主体下多个应用共享）
//   - errcode: 错误码，0 表示成功
//   - errmsg: 错误信息
type WechatSession struct {
	Openid     string `json:"openid"`      // 用户唯一标识
	SessionKey string `json:"session_key"` // 会话密钥（用于数据解密）
	Unionid    string `json:"unionid"`     // 开放平台统一标识
	ErrCode    int    `json:"errcode"`     // 错误码（0=成功）
	ErrMsg     string `json:"errmsg"`      // 错误消息
}

// WechatService 定义微信小程序相关的服务接口。
// 上层服务（如 AuthService、NotificationService）依赖此接口而非具体实现，
// 实现了依赖倒置原则（DIP）。
type WechatService interface {
	// Code2Session 将微信小程序临时登录 code 换取 session 信息。
	// 这是微信登录流程中的关键一步：小程序端调用 wx.login() 获取 code，
	// 服务端用 code 调用此接口获取 openid 和 session_key。
	//
	// 参数：
	//   - ctx: 上下文
	//   - code: 微信小程序登录的临时凭证（有效期为5分钟，一次性使用）
	//
	// 返回值：
	//   - *WechatSession: openid、session_key、unionid 等信息
	//   - error: code 无效、appid 不匹配或微信服务不可用时返回错误
	Code2Session(ctx context.Context, code string) (*WechatSession, error)

	// SendSubscribeMessage 发送微信小程序订阅消息。
	// 需要用户事先在小程序内授权订阅对应模板。
	//
	// 参数：
	//   - ctx: 上下文
	//   - openid: 接收消息的用户的 openid
	//   - templateID: 微信公众平台的模板 ID
	//   - data: 模板变量数据，格式为 map[string]map[string]string，
	//           如 {"thing1": {"value": "阿司匹林"}}
	//   - page: 点击消息后跳转的小程序页面路径（可选）
	//
	// 返回值：发送结果，微信 API 返回非0 errcode 时返回 error
	SendSubscribeMessage(ctx context.Context, openid, templateID string, data map[string]map[string]string, page string) error
}

// MockWechatService 是 WechatService 的 Mock 实现，用于开发和测试环境。
// 不调用真实的微信 API，而是在控制台打印日志。
//
// Code2Session 会根据传入的 code 生成模拟的 openid（格式：mock_openid_{code}），
// 这样在开发阶段无需配置真实的微信小程序即可测试完整的登录流程。
type MockWechatService struct{}

// NewMockWechatService 创建 MockWechatService 实例。
func NewMockWechatService() *MockWechatService {
	return &MockWechatService{}
}

// Code2Session Mock 实现：根据 code 生成模拟的微信 session 信息。
// 生成的 openid 格式为 mock_openid_{code}（截断至64字符），
// session_key 固定为 "mock_session_key"。
//
// 校验规则：
//   - code 不能为空，为空时返回错误模拟微信 API 的校验行为
func (m *MockWechatService) Code2Session(ctx context.Context, code string) (*WechatSession, error) {
	if code == "" {
		return nil, fmt.Errorf("code不能为空")
	}
	// 使用 code 生成确定性但唯一的 mock openid
	// 格式：mock_openid_{code}（截断至64字符以内）
	mockOpenid := fmt.Sprintf("mock_openid_%s", code)
	if len(mockOpenid) > 64 {
		mockOpenid = mockOpenid[:64]
	}
	return &WechatSession{
		Openid:     mockOpenid,
		SessionKey: "mock_session_key",
		Unionid:    fmt.Sprintf("mock_unionid_%s", code),
	}, nil
}

// SendSubscribeMessage Mock 实现：在控制台打印订阅消息内容，不实际发送。
// 日志格式包含 openid、模板 ID、模板数据和跳转页面路径。
func (m *MockWechatService) SendSubscribeMessage(ctx context.Context, openid, templateID string, data map[string]map[string]string, page string) error {
	// 开发环境仅打印日志，便于调试消息模板内容
	fmt.Printf("[MockWechat] Sending subscribe message to %s, template=%s, data=%v, page=%s\n", openid, templateID, data, page)
	return nil
}
