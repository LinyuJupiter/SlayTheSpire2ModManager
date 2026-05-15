package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func shouldSkipNormalizePath(modsRoot, absPath string) bool {
	rel, err := filepath.Rel(modsRoot, absPath)
	if err != nil {
		return true
	}
	return relContainsHeybox(rel)
}

// NormalizeLayout 将 mods 目录整理为「slug/版本目录」结构；同一目录多 mod 时，无主文件复制到各目标目录。
func NormalizeLayout(modsRoot string) (*NormalizeReport, error) {
	modsRoot = filepath.Clean(modsRoot)
	rep := &NormalizeReport{Migrated: []string{}, Skipped: []string{}, Errors: []string{}}
	st, err := os.Stat(modsRoot)
	if err != nil {
		return nil, fmt.Errorf("mods 目录无效: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("mods 路径不是目录")
	}

	const maxPasses = 80
	for pass := 0; pass < maxPasses; pass++ {
		dirs, err := collectManifestHostDirs(modsRoot)
		if err != nil {
			return rep, err
		}
		progress := false
		for _, relKey := range dirs {
			abs := filepath.Join(modsRoot, filepath.FromSlash(relKey))
			if shouldSkipNormalizePath(modsRoot, abs) {
				continue
			}
			discovered, err := findAllManifestsInFolder(abs)
			if err != nil || len(discovered) == 0 {
				continue
			}
			if !needsNormalize(modsRoot, abs, discovered) {
				continue
			}
			relBefore := relKey
			if relBefore == "" {
				relBefore = "."
			}
			if len(discovered) == 1 {
				err = normalizeSingleModDir(modsRoot, abs, discovered[0])
			} else {
				err = normalizeMultiModDir(modsRoot, abs, discovered)
			}
			if err != nil {
				rep.Errors = append(rep.Errors, fmt.Sprintf("%s: %v", filepath.ToSlash(relBefore), err))
				continue
			}
			progress = true
			rep.Migrated = append(rep.Migrated, filepath.ToSlash(relBefore))
		}
		if !progress {
			break
		}
	}
	if err := DedupeEnabledRetainHighestPerSlugID(modsRoot); err != nil {
		rep.Errors = append(rep.Errors, fmt.Sprintf("dedupe 启用版本: %v", err))
	}
	return rep, nil
}

func collectManifestHostDirs(modsRoot string) ([]string, error) {
	dirs, err := enumerateModDirsWithManifestJSON(modsRoot)
	if err != nil {
		return nil, err
	}
	sortDirsByDepthDesc(dirs)
	return dirs, nil
}

func sortDirsByDepthDesc(dirs []string) {
	sort.Slice(dirs, func(i, j int) bool {
		return depth(dirs[i]) > depth(dirs[j])
	})
}

func depth(p string) int {
	return strings.Count(filepath.ToSlash(filepath.Clean(p)), "/")
}

func needsNormalize(modsRoot, absDir string, discovered []manifestDiscovered) bool {
	if len(discovered) == 0 {
		return false
	}
	if len(discovered) > 1 {
		return true
	}
	me := discovered[0]
	rel, err := filepath.Rel(modsRoot, absDir)
	if err != nil {
		return true
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return true
	}
	segs := strings.Split(rel, "/")
	if len(segs) == 2 {
		s0 := strings.ToLower(segs[0])
		if s0 == strings.ToLower(slugFolderFromModID(me.Man)) {
			return false
		}
		if s0 == strings.ToLower(slugCandidate(me.Man)) {
			return false
		}
	}
	return true
}

func slugCandidate(man ModManifest) string {
	if s := SanitizeFolderName(man.Name); s != "" {
		return s
	}
	return SanitizeFolderName(man.ID)
}

// slugFolderFromModID mods 根下 slug 目录名：仅由 manifest id 派生（导入与规范化新建目录用）。
func slugFolderFromModID(man ModManifest) string {
	return strings.TrimSpace(SanitizeFolderName(man.ID))
}

func verCandidate(man ModManifest) string {
	if s := SanitizeFolderName(man.Version); s != "" {
		return s
	}
	return "default"
}

// pickSlugDirFromID 返回 mods 根下 slug 目录名：不存在则新建名；已存在且为目录则复用；为文件则换名。
func pickSlugDirFromID(modsRoot string, man ModManifest) (string, error) {
	base := slugFolderFromModID(man)
	if base == "" {
		base = "mod"
	}
	for i := 0; ; i++ {
		cand := base
		if i > 0 {
			cand = fmt.Sprintf("%s_%d", base, i)
		}
		p := filepath.Join(modsRoot, cand)
		st, err := os.Stat(p)
		if os.IsNotExist(err) {
			return cand, nil
		}
		if err != nil {
			return "", err
		}
		if st.IsDir() {
			return cand, nil
		}
	}
}

func uniqueChildUnder(parent, base string) (string, error) {
	b := SanitizeFolderName(base)
	if b == "" {
		b = "default"
	}
	for i := 0; ; i++ {
		cand := b
		if i > 0 {
			cand = fmt.Sprintf("%s_%d", b, i)
		}
		p := filepath.Join(parent, cand)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return cand, nil
		}
	}
}

func normalizeSingleModDir(modsRoot, absDir string, me manifestDiscovered) error {
	slug, err := pickSlugDirFromID(modsRoot, me.Man)
	if err != nil {
		return err
	}
	parent := filepath.Join(modsRoot, slug)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	ver, err := uniqueChildUnder(parent, verCandidate(me.Man))
	if err != nil {
		return err
	}
	dest := filepath.Join(parent, ver)
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("目标已存在: %s/%s", slug, ver)
	}
	if err := os.Rename(absDir, dest); err == nil {
		return nil
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if err := moveDirContents(absDir, dest); err != nil {
		return err
	}
	// absDir 已是 slug 根目录且 dest 为其子目录 slug/ver 时，移动后仍会留下子目录 ver，不能删 absDir。
	if filepath.Clean(absDir) == filepath.Clean(parent) {
		return nil
	}
	return os.Remove(absDir)
}

func normalizeMultiModDir(modsRoot, absDir string, discovered []manifestDiscovered) error {
	type plan struct {
		me   manifestDiscovered
		dest string
	}
	var plans []plan
	for _, me := range discovered {
		slug, err := pickSlugDirFromID(modsRoot, me.Man)
		if err != nil {
			return err
		}
		parent := filepath.Join(modsRoot, slug)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
		ver, err := uniqueChildUnder(parent, verCandidate(me.Man))
		if err != nil {
			return err
		}
		dest := filepath.Join(parent, ver)
		if err := os.MkdirAll(dest, 0o755); err != nil {
			return err
		}
		plans = append(plans, plan{me: me, dest: dest})
	}

	claimed := map[string]struct{}{}
	ids := map[string]struct{}{}
	for _, me := range discovered {
		claimed[me.FileName] = struct{}{}
		ids[me.Man.ID] = struct{}{}
	}
	for id := range ids {
		for _, suf := range []string{id + ".pck", id + ".pck.bak", id + ".dll", id + ".dll.bak"} {
			p := filepath.Join(absDir, suf)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				claimed[suf] = struct{}{}
			}
		}
	}

	entries, err := listEntryBaseNamesInDir(absDir)
	if err != nil {
		return err
	}
	var unclaimed []string
	for _, name := range entries {
		if _, ok := claimed[name]; !ok {
			unclaimed = append(unclaimed, name)
		}
	}

	for _, pl := range plans {
		me := pl.me
		from := filepath.Join(absDir, me.FileName)
		to := filepath.Join(pl.dest, me.FileName)
		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("移动 manifest %s: %w", me.FileName, err)
		}
		id := me.Man.ID
		for _, suf := range []string{id + ".pck", id + ".pck.bak", id + ".dll", id + ".dll.bak"} {
			fp := filepath.Join(absDir, suf)
			if _, err := os.Stat(fp); err != nil {
				continue
			}
			if err := os.Rename(fp, filepath.Join(pl.dest, suf)); err != nil {
				return fmt.Errorf("移动 %s: %w", suf, err)
			}
		}
	}

	for _, name := range unclaimed {
		src := filepath.Join(absDir, name)
		for _, pl := range plans {
			dst := filepath.Join(pl.dest, name)
			if err := copyPathRecursive(src, dst); err != nil {
				return fmt.Errorf("复制无主 %s: %w", name, err)
			}
		}
		if err := os.RemoveAll(src); err != nil {
			return fmt.Errorf("删除原无主项 %s: %w", name, err)
		}
	}

	_ = os.Remove(absDir)
	return nil
}

func moveDirContents(src, dst string) error {
	names, err := listEntryBaseNamesInDir(src)
	if err != nil {
		return err
	}
	for _, name := range names {
		from := filepath.Join(src, name)
		to := filepath.Join(dst, name)
		if filepath.Clean(from) == filepath.Clean(to) {
			continue
		}
		// 不可将子项移入其自身子路径（典型：slug 下已有与版本同名的子目录 ver，
		// 目标 dest 为 slug/ver 时会把 slug/ver 挪成 slug/ver/ver 无限嵌套）。
		if pathIsStrictDescendant(from, to) {
			continue
		}
		st, err := os.Stat(from)
		if err != nil {
			continue
		}
		if err := os.Rename(from, to); err != nil {
			if err := copyPathRecursive(from, to); err != nil {
				return err
			}
			if st.IsDir() {
				_ = os.RemoveAll(from)
			} else {
				_ = os.Remove(from)
			}
		}
	}
	return nil
}
