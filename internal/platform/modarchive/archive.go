// Package modarchive contains archive-related infrastructure used by the app layer.
package modarchive

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ExtractArchiveToTemp 将压缩包解压到新建临时目录；cleanup 务必调用以删除临时文件。
func ExtractArchiveToTemp(archivePath string) (tmp string, cleanup func(), err error) {
	st, err := os.Stat(archivePath)
	if err != nil {
		return "", nil, err
	}
	if st.IsDir() {
		return "", nil, fmt.Errorf("请选择压缩文件而非文件夹")
	}
	tmp, err = os.MkdirTemp("", "sts2-mod-import-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(tmp) }
	ext := strings.ToLower(filepath.Ext(archivePath))
	if ext == ".zip" {
		if err := unzipZIPDecoded(archivePath, tmp); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("解压 zip 失败: %w", err)
		}
	} else {
		if err := archiver.Unarchive(archivePath, tmp); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("解压失败（若为 RAR 请尝试 ZIP）: %w", err)
		}
	}
	return tmp, cleanup, nil
}

// ZipDirectory 将目录下所有文件写入 zip；ZIP 内路径为「文件夹名/相对路径」。
func ZipDirectory(srcDir, zipPath string) error {
	srcDir = filepath.Clean(srcDir)
	rootName := filepath.Base(srcDir)
	if rootName == "." || rootName == "" {
		rootName = "mod"
	}
	zf, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zf.Close()
	zw := zip.NewWriter(zf)
	defer zw.Close()

	var fileCount int
	err = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(filepath.Join(rootName, rel))
		info, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = rel
		hdr.Method = zip.Deflate
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		fileCount++
		return nil
	})
	if err != nil {
		return err
	}
	if fileCount == 0 {
		return fmt.Errorf("文件夹内没有可打包的文件")
	}
	return nil
}

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
