package repository

import (
	"context"

	"github.com/medical-monitor/internal/model"
	"gorm.io/gorm"
)

// MedicineRepository 处理药品与库存的数据库操作（家庭 + 个人两套表）。
//
// 方法列表：
//   - 家庭药品：
//   - CreateMedicine: 插入新药品 [medicines]
//   - FindMedicineByID: 按家庭组+ID查药品 [medicines]
//   - FindMedicinesByFamilyID: 分页查家庭药品列表（支持搜索）[medicines]
//   - UpdateMedicine: 全量更新药品 [medicines]
//   - DeleteMedicine: 软删除药品 [medicines]
//   - HasActiveSchedules: 检查药品是否被活跃计划引用 [medicine_schedules]
//   - 家庭库存：
//   - CreateStock: 创建库存记录 [medicine_stocks]
//   - FindStockByMedicineID: 按药品ID查库存 [medicine_stocks]
//   - FindStocksByFamilyID: 按家庭组查库存列表 [medicine_stocks]
//   - UpdateStock: 更新库存 [medicine_stocks]
//   - DecrementStock: 原子扣减库存 [medicine_stocks]
//   - IncrementStock: 原子增加库存（撤销恢复）[medicine_stocks]
//   - 个人药品：
//   - FindPersonalMedicines: 查个人药品列表 [personal_medicines]
//   - CreatePersonalMedicine: 新增个人药品 [personal_medicines]
//   - UpdatePersonalMedicine: 更新个人药品 [personal_medicines]
//   - FindPersonalMedicineByID: 按ID查个人药品 [personal_medicines]
//   - DeletePersonalMedicine: 删除个人药品 [personal_medicines]
//   - 个人库存：
//   - FindPersonalStock: 查个人库存 [personal_medicine_stocks]
//   - CreatePersonalStock: 创建个人库存 [personal_medicine_stocks]
//   - DecrementPersonalStock: 扣减个人库存 [personal_medicine_stocks]
//   - IncrementPersonalStock: 增加个人库存 [personal_medicine_stocks]
type MedicineRepository struct {
	db *gorm.DB
}

// NewMedicineRepository 创建 MedicineRepository。
func NewMedicineRepository(db *gorm.DB) *MedicineRepository {
	return &MedicineRepository{db: db}
}

// --- 药品操作 ---

// CreateMedicine 插入新药品记录。
func (r *MedicineRepository) CreateMedicine(ctx context.Context, medicine *model.Medicine) error {
	return r.db.WithContext(ctx).Create(medicine).Error
}

// FindMedicineByID 在指定家庭组内查找药品（双重条件防止跨组访问）。
func (r *MedicineRepository) FindMedicineByID(ctx context.Context, familyID, medicineID uint64) (*model.Medicine, error) {
	var medicine model.Medicine
	err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ?", medicineID, familyID).
		First(&medicine).Error
	if err != nil {
		return nil, err
	}
	return &medicine, nil
}

// FindMedicinesByFamilyID 分页查询家庭组的药品列表，支持按名称模糊搜索。
func (r *MedicineRepository) FindMedicinesByFamilyID(ctx context.Context, familyID uint64, keyword string, page, pageSize int) ([]*model.Medicine, int64, error) {
	var medicines []*model.Medicine
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Medicine{}).Where("family_id = ?", familyID)
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&medicines).Error
	return medicines, total, err
}

// UpdateMedicine 全量更新药品记录。
func (r *MedicineRepository) UpdateMedicine(ctx context.Context, medicine *model.Medicine) error {
	return r.db.WithContext(ctx).Save(medicine).Error
}

// DeleteMedicine 软删除药品。
// 软删除作用: 删除时只打 deleted_at 标记、不真删数据，查询时自动过滤。
// 防止误删可恢复、保留历史用药记录（医疗场景刚需）、避免破坏关联数据。
func (r *MedicineRepository) DeleteMedicine(ctx context.Context, medicineID uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Medicine{}, medicineID).Error
}

// HasActiveSchedules 检查某个药品是否还被「用药计划」引用，用于删除药品前的校验。
// 	用户点「删除药品」
//   → 先查：这个药有没有被用药计划引用？
//        ├─ 有（HasActiveSchedules 返回 true）→ 拒绝删除，提示「药品已关联用药计划」
//        └─ 没有（返回 false）→ 允许删除

func (r *MedicineRepository) HasActiveSchedules(ctx context.Context, medicineID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.MedicineSchedule{}).
		Where("medicine_id = ?", medicineID).
		Count(&count).Error
	return count > 0, err
}

// HasActivePersonalSchedules 检查个人药品是否被个人用药计划引用（删除校验用）。
func (r *MedicineRepository) HasActivePersonalSchedules(ctx context.Context, medicineID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PersonalMedicineSchedule{}).
		Where("medicine_id = ?", medicineID).
		Count(&count).Error
	return count > 0, err
}

// --- 药品余量操作 ---

// CreateStock 创建药品余量记录。
func (r *MedicineRepository) CreateStock(ctx context.Context, stock *model.MedicineStock) error {
	return r.db.WithContext(ctx).Create(stock).Error
}

// FindStockByMedicineID 按药品 ID 查找余量记录。
func (r *MedicineRepository) FindStockByMedicineID(ctx context.Context, medicineID uint64) (*model.MedicineStock, error) {
	var stock model.MedicineStock
	err := r.db.WithContext(ctx).Where("medicine_id = ?", medicineID).First(&stock).Error
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

// FindStocksByFamilyID 按家庭组查询余量列表，支持低库存筛选。
func (r *MedicineRepository) FindStocksByFamilyID(ctx context.Context, familyID uint64, statusFilter string) ([]*model.MedicineStock, error) {
	var stocks []*model.MedicineStock
	query := r.db.WithContext(ctx).Where("family_id = ?", familyID)

	if statusFilter == "low" {
		// 剩余数 > 0 但 <= 预警阈值
		query = query.Where("remaining_count > 0 AND remaining_count <= alert_threshold")
	}

	err := query.Find(&stocks).Error
	return stocks, err
}

// UpdateStock 更新余量记录。
func (r *MedicineRepository) UpdateStock(ctx context.Context, stock *model.MedicineStock) error {
	return r.db.WithContext(ctx).Save(stock).Error
}

// DecrementStock 原子性地减少药品余量，条件检查 remaining_count >= amount。
func (r *MedicineRepository) DecrementStock(ctx context.Context, medicineID uint64, amount int) error {
	return r.db.WithContext(ctx).Model(&model.MedicineStock{}).
		Where("medicine_id = ? AND remaining_count >= ?", medicineID, amount).
		UpdateColumn("remaining_count", gorm.Expr("remaining_count - ?", amount)).Error
}

// --- 个人药品（独立表） ---

func (r *MedicineRepository) FindPersonalMedicines(ctx context.Context, userID uint64, keyword string) ([]*model.PersonalMedicine, error) {
	var list []*model.PersonalMedicine
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
	}
	err := q.Order("id DESC").Find(&list).Error
	return list, err
}

func (r *MedicineRepository) CreatePersonalMedicine(ctx context.Context, m *model.PersonalMedicine) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// UpdatePersonalMedicine 更新个人药品（按 user_id + id 双重条件，防止越权）。
func (r *MedicineRepository) UpdatePersonalMedicine(ctx context.Context, userID, id uint64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.PersonalMedicine{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}

func (r *MedicineRepository) FindPersonalMedicineByID(ctx context.Context, userID, id uint64) (*model.PersonalMedicine, error) {
	var m model.PersonalMedicine
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MedicineRepository) DeletePersonalMedicine(ctx context.Context, userID, id uint64) error {
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.PersonalMedicine{}).Error
}

func (r *MedicineRepository) FindPersonalStock(ctx context.Context, medicineID uint64) (*model.PersonalMedicineStock, error) {
	var s model.PersonalMedicineStock
	err := r.db.WithContext(ctx).Where("medicine_id = ?", medicineID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *MedicineRepository) CreatePersonalStock(ctx context.Context, s *model.PersonalMedicineStock) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// UpdatePersonalStock 全量更新个人药品库存（手动校准库存）。
func (r *MedicineRepository) UpdatePersonalStock(ctx context.Context, s *model.PersonalMedicineStock) error {
	return r.db.WithContext(ctx).Save(s).Error
}

// DecrementPersonalStock 原子性地减少个人药品余量，条件检查 remaining_count >= amount。
func (r *MedicineRepository) DecrementPersonalStock(ctx context.Context, medicineID uint64, amount int) error {
	return r.db.WithContext(ctx).Model(&model.PersonalMedicineStock{}).
		Where("medicine_id = ? AND remaining_count >= ?", medicineID, amount).
		UpdateColumn("remaining_count", gorm.Expr("remaining_count - ?", amount)).Error
}

// IncrementPersonalStock 原子性地增加个人药品余量（撤销确认时恢复库存）。
func (r *MedicineRepository) IncrementPersonalStock(ctx context.Context, medicineID uint64, amount int) error {
	return r.db.WithContext(ctx).Model(&model.PersonalMedicineStock{}).
		Where("medicine_id = ?", medicineID).
		UpdateColumn("remaining_count", gorm.Expr("remaining_count + ?", amount)).Error
}

// IncrementStock 原子性地增加药品余量（撤销确认时恢复库存）。
func (r *MedicineRepository) IncrementStock(ctx context.Context, medicineID uint64, amount int) error {
	return r.db.WithContext(ctx).Model(&model.MedicineStock{}).
		Where("medicine_id = ?", medicineID).
		UpdateColumn("remaining_count", gorm.Expr("remaining_count + ?", amount)).Error
}
