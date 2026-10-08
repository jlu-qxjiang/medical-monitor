// Package middleware 提供 Gin 框架的中间件组件。
package middleware

import (
	"net"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/pkg/errcode"
	"github.com/medical-monitor/internal/pkg/response"
	"go.uber.org/zap"
)

// RecoveryMiddleware 实现 panic 恢复中间件，确保单个请求的 panic 不会导致整个服务进程崩溃。
//
// 恢复策略：
//  1. 捕获 handler 链中发生的所有 panic
//  2. 判断 panic 类型：
//     - net.OpError（通常是客户端断开连接 Broken Pipe）→ Warn 级别日志，不做 500 响应
//     - 其他 panic → Error 级别日志，记录完整调用栈，返回 500 Internal Server Error
//  3. 使用 zap 记录错误详情和完整的 goroutine 堆栈信息
//
// 使用方式：
//
//	router.Use(recoveryMiddleware.Handler())
//	// Recovery 中间件应该是第一个注册的中间件，确保所有 panic 都能被捕获
type RecoveryMiddleware struct {
	logger *zap.Logger // 结构化日志实例
}

// NewRecoveryMiddleware 创建 RecoveryMiddleware 实例。
func NewRecoveryMiddleware(logger *zap.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{logger: logger}
}

// Handler 返回 Gin 中间件处理函数。
// 使用 defer + recover 捕获 panic。
func (r *RecoveryMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// 情况 1：Broken Pipe（客户端在服务端响应前断开连接）
				// 这种错误无需返回 500，也无需记录完整堆栈，因为不是代码 Bug
				if ne, ok := rec.(*net.OpError); ok {
					r.logger.Warn("broken pipe, client disconnected",
						zap.Error(ne),
					)
					c.Abort()
					return
				}

				// 情况 2：代码级 panic（nil 指针、数组越界等）
				// 记录完整调用栈便于排查根因
				r.logger.Error("panic recovered",
					zap.Any("panic", rec),
					zap.String("stack", string(debug.Stack())), // 完整的 goroutine 堆栈
				)

				// 返回统一的 500 错误给客户端，不暴露内部错误详情
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					response.Response{
						Code:    errcode.InternalError.Int(),
						Message: "服务器内部错误，请稍后重试",
					},
				)
			}
		}()
		c.Next()
	}
}
