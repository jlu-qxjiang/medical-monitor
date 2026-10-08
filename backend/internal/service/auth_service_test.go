package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/jwt"
	"github.com/medical-monitor/internal/repository"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupServiceTest 创建 MySQL 测试数据库，返回 authSvc。
func setupServiceTest(t *testing.T) (*AuthService, *gorm.DB) {
	rootDB, _ := gorm.Open(mysql.Open("root:123456@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	dbName := "medical_monitor_svc_test"
	rootDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
	rootDB.Exec(fmt.Sprintf("CREATE DATABASE `%s` DEFAULT CHARSET utf8mb4", dbName))

	dsn := fmt.Sprintf("root:123456@tcp(127.0.0.1:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal("failed to open test db:", err)
	}
	db.AutoMigrate(
		&model.User{}, &model.Family{}, &model.FamilyMember{},
		&model.RefreshToken{},
	)

	userRepo := repository.NewUserRepository(db)
	familyRepo := repository.NewFamilyRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db, nil)
	jwtManager := jwt.NewJWTManager("test-secret-key-32bytes-long!!", 168*time.Hour, 720*time.Hour)

	// 用 nil wechatSvc（不测微信登录）
	svc := NewAuthService(userRepo, refreshTokenRepo, jwtManager, nil, familyRepo)
	return svc, db
}

func TestAuthService_RegisterAndLogin(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	// 1. 注册
	resp, err := svc.RegisterByPassword(ctx, "13800000001", "测试员", "123456", model.UserRoleChild)
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("注册后应该返回 Token")
	}
	if resp.User.Role != model.UserRoleChild {
		t.Fatalf("角色不匹配: 期望 2, 实际 %d", resp.User.Role)
	}
	if !resp.User.IsNewUser {
		t.Fatal("新用户 IsNewUser 应为 true")
	}

	// 2. 等 1 秒避免 Token 时间戳相同导致哈希冲突
	time.Sleep(time.Second)
	resp2, err := svc.LoginByPassword(ctx, "13800000001", "123456")
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if resp2.User.IsNewUser {
		t.Fatal("老用户 IsNewUser 应为 false")
	}
}

func TestAuthService_LoginWrongPassword(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	svc.RegisterByPassword(ctx, "13800000002", "测试", "right-pwd", model.UserRoleChild)

	// 错误密码
	_, err := svc.LoginByPassword(ctx, "13800000002", "wrong-pwd")
	if err == nil {
		t.Fatal("错误密码应该返回错误")
	}
}

func TestAuthService_DuplicatePhone(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	svc.RegisterByPassword(ctx, "13800000003", "用户A", "123456", model.UserRoleChild)
	_, err := svc.RegisterByPassword(ctx, "13800000003", "用户B", "654321", model.UserRoleElder)
	if err == nil {
		t.Fatal("重复手机号应该返回错误")
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	// 1. 注册获取 Token 对
	resp, _ := svc.RegisterByPassword(ctx, "13800000004", "测试", "123456", model.UserRoleChild)

	// 2. 用 Refresh Token 换新 Token 对
	refreshResp, err := svc.RefreshTokens(ctx, resp.RefreshToken)
	if err != nil {
		t.Fatalf("刷新 Token 失败: %v", err)
	}
	if refreshResp.AccessToken == "" || refreshResp.RefreshToken == "" {
		t.Fatal("刷新后应该返回新 Token")
	}

	// 3. 旧 Refresh Token 应该失效（Token 轮换）
	_, err = svc.RefreshTokens(ctx, resp.RefreshToken)
	if err == nil {
		t.Fatal("旧 Refresh Token 被撤销后应该失效")
	}
}

func TestAuthService_GetProfile(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	resp, _ := svc.RegisterByPassword(ctx, "13800000005", "王五", "123456", model.UserRoleElder)

	profile, err := svc.GetProfile(ctx, resp.User.ID)
	if err != nil {
		t.Fatalf("GetProfile 失败: %v", err)
	}
	if profile.Nickname != "王五" {
		t.Fatalf("昵称不匹配")
	}
}
