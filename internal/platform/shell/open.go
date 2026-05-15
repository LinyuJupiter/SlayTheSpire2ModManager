// Package shell 在系统文件管理器中打开目录。
package shell

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenDirectory 在资源管理器 / Finder / 默认文件管理器中打开 path。
func OpenDirectory(path string) error {
	path = filepath.Clean(path)
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
