package update

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// IsHelperInvocation reports whether args request update-helper mode.
func IsHelperInvocation(args []string) bool {
	for _, arg := range args {
		if arg == helperArg {
			return true
		}
	}
	return false
}

// RunHelper runs the replacement workflow. It is intended to be called from main
// before Wails starts.
func RunHelper(args []string) int {
	fs := flag.NewFlagSet("update-helper", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pid := fs.Int("pid", 0, "process id to wait for")
	src := fs.String("src", "", "downloaded exe path")
	dst := fs.String("dst", "", "target exe path")
	restart := fs.Bool("restart", false, "restart target after replacement")
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != helperArg {
			filtered = append(filtered, arg)
		}
	}
	if err := fs.Parse(filtered); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *pid <= 0 || *src == "" || *dst == "" {
		fmt.Fprintln(os.Stderr, "update-helper: missing required arguments")
		return 2
	}
	waitForProcessExit(*pid)
	if err := replaceExecutable(*src, *dst); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *restart {
		cmd := exec.Command(*dst)
		cmd.Dir = filepath.Dir(*dst)
		_ = cmd.Start()
	}
	return 0
}

func waitForProcessExit(pid int) {
	proc, err := os.FindProcess(pid)
	if err == nil {
		_, _ = proc.Wait()
	}
	// Process handles may not be waitable on every platform. The replacement loop
	// below is the authoritative readiness check.
	time.Sleep(500 * time.Millisecond)
}

func replaceExecutable(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	backup := dst + ".bak"
	var lastErr error
	for i := 0; i < 60; i++ {
		_ = os.Remove(backup)
		if _, err := os.Stat(dst); err == nil {
			if err := os.Rename(dst, backup); err != nil {
				lastErr = err
				time.Sleep(500 * time.Millisecond)
				continue
			}
		}
		if err := copyFile(src, dst, 0o755); err != nil {
			lastErr = err
			_ = os.Rename(backup, dst)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		_ = os.Remove(backup)
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("替换程序失败: %w", lastErr)
	}
	return fmt.Errorf("替换程序失败: helper pid %d", os.Getpid())
}
