package mods

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ResolveSubfolder 返回 mods 根目录下子文件夹的绝对路径，并防止目录穿越。
// folderName 可为多级相对路径（如 dir1/dir2），使用正斜杠或系统分隔符均可。
func ResolveSubfolder(modsRoot, folderName string) (string, error) {
	modsRoot = filepath.Clean(modsRoot)
	folderName = strings.TrimSpace(folderName)
	if folderName == "" {
		return "", fmt.Errorf("文件夹名为空")
	}
	rel := filepath.Clean(filepath.FromSlash(folderName))
	if rel == "." || rel == ".." {
		return "", fmt.Errorf("无效的 mod 路径")
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("无效的 mod 路径")
	}
	folder := filepath.Join(modsRoot, rel)
	folder = filepath.Clean(folder)
	if folder == modsRoot || !strings.HasPrefix(folder+string(os.PathSeparator), modsRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("无效的 mod 路径")
	}
	return folder, nil
}

// SanitizeFolderName 去除 Windows 非法字符与空白，生成可作为 mod slug / version 目录名的片段。
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

// pathIsStrictDescendant 当 child 位于 parent 目录树内部（且不等于 parent）时为 true。
func pathIsStrictDescendant(parentPath, childPath string) bool {
	pp := filepath.Clean(parentPath)
	cc := filepath.Clean(childPath)
	if pp == cc {
		return false
	}
	rel, err := filepath.Rel(pp, cc)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..")
}

// listEntryBaseNamesInDir 分批列出目录下条目名称（跳过以 . 开头的名称），避免单目录海量条目 OOM。
func listEntryBaseNamesInDir(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	const chunk = 4096
	for {
		names, rdErr := f.Readdirnames(chunk)
		for _, name := range names {
			if strings.HasPrefix(name, ".") {
				continue
			}
			out = append(out, name)
		}
		if rdErr != nil {
			if rdErr != io.EOF {
				return nil, rdErr
			}
			break
		}
		if len(names) < chunk {
			break
		}
	}
	return out, nil
}

// copyPathRecursive 将 src 文件或目录树复制到 dst（dst 不存在则创建）。
func copyPathRecursive(src, dst string) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dst, rel)
			if d.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			return copyFileMode(path, target, info.Mode())
		})
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return copyFileMode(src, dst, st.Mode())
}

func copyFileMode(src, dst string, mode fs.FileMode) error {
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
