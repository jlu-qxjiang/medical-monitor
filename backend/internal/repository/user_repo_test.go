package repository

import (
	"context"
	"testing"

	"github.com/medical-monitor/internal/model"
)

func TestUserRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Openid:       "web_13800000001",
		Nickname:     "测试用户",
		Phone:        "13800000001",
		PasswordHash: "$2a$10$test",
		Role:         model.UserRoleChild,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("创建后 ID 应该 != 0")
	}
}

func TestUserRepo_FindByPhone(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 先创建
	repo.Create(ctx, &model.User{
		Openid: "web_13800000001", Nickname: "张三",
		Phone: "13800000001", PasswordHash: "hash", Role: model.UserRoleElder,
	})

	// 能找到
	found, err := repo.FindByPhone(ctx, "13800000001")
	if err != nil {
		t.Fatalf("FindByPhone 失败: %v", err)
	}
	if found.Nickname != "张三" {
		t.Fatalf("昵称不匹配: 期望 张三, 实际 %s", found.Nickname)
	}
	if found.Role != model.UserRoleElder {
		t.Fatalf("角色不匹配: 期望 1, 实际 %d", found.Role)
	}

	// 不存在的手机号
	_, err = repo.FindByPhone(ctx, "99999999999")
	if err == nil {
		t.Fatal("不存在的手机号应该返回错误")
	}
}

func TestUserRepo_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	u := &model.User{Openid: "wx_test", Nickname: "李四", Role: model.UserRoleChild}
	repo.Create(ctx, u)

	found, err := repo.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByID 失败: %v", err)
	}
	if found.Nickname != "李四" {
		t.Fatalf("昵称不匹配")
	}
}

func TestUserRepo_UpdateFields(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	u := &model.User{Openid: "wx_test", Nickname: "旧名", Role: model.UserRoleChild}
	repo.Create(ctx, u)

	err := repo.UpdateFields(ctx, u.ID, map[string]interface{}{"nickname": "新名"})
	if err != nil {
		t.Fatalf("UpdateFields 失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, u.ID)
	if found.Nickname != "新名" {
		t.Fatalf("更新后昵称不匹配")
	}
}

func TestUserRepo_DuplicatePhone(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &model.User{Openid: "w1", Phone: "13800000001", Role: 1})
	err := repo.Create(ctx, &model.User{Openid: "w2", Phone: "13800000001", Role: 2})

	// 注意：数据库层面 phone 没有 unique 约束，所以不会报错
	// 去重逻辑在 Service 层通过 FindByPhone 实现
	if err != nil {
		t.Logf("数据库层面报错（预期内）: %v", err)
	}
}
