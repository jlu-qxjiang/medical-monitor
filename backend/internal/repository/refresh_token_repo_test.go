package repository

import (
	"context"
	"testing"
	"time"
)

func TestRefreshTokenRepo_StoreAndValidate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db, nil) // 不连 Redis
	ctx := context.Background()

	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 天后
	err := repo.Store(ctx, 1, "test-token-abc", expiresAt)
	if err != nil {
		t.Fatalf("Store 失败: %v", err)
	}

	// 校验正确的 Token
	valid, err := repo.Validate(ctx, 1, "test-token-abc")
	if err != nil {
		t.Fatalf("Validate 失败: %v", err)
	}
	if !valid {
		t.Fatal("应该返回 valid=true")
	}
}

func TestRefreshTokenRepo_ValidateWrongToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db, nil)
	ctx := context.Background()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	repo.Store(ctx, 1, "correct-token", expiresAt)

	// 校验错误的 Token
	valid, err := repo.Validate(ctx, 1, "wrong-token")
	if err != nil {
		t.Fatalf("Validate 报错: %v", err)
	}
	if valid {
		t.Fatal("错误的 Token 应该返回 valid=false")
	}
}

func TestRefreshTokenRepo_Revoke(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db, nil)
	ctx := context.Background()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	repo.Store(ctx, 1, "token-to-revoke", expiresAt)

	// 撤销前 → 有效
	valid, _ := repo.Validate(ctx, 1, "token-to-revoke")
	if !valid {
		t.Fatal("撤销前应该有效")
	}

	// 撤销
	if err := repo.Revoke(ctx, "token-to-revoke"); err != nil {
		t.Fatalf("Revoke 失败: %v", err)
	}

	// 撤销后 → 无效
	valid, _ = repo.Validate(ctx, 1, "token-to-revoke")
	if valid {
		t.Fatal("撤销后应该无效")
	}
}

func TestRefreshTokenRepo_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db, nil)
	ctx := context.Background()

	// Token 1 小时前就过期了
	expiresAt := time.Now().Add(-1 * time.Hour)
	repo.Store(ctx, 1, "expired-token", expiresAt)

	valid, _ := repo.Validate(ctx, 1, "expired-token")
	if valid {
		t.Fatal("过期的 Token 应该无效")
	}
}

func TestRefreshTokenRepo_RevokeAllForUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db, nil)
	ctx := context.Background()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	repo.Store(ctx, 1, "token-1", expiresAt)
	repo.Store(ctx, 1, "token-2", expiresAt)
	repo.Store(ctx, 2, "token-3", expiresAt) // 另一个用户

	// 撤销用户 1 的所有 Token
	if err := repo.RevokeAllForUser(ctx, 1); err != nil {
		t.Fatalf("RevokeAllForUser 失败: %v", err)
	}

	// 用户 1 的 Token 都无效了
	v1, _ := repo.Validate(ctx, 1, "token-1")
	v2, _ := repo.Validate(ctx, 1, "token-2")
	if v1 || v2 {
		t.Fatal("用户 1 的 Token 应该全部无效")
	}

	// 用户 2 的 Token 不受影响
	v3, _ := repo.Validate(ctx, 2, "token-3")
	if !v3 {
		t.Fatal("用户 2 的 Token 不受影响，应该有效")
	}
}
