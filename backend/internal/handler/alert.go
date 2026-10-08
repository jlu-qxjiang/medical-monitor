package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/pkg/utils"
	"github.com/medical-monitor/internal/repository"
)

// AlertHandler 处理告警记录查询。
type AlertHandler struct {
	reminderRepo *repository.ReminderRepository
}

func NewAlertHandler(reminderRepo *repository.ReminderRepository) *AlertHandler {
	return &AlertHandler{reminderRepo: reminderRepo}
}

// List 处理 GET /api/v1/alerts —— 查询告警记录。
// 默认只返回今天的告警，避免历史数据干扰。
func (h *AlertHandler) List(c *gin.Context) {
	familyID, err := strconv.ParseUint(c.Query("family_id"), 10, 64)
	if err != nil || familyID == 0 {
		response.Error(c, errcode.InvalidParam, "请指定家庭组")
		return
	}

	elderUserID, _ := strconv.ParseUint(c.Query("elder_user_id"), 10, 64)
	date := c.Query("date")
	if date == "" {
		date = utils.Today() // 不传日期默认今天
	}

	reminders, err := h.reminderRepo.FindAlerts(c.Request.Context(), familyID, elderUserID, date)
	if err != nil {
		response.Error(c, errcode.InternalError, "查询告警记录失败: "+err.Error())
		return
	}

	list := make([]AlertItem, 0, len(reminders))
	for _, r := range reminders {
		alertLabel := h.getAlertLevelLabel(r.AlertLevel)
		medicines := make([]string, 0, len(r.Items))
		for _, item := range r.Items {
			medicines = append(medicines, fmt.Sprintf("%s %.0f%s", item.MedicineName, item.Dosage, item.Unit))
		}

		resolved := r.Status == model.ReminderStatusConfirmed || r.Status == model.ReminderStatusSkipped
		var resolvedAt *string
		var resolvedOffset *int
		if resolved && r.ConfirmedAt != nil {
			s := r.ConfirmedAt.Format("2006-01-02T15:04:05Z")
			resolvedAt = &s
			offset := utils.TakenOffsetMinutes(r.RemindTime, *r.ConfirmedAt)
			resolvedOffset = &offset
		}

		list = append(list, AlertItem{
			ReminderID:            r.ID,
			AlertLevel:            r.AlertLevel,
			AlertLabel:            alertLabel,
			RemindTime:            r.RemindTime,
			AlertTime:             r.CreatedAt,
			Medicines:             strings.Join(medicines, "、"),
			Resolved:              resolved,
			ResolvedAt:            resolvedAt,
			ResolvedOffsetMinutes: resolvedOffset,
		})
	}

	response.Success(c, map[string]interface{}{"list": list})
}

// getAlertLevelLabel 告警级别 → 中文标签。
// 1=10分钟 / 2=30分钟
func (h *AlertHandler) getAlertLevelLabel(level int8) string {
	switch level {
	case 1:
		return "10分钟未确认"
	case 2:
		return "30分钟未确认"
	default:
		return "未知"
	}
}

type AlertItem struct {
	ReminderID            uint64  `json:"reminder_id"`
	AlertLevel            int8    `json:"alert_level"`
	AlertLabel            string  `json:"alert_label"`
	RemindTime            any     `json:"remind_time"`
	AlertTime             any     `json:"alert_time"`
	Medicines             string  `json:"medicines"`
	Resolved              bool    `json:"resolved"`
	ResolvedAt            *string `json:"resolved_at"`
	ResolvedOffsetMinutes *int    `json:"resolved_offset_minutes,omitempty"` // 服药时间偏移（正=超时，负=提前）
}
