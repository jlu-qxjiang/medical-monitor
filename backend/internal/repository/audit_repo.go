package repository

import (
	"context"

	"github.com/medical-monitor/internal/model"
	"gorm.io/gorm"
)

// AuditRepository 处理 audit_logs 表的数据库操作。
// 审计日志只追加不修改不删除，用于安全追踪和问题排查。
//
// 方法列表：
//   - Create: 插入一条审计日志 [audit_logs]
//   - LogAction: 记录一次操作（封装 Create，带默认字段）[audit_logs]
type AuditRepository struct {
	db *gorm.DB
}

// NewAuditRepository 创建 AuditRepository。
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create 插入一条审计日志。
func (r *AuditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// LogAction 便捷方法，用较少参数快速记录一条操作日志。
// detail 中敏感字段（手机号等）应在调用前完成脱敏。
func (r *AuditRepository) LogAction(ctx context.Context, userID, familyID *uint64, action, targetType string, targetID *uint64, detail, ip string) error {
	log := &model.AuditLog{
		UserID:     userID,
		FamilyID:   familyID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		IP:         ip,
	}
	return r.db.WithContext(ctx).Create(log).Error
}
