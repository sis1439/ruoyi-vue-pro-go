package types

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func QueryTimeRange(values url.Values, key string) []string {
	for _, name := range []string{key, key + "[]"} {
		if v := values[name]; len(v) == 2 {
			return v
		}
	}
	if a, b := values.Get(key+"[0]"), values.Get(key+"[1]"); a != "" && b != "" {
		return []string{a, b}
	}
	if v := values[key]; len(v) == 1 {
		parts := strings.Split(v[0], ",")
		if len(parts) == 2 {
			return []string{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])}
		}
	}
	return nil
}
func ParseTimeRange(values []string) (time.Time, time.Time, error) {
	var result [2]time.Time
	if len(values) != 2 {
		return result[0], result[1], fmt.Errorf("时间范围必须包含起止时间")
	}
	for i, value := range values {
		if ms, err := strconv.ParseInt(value, 10, 64); err == nil {
			result[i] = time.UnixMilli(ms)
			continue
		}
		for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02"} {
			parsed, err := time.ParseInLocation(layout, strings.TrimSpace(value), time.Local)
			if err == nil {
				result[i] = parsed
				break
			}
		}
		if result[i].IsZero() {
			return result[0], result[1], fmt.Errorf("无效时间: %s", value)
		}
	}
	if result[0].After(result[1]) {
		return result[0], result[1], fmt.Errorf("开始时间不能晚于结束时间")
	}
	return result[0], result[1], nil
}
