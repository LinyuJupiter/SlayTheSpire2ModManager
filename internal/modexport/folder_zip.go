// Package modexport 将 mod 子文件夹导出为 zip。
package modexport

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ResolvedSubfolder 返回 mods 根目录下子文件夹的绝对路径，并防止目录穿越。
func ResolvedSubfolder(modsRoot, folderName string) (string, error) {
	modsRoot = filepath.Clean(modsRoot)
	folderName = strings.TrimSpace(folderName)
	if folderName == "" {
		return "", fmt.Errorf("文件夹名为空")
	}
	folder := filepath.Join(modsRoot, folderName)
	folder = filepath.Clean(folder)
	if folder == modsRoot || !strings.HasPrefix(folder+string(os.PathSeparator), modsRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("无效的 mod 路径")
	}
	return folder, nil
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
