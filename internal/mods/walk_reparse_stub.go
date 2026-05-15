//go:build !windows

package mods

// shouldSkipReparsePointDir 在非 Windows 上无额外判定（依赖 dirEntrySkipSubdir 对 ModeSymlink 的处理）。
func shouldSkipReparsePointDir(absDir string) bool {
	_ = absDir
	return false
}
