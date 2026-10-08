// Package model 定义全部 GORM 数据模型，与数据库表一一映射。
// 每个模型通过 TableName() 方法显式指定表名。
// JSON tag 中 "-" 表示不通过 API 暴露该字段（如 openid、deleted_at）。
package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户表，通过微信 OpenID 唯一标识。
// phone 仅存后4位明文，完整手机号 AES-256-GCM 加密存在 phone_enc。
type User struct {
	ID        uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Openid    string         `gorm:"column:openid;type:varchar(64);not null;uniqueIndex:uk_openid" json:"-"` // 不通过 API 返回
	Unionid   string         `gorm:"column:unionid;type:varchar(64);default:null" json:"-"`                   // 预留跨应用标识
	Nickname  string         `gorm:"column:nickname;type:varchar(64);not null;default:''" json:"nickname"`
	AvatarURL string         `gorm:"column:avatar_url;type:varchar(512);not null;default:''" json:"avatar_url"`
	Phone     string         `gorm:"column:phone;type:varchar(20);not null;default:''" json:"-"`    // 仅存后4位明文
	PhoneEnc  string         `gorm:"column:phone_enc;type:varchar(256);default:null" json:"-"`      // AES-256-GCM 密文完整号
	PasswordHash string     `gorm:"column:password_hash;type:varchar(256);not null;default:''" json:"-"` // bcrypt 密码哈希（Web 开发阶段使用）
	Role         int8       `gorm:"column:role;type:tinyint;not null;default:0" json:"role"`              // 角色：0=未选 1=老人 2=子女 3=管理员
		NotifyReminder  bool       `gorm:"column:notify_reminder;not null;default:true" json:"notify_reminder"` // 用药提醒开关
		NotifyStock     bool       `gorm:"column:notify_stock;not null;default:true" json:"notify_stock"`         // 库存提醒开关
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"` // GORM 软删除
}

// TableName 返回 users 表名。
func (User) TableName() string {
	return "users"
}

// Family 家庭组表，一个家庭组可包含多个成员（老人+子女），支持多对多关系。
type Family struct {
	ID         uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string         `gorm:"column:name;type:varchar(64);not null" json:"name"`
	InviteCode         string     `gorm:"column:invite_code;type:varchar(8);not null;uniqueIndex:uk_invite_code" json:"invite_code"`
	InviteCodeExpiresAt *time.Time `gorm:"column:invite_code_expires_at" json:"invite_code_expires_at"` // 邀请码过期时间（1分钟有效期）
	IsPersonal bool           `gorm:"column:is_personal;type:tinyint(1);not null;default:0" json:"is_personal"` // 是否个人私有家庭（无邀请码）
	CreatedBy  uint64         `gorm:"column:created_by;not null;index:idx_created_by" json:"created_by"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 families 表名。
func (Family) TableName() string {
	return "families"
}

// FamilyMember 家庭成员关联表，记录用户在某家庭组中的角色。
// 联合唯一索引 (family_id, user_id, deleted_at) 允许同用户退出后重新加入。
type FamilyMember struct {
	ID        uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FamilyID  uint64         `gorm:"column:family_id;not null;uniqueIndex:uk_family_user,priority:1" json:"family_id"`
	UserID    uint64         `gorm:"column:user_id;not null;uniqueIndex:uk_family_user,priority:2;index:idx_user_id" json:"user_id"`
	Role      string         `gorm:"column:role;type:varchar(16);not null;default:'member';index:idx_family_role,priority:2" json:"role"` // owner/admin/member/elder
	JoinedAt  time.Time      `gorm:"column:joined_at;autoCreateTime" json:"joined_at"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"-"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;uniqueIndex:uk_family_user,priority:3" json:"-"` // 退出家庭组 = 软删除

	// 关联（非数据库字段，通过 Preload 填充）
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 返回 family_members 表名。
func (FamilyMember) TableName() string {
	return "family_members"
}

// 成员角色常量。
const (
	RoleOwner  = "owner"  // 创建者
	RoleAdmin  = "admin"  // 管理员
	RoleMember = "member" // 子女（普通成员）
	RoleElder  = "elder"  // 老人
)

// ValidRoles 所有合法角色的列表。
var ValidRoles = []string{RoleOwner, RoleAdmin, RoleMember, RoleElder}

// IsValidRole 校验角色字符串是否为合法角色。
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// Medicine 药品表，存储家庭组内录入的药品信息。
// external_code + external_source 为 P2 第三方药品库对接预留扩展字段。
type Medicine struct {
	ID             uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FamilyID       uint64         `gorm:"column:family_id;not null;index:idx_family_id;index:idx_family_name,priority:1" json:"family_id"`
	Name           string         `gorm:"column:name;type:varchar(128);not null;index:idx_family_name,priority:2" json:"name"`
	Specification  string         `gorm:"column:specification;type:varchar(128);not null;default:''" json:"specification"` // 规格（如"30mg×7片"）
	Manufacturer   string         `gorm:"column:manufacturer;type:varchar(128);not null;default:''" json:"manufacturer"`   // 生产厂家
	ImageURL       string         `gorm:"column:image_url;type:varchar(512);not null;default:''" json:"image_url"`        // 药品照片 URL
	Notes          string         `gorm:"column:notes;type:varchar(256);not null;default:''" json:"notes"`                 // 备注（如"饭后服用"）
	Expiry         string         `gorm:"column:expiry;type:varchar(10);not null;default:''" json:"expiry"` // 有效期（YYYY-MM-DD）
	ExternalCode   string         `gorm:"column:external_code;type:varchar(64);not null;default:'';index:idx_external,priority:2" json:"external_code"`     // 第三方药品编码（预留）
	ExternalSource string         `gorm:"column:external_source;type:varchar(32);not null;default:'';index:idx_external,priority:1" json:"external_source"` // 来源标识（预留，如 ypk）
	CreatedBy      uint64         `gorm:"column:created_by;not null" json:"created_by"` // 录入者
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 medicines 表名。
func (Medicine) TableName() string {
	return "medicines"
}

// MedicineSchedule 用药计划表，定义某位老人服用某种药品的完整计划。
// 一个计划对应一种药品，可包含多个每日服药时间段。
type MedicineSchedule struct {
	ID             uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FamilyID       uint64         `gorm:"column:family_id;not null;index:idx_family_elder,priority:1" json:"family_id"`
	ElderUserID    uint64         `gorm:"column:elder_user_id;not null;index:idx_family_elder,priority:2" json:"elder_user_id"`
	MedicineID     uint64         `gorm:"column:medicine_id;not null;index:idx_medicine_id" json:"medicine_id"`
	TotalQuantity  int            `gorm:"column:total_quantity;not null;default:0" json:"total_quantity"`        // 药品总数量（如 30 粒）
	Dosage         float64        `gorm:"column:dosage;type:decimal(10,2);not null" json:"dosage"`                // 单次服用剂量
	Unit           string         `gorm:"column:unit;type:varchar(16);not null;default:'粒'" json:"unit"`         // 剂量单位
	Relationship   string         `gorm:"column:relationship;type:varchar(32);not null;default:''" json:"relationship"`  // 服药关系：饭前/饭后/饭后1小时/饭后2小时/不限
	StartDate      string         `gorm:"column:start_date;type:date;default:null" json:"start_date"`                  // 开始日期
	EndDate        string         `gorm:"column:end_date;type:date;default:null" json:"end_date"`                      // 结束日期
	IsActive       bool           `gorm:"column:is_active;type:tinyint(1);not null;default:1" json:"is_active"`
	CreatedBy      uint64         `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`

	// 关联
	Times    []MedicineScheduleTime `gorm:"foreignKey:ScheduleID" json:"times,omitempty"`
	Medicine *Medicine              `gorm:"foreignKey:MedicineID" json:"medicine,omitempty"`
}

// TableName 返回 medicine_schedules 表名。
func (MedicineSchedule) TableName() string {
	return "medicine_schedules"
}

// MedicineScheduleTime 用药计划的每日服药时间点（可多个）。
type MedicineScheduleTime struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ScheduleID uint64    `gorm:"column:schedule_id;not null;index:idx_schedule_id" json:"schedule_id"`
	TimeOfDay  string    `gorm:"column:time_of_day;type:varchar(8);not null" json:"time_of_day"` // 如 "08:00"
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"-"`
}

// TableName 返回 medicine_schedule_times 表名。
func (MedicineScheduleTime) TableName() string {
	return "medicine_schedule_times"
}

// Reminder 提醒记录表，每日由调度器从用药计划自动生成。
// 每次提醒包含一条 reminder 记录 + 多条 reminder_items 快照。
type Reminder struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ScheduleID   uint64     `gorm:"column:schedule_id;not null;index:idx_schedule_id" json:"schedule_id"`
	FamilyID     uint64     `gorm:"column:family_id;not null;index:idx_family_date,priority:1" json:"family_id"`
	ElderUserID  uint64     `gorm:"column:elder_user_id;not null;index:idx_elder_date,priority:1" json:"elder_user_id"`
	RemindTime   time.Time  `gorm:"column:remind_time;not null;index:idx_remind_time_status,priority:1" json:"remind_time"`       // 具体提醒时间
	RemindDate   string     `gorm:"column:remind_date;type:date;not null;index:idx_elder_date,priority:2;index:idx_family_date,priority:2" json:"remind_date"` // 提醒日期（冗余，加速按日查询）
	Status       string     `gorm:"column:status;type:varchar(16);not null;default:'pending';index:idx_remind_time_status,priority:2;index:idx_status_alert,priority:1" json:"status"` // pending/sent/confirmed/missed/skipped
	WechatMsgID  string     `gorm:"column:wechat_msg_id;type:varchar(128);not null;default:''" json:"-"` // 微信订阅消息 ID
	SentAt       *time.Time `gorm:"column:sent_at" json:"sent_at"`
	ConfirmedAt  *time.Time `gorm:"column:confirmed_at" json:"confirmed_at"`
	ConfirmedBy  *uint64    `gorm:"column:confirmed_by" json:"confirmed_by"`
	AlertLevel   int8       `gorm:"column:alert_level;type:tinyint;not null;default:0;index:idx_status_alert,priority:2" json:"alert_level"` // 告警级别: 0=正常 2=30分钟
	IsPersonal   bool       `gorm:"column:is_personal;type:tinyint(1);not null;default:0" json:"is_personal"`                              // 是否个人模式提醒（个人用药计划生成）
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	Items []ReminderItem `gorm:"foreignKey:ReminderID" json:"items,omitempty"`
}

// TableName 返回 reminders 表名。
func (Reminder) TableName() string {
	return "reminders"
}

// 用户角色常量（users.role）。
const (
	UserRoleNone  int8 = 0 // 未选择
	UserRoleElder int8 = 1 // 老人
	UserRoleChild int8 = 2 // 子女
	UserRoleAdmin int8 = 3 // 管理员
)

// 提醒状态常量。
const (
	ReminderStatusPending   = "pending"   // 待推送
	ReminderStatusSent      = "sent"      // 已推送，待确认
	ReminderStatusConfirmed = "confirmed" // 已确认服用
	ReminderStatusMissed    = "missed"    // 未服用（超时）
	ReminderStatusSkipped   = "skipped"   // 已跳过
)

// 告警级别常量。
const (
	AlertLevelNormal int8 = 0 // 正常
	AlertLevel30Min  int8 = 2 // 超时 30 分钟 → 通知子女 + 再提醒本人
)

// ReminderItem 提醒明细表，记录每次提醒中包含的药品清单快照。
// 快照设计：药品名、剂量在生成提醒时固化，后续修改药品不影响历史记录准确性。
type ReminderItem struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ReminderID   uint64     `gorm:"column:reminder_id;not null;index:idx_reminder_id" json:"reminder_id"`
	MedicineID   uint64     `gorm:"column:medicine_id;not null;index:idx_medicine_id" json:"medicine_id"`
	MedicineName string     `gorm:"column:medicine_name;type:varchar(128);not null" json:"medicine_name"` // 药品名快照
	Dosage       float64    `gorm:"column:dosage;type:decimal(10,2);not null" json:"dosage"`              // 剂量快照
	Unit         string     `gorm:"column:unit;type:varchar(16);not null;default:'粒'" json:"unit"`        // 单位快照
	Taken        *bool      `gorm:"column:taken;type:tinyint(1);default:null" json:"taken"`               // NULL=未操作 true=已服用 false=未服用
	TakenAt      *time.Time `gorm:"column:taken_at" json:"taken_at"`
	SkipReason   string     `gorm:"column:skip_reason;type:varchar(64);not null;default:''" json:"skip_reason"` // 跳过/未服用原因
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"-"`
}

// TableName 返回 reminder_items 表名。
func (ReminderItem) TableName() string {
	return "reminder_items"
}

// MedicineStock 药品余量表，每种药品在家庭组内只有一条余量记录。
type MedicineStock struct {
	ID             uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MedicineID     uint64     `gorm:"column:medicine_id;not null;uniqueIndex:uk_medicine_id" json:"medicine_id"`
	FamilyID       uint64     `gorm:"column:family_id;not null;index:idx_family_id" json:"family_id"`
	TotalCount     int        `gorm:"column:total_count;not null;default:0" json:"total_count"`         // 总数量
	RemainingCount int        `gorm:"column:remaining_count;not null;default:0;index:idx_remaining" json:"remaining_count"` // 剩余数量
	AlertThreshold int        `gorm:"column:alert_threshold;not null;default:10" json:"alert_threshold"` // 预警阈值
	LastRefillAt   *time.Time `gorm:"column:last_refill_at" json:"last_refill_at"` // 上次补充时间
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 返回 medicine_stocks 表名。
func (MedicineStock) TableName() string {
	return "medicine_stocks"
}

// 余量状态常量。
const (
	StockStatusNormal = "normal" // 充足
	StockStatusLow    = "low"    // 不足
	StockStatusEmpty  = "empty"  // 已用完
)

// RefreshToken Refresh Token 持久化表，解决 Redis 重启后 Token 校验丢失问题。
// 采用 Token 轮转策略：每次刷新时旧 Token 作废，新 Token 入库。
type RefreshToken struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    uint64     `gorm:"column:user_id;not null;index:idx_user_id" json:"user_id"`
	TokenHash string     `gorm:"column:token_hash;type:varchar(128);not null;uniqueIndex:uk_token_hash" json:"-"` // SHA-256 哈希值，不通过 API 返回
	ExpiresAt time.Time  `gorm:"column:expires_at;not null;index:idx_expires_at" json:"expires_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at" json:"revoked_at"` // 非 nil 表示已撤销
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 返回 refresh_tokens 表名。
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// AuditLog 操作审计日志，记录关键操作的完整上下文。
type AuditLog struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     *uint64   `gorm:"column:user_id;index:idx_user_action,priority:1" json:"user_id"`
	FamilyID   *uint64   `gorm:"column:family_id" json:"family_id"`
	Action     string    `gorm:"column:action;type:varchar(64);not null;index:idx_user_action,priority:2" json:"action"`       // 操作类型，如 medicine.create
	TargetType string    `gorm:"column:target_type;type:varchar(32);not null;index:idx_target,priority:1" json:"target_type"` // 目标类型，如 medicine
	TargetID   *uint64   `gorm:"column:target_id;index:idx_target,priority:2" json:"target_id"`
	Detail     string    `gorm:"column:detail;type:json" json:"detail"` // JSON 格式详情，敏感字段已脱敏
	IP         string    `gorm:"column:ip;type:varchar(64);not null;default:''" json:"ip"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;index:idx_created_at;index:idx_user_action,priority:3" json:"created_at"`
}

// TableName 返回 audit_logs 表名。
func (AuditLog) TableName() string {
	return "audit_logs"
}

// 审计操作类型常量。
const (
	AuditActionMedicineCreate  = "medicine.create"
	AuditActionMedicineUpdate  = "medicine.update"
	AuditActionMedicineDelete  = "medicine.delete"
	AuditActionScheduleCreate  = "schedule.create"
	AuditActionScheduleUpdate  = "schedule.update"
	AuditActionScheduleDelete  = "schedule.delete"
	AuditActionReminderConfirm = "reminder.confirm"
	AuditActionReminderMiss    = "reminder.miss"
	AuditActionReminderSkip    = "reminder.skip"
	AuditActionReminderRevoke  = "reminder.revoke"
	AuditActionFamilyJoin      = "family.join"
	AuditActionFamilyLeave     = "family.leave"
	AuditActionMemberRemove    = "family.member_remove"
	AuditActionRoleChange      = "family.role_change"
	AuditActionStockUpdate     = "stock.update"
)

// 审计目标类型常量。
const (
	AuditTargetMedicine = "medicine"
	AuditTargetReminder = "reminder"
	AuditTargetFamily   = "family"
	AuditTargetUser     = "user"
	AuditTargetStock    = "stock"
)

// ====== 个人用药表（独立于家庭用药） ======

// PersonalMedicine 个人药品表，按 user_id 归属。
type PersonalMedicine struct {
	ID             uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID         uint64         `gorm:"column:user_id;not null;index:idx_user_id" json:"user_id"`
	Name           string         `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Specification  string         `gorm:"column:specification;type:varchar(128);not null;default:''" json:"specification"`
	Manufacturer   string         `gorm:"column:manufacturer;type:varchar(128);not null;default:''" json:"manufacturer"`
	ImageURL       string         `gorm:"column:image_url;type:varchar(512);not null;default:''" json:"image_url"`
	Notes          string         `gorm:"column:notes;type:varchar(256);not null;default:''" json:"notes"`
	Expiry         string         `gorm:"column:expiry;type:varchar(10);not null;default:''" json:"expiry"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (PersonalMedicine) TableName() string { return "personal_medicines" }

// PersonalMedicineStock 个人药品余量表。
type PersonalMedicineStock struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MedicineID      uint64    `gorm:"column:medicine_id;not null;uniqueIndex:uk_personal_medicine_id" json:"medicine_id"`
	TotalCount      int       `gorm:"column:total_count;not null;default:0" json:"total_count"`
	RemainingCount  int       `gorm:"column:remaining_count;not null;default:0" json:"remaining_count"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"-"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"-"`
}

func (PersonalMedicineStock) TableName() string { return "personal_medicine_stocks" }

// PersonalMedicineSchedule 个人用药计划表。
type PersonalMedicineSchedule struct {
	ID             uint64                        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID         uint64                        `gorm:"column:user_id;not null;index:idx_user_id" json:"user_id"`
	MedicineID     uint64                        `gorm:"column:medicine_id;not null" json:"medicine_id"`
	TotalQuantity  int                           `gorm:"column:total_quantity;not null;default:0" json:"total_quantity"`
	Dosage         float64                       `gorm:"column:dosage;type:decimal(10,2);not null" json:"dosage"`
	Unit           string                        `gorm:"column:unit;type:varchar(16);not null;default:'粒'" json:"unit"`
	Relationship   string                        `gorm:"column:relationship;type:varchar(32);not null;default:''" json:"relationship"`
	StartDate      string                        `gorm:"column:start_date;type:date;default:null" json:"start_date"`
	EndDate        string                        `gorm:"column:end_date;type:date;default:null" json:"end_date"`
	IsActive       bool                          `gorm:"column:is_active;type:tinyint(1);not null;default:1" json:"is_active"`
	CreatedAt      time.Time                     `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time                     `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt                `gorm:"column:deleted_at;index" json:"-"`
	Times          []PersonalMedicineScheduleTime `gorm:"foreignKey:ScheduleID" json:"times,omitempty"`
}

func (PersonalMedicineSchedule) TableName() string { return "personal_medicine_schedules" }

// PersonalMedicineScheduleTime 个人用药计划的服药时间点。
type PersonalMedicineScheduleTime struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ScheduleID uint64    `gorm:"column:schedule_id;not null" json:"schedule_id"`
	TimeOfDay  string    `gorm:"column:time_of_day;type:varchar(8);not null" json:"time_of_day"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"-"`
}

func (PersonalMedicineScheduleTime) TableName() string { return "personal_medicine_schedule_times" }
