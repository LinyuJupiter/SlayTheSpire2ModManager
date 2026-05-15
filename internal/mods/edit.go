package mods

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func normalizeFolderKey(s string) string {
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(s))))
}

func validateRelFolderPath(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("文件夹名不能为空")
	}
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
	if rel == "." || rel == ".." {
		return errors.New("文件夹名无效")
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return errors.New("文件夹名无效")
		}
		if strings.HasPrefix(seg, ".") {
			return errors.New("文件夹名不能包含以 . 开头的路径段")
		}
	}
	return nil
}

func splitFolderForEdit(folderName string, layoutNormalized bool) (outer, inner string) {
	fn := strings.Trim(strings.TrimSpace(folderName), `/\`)
	fn = filepath.ToSlash(filepath.Clean(filepath.FromSlash(fn)))
	if fn == "." || fn == "" {
		return "", ""
	}
	if layoutNormalized && isNormTwoSegmentLayout(fn) {
		segs := strings.Split(fn, "/")
		return segs[0], segs[1]
	}
	return fn, ""
}

func joinFolderForEdit(outer, inner string) string {
	outer = strings.Trim(strings.TrimSpace(outer), `/\`)
	inner = strings.Trim(strings.TrimSpace(inner), `/\`)
	if outer == "" {
		return inner
	}
	if inner == "" {
		return outer
	}
	return outer + "/" + inner
}

// SaveEdits 保存对 manifest 与文件夹名的修改。
func SaveEdits(modsRoot string, payload ModEditPayload) error {
	oldFolderPath, err := ResolveSubfolder(modsRoot, payload.FolderName)
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(oldFolderPath, payload.ManifestFile)
	old, err := loadManifestFromPath(jsonPath)
	if err != nil {
		return err
	}
	oldID := old.ID
	newID := strings.TrimSpace(payload.ID)
	if newID == "" {
		return errors.New("id 不能为空")
	}

	overview, err := ListInstalled(modsRoot)
	if err != nil {
		return err
	}
	for _, m := range overview.Mods {
		if m.FolderName == payload.FolderName && m.ManifestFile == payload.ManifestFile {
			continue
		}
		if strings.EqualFold(m.Manifest.ID, newID) || m.Manifest.ID == newID {
			return fmt.Errorf("id 已被其他 mod 使用: %s（%s / %s）", newID, m.FolderName, m.ManifestFile)
		}
	}

	old.ID = newID
	old.Name = payload.Name
	old.Description = payload.Description
	old.AffectsGameplay = payload.AffectsGameplay
	b, err := json.MarshalIndent(old, "", "  ")
	if err != nil {
		return err
	}

	outerOld, innerOld := splitFolderForEdit(payload.FolderName, payload.LayoutNormalized)
	newFolder := strings.TrimSpace(payload.NewFolderName)
	if newFolder == "" {
		newFolder = joinFolderForEdit(outerOld, innerOld)
	}
	if err := validateRelFolderPath(newFolder); err != nil {
		return err
	}
	newOuter, newInner := splitFolderForEdit(newFolder, payload.LayoutNormalized)
	if innerOld != "" && newInner != "" && newInner != innerOld {
		return errors.New("不允许单独修改版本子目录名，请只改最外层文件夹名")
	}
	if innerOld != "" {
		newFolder = joinFolderForEdit(newOuter, innerOld)
		if err := validateRelFolderPath(newFolder); err != nil {
			return err
		}
	}

	workingFolder := oldFolderPath
	if normalizeFolderKey(newFolder) != normalizeFolderKey(payload.FolderName) {
		if innerOld != "" {
			if normalizeFolderKey(newOuter) != normalizeFolderKey(outerOld) {
				oldSlugAbs := filepath.Join(modsRoot, filepath.FromSlash(normalizeFolderKey(outerOld)))
				newSlugAbs := filepath.Join(modsRoot, filepath.FromSlash(normalizeFolderKey(newOuter)))
				if _, err := os.Stat(newSlugAbs); err == nil {
					return fmt.Errorf("目标文件夹已存在: %s", newOuter)
				}
				if err := os.Rename(oldSlugAbs, newSlugAbs); err != nil {
					return fmt.Errorf("重命名 mod 目录: %w", err)
				}
				workingFolder = filepath.Join(newSlugAbs, filepath.FromSlash(innerOld))
			}
		} else {
			destFolder, err := ResolveSubfolder(modsRoot, newFolder)
			if err != nil {
				return err
			}
			if _, err := os.Stat(destFolder); err == nil {
				return fmt.Errorf("目标文件夹已存在: %s", newFolder)
			}
			if err := os.Rename(oldFolderPath, destFolder); err != nil {
				return err
			}
			workingFolder = destFolder
		}
	}

	outJSON := filepath.Join(workingFolder, payload.ManifestFile)
	if err := os.WriteFile(outJSON, b, 0o644); err != nil {
		return err
	}

	if oldID != newID {
		renameIfExists(filepath.Join(workingFolder, oldID+".pck"), filepath.Join(workingFolder, newID+".pck"))
		renameIfExists(filepath.Join(workingFolder, oldID+".dll"), filepath.Join(workingFolder, newID+".dll"))
	}
	return nil
}
