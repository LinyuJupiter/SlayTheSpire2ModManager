package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListInstalled 扫描 mods 根目录（含任意层级子文件夹）并返回全部有效 mod 条目。
func ListInstalled(modsRoot string) (*ModsOverview, error) {
	modsRoot = filepath.Clean(modsRoot)
	out := &ModsOverview{ModsDir: modsRoot, Mods: []InstalledMod{}, DuplicateIDs: []string{}}
	idToLocs := map[string][]modInstanceKey{}

	dirKeys, err := enumerateModDirsWithManifestJSON(modsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	sort.Strings(dirKeys)

	for _, folderName := range dirKeys {
		dir := filepath.Join(modsRoot, filepath.FromSlash(folderName))
		discovered, ferr := findAllManifestsInFolder(dir)
		if ferr != nil || len(discovered) == 0 {
			continue
		}
		for _, me := range discovered {
			id := me.Man.ID
			idToLocs[id] = append(idToLocs[id], modInstanceKey{folderName, me.FileName})
			out.Mods = append(out.Mods, InstalledMod{
				FolderName:   folderName,
				ManifestFile: me.FileName,
				Disabled:     me.Disabled,
				Manifest:     me.Man,
			})
		}
	}
	for id, locs := range idToLocs {
		if len(locs) <= 1 {
			continue
		}
		groups := map[string]struct{}{}
		for _, loc := range locs {
			groups[folderConflictGroupKey(loc.Folder)] = struct{}{}
		}
		if len(groups) > 1 {
			out.DuplicateIDs = append(out.DuplicateIDs, id)
		}
	}
	sort.Strings(out.DuplicateIDs)
	sort.Slice(out.Mods, func(i, j int) bool {
		a, b := out.Mods[i], out.Mods[j]
		ga, gb := installedSlugGroupKey(a), installedSlugGroupKey(b)
		if ga != gb {
			return strings.ToLower(a.FolderName) < strings.ToLower(b.FolderName)
		}
		if cmp := CompareModVersionStrings(a.Manifest.Version, b.Manifest.Version); cmp != 0 {
			return cmp > 0
		}
		return a.ManifestFile < b.ManifestFile
	})
	allIDs := map[string]struct{}{}
	for _, m := range out.Mods {
		allIDs[m.Manifest.ID] = struct{}{}
	}
	for i := range out.Mods {
		cur := modInstanceKey{out.Mods[i].FolderName, out.Mods[i].ManifestFile}
		id := out.Mods[i].Manifest.ID
		locs := idToLocs[id]
		var conflicts []string
		for _, loc := range locs {
			if loc.Folder == cur.Folder && loc.Manifest == cur.Manifest {
				continue
			}
			if sameSlugVersionSiblings(out.Mods[i].FolderName, loc.Folder) {
				continue
			}
			if loc.Folder == cur.Folder {
				conflicts = append(conflicts, fmt.Sprintf("同目录「%s」", loc.Manifest))
			} else {
				conflicts = append(conflicts, fmt.Sprintf("文件夹「%s」", loc.Folder))
			}
		}
		sort.Strings(conflicts)
		out.Mods[i].ConflictWith = conflicts
		out.Mods[i].IDUnique = len(conflicts) == 0

		var missing []string
		for _, dep := range out.Mods[i].Manifest.Dependencies {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			if _, ok := allIDs[dep]; !ok {
				missing = append(missing, dep)
			}
		}
		sort.Strings(missing)
		out.Mods[i].MissingDependencies = missing
		out.Mods[i].Available = out.Mods[i].IDUnique && len(missing) == 0
	}
	enrichModsMeta(out)
	return out, nil
}

func installedSlugGroupKey(m InstalledMod) string {
	s := strings.Trim(strings.TrimSpace(m.FolderName), `/\`)
	if isNormTwoSegmentLayout(s) {
		segs := strings.Split(filepath.ToSlash(s), "/")
		return strings.ToLower(segs[0])
	}
	return strings.ToLower(s)
}

func isNormTwoSegmentLayout(folderName string) bool {
	s := strings.Trim(strings.TrimSpace(folderName), `/\`)
	segs := strings.Split(filepath.ToSlash(s), "/")
	if len(segs) != 2 {
		return false
	}
	for _, seg := range segs {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
		if strings.HasPrefix(seg, ".") {
			return false
		}
	}
	return true
}

// folderConflictGroupKey 用于判断「同 id 是否算冲突」：同一 slug 下两段式 slug/版本 归为同一组，不算跨文件夹重复。
func folderConflictGroupKey(folderName string) string {
	s := strings.Trim(strings.TrimSpace(folderName), `/\`)
	if !isNormTwoSegmentLayout(s) {
		return "flat:" + strings.ToLower(filepath.ToSlash(s))
	}
	segs := strings.Split(filepath.ToSlash(s), "/")
	return "norm:" + strings.ToLower(segs[0])
}

// sameSlugVersionSiblings 两段式 layout 且第一段（slug）相同，视为同一 mod 的多版本目录，不记 id 冲突。
func sameSlugVersionSiblings(folderA, folderB string) bool {
	if folderA == folderB {
		return false
	}
	if !isNormTwoSegmentLayout(folderA) || !isNormTwoSegmentLayout(folderB) {
		return false
	}
	return folderConflictGroupKey(folderA) == folderConflictGroupKey(folderB)
}

func enrichModsMeta(out *ModsOverview) {
	byID := map[string][]int{}
	for i := range out.Mods {
		id := out.Mods[i].Manifest.ID
		byID[id] = append(byID[id], i)
	}
	for i := range out.Mods {
		id := out.Mods[i].Manifest.ID
		var alt []ModVersionRef
		for _, j := range byID[id] {
			if j == i {
				continue
			}
			m := out.Mods[j]
			alt = append(alt, ModVersionRef{
				FolderName:   m.FolderName,
				ManifestFile: m.ManifestFile,
				Disabled:     m.Disabled,
			})
		}
		sort.Slice(alt, func(a, b int) bool {
			if alt[a].FolderName != alt[b].FolderName {
				return alt[a].FolderName < alt[b].FolderName
			}
			return alt[a].ManifestFile < alt[b].ManifestFile
		})
		out.Mods[i].AlternateVersions = alt
		out.Mods[i].LayoutNormalized = isNormTwoSegmentLayout(out.Mods[i].FolderName)
	}
}

// EnsureDirectory 确保游戏 exe 旁存在 mods 目录并返回其路径。
func EnsureDirectory(gameExe string) (string, error) {
	root := filepath.Dir(gameExe)
	mods := filepath.Join(root, "mods")
	if err := os.MkdirAll(mods, 0o755); err != nil {
		return "", err
	}
	return mods, nil
}
