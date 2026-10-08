// Package service 是应用的核心业务逻辑层。
package service

import (
	"context"
	"fmt"

	"github.com/medical-monitor/internal/external"
	"go.uber.org/zap"
)

// NotificationService 负责多通道通知的分发。
//
// 实现两级递进通知策略（Escalation Strategy）：
//
//	第一级（10分钟）：发送微信订阅消息给子女 —— "xxx的药还未确认"
//	第二级（30分钟）：发送微信订阅消息给子女 + 短信提醒老人
//
// 通知通道：
//   - 微信订阅消息：通过模板消息推送到微信小程序内
//   - 短信：通过腾讯云 SMS SDK 发送（P1）
//
// 所有外部服务接口均使用 interface 定义，方便在测试环境使用 Mock 实现。
type NotificationService struct {
	wechatSvc external.WechatService // 微信订阅消息服务
	smsSvc    external.SMSService    // 短信服务
	logger    *zap.Logger            // 结构化日志
}

// NewNotificationService 创建 NotificationService 实例。
// smsSvc 可以为 nil（未配置时不发送）。
func NewNotificationService(
	wechatSvc external.WechatService,
	smsSvc external.SMSService,
	logger *zap.Logger,
) *NotificationService {
	return &NotificationService{
		wechatSvc: wechatSvc,
		smsSvc:    smsSvc,
		logger:    logger,
	}
}

// SendReminder 向老人发送用药提醒微信订阅消息。
// 使用微信小程序订阅消息模板 "medication_reminder"，
// 模板字段映射：
//   - thing1: 药品名称列表
//   - time2:  用药时间标签
//   - thing3: 提醒备注文字
//
// 参数：
//   - ctx: 上下文
//   - openid: 老人的微信 openid
//   - elderName: 老人名称
//   - medicines: 药品名称+剂量文本，如"阿司匹林 1粒、维生素C 2片"
//   - timeLabel: 时间标签，如"早上 08:00"
//
// 返回值：微信 API 调用错误或 nil
func (s *NotificationService) SendReminder(ctx context.Context, openid, elderName string, medNames string, dosages string, timeLabel string) error {
	// 构建提醒消息内容（用于日志记录）
	content := fmt.Sprintf("您%s需要服用：%s %s", timeLabel, medNames, dosages)

	s.logger.Info("sending reminder to elder",
		zap.String("openid", openid),
		zap.String("content", content),
	)

	// 使用微信订阅消息模板 50301「按时吃药通知」
	// 字段：short_thing1(吃药时间)、thing2(药品名称)、thing3(药品用法)、short_thing4(药量)
	data := map[string]map[string]string{
		"short_thing1": {"value": timeLabel},
		"thing2":       {"value": medNames},
		"thing3":       {"value": "请按时服药"},
		"short_thing4": {"value": dosages},
	}

	return s.wechatSvc.SendSubscribeMessage(ctx, openid, "eX7azyEwYahM7JoFMrLOMterZzyKrvqUBjo_2Hm2d1s", data, "")
}

// SendAlertToChildren 向子女发送老人未服药的告警通知。
// 使用微信订阅消息模板 "medication_alert"，
// 模板字段映射：
//   - thing1: 老人名称
//   - thing2: 错过时间
//   - thing3: 超时时长描述
//
// 对每个子女的 openid 逐个发送，单个发送失败不影响其他人。
//
// 参数：
//   - ctx: 上下文
//   - elderName: 老人名称
//   - missedTime: 错过的服药时间，如"08:00"
//   - alertLevel: 告警级别（1=10分钟、2=30分钟）
//   - childOpenids: 所有子女的微信 openid 列表
func (s *NotificationService) SendAlertToChildren(ctx context.Context, elderName string, missedTime string, alertLevel int8, medicines string, childOpenids []string) {
	levelLabel := s.getAlertLevelLabel(alertLevel)

	for _, openid := range childOpenids {
		content := fmt.Sprintf("%s %s的药还未确认，已超时超过%s", elderName, missedTime, levelLabel)
		s.logger.Info("sending alert to child",
			zap.String("openid", openid),
			zap.String("content", content),
		)

		data := map[string]map[string]string{
			"thing6": {"value": elderName},
			"thing2": {"value": medicines},
			"thing1": {"value": fmt.Sprintf("超时%s未服药", levelLabel)},
		}

		// 单个子女发送失败仅记录日志，不影响其他子女的通知
		if err := s.wechatSvc.SendSubscribeMessage(ctx, openid, "OpY3dd8lVTQyq_Wh-5ow21boUYK95-KeKLreH8Snw00", data, ""); err != nil {
			s.logger.Error("failed to send alert to child",
				zap.String("openid", openid),
				zap.Error(err),
			)
		}
	}
}

// SendSMS 向老人发送短信提醒（第二级通知）。
// 短信内容包含老人名称和未服用的药品列表。
//
// 参数：
//   - ctx: 上下文
//   - phone: 老人的手机号（加密存储，此处为已解密的值）
//   - elderName: 老人名称
//   - medicines: 药品列表文本
//
// 返回值：短信发送结果
func (s *NotificationService) SendSMS(ctx context.Context, phone, elderName string, medicines string) error {
	if s.smsSvc == nil {
		return fmt.Errorf("短信服务未配置")
	}
	// 短信模板：【朝夕小叮】{老人名称}，您有药品尚未服用：{药品列表}，请及时服药。
	content := fmt.Sprintf("【朝夕小叮】%s，您有药品尚未服用：%s，请及时服药。", elderName, medicines)
	return s.smsSvc.Send(ctx, phone, content)
}

// getAlertLevelLabel 将告警级别数字转为中文描述。
// 级别 1: 10分钟 / 级别 2: 30分钟
func (s *NotificationService) getAlertLevelLabel(level int8) string {
	switch level {
	case 1:
		return "10分钟"
	case 2:
		return "30分钟"
	default:
		return "未知"
	}
}
