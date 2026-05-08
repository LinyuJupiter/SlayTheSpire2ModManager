// Package app 实现 Wails 绑定：游戏路径、mod 列表与导入导出等。
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ModManager/internal/config"
	"ModManager/internal/importer"
	"ModManager/internal/modexport"
	"ModManager/internal/mods"
	"ModManager/internal/shell"
	"ModManager/internal/steam"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App Wails 绑定入口。
type App struct {
	ctx       context.Context
	cfg       config.Config
	heyboxZip []byte
}

// New 创建 App；heyboxZip 为嵌入的 Heybox 支持包字节（可为空则跳过解压）。
func New(heyboxZip []byte) *App {
	return &App{heyboxZip: heyboxZip}
}

// Startup Wails 生命周期：加载配置并初始化 mods。
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if c, err := config.Load(); err == nil {
		a.cfg = c
	}
	if strings.TrimSpace(a.cfg.GameExePath) != "" && validateExe(a.cfg.GameExePath) == nil {
		modsRoot, err := mods.EnsureDirectory(a.cfg.GameExePath)
		if err == nil {
			_ = mods.EnsureHeyboxSupport(modsRoot, a.heyboxZip)
		}
	}
}

// UIState 界面所需路径状态。
type UIState struct {
	GameExePath string `json:"gameExePath"`
	ModsRoot    string `json:"modsRoot"`
	ConfigOK    bool   `json:"configOK"`
}

// GetUIState 返回当前保存的游戏路径与 mods 根目录。
func (a *App) GetUIState() UIState {
	st := UIState{
		GameExePath: a.cfg.GameExePath,
		ConfigOK:    strings.TrimSpace(a.cfg.GameExePath) != "",
	}
	if st.ConfigOK {
		st.ModsRoot = filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	}
	return st
}

// DetectSteamGameExe 在 Steam 库中查找 SlayTheSpire2.exe（仅 Windows 有效）。
func (a *App) DetectSteamGameExe() string {
	return steam.FindSlayTheSpire2Exe()
}

func validateExe(p string) error {
	st, err := os.Stat(p)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return fmt.Errorf("路径指向目录而非可执行文件")
	}
	base := strings.ToLower(filepath.Base(p))
	if base != "slaythespire2.exe" {
		return fmt.Errorf("请选择 SlayThe Spire 2 主程序 SlayTheSpire2.exe")
	}
	return nil
}

// SetGameExe 保存游戏 exe 路径并初始化 mods 目录与 Heybox 支持包。
func (a *App) SetGameExe(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("路径为空")
	}
	if err := validateExe(path); err != nil {
		return err
	}
	a.cfg.GameExePath = path
	if err := config.Save(a.cfg); err != nil {
		return err
	}
	modsRoot, err := mods.EnsureDirectory(path)
	if err != nil {
		return err
	}
	return mods.EnsureHeyboxSupport(modsRoot, a.heyboxZip)
}

// PickGameExe 打开文件对话框选择游戏主程序。
func (a *App) PickGameExe() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 SlayTheSpire2.exe",
		Filters: []runtime.FileFilter{
			{DisplayName: "可执行文件 (*.exe)", Pattern: "*.exe"},
		},
	})
}

// PickImportArchive 打开文件对话框选择 mod 压缩包。
func (a *App) PickImportArchive() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 Mod 压缩包",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP", Pattern: "*.zip"},
			{DisplayName: "RAR", Pattern: "*.rar"},
		},
	})
}

// ListMods 列出 mods 目录下符合格式的 mod。
func (a *App) ListMods() (*mods.ModsOverview, error) {
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return nil, fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	return mods.ListInstalled(modsRoot)
}

// SaveModEdit 保存对 mod 元数据的修改。
func (a *App) SaveModEdit(payload mods.ModEditPayload) error {
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	return mods.SaveEdits(modsRoot, payload)
}

// ImportModArchive 从 zip/rar 导入 mod 到 mods 目录。
func (a *App) ImportModArchive(archivePath string, folderName string) error {
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		return err
	}
	return importer.ImportArchive(archivePath, modsRoot, strings.TrimSpace(folderName))
}

// DeleteModEntry 删除当前 manifest 对应的 mod 文件（不含同目录下其它 mod）。
func (a *App) DeleteModEntry(folderName string, manifestFile string) error {
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	return mods.DeleteEntry(modsRoot, strings.TrimSpace(folderName), strings.TrimSpace(manifestFile))
}

// OpenModsFolder 在资源管理器中打开游戏 mods 目录（若不存在会先创建）。
func (a *App) OpenModsFolder() error {
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		return err
	}
	return shell.OpenDirectory(modsRoot)
}

// OpenModFolder 在资源管理器中打开 mods 下指定名称的 mod 文件夹。
func (a *App) OpenModFolder(folderName string) error {
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	folderPath, err := modexport.ResolvedSubfolder(modsRoot, folderName)
	if err != nil {
		return err
	}
	st, err := os.Stat(folderPath)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("不是目录")
	}
	return shell.OpenDirectory(folderPath)
}

// ExportModFolderZip 将 mods 下某一文件夹打包为 zip；弹出对话框由用户选择保存路径。
func (a *App) ExportModFolderZip(folderName string) error {
	if a.ctx == nil {
		return fmt.Errorf("应用未就绪")
	}
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	folderPath, err := modexport.ResolvedSubfolder(modsRoot, folderName)
	if err != nil {
		return err
	}
	st, err := os.Stat(folderPath)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("不是目录")
	}
	safeName := filepath.Base(folderPath)
	if safeName == "." || safeName == "" {
		safeName = "mod"
	}
	dest, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出 Mod 文件夹为 ZIP",
		DefaultFilename: safeName + ".zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 压缩包 (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return err
	}
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return nil
	}
	if strings.ToLower(filepath.Ext(dest)) != ".zip" {
		dest += ".zip"
	}
	return modexport.ZipDirectory(folderPath, dest)
}

// DisableMod 关闭 mod。
func (a *App) DisableMod(folderName string, manifestFile string) error {
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	return mods.Disable(modsRoot, strings.TrimSpace(folderName), strings.TrimSpace(manifestFile))
}

// EnableMod 根据 *.json.bak 重新启用 mod。
func (a *App) EnableMod(folderName string, manifestBakFile string) error {
	if strings.TrimSpace(a.cfg.GameExePath) == "" {
		return fmt.Errorf("请先设置游戏路径")
	}
	modsRoot := filepath.Join(filepath.Dir(a.cfg.GameExePath), "mods")
	return mods.Enable(modsRoot, strings.TrimSpace(folderName), strings.TrimSpace(manifestBakFile))
}
