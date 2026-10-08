// Package errcode 定义统一错误码，前端据此做统一拦截处理。
// 错误码分段：1xxx=认证/参数，2xxx=服务端，3xxx=业务。
// 所有 API 响应中的 code 字段必须使用此处定义的常量。
package errcode

// ErrorCode 是统一错误码类型，对应响应 JSON 中的 code 字段。
type ErrorCode int

const (
	// Success 操作成功。
	Success ErrorCode = 0

	// --- 认证与参数校验 (1xxx) ---

	// TokenInvalid JWT Token 无效或已过期。
	TokenInvalid ErrorCode = 1001
	// Forbidden 用户无此操作权限（如老人试图管理药品）。
	Forbidden ErrorCode = 1002
	// InvalidParam 请求参数校验未通过。
	InvalidParam ErrorCode = 1003
	// NotFound 请求的资源不存在。
	NotFound ErrorCode = 1004
	// Conflict 资源冲突（如重复加入家庭组）。
	Conflict ErrorCode = 1005
	// RateLimitExceeded 请求频率超过限流阈值。
	RateLimitExceeded ErrorCode = 1006

	// --- 服务端错误 (2xxx) ---

	// InternalError 服务器内部未预期的错误。
	InternalError ErrorCode = 2001
	// ThirdPartyError 调用第三方服务失败（微信/SMS/Voice/OCR）。
	ThirdPartyError ErrorCode = 2002

	// --- 业务错误 (3xxx) ---

	// InviteCodeInvalid 邀请码无效或已过期。
	InviteCodeInvalid ErrorCode = 3001
	// NotFamilyMember 非家庭组成员，无权访问该家庭组数据。
	NotFamilyMember ErrorCode = 3002
	// MedicineHasSchedule 药品已关联用药计划，不可删除。
	MedicineHasSchedule ErrorCode = 3003
)

// message 错误码到中文提示的映射表。
var message = map[ErrorCode]string{
	Success:             "success",
	TokenInvalid:        "Token无效或已过期",
	Forbidden:           "无操作权限",
	InvalidParam:        "参数校验失败",
	NotFound:            "资源不存在",
	Conflict:            "资源冲突",
	RateLimitExceeded:   "请求频率超限",
	InternalError:       "服务器内部错误",
	ThirdPartyError:     "第三方服务异常",
	InviteCodeInvalid:   "邀请码无效或已过期",
	NotFamilyMember:     "非家庭组成员，无权访问",
	MedicineHasSchedule: "药品已关联用药计划，无法删除",
}

// Message 返回错误码对应的中文提示文案。
func (c ErrorCode) Message() string {
	if msg, ok := message[c]; ok {
		return msg
	}
	return "未知错误"
}

// Int 返回错误码的整数形式。
func (c ErrorCode) Int() int {
	return int(c)
}
