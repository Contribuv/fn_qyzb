package utils

import (
	"html/template"
	"strconv"
	"strings"
	"time"
)

func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"add":        add,
		"sub":        sub,
		"mul":        mul,
		"div":        div,
		"until":      until,
		"safeHTML":   safeHTML,
		"formatDate": formatDate,
		"truncate":   truncate,
		"slice":      sliceString,
		"dict":       dict,
		"pageList":   pageList,
		"addInt":     addInt,
		"subInt":     subInt,
	}
}

func addInt(a, b interface{}) int {
	return toInt(a) + toInt(b)
}

func subInt(a, b interface{}) int {
	return toInt(a) - toInt(b)
}

func toInt(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case uint:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	default:
		return 0
	}
}

func pageList(current, total int) []int {
	if total <= 0 {
		return []int{}
	}
	if total <= 7 {
		pages := make([]int, total)
		for i := 0; i < total; i++ {
			pages[i] = i + 1
		}
		return pages
	}
	pages := []int{}
	if current <= 4 {
		for i := 1; i <= 5; i++ {
			pages = append(pages, i)
		}
		pages = append(pages, total)
	} else if current >= total-3 {
		pages = append(pages, 1)
		for i := total - 4; i <= total; i++ {
			pages = append(pages, i)
		}
	} else {
		pages = append(pages, 1)
		for i := current - 1; i <= current+1; i++ {
			pages = append(pages, i)
		}
		pages = append(pages, total)
	}
	return pages
}

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, nil
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, nil
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func mul(a, b int) int {
	return a * b
}

func div(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}

func until(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

// ParseTimeFlexible 兼容解析多种时间字符串：
//   - GORM/glebarez 自动填充格式: "2026-07-07 16:52:35.0400541+08:00"
//   - SQLite CURRENT_TIMESTAMP (UTC): "2026-07-07 08:52:35"
//   - 手动写入的本地时间: "2026-07-07 16:52:35"
//   - RFC3339: "2026-07-07T16:52:35+08:00"
func ParseTimeFlexible(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// FormatTimeString 返回 ISO 8601 字符串和本地化显示文本
func FormatTimeString(s string) (iso, text string) {
	t, ok := ParseTimeFlexible(s)
	if !ok {
		return s, s
	}
	iso = t.Format("2006-01-02T15:04:05Z07:00")
	text = t.Format("2006-01-02 15:04")
	return
}

func formatDate(t interface{}, format string) string {
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	switch v := t.(type) {
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.Format(format)
	case string:
		if v == "" {
			return ""
		}
		parsed, ok := ParseTimeFlexible(v)
		if !ok {
			return v
		}
		return parsed.Format(format)
	default:
		return ""
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func sliceString(start, end int, s string) string {
	runes := []rune(s)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

func ToUint(s string) uint {
	n, _ := strconv.Atoi(s)
	return uint(n)
}

func IntToUint(n int) uint {
	return uint(n)
}

func UintToString(n uint) string {
	return strconv.Itoa(int(n))
}

func EscapeQuotes(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "\"", "\\\"")
}

const DefaultAvatarPath = "/static/images/avatar.png"

func FixAvatarPath(avatar string) string {
	if avatar == "" {
		return DefaultAvatarPath
	}
	if strings.HasPrefix(avatar, "http://") || strings.HasPrefix(avatar, "https://") {
		return avatar
	}
	if strings.HasPrefix(avatar, "/") {
		return avatar
	}
	return "/static/images/" + avatar
}
