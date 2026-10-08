// Package middleware 提供 Gin 框架的中间件组件。
package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware 基于令牌桶算法（Token Bucket）的 API 限流中间件。
//
// 双层限流策略：
//  1. 全局限流（globalRate）：所有请求共享一个全局令牌桶，防止分布式攻击打垮服务
//  2. 按 IP 限流（rate）：每个客户端 IP 拥有独立的令牌桶，防止单 IP 恶意刷接口
//
// 技术细节：
//   - 以客户端 IP 地址作为限流键（key），每个 IP 独立令牌桶
//   - 令牌以固定速率补充（rate），最大突发容量为 burst
//   - 当桶内没有令牌时，返回 429 Too Many Requests
//   - 使用 sync.RWMutex 保护并发访问（读多写少场景用 RWMutex 性能更优）
//   - 后台协程每10分钟清理一次超过10分钟未访问的 IP 条目，防止内存泄漏
//   - 令牌桶使用 golang.org/x/time/rate 标准库实现
//
// 使用方式：
//
//	// 每 IP 每秒 50 个请求、突发 100；全局每秒 3000 个请求
//	rateLimitMiddleware := NewRateLimitMiddleware(50, 100, 3000)
//	router.Use(rateLimitMiddleware.Handler())
type RateLimitMiddleware struct {
	mu            sync.RWMutex                // 保护 limiters map 的并发安全
	limiters      map[string]*rateLimiterEntry // IP → 令牌桶 映射
	rate          rate.Limit                   // 每 IP 令牌补充速率（请求数/秒）
	burst         int                          // 每 IP 最大突发容量（令牌桶容量）
	globalLimiter *rate.Limiter                // 全局令牌桶（防分布式攻击）
}

// rateLimiterEntry 包装了令牌桶及其最后访问时间。
// lastSeen 用于后台协程判断是否可以清理该条目。
type rateLimiterEntry struct {
	limiter  *rate.Limiter // 该 IP 的令牌桶实例
	lastSeen time.Time     // 最后一次请求的时间
}

// NewRateLimitMiddleware 创建 RateLimitMiddleware 实例。
//
// 参数：
//   - r: 令牌补充速率（如 rate.Limit(5) 表示每秒5个请求）
//   - burst: 令牌桶容量（最大突发请求数）
func NewRateLimitMiddleware(r rate.Limit, burst int, globalRate rate.Limit) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiters:      make(map[string]*rateLimiterEntry),
		rate:          r,
		burst:         burst,
		globalLimiter: rate.NewLimiter(globalRate, int(globalRate)*2), // 全局突发 = 2 倍速率
	}
}

// Handler 返回 Gin 中间件处理函数。
// 对每个请求：获取或创建该 IP 的令牌桶，尝试消耗一个令牌。
// 如果令牌不足，返回 429 错误并中止请求。
func (rl *RateLimitMiddleware) Handler() gin.HandlerFunc {
	// 启动后台协程清理过期条目
	go rl.cleanupExpired()

	return func(c *gin.Context) {
		// ① 全局限流（最后防线，防分布式攻击）
		if !rl.globalLimiter.Allow() {
			response.Error(c, errcode.RateLimitExceeded, "")
			c.Abort()
			return
		}

		// ② 按 IP 限流（防单 IP 恶意刷）
		// 使用客户端 IP 作为限流键
		key := c.ClientIP()

		rl.mu.Lock()
		entry, exists := rl.limiters[key]
		if !exists {
			// 首次访问该 IP：创建新的令牌桶
			entry = &rateLimiterEntry{
				limiter:  rate.NewLimiter(rl.rate, rl.burst),
				lastSeen: time.Now(),
			}
			rl.limiters[key] = entry
		}
		entry.lastSeen = time.Now() // 更新最后访问时间
		rl.mu.Unlock()

		// 尝试获取令牌，如果失败则返回 429
		if !entry.limiter.Allow() {
			response.Error(c, errcode.RateLimitExceeded, "")
			c.Abort()
			return
		}

		c.Next()
	}
}

// cleanupExpired 定期清理超过10分钟未访问的 IP 条目。
// 在后台协程中运行，每10分钟执行一次扫描。
// 这样可以防止长期未访问的 IP 条目导致的内存持续增长。
func (rl *RateLimitMiddleware) cleanupExpired() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for key, entry := range rl.limiters {
			// 超过10分钟未访问 → 删除
			if time.Since(entry.lastSeen) > 10*time.Minute {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}
