package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/repository"
	"github.com/medical-monitor/internal/service"
	"go.uber.org/zap"
)

// ReminderScheduler 负责扫描用药计划并生成每日用药提醒。
type ReminderScheduler struct {
	scheduleRepo *repository.ScheduleRepository
	reminderRepo *repository.ReminderRepository
	medicineRepo *repository.MedicineRepository
	notifSvc     *service.NotificationService
	familyRepo   *repository.FamilyRepository
	userRepo     *repository.UserRepository
	logger       *zap.Logger
}

func NewReminderScheduler(
	scheduleRepo *repository.ScheduleRepository,
	reminderRepo *repository.ReminderRepository,
	medicineRepo *repository.MedicineRepository,
	notifSvc *service.NotificationService,
	familyRepo *repository.FamilyRepository,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
) *ReminderScheduler {
	return &ReminderScheduler{
		scheduleRepo: scheduleRepo,
		reminderRepo: reminderRepo,
		medicineRepo: medicineRepo,
		notifSvc:     notifSvc,
		familyRepo:   familyRepo,
		userRepo:     userRepo,
		logger:       logger,
	}
}

func (s *ReminderScheduler) Run() {
	ctx := context.Background()
	now := time.Now()
	today := now.Format("2006-01-02")
	nowTime := now.Format("15:04")

	s.logger.Info("reminder scheduler running", zap.String("time", nowTime))

	// 步骤 0：清理过期计划的旧提醒（pending 提醒不再删除，由创建计划和调度器去重保证唯一性）
	if err := s.reminderRepo.DeleteRemindersForExpiredSchedules(ctx); err != nil {
		s.logger.Error("failed to delete reminders for expired schedules", zap.Error(err))
	}

	// 步骤 1：生成今日提醒（全天窗口 00:00-23:59）
	schedules, err := s.scheduleRepo.FindActiveSchedulesByTimeWindow(ctx, "00:00", "23:59")
	if err != nil {
		s.logger.Error("failed to find active schedules", zap.Error(err))
		return
	}

	for _, schedule := range schedules {
		// 取药品名称
		medicineName := ""
		if schedule.Medicine != nil {
			medicineName = schedule.Medicine.Name
		} else {
			med, err := s.medicineRepo.FindMedicineByID(ctx, schedule.FamilyID, schedule.MedicineID)
			if err == nil {
				medicineName = med.Name
			}
		}

		// 遍历每个服药时间点
		for _, st := range schedule.Times {
			remindTime, err := repository.BuildReminderTime(today, st.TimeOfDay)
			if err != nil {
				continue
			}

			// 去重：该计划+日期+时间是否已有提醒
			dupCheck, _ := s.reminderRepo.FindReminderByScheduleTime(ctx, schedule.ID, today, st.TimeOfDay, false)
			if dupCheck != nil {
				continue
			}

			reminderItems := []*model.ReminderItem{{
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
				s.logger.Error("failed to create reminder", zap.Error(err))
				continue
			}

			for _, item := range reminderItems {
				item.ReminderID = reminder.ID
			}
			if err := s.reminderRepo.CreateReminderItems(ctx, reminderItems); err != nil {
				s.logger.Error("failed to create reminder items", zap.Error(err))
				continue
			}

			s.logger.Info("reminder generated",
				zap.Uint64("reminder_id", reminder.ID),
				zap.Uint64("elder_user_id", schedule.ElderUserID),
				zap.String("time_of_day", st.TimeOfDay),
				zap.String("medicine", medicineName),
			)
		}
	}

	// 步骤 1.5：生成个人模式今日提醒（个人用药计划 → IsPersonal 提醒）
	personalSchedules, err := s.scheduleRepo.FindActivePersonalSchedulesByTimeWindow(ctx, "00:00", "23:59")
	if err != nil {
		s.logger.Error("failed to find active personal schedules", zap.Error(err))
	} else {
		for _, ps := range personalSchedules {
			// 取药品名称（个人计划只存 medicine_id，需单独查）
			medName := ""
			if med, err := s.medicineRepo.FindPersonalMedicineByID(ctx, ps.UserID, ps.MedicineID); err == nil {
				medName = med.Name
			}

			for _, pt := range ps.Times {
				remindTime, err := repository.BuildReminderTime(today, pt.TimeOfDay)
				if err != nil {
					continue
				}

				// 去重：个人计划+日期+时间是否已有提醒
				dupCheck, _ := s.reminderRepo.FindReminderByScheduleTime(ctx, ps.ID, today, pt.TimeOfDay, true)
				if dupCheck != nil {
					continue
				}

				reminderItems := []*model.ReminderItem{{
					MedicineID:   ps.MedicineID,
					MedicineName: medName,
					Dosage:       ps.Dosage,
					Unit:         ps.Unit,
				}}

				reminder := &model.Reminder{
					ScheduleID:  ps.ID,
					FamilyID:    0, // 个人模式无家庭组
					ElderUserID: ps.UserID,
					RemindTime:  remindTime,
					RemindDate:  today,
					Status:      model.ReminderStatusPending,
					AlertLevel:  model.AlertLevelNormal,
					IsPersonal:  true,
				}
				if err := s.reminderRepo.CreateReminder(ctx, reminder); err != nil {
					s.logger.Error("failed to create personal reminder", zap.Error(err))
					continue
				}

				for _, item := range reminderItems {
					item.ReminderID = reminder.ID
				}
				if err := s.reminderRepo.CreateReminderItems(ctx, reminderItems); err != nil {
					s.logger.Error("failed to create personal reminder items", zap.Error(err))
					continue
				}

				s.logger.Info("personal reminder generated",
					zap.Uint64("reminder_id", reminder.ID),
					zap.Uint64("user_id", ps.UserID),
					zap.String("time_of_day", pt.TimeOfDay),
					zap.String("medicine", medName),
				)
			}
		}
	}

	// 步骤 2：推送当前时间窗口的提醒（覆盖家庭 + 个人模式）
	pendingReminders, err := s.reminderRepo.FindRemindersByDate(ctx, today)
	if err != nil {
		s.logger.Error("failed to find pending reminders", zap.Error(err))
		return
	}

	for _, reminder := range pendingReminders {
		if reminder.Status != model.ReminderStatusPending {
			continue
		}
		if now.Before(reminder.RemindTime) {
			continue
		}

		// 获取老人 openid
		elderUser, err := s.userRepo.FindByID(ctx, reminder.ElderUserID)
		if err != nil {
			continue
		}

		// 获取快照药品名和剂量（分开，对应模板的药品名称/药量字段）
		items, _ := s.reminderRepo.FindReminderItemsByReminderID(ctx, reminder.ID)
		medNames := ""
		dosages := ""
		for i, it := range items {
			if i > 0 {
				medNames += "、"
				dosages += "、"
			}
			medNames += it.MedicineName
			dosages += fmt.Sprintf("%.0f%s", it.Dosage, it.Unit)
		}

		// 构建时间标签
		timeLabel := fmt.Sprintf("%s %s", reminder.RemindDate, reminder.RemindTime.Format("15:04"))

		// 发送微信订阅消息
		s.notifSvc.SendReminder(ctx, elderUser.Openid, elderUser.Nickname, medNames, dosages, timeLabel)

		// 更新状态为已发送
		s.reminderRepo.UpdateReminderStatus(ctx, reminder.ID, model.ReminderStatusSent)
		nowTime := time.Now()
		reminder.SentAt = &nowTime
		s.reminderRepo.UpdateReminder(ctx, reminder)
	}
}
