// Package handler 是 HTTP 请求处理层（Controller 层）。
package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/middleware"
	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/repository"
)

// StockHandler 处理药品库存（余量）相关的 HTTP 端点：
//   - GET  /api/v1/families/:fid/stocks          —— 查询家庭库存列表
//   - PUT  /api/v1/families/:fid/stocks/:id      —— 手动校准家庭库存
//   - PUT  /api/v1/personal/medicines/:id/stock  —— 手动校准个人药品库存
//
// 库存端点直接操作 Repository 层而非通过 Service 层，
// 因为它主要做数据查询和简单的字段更新，业务逻辑相对简单。
type StockHandler struct {
	medicineRepo *repository.MedicineRepository // 药品及库存数据操作
	auditRepo    *repository.AuditRepository    // 审计日志
}

// NewStockHandler 创建 StockHandler 实例。
func NewStockHandler(medicineRepo *repository.MedicineRepository, auditRepo *repository.AuditRepository) *StockHandler {
	return &StockHandler{
		medicineRepo: medicineRepo,
		auditRepo:    auditRepo,
	}
}

// List 处理 GET /api/v1/families/:fid/stocks —— 查询家庭组的药品库存列表。
//
// 可选查询参数：
//   - status: 过滤库存状态（low=低于阈值, empty=已用完, 不传=全部）
//
// 返回每个药品的库存快照，包括：
//   - 剩余数量和总数量
//   - 预警状态标签
//   - 预估剩余天数（基于每日3粒的保守估计）
func (h *StockHandler) List(c *gin.Context) {
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}

	// status 可选过滤：normal / low / empty
	statusFilter := c.Query("status")

	stocks, err := h.medicineRepo.FindStocksByFamilyID(c.Request.Context(), familyID, statusFilter)
	if err != nil {
		response.Error(c, errcode.InternalError, "查询余量失败: "+err.Error())
		return
	}

	list := make([]StockListItem, 0, len(stocks))
	for _, stock := range stocks {
		// 查询药品名称用于展示（药品已删除则跳过该库存记录）
		medicine, err := h.medicineRepo.FindMedicineByID(c.Request.Context(), familyID, stock.MedicineID)
		if err != nil {
			continue
		}
		medicineName := medicine.Name

		// 计算库存状态
		status := model.StockStatusNormal
		statusLabel := ""
		if stock.RemainingCount == 0 {
			status = model.StockStatusEmpty
			statusLabel = "已用完"
		} else if stock.RemainingCount <= stock.AlertThreshold {
			status = model.StockStatusLow
			statusLabel = fmt.Sprintf("即将用完（剩余 %d 粒）", stock.RemainingCount)
		}

		// 预估剩余天数（按每天3粒计算，保守估计）
		estimatedDaysLeft := float64(0)
		if stock.RemainingCount > 0 {
			estimatedDaysLeft = float64(stock.RemainingCount) / 3.0
		}

		list = append(list, StockListItem{
			MedicineID:        stock.MedicineID,
			MedicineName:      medicineName,
			RemainingCount:    stock.RemainingCount,
			TotalCount:        stock.TotalCount,
			AlertThreshold:    stock.AlertThreshold,
			Status:            status,
			StatusLabel:       statusLabel,
			EstimatedDaysLeft: estimatedDaysLeft,
		})
	}

	response.Success(c, map[string]interface{}{
		"list": list,
	})
}

// Update 处理 PUT /api/v1/families/:fid/stocks/:id —— 手动校准库存。
//
// 场景：子女或老人补充药品后手动更新库存，或定期核对实际数量后校准。
// 更新后会自动记录 refill 时间和审计日志。
// 注意：此处的 stock ID 就是 medicine ID（一对一的库存记录）。
func (h *StockHandler) Update(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	familyID, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的家庭组ID")
		return
	}
	stockID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的余量记录ID")
		return
	}

	var req StockUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "参数错误")
		return
	}

	ctx := c.Request.Context()
	var stock model.MedicineStock
	// 尝试通过 medicine_id 查找库存记录
	stockPtr, err := h.medicineRepo.FindStockByMedicineID(ctx, stockID)
	if err != nil || stockPtr == nil {
		// 备用路径：如果找不到，尝试将 stockID 当作 medicine 的 ID
		// 查询该 medicine 是否存在，再通过 medicine.ID 查库存
		medicine, merr := h.medicineRepo.FindMedicineByID(ctx, familyID, stockID)
		if merr != nil {
			response.Error(c, errcode.NotFound, "余量记录不存在")
			return
		}
		s, serr := h.medicineRepo.FindStockByMedicineID(ctx, medicine.ID)
		if serr != nil {
			response.Error(c, errcode.NotFound, "余量记录不存在")
			return
		}
		stock = *s
	} else {
		stock = *stockPtr
	}

	// 更新库存数量和记录补充时间
	stock.RemainingCount = req.RemainingCount
	stock.TotalCount = req.TotalCount
	now := time.Now()
	stock.LastRefillAt = &now // 记录最近一次补充时间

	if err := h.medicineRepo.UpdateStock(ctx, &stock); err != nil {
		response.Error(c, errcode.InternalError, "更新余量失败: "+err.Error())
		return
	}

	// 审计日志
	uid := userID
	fid := familyID
	sid := stockID
	_ = h.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionStockUpdate, model.AuditTargetStock, &sid,
		fmt.Sprintf(`{"remaining_count":%d,"total_count":%d}`, req.RemainingCount, req.TotalCount), "")

	response.SuccessMessage(c, "余量已更新")
}

// UpdatePersonalStock 处理 PUT /api/v1/personal/medicines/:id/stock —— 手动校准个人药品库存。
//
// 场景：个人补充药品后手动更新库存，或定期核对实际数量后校准。
// 与家庭库存（Update）的区别：按 user_id 归属校验，库存不存在时自动创建。
//
// 流程：
//  1. 校验药品属于当前用户（防止越权修改他人药品库存）
//  2. 查找库存记录，不存在则自动创建（保证库存记录一定存在）
//  3. 更新剩余数量和总数量
//  4. 记录审计日志
func (h *StockHandler) UpdatePersonalStock(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	medID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParam, "无效的药品ID")
		return
	}

	var req StockUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParam, "参数错误")
		return
	}

	ctx := c.Request.Context()

	// 校验药品属于当前用户
	if _, err := h.medicineRepo.FindPersonalMedicineByID(ctx, userID, medID); err != nil {
		response.Error(c, errcode.NotFound, "药品不存在")
		return
	}

	// 查库存，不存在则创建
	stock, err := h.medicineRepo.FindPersonalStock(ctx, medID)
	if err != nil {
		stock = &model.PersonalMedicineStock{MedicineID: medID}
		if err := h.medicineRepo.CreatePersonalStock(ctx, stock); err != nil {
			response.Error(c, errcode.InternalError, "创建库存失败: "+err.Error())
			return
		}
	}

	// 更新库存数量
	stock.RemainingCount = req.RemainingCount
	stock.TotalCount = req.TotalCount

	if err := h.medicineRepo.UpdatePersonalStock(ctx, stock); err != nil {
		response.Error(c, errcode.InternalError, "更新余量失败: "+err.Error())
		return
	}

	// 审计日志
	uid := userID
	sid := medID
	_ = h.auditRepo.LogAction(ctx, &uid, nil, model.AuditActionStockUpdate, model.AuditTargetStock, &sid,
		fmt.Sprintf(`{"remaining_count":%d,"total_count":%d}`, req.RemainingCount, req.TotalCount), "")

	response.SuccessMessage(c, "余量已更新")
}

// StockListItem 库存列表中的单项数据。
// estimated_days_left 基于每日3粒的保守估计。
type StockListItem struct {
	MedicineID        uint64  `json:"medicine_id"`         // 药品 ID
	MedicineName      string  `json:"medicine_name"`       // 药品名称
	RemainingCount    int     `json:"remaining_count"`     // 剩余数量（粒/片）
	TotalCount        int     `json:"total_count"`         // 总容量
	AlertThreshold    int     `json:"alert_threshold"`     // 预警阈值
	Status            string  `json:"status"`              // 状态：normal / low / empty
	StatusLabel       string  `json:"status_label"`        // 状态描述文本
	EstimatedDaysLeft float64 `json:"estimated_days_left"` // 预估剩余天数
}

// StockUpdateRequest 是库存更新请求体（家庭 + 个人通用）。
// remaining_count 和 total_count 都必须 >= 0。
type StockUpdateRequest struct {
	RemainingCount int `json:"remaining_count" binding:"required,min=0"` // 剩余数量（实际数量）
	TotalCount     int `json:"total_count" binding:"required,min=0"`     // 总数量（一瓶/一盒的总容量）
}
