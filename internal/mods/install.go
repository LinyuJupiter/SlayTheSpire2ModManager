package mods

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstallFromExtractedTree 将已解压到 extractedRoot 的目录树安装到 modsRoot：
// 按 manifest id 合并到已有 slug 目录；新建时顶层文件夹名为 id（经合法化）；版本为子目录。
func InstallFromExtractedTree(extractedRoot, modsRoot string) error {
	modsRoot = filepath.Clean(modsRoot)
	entries, err := os.ReadDir(extractedRoot)
	if err != nil {
		return err
	}
	var visible []os.DirEntry
	for _, e := range entries {
		if e.Name() == "." || e.Name() == ".." {
			continue
		}
		visible = append(visible, e)
	}
	if len(visible) == 0 {
		return errors.New("压缩包为空")
	}

	var topDirs []os.DirEntry
	for _, e := range visible {
		if e.IsDir() {
			topDirs = append(topDirs, e)
		}
	}

	// 经典：根下只有一个文件夹，mod 在该文件夹内
	if len(visible) == 1 && len(topDirs) == 1 {
		return installOneModTree(filepath.Join(extractedRoot, visible[0].Name()), modsRoot)
	}

	// 根下多个并列文件夹：每个顶层文件夹内递归查找一份 mod（允许嵌套子目录）
	if len(topDirs) >= 2 {
		for _, d := range topDirs {
			from := filepath.Join(extractedRoot, d.Name())
			if err := installOneModTree(from, modsRoot); err != nil {
				return fmt.Errorf("文件夹 %q: %w", d.Name(), err)
			}
		}
		return nil
	}

	// 根下没有文件夹（仅散落 dll/json/pck 等）：整棵根当作一个 mod
	if len(topDirs) == 0 {
		return installOneModTree(extractedRoot, modsRoot)
	}

	// 恰好一个子文件夹 + 根上还有其它文件：若根上能读到 manifest 则视为「平铺式」整包导入
	if _, err := readFirstManifestInDir(extractedRoot); err == nil {
		return installOneModTree(extractedRoot, modsRoot)
	}
	// 否则 manifest 只在唯一子目录内
	if len(topDirs) == 1 {
		return installOneModTree(filepath.Join(extractedRoot, topDirs[0].Name()), modsRoot)
	}

	return errors.New("无法识别的压缩包结构")
}

func existingSlugDirForModID(modsRoot, id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}
	ov, err := ListInstalled(modsRoot)
	if err != nil {
		return "", false
	}
	for _, m := range ov.Mods {
		if m.Manifest.ID != id {
			continue
		}
		s := slugFirstSegment(m.FolderName)
		if s != "" {
			return s, true
		}
	}
	return "", false
}

// allocTopLevelSlugForNewMod 为「尚无该 id」的导入分配顶层目录名：优先 id，若与已有其它 mod 的目录冲突则 id_N。
func allocTopLevelSlugForNewMod(modsRoot string, man ModManifest) (string, error) {
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
		if !st.IsDir() {
			continue
		}
		disc, ferr := findAllManifestsInFolder(p)
		if ferr != nil {
			continue
		}
		allSame := true
		for _, d := range disc {
			if d.Man.ID != man.ID {
				allSame = false
				break
			}
		}
		if len(disc) == 0 || allSame {
			return cand, nil
		}
	}
}

func dirEntrySkipModDiscovery(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	if strings.EqualFold(name, "__MACOSX") {
		return true
	}
	return false
}

// findModSourceDir 在 start 目录树中广度优先查找「直接含有有效 manifest 的目录」作为 mod 根目录。
func findModSourceDir(start string) (string, ModManifest, error) {
	start = filepath.Clean(start)
	st, err := os.Stat(start)
	if err != nil {
		return "", ModManifest{}, err
	}
	if !st.IsDir() {
		return "", ModManifest{}, errors.New("不是目录")
	}
	const maxDepth = 32
	const maxVisited = 4000
	visited := 0
	type node struct {
		path  string
		depth int
	}
	q := []node{{start, 0}}
	for len(q) > 0 {
		cur := q[0].path
		dep := q[0].depth
		q = q[1:]
		if visited >= maxVisited {
			return "", ModManifest{}, errors.New("扫描目录过多，请将 manifest 移近压缩包根或减少嵌套")
		}
		visited++
		if dep > maxDepth {
			continue
		}
		if man, err := readFirstManifestInDir(cur); err == nil {
			return cur, man, nil
		}
		entries, err := os.ReadDir(cur)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || dirEntrySkipModDiscovery(e.Name()) {
				continue
			}
			q = append(q, node{filepath.Join(cur, e.Name()), dep + 1})
		}
	}
	return "", ModManifest{}, errors.New("未在目录树中找到有效的 mod manifest（需在某一目录下放置 *.json）")
}

func installOneModTree(srcDir, modsRoot string) error {
	modRoot, man, err := findModSourceDir(srcDir)
	if err != nil {
		return fmt.Errorf("无法在目录内找到有效 mod manifest: %w", err)
	}
	if err := assertVersionNotInstalled(modsRoot, man.ID, man.Version); err != nil {
		return err
	}
	slugDir, haveExisting := existingSlugDirForModID(modsRoot, man.ID)
	if !haveExisting {
		slugDir, err = allocTopLevelSlugForNewMod(modsRoot, man)
		if err != nil {
			return err
		}
	}
	destAbs := filepath.Join(modsRoot, filepath.FromSlash(slugDir))
	st, err := os.Stat(destAbs)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(destAbs, 0o755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !st.IsDir() {
		return fmt.Errorf("目标路径已存在且非目录: %s", slugDir)
	}
	verDir, err := uniqueChildUnder(destAbs, verCandidate(man))
	if err != nil {
		return err
	}
	target := filepath.Join(destAbs, verDir)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("版本目录已存在: %s/%s", slugDir, verDir)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	return copyPathRecursive(modRoot, target)
}

func readFirstManifestInDir(dir string) (ModManifest, error) {
	en, bak, err := listManifestJSONBaseNames(dir)
	if err != nil {
		return ModManifest{}, err
	}
	for _, name := range append(en, bak...) {
		if strings.HasSuffix(strings.ToLower(name), ".json.bak") {
			continue
		}
		p := filepath.Join(dir, name)
		m, err := loadManifestFromPath(p)
		if err == nil {
			return m, nil
		}
	}
	return ModManifest{}, errors.New("无可用 manifest")
}

func assertVersionNotInstalled(modsRoot, id, version string) error {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	ov, err := ListInstalled(modsRoot)
	if err != nil {
		return err
	}
	for _, m := range ov.Mods {
		if m.Manifest.ID != id {
			continue
		}
		if versionsConsideredEqual(m.Manifest.Version, version) {
			return fmt.Errorf("已安装相同版本: %s (%s)", id, version)
		}
	}
	return nil
}

func versionsConsideredEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	return CompareModVersionStrings(a, b) == 0
}
