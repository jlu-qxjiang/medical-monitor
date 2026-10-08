// Package utils 提供通用工具函数。
// 包含邀请码生成、手机号脱敏、时间格式化、指针转换等高频使用的纯函数。
package utils

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"
)

// GenerateInviteCode 生成 6 位随机数字邀请码（密码学安全随机数）。
func GenerateInviteCode() (string, error) {
	max := big.NewInt(1000000) // [0, 1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("生成邀请码失败: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// ParseUint64 将字符串转为 uint64。
func ParseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// TimeToTimeStr 将 time.Time 格式化为 "HH:MM" 字符串。
func TimeToTimeStr(t time.Time) string {
	return t.Format("15:04")
}

// TimeStrToTime 将 "HH:MM" 或 "HH:MM:SS" 字符串解析为今日对应时刻。
func TimeStrToTime(timeStr string) (time.Time, error) {
	parsed, err := time.Parse("15:04", timeStr)
	if err != nil {
		parsed, err = time.Parse("15:04:05", timeStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("无效的时间格式: %s", timeStr)
		}
	}
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(),
		parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.Local), nil
}

// Today 返回今日日期字符串 "2006-01-02"。
// 用于 reminders 表 remind_date 字段的赋值和查询。
func Today() string {
	return time.Now().Format("2006-01-02")
}

// FormatTimeForJSON 将 *time.Time 格式化为 RFC3339 字符串指针，nil 输入返回 nil。
func FormatTimeForJSON(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// MaskPhone 手机号脱敏显示: "13812345678" → "138****5678"。
func MaskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// GetLast4Digits 取手机号末尾 4 位，用于 phone 字段明文存储。
func GetLast4Digits(phone string) string {
	if len(phone) < 4 {
		return phone
	}
	return phone[len(phone)-4:]
}

// TakenOffsetMinutes 计算实际服药时间相对计划时间的偏移（分钟）。
// 正数表示超时服药，负数表示提前服药，0 表示准时。
func TakenOffsetMinutes(remindTime, takenAt time.Time) int {
	return int(math.Round(takenAt.Sub(remindTime).Minutes()))
}

// FormatTakenOffset 将服药时间偏移格式化为中文可读文本。
// 示例："提前30分钟"、"超时1小时15分钟"、"准时"。
func FormatTakenOffset(remindTime, takenAt time.Time) string {
	offset := TakenOffsetMinutes(remindTime, takenAt)
	if offset == 0 {
		return "准时"
	}
	totalMin := offset
	if totalMin < 0 {
		totalMin = -totalMin
	}
	h := totalMin / 60
	m := totalMin % 60
	var s string
	if h > 0 {
		s = fmt.Sprintf("%d小时", h)
		if m > 0 {
			s += fmt.Sprintf("%d分钟", m)
		}
	} else {
		s = fmt.Sprintf("%d分钟", m)
	}
	if offset < 0 {
		return "提前" + s
	}
	return "超时" + s
}
