package validation

import "strings"

func Required(s string) bool {
	return strings.TrimSpace(s) != ""
}

func MinInt(v, min int) bool {
	return v >= min
}

func GreaterThan(v, min int) bool {
	return v > min
}
