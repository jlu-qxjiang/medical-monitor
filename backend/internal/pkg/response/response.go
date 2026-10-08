// Package response 提供统一 JSON 响应格式。
// 所有 Handler 必须使用 Success / SuccessPage / Error 系列函数封装返回值，
// 禁止直接 c.JSON(...) 裸写。
//
// 统一格式: { "code": 0, "message": "success", "data": {...} }
// 分页格式: { "code": 0, "data": { "list": [...], "total": N, "page": 1, "page_size": 20 } }
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medical-monitor/internal/pkg/errcode"
)

// Response 统一 JSON 响应体。
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResponse 统一分页数据结构，嵌套在 Response.Data 中。
type PageResponse struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success 返回 200 + code=0 + data 的通用成功响应。
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errcode.Success.Int(),
		Message: errcode.Success.Message(),
		Data:    data,
	})
}

// SuccessMessage 返回仅含提示信息的成功响应（如"更新成功"）。
func SuccessMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    errcode.Success.Int(),
		Message: message,
	})
}

// SuccessPage 返回 200 + code=0 + 分页数据。
func SuccessPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    errcode.Success.Int(),
		Message: errcode.Success.Message(),
		Data: PageResponse{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// Error 返回业务错误响应，HTTP 状态码根据错误码自动映射。
// customMsg 为空时使用错误码默认中文提示。
func Error(c *gin.Context, code errcode.ErrorCode, customMsg string) {
	msg := code.Message()
	if customMsg != "" {
		msg = customMsg
	}
	c.JSON(httpStatusFromCode(code), Response{
		Code:    code.Int(),
		Message: msg,
	})
}

// ErrorWithData 返回带附加数据的错误响应（如表单校验详情）。
func ErrorWithData(c *gin.Context, code errcode.ErrorCode, customMsg string, data interface{}) {
	msg := code.Message()
	if customMsg != "" {
		msg = customMsg
	}
	c.JSON(httpStatusFromCode(code), Response{
		Code:    code.Int(),
		Message: msg,
		Data:    data,
	})
}

// httpStatusFromCode 将业务错误码映射为合适的 HTTP 状态码。
func httpStatusFromCode(code errcode.ErrorCode) int {
	switch code {
	case errcode.TokenInvalid:
		return http.StatusUnauthorized // 401
	case errcode.Forbidden, errcode.NotFamilyMember:
		return http.StatusForbidden // 403
	case errcode.InvalidParam, errcode.InviteCodeInvalid, errcode.MedicineHasSchedule:
		return http.StatusBadRequest // 400
	case errcode.NotFound:
		return http.StatusNotFound // 404
	case errcode.Conflict:
		return http.StatusConflict // 409
	case errcode.RateLimitExceeded:
		return http.StatusTooManyRequests // 429
	case errcode.ThirdPartyError:
		return http.StatusBadGateway // 502
	default:
		return http.StatusInternalServerError // 500
	}
}
