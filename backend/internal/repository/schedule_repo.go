package repository

import (
	"context"
	"time"

	"github.com/medical-monitor/internal/model"
	"gorm.io/gorm"
)

// ScheduleRepository 处理用药计划的数据库操作（家庭 + 个人两套计划表）。
//
// 方法列表：
//   - 家庭计划：
//   - CreateSchedule: 创建用药计划 [medicine_schedules]
//   - FindScheduleByID: 按ID+家庭组查计划 [medicine_schedules]
//   - FindSchedulesByFamilyID: 查家庭组所有计划 [medicine_schedules]
//   - FindActiveSchedulesByTimeWindow: 扫描时间窗口内活跃计划 [medicine_schedules + medicine_schedule_times]
//   - UpdateSchedule: 更新计划 [medicine_schedules]
//   - DeleteSchedule: 软删除计划 [medicine_schedules]
//   - CreateTimes: 批量创建时间点 [medicine_schedule_times]
//   - DeleteTimesByScheduleID: 删计划所有时间点 [medicine_schedule_times]
//   - HasExistingReminder: 检查今天是否已生成提醒 [reminders]
//   - DeactivateExpired: 停用过期计划 [medicine_schedules + personal_medicine_schedules]
//   - 个人计划：
//   - CreatePersonalSchedule: 创建个人计划 [personal_medicine_schedules]
//   - CreatePersonalTimes: 批量创建个人时间点 [personal_medicine_schedule_times]
//   - FindPersonalSchedules: 查个人计划列表 [personal_medicine_schedules]
//   - UpdatePersonalSchedule: 更新个人计划 [personal_medicine_schedules]
//   - DeletePersonalSchedule: 删除个人计划 [personal_medicine_schedules]
//   - DeletePersonalTimes: 删除个人时间点 [personal_medicine_schedule_times]
//   - FindActivePersonalSchedulesByTimeWindow: 扫描活跃个人计划 [personal_medicine_schedules + personal_medicine_schedule_times]
type ScheduleRepository struct {
	db *gorm.DB
}

func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// CreateSchedule 创建用药计划。
func (r *ScheduleRepository) CreateSchedule(ctx context.Context, s *model.MedicineSchedule) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// FindScheduleByID 按 用药计划ID 和家庭组查找计划（含时间点和药品关联）。
// 同时加上 familyID 组成双重条件 WHERE id=? AND family_id=?，确保「只能查到自己家庭的计划」，
// 防止用户改 URL 的 id 去偷看别人家庭的计划
func (r *ScheduleRepository) FindScheduleByID(ctx context.Context, familyID, id uint64) (*model.MedicineSchedule, error) {
	var s model.MedicineSchedule
	err := r.db.WithContext(ctx).
		Preload("Times").Preload("Medicine").
		Where("id = ? AND family_id = ?", id, familyID).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// FindSchedulesByFamilyID 返回家庭组所有计划列表（可按老人筛选）。
func (r *ScheduleRepository) FindSchedulesByFamilyID(ctx context.Context, familyID uint64, elderUserID uint64) ([]*model.MedicineSchedule, error) {
	var list []*model.MedicineSchedule
	// Preload("Times") 是把每条计划的时间点一起查出来，
	// Preload("Medicine") 是把药品信息一起查出来，避免循环里逐条查。
	q := r.db.WithContext(ctx).Preload("Times").Preload("Medicine").
		Where("family_id = ?", familyID)
	if elderUserID > 0 {
		q = q.Where("elder_user_id = ?", elderUserID)
	}
	err := q.Order("created_at DESC").Find(&list).Error
	return list, err
}

// FindActiveSchedulesByTimeWindow 扫描时间窗口内活跃计划的药品信息和时间点。
// 找出所有该生成提醒的计划
// 日期过滤放在 SQL 外（end_date 可能为空字符串，MySQL strict mode 下 LEFT/STR_TO_DATE 会报错）。
func (r *ScheduleRepository) FindActiveSchedulesByTimeWindow(ctx context.Context, timeStart, timeEnd string) ([]*model.MedicineSchedule, error) {
	var all []*model.MedicineSchedule
	err := r.db.WithContext(ctx).
		Preload("Times").Preload("Medicine").
		Joins("JOIN medicine_schedule_times ON medicine_schedule_times.schedule_id = medicine_schedules.id").
		Where("medicine_schedules.is_active = ? AND medicine_schedule_times.time_of_day >= ? AND medicine_schedule_times.time_of_day <= ?", true, timeStart, timeEnd).
		Find(&all).Error
	if err != nil {
		return nil, err
	}

	// 在 Go 侧过滤已过期的计划（end_date 为空表示无期限）
	todayStr := today()
	var list []*model.MedicineSchedule
	for _, s := range all {
		if s.EndDate == "" {
			list = append(list, s)
		} else if len(s.EndDate) >= 10 && s.EndDate[:10] >= todayStr {
			list = append(list, s)
		}
	}
	return list, nil
}

// UpdateSchedule 更新计划。
func (r *ScheduleRepository) UpdateSchedule(ctx context.Context, s *model.MedicineSchedule) error {
	return r.db.WithContext(ctx).Save(s).Error
}

// DeleteSchedule 软删除计划。
func (r *ScheduleRepository) DeleteSchedule(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.MedicineSchedule{}, id).Error
}

// CreateTimes 批量创建时间点。
// 一条用药计划 = 一种药 + 剂量 + 起止日期，但可以有多个服药时间
func (r *ScheduleRepository) CreateTimes(ctx context.Context, times []*model.MedicineScheduleTime) error {
	return r.db.WithContext(ctx).Create(&times).Error
}

// DeleteTimesByScheduleID 删除计划的所有时间点。
func (r *ScheduleRepository) DeleteTimesByScheduleID(ctx context.Context, scheduleID uint64) error {
	return r.db.WithContext(ctx).Where("schedule_id = ?", scheduleID).Delete(&model.MedicineScheduleTime{}).Error
}

// HasExistingReminder 检查今天是否已生成提醒（去重）。
func (r *ScheduleRepository) HasExistingReminder(ctx context.Context, scheduleID uint64, remindDate string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Reminder{}).
		Where("schedule_id = ? AND remind_date = ?", scheduleID, remindDate).
		Count(&count).Error
	return count > 0, err
}

// DeactivateExpired 停用所有已过期的计划（end_date < 今天）。
func (r *ScheduleRepository) DeactivateExpired(ctx context.Context) (int64, error) {
	today := time.Now().Format("2006-01-02")
	result := r.db.WithContext(ctx).Model(&model.MedicineSchedule{}).
		Where("is_active = ? AND end_date IS NOT NULL AND end_date != '' AND LENGTH(end_date) >= 10 AND LEFT(end_date,10) < ?", true, today).
		Update("is_active", false)
	// 同样处理个人计划
	r.db.WithContext(ctx).Model(&model.PersonalMedicineSchedule{}).
		Where("is_active = ? AND end_date IS NOT NULL AND end_date != '' AND LENGTH(end_date) >= 10 AND LEFT(end_date,10) < ?", true, today).
		Update("is_active", false)
	return result.RowsAffected, result.Error
}

// today 返回今日日期字符串。
func today() string { return time.Now().Format("2006-01-02") }

// BuildReminderTime 将日期和时间拼接为 time.Time。
func BuildReminderTime(remindDate, timeOfDay string) (time.Time, error) {
	loc := time.Local
	layout := "2006-01-02 15:04"
	return time.ParseInLocation(layout, remindDate+" "+timeOfDay, loc)
}

// --- 个人用药计划 ---

// CreatePersonalSchedule 创建个人用药计划。
func (r *ScheduleRepository) CreatePersonalSchedule(ctx context.Context, s *model.PersonalMedicineSchedule) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// CreatePersonalTimes 批量创建个人计划的时间点。
func (r *ScheduleRepository) CreatePersonalTimes(ctx context.Context, times []*model.PersonalMedicineScheduleTime) error {
	return r.db.WithContext(ctx).Create(&times).Error
}

// FindPersonalSchedules 查询用户的全部个人计划（含时间点，按创建时间倒序）。
func (r *ScheduleRepository) FindPersonalSchedules(ctx context.Context, userID uint64) ([]*model.PersonalMedicineSchedule, error) {
	var list []*model.PersonalMedicineSchedule
	err := r.db.WithContext(ctx).Preload("Times").Where("user_id = ?", userID).Order("created_at DESC").Find(&list).Error
	return list, err
}

// UpdatePersonalSchedule 更新个人计划（按 user_id + id 双重条件，防止越权修改他人计划）。
func (r *ScheduleRepository) UpdatePersonalSchedule(ctx context.Context, userID, id uint64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.PersonalMedicineSchedule{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}

// DeletePersonalSchedule 删除个人计划（软删除，按 user_id + id 双重条件）。
func (r *ScheduleRepository) DeletePersonalSchedule(ctx context.Context, userID, id uint64) error {
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.PersonalMedicineSchedule{}).Error
}

// DeletePersonalTimes 删除个人计划的所有时间点。
func (r *ScheduleRepository) DeletePersonalTimes(ctx context.Context, scheduleID uint64) error {
	return r.db.WithContext(ctx).Where("schedule_id = ?", scheduleID).Delete(&model.PersonalMedicineScheduleTime{}).Error
}

// FindActivePersonalSchedulesByTimeWindow 扫描时间窗口内活跃的个人用药计划（含时间点）。
// 日期过滤放在 SQL 外（end_date 可能为空，MySQL strict mode 下会报错）。
func (r *ScheduleRepository) FindActivePersonalSchedulesByTimeWindow(ctx context.Context, timeStart, timeEnd string) ([]*model.PersonalMedicineSchedule, error) {
	var all []*model.PersonalMedicineSchedule
	err := r.db.WithContext(ctx).
		Preload("Times").
		Joins("JOIN personal_medicine_schedule_times ON personal_medicine_schedule_times.schedule_id = personal_medicine_schedules.id").
		Where("personal_medicine_schedules.is_active = ? AND personal_medicine_schedule_times.time_of_day >= ? AND personal_medicine_schedule_times.time_of_day <= ?", true, timeStart, timeEnd).
		Find(&all).Error
	if err != nil {
		return nil, err
	}

	// 在 Go 侧过滤已过期的计划（end_date 为空表示无期限）
	todayStr := today()
	var list []*model.PersonalMedicineSchedule
	for _, s := range all {
		if s.EndDate == "" {
			list = append(list, s)
		} else if len(s.EndDate) >= 10 && s.EndDate[:10] >= todayStr {
			list = append(list, s)
		}
	}
	return list, nil
}
