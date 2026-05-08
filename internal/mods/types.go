package mods

// ModManifest 与游戏 mod 描述 JSON 对应。
type ModManifest struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Author          string   `json:"author"`
	Description     string   `json:"description"`
	Version         string   `json:"version"`
	HasPck          bool     `json:"has_pck"`
	HasDll          bool     `json:"has_dll"`
	Dependencies    []string `json:"dependencies"`
	AffectsGameplay bool     `json:"affects_gameplay"`
}

// InstalledMod 一个已识别的 mod 条目。
type InstalledMod struct {
	FolderName          string      `json:"folderName"`
	ManifestFile        string      `json:"manifestFile"`
	Disabled            bool        `json:"disabled"`
	Manifest            ModManifest `json:"manifest"`
	IDUnique            bool        `json:"idUnique"`
	ConflictWith        []string    `json:"conflictWith"`
	MissingDependencies []string    `json:"missingDependencies"`
	Available           bool        `json:"available"`
}

// ModsOverview 列表与全局校验结果。
type ModsOverview struct {
	ModsDir      string         `json:"modsDir"`
	Mods         []InstalledMod `json:"mods"`
	DuplicateIDs []string       `json:"duplicateIDs"`
}

// ModEditPayload 仅允许修改的字段；NewFolderName 为空则不改文件夹名。
type ModEditPayload struct {
	FolderName      string `json:"folderName"`
	NewFolderName   string `json:"newFolderName"`
	ManifestFile    string `json:"manifestFile"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	AffectsGameplay bool   `json:"affects_gameplay"`
}

type modInstanceKey struct {
	Folder   string
	Manifest string
}
