package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/middleware"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/service"
)

// MedicineHandler 处理药品管理相关的 HTTP 端点：
//   - POST   /api/v1/families/:fid/medicines      —— 添加药品
//   - GET    /api/v1/families/:fid/medicines      —— 药品列表（支持搜索和分页）
//   - GET    /api/v1/families/:fid/medicines/:id  —— 药品详情
//   - PUT    /api/v1/families/:fid/medicines/:id  —— 更新药品
//   - DELETE /api/v1/families/:fid/medicines/:id  —— 删除药品
//   - POST   /api/v1/personal/medicines           —— 添加个人药品
//   - GET    /api/v1/personal/medicines           —— 个人药品列表
//   - GET    /api/v1/personal/medicines/:id       —— 个人药品详情
//   - PUT    /api/v1/personal/medicines/:id       —— 更新个人药品
//   - DELETE /api/v1/personal/medicines/:id       —— 删除个人药品
//
// 家庭端点需要通过 FamilyMiddleware 校验家庭组成员身份，
// 通过 RoleMiddleware 限制老人角色的操作权限。
// 个人端点按 user_id 归属，仅需 JWT 鉴权。
type MedicineHandler struct {
	medicineService *service.MedicineService // 药品业务逻辑
}

// NewMedicineHandler 创建 MedicineHandler 实例。
func NewMedicineHandler(medicineService *service.MedicineService) *MedicineHandler {
	return &MedicineHandler{medicineService: medicineService}
}

// Create 处理 POST /api/v1/families/:fid/medicines —— 添加药品。
// 支持同时初始化库存（initial_stock 可选字段）。
func (h *MedicineHandler) Create(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}

	var req service.CreateMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请填写药品信息")
		return
	}

	result, err := h.medicineService.Create(c.Request.Context(), familyID, userID, &req)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// List 处理 GET /api/v1/families/:fid/medicines —— 药品搜索/列表。
// 支持关键词搜索（keyword 查询参数）和分页（page / page_size）。
// 返回列表包含库存状态和是否有活跃用药计划。
func (h *MedicineHandler) List(c *gin.Context) {
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}

	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.medicineService.List(c.Request.Context(), familyID, keyword, page, pageSize)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// Detail 处理 GET /api/v1/families/:fid/medicines/:id —— 药品详情。
// 返回药品完整信息，包括当前库存快照。
func (h *MedicineHandler) Detail(c *gin.Context) {
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	medicineID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的药品ID")
		return
	}

	result, err := h.medicineService.Detail(c.Request.Context(), familyID, medicineID)
	if err != nil {
		response.Error(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, result)
}

// Update 处理 PUT /api/v1/families/:fid/medicines/:id —— 更新药品信息。
// 支持增量更新：仅更新请求中传入的非空字段。
func (h *MedicineHandler) Update(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	medicineID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的药品ID")
		return
	}

	var req service.UpdateMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "参数错误")
		return
	}

	if err := h.medicineService.Update(c.Request.Context(), familyID, userID, medicineID, &req); err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.SuccessMessage(c, "药品信息已更新")
}

// Delete 处理 DELETE /api/v1/families/:fid/medicines/:id —— 删除药品。
// 如果药品关联了活跃的用药计划，删除会被拒绝，返回专门错误码。
func (h *MedicineHandler) Delete(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	medicineID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的药品ID")
		return
	}

	if err := h.medicineService.Delete(c.Request.Context(), familyID, userID, medicineID); err != nil {
		// 药品有关联计划时返回特定的业务错误码
		if err.Error() == "药品已关联用药计划，无法删除" {
			response.Error(c, errcode.MedicineHasSchedule, err.Error())
			return
		}
		response.Error(c, errcode.InternalError, err.Error())
		return
	}

	response.SuccessMessage(c, "药品已删除")
}

// --- 个人药品端点 ---

// ListPersonal 处理 GET /api/v1/personal/medicines —— 个人药品列表。
// 支持关键词搜索（keyword 查询参数）。
// 返回列表包含每个药品的库存余量。
func (h *MedicineHandler) ListPersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	keyword := c.DefaultQuery("keyword", "")
	list, err := h.medicineService.ListPersonal(c.Request.Context(), userID, keyword)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}
	response.Success(c, map[string]interface{}{"list": list})
}

// CreatePersonal 处理 POST /api/v1/personal/medicines —— 添加个人药品。
// 支持同时初始化库存（initial_stock 可选字段）。
func (h *MedicineHandler) CreatePersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	var req service.CreatePersonalMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请填写药品信息")
		return
	}
	result, err := h.medicineService.CreatePersonal(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}
	response.Success(c, result)
}

// UpdatePersonal 处理 PUT /api/v1/personal/medicines/:id —— 编辑个人药品。
// 支持增量更新：仅更新请求中传入的非空字段（名称/规格/厂家/图片/备注）。
func (h *MedicineHandler) UpdatePersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	medID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的药品ID")
		return
	}
	var req service.UpdatePersonalMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "请填写药品信息")
		return
	}
	if err := h.medicineService.UpdatePersonal(c.Request.Context(), userID, medID, &req); err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}
	response.SuccessMessage(c, "已更新")
}

// DetailPersonal 处理 GET /api/v1/personal/medicines/:id —— 个人药品详情。
// 返回药品完整信息，包括当前库存快照。
func (h *MedicineHandler) DetailPersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	medID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { response.Error(c, errcode.InvalidParam, "无效的药品ID"); return }
	result, err := h.medicineService.DetailPersonal(c.Request.Context(), userID, medID)
	if err != nil { response.Error(c, errcode.InternalError, err.Error()); return }
	response.Success(c, result)
}

// DeletePersonal 处理 DELETE /api/v1/personal/medicines/:id —— 删除个人药品。
// 如果药品关联了个人用药计划，删除会被拒绝。
func (h *MedicineHandler) DeletePersonal(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	medID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的药品ID")
		return
	}
	if err := h.medicineService.DeletePersonal(c.Request.Context(), userID, medID); err != nil {
		response.Error(c, errcode.InternalError, err.Error())
		return
	}
	response.SuccessMessage(c, "已删除")
}
