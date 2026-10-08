package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/utils"
	"github.com/medical-monitor/internal/repository"
)

// ReminderService 负责用药提醒的确认、跳过、未服用标记和撤销等交互操作。
// 提醒由定时任务（ReminderScheduler）根据用药计划自动生成。
// 此 Service 处理老人对提醒的操作反馈。
//
// 核心业务规则：
//   - 只有老人本人可以确认/跳过/标记未服用/撤销
//   - 已确认的提醒在5分钟内可以撤销
//   - 确认药品时会自动扣减对应药品的库存余量
//   - 撤销确认时会恢复库存余量
type ReminderService struct {
	reminderRepo *repository.ReminderRepository // 提醒记录持久化
	medicineRepo *repository.MedicineRepository // 药品库存扣减/恢复
	auditRepo    *repository.AuditRepository    // 审计日志
}

// NewReminderService 创建 ReminderService 实例。
func NewReminderService(
	reminderRepo *repository.ReminderRepository,
	medicineRepo *repository.MedicineRepository,
	auditRepo *repository.AuditRepository,
) *ReminderService {
	return &ReminderService{
		reminderRepo: reminderRepo,
		medicineRepo: medicineRepo,
		auditRepo:    auditRepo,
	}
}

// GetToday 获取当前用户（老人）今天的用药提醒列表。
// 返回的每个提醒包含药品明细和服用状态。
//
// 参数：
//   - ctx: 上下文
//   - elderUserID: 老人用户 ID
//   - familyID: 家庭组 ID（权限范围）
//
// 返回值：
//   - *TodayRemindersResponse: 今日提醒列表及日期
//   - error: 查询失败
func (s *ReminderService) GetToday(ctx context.Context, elderUserID, familyID uint64) (*TodayRemindersResponse, error) {
	today := utils.Today()
	reminders, err := s.reminderRepo.FindTodayReminders(ctx, elderUserID, familyID, today)
	if err != nil {
		return nil, fmt.Errorf("查询今日提醒失败: %w", err)
	}

	reminderList := make([]TodayReminder, 0, len(reminders))
	for _, r := range reminders {
		tr := TodayReminder{
			ID:         r.ID,
			RemindTime: r.RemindTime,
			TimeLabel:  s.formatTimeLabel(r.RemindTime), // 生成人性化时间标签，如"早上 08:00"
			Status:     r.Status,
			Items:      make([]TodayReminderItem, 0, len(r.Items)),
		}
		for _, item := range r.Items {
			tri := TodayReminderItem{
				ID:           item.ID,
				MedicineName: item.MedicineName,
				Dosage:       item.Dosage,
				Unit:         item.Unit,
				Taken:        item.Taken,   // 是否已服用
				TakenAt:      item.TakenAt, // 服用时间
			}
			tr.Items = append(tr.Items, tri)
		}
		reminderList = append(reminderList, tr)
	}

	return &TodayRemindersResponse{
		Date:      today,
		Reminders: reminderList,
	}, nil
}

// Confirm 确认已服用指定药品。
// 业务流程：
//  1. 校验提醒记录存在且属于该老人
//  2. 校验提醒未处于已完成/已跳过/已忽略等终态
//  3. 标记指定的药品明细为"已服用"
//  4. 扣减对应药品库存（每确认一项扣1粒）
//  5. 若所有药品都已确认，将提醒状态更新为"已确认"
//  6. 返回撤销截止时间（确认后5分钟内可撤销）
//
// 参数：
//   - ctx: 上下文
//   - userID: 当前用户（必须是老人本人）
//   - reminderID: 提醒记录 ID
//   - itemIDs: 要确认的药品明细 ID 列表（支持部分确认）
//
// 返回值：
//   - *ConfirmResponse: 包含撤销截止时间
//   - error: 权限不足或状态不对
func (s *ReminderService) Confirm(ctx context.Context, userID, reminderID uint64, itemIDs []uint64) (*ConfirmResponse, error) {
	// 查找提醒记录
	reminder, err := s.reminderRepo.FindReminderByID(ctx, reminderID)
	if err != nil {
		return nil, fmt.Errorf("提醒记录不存在")
	}

	// 权限校验：只有老人本人可以确认用药
	if reminder.ElderUserID != userID {
		return nil, fmt.Errorf("只有本人可以确认用药")
	}

	// 状态校验：不能重复确认已处理过的提醒
	if reminder.Status == model.ReminderStatusConfirmed || reminder.Status == model.ReminderStatusMissed || reminder.Status == model.ReminderStatusSkipped {
		return nil, fmt.Errorf("该提醒已处理，无需重复操作")
	}

	now := time.Now()
	nowTime := now

	// 构建请求确认的药品 ID 集合（用于快速查找）
	itemIDSet := make(map[uint64]bool)
	for _, id := range itemIDs {
		itemIDSet[id] = true
	}

	allItemsConfirmed := true
	for _, item := range reminder.Items {
		// 检查该药品是否在此次确认范围内
		if itemIDSet[item.ID] {
			// 如果已标记为已服用，跳过
			if item.Taken != nil && *item.Taken {
				continue
			}
			// 更新该药品明细为已服用
			if err := s.reminderRepo.UpdateReminderItemTaken(ctx, item.ID, true, &nowTime); err != nil {
				return nil, fmt.Errorf("更新确认状态失败: %w", err)
			}
			// 扣减库存（每确认一项减1粒/片），按家庭/个人分流
			if reminder.IsPersonal {
				_ = s.medicineRepo.DecrementPersonalStock(ctx, item.MedicineID, int(math.Max(1, item.Dosage)))
			} else {
				_ = s.medicineRepo.DecrementStock(ctx, item.MedicineID, int(math.Max(1, item.Dosage)))
			}
		} else {
			// 检查未在请求范围内但之前也未服用的项
			if item.Taken == nil || !*item.Taken {
				allItemsConfirmed = false
			}
		}
	}

	// 如果该提醒的所有药品都已确认，更新提醒状态为"已确认"
	if allItemsConfirmed {
		if err := s.reminderRepo.UpdateReminderFields(ctx, reminderID, map[string]interface{}{
			"status":       model.ReminderStatusConfirmed,
			"confirmed_at": now,
			"confirmed_by": userID,
		}); err != nil {
			return nil, fmt.Errorf("更新提醒状态失败: %w", err)
		}
	}

	// 计算撤销截止时间：确认后5分钟内可撤销
	canRevokeUntil := now.Add(5 * time.Minute)

	// 审计日志
	uid := userID
	fid := reminder.FamilyID
	rid := reminderID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionReminderConfirm, model.AuditTargetReminder, &rid, "", "")

	return &ConfirmResponse{
		CanRevokeUntil: canRevokeUntil, // 前端用于展示倒计时
	}, nil
}

// MarkMissed 将提醒标记为"未服用"（老人忘记服药的情况）。
// 会在每个药品明细上记录原因。
//
// 参数：
//   - ctx: 上下文
//   - userID: 当前用户
//   - reminderID: 提醒记录 ID
//   - reason: 未服用原因
//
// 返回值：错误或 nil
func (s *ReminderService) MarkMissed(ctx context.Context, userID, reminderID uint64, reason string) error {
	reminder, err := s.reminderRepo.FindReminderByID(ctx, reminderID)
	if err != nil {
		return fmt.Errorf("提醒记录不存在")
	}

	// 仅本人可标记
	if reminder.ElderUserID != userID {
		return fmt.Errorf("只有本人可以标记")
	}

	// 更新提醒状态为"未服用"
	if err := s.reminderRepo.UpdateReminderStatus(ctx, reminderID, model.ReminderStatusMissed); err != nil {
		return fmt.Errorf("更新提醒状态失败: %w", err)
	}

	// 为每个药品明细设置跳过原因
	_ = s.reminderRepo.UpdateReminderItemsSkipped(ctx, reminderID, true, reason)

	// 审计日志
	uid := userID
	fid := reminder.FamilyID
	rid := reminderID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionReminderMiss, model.AuditTargetReminder, &rid, fmt.Sprintf(`{"reason":"%s"}`, reason), "")

	return nil
}

// Skip 跳过本次用药提醒（医生建议或特殊情况）。
// 与 MarkMissed 的区别：Skip 是主动选择不服用，Missed 是忘记服用。
//
// 参数：
//   - ctx: 上下文
//   - userID: 当前用户
//   - reminderID: 提醒记录 ID
//   - reason: 跳过原因
//
// 返回值：错误或 nil
func (s *ReminderService) Skip(ctx context.Context, userID, reminderID uint64, reason string) error {
	reminder, err := s.reminderRepo.FindReminderByID(ctx, reminderID)
	if err != nil {
		return fmt.Errorf("提醒记录不存在")
	}

	if reminder.ElderUserID != userID {
		return fmt.Errorf("只有本人可以跳过")
	}

	if err := s.reminderRepo.UpdateReminderStatus(ctx, reminderID, model.ReminderStatusSkipped); err != nil {
		return fmt.Errorf("更新提醒状态失败: %w", err)
	}

	// 为每个药品明细设置跳过原因
	_ = s.reminderRepo.UpdateReminderItemsSkipped(ctx, reminderID, true, reason)

	// 审计日志
	uid := userID
	fid := reminder.FamilyID
	rid := reminderID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionReminderSkip, model.AuditTargetReminder, &rid, fmt.Sprintf(`{"reason":"%s"}`, reason), "")

	return nil
}

// Revoke 撤销已确认的用药提醒（5分钟窗口内有效）。
// 业务流程：
//  1. 验证提醒状态为"已确认"
//  2. 检查确认时间距今不超过5分钟
//  3. 将提醒状态恢复为"已发送"
//  4. 将所有药品明细恢复为"未服用"
//  5. 恢复被扣减的库存
//
// 参数：
//   - ctx: 上下文
//   - userID: 当前用户
//   - reminderID: 提醒记录 ID
//
// 返回值：错误或 nil
func (s *ReminderService) Revoke(ctx context.Context, userID, reminderID uint64) error {
	reminder, err := s.reminderRepo.FindReminderByID(ctx, reminderID)
	if err != nil {
		return fmt.Errorf("提醒记录不存在")
	}

	if reminder.ElderUserID != userID {
		return fmt.Errorf("只有本人可以撤销")
	}

	if reminder.Status != model.ReminderStatusConfirmed {
		return fmt.Errorf("只能撤销已确认的提醒")
	}

	// 检查5分钟撤销窗口
	if reminder.ConfirmedAt == nil {
		return fmt.Errorf("确认时间异常")
	}
	if time.Since(*reminder.ConfirmedAt) > 5*time.Minute {
		return fmt.Errorf("已超过撤销时间")
	}

	// 将提醒状态恢复为 sent，清除确认信息
	if err := s.reminderRepo.UpdateReminderFields(ctx, reminderID, map[string]interface{}{
		"status":       model.ReminderStatusSent,
		"confirmed_at": nil,
		"confirmed_by": nil,
	}); err != nil {
		return fmt.Errorf("撤销确认失败: %w", err)
	}

	// 逐个药品明细恢复：重置服用状态 + 恢复库存
	for _, item := range reminder.Items {
		if item.Taken != nil && *item.Taken {
			_ = s.reminderRepo.UpdateReminderItemTaken(ctx, item.ID, false, nil)
			// 恢复被扣减的库存（每撤销一项加1粒），按家庭/个人分流
			if reminder.IsPersonal {
				_ = s.medicineRepo.IncrementPersonalStock(ctx, item.MedicineID, 1)
			} else {
				_ = s.medicineRepo.IncrementStock(ctx, item.MedicineID, 1)
			}
		}
	}

	// 审计日志
	uid := userID
	fid := reminder.FamilyID
	rid := reminderID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionReminderRevoke, model.AuditTargetReminder, &rid, "", "")

	return nil
}

// formatTimeLabel 将时间对象生成中文时间标签，增强前端可读性。
// 示例：08:00 → "上午 08:00"，14:30 → "中午 14:30"，19:00 → "晚上 19:00"
// 时段划分：凌晨(0-6) / 上午(6-12) / 中午(12-18) / 晚上(18-24)
func (s *ReminderService) formatTimeLabel(t time.Time) string {
	hour := t.Hour()
	prefix := "上午"
	if hour >= 12 && hour < 18 {
		prefix = "中午"
	} else if hour >= 18 {
		prefix = "晚上"
	} else if hour < 6 {
		prefix = "凌晨"
	}
	return fmt.Sprintf("%s %s", prefix, t.Format("15:04"))
}

// --- Reminder request/response 类型定义 ---

// TodayRemindersResponse 是 GET /reminders/today 的响应体。
// date 格式为 "YYYY-MM-DD"。
type TodayRemindersResponse struct {
	Date      string          `json:"date"`      // 日期，如"2024-01-15"
	Reminders []TodayReminder `json:"reminders"` // 今日提醒列表
}

// TodayReminder 是今日提醒中的单个提醒条目。
// time_label 字段用于前端 UI 分组展示。
type TodayReminder struct {
	ID         uint64              `json:"id"`          // 提醒 ID
	RemindTime time.Time           `json:"remind_time"` // 预计提醒时间
	TimeLabel  string              `json:"time_label"`  // 人性化时间标签，如"早上 08:00"
	Status     string              `json:"status"`      // 提醒状态：pending/sent/confirmed/missed/skipped
	Items      []TodayReminderItem `json:"items"`       // 该提醒包含的药品明细
}

// TodayReminderItem 是今日提醒中的药品明细。
// taken 为 nil 表示尚未操作，true 表示已服用，false 表示未服用。
type TodayReminderItem struct {
	ID           uint64     `json:"id"`            // 提醒明细 ID
	MedicineName string     `json:"medicine_name"` // 药品名称
	Dosage       float64    `json:"dosage"`         // 剂量
	Unit         string     `json:"unit"`           // 剂量单位
	Taken        *bool      `json:"taken"`          // 是否已服用（nil=未操作）
	TakenAt      *time.Time `json:"taken_at"`       // 服用时间
}

// ConfirmRequest 是 POST /reminders/:id/confirm 的请求体。
// item_ids 支持部分确认（一次不一定要确认所有药品）。
type ConfirmRequest struct {
	ItemIDs []uint64 `json:"item_ids" binding:"required,min=1"` // 确认的药品明细 ID 列表
}

// ConfirmResponse 是 POST /reminders/:id/confirm 的成功响应体。
// 前端根据 can_revoke_until 展示撤销倒计时。
type ConfirmResponse struct {
	CanRevokeUntil time.Time `json:"can_revoke_until"` // 撤销截止时间（确认后5分钟）
}

// MissRequest 是 POST /reminders/:id/miss 的请求体。
type MissRequest struct {
	Reason string `json:"reason" binding:"required"` // 未服用原因
}

// SkipRequest 是 POST /reminders/:id/skip 的请求体。
type SkipRequest struct {
	Reason string `json:"reason" binding:"required"` // 跳过原因
}
