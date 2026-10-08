package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/middleware"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/service"
)

// FamilyHandler 处理家庭组相关的 HTTP 端点：
//   - POST   /api/v1/families              —— 创建家庭组
//   - GET    /api/v1/families              —— 我的家庭组列表
//   - GET    /api/v1/families/:id          —— 家庭组详情
//   - POST   /api/v1/families/:id/join     —— 通过邀请码加入
//   - PUT    /api/v1/families/:id/members/:uid/role —— 设置成员角色
//   - DELETE /api/v1/families/:id/members/:uid      —— 移除成员
//   - DELETE /api/v1/families/:id/leave             —— 退出家庭组
//
// 除 List 和 Create 外，其他端点通过 FamilyMiddleware + RoleMiddleware 保护，
// 会校验当前用户是否属于目标家庭组以及是否有足够的权限。
type FamilyHandler struct {
	familyService *service.FamilyService // 家庭组业务逻辑
}

// NewFamilyHandler 创建 FamilyHandler 实例。
func NewFamilyHandler(familyService *service.FamilyService) *FamilyHandler {
	return &FamilyHandler{familyService: familyService}
}

// Create 处理 POST /api/v1/families —— 创建家庭组。
// 创建者自动成为 owner（最高权限角色），系统生成6位邀请码供其他成员加入。
func (h *FamilyHandler) Create(c *gin.Context) {
	userID := middleware.MustGetUserID(c)

	var req service.CreateFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "家庭组名称不能为空")
		return
	}

	result, err := h.familyService.Create(c.Request.Context(), userID, req.Name)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// List 处理 GET /api/v1/families —— 列出当前用户所在的所有家庭组。
// 每个家庭组项包含成员数、当前用户角色和老人列表。
func (h *FamilyHandler) List(c *gin.Context) {
	userID := middleware.MustGetUserID(c)

	result, err := h.familyService.List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// Detail 处理 GET /api/v1/families/:id —— 获取家庭组详情。
// 返回完整成员列表，包含每个成员的角色、昵称和头像。
func (h *FamilyHandler) Detail(c *gin.Context) {
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}

	result, err := h.familyService.Detail(c.Request.Context(), familyID)
	if err != nil {
		response.Error(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, result)
}

// Join 处理 POST /api/v1/families/:id/join —— 通过邀请码加入家庭组。
// 加入后默认获得 member（普通成员）角色。
// 重复加入会返回 409 Conflict。
func (h *FamilyHandler) Join(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}

	var req service.JoinFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "邀请码不能为空")
		return
	}

	msg, err := h.familyService.Join(c.Request.Context(), userID, req.InviteCode)
	if err != nil {
		// 根据具体错误信息映射 HTTP 状态码
		// 重复加入返回 409 Conflict
		if err.Error() == "您已在该家庭组中" {
			response.Error(c, errcode.Conflict, err.Error())
			return
		}
		// 邀请码无效返回专门错误码
		response.Error(c, errcode.InviteCodeInvalid, err.Error())
		return
	}

	// 标记变量已使用（避免编译器警告）
	_ = familyID

	response.SuccessMessage(c, msg)
}

// JoinByCode 处理 POST /api/v1/families/join —— 通过邀请码加入家庭组（无需已是成员）。
func (h *FamilyHandler) JoinByCode(c *gin.Context) {
	userID := middleware.MustGetUserID(c)

	var req service.JoinFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "邀请码不能为空")
		return
	}

	msg, err := h.familyService.Join(c.Request.Context(), userID, req.InviteCode)
	if err != nil {
		if err.Error() == "您已在该家庭组中" {
			response.Error(c, errcode.Conflict, err.Error())
			return
		}
		response.Error(c, errcode.InviteCodeInvalid, err.Error())
		return
	}

	response.SuccessMessage(c, msg)
}

// RefreshInvite 处理 POST /api/v1/families/:fid/refresh-invite —— 刷新邀请码。
func (h *FamilyHandler) RefreshInvite(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	newCode, err := h.familyService.RefreshInviteCode(c.Request.Context(), familyID, userID)
	if err != nil {
		response.Error(c, errcode.Forbidden, err.Error())
		return
	}
	response.Success(c, map[string]string{"invite_code": newCode})
}

// SetMemberRole 处理 PUT /api/v1/families/:id/members/:uid/role —— 设置成员角色。
// 仅 owner 和 admin 可以操作，操作结果受 service 层进一步校验。
func (h *FamilyHandler) SetMemberRole(c *gin.Context) {
	operatorUserID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	targetUserID, err := strconv.ParseUint(c.Param("uid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的用户ID")
		return
	}

	var req service.SetRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "角色不能为空")
		return
	}

	if err := h.familyService.SetMemberRole(c.Request.Context(), familyID, operatorUserID, targetUserID, req.Role); err != nil {
		// 权限不足返回 403 Forbidden
		response.Error(c, errcode.Forbidden, err.Error())
		return
	}

	response.SuccessMessage(c, "角色修改成功")
}

// RemoveMember 处理 DELETE /api/v1/families/:id/members/:uid —— 移除成员。
// 仅 owner 和 admin 可以移除他人，不允许移除自己。
func (h *FamilyHandler) RemoveMember(c *gin.Context) {
	operatorUserID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	targetUserID, err := strconv.ParseUint(c.Param("uid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的用户ID")
		return
	}

	if err := h.familyService.RemoveMember(c.Request.Context(), familyID, operatorUserID, targetUserID); err != nil {
		response.Error(c, errcode.Forbidden, err.Error())
		return
	}

	response.SuccessMessage(c, "成员已移除")
}

// Leave 处理 DELETE /api/v1/families/:id/leave —— 主动退出家庭组。
// owner 不能直接退出，需要先转让 ownership。
func (h *FamilyHandler) Leave(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}

	if err := h.familyService.Leave(c.Request.Context(), familyID, userID); err != nil {
		response.Error(c, errcode.Forbidden, err.Error())
		return
	}

	response.SuccessMessage(c, "已退出家庭组")
}
