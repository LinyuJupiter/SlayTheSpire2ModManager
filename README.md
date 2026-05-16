# 杀戮尖塔 2 Mod 管理器

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)

面向《杀戮尖塔 2》（Slay the Spire 2）的第三方桌面端 Mod 管理工具。基于 [Wails](https://wails.io/) 与 [Vue 3](https://vuejs.org/)，在本地管理游戏目录旁的 `mods` 文件夹：浏览、导入压缩包、编辑元数据、启用/禁用、导出与打开文件夹等。

> **说明**：本仓库为爱好者开发的辅助工具，与游戏开发商 MegaCrit 无关联。使用 Mod 可能影响存档或联机体验，请自行承担风险。

---

## 功能概览

- **游戏路径**：手动选择 `SlayTheSpire2.exe`，或在 Windows 上尝试从 Steam 库自动检测（读取 Steam 安装路径与库文件夹）。
- **Mod 列表**：扫描 `游戏目录/mods` 下符合约定的 manifest（`*.json` / 禁用时的 `*.json.bak`），展示 id、描述等；支持编辑并保存。
- **导入**：从 ZIP / RAR 导入到 `mods` 子目录（可指定文件夹名）。
- **启用 / 禁用**：通过 manifest 重命名策略切换启用状态。
- **导出**：将某一 mod 文件夹打包为 ZIP。
- **系统集成**：在文件管理器中打开 mods 根目录或单个 mod 目录。
- **小黑盒相关**：首次就绪时可解压内置的 Heybox 支持包（`sts2-heybox-support`），便于与社区分发流程配合。

---

## 环境要求

| 依赖 | 说明 |
|------|------|
| [Go](https://go.dev/dl/) | `go.mod` 要求 **1.25+** |
| [Node.js](https://nodejs.org/) | 用于前端构建（建议当前 LTS） |
| [Wails](https://wails.io/docs/gettingstarted/installation) | v2，用于开发与打包桌面应用 |

构建产物主要为 **Windows** 场景（Steam 检测、路径约定等）；其他操作系统上 Steam 自动检测不可用，仍可自行指定游戏路径使用（若游戏与目录布局一致）。

---

## 从源码构建

### 1. 克隆仓库

```bash
git clone https://github.com/LinyuJupiter/SlayTheSpire2ModManager.git
cd SlayTheSpire2ModManager
```

### 2. 安装依赖

**前端（首次或 `package.json` 变更后）：**

```bash
cd frontend
npm install
cd ..
```

**Wails CLI**：请按[官方文档](https://wails.io/docs/gettingstarted/installation)安装，并确保 `wails doctor` 通过。

### 3. 开发调试（热重载）

在项目根目录执行：

```bash
wails dev
```

前端由 Vite 提供快速刷新；也可按 Wails 文档使用浏览器连接本地开发服务调试绑定方法。

### 4. 发布构建

```bash
wails build
```

可执行文件输出目录见 Wails 默认配置（一般为 `build/bin`）。NSI 安装包等可在 `build/windows/installer` 中按需配置。

---

## 使用说明（简要）

1. 启动应用后设置 **《杀戮尖塔 2》主程序** 路径（须为 `SlayTheSpire2.exe`）。
2. 应用会在游戏 exe **同级目录**下使用 `mods` 文件夹（若不存在会创建）。
3. 在界面中导入、开关 Mod，或通过「打开文件夹」手动调整文件。

---

## 配置文件位置

应用配置保存在用户配置目录下的 `ModManager/config.json`（字段如 `game_exe_path`），用于记住游戏路径。具体路径因操作系统而异（例如 Windows 下多为 `%AppData%` 下的相应子目录）。

---

## 技术栈

- **后端 / 桌面壳**：Go，[Wails v2](https://github.com/wailsapp/wails)
- **前端**：Vue 3、Vite
- **其他**：压缩与归档（如 [archiver](https://github.com/mholt/archiver)）等，详见 `go.mod` / `frontend/package.json`

---

## 参与贡献

欢迎通过 Issue 讨论缺陷与需求，通过 Pull Request 提交改进。提交前请尽量：

- 保持改动与议题相关，避免无关重排或大范围格式化。
- 本地验证 `wails build` 或你修改所涉及的路径能通过构建。

---

## 许可证

本项目采用 **MIT License**，详见仓库根目录 [`LICENSE`](LICENSE) 文件。

---

## 致谢

- [Wails](https://wails.io/) — Go + Web 技术构建桌面应用  
- [Vue](https://vuejs.org/) / [Vite](https://vitejs.dev/) — 前端工具链  
- 《杀戮尖塔 2》由 MegaCrit 开发；本项目名称仅用于识别游戏兼容性与玩家社区场景。
