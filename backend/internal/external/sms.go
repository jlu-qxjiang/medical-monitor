package external

import (
	"context"
	"fmt"
)

// SMSService 定义短信发送服务的接口。
//
// 生产环境对接：腾讯云短信 SDK（qcloudsms）
// 接入时机：P1（MVP 之后的第一优先级）
// 用途：在用药提醒超时60分钟时向老人发送短信提醒
//
// 设计说明：
// 短信发送是第二级告警通知的组成部分。当老人超时60分钟未确认服药后，
// 系统在微信告警子女的同时，还会发短信给老人本人以加强提醒效果。
type SMSService interface {
	// Send 发送一条短信到指定手机号。
	//
	// 参数：
	//   - ctx: 上下文
	//   - phone: 接收短信的手机号码（已解密的真实号码）
	//   - content: 短信内容（不需要签名，生产实现会自动添加签名）
	//
	// 返回值：发送结果
	//   腾讯云 SMS 返回非0状态码时返回 error
	Send(ctx context.Context, phone, content string) error
}

// MockSMSService 是 SMSService 的 Mock 实现，用于开发和测试环境。
// 不实际发送短信，仅在控制台输出日志。
type MockSMSService struct{}

// NewMockSMSService 创建 MockSMSService 实例。
func NewMockSMSService() *MockSMSService {
	return &MockSMSService{}
}

// Send Mock 实现：在控制台打印短信内容。
// 校验规则：手机号不能为空。
//
// 输出格式：[MockSMS] Sending SMS to {phone}: {content}
func (m *MockSMSService) Send(ctx context.Context, phone, content string) error {
	if phone == "" {
		return fmt.Errorf("手机号不能为空")
	}
	fmt.Printf("[MockSMS] Sending SMS to %s: %s\n", phone, content)
	return nil
}
