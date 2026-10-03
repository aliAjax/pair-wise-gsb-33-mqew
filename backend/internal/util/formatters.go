package util

import (
	"fmt"
	"time"
)

// Shared formatters. Date/time, temperature, status text, plant type text and
// care topic text live together intentionally so a wording change ripples
// through every handler and service that renders text.

// FormatDate renders a time as YYYY-MM-DD.
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// FormatDateTime renders a time as YYYY-MM-DD HH:mm.
func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// FormatTemp renders a temperature range.
func FormatTemp(min, max float64) string {
	if min == 0 && max == 0 {
		return "不限"
	}
	return fmt.Sprintf("%.0f°C ~ %.0f°C", min, max)
}

// PlantTypeText maps a plant type code to Chinese text.
func PlantTypeText(t string) string {
	switch t {
	case "flower":
		return "观花"
	case "foliage":
		return "观叶"
	case "succulent":
		return "多肉"
	case "aquatic":
		return "水生"
	default:
		return "未知"
	}
}

// CareTopicText maps a care topic tag to Chinese text.
func CareTopicText(t string) string {
	switch t {
	case "fertilizing":
		return "施肥"
	case "pruning":
		return "修剪"
	case "repotting":
		return "换盆"
	case "pest_control":
		return "病虫害"
	case "propagation":
		return "繁殖"
	default:
		return "通用"
	}
}

// ReminderStatusText maps a reminder status to Chinese text.
func ReminderStatusText(s string) string {
	switch s {
	case "pending":
		return "待处理"
	case "done":
		return "已完成"
	case "overdue":
		return "已逾期"
	default:
		return "未知"
	}
}

// QuizCategoryText maps a quiz category to Chinese text.
func QuizCategoryText(c string) string {
	switch c {
	case "plant":
		return "品种知识"
	case "article":
		return "养护文章"
	case "pest":
		return "病虫害防治"
	default:
		return "综合"
	}
}

// QuizAttemptStatusText maps a quiz attempt status to Chinese text.
func QuizAttemptStatusText(s string) string {
	switch s {
	case "in_progress":
		return "进行中"
	case "submitted":
		return "已交卷"
	default:
		return "未知"
	}
}

// QuizMyStatusText maps the per-user quiz set status to Chinese text.
func QuizMyStatusText(s string) string {
	switch s {
	case "not_started":
		return "未开始"
	case "in_progress":
		return "进行中"
	case "completed":
		return "已完成"
	default:
		return "未知"
	}
}
