package mods

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ModManager/internal/modexport"
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

// SaveEdits 保存对 manifest 与文件夹名的修改。
func SaveEdits(modsRoot string, payload ModEditPayload) error {
	newFolder := strings.TrimSpace(payload.NewFolderName)
	if newFolder == "" {
		newFolder = payload.FolderName
	}
	if err := validateRelFolderPath(newFolder); err != nil {
		return err
	}

	oldFolderPath, err := modexport.ResolvedSubfolder(modsRoot, payload.FolderName)
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

	workingFolder := oldFolderPath
	if normalizeFolderKey(newFolder) != normalizeFolderKey(payload.FolderName) {
		destFolder, err := modexport.ResolvedSubfolder(modsRoot, newFolder)
		if err != nil {
			return err
		}
		if _, err := os.Stat(destFolder); err == nil {
			return errors.New("目标文件夹已存在: " + newFolder)
		}
		if err := os.Rename(oldFolderPath, destFolder); err != nil {
			return err
		}
		workingFolder = destFolder
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
