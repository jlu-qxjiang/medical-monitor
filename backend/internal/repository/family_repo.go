package repository

import (
	"context"
	"time"

	"github.com/medical-monitor/internal/model"
	"gorm.io/gorm"
)

// FamilyRepository 处理 families 和 family_members 两张表的数据库操作。
// 家庭组成员关系通过软删除实现退出/重新加入。
//
// 方法列表：
//   - 家庭组操作：
//     - CreateFamily: 创建新家庭组 [families]
//     - FindFamilyByID: 通过主键查找家庭组 [families]
//     - FindFamilyByInviteCode: 通过6位邀请码查找家庭组 [families]
//     - UpdateInviteCode: 刷新家庭组邀请码 [families]
//     - FindFamiliesByUserID: 返回用户所属的所有家庭组 [family_members]
//   - 家庭成员操作：
//     - AddMember: 添加用户到家庭组 [family_members]
//     - FindMemberUser: 直接查 user 表取昵称 [users]
//     - FindMember: 查找指定家庭组中的某成员 [family_members]
//     - FindMembersByFamilyID: 返回家庭组全部活跃成员 [family_members]
//     - FindEldersByFamilyID: 返回家庭组中所有老人成员 [family_members]
//     - CountMembers: 统计家庭组活跃成员数 [family_members]
//     - UpdateMemberRole: 修改成员角色 [family_members]
//     - RemoveMember: 软删除成员记录 [family_members]
//     - IsMember: 检查用户是否为家庭组的活跃成员 [family_members]
type FamilyRepository struct {
	db *gorm.DB
}

// NewFamilyRepository 创建 FamilyRepository。
func NewFamilyRepository(db *gorm.DB) *FamilyRepository {
	return &FamilyRepository{db: db}
}

// --- 家庭组操作 ---

// CreateFamily 创建新家庭组。
func (r *FamilyRepository) CreateFamily(ctx context.Context, family *model.Family) error {
	return r.db.WithContext(ctx).Create(family).Error
}

// FindFamilyByID 通过主键查找家庭组。
func (r *FamilyRepository) FindFamilyByID(ctx context.Context, id uint64) (*model.Family, error) {
	var family model.Family
	err := r.db.WithContext(ctx).First(&family, id).Error
	if err != nil {
		return nil, err
	}
	return &family, nil
}

// FindFamilyByInviteCode 通过6位邀请码查找家庭组（加入家庭组时用）。
func (r *FamilyRepository) FindFamilyByInviteCode(ctx context.Context, code string) (*model.Family, error) {
	var family model.Family
	err := r.db.WithContext(ctx).Where("invite_code = ?", code).First(&family).Error
	if err != nil {
		return nil, err
	}
	return &family, nil
}

// UpdateInviteCode 刷新家庭组邀请码。
func (r *FamilyRepository) UpdateInviteCode(ctx context.Context, familyID uint64, newCode string, expiresAt *time.Time) error {
	return r.db.WithContext(ctx).Model(&model.Family{}).Where("id = ?", familyID).
		Updates(map[string]interface{}{"invite_code": newCode, "invite_code_expires_at": expiresAt}).Error
}

// FindFamiliesByUserID 返回用户所属的所有家庭组（成员列表页用）。
func (r *FamilyRepository) FindFamiliesByUserID(ctx context.Context, userID uint64) ([]*model.FamilyMember, error) {
	var members []*model.FamilyMember
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&members).Error
	return members, err
}

// --- 家庭成员操作 ---

// AddMember 添加用户到家庭组。
func (r *FamilyRepository) AddMember(ctx context.Context, member *model.FamilyMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// FindMemberUser 直接查 user 表取昵称（兜底）。
func (r *FamilyRepository) FindMemberUser(ctx context.Context, userID uint64) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).First(&u, userID).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindMember 查找指定家庭组中的某成员。
func (r *FamilyRepository) FindMember(ctx context.Context, familyID, userID uint64) (*model.FamilyMember, error) {
	var member model.FamilyMember
	err := r.db.WithContext(ctx).Preload("User").Where("family_id = ? AND user_id = ?", familyID, userID).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// FindMembersByFamilyID 返回家庭组全部活跃成员（含用户信息）。
func (r *FamilyRepository) FindMembersByFamilyID(ctx context.Context, familyID uint64) ([]*model.FamilyMember, error) {
	var members []*model.FamilyMember
	err := r.db.WithContext(ctx).
		Preload("User"). // 预加载用户昵称头像
		Where("family_id = ?", familyID).
		Find(&members).Error
	return members, err
}

// FindEldersByFamilyID 返回家庭组中所有老人成员（子女看板用）。
func (r *FamilyRepository) FindEldersByFamilyID(ctx context.Context, familyID uint64) ([]*model.FamilyMember, error) {
	var members []*model.FamilyMember
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("family_id = ? AND role = ?", familyID, model.RoleElder).
		Find(&members).Error
	return members, err
}

// CountMembers 统计家庭组活跃成员数。
func (r *FamilyRepository) CountMembers(ctx context.Context, familyID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.FamilyMember{}).
		Where("family_id = ?", familyID).
		Count(&count).Error
	return count, err
}

// UpdateMemberRole 修改成员角色（如将 member 升级为 admin）。
func (r *FamilyRepository) UpdateMemberRole(ctx context.Context, memberID uint64, role string) error {
	return r.db.WithContext(ctx).Model(&model.FamilyMember{}).
		Where("id = ?", memberID).
		Update("role", role).Error
}

// RemoveMember 软删除成员记录，表示退出或被移出家庭组。
func (r *FamilyRepository) RemoveMember(ctx context.Context, memberID uint64) error {
	return r.db.WithContext(ctx).Delete(&model.FamilyMember{}, memberID).Error
}

// IsMember 检查用户是否为家庭组的活跃成员，返回 (是否成员, 角色, 错误)。
// IsMember(familyID=2, userID=1)、 查数据库 family_members 表：
func (r *FamilyRepository) IsMember(ctx context.Context, familyID, userID uint64) (bool, string, error) {
	member, err := r.FindMember(ctx, familyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, "", nil
		}
		return false, "", err
	}
	return true, member.Role, nil
}
