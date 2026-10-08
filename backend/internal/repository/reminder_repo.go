package repository

import (
	"context"
	"time"

	"github.com/medical-monitor/internal/model"
	"gorm.io/gorm"
)

// ReminderRepository 处理 reminders 和 reminder_items 两张表的数据库操作。
// 提醒由调度器定时生成，每条 reminder 记录包含多条快照 reminder_items。
//
// reminders 表记录「谁（elder_user_id）、哪个计划（schedule_id）、哪天几点（remind_date/time）该吃药」，
// 用 status 追踪状态流转（待推→已推→已确认/未服/跳过），
// 用 alert_level 标记超时告警，用 confirmed_at/by 记录确认信息。
// 方法列表：
//   - 提醒记录：
//   - CreateReminder: 插入新提醒 [reminders]
//   - FindReminderByID: 按主键查提醒（预加载明细）[reminders]
//   - FindTodayReminders: 查老人今天提醒（首页）[reminders]
//   - FindRemindersByFamilyAndDate: 查家庭某天提醒（看板）[reminders]
//   - FindRemindersByElderAndDate: 查老人某天提醒（历史）[reminders]
//   - FindRemindersByDate: 查某天全部提醒（推送调度）[reminders]
//   - DeleteRemindersForExpiredSchedules: 删过期计划未来提醒 [reminders + reminder_items]
//   - DeletePendingRemindersByDate: 删某天待处理提醒 [reminders + reminder_items]
//   - FindReminderByScheduleTime: 按计划+日期+时间去重 [reminders]
//   - FindExpiredPendingReminders: 查超时未升级提醒（告警）[reminders]
//   - FindRemindersWithAlertLevel: 查特定告警级别提醒 [reminders]
//   - UpdateReminderStatus: 更新提醒状态 [reminders]
//   - UpdateReminder: 全量更新提醒 [reminders]
//   - UpdateReminderFields: 按需更新字段 [reminders]
//   - FindAlerts: 筛选告警记录 [reminders]
//   - 提醒明细（快照）：
//   - CreateReminderItem: 插入单条明细 [reminder_items]
//   - CreateReminderItems: 批量插入明细 [reminder_items]
//   - UpdateReminderItemTaken: 更新服用状态 [reminder_items]
//   - UpdateReminderItemSkipReason: 设置跳过原因 [reminder_items]
//   - UpdateReminderItemsSkipped: 批量标记未服用 [reminder_items]
//   - FindReminderItemsByReminderID: 查提醒全部明细 [reminder_items]
//   - 统计：
//   - GetTodayStats: 统计今日提醒数据 [reminders]
type ReminderRepository struct {
	db *gorm.DB
}

// NewReminderRepository 创建 ReminderRepository。
func NewReminderRepository(db *gorm.DB) *ReminderRepository {
	return &ReminderRepository{db: db}
}

// --- 提醒记录操作 ---

// CreateReminder 插入新提醒记录。
func (r *ReminderRepository) CreateReminder(ctx context.Context, reminder *model.Reminder) error {
	return r.db.WithContext(ctx).Create(reminder).Error
}

// FindReminderByID 按主键查找提醒，预加载 Items 快照。
func (r *ReminderRepository) FindReminderByID(ctx context.Context, id uint64) (*model.Reminder, error) {
	var reminder model.Reminder
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&reminder, id).Error
	if err != nil {
		return nil, err
	}
	return &reminder, nil
}

// FindTodayReminders 查询老人今天的全部提醒（老人端首页）。
func (r *ReminderRepository) FindTodayReminders(ctx context.Context, elderUserID, familyID uint64, date string) ([]*model.Reminder, error) {
	var reminders []*model.Reminder
	query := r.db.WithContext(ctx).
		Preload("Items").
		Where("elder_user_id = ? AND remind_date = ?", elderUserID, date)
	if familyID > 0 {
		query = query.Where("family_id = ?", familyID)
	}
	err := query.Order("remind_time ASC").Find(&reminders).Error
	return reminders, err
}

// FindRemindersByFamilyAndDate 查询某家庭组某天的全部提醒（子女看板）。
func (r *ReminderRepository) FindRemindersByFamilyAndDate(ctx context.Context, familyID uint64, date string) ([]*model.Reminder, error) {
	var reminders []*model.Reminder
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("family_id = ? AND remind_date = ?", familyID, date).
		Order("remind_time ASC").
		Find(&reminders).Error
	return reminders, err
}

// FindRemindersByElderAndDate 查询某老人某天的全部提醒（用药历史）。
func (r *ReminderRepository) FindRemindersByElderAndDate(ctx context.Context, elderUserID uint64, date string) ([]*model.Reminder, error) {
	var reminders []*model.Reminder
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("elder_user_id = ? AND remind_date = ?", elderUserID, date).
		Order("remind_time ASC").
		Find(&reminders).Error
	return reminders, err
}

// FindRemindersByDate 查询某天全部提醒（不限用户，覆盖家庭 + 个人模式，供推送调度使用）。
func (r *ReminderRepository) FindRemindersByDate(ctx context.Context, date string) ([]*model.Reminder, error) {
	var reminders []*model.Reminder
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("remind_date = ?", date).
		Order("remind_time ASC").
		Find(&reminders).Error
	return reminders, err
}

// DeleteRemindersForExpiredSchedules 删除所有过期计划生成的「今天及以后」提醒及其明细。
// 历史提醒（remind_date < 今天）予以保留，作为用药历史记录。
func (r *ReminderRepository) DeleteRemindersForExpiredSchedules(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	var ids []uint64
	// 找到所有过期计划中今天及以后的提醒（只清理未来残留，不删历史）
	if err := r.db.WithContext(ctx).Model(&model.Reminder{}).
		Joins("JOIN medicine_schedules ON medicine_schedules.id = reminders.schedule_id").
		Where("medicine_schedules.end_date IS NOT NULL AND medicine_schedules.end_date != '' AND LEFT(medicine_schedules.end_date,10) < ?", today).
		Where("reminders.remind_date >= ?", today).
		Pluck("reminders.id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Where("reminder_id IN ?", ids).Delete(&model.ReminderItem{}).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Reminder{}).Error; err != nil {
		return err
	}
	return nil
}

// DeletePendingRemindersByDate 删除指定日期所有待处理提醒及其明细。
func (r *ReminderRepository) DeletePendingRemindersByDate(ctx context.Context, date string) error {
	var ids []uint64
	r.db.WithContext(ctx).Model(&model.Reminder{}).
		Where("remind_date = ? AND status = ?", date, model.ReminderStatusPending).
		Pluck("id", &ids)
	if len(ids) > 0 {
		r.db.WithContext(ctx).Where("reminder_id IN ?", ids).Delete(&model.ReminderItem{})
		r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Reminder{})
	}
	return nil
}

// FindReminderByScheduleTime 按计划+日期+时间查找已有提醒（去重用）。
// isPersonal 用于区分家庭计划与个人计划（两者 schedule_id 分属不同自增序列，可能相同）。
func (r *ReminderRepository) FindReminderByScheduleTime(ctx context.Context, scheduleID uint64, date, timeOfDay string, isPersonal bool) (*model.Reminder, error) {
	var reminder model.Reminder
	remindTime, _ := BuildReminderTime(date, timeOfDay)
	err := r.db.WithContext(ctx).
		Where("schedule_id = ? AND remind_date = ? AND remind_time = ? AND is_personal = ?", scheduleID, date, remindTime, isPersonal).
		First(&reminder).Error
	if err != nil {
		return nil, err
	}
	return &reminder, nil
}

// FindExpiredPendingReminders 查找已超时但未升级到最高告警级别的提醒（告警调度器用）。
func (r *ReminderRepository) FindExpiredPendingReminders(ctx context.Context, cutoff time.Time) ([]*model.Reminder, error) {
	var reminders []*model.Reminder
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("remind_time <= ? AND status IN (?, ?)", cutoff, model.ReminderStatusPending, model.ReminderStatusSent).
		Where("alert_level < ?", model.AlertLevel30Min).
		Find(&reminders).Error
	return reminders, err
}

// FindRemindersWithAlertLevel 查找特定告警级别的提醒（分级升级用）。
func (r *ReminderRepository) FindRemindersWithAlertLevel(ctx context.Context, alertLevel int8, maxAlertLevel int8) ([]*model.Reminder, error) {
	var reminders []*model.Reminder
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("alert_level = ?", alertLevel).
		Where("status IN (?, ?)", model.ReminderStatusPending, model.ReminderStatusSent).
		Find(&reminders).Error
	return reminders, err
}

// UpdateReminderStatus 更新提醒状态（如 sent → confirmed）。
func (r *ReminderRepository) UpdateReminderStatus(ctx context.Context, id uint64, status string) error {
	return r.db.WithContext(ctx).Model(&model.Reminder{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateReminder 全量更新提醒记录。
func (r *ReminderRepository) UpdateReminder(ctx context.Context, reminder *model.Reminder) error {
	return r.db.WithContext(ctx).Save(reminder).Error
}

// UpdateReminderFields 按需更新指定字段（支持 nil 值更新，如 confirmed_at = NULL 用于撤销）。
func (r *ReminderRepository) UpdateReminderFields(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Reminder{}).Where("id = ?", id).Updates(updates).Error
}

// FindAlerts 按家庭组/老人/日期筛选告警记录（告警列表页）。
func (r *ReminderRepository) FindAlerts(ctx context.Context, familyID, elderUserID uint64, date string) ([]*model.Reminder, error) {
	var reminders []*model.Reminder
	query := r.db.WithContext(ctx).
		Preload("Items").
		Where("family_id = ? AND alert_level > 0", familyID)
	if elderUserID > 0 {
		query = query.Where("elder_user_id = ?", elderUserID)
	}
	if date != "" {
		query = query.Where("remind_date = ?", date)
	}
	err := query.Order("remind_time DESC").Find(&reminders).Error
	return reminders, err
}

// --- 提醒明细（快照）操作 ---

// CreateReminderItem 插入单条提醒明细快照。
func (r *ReminderRepository) CreateReminderItem(ctx context.Context, item *model.ReminderItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// CreateReminderItems 批量插入提醒明细快照（调度器生成提醒时用）。
func (r *ReminderRepository) CreateReminderItems(ctx context.Context, items []*model.ReminderItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

// UpdateReminderItemTaken 更新单条明细的服用状态和时间。
func (r *ReminderRepository) UpdateReminderItemTaken(ctx context.Context, itemID uint64, taken bool, takenAt *time.Time) error {
	return r.db.WithContext(ctx).Model(&model.ReminderItem{}).
		Where("id = ?", itemID).
		Updates(map[string]interface{}{
			"taken":    taken,
			"taken_at": takenAt,
		}).Error
}

// UpdateReminderItemSkipReason 设置单条明细的跳过原因。
func (r *ReminderRepository) UpdateReminderItemSkipReason(ctx context.Context, itemID uint64, reason string) error {
	return r.db.WithContext(ctx).Model(&model.ReminderItem{}).
		Where("id = ?", itemID).
		Update("skip_reason", reason).Error
}

// UpdateReminderItemsSkipped 批量将提醒下所有未操作明细标记为未服用，并写入原因。
func (r *ReminderRepository) UpdateReminderItemsSkipped(ctx context.Context, reminderID uint64, skipped bool, reason string) error {
	var takenVal *bool
	if skipped {
		f := false
		takenVal = &f
	}
	return r.db.WithContext(ctx).Model(&model.ReminderItem{}).
		Where("reminder_id = ?", reminderID).
		Updates(map[string]interface{}{
			"taken":       takenVal,
			"skip_reason": reason,
		}).Error
}

// FindReminderItemsByReminderID 查询某提醒的全部明细。
func (r *ReminderRepository) FindReminderItemsByReminderID(ctx context.Context, reminderID uint64) ([]*model.ReminderItem, error) {
	var items []*model.ReminderItem
	err := r.db.WithContext(ctx).Where("reminder_id = ?", reminderID).Find(&items).Error
	return items, err
}

func (r *ReminderRepository) GetTodayStats(ctx context.Context) (map[string]int64, error) {
	today := time.Now().Format("2006-01-02")
	var total, confirmed, missed int64
	r.db.WithContext(ctx).Model(&model.Reminder{}).Where("remind_date = ?", today).Count(&total)
	r.db.WithContext(ctx).Model(&model.Reminder{}).Where("remind_date = ? AND status = ?", today, model.ReminderStatusConfirmed).Count(&confirmed)
	r.db.WithContext(ctx).Model(&model.Reminder{}).Where("remind_date = ? AND status = ?", today, model.ReminderStatusMissed).Count(&missed)
	return map[string]int64{"total": total, "confirmed": confirmed, "missed": missed}, nil
}
