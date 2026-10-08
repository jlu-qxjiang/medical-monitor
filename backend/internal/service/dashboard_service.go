package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/utils"
	"github.com/medical-monitor/internal/repository"
)

// DashboardService 负责看板数据的聚合计算，为前端提供今日服药状态和家庭概览。
//
// 提供两个核心视角：
//   - 今日看板（TodayDashboard）：针对单个老人，展示今日所有时间段的服药完成情况
//   - 家庭概览（Overview）：针对整个家庭组，展示所有老人的服药状态汇总
//
// 状态计算逻辑：
//   - 已确认（confirmed）：提醒状态为 confirmed
//   - 已跳过（skipped）：提醒状态为 skipped
//   - 未确认（missed）：提醒状态为 sent/pending 但当前时间已超过提醒时间
//   - 等待中（pending）：提醒状态为 sent/pending 且当前时间尚未到提醒时间
//
// 危险等级判断：
//   - normal：全部正常
//   - warning：存在超时未确认的提醒
//   - danger：存在超时超过60分钟的提醒
type DashboardService struct {
	reminderRepo *repository.ReminderRepository // 提醒数据查询
	familyRepo   *repository.FamilyRepository   // 家庭成员信息
	userRepo     *repository.UserRepository     // 用户信息
}

// NewDashboardService 创建 DashboardService 实例。
func NewDashboardService(
	reminderRepo *repository.ReminderRepository,
	familyRepo *repository.FamilyRepository,
	userRepo *repository.UserRepository,
) *DashboardService {
	return &DashboardService{
		reminderRepo: reminderRepo,
		familyRepo:   familyRepo,
		userRepo:     userRepo,
	}
}

// GetTodayDashboard 获取指定老人今日的用药看板数据。
// 包括：
//   - 老人的基本信息
//   - 今日完成情况汇总（总数、已确认、待处理、未确认、已跳过、完成率）
//   - 各个时间段的药品明细和状态
//
// 场景：子女点击某个老人后查看的详细看板页面。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - elderUserID: 老人用户 ID
//
// 返回值：
//   - *TodayDashboardResponse: 完整的今日看板数据
//   - error: 老人不存在或查询失败
func (s *DashboardService) GetTodayDashboard(ctx context.Context, familyID, elderUserID uint64) (*TodayDashboardResponse, error) {
	today := utils.Today()

	// 获取老人的基本信息
	elderUser, err := s.userRepo.FindByID(ctx, elderUserID)
	if err != nil {
		return nil, fmt.Errorf("老人用户不存在")
	}

	// 获取该老人今日所有提醒记录
	reminders, err := s.reminderRepo.FindRemindersByElderAndDate(ctx, elderUserID, today)
	if err != nil {
		return nil, fmt.Errorf("查询提醒记录失败: %w", err)
	}

	// 计算汇总统计
	total := len(reminders)
	confirmed := 0
	pending := 0
	skipped := 0
	var missed int

	for _, r := range reminders {
		switch r.Status {
		case model.ReminderStatusConfirmed:
			confirmed++
		case model.ReminderStatusSkipped:
			skipped++
		case model.ReminderStatusMissed:
			missed++
		default:
			// sent 或 pending 状态：如果提醒时间已过，视为未确认
			if time.Now().After(r.RemindTime) {
				missed++
			} else {
				pending++
			}
		}
	}

	// 计算完成率（已确认 / 总数）
	completionRate := float64(0)
	if total > 0 {
		completionRate = float64(confirmed) / float64(total)
	}

	// 构建各个时间段的数据
	slots := make([]TimeSlot, 0, len(reminders))
	for _, r := range reminders {
		slot := TimeSlot{
			ReminderID:  r.ID,
			TimeOfDay:   r.RemindTime.Format("15:04"),
			TimeLabel:   s.getTimeLabel(r.RemindTime), // 中文时段标签
			Status:      r.Status,
			ConfirmedAt: r.ConfirmedAt,
			Items:       make([]SlotMedicineItem, 0, len(r.Items)),
		}

		// 对于 sent/pending 状态但已过期的提醒，在前端展示为 missed
		if (slot.Status == model.ReminderStatusSent || slot.Status == model.ReminderStatusPending) && time.Now().After(r.RemindTime) {
			slot.Status = model.ReminderStatusMissed
		}

		// 生成状态标签（含时间信息和提前/超时偏移，如"已服用 08:05（提前30分钟）"）
		slot.StatusLabel = s.getStatusLabel(slot.Status, r.RemindTime, r.ConfirmedAt)

		// 已确认时，计算服药时间相对计划时间的偏移（正=超时，负=提前）
		if slot.Status == model.ReminderStatusConfirmed && r.ConfirmedAt != nil {
			offset := utils.TakenOffsetMinutes(r.RemindTime, *r.ConfirmedAt)
			slot.TakenOffsetMinutes = &offset
		}

		// 填充药品明细
		for _, item := range r.Items {
			slot.Items = append(slot.Items, SlotMedicineItem{
				MedicineName: item.MedicineName,
				Dosage:       item.Dosage,
				Unit:         item.Unit,
				Taken:        item.Taken,
			})
		}

		slots = append(slots, slot)
	}

	return &TodayDashboardResponse{
		Elder: ElderDashboardInfo{
			UserID:   elderUser.ID,
			Nickname: elderUser.Nickname,
		},
		Date: today,
		Summary: DashboardSummary{
			Total:          total,
			Confirmed:      confirmed,
			Pending:        pending,
			Missed:         missed,
			Skipped:        skipped,
			CompletionRate: completionRate,
		},
		Slots: slots,
	}, nil
}

// GetOverview 获取家庭组级别的概览数据，展示所有老人的今日服药状态。
//
// 对每个老人计算：
//   - 整体状态：normal（正常）/ warning（有超时）/ danger（超时超过60分钟）
//   - 确认数 / 总数
//   - 下一次用药提醒的时间和药品
//
// 场景：家庭首页，子女可以一目了然地看到所有老人的情况。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//
// 返回值：
//   - *OverviewResponse: 家庭概览数据
//   - error: 家庭组不存在或查询失败
func (s *DashboardService) GetOverview(ctx context.Context, familyID uint64) (*OverviewResponse, error) {
	family, err := s.familyRepo.FindFamilyByID(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("家庭组不存在")
	}

	// 获取家庭组中所有老人
	elders, err := s.familyRepo.FindEldersByFamilyID(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("查询老人列表失败: %w", err)
	}

	today := utils.Today()
	elderOverviews := make([]ElderOverview, 0, len(elders))

	for _, elderMember := range elders {
		reminders, err := s.reminderRepo.FindRemindersByElderAndDate(ctx, elderMember.UserID, today)
		if err != nil {
			continue // 跳过查询失败的老人
		}

		total := len(reminders)
		confirmed := 0
		hasOverdue := false // 是否已有超时的提醒
		hasDanger := false  // 是否超时超过60分钟（危险等级）

		var nextReminder *NextReminderInfo

		for _, r := range reminders {
			switch r.Status {
			case model.ReminderStatusConfirmed:
				confirmed++
			default:
				// 检查是否已超时
				if time.Now().After(r.RemindTime) {
					hasOverdue = true
					elapsed := time.Since(r.RemindTime)
					if elapsed > 60*time.Minute {
						hasDanger = true // 超时超过60分钟，升级为危险
					}
				} else {
					// 这是未来的提醒，找出最近的一个
					if nextReminder == nil || r.RemindTime.Before(nextReminder.NextTime) {
						medicines := make([]string, 0)
						for _, item := range r.Items {
							medicines = append(medicines, fmt.Sprintf("%s %.0f%s", item.MedicineName, item.Dosage, item.Unit))
						}
						nextReminder = &NextReminderInfo{
							TimeOfDay: r.RemindTime.Format("15:04"),
							TimeLabel: fmt.Sprintf("今晚 %s", r.RemindTime.Format("15:04")),
							Medicines: strings.Join(medicines, ", "),
						}
						nextReminder.NextTime = r.RemindTime // 内部比较用，不暴露
					}
				}
			}
		}

		// 确定整体状态：danger > warning > normal
		overallStatus := "normal"
		if hasDanger {
			overallStatus = "danger"
		} else if hasOverdue {
			overallStatus = "warning"
		}

		// 生成摘要文本，如"3/5 已确认，存在超时未确认"
		summary := fmt.Sprintf("%d/%d 已确认", confirmed, total)
		if hasOverdue {
			summary += "，存在超时未确认"
		}

		elderInfo := ElderOverview{
			UserID:        elderMember.UserID,
			OverallStatus: overallStatus,
			Summary:       summary,
			Confirmed:     confirmed,
			Total:         total,
		}
		if elderMember.User != nil {
			elderInfo.Nickname = elderMember.User.Nickname
			elderInfo.AvatarURL = elderMember.User.AvatarURL
		}

		// 仅在有未来提醒时返回 next_reminder
		if nextReminder != nil {
			nextTime := nextReminder.NextTime
			_ = nextTime // 避免未使用变量警告（保留用于未来扩展）
			elderInfo.NextReminder = &NextReminderResponse{
				TimeOfDay: nextReminder.TimeOfDay,
				TimeLabel: nextReminder.TimeLabel,
				Medicines: nextReminder.Medicines,
			}
		}

		elderOverviews = append(elderOverviews, elderInfo)
	}

	return &OverviewResponse{
		FamilyName: family.Name,
		Elders:     elderOverviews,
	}, nil
}

// getTimeLabel 根据时间返回中文时段标签。
// 时段划分：
//   凌晨 00:00-05:59
//   早上 06:00-11:59
//   中午 12:00-13:59
//   下午 14:00-17:59
//   晚上 18:00-23:59
func (s *DashboardService) getTimeLabel(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour < 6:
		return "凌晨"
	case hour < 12:
		return "早上"
	case hour < 14:
		return "中午"
	case hour < 18:
		return "下午"
	default:
		return "晚上"
	}
}

// getStatusLabel 生成前端展示用的状态标签文本。
// 已服用状态会附带服用时间和提前/超时偏移，如"已服用 08:05（提前30分钟）"方便子女核对。
func (s *DashboardService) getStatusLabel(status string, remindTime time.Time, confirmedAt *time.Time) string {
	switch status {
	case model.ReminderStatusConfirmed:
		if confirmedAt != nil {
			return fmt.Sprintf("已服用 %s（%s）", confirmedAt.Format("15:04"), utils.FormatTakenOffset(remindTime, *confirmedAt))
		}
		return "已服用"
	case model.ReminderStatusPending:
		return "等待中"
	case model.ReminderStatusSent:
		return "等待中"
	case model.ReminderStatusMissed:
		return "未确认"
	case model.ReminderStatusSkipped:
		return "已跳过"
	default:
		return "未知"
	}
}

// --- Dashboard request/response 类型定义 ---

// TodayDashboardResponse 是 GET /dashboard/today 的响应体。
// 是看板页面的核心数据结构。
type TodayDashboardResponse struct {
	Elder   ElderDashboardInfo `json:"elder"`   // 老人基本信息
	Date    string             `json:"date"`    // 日期（YYYY-MM-DD）
	Summary DashboardSummary   `json:"summary"` // 今日汇总统计
	Slots   []TimeSlot         `json:"slots"`   // 各时间段详情
}

// ElderDashboardInfo 是看板中老人的基本信息。
type ElderDashboardInfo struct {
	UserID   uint64 `json:"user_id"`   // 老人用户 ID
	Nickname string `json:"nickname"`  // 老人昵称
}

// DashboardSummary 是今日用药完成情况的汇总统计。
// completion_rate 范围 0.0 - 1.0。
type DashboardSummary struct {
	Total          int     `json:"total"`           // 今日总提醒数
	Confirmed      int     `json:"confirmed"`       // 已确认数量
	Pending        int     `json:"pending"`         // 等待中数量
	Missed         int     `json:"missed"`          // 未确认数量
	Skipped        int     `json:"skipped"`         // 已跳过数量
	CompletionRate float64 `json:"completion_rate"` // 完成率（0.0~1.0）
}

// TimeSlot 表示看板中一个用药时间段。
// 提醒从用药计划的时间生成，一个时间段可能包含多种药品。
type TimeSlot struct {
	ReminderID         uint64             `json:"reminder_id"`                    // 提醒记录 ID
	TimeOfDay          string             `json:"time_of_day"`                    // 时间（HH:MM 格式）
	TimeLabel          string             `json:"time_label"`                     // 中文时段标签（早上/中午/晚上）
	Status             string             `json:"status"`                         // 提醒状态
	StatusLabel        string             `json:"status_label"`                   // 状态文本标签（含时间和偏移）
	ConfirmedAt        *time.Time         `json:"confirmed_at,omitempty"`         // 实际确认服药时间
	TakenOffsetMinutes *int               `json:"taken_offset_minutes,omitempty"` // 服药时间偏移（正=超时，负=提前）
	Items              []SlotMedicineItem `json:"items"`                          // 该时间段的药品明细
}

// SlotMedicineItem 是时间段中的单个药品明细。
type SlotMedicineItem struct {
	MedicineName string  `json:"medicine_name"` // 药品名称
	Dosage       float64 `json:"dosage"`         // 剂量
	Unit         string  `json:"unit"`           // 单位
	Taken        *bool   `json:"taken"`          // 是否已服用
}

// OverviewResponse 是 GET /dashboard/overview 的响应体。
// 家庭首页使用，展示所有老人状态概览。
type OverviewResponse struct {
	FamilyName string          `json:"family_name"` // 家庭组名称
	Elders     []ElderOverview `json:"elders"`      // 所有老人概览
}

// ElderOverview 是家庭概览中单个老人的状态卡片数据。
type ElderOverview struct {
	UserID        uint64                `json:"user_id"`                  // 老人用户 ID
	Nickname      string                `json:"nickname"`                 // 老人昵称
	AvatarURL     string                `json:"avatar_url"`               // 老人头像
	OverallStatus string                `json:"overall_status"`           // 整体状态：normal / warning / danger
	Summary       string                `json:"summary"`                 // 摘要文本，如"2/5 已确认"
	Confirmed     int                   `json:"confirmed"`               // 已确认数
	Total         int                   `json:"total"`                   // 总提醒数
	NextReminder  *NextReminderResponse `json:"next_reminder,omitempty"` // 下一次用药提醒
}

// NextReminderResponse 是下一次用药提醒的信息。
// 仅在存在未来提醒时返回。
type NextReminderResponse struct {
	TimeOfDay string `json:"time_of_day"` // 时间（HH:MM）
	TimeLabel string `json:"time_label"`  // 时段标签，如"今晚 19:00"
	Medicines string `json:"medicines"`   // 药品列表文本，如"阿司匹林 1粒, 维生素C 2片"
}

// NextReminderInfo 下一次用药提醒的内部跟踪数据结构。
// 包含时间对象用于比较排序，不对外暴露。
type NextReminderInfo struct {
	NextTime  time.Time // 提醒时间（仅用于内部比较找最近的一个）
	TimeOfDay string    // HH:MM 格式
	TimeLabel string    // 中文标签
	Medicines string    // 药品文本
}
