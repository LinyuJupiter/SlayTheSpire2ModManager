package update

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

const helperArg = "--sts2-update-helper"

// InstallLatest downloads the latest update and starts a temporary helper process.
// The caller should exit the main app immediately after this returns nil.
func InstallLatest(ctx context.Context, currentVersion string) error {
	info, err := CheckLatest(ctx, currentVersion)
	if err != nil {
		return err
	}
	if !info.HasUpdate {
		return fmt.Errorf("当前已是最新版本")
	}
	src, err := DownloadLatest(ctx, info)
	if err != nil {
		return err
	}
	return StartHelper(src)
}

// StartHelper copies the currently running executable to a temp helper path and
// runs it in update-helper mode. The helper waits for this process to exit before
// replacing the original executable.
func StartHelper(newExePath string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, err = filepath.Abs(currentExe)
	if err != nil {
		return err
	}
	helperDir, err := os.MkdirTemp("", "sts2-mod-manager-updater-*")
	if err != nil {
		return err
	}
	helperPath := filepath.Join(helperDir, "ModManager-updater.exe")
	if err := copyFile(currentExe, helperPath, 0o755); err != nil {
		_ = os.RemoveAll(helperDir)
		return err
	}
	cmd := exec.Command(helperPath,
		helperArg,
		"--pid", strconv.Itoa(os.Getpid()),
		"--src", newExePath,
		"--dst", currentExe,
		"--restart",
	)
	cmd.Dir = filepath.Dir(currentExe)
	return cmd.Start()
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
