//go:build windows

package mods

import "golang.org/x/sys/windows"

// shouldSkipReparsePointDir 为 true 时不进入该目录，避免目录联接/挂载点把 Walk 引入无关巨型树（常见爆内存原因）。
func shouldSkipReparsePointDir(abs string) bool {
	if abs == "" {
		return false
	}
	p, err := windows.UTF16PtrFromString(abs)
	if err != nil {
		return false
	}
	a, err := windows.GetFileAttributes(p)
	if err != nil || a == windows.INVALID_FILE_ATTRIBUTES {
		return false
	}
	return a&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
