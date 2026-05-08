package importer

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ZIP 通用目的位：bit 11 表示文件名/注释为 UTF-8（PKZIP App Note）。
const zipUTF8Flag = 0x800

func decodeZIPEntryName(raw string, flags uint16) string {
	if flags&zipUTF8Flag != 0 {
		return raw
	}
	dec := simplifiedchinese.GBK.NewDecoder()
	out, _, err := transform.String(dec, raw)
	if err != nil {
		return raw
	}
	return out
}

func unzipZIPDecoded(srcZip, destDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer r.Close()
	destRoot := filepath.Clean(destDir)
	for _, f := range r.File {
		name := decodeZIPEntryName(f.Name, f.Flags)
		rel := filepath.FromSlash(name)
		if rel == "" {
			continue
		}
		if strings.Contains(rel, "..") {
			continue
		}
		outPath := filepath.Join(destRoot, rel)
		cleanOut := filepath.Clean(outPath)
		cleanDest := filepath.Clean(destRoot)
		if cleanOut != cleanDest && !strings.HasPrefix(cleanOut+string(os.PathSeparator), cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("非法压缩包路径: %s", name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, f.Mode()); err != nil {
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
		_ = rc.Close()
		_ = wf.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}
