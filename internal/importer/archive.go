package importer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
)

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ImportArchive 将 zip/rar 解压到 modsRoot 下适当子目录。
func ImportArchive(archivePath, modsRoot, optionalFolderName string) error {
	st, err := os.Stat(archivePath)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return fmt.Errorf("请选择压缩文件而非文件夹")
	}
	tmp, err := os.MkdirTemp("", "sts2-mod-import-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	ext := strings.ToLower(filepath.Ext(archivePath))
	if ext == ".zip" {
		if err := unzipZIPDecoded(archivePath, tmp); err != nil {
			return fmt.Errorf("解压 zip 失败: %w", err)
		}
	} else {
		if err := archiver.Unarchive(archivePath, tmp); err != nil {
			return fmt.Errorf("解压失败（若为 RAR 请尝试 ZIP）: %w", err)
		}
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return err
	}
	visible := []os.DirEntry{}
	for _, e := range entries {
		if e.Name() == "." || e.Name() == ".." {
			continue
		}
		visible = append(visible, e)
	}
	if len(visible) == 0 {
		return errors.New("压缩包为空")
	}
	if len(visible) == 1 && visible[0].IsDir() {
		srcDir := filepath.Join(tmp, visible[0].Name())
		destName := visible[0].Name()
		if strings.TrimSpace(optionalFolderName) != "" {
			destName = SanitizeFolderName(optionalFolderName)
			if destName == "" {
				return errors.New("无效的文件夹名称")
			}
		}
		dest := filepath.Join(modsRoot, destName)
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("目标已存在: %s", destName)
		}
		return copyTree(srcDir, dest)
	}
	base := strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath))
	folderName := optionalFolderName
	if strings.TrimSpace(folderName) == "" {
		folderName = base
	}
	folderName = SanitizeFolderName(folderName)
	if folderName == "" {
		return errors.New("无效的文件夹名称")
	}
	dest := filepath.Join(modsRoot, folderName)
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("目标已存在: %s", folderName)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for _, e := range visible {
		from := filepath.Join(tmp, e.Name())
		to := filepath.Join(dest, e.Name())
		if e.IsDir() {
			if err := copyTree(from, to); err != nil {
				return err
			}
		} else {
			if err := copyFile(from, to, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// SanitizeFolderName 去除 Windows 非法字符与空白。
func SanitizeFolderName(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			continue
		default:
			if r < 32 {
				continue
			}
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
