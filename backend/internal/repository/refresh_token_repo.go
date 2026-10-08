package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RefreshTokenRepository 管理 Refresh Token 的持久化与缓存。
// Redis 作为热数据缓存（TTL 自动过期），MySQL 作为冷数据持久化。
// 读取路径：Redis 命中 → 直接返回；Miss → MySQL 查询 → 回写 Redis。
//
// 方法列表：
//   - Store: 存储 Refresh Token（Redis 缓存 + MySQL 持久化双写）[refresh_tokens + Redis]
//   - Validate: 校验 Token 是否有效（Redis 优先，Miss 查 MySQL 回写）[refresh_tokens + Redis]
//   - Revoke: 撤销单个 Token（删 Redis + 标记 MySQL）[refresh_tokens + Redis]
//   - RevokeAllForUser: 撤销某用户全部 Token [refresh_tokens + Redis]
//   - CleanExpired: 清理过期 Token [refresh_tokens]
type RefreshTokenRepository struct {
	db    *gorm.DB
	rdb   *redis.Client // 可选，nil 时仅用 MySQL
}

func NewRefreshTokenRepository(db *gorm.DB, rdb *redis.Client) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db, rdb: rdb}
}

func tokenKey(hash string) string { return "rt:" + hash }

// Store 将 Refresh Token 写入 Redis（TTL 过期）+ MySQL（持久化）。
func (r *RefreshTokenRepository) Store(ctx context.Context, userID uint64, token string, expiresAt time.Time) error {
	hash := hashToken(token)

	// MySQL 持久化
	record := &model.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return err
	}

	// Redis 缓存（TTL = 存活时间）
	if r.rdb != nil {
		ttl := time.Until(expiresAt)
		if ttl > 0 {
			_ = r.rdb.Set(ctx, tokenKey(hash), userID, ttl).Err()
		}
	}
	return nil
}

// Validate 校验 Token：Redis 命中直接返回；Miss 时查 MySQL 并回写 Redis。
func (r *RefreshTokenRepository) Validate(ctx context.Context, userID uint64, token string) (bool, error) {
	hash := hashToken(token)
	key := tokenKey(hash)

	// 1. Redis 先查
	if r.rdb != nil {
		val, err := r.rdb.Get(ctx, key).Result()
		if err == nil && val != "" {
			return true, nil
		}
	}

	// 2. MySQL 兜底
	var record model.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?", hash, userID, time.Now()).
		First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	// 3. 回写 Redis
	if r.rdb != nil {
		ttl := time.Until(record.ExpiresAt)
		if ttl > 0 {
			_ = r.rdb.Set(ctx, key, userID, ttl).Err()
		}
	}
	return true, nil
}

// Revoke 撤销 Token：删 Redis + 标记 MySQL。
func (r *RefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	hash := hashToken(token)
	now := time.Now()

	// Redis 立即删除
	if r.rdb != nil {
		_ = r.rdb.Del(ctx, tokenKey(hash)).Err()
	}

	// MySQL 标记撤销
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", now).Error
}

// RevokeAllForUser 撤销用户所有 Token。
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uint64) error {
	now := time.Now()
	// Redis：扫描删除（生产环境可用 SET 存储用户所有 token 列表）
	if r.rdb != nil {
		var cursor uint64
		for {
			keys, nextCursor, err := r.rdb.Scan(ctx, cursor, "rt:*", 100).Result()
			if err == nil {
				for _, k := range keys {
					uid, _ := r.rdb.Get(ctx, k).Uint64()
					if uid == userID {
						r.rdb.Del(ctx, k)
					}
				}
			}
			cursor = nextCursor
			if cursor == 0 { break }
		}
	}
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (r *RefreshTokenRepository) CleanExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at <= ?", time.Now()).
		Delete(&model.RefreshToken{})
	return result.RowsAffected, result.Error
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
