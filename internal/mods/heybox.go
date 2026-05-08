package mods

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EnsureHeyboxSupport 若尚未解压 Heybox 支持包，则从 embeddedZip 写入 mods 目录。
func EnsureHeyboxSupport(modsDir string, embeddedZip []byte) error {
	if heyboxSupportPresent(modsDir) {
		return nil
	}
	return extractEmbeddedHeybox(modsDir, embeddedZip)
}

func heyboxSupportPresent(modsDir string) bool {
	dll := filepath.Join(modsDir, "sts2-heybox-support", "sts2-heybox-support.dll")
	st, err := os.Stat(dll)
	return err == nil && !st.IsDir()
}

func extractEmbeddedHeybox(modsDir string, embeddedZip []byte) error {
	if len(embeddedZip) == 0 {
		return errors.New("内置 sts2-heybox-support.zip 缺失，请重新构建应用")
	}
	r, err := zip.NewReader(bytes.NewReader(embeddedZip), int64(len(embeddedZip)))
	if err != nil {
		return err
	}
	destRoot := filepath.Clean(modsDir)
	for _, f := range r.File {
		rel := filepath.FromSlash(f.Name)
		if rel == "" || rel == "." {
			continue
		}
		outPath := filepath.Join(destRoot, rel)
		cleanOut := filepath.Clean(outPath)
		cleanDest := filepath.Clean(destRoot)
		if cleanOut != cleanDest && !strings.HasPrefix(cleanOut+string(os.PathSeparator), cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("非法压缩包路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		wf, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(wf, rc)
		close1 := rc.Close()
		close2 := wf.Close()
		if copyErr != nil {
			return copyErr
		}
		if close1 != nil {
			return close1
		}
		if close2 != nil {
			return close2
		}
	}
	return nil
}
