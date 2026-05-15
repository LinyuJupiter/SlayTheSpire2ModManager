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

// ModVersionRef 同一 manifest id 的另一安装位置（用于多版本切换）。
type ModVersionRef struct {
	FolderName   string `json:"folderName"`
	ManifestFile string `json:"manifestFile"`
	Disabled     bool   `json:"disabled"`
}

// InstalledMod 一个已识别的 mod 条目。
type InstalledMod struct {
	FolderName          string          `json:"folderName"`
	ManifestFile        string          `json:"manifestFile"`
	Disabled            bool            `json:"disabled"`
	Manifest            ModManifest     `json:"manifest"`
	IDUnique            bool            `json:"idUnique"`
	ConflictWith        []string        `json:"conflictWith"`
	MissingDependencies []string        `json:"missingDependencies"`
	Available           bool            `json:"available"`
	LayoutNormalized    bool            `json:"layoutNormalized"`
	AlternateVersions   []ModVersionRef `json:"alternateVersions"`
}

// ModsOverview 列表与全局校验结果。
type ModsOverview struct {
	ModsDir      string         `json:"modsDir"`
	Mods         []InstalledMod `json:"mods"`
	DuplicateIDs []string       `json:"duplicateIDs"`
}

// ModEditPayload 仅允许修改的字段；NewFolderName 为空则不改文件夹名。
// LayoutNormalized 为 true 且 FolderName 为两段式时，NewFolderName 表示最外层 slug（保存时会拼回版本子路径）。
type ModEditPayload struct {
	FolderName       string `json:"folderName"`
	NewFolderName    string `json:"newFolderName"`
	LayoutNormalized bool   `json:"layoutNormalized"`
	ManifestFile     string `json:"manifestFile"`
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	AffectsGameplay  bool   `json:"affects_gameplay"`
}

type modInstanceKey struct {
	Folder   string
	Manifest string
}

// NormalizeReport 规范化 mods 目录后的摘要。
type NormalizeReport struct {
	Migrated []string `json:"migrated"`
	Skipped  []string `json:"skipped"`
	Errors   []string `json:"errors"`
}
