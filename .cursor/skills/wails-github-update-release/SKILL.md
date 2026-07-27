---
name: wails-github-update-release
description: >-
  Adds Wails v3 in-app GitHub Releases update checks and multi-platform
  GitHub Actions publishing (Windows NSIS / Linux deb-rpm / macOS zip).
  Use when the user asks to add auto-update, version bump, GitHub Action
  release, publish-release, set-version, NSIS installer packaging, or
  replicate z-disk-tools / z-git-tools-wails release workflow. For brand-new
  apps, use wails-create-project first, then this skill.
---

# Wails GitHub 更新与发版

**存放位置：** 本仓库 `.cursor/skills/`（项目技能，勿写入 `~/.cursor/skills/`）。

为 **已有** Wails v3 + Go + Vue（Ant Design Vue）桌面应用接入：

1. 应用内检查/下载/安装更新（GitHub Releases Provider）
2. GitHub Actions 三平台发版
3. 版本脚本与发布注意项

参考实现：本仓库 `z-disk-tools`，以及 `z-git-tools-wails`。

## 与创建技能的关系（必读）

**新建 Wails 项目时固定顺序：**

1. **先**使用 `.cursor/skills/wails-create-project/SKILL.md`（Vue3 + Ant Design Vue 中文 + Tailwind）
2. **再**使用本技能接入更新与 GitHub Actions 发版

若仓库里还没有可 `wails3 task build/package` 的工程，先停下来执行创建技能，不要直接铺更新代码。

已有可运行工程 → 可直接按下方清单接入。

## 开始前先收集

向用户确认（缺一不可）：

| 占位符 | 含义 | 示例 |
|--------|------|------|
| `PRODUCT` | 显示名 / 二进制前缀 | `ZDisk` |
| `GITHUB_REPO` | 发布仓库 `owner/repo` | `psvmc/z-disk-tools-releases` |
| `VERSION_CONST` | version.go 常量名 | `AppVersion` 或 `Version` |
| `BIN_DIR` | 构建输出目录 | `dist` 或 `bin` |
| `INSTALL_DIR` | Windows 默认安装目录名 | 可与 PRODUCT 不同，如 `z-disk-tools` |
| `DEFAULT_BRANCH` | 默认分支 | `master` / `main` |

前端默认 **Ant Design Vue + 中文**（与创建技能一致）。仅当用户明确要求时才用 Naive UI。

**仓库模式（优先问清）：**

- **同源同仓**：源码与 Releases 同一 GitHub 仓库 → checkout tag 即可（本仓库）
- **源码另仓**：源码在 Gitee、发版到 GitHub Releases 仓 → 需 `GITEE_TOKEN` 克隆（`z-git-tools-wails`）

## 落地检查清单

复制并勾选：

```
- [ ] version.go + build/config.yml 版本同源
- [ ] backend/update（GitHub provider + asset matcher + enabled_dev/prod）
- [ ] UpdateService + 平台 apply（windows installer / linux pkg / darwin swap）
- [ ] main.go：InitUpdater + RegisterService
- [ ] 前端：启动检查 + 手动检查 + 确认/进度弹窗（Ant Design Vue）
- [ ] scripts/set-version.* + publish-release.bat
- [ ] .github/workflows/release-all.yml
- [ ] NSIS：中文、user 域、InstallLocation、UTF-8 BOM
- [ ] 资产名与 ProductAssetName 一致
- [ ] Linux/Darwin 系统 API 按平台拆分（勿共用 !windows）
```

## 1. 版本定义

`version.go` 与 `build/config.yml` 的 `info.version` **必须同步**。

```go
const (
	AppName    = "PRODUCT"
	AppVersion = "1.0.0"
)
```

`Taskfile.yml`：

```yaml
APP_NAME: "PRODUCT"
BIN_DIR: "dist"   # 或 bin
```

## 2. 应用内更新（后端）

### 包结构（按项目调整 import path）

```
backend/update/          # 或 internal/update/
  constants.go           # GitHubRepository, ProductAssetName
  enabled_dev.go         # //go:build !production → Enabled=false
  enabled_prod.go        # //go:build production → Enabled=true
  setup.go               # InitUpdater
  asset.go               # PreferredAssetName + AssetMatcher
  httpclient.go          # Timeout=0，避免大安装包下载被掐断
  skip.go                # 持久化跳过版本
  staging.go / helper_* / swap_target_*   # Darwin 原地替换用
backend/config/store.go  # SkippedUpdateVersion
backend/service/
  update_service.go
  update_apply_windows.go  # Download → 启动 installer → Quit
  update_apply_linux.go    # deb/rpm + pkexec/sudo
  update_apply_other.go    # darwin：ApplyStagedUpdate
```

### 关键约定

```go
const (
	GitHubRepository = "owner/repo"
	ProductAssetName = "PRODUCT" // 必须与 Release 资产文件名一致
)
```

资产命名：

| 平台 | 文件名 |
|------|--------|
| Windows | `PRODUCT-amd64-installer.exe` |
| macOS | `PRODUCT-macos-universal.zip` |
| Linux | `PRODUCT.deb` / `PRODUCT.rpm` |

`main.go`（production 才真正启用）：

```go
if err := update.InitUpdater(app, AppVersion); err != nil {
	log.Fatal(err)
}
app.RegisterService(application.NewService(service.NewUpdateService(app, AppVersion)))
```

### 前端（Ant Design Vue）

- 启动：`checkOnStartup()`（失败只打日志）
- 手动：标题栏版本号点击 / 设置页「检查更新」
- 弹窗：`a-modal` 确认（立即/稍后/跳过）+ `a-progress` 下载进度
- 提示：`message.loading` / `message.success` / `message.error`
- 事件：`wails:updater:download-started|download-progress|verifying|installing`
- 开发构建：`enabled=false`，提示「开发版本不支持自动更新」
- 生成绑定：`wails3 generate bindings -ts`（`frontend/bindings/` 通常 gitignore）

## 3. GitHub Actions 发版

### 同源同仓（推荐默认）

`.github/workflows/release-all.yml`：

- `workflow_dispatch` 输入 `tag`
- matrix：`windows-latest` / `ubuntu-24.04` / `macos-latest`
- `actions/checkout@v4` + `ref: ${{ inputs.tag }}`
- Windows：装 NSIS → `wails3 task windows:package` → 收集 `*-installer.exe`
- Linux：GTK4/WebKit 依赖 → deb/rpm（失败则裸二进制）
- macOS：`darwin:package:universal` → zip
- publish：`softprops/action-gh-release@v2`，`target_commitish` = 默认分支

详细模板与 Gitee 源码模式见 [reference.md](reference.md)。

### 本地脚本

- `scripts/set-version.bat|ps1`：改 version.go + config.yml → `wails3 task common:update:build-assets` → 修正 nfpm `./bin/`→`./BIN_DIR/`
- `scripts/publish-release.bat`：`gh workflow run release-all.yml -R REPO -f tag=...`
- `scripts/build-release.bat`：自动把 NSIS 目录加入 PATH

### 发版流程

```bat
scripts\set-version.bat 1.0.1
git add -A && git commit -m "chore: bump version to 1.0.1"
git push origin master
git tag 1.0.1 && git push origin 1.0.1
scripts\publish-release.bat 1.0.1
```

## 4. NSIS 必做项（Windows）

1. **默认 user 域**（无 UAC）：`WAILS_INSTALL_SCOPE=user`，Taskfile `INSTALL_SCOPE` 默认 `user`
2. **安装目录可与产品名分离**：硬编码  
   `InstallDir "$LOCALAPPDATA\Programs\INSTALL_DIR"`
3. **记住安装位置**：`InstallDirRegKey` + `RestorePreviousInstallDir` + 写入 `InstallLocation`；若改名，固定旧 `UNINST_KEY_NAME`
4. **简体中文**：`MUI_LANGUAGE "SimpChinese"`，`.onInit` 里 `StrCpy $LANGUAGE ${LANG_SIMPCHINESE}`
5. **UTF-8 BOM**：含中文的 `project.nsi` 必须以 UTF-8 BOM 保存，否则 makensis ACP 解析失败
6. **WebView2**：`wails.webview2runtime` 会嵌入 Online Bootstrapper
7. **本地 PATH**：`makensis` 常在 `C:\Program Files (x86)\NSIS`，脚本需自动探测

## 5. 致命注意项

| 问题 | 处理 |
|------|------|
| `unix.O_DIRECT` / `F_NOCACHE` 混用 | **按平台拆分** `*_linux.go` / `*_darwin.go`，禁止笼统 `!windows` |
| `update:build-assets` 把 nfpm 写成 `./bin/` | set-version 后强制改成 `./BIN_DIR/` |
| 资产名 ≠ ProductAssetName | 更新匹配失败，装完也无法提示更新 |
| tag 未推送就跑 workflow | checkout / changelog 失败 |
| 已存在 tag 再改代码 | **升版新 tag**，勿强推覆盖（除非用户明确要求） |
| 开发模式测更新 | `Enabled=false`，必须打 production 包 |
| bindings 未进 git | CI 里靠 `wails3 task` 重新生成即可 |
| 同源仓 vs Gitee | 勿混用两套 workflow |

## 6. 实施顺序建议

**新项目：** 先完成 `wails-create-project` → 再继续本清单。

1. 确认占位符与仓库模式  
2. 后端 update + UpdateService + main 接入  
3. 前端检查 UI（Ant Design Vue）  
4. set-version / publish / build-release 脚本  
5. NSIS（中文 / user / InstallLocation / BOM）  
6. release-all.yml  
7. 本地 `scripts\build-release.bat` 验证  
8. 打 tag → publish → `gh run watch`

## 附加资源

- 细节参考：[reference.md](reference.md)
- 新项目脚手架：[../wails-create-project/SKILL.md](../wails-create-project/SKILL.md)
