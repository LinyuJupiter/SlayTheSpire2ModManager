//go:build !windows

package steam

// FindSlayTheSpire2Exe 非 Windows 平台返回空。
func FindSlayTheSpire2Exe() string {
	return ""
}
