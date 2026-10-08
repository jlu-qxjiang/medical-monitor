package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/middleware"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/service"
)

// ReminderHandler 处理用药提醒交互相关的 HTTP 端点：
//   - GET  /api/v1/reminders/today         —— 今日提醒列表
//   - POST /api/v1/reminders/:id/confirm   —— 确认服药
//   - POST /api/v1/reminders/:id/miss      —— 标记未服用
//   - POST /api/v1/reminders/:id/skip      —— 跳过本次用药
//   - POST /api/v1/reminders/:id/revoke    —— 撤销确认（5分钟内）
//
// 这些端点主要供老人端（小程序老人侧）使用。
// 只有老人本人可以进行服药确认操作（service 层有权限校验）。
type ReminderHandler struct {
	reminderService *service.ReminderService // 提醒交互业务逻辑
}

// NewReminderHandler 创建 ReminderHandler 实例。
func NewReminderHandler(reminderService *service.ReminderService) *ReminderHandler {
	return &ReminderHandler{reminderService: reminderService}
}

// GetToday 处理 GET /api/v1/reminders/today —— 获取当前用户今日的用药提醒。
// family_id 为可选查询参数，用于限定家庭组范围。
// 返回数据按时间段分组，每个时间段包含药品明细和服用状态。
func (h *ReminderHandler) GetToday(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, _ := strconv.ParseUint(c.DefaultQuery("family_id", "0"), 10, 64)

	result, err := h.reminderService.GetToday(c.Request.Context(), userID, familyID)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// Confirm 处理 POST /api/v1/reminders/:id/confirm —— 确认已服用药品。
// 支持部分确认（可以只确认某些药品，不必全选）。
// 返回撤销截止时间，前端用此展示5分钟倒计时撤销按钮。
func (h *ReminderHandler) Confirm(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	reminderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的提醒ID")
		return
	}

	var req service.ConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请指定确认的药品")
		return
	}

	result, err := h.reminderService.Confirm(c.Request.Context(), userID, reminderID, req.ItemIDs)
	if err != nil {
		// 权限不足或状态不对返回 403
		response.Error(c, errcode.Forbidden, err.Error())
		return
	}

	// 返回 ConfirmResponse（含撤销截止时间）
	response.Success(c, result)
}

// MarkMiss 处理 POST /api/v1/reminders/:id/miss —— 标记本次用药未服用。
// 老人忘记服药时使用，需要提供未服用原因。
// 标记后相关药品的库存不会被扣减（不同于确认操作）。
func (h *ReminderHandler) MarkMiss(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	reminderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的提醒ID")
		return
	}

	var req service.MissRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请选择原因")
		return
	}

	if err := h.reminderService.MarkMissed(c.Request.Context(), userID, reminderID, req.Reason); err != nil {
		response.Error(c, errcode.Forbidden, err.Error())
		return
	}

	response.SuccessMessage(c, "已标记为未服用")
}

// Skip 处理 POST /api/v1/reminders/:id/skip —— 跳过本次用药。
// 适用于医生建议暂停或老人当天身体不适等场景。
// 需要提供跳过原因。
func (h *ReminderHandler) Skip(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	reminderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的提醒ID")
		return
	}

	var req service.SkipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请选择原因")
		return
	}

	if err := h.reminderService.Skip(c.Request.Context(), userID, reminderID, req.Reason); err != nil {
		response.Error(c, errcode.Forbidden, err.Error())
		return
	}

	response.SuccessMessage(c, "已跳过本次用药")
}

// Revoke 处理 POST /api/v1/reminders/:id/revoke —— 撤销服药确认。
// 仅在确认后5分钟内有效，超时返回错误。
// 撤销后会恢复库存并重置服用状态。
func (h *ReminderHandler) Revoke(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	reminderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的提醒ID")
		return
	}

	if err := h.reminderService.Revoke(c.Request.Context(), userID, reminderID); err != nil {
		response.Error(c, errcode.InvalidParam, err.Error())
		return
	}

	response.SuccessMessage(c, "已撤销确认")
}
