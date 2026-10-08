package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/middleware"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/service"
)

// ScheduleHandler 处理用药计划相关的 HTTP 端点：
//   - POST   /api/v1/families/:fid/schedules      —— 创建用药计划
//   - GET    /api/v1/families/:fid/schedules      —— 用药计划列表
//   - PUT    /api/v1/families/:fid/schedules/:id  —— 更新用药计划
//   - DELETE /api/v1/families/:fid/schedules/:id  —— 删除用药计划
//
// 用药计划是定时提醒生成的基础数据，定义了哪个老人、在什么时间、需要服用哪些药品。
// 所有端点都需要家庭组成员身份校验，老人角色通常只能查看（Query 操作）。
type ScheduleHandler struct {
	scheduleService *service.ScheduleService // 用药计划业务逻辑
}

// NewScheduleHandler 创建 ScheduleHandler 实例。
func NewScheduleHandler(scheduleService *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleService: scheduleService}
}

// Create 处理 POST /api/v1/families/:fid/schedules —— 创建用药计划。
// 为指定老人创建包含多个药品明细的用药计划。
// 会校验老人是否属于该家庭组且角色为 elder。
func (h *ScheduleHandler) Create(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}

	var req service.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请填写用药计划信息")
		return
	}

	result, err := h.scheduleService.Create(c.Request.Context(), familyID, userID, &req)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// List 处理 GET /api/v1/families/:fid/schedules —— 查询用药计划列表。
// 支持按老人筛选（elder_user_id 查询参数，传 0 或不传表示查所有）。
func (h *ScheduleHandler) List(c *gin.Context) {
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}

	// elder_user_id 可选参数，0 表示不过滤
	elderUserID, _ := strconv.ParseUint(c.DefaultQuery("elder_user_id", "0"), 10, 64)

	result, err := h.scheduleService.List(c.Request.Context(), familyID, elderUserID)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// Update 处理 PUT /api/v1/families/:fid/schedules/:id —— 更新用药计划。
// 支持修改用药时间、启用/停用状态，以及替换药品明细列表。
// 传入 items 时会先清空旧明细再批量创建新明细。
func (h *ScheduleHandler) Update(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	scheduleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的计划ID")
		return
	}

	var req service.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "参数错误")
		return
	}

	if err := h.scheduleService.Update(c.Request.Context(), familyID, userID, scheduleID, &req); err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.SuccessMessage(c, "用药计划已更新")
}

// Delete 处理 DELETE /api/v1/families/:fid/schedules/:id —— 删除用药计划。
// 会同时删除该计划下的所有药品明细。
func (h *ScheduleHandler) Delete(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	scheduleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的计划ID")
		return
	}

	if err := h.scheduleService.Delete(c.Request.Context(), familyID, userID, scheduleID); err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.SuccessMessage(c, "用药计划已删除")
}

// --- 个人用药计划端点 ---

func (h *ScheduleHandler) CreatePersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	var req service.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, errcode.InvalidParam, "请填写计划信息"); return }
	result, err := h.scheduleService.CreatePersonal(c.Request.Context(), userID, &req)
	if err != nil { response.Error(c, errcode.InternalError, err.Error()); return }
	response.Success(c, result)
}

func (h *ScheduleHandler) ListPersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	result, err := h.scheduleService.ListPersonal(c.Request.Context(), userID)
	if err != nil { response.Error(c, errcode.InternalError, err.Error()); return }
	response.Success(c, result)
}

func (h *ScheduleHandler) UpdatePersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	scheduleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { response.Error(c, errcode.InvalidParam, "无效的计划ID"); return }
	var req service.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, errcode.InvalidParam, "请填写计划信息"); return }
	if err := h.scheduleService.UpdatePersonal(c.Request.Context(), userID, scheduleID, &req); err != nil { response.Error(c, errcode.InternalError, err.Error()); return }
	response.SuccessMessage(c, "已更新")
}

func (h *ScheduleHandler) DeletePersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	scheduleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { response.Error(c, errcode.InvalidParam, "无效的计划ID"); return }
	if err := h.scheduleService.DeletePersonal(c.Request.Context(), userID, scheduleID); err != nil { response.Error(c, errcode.InternalError, err.Error()); return }
	response.SuccessMessage(c, "已删除")
}

// Calendar 处理 GET /api/v1/families/:fid/schedules/calendar
func (h *ScheduleHandler) Calendar(c *gin.Context) {
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil { response.Error(c, errcode.InvalidParam, "无效的家庭组ID"); return }
	year, _ := strconv.Atoi(c.DefaultQuery("year", "0"))
	month, _ := strconv.Atoi(c.DefaultQuery("month", "0"))
	if year < 2000 || month < 1 || month > 12 { response.Error(c, errcode.InvalidParam, "请指定 year 和 month"); return }
	result, err := h.scheduleService.GetCalendar(c.Request.Context(), familyID, 0, year, month)
	if err != nil { response.Error(c, errcode.InternalError, err.Error()); return }
	response.Success(c, result)
}

// CalendarPersonal 处理 GET /api/v1/personal/schedules/calendar
func (h *ScheduleHandler) CalendarPersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	year, _ := strconv.Atoi(c.DefaultQuery("year", "0"))
	month, _ := strconv.Atoi(c.DefaultQuery("month", "0"))
	if year < 2000 || month < 1 || month > 12 { response.Error(c, errcode.InvalidParam, "请指定 year 和 month"); return }
	result, err := h.scheduleService.GetPersonalCalendar(c.Request.Context(), userID, year, month)
	if err != nil { response.Error(c, errcode.InternalError, err.Error()); return }
	response.Success(c, result)
}
