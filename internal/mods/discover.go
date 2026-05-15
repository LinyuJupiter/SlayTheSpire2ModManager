package mods

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

const heyboxSupportSegment = "sts2-heybox-support"

func relContainsHeybox(rel string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(rel), "/") {
		if seg == heyboxSupportSegment {
			return true
		}
	}
	return false
}

// dirEntrySkipSubdir 是否应跳过该目录及其子树（. 开头、Heybox 支持包、符号链接目录）。
func dirEntrySkipSubdir(modsRoot, absPath string, d fs.DirEntry) bool {
	if absPath == modsRoot {
		return false
	}
	if strings.HasPrefix(d.Name(), ".") {
		return true
	}
	rel, err := filepath.Rel(modsRoot, absPath)
	if err != nil {
		return true
	}
	if relContainsHeybox(rel) {
		return true
	}
	if d.IsDir() && d.Type() != 0 && d.Type()&fs.ModeSymlink != 0 {
		return true
	}
	return false
}

// enumerateRelSkipBadTower 跳过「同一段目录名连续重复 3 次及以上」的相对路径（异常嵌套）。
func enumerateRelSkipBadTower(rel string) bool {
	segs := strings.Split(filepath.ToSlash(rel), "/")
	run := 1
	for i := 1; i < len(segs); i++ {
		s := segs[i]
		if s == "" {
			continue
		}
		if s == segs[i-1] {
			run++
			if run >= 3 {
				return true
			}
		} else {
			run = 1
		}
	}
	return false
}

// 常见工具链 / 缓存目录：整棵子树不再扫描。
func shouldSkipEnumerateModsRel(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, seg := range strings.Split(rel, "/") {
		switch strings.ToLower(seg) {
		case "node_modules", ".git", "__pycache__", ".gradle", "packagecache", ".vs", ".idea":
			return true
		}
	}
	return false
}

const maxWalkFilesystemEntries = 250000

// 防止把全盘的 node_modules / .git 等里的 json 全扫进来。
const maxModJSONFilesGlobally = 50000
const maxDistinctManifestDirs = 12000

// 超过该相对深度不再向下枚举，防止异常嵌套目录拖垮扫描。
const maxEnumerateRelDepthSegments = 48

// enumerateModDirsWithManifestJSON 通过一次遍历找出「直接含有 *.json / *.json.bak 文件」的目录（相对 mods 根，用 / 分隔；根下文件为 ""）。
func enumerateModDirsWithManifestJSON(modsRoot string) ([]string, error) {
	modsRoot = filepath.Clean(modsRoot)
	dirSet := map[string]struct{}{}
	var nEntries int
	var jsonFileCount int
	err := filepath.WalkDir(modsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		nEntries++
		if nEntries > maxWalkFilesystemEntries {
			return fmt.Errorf("mods 下文件/目录数量超过单次扫描上限（%d），请移走无关大目录后再试", maxWalkFilesystemEntries)
		}
		if d.IsDir() {
			if path == modsRoot {
				return nil
			}
			if shouldSkipReparsePointDir(path) {
				return filepath.SkipDir
			}
			rel, err := filepath.Rel(modsRoot, path)
			if err == nil {
				if shouldSkipEnumerateModsRel(rel) || enumerateRelSkipBadTower(rel) {
					return filepath.SkipDir
				}
				if strings.Count(filepath.ToSlash(rel), "/")+1 > maxEnumerateRelDepthSegments {
					return filepath.SkipDir
				}
			}
			if dirEntrySkipSubdir(modsRoot, path, d) {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if isIgnoredNonModJSONBaseName(name) {
			return nil
		}
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".json.bak") && !strings.HasSuffix(lower, ".json") {
			return nil
		}
		jsonFileCount++
		if jsonFileCount > maxModJSONFilesGlobally {
			return fmt.Errorf("mods 下 .json/.json.bak 文件过多（>%d），请确认未把 node_modules、游戏缓存等放在 mods 内", maxModJSONFilesGlobally)
		}
		dir := filepath.Dir(path)
		rel, err := filepath.Rel(modsRoot, dir)
		if err != nil {
			return nil
		}
		key := filepath.ToSlash(rel)
		if key == "." {
			key = ""
		}
		dirSet[key] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(dirSet) > maxDistinctManifestDirs {
		return nil, fmt.Errorf("含 manifest 候选 .json 的目录过多（>%d），请精简 mods 目录结构", maxDistinctManifestDirs)
	}
	out := make([]string, 0, len(dirSet))
	for k := range dirSet {
		out = append(out, k)
	}
	return out, nil
}
