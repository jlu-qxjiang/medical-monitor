package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"github.com/medical-monitor/internal/repository"
)

type AdminHandler struct {
	userRepo *repository.UserRepository
	familyRepo *repository.FamilyRepository
	reminderRepo *repository.ReminderRepository
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	familyRepo *repository.FamilyRepository,
	reminderRepo *repository.ReminderRepository,
) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, familyRepo: familyRepo, reminderRepo: reminderRepo}
}

// Stats 处理 GET /api/v1/admin/stats —— 管理后台统计数据。
func (h *AdminHandler) Stats(c *gin.Context) {
	totalUsers, _ := h.userRepo.CountAll(c.Request.Context())
	usersToday, _ := h.userRepo.CountToday(c.Request.Context())
	reminderStats, _ := h.reminderRepo.GetTodayStats(c.Request.Context())

	response.Success(c, map[string]interface{}{
		"total_users": totalUsers,
		"users_today": usersToday,
		"reminders_total": reminderStats["total"],
		"reminders_confirmed": reminderStats["confirmed"],
		"reminders_missed": reminderStats["missed"],
	})
}

// Users 处理 GET /api/v1/admin/users —— 用户列表。
func (h *AdminHandler) Users(c *gin.Context) {
	users, err := h.userRepo.FindAll(c.Request.Context())
	if err != nil {
		response.Error(c, errcode.InternalError, "查询失败")
		return
	}
	type UserItem struct {
		ID        uint64 `json:"id"`
		Nickname  string `json:"nickname"`
		Phone     string `json:"phone"`
		Role      int8   `json:"role"`
		CreatedAt string `json:"created_at"`
	}
	list := make([]UserItem, 0, len(users))
	for _, u := range users {
		list = append(list, UserItem{
			ID: u.ID, Nickname: u.Nickname, Phone: u.Phone,
			Role: u.Role, CreatedAt: u.CreatedAt.Format("2006-01-02"),
		})
	}
	response.Success(c, map[string]interface{}{"list": list})
}
