package mods

import (
	"fmt"
	"io/fs"
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

	err := filepath.WalkDir(modsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if path == modsRoot {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		discovered, err := findAllManifestsInFolder(path)
		if err != nil || len(discovered) == 0 {
			return nil
		}
		rel, err := filepath.Rel(modsRoot, path)
		if err != nil {
			return nil
		}
		folderName := filepath.ToSlash(rel)
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
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	for id, locs := range idToLocs {
		if len(locs) > 1 {
			out.DuplicateIDs = append(out.DuplicateIDs, id)
		}
	}
	sort.Strings(out.DuplicateIDs)
	sort.Slice(out.Mods, func(i, j int) bool {
		a, b := out.Mods[i], out.Mods[j]
		fa, fb := strings.ToLower(a.FolderName), strings.ToLower(b.FolderName)
		if fa != fb {
			return a.FolderName < b.FolderName
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
	return out, nil
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
