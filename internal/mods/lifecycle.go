package mods

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func normalizeManifestBaseName(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(name), ".bak")
}

// disableOtherSameIDEnabled 关闭除 keepFolder 下 keepManifestAnyName（可为 .json 或 .json.bak）外、同 id 且当前处于启用状态的其它 mod。
func disableOtherSameIDEnabled(modsRoot, keepFolder, keepManifestAnyName string) error {
	folder, err := ResolveSubfolder(modsRoot, keepFolder)
	if err != nil {
		return err
	}
	mp := filepath.Join(folder, strings.TrimSpace(keepManifestAnyName))
	b, err := readModManifestBytes(mp)
	if err != nil {
		return err
	}
	if !quickProbablyModManifest(b) {
		return errors.New("manifest 无效")
	}
	man, ok := tryParseModManifest(b)
	if !ok {
		return errors.New("manifest 无效")
	}
	id := man.ID
	keepStem := normalizeManifestBaseName(keepManifestAnyName)

	ov, err := ListInstalled(modsRoot)
	if err != nil {
		return err
	}
	var toDisable []modInstanceKey
	for _, m := range ov.Mods {
		if m.Manifest.ID != id {
			continue
		}
		if m.FolderName == keepFolder && normalizeManifestBaseName(m.ManifestFile) == keepStem {
			continue
		}
		if !m.Disabled {
			toDisable = append(toDisable, modInstanceKey{m.FolderName, m.ManifestFile})
		}
	}
	for _, t := range toDisable {
		if err := Disable(modsRoot, t.Folder, t.Manifest); err != nil {
			return err
		}
	}
	return nil
}

func renameIfExists(from, to string) error {
	_, err := os.Stat(from)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	_ = os.Remove(to)
	return os.Rename(from, to)
}

// enableHighestRemainingForID 在删除启用实例后，若仍有同 id 则启用 version 最高的一份（含 .bak）。
func enableHighestRemainingForID(modsRoot, id string) error {
	ov, err := ListInstalled(modsRoot)
	if err != nil {
		return err
	}
	var cand []InstalledMod
	for _, m := range ov.Mods {
		if m.Manifest.ID == id {
			cand = append(cand, m)
		}
	}
	if len(cand) == 0 {
		return nil
	}
	var on []InstalledMod
	for _, m := range cand {
		if !m.Disabled {
			on = append(on, m)
		}
	}
	if len(on) > 1 {
		return DedupeEnabledRetainHighestPerSlugID(modsRoot)
	}
	if len(on) == 1 {
		return nil
	}
	sort.Slice(cand, func(i, j int) bool {
		return CompareModVersionStrings(cand[i].Manifest.Version, cand[j].Manifest.Version) > 0
	})
	best := cand[0]
	return ActivateModVersion(modsRoot, best.FolderName, best.ManifestFile)
}

// DeleteMod deleteEntireSlug 为 true 时删除整棵 slug 目录；false 时删除 manifest 所在版本目录（整文件夹）。
// 若删除的是当前启用中的版本，则自动启用剩余版本中 version 最高者。
func DeleteMod(modsRoot, folderName, manifestFile string, deleteEntireSlug bool) error {
	folderName = strings.TrimSpace(folderName)
	manifestFile = strings.TrimSpace(manifestFile)
	folder, err := ResolveSubfolder(modsRoot, folderName)
	if err != nil {
		return err
	}
	mp := filepath.Join(folder, manifestFile)
	man, err := loadManifestFromPath(mp)
	if err != nil {
		return err
	}
	id := man.ID
	wasEnabled := !strings.HasSuffix(strings.ToLower(manifestFile), ".json.bak")

	rel, err := filepath.Rel(modsRoot, folder)
	if err != nil {
		return err
	}
	relSlash := filepath.ToSlash(rel)

	var absDelete string
	if deleteEntireSlug {
		if isNormTwoSegmentLayout(relSlash) {
			segs := strings.Split(relSlash, "/")
			absDelete = filepath.Join(modsRoot, filepath.FromSlash(segs[0]))
		} else {
			absDelete = folder
		}
	} else {
		absDelete = folder
	}
	if err := os.RemoveAll(absDelete); err != nil && !os.IsNotExist(err) {
		return err
	}
	if wasEnabled {
		return enableHighestRemainingForID(modsRoot, id)
	}
	return nil
}

// Disable 关闭 mod：将 manifest 与对应 pck/dll 加 .bak。
func Disable(modsRoot, folderName, manifestFile string) error {
	folder, err := ResolveSubfolder(modsRoot, folderName)
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(folder, manifestFile)
	if strings.HasSuffix(strings.ToLower(manifestFile), ".json.bak") {
		return errors.New("该 mod 已处于关闭状态")
	}
	man, err := loadManifestFromPath(jsonPath)
	if err != nil {
		return err
	}
	id := man.ID
	bakPath := jsonPath + ".bak"
	if _, err := os.Stat(bakPath); err == nil {
		return errors.New("已存在文件: " + filepath.Base(bakPath))
	}
	if err := os.Rename(jsonPath, bakPath); err != nil {
		return err
	}
	renameIfExists(filepath.Join(folder, id+".pck"), filepath.Join(folder, id+".pck.bak"))
	renameIfExists(filepath.Join(folder, id+".dll"), filepath.Join(folder, id+".dll.bak"))
	return nil
}

// Enable 根据 *.json.bak 还原 manifest 与 pck/dll。
func Enable(modsRoot, folderName, manifestBakFile string) error {
	manifestBakFile = strings.TrimSpace(manifestBakFile)
	if !strings.HasSuffix(strings.ToLower(manifestBakFile), ".json.bak") {
		return errors.New("请指定关闭状态的 manifest 文件（*.json.bak）")
	}
	folder, err := ResolveSubfolder(modsRoot, folderName)
	if err != nil {
		return err
	}
	oldPath := filepath.Join(folder, manifestBakFile)
	man, err := loadManifestFromPath(oldPath)
	if err != nil {
		return err
	}
	id := man.ID
	if err := disableOtherSameIDEnabled(modsRoot, folderName, manifestBakFile); err != nil {
		return err
	}
	trimmed := strings.TrimSuffix(manifestBakFile, ".bak")
	if trimmed == manifestBakFile {
		return errors.New("manifest 文件名格式异常")
	}
	newPath := filepath.Join(folder, trimmed)
	if _, err := os.Stat(newPath); err == nil {
		return errors.New("已存在 " + trimmed + "，无法启用（请先处理冲突文件）")
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	renameIfExists(filepath.Join(folder, id+".pck.bak"), filepath.Join(folder, id+".pck"))
	renameIfExists(filepath.Join(folder, id+".dll.bak"), filepath.Join(folder, id+".dll"))
	return nil
}

// ActivateModVersion 将指定 manifest 作为该 id 的唯一启用版本：先关闭其它路径上同 id 的启用实例，若目标为 .json.bak 则再执行启用。
func ActivateModVersion(modsRoot, folderName, manifestFile string) error {
	folderName = strings.TrimSpace(folderName)
	manifestFile = strings.TrimSpace(manifestFile)
	if folderName == "" || manifestFile == "" {
		return fmt.Errorf("路径参数为空")
	}
	folder, err := ResolveSubfolder(modsRoot, folderName)
	if err != nil {
		return err
	}
	mp := filepath.Join(folder, manifestFile)
	if _, err := os.Stat(mp); err != nil {
		return fmt.Errorf("manifest 不存在: %w", err)
	}
	if err := disableOtherSameIDEnabled(modsRoot, folderName, manifestFile); err != nil {
		return err
	}
	if strings.HasSuffix(strings.ToLower(manifestFile), ".json.bak") {
		return Enable(modsRoot, folderName, manifestFile)
	}
	return nil
}
