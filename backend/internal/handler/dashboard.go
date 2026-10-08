package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/middleware"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/service"
)

// DashboardHandler 处理看板相关的 HTTP 端点：
//   - GET /api/v1/dashboard/today    —— 今日用药看板（针对单个老人）
//   - GET /api/v1/dashboard/overview —— 家庭概览（所有老人状态汇总）
//
// 看板是子女端小程序的首页和老人详情页的核心数据来源。
type DashboardHandler struct {
	dashboardService *service.DashboardService // 看板数据聚合逻辑
}

// NewDashboardHandler 创建 DashboardHandler 实例。
func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetToday 处理 GET /api/v1/dashboard/today —— 获取指定老人今日的用药看板。
//
// 必填查询参数：
//   - family_id: 家庭组 ID
//   - elder_user_id: 老人用户 ID
//
// 返回数据包含：
//   - 汇总统计（总数/已确认/待处理/未确认/完成率）
//   - 各个时间段的药品服用状态详情
//
// 场景：子女在首页点击某个老人后进入的详细看板页面。
func (h *DashboardHandler) GetToday(c *gin.Context) {
	// 获取当前用户 ID（用于日志追踪，实际业务中使用 family_id 和 elder_user_id）
	_ = middleware.MustGetUserID(c)

	familyID, err := strconv.ParseUint(c.Query("family_id"), 10, 64)
	if err != nil || familyID == 0 {
		response.Error(c, errcode.InvalidParam, "请指定家庭组")
		return
	}
	elderUserID, err := strconv.ParseUint(c.Query("elder_user_id"), 10, 64)
	if err != nil || elderUserID == 0 {
		response.Error(c, errcode.InvalidParam, "请指定老人")
		return
	}

	result, err := h.dashboardService.GetTodayDashboard(c.Request.Context(), familyID, elderUserID)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetOverview 处理 GET /api/v1/dashboard/overview —— 获取家庭组所有老人的用药概览。
//
// 必填查询参数：
//   - family_id: 家庭组 ID
//
// 返回数据包含每个老人的：
//   - 整体状态卡片（normal / warning / danger）
//   - 确认数/总数 摘要
//   - 下一次用药提醒（如果还有未到时间的提醒）
//
// 场景：家庭首页，子女可以一览所有老人的服药完成情况。
// warning 表示有超时未确认，danger 表示超时超过60分钟。
func (h *DashboardHandler) GetOverview(c *gin.Context) {
	_ = middleware.MustGetUserID(c)

	familyID, err := strconv.ParseUint(c.Query("family_id"), 10, 64)
	if err != nil || familyID == 0 {
		response.Error(c, errcode.InvalidParam, "请指定家庭组")
		return
	}

	result, err := h.dashboardService.GetOverview(c.Request.Context(), familyID)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}
