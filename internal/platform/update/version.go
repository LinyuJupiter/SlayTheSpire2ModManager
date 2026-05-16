package update

import (
	"strconv"
	"strings"
)

// CompareVersions returns +1 when a>b, 0 when equal, -1 when a<b.
func CompareVersions(a, b string) int {
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
		ia, okA := parseUintPrefix(sa)
		ib, okB := parseUintPrefix(sb)
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

func parseUintPrefix(s string) (uint64, bool) {
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
