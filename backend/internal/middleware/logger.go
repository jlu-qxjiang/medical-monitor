// Package middleware 提供 Gin 框架的中间件组件。
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerMiddleware 记录每个 HTTP 请求的结构化日志。
// 日志包含请求方法、路径、状态码、响应耗时、客户端 IP 和 User-Agent。
//
// 日志级别根据状态码自动调整：
//   - 5xx → Error 级别（服务端错误）
//   - 4xx → Warn 级别（客户端错误）
//   - 2xx/3xx → Info 级别（正常请求）
//
// 使用 zap 结构化日志库，输出 JSON 格式方便日志收集系统（如 ELK、Loki）解析。
type LoggerMiddleware struct {
	logger *zap.Logger // 结构化日志实例
}

// NewLoggerMiddleware 创建 LoggerMiddleware 实例。
func NewLoggerMiddleware(logger *zap.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{logger: logger}
}

// Handler 返回 Gin 中间件处理函数。
// 记录请求开始时间，在请求处理后输出带有历时和状态的日志。
func (l *LoggerMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间，用于计算处理耗时
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 继续处理请求（包含后续中间件和 handler）
		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// 将 query string 格式化（加 ? 前缀）
		if query != "" {
			query = "?" + query
		}

		// 构建结构化日志字段
		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Duration("latency", latency),                // 请求处理耗时
			zap.String("client_ip", c.ClientIP()),           // 客户端真实 IP
			zap.String("user_agent", c.Request.UserAgent()), // User-Agent 头
		}

		// 如果上下文中存在 user_id（已认证的用户），添加到日志字段
		if uid := GetUserID(c); uid != 0 {
			fields = append(fields, zap.Uint64("user_id", uid))
		}

		// 根据 HTTP 状态码选择合适的日志级别
		if statusCode >= 500 {
			l.logger.Error("request completed with server error", fields...)
		} else if statusCode >= 400 {
			l.logger.Warn("request completed with client error", fields...)
		} else {
			l.logger.Info("request completed", fields...)
		}
	}
}
