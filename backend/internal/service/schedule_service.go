package service

import (
	"context"
	"fmt"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/utils"
	"github.com/medical-monitor/internal/repository"
	"gorm.io/gorm"
)

// 用药计划服务（ScheduleService）—— 处理家庭 + 个人两套用药计划的业务逻辑。
//
// 【家庭计划】子女给老人设药（medicine_schedules 表）：
//   - Create: 创建计划 → 校验老人/药品归属 → 建计划+时间点 → 实时生成今日提醒
//   - List: 查询计划列表（可按老人筛选）
//   - Update: 更新计划（字段增量更新 + 时间点重建）
//   - Delete: 删除计划（含时间点）
//   - GetCalendar: 家庭计划日历（每月每天完成状态）
//
// 【个人计划】自己给自己设药（personal_medicine_schedules 表，独立）：
//   - CreatePersonal / ListPersonal / UpdatePersonal / DeletePersonal
//   - GetPersonalCalendar: 个人计划日历
//
// 关键业务规则：
//  1. 创建计划时校验归属：老人必须属于该家庭组、药品必须属于该家庭组（防止越权）
//  2. 实时生成提醒：创建计划后立即生成今日提醒（不等 60 秒调度器），当天就生效
//  3. 时间点重建：更新计划时先删旧时间点、再建新时间点（保证时间点一致性）
//  4. 家庭/个人解耦：两套独立计划表，个人计划按 user_id 归属、不依赖家庭组
//
// ScheduleService 用药计划服务，处理家庭 + 个人两套用药计划的业务逻辑。
type ScheduleService struct {
	scheduleRepo *repository.ScheduleRepository
	medicineRepo *repository.MedicineRepository
	familyRepo   *repository.FamilyRepository
	reminderRepo *repository.ReminderRepository
	auditRepo    *repository.AuditRepository
}

// NewScheduleService 创建 ScheduleService。
func NewScheduleService(
	scheduleRepo *repository.ScheduleRepository,
	medicineRepo *repository.MedicineRepository,
	familyRepo *repository.FamilyRepository,
	reminderRepo *repository.ReminderRepository,
	auditRepo *repository.AuditRepository,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		medicineRepo: medicineRepo,
		familyRepo:   familyRepo,
		reminderRepo: reminderRepo,
		auditRepo:    auditRepo,
	}
}

// Create 在指定家庭组中创建用药计划。
// 校验老人/药品归属，创建计划+时间点，并实时生成今日提醒。
//
// 业务校验：
//   - 目标老人必须属于该家庭组
//   - 药品必须属于该家庭组
//   - 至少一个服药时间
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - userID: 操作者（子女）用户 ID
//   - req: 创建计划请求，包含老人、药品、剂量、时间点等
//
// 返回值：
//   - *ScheduleResponse: 创建后的计划完整信息
//   - error: 校验失败或数据库错误
func (s *ScheduleService) Create(ctx context.Context, familyID, userID uint64, req *CreateScheduleRequest) (*ScheduleResponse, error) {
	// 校验目标用户属于该家庭组
	_, err := s.familyRepo.FindMember(ctx, familyID, req.ElderUserID)
	if err != nil {
		return nil, fmt.Errorf("目标用户不在该家庭组中")
	}

	// 校验药品属于该家庭组
	medicine, err := s.medicineRepo.FindMedicineByID(ctx, familyID, req.MedicineID)
	if err != nil {
		return nil, fmt.Errorf("药品不存在或不属于该家庭组")
	}

	// 校验时间点
	if len(req.Times) == 0 {
		return nil, fmt.Errorf("请至少添加一个服药时间")
	}

	// 创建计划
	schedule := &model.MedicineSchedule{
		FamilyID:      familyID,
		ElderUserID:   req.ElderUserID,
		MedicineID:    req.MedicineID,
		TotalQuantity: req.TotalQuantity,
		Dosage:        req.Dosage,
		Unit:          req.Unit,
		Relationship:  req.Relationship,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		IsActive:      true,
		CreatedBy:     userID,
	}
	if err := s.scheduleRepo.CreateSchedule(ctx, schedule); err != nil {
		return nil, fmt.Errorf("创建计划失败: %w", err)
	}

	// 创建时间点
	times := make([]*model.MedicineScheduleTime, 0, len(req.Times))
	for _, t := range req.Times {
		times = append(times, &model.MedicineScheduleTime{
			ScheduleID: schedule.ID,
			TimeOfDay:  t,
		})
	}
	if err := s.scheduleRepo.CreateTimes(ctx, times); err != nil {
		return nil, fmt.Errorf("创建时间点失败: %w", err)
	}
	schedule.Times = make([]model.MedicineScheduleTime, len(times))
	for i, t := range times {
		schedule.Times[i] = *t
	}
	schedule.Medicine = medicine

	// 实时生成今天的提醒
	s.generateTodayReminder(ctx, schedule)

	_ = s.auditRepo.LogAction(ctx, &userID, &familyID, model.AuditActionScheduleCreate,
		model.AuditTargetMedicine, &schedule.ID,
		fmt.Sprintf(`{"medicine_id":%d,"times":%v}`, req.MedicineID, req.Times), "")

	return s.buildResponse(schedule), nil
}

// List 查询家庭用药计划列表（可按老人筛选）。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - elderUserID: 老人 ID（0 表示查全部）
//
// 返回值：
//   - *ScheduleListResponse: 计划列表
//   - error: 查询失败
func (s *ScheduleService) List(ctx context.Context, familyID uint64, elderUserID uint64) (*ScheduleListResponse, error) {
	schedules, err := s.scheduleRepo.FindSchedulesByFamilyID(ctx, familyID, elderUserID)
	if err != nil {
		return nil, fmt.Errorf("查询计划失败: %w", err)
	}
	list := make([]ScheduleResponse, 0, len(schedules))
	for _, sc := range schedules {
		list = append(list, *s.buildResponse(sc))
	}
	return &ScheduleListResponse{List: list}, nil
}

// Update 更新家庭用药计划。
// 字段增量更新 + 时间点重建（先删旧、再建新）。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - userID: 操作者用户 ID
//   - scheduleID: 计划 ID
//   - req: 更新请求，非零/非空字段才更新
//
// 返回值：
//   - error: 计划不存在或数据库错误
func (s *ScheduleService) Update(ctx context.Context, familyID, userID, scheduleID uint64, req *UpdateScheduleRequest) error {
	schedule, err := s.scheduleRepo.FindScheduleByID(ctx, familyID, scheduleID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("计划不存在")
		}
		return err
	}

	if req.TotalQuantity > 0 {
		schedule.TotalQuantity = req.TotalQuantity
	}
	if req.Dosage > 0 {
		schedule.Dosage = req.Dosage
	}
	if req.Unit != "" {
		schedule.Unit = req.Unit
	}
	if req.Relationship != "" {
		schedule.Relationship = req.Relationship
	}
	if req.StartDate != "" {
		schedule.StartDate = req.StartDate
	}
	if req.EndDate != "" {
		schedule.EndDate = req.EndDate
	}
	schedule.IsActive = req.IsActive

	if err := s.scheduleRepo.UpdateSchedule(ctx, schedule); err != nil {
		return fmt.Errorf("更新失败: %w", err)
	}

	// 更新服药时间点
	if len(req.Times) > 0 {
		if err := s.scheduleRepo.DeleteTimesByScheduleID(ctx, scheduleID); err != nil {
			return err
		}
		times := make([]*model.MedicineScheduleTime, 0, len(req.Times))
		for _, t := range req.Times {
			times = append(times, &model.MedicineScheduleTime{
				ScheduleID: scheduleID,
				TimeOfDay:  t,
			})
		}
		if err := s.scheduleRepo.CreateTimes(ctx, times); err != nil {
			return err
		}
	}

	return nil
}

// Delete 删除家庭用药计划（含时间点）。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - userID: 操作者用户 ID
//   - scheduleID: 计划 ID
//
// 返回值：
//   - error: 计划不存在或数据库错误
func (s *ScheduleService) Delete(ctx context.Context, familyID, userID, scheduleID uint64) error {
	_, err := s.scheduleRepo.FindScheduleByID(ctx, familyID, scheduleID)
	if err != nil {
		return fmt.Errorf("计划不存在")
	}
	s.scheduleRepo.DeleteTimesByScheduleID(ctx, scheduleID)
	return s.scheduleRepo.DeleteSchedule(ctx, scheduleID)
}

// buildResponse 构建计划响应（补全老人昵称、药品名）。
func (s *ScheduleService) buildResponse(schedule *model.MedicineSchedule) *ScheduleResponse {
	elderNickname := ""
	if schedule.ElderUserID > 0 {
		fm, _ := s.familyRepo.FindMember(context.Background(), schedule.FamilyID, schedule.ElderUserID)
		if fm != nil && fm.User != nil {
			elderNickname = fm.User.Nickname
		}
		// fallback: 直接查 users
		if elderNickname == "" {
			if ub, err := s.familyRepo.FindMemberUser(context.Background(), schedule.ElderUserID); err == nil && ub != nil {
				elderNickname = ub.Nickname
			}
		}
	}

	medicineName := ""
	if schedule.Medicine != nil {
		medicineName = schedule.Medicine.Name
	}

	times := make([]string, 0, len(schedule.Times))
	for _, t := range schedule.Times {
		times = append(times, t.TimeOfDay)
	}

	return &ScheduleResponse{
		ID:            schedule.ID,
		ElderUserID:   schedule.ElderUserID,
		ElderNickname: elderNickname,
		MedicineID:    schedule.MedicineID,
		MedicineName:  medicineName,
		TotalQuantity: schedule.TotalQuantity,
		Dosage:        schedule.Dosage,
		Unit:          schedule.Unit,
		Relationship:  schedule.Relationship,
		StartDate:     schedule.StartDate,
		EndDate:       schedule.EndDate,
		Times:         times,
		IsActive:      schedule.IsActive,
	}
}

// generateTodayReminder 为计划实时生成今日提醒（创建计划当天就生效）。
func (s *ScheduleService) generateTodayReminder(ctx context.Context, schedule *model.MedicineSchedule) {
	today := utils.Today()

	// 计划已过期则不生成提醒（取前10位日期比较，兼容带时间戳格式）
	if schedule.EndDate != "" && len(schedule.EndDate) >= 10 && schedule.EndDate[:10] < today {
		return
	}

	for _, t := range schedule.Times {
		remindTime, err := repository.BuildReminderTime(today, t.TimeOfDay)
		if err != nil {
			continue
		}

		medicineName := ""
		if schedule.Medicine != nil {
			medicineName = schedule.Medicine.Name
		}

		items := []*model.ReminderItem{{
			MedicineID:   schedule.MedicineID,
			MedicineName: medicineName,
			Dosage:       schedule.Dosage,
			Unit:         schedule.Unit,
		}}

		reminder := &model.Reminder{
			ScheduleID:  schedule.ID,
			FamilyID:    schedule.FamilyID,
			ElderUserID: schedule.ElderUserID,
			RemindTime:  remindTime,
			RemindDate:  today,
			Status:      model.ReminderStatusPending,
			AlertLevel:  model.AlertLevelNormal,
		}
		if err := s.reminderRepo.CreateReminder(ctx, reminder); err != nil {
			continue
		}
		for _, item := range items {
			item.ReminderID = reminder.ID
		}
		_ = s.reminderRepo.CreateReminderItems(ctx, items)
	}
}

// --- 请求/响应类型 ---

// CreateScheduleRequest 创建用药计划请求。
type CreateScheduleRequest struct {
	ElderUserID   uint64   `json:"elder_user_id" binding:"required"`
	MedicineID    uint64   `json:"medicine_id" binding:"required"`
	TotalQuantity int      `json:"total_quantity"`
	Dosage        float64  `json:"dosage" binding:"required,gt=0"`
	Unit          string   `json:"unit"`
	Relationship  string   `json:"relationship"`
	StartDate     string   `json:"start_date" binding:"required"`
	EndDate       string   `json:"end_date" binding:"required"`
	Times         []string `json:"times" binding:"required,min=1"`
}

// ScheduleResponse 用药计划响应（含老人昵称、药品名）。
type ScheduleResponse struct {
	ID            uint64   `json:"id"`
	ElderUserID   uint64   `json:"elder_user_id"`
	ElderNickname string   `json:"elder_nickname"`
	MedicineID    uint64   `json:"medicine_id"`
	MedicineName  string   `json:"medicine_name"`
	TotalQuantity int      `json:"total_quantity"`
	Dosage        float64  `json:"dosage"`
	Unit          string   `json:"unit"`
	Relationship  string   `json:"relationship"`
	StartDate     string   `json:"start_date"`
	EndDate       string   `json:"end_date"`
	Times         []string `json:"times"`
	IsActive      bool     `json:"is_active"`
}

// ScheduleListResponse 用药计划列表响应。
type ScheduleListResponse struct {
	List []ScheduleResponse `json:"list"`
}

// UpdateScheduleRequest 更新用药计划请求。
type UpdateScheduleRequest struct {
	TotalQuantity int      `json:"total_quantity"`
	Dosage        float64  `json:"dosage"`
	Unit          string   `json:"unit"`
	Relationship  string   `json:"relationship"`
	StartDate     string   `json:"start_date"`
	EndDate       string   `json:"end_date"`
	Times         []string `json:"times"`
	IsActive      bool     `json:"is_active"`
}

// --- 个人用药计划（独立表） ---

// CreatePersonal 创建个人用药计划。
//
// 参数：
//   - ctx: 上下文
//   - userID: 用户 ID（本人）
//   - req: 创建计划请求
//
// 返回值：
//   - *ScheduleResponse: 创建后的计划信息
//   - error: 数据库错误
func (s *ScheduleService) CreatePersonal(ctx context.Context, userID uint64, req *CreateScheduleRequest) (*ScheduleResponse, error) {
	schedule := &model.PersonalMedicineSchedule{
		UserID: userID, MedicineID: req.MedicineID,
		TotalQuantity: req.TotalQuantity, Dosage: req.Dosage, Unit: req.Unit,
		Relationship: req.Relationship, StartDate: req.StartDate, EndDate: req.EndDate,
		IsActive: true,
	}
	if err := s.scheduleRepo.CreatePersonalSchedule(ctx, schedule); err != nil {
		return nil, err
	}
	times := make([]*model.PersonalMedicineScheduleTime, 0, len(req.Times))
	for _, t := range req.Times {
		times = append(times, &model.PersonalMedicineScheduleTime{ScheduleID: schedule.ID, TimeOfDay: t})
	}
	s.scheduleRepo.CreatePersonalTimes(ctx, times)
	// Build response
	timesStr := req.Times
	return &ScheduleResponse{
		ID: schedule.ID, ElderUserID: userID, MedicineID: req.MedicineID,
		TotalQuantity: req.TotalQuantity, Dosage: req.Dosage, Unit: req.Unit,
		Relationship: req.Relationship, StartDate: req.StartDate, EndDate: req.EndDate,
		Times: timesStr, IsActive: true,
	}, nil
}

// ListPersonal 查询个人用药计划列表。
//
// 参数：
//   - ctx: 上下文
//   - userID: 用户 ID
//
// 返回值：
//   - *ScheduleListResponse: 计划列表
//   - error: 查询失败
func (s *ScheduleService) ListPersonal(ctx context.Context, userID uint64) (*ScheduleListResponse, error) {
	schedules, _ := s.scheduleRepo.FindPersonalSchedules(ctx, userID)
	list := make([]ScheduleResponse, 0)
	for _, sc := range schedules {
		t := make([]string, 0)
		for _, tt := range sc.Times {
			t = append(t, tt.TimeOfDay)
		}
		list = append(list, ScheduleResponse{
			ID: sc.ID, ElderUserID: sc.UserID, MedicineID: sc.MedicineID,
			TotalQuantity: sc.TotalQuantity, Dosage: sc.Dosage, Unit: sc.Unit,
			Relationship: sc.Relationship, StartDate: sc.StartDate, EndDate: sc.EndDate,
			Times: t, IsActive: sc.IsActive,
		})
	}
	return &ScheduleListResponse{List: list}, nil
}

// UpdatePersonal 更新个人用药计划（字段增量更新）。
//
// 参数：
//   - ctx: 上下文
//   - userID: 用户 ID
//   - scheduleID: 计划 ID
//   - req: 更新请求，非零/非空字段才更新
//
// 返回值：
//   - error: 数据库错误
func (s *ScheduleService) UpdatePersonal(ctx context.Context, userID, scheduleID uint64, req *UpdateScheduleRequest) error {
	updates := map[string]interface{}{}
	if req.TotalQuantity > 0 {
		updates["total_quantity"] = req.TotalQuantity
	}
	if req.Dosage > 0 {
		updates["dosage"] = req.Dosage
	}
	if req.Unit != "" {
		updates["unit"] = req.Unit
	}
	if req.Relationship != "" {
		updates["relationship"] = req.Relationship
	}
	if req.StartDate != "" {
		updates["start_date"] = req.StartDate
	}
	if req.EndDate != "" {
		updates["end_date"] = req.EndDate
	}
	updates["is_active"] = req.IsActive
	return s.scheduleRepo.UpdatePersonalSchedule(ctx, userID, scheduleID, updates)
}

// DeletePersonal 删除个人用药计划（含时间点）。
//
// 参数：
//   - ctx: 上下文
//   - userID: 用户 ID
//   - scheduleID: 计划 ID
//
// 返回值：
//   - error: 数据库错误
func (s *ScheduleService) DeletePersonal(ctx context.Context, userID, scheduleID uint64) error {
	s.scheduleRepo.DeletePersonalTimes(ctx, scheduleID)
	return s.scheduleRepo.DeletePersonalSchedule(ctx, userID, scheduleID)
}

// CalendarResponse 日历接口响应。
type CalendarResponse struct {
	Dates map[string]string `json:"dates"`
}

// GetCalendar 家庭计划日历——指定月份每天的完成状态。
//
// 参数：
//   - ctx: 上下文
//   - familyID: 家庭组 ID
//   - elderUserID: 老人 ID（0 表示查全部）
//   - year: 年份
//   - month: 月份
//
// 返回值：
//   - *CalendarResponse: map[日期]状态（planned/completed/partial/pending）
//   - error: 查询失败
func (s *ScheduleService) GetCalendar(ctx context.Context, familyID, elderUserID uint64, year, month int) (*CalendarResponse, error) {
	schedules, err := s.scheduleRepo.FindSchedulesByFamilyID(ctx, familyID, elderUserID)
	if err != nil {
		return nil, err
	}
	dates := make(map[string]string)
	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	now := time.Now()
	todayStr := now.Format("2006-01-02")

	for day := 1; day <= daysInMonth; day++ {
		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		// 检查是否有计划覆盖该日期
		var hasPlan bool
		for _, sch := range schedules {
			if scheduleCoversDateStr(sch.StartDate, sch.EndDate, dateStr) {
				hasPlan = true
				break
			}
		}
		if !hasPlan {
			continue
		}

		if dateStr > todayStr {
			dates[dateStr] = "planned"
			continue
		}
		// 查提醒表获取完成状态
		reminders, _ := s.reminderRepo.FindRemindersByFamilyAndDate(ctx, familyID, dateStr)
		var total, taken int
		for _, r := range reminders {
			if elderUserID > 0 && r.ElderUserID != elderUserID {
				continue
			}
			for _, item := range r.Items {
				total++
				if item.Taken != nil && *item.Taken {
					taken++
				}
			}
		}
		if total > 0 && taken == total {
			dates[dateStr] = "completed"
		} else if taken > 0 {
			dates[dateStr] = "partial"
		} else {
			dates[dateStr] = "pending"
		}
	}
	return &CalendarResponse{Dates: dates}, nil
}

// GetPersonalCalendar 个人计划日历。
//
// 参数：
//   - ctx: 上下文
//   - userID: 用户 ID
//   - year: 年份
//   - month: 月份
//
// 返回值：
//   - *CalendarResponse: map[日期]状态
//   - error: 查询失败
func (s *ScheduleService) GetPersonalCalendar(ctx context.Context, userID uint64, year, month int) (*CalendarResponse, error) {
	schedules, err := s.scheduleRepo.FindPersonalSchedules(ctx, userID)
	if err != nil {
		return nil, err
	}
	dates := make(map[string]string)
	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	now := time.Now()
	todayStr := now.Format("2006-01-02")

	for day := 1; day <= daysInMonth; day++ {
		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		var hasPlan bool
		for _, sch := range schedules {
			if scheduleCoversDateStr(sch.StartDate, sch.EndDate, dateStr) {
				hasPlan = true
				break
			}
		}
		if !hasPlan {
			continue
		}
		if dateStr > todayStr {
			dates[dateStr] = "planned"
			continue
		}
		// 个人计划：通过 reminder 表查完成状态
		reminders, _ := s.reminderRepo.FindTodayReminders(ctx, userID, 0, dateStr)
		var total, taken int
		for _, r := range reminders {
			for _, item := range r.Items {
				total++
				if item.Taken != nil && *item.Taken {
					taken++
				}
			}
		}
		if total > 0 && taken == total {
			dates[dateStr] = "completed"
		} else if taken > 0 {
			dates[dateStr] = "partial"
		} else {
			dates[dateStr] = "pending"
		}
	}
	return &CalendarResponse{Dates: dates}, nil
}

// scheduleCoversDateStr 判断计划是否覆盖某日期（用于日历）。
func scheduleCoversDateStr(startDate, endDate, date string) bool {
	if startDate != "" && len(startDate) >= 10 && startDate[:10] > date {
		return false
	}
	if endDate != "" && len(endDate) >= 10 && endDate[:10] < date {
		return false
	}
	return true
}
