package service

import (
	"context"
	"fmt"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/repository"
)

// MedicineService 负责药品管理的业务逻辑，包括：
//   - 药品的增删改查（CRUD）
//   - 药品库存（余量）的初始化与管理
//   - 药品与用药计划的关联校验
//
// 业务规则：
//   - 药品归属于家庭组，同一家庭组的药品对所有成员可见
//   - 删除药品前需检查是否有关联的活跃用药计划，有则禁止删除
//   - 创建药品时可同时初始化库存信息
type MedicineService struct {
	medicineRepo *repository.MedicineRepository // 药品及库存数据持久化
	auditRepo    *repository.AuditRepository    // 审计日志
}

// NewMedicineService 创建 MedicineService 实例。
func NewMedicineService(medicineRepo *repository.MedicineRepository, auditRepo *repository.AuditRepository) *MedicineService {
	return &MedicineService{
		medicineRepo: medicineRepo,
		auditRepo:    auditRepo,
	}
}

// Create 在指定家庭组中添加新药品。
// 支持同时设置初始库存（initial_stock），若未设置 alert_threshold 则默认10粒。
// 业务校验：
//   - 药品名称必填，最长128字符
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - userID: 操作者用户 ID
//   - req: 药品创建请求，包含名称、规格、厂家、图片等
//
// 返回值：
//   - *MedicineResponse: 创建后的药品完整信息
//   - error: 数据校验失败或数据库错误
func (s *MedicineService) Create(ctx context.Context, familyID, userID uint64, req *CreateMedicineRequest) (*MedicineResponse, error) {
	// 构建药品模型
	medicine := &model.Medicine{
		FamilyID:       familyID,
		Name:           req.Name,
		Specification:  req.Specification,
		Manufacturer:   req.Manufacturer,
		ImageURL:       req.ImageURL,
		Notes:          req.Notes,
		Expiry:         req.Expiry,
		ExternalCode:   req.ExternalCode,   // 外部药品编码（如医保编码）
		ExternalSource: req.ExternalSource, // 外部数据来源标记
		CreatedBy:      userID,
	}
	if err := s.medicineRepo.CreateMedicine(ctx, medicine); err != nil {
		return nil, fmt.Errorf("添加药品失败: %w", err)
	}

	// 如果传入了初始库存信息，创建对应的库存记录
	var stock *model.MedicineStock
	if req.InitialStock != nil {
		stock = &model.MedicineStock{
			MedicineID:     medicine.ID,
			FamilyID:       familyID,
			TotalCount:     req.InitialStock.TotalCount,
			RemainingCount: req.InitialStock.TotalCount, // 初始余量 = 总数量
			AlertThreshold: req.InitialStock.AlertThreshold,
		}
		// 默认预警阈值为10粒
		if stock.AlertThreshold == 0 {
			stock.AlertThreshold = 10
		}
		if err := s.medicineRepo.CreateStock(ctx, stock); err != nil {
			return nil, fmt.Errorf("创建余量记录失败: %w", err)
		}
	}

	// 审计日志
	uid := userID
	fid := familyID
	mid := medicine.ID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionMedicineCreate, model.AuditTargetMedicine, &mid, fmt.Sprintf(`{"name":"%s"}`, medicine.Name), "")

	return s.buildMedicineResponse(medicine, stock), nil
}

// List 返回指定家庭组的药品分页列表，支持关键词搜索。
// 每个药品项额外包含：
//   - 库存剩余数量和预警阈值
//   - 是否有关联的活跃用药计划
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - keyword: 搜索关键词（匹配药品名称）
//   - page: 页码（从1开始，默认1）
//   - pageSize: 每页条数（1-50，默认20）
//
// 返回值：
//   - *MedicineListResponse: 分页药品列表
//   - error: 数据库查询错误
func (s *MedicineService) List(ctx context.Context, familyID uint64, keyword string, page, pageSize int) (*MedicineListResponse, error) {
	// 参数归一化：页码最小为1，每页条数限制1-50
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	medicines, total, err := s.medicineRepo.FindMedicinesByFamilyID(ctx, familyID, keyword, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询药品列表失败: %w", err)
	}

	list := make([]MedicineListItem, 0, len(medicines))
	for _, m := range medicines {
		item := MedicineListItem{
			ID:            m.ID,
			Name:          m.Name,
			Specification: m.Specification,
			Manufacturer:  m.Manufacturer,
			ImageURL:      m.ImageURL,
		}
		// 关联查询库存信息
		stock, err := s.medicineRepo.FindStockByMedicineID(ctx, m.ID)
		if err == nil {
			item.StockID = stock.ID
				item.StockRemaining = stock.RemainingCount
				item.StockTotal = stock.TotalCount
			item.StockAlertThreshold = stock.AlertThreshold
		}
		// 检查是否存在活跃的用药计划（用于前端控制删除提示）
		hasSchedule, _ := s.medicineRepo.HasActiveSchedules(ctx, m.ID)
		item.HasSchedule = hasSchedule

		list = append(list, item)
	}

	return &MedicineListResponse{
		List:  list,
		Total: total,
	}, nil
}

// Detail 返回指定药品的完整信息，包含库存快照。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID（用于权限校验）
//   - medicineID: 药品 ID
//
// 返回值：
//   - *MedicineDetailResponse: 药品详细信息
//   - error: 药品不存在或不属于该家庭组
func (s *MedicineService) Detail(ctx context.Context, familyID, medicineID uint64) (*MedicineDetailResponse, error) {
	medicine, err := s.medicineRepo.FindMedicineByID(ctx, familyID, medicineID)
	if err != nil {
		return nil, fmt.Errorf("药品不存在")
	}

	stock, _ := s.medicineRepo.FindStockByMedicineID(ctx, medicineID)

	return &MedicineDetailResponse{
		ID:             medicine.ID,
		Name:           medicine.Name,
		Specification:  medicine.Specification,
		Manufacturer:   medicine.Manufacturer,
		ImageURL:       medicine.ImageURL,
		Notes:          medicine.Notes,
		ExternalCode:   medicine.ExternalCode,
		ExternalSource: medicine.ExternalSource,
		CreatedBy:      medicine.CreatedBy,
		CreatedAt:      medicine.CreatedAt,
		Stock:          s.buildStockInfo(stock),
	}, nil
}

// Update 修改药品信息（增量更新，仅更新非空字段）。
// 不会影响该药品关联的用药计划。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - userID: 操作者
//   - medicineID: 药品 ID
//   - req: 要更新的字段集合
//
// 返回值：错误或 nil
func (s *MedicineService) Update(ctx context.Context, familyID, userID, medicineID uint64, req *UpdateMedicineRequest) error {
	medicine, err := s.medicineRepo.FindMedicineByID(ctx, familyID, medicineID)
	if err != nil {
		return fmt.Errorf("药品不存在")
	}

	// 增量更新：仅替换非空字符串字段
	if req.Name != "" {
		medicine.Name = req.Name
	}
	if req.Specification != "" {
		medicine.Specification = req.Specification
	}
	if req.Manufacturer != "" {
		medicine.Manufacturer = req.Manufacturer
	}
	if req.ImageURL != "" {
		medicine.ImageURL = req.ImageURL
	}
	if req.Notes != "" {
		medicine.Notes = req.Notes
	}

	if err := s.medicineRepo.UpdateMedicine(ctx, medicine); err != nil {
		return fmt.Errorf("更新药品失败: %w", err)
	}

	// 审计日志
	uid := userID
	fid := familyID
	mid := medicineID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionMedicineUpdate, model.AuditTargetMedicine, &mid, fmt.Sprintf(`{"name":"%s"}`, medicine.Name), "")

	return nil
}

// Delete 删除药品。删除前会检查该药品是否关联了活跃的用药计划。
// 如果有关联的活跃计划，拒绝删除并提示用户先删除计划。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - userID: 操作者
//   - medicineID: 药品 ID
//
// 返回值：错误或 nil
func (s *MedicineService) Delete(ctx context.Context, familyID, userID, medicineID uint64) error {
	// 校验药品存在且属于该家庭组
	_, err := s.medicineRepo.FindMedicineByID(ctx, familyID, medicineID)
	if err != nil {
		return fmt.Errorf("药品不存在")
	}

	// 检查是否关联活跃的用药计划（is_active = true 的计划）
	hasSchedule, err := s.medicineRepo.HasActiveSchedules(ctx, medicineID)
	if err != nil {
		return fmt.Errorf("检查用药计划失败: %w", err)
	}
	if hasSchedule {
		return fmt.Errorf("药品已关联用药计划，无法删除")
	}

	if err := s.medicineRepo.DeleteMedicine(ctx, medicineID); err != nil {
		return fmt.Errorf("删除药品失败: %w", err)
	}

	// 审计日志
	uid := userID
	fid := familyID
	mid := medicineID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionMedicineDelete, model.AuditTargetMedicine, &mid, "", "")

	return nil
}

// buildMedicineResponse 将 Medicine 模型和 Stock 模型组装为 API 响应体。
func (s *MedicineService) buildMedicineResponse(medicine *model.Medicine, stock *model.MedicineStock) *MedicineResponse {
	resp := &MedicineResponse{
		ID:             medicine.ID,
		Name:           medicine.Name,
		Specification:  medicine.Specification,
		Manufacturer:   medicine.Manufacturer,
		ImageURL:       medicine.ImageURL,
		Notes:          medicine.Notes,
		ExternalCode:   medicine.ExternalCode,
		ExternalSource: medicine.ExternalSource,
		CreatedAt:      medicine.CreatedAt,
	}
	if stock != nil {
		resp.Stock = s.buildStockInfo(stock)
	}
	return resp
}

// buildStockInfo 将 MedicineStock 模型转为 API 响应结构。
func (s *MedicineService) buildStockInfo(stock *model.MedicineStock) *StockInfo {
	if stock == nil {
		return nil
	}
	return &StockInfo{
		TotalCount:     stock.TotalCount,     // 总数量
		RemainingCount: stock.RemainingCount, // 剩余数量
		AlertThreshold: stock.AlertThreshold, // 预警阈值
	}
}

// --- Medicine request/response 类型定义 ---

// CreateMedicineRequest 是 POST /families/:fid/medicines 的请求体。
// initial_stock 可选，用于创建药品的同时初始化库存。
type CreateMedicineRequest struct {
	Name           string            `json:"name" binding:"required,max=128"` // 药品通用名，必填
	Expiry         string            `json:"expiry"`                           // 有效期（YYYY-MM-DD）
	Specification  string            `json:"specification"`                    // 规格，如"5mg*30片"
	Manufacturer   string            `json:"manufacturer"`                     // 生产厂家
	ImageURL       string            `json:"image_url"`                        // 药品图片 URL
	Notes          string            `json:"notes"`                            // 备注，如服用注意事项
	ExternalCode   string            `json:"external_code"`                    // 外部编码（医保编码等）
	ExternalSource string            `json:"external_source"`                  // 外部来源标记
	InitialStock   *InitialStockInfo `json:"initial_stock"`                    // 初始库存（可选）
}

// InitialStockInfo 药品初始库存信息，嵌入在创建请求中。
type InitialStockInfo struct {
	TotalCount     int `json:"total_count"`     // 总数量（粒/片）
	AlertThreshold int `json:"alert_threshold"` // 预警阈值，低于此值时触发提醒（默认10）
}

// MedicineResponse 是药品创建和详情接口的统一响应体。
// json:",omitempty" 标签使得空值字段不出现在 JSON 输出中，减少冗余。
type MedicineResponse struct {
	ID             uint64     `json:"id"`                        // 药品 ID
	Name           string     `json:"name"`                      // 药品名称
	Specification  string     `json:"specification"`             // 规格
	Manufacturer   string     `json:"manufacturer"`              // 厂家
	ImageURL       string     `json:"image_url,omitempty"`       // 图片 URL（空时不输出）
	Notes          string     `json:"notes"`                     // 备注
	ExternalCode   string     `json:"external_code,omitempty"`   // 外部编码（空时不输出）
	ExternalSource string     `json:"external_source,omitempty"` // 来源（空时不输出）
	Stock          *StockInfo `json:"stock,omitempty"`           // 库存信息（无库存时不输出）
	CreatedAt      time.Time  `json:"created_at"`                // 创建时间
}

// StockInfo 药品库存快照，嵌入在各药品响应中。
type StockInfo struct {
	TotalCount     int `json:"total_count"`     // 总数量
	RemainingCount int `json:"remaining_count"` // 剩余数量
	AlertThreshold int `json:"alert_threshold"` // 预警阈值
}

// MedicineListResponse 是 GET /families/:fid/medicines 的分页响应体。
type MedicineListResponse struct {
	List  []MedicineListItem `json:"list"`  // 药品列表
	Total int64              `json:"total"` // 总记录数（用于前端分页计算）
}

// MedicineListItem 药品列表单项，包含库存和计划关联状态。
type MedicineListItem struct {
	ID                  uint64 `json:"id"`                    // 药品 ID
	Name                string `json:"name"`                  // 药品名称
	Specification       string `json:"specification"`         // 规格
	Manufacturer        string `json:"manufacturer"`          // 厂家
	ImageURL            string `json:"image_url"`             // 图片 URL
	StockID             uint64 `json:"stock_id"`
	StockRemaining      int    `json:"stock_remaining"`
	StockTotal          int    `json:"stock_total"`       // 剩余数量
	StockAlertThreshold int    `json:"stock_alert_threshold"` // 预警阈值
	HasSchedule         bool   `json:"has_schedule"`          // 是否有关联的活跃用药计划
}

// MedicineDetailResponse 是药品详情接口的响应体。
// 包含所有字段和库存信息。
type MedicineDetailResponse struct {
	ID             uint64     `json:"id"`              // 药品 ID
	Name           string     `json:"name"`            // 名称
	Specification  string     `json:"specification"`   // 规格
	Manufacturer   string     `json:"manufacturer"`    // 厂家
	ImageURL       string     `json:"image_url"`       // 图片
	Notes          string     `json:"notes"`           // 备注
	ExternalCode   string     `json:"external_code"`   // 外部编码
	ExternalSource string     `json:"external_source"` // 来源
	CreatedBy      uint64     `json:"created_by"`      // 创建者
	CreatedAt      time.Time  `json:"created_at"`      // 创建时间
	Stock          *StockInfo `json:"stock,omitempty"` // 库存信息
}

// UpdateMedicineRequest 是 PUT /families/:fid/medicines/:id 的请求体。
// 所有字段均为可选，仅更新传入的非空值。
type UpdateMedicineRequest struct {
	Name          string `json:"name"`          // 新名称（可选）
	Specification string `json:"specification"` // 新规格（可选）
	Manufacturer  string `json:"manufacturer"`  // 新厂家（可选）
	ImageURL      string `json:"image_url"`     // 新图片（可选）
	Notes         string `json:"notes"`         // 新备注（可选）
}

// --- 个人药品 ---

// ListPersonal 查询个人药品列表（含库存余量）。
func (s *MedicineService) ListPersonal(ctx context.Context, userID uint64, keyword string) ([]map[string]interface{}, error) {
	medicines, err := s.medicineRepo.FindPersonalMedicines(ctx, userID, keyword)
	if err != nil { return nil, err }
	list := make([]map[string]interface{}, 0, len(medicines))
	for _, m := range medicines {
		item := map[string]interface{}{"id": m.ID, "name": m.Name, "specification": m.Specification, "manufacturer": m.Manufacturer, "image_url": m.ImageURL, "notes": m.Notes, "created_at": m.CreatedAt}
		stock, _ := s.medicineRepo.FindPersonalStock(ctx, m.ID)
		if stock != nil { item["stock_remaining"] = stock.RemainingCount }
		list = append(list, item)
	}
	return list, nil
}

// CreatePersonal 创建个人药品（可选初始化库存）。
func (s *MedicineService) CreatePersonal(ctx context.Context, userID uint64, req *CreatePersonalMedicineRequest) (map[string]interface{}, error) {
	m := &model.PersonalMedicine{UserID: userID, Name: req.Name, Specification: req.Specification, Manufacturer: req.Manufacturer, ImageURL: req.ImageURL, Notes: req.Notes}
	if err := s.medicineRepo.CreatePersonalMedicine(ctx, m); err != nil { return nil, err }
	if req.InitialStock != nil {
		s.medicineRepo.CreatePersonalStock(ctx, &model.PersonalMedicineStock{MedicineID: m.ID, TotalCount: req.InitialStock.TotalCount, RemainingCount: req.InitialStock.TotalCount})
	}
	return map[string]interface{}{"id": m.ID, "name": m.Name}, nil
}

// UpdatePersonal 更新个人药品（名称/规格/厂家/图片/备注）。
func (s *MedicineService) UpdatePersonal(ctx context.Context, userID, id uint64, req *UpdatePersonalMedicineRequest) error {
	updates := map[string]interface{}{}
	if req.Name != "" { updates["name"] = req.Name }
	if req.Specification != "" { updates["specification"] = req.Specification }
	if req.Manufacturer != "" { updates["manufacturer"] = req.Manufacturer }
	if req.ImageURL != "" { updates["image_url"] = req.ImageURL }
	if req.Notes != "" { updates["notes"] = req.Notes }
	return s.medicineRepo.UpdatePersonalMedicine(ctx, userID, id, updates)
}

// DetailPersonal 查询个人药品详情（含库存）。
func (s *MedicineService) DetailPersonal(ctx context.Context, userID, id uint64) (map[string]interface{}, error) {
	m, err := s.medicineRepo.FindPersonalMedicineByID(ctx, userID, id)
	if err != nil { return nil, fmt.Errorf("药品不存在") }
	result := map[string]interface{}{"id": m.ID, "name": m.Name, "specification": m.Specification, "manufacturer": m.Manufacturer, "image_url": m.ImageURL, "notes": m.Notes, "created_at": m.CreatedAt}
	stock, _ := s.medicineRepo.FindPersonalStock(ctx, m.ID)
	if stock != nil { result["stock"] = map[string]interface{}{"remaining_count": stock.RemainingCount, "total_count": stock.TotalCount} }
	return result, nil
}

// DeletePersonal 删除个人药品（先校验是否被个人计划引用）。
func (s *MedicineService) DeletePersonal(ctx context.Context, userID, id uint64) error {
	// 校验是否被个人用药计划引用，防止删除后计划指向不存在的药品
	hasSchedule, err := s.medicineRepo.HasActivePersonalSchedules(ctx, id)
	if err != nil {
		return fmt.Errorf("检查用药计划失败: %w", err)
	}
	if hasSchedule {
		return fmt.Errorf("药品已关联用药计划，无法删除")
	}
	return s.medicineRepo.DeletePersonalMedicine(ctx, userID, id)
}

// CreatePersonalMedicineRequest 创建个人药品请求。
type CreatePersonalMedicineRequest struct {
	Name          string            `json:"name" binding:"required"`
	Specification string            `json:"specification"`
	Manufacturer  string            `json:"manufacturer"`
	ImageURL      string            `json:"image_url"`
	Notes         string            `json:"notes"`
	InitialStock  *InitialStockInfo `json:"initial_stock,omitempty"`
}

// UpdatePersonalMedicineRequest 更新个人药品请求。
type UpdatePersonalMedicineRequest struct {
	Name          string `json:"name"`
	Specification string `json:"specification"`
	Manufacturer  string `json:"manufacturer"`
	ImageURL      string `json:"image_url"`
	Notes         string `json:"notes"`
}
