package mods

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

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

// DeleteEntry 删除一条 mod：manifest 及对应 {id}.pck / .dll（含 .bak）；不删整个文件夹。
func DeleteEntry(modsRoot, folderName, manifestFile string) error {
	folder := filepath.Join(modsRoot, folderName)
	folder = filepath.Clean(folder)
	root := filepath.Clean(modsRoot)
	if folder == root || !strings.HasPrefix(folder+string(os.PathSeparator), root+string(os.PathSeparator)) {
		return errors.New("非法路径")
	}
	mp := filepath.Join(folder, manifestFile)
	b, err := os.ReadFile(mp)
	if err != nil {
		return err
	}
	man, ok := tryParseModManifest(b)
	if !ok {
		return errors.New("manifest 无效")
	}
	id := man.ID
	_ = os.Remove(filepath.Join(folder, id+".pck"))
	_ = os.Remove(filepath.Join(folder, id+".dll"))
	_ = os.Remove(filepath.Join(folder, id+".pck.bak"))
	_ = os.Remove(filepath.Join(folder, id+".dll.bak"))
	if err := os.Remove(mp); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Disable 关闭 mod：将 manifest 与对应 pck/dll 加 .bak。
func Disable(modsRoot, folderName, manifestFile string) error {
	folder := filepath.Join(modsRoot, folderName)
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
	folder := filepath.Join(modsRoot, folderName)
	oldPath := filepath.Join(folder, manifestBakFile)
	b, err := os.ReadFile(oldPath)
	if err != nil {
		return err
	}
	man, ok := tryParseModManifest(b)
	if !ok {
		return errors.New("无法解析 manifest")
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
	id := man.ID
	renameIfExists(filepath.Join(folder, id+".pck.bak"), filepath.Join(folder, id+".pck"))
	renameIfExists(filepath.Join(folder, id+".dll.bak"), filepath.Join(folder, id+".dll"))
	return nil
}
