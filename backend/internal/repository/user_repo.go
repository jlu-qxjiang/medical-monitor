// Package repository 提供基于 GORM 的数据访问层。
// 每个 Repository 封装一组相关表的数据库操作，通过 ctx 传递实现链路追踪和超时控制。
// 所有 SQL 查询均使用参数化，杜绝注入风险。
//
// 设计约定：
// - 上下文始终作为第一个参数传递（context.Context）。
// - 错误透传，不在 Repository 层做业务判断。
// - 关联查询使用 GORM Preload，避免 N+1 问题。
package repository

import (
	"context"

	"github.com/medical-monitor/internal/model"
	"gorm.io/gorm"
)

// UserRepository 处理 users 表的数据库操作。
//
// 方法列表：
//   - Create: 插入新用户记录 [users]
//   - FindByOpenid: 通过微信 OpenID 查找用户 [users]
//   - FindByID: 通过主键 ID 查找用户 [users]
//   - FindByIDs: 批量查询用户（Dashboard 聚合展示）[users]
//   - Update: 全量更新用户记录（不推荐，优先用 UpdateFields）[users]
//   - FindByPhone: 通过手机号查找用户（Web 开发阶段）[users]
//   - CountAll: 统计所有用户数量 [users]
//   - CountToday: 统计今日新增用户 [users]
//   - FindAll: 查询所有用户（管理员列表，最多50条）[users]
//   - UpdateFields: 仅更新指定字段（避免零值覆盖）[users]
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建 UserRepository。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// 指针类型所有 UserRepository 实例应该共享同一个db 连接池，
// 用指针保证大家操作的是同一个连接池，不会出现「你用一个池、我复制出另一个池」的混乱
// Create 插入新用户记录。
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByOpenid 通过微信 OpenID 查找用户。
func (r *UserRepository) FindByOpenid(ctx context.Context, openid string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("openid = ?", openid).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 通过主键 ID 查找用户。
func (r *UserRepository) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByIDs 批量查询用户（用于 Dashboard 聚合展示成员昵称头像）。
func (r *UserRepository) FindByIDs(ctx context.Context, ids []uint64) ([]*model.User, error) {
	var users []*model.User
	if len(ids) == 0 {
		return users, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// Update 全量更新用户记录（不推荐；优先使用 UpdateFields）。
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// FindByPhone 通过手机号查找用户（Web 开发阶段使用，后续小程序上线走微信 code 登录）。
func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CountAll 统计所有用户数量。
func (r *UserRepository) CountAll(ctx context.Context) (int64, error) {
	var c int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Count(&c).Error
	return c, err
}

// CountToday 统计今日新增用户。
func (r *UserRepository) CountToday(ctx context.Context) (int64, error) {
	var c int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("DATE(created_at) = CURDATE()").Count(&c).Error
	return c, err
}

// FindAll 查询所有用户（管理员列表）。
func (r *UserRepository) FindAll(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	err := r.db.WithContext(ctx).Order("id DESC").Limit(50).Find(&users).Error
	return users, err
}

// UpdateFields 仅更新指定字段，避免零值覆盖问题。
// UpdateFields 仅更新指定字段，避免零值覆盖问题。
// updates 为 map[string]interface{}，零值字段（空字符串、0）不会被写入数据库。
// 执行 SQL：UPDATE users SET k1=v1, k2=v2 WHERE id = ?
func (r *UserRepository) UpdateFields(ctx context.Context, userID uint64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}
