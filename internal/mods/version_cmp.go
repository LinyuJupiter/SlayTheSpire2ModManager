package mods

import (
	"strconv"
	"strings"
)

// CompareModVersionStrings 比较 manifest 的 version 字符串；返回 +1 表示 a>b，0 相等，-1 表示 a<b。
// 用于排序与「保留最高版本」；非严格 semver，按段数值优先再字典序。
func CompareModVersionStrings(a, b string) int {
	a = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(a), "v"))
	b = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(b), "v"))
	if a == b {
		return 0
	}
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	n := len(partsA)
	if len(partsB) > n {
		n = len(partsB)
	}
	for i := 0; i < n; i++ {
		var sa, sb string
		if i < len(partsA) {
			sa = partsA[i]
		}
		if i < len(partsB) {
			sb = partsB[i]
		}
		ia, okA := tryParseUintPrefix(sa)
		ib, okB := tryParseUintPrefix(sb)
		if okA && okB {
			if ia != ib {
				if ia > ib {
					return 1
				}
				return -1
			}
			continue
		}
		if c := strings.Compare(sa, sb); c != 0 {
			return c
		}
	}
	return strings.Compare(a, b)
}

func tryParseUintPrefix(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	j := 0
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	if j == 0 {
		return 0, false
	}
	n, err := strconv.ParseUint(s[:j], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
