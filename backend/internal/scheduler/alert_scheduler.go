package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/repository"
	"github.com/medical-monitor/internal/service"
	"go.uber.org/zap"
)

// AlertScheduler 负责告警升级检测的定时任务。
// 每60秒运行一次，检测已超时未确认的提醒并进行告警。
//
// 告警策略（单级）：
//
//	超时30分钟（AlertLevel30Min）：
//	  家庭模式：微信告警所有子女 + 微信再提醒老人本人
//	  个人模式：微信再提醒本人
//
// 告警级别定义在 model 包中：
//
//	AlertLevelNormal = 0  // 正常（未超时）
//	AlertLevel30Min  = 2  // 超时30分钟
type AlertScheduler struct {
	reminderRepo *repository.ReminderRepository // 提醒记录查询和更新
	familyRepo   *repository.FamilyRepository   // 家庭成员查询（获取子女列表）
	userRepo     *repository.UserRepository     // 用户 openid 查询
	notifSvc     *service.NotificationService   // 多通道通知分发
	logger       *zap.Logger                    // 结构化日志
}

// NewAlertScheduler 创建 AlertScheduler 实例。
func NewAlertScheduler(
	reminderRepo *repository.ReminderRepository,
	familyRepo *repository.FamilyRepository,
	userRepo *repository.UserRepository,
	notifSvc *service.NotificationService,
	logger *zap.Logger,
) *AlertScheduler {
	return &AlertScheduler{
		reminderRepo: reminderRepo,
		familyRepo:   familyRepo,
		userRepo:     userRepo,
		notifSvc:     notifSvc,
		logger:       logger,
	}
}

// Run 是定时任务的入口方法，由 robfig/cron 每60秒调用一次。
// 单级检测：checkTimeout(30分)：正常 → 30分钟告警（通知子女 + 再提醒本人）。
func (s *AlertScheduler) Run() {
	ctx := context.Background()
	now := time.Now()

	s.logger.Info("alert scheduler running", zap.String("time", now.Format("15:04:05")))

	// 单级告警：超时 30 分钟 → 通知子女 + 微信再提醒本人
	s.checkTimeout(ctx, now, 30*time.Minute, model.AlertLevelNormal, model.AlertLevel30Min, "30分钟")
}

// checkTimeout 执行单一级别的超时检测与告警升级。
//
// 参数：
//   - ctx: 上下文
//   - now: 当前时间
//   - threshold: 超时阈值（如30分钟、60分钟、90分钟）
//   - fromLevel: 当前告警级别（仅升级该级别的记录）
//   - toLevel: 目标告警级别
//   - levelLabel: 级别中文标签（用于日志）
//
// 升级逻辑：
//  1. 查询 remind_time < (now - threshold) 且 alert_level == fromLevel 的记录
//  2. 更新 alert_level 为 toLevel
//  3. 根据 toLevel 选择对应的通知方式
func (s *AlertScheduler) checkTimeout(ctx context.Context, now time.Time, threshold time.Duration, fromLevel, toLevel int8, levelLabel string) {
	// 计算截止时间：提醒时间必须在此时间之前才算超时
	cutoff := now.Add(-threshold)

	// 查询已超时且告警级别为 fromLevel 的提醒
	reminders, err := s.reminderRepo.FindExpiredPendingReminders(ctx, cutoff)
	if err != nil {
		s.logger.Error("failed to find expired reminders", zap.Error(err))
		return
	}

	for _, reminder := range reminders {
		// 仅升级当前级别为 fromLevel 的记录
		// 确保告警的逐步递进，不会跳过中间的级别
		if reminder.AlertLevel != fromLevel {
			continue
		}

		s.logger.Info("escalating alert",
			zap.Uint64("reminder_id", reminder.ID),
			zap.Int8("from_level", fromLevel),
			zap.Int8("to_level", toLevel),
			zap.String("threshold", levelLabel),
		)

		// 更新告警级别（先更新数据库再发送通知，防止通知重复）
		if err := s.reminderRepo.UpdateReminderFields(ctx, reminder.ID, map[string]interface{}{
			"alert_level": toLevel,
		}); err != nil {
			s.logger.Error("failed to update alert level", zap.Error(err))
			continue
		}

		// 获取老人基本信息（昵称用于通知内容）
		elderUser, err := s.userRepo.FindByID(ctx, reminder.ElderUserID)
		if err != nil {
			s.logger.Error("failed to find elder user", zap.Error(err))
			continue
		}

		// 构建药品名列表和剂量列表（分开，对应模板字段）
		medicines := ""
		dosages := ""
		for i, item := range reminder.Items {
			if i > 0 {
				medicines += "、"
				dosages += "、"
			}
			medicines += item.MedicineName
			dosages += fmt.Sprintf("%.0f%s", item.Dosage, item.Unit)
		}

		// 老人错过的服药时间
		missedTime := reminder.RemindTime.Format("15:04")

		// 获取家庭组中所有非老人成员（子女）的 openid
		members, err := s.familyRepo.FindMembersByFamilyID(ctx, reminder.FamilyID)
		if err != nil {
			s.logger.Error("failed to find family members", zap.Error(err))
			continue
		}

		childOpenids := make([]string, 0)
		for _, m := range members {
			if m.Role == model.RoleElder {
				continue // 跳过老人本人
			}
			childUser, err := s.userRepo.FindByID(ctx, m.UserID)
			if err != nil {
				continue
			}
			childOpenids = append(childOpenids, childUser.Openid)
		}

		// 根据目标告警级别选择通知方式
		switch toLevel {
		case model.AlertLevel30Min:
			// 30分钟 → 微信告警子女 + 微信再提醒本人（个人模式只再提醒本人）
			timeLabel := fmt.Sprintf("%s %s", reminder.RemindDate, missedTime)
			if reminder.IsPersonal {
				s.notifSvc.SendReminder(ctx, elderUser.Openid, elderUser.Nickname, medicines, dosages, timeLabel)
			} else {
				s.notifSvc.SendAlertToChildren(ctx, elderUser.Nickname, missedTime, toLevel, medicines, childOpenids)
				s.notifSvc.SendReminder(ctx, elderUser.Openid, elderUser.Nickname, medicines, dosages, timeLabel)
			}
		}
	}
}
