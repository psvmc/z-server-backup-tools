# Wails GitHub 更新与发版 — 参考细节

配合 [SKILL.md](SKILL.md) 使用。按需阅读，勿整份塞进上下文。

## 参考仓库

| 项目 | 模式 | 要点 |
|------|------|------|
| `z-disk-tools`（本仓库） | 同源同仓 GitHub | `BIN_DIR=dist`，产品名 `ZDisk`，安装目录可仍为 `z-disk-tools` |
| `z-git-tools-wails` | Gitee 源码 → GitHub Releases 仓 | 需 `GITEE_TOKEN`；`BIN_DIR=bin`；产品名 `ZGit` |

## 后端文件职责

### UpdateService API

- `GetCurrentVersion() string`
- `CheckForUpdate() (UpdateCheckResult, error)` — 调 `app.Updater.Check`
- `ApplyUpdate() error` — 平台相关
- `SkipUpdateVersion(version string) error` — `Updater.SkipVersion` + 本地 store

### UpdateCheckResult JSON

```go
type UpdateCheckResult struct {
	Available      bool   `json:"available"`
	Enabled        bool   `json:"enabled"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseName    string `json:"releaseName"`
	Notes          string `json:"notes"`
	ReleaseURL     string `json:"releaseURL"`
}
```

### 平台 Apply

**Windows**

1. `DownloadAndInstall(ctx)`
2. `DownloadedPath()` → `.exe` installer
3. `exec.Command(installer)` + `ApplyVisibleDetachAttrs`（勿用 CommandContext，避免 cancel 杀掉安装进程）
4. `app.Quit()`

**Linux**

1. 下载 `.deb` / `.rpm`
2. 优先 `pkexec` / `sudo` 执行 `dpkg -i` / `rpm -U`
3. 失败则 `xdg-open` 打开包

**Darwin**

1. 下载并解压到 staged path
2. `ApplyStagedUpdate`：复制到可执行文件旁 → spawn Wails helper 换包 → Quit

### HTTP Client

Wails 默认 GitHub provider 客户端超时偏短。自定义 `Timeout: 0`，由调用方 context（如 10 分钟）控制下载。

### enabled 构建标签

```go
//go:build !production
const Enabled = false

//go:build production
const Enabled = true
```

正式包必须带 `-tags production`（Wails package/build 默认会加）。

## 前端事件

进度监听（`@wailsio/runtime` Events）：

- `wails:updater:download-started`
- `wails:updater:download-progress` — `{ written, total, rate }`
- `wails:updater:verifying`
- `wails:updater:installing`

默认 Ant Design Vue：`a-modal` / `a-progress` / `message`。仅用户明确要求时再用 Naive UI。

## 同源同仓 workflow 骨架

```yaml
name: Release All Platforms
on:
  workflow_dispatch:
    inputs:
      tag:
        description: Release tag (e.g. 1.0.1)
        required: true
permissions:
  contents: write
jobs:
  build:
    strategy:
      fail-fast: false
      matrix:
        include:
          - os: windows-latest
            platform: windows
          - os: ubuntu-24.04
            platform: linux
          - os: macos-latest
            platform: macos
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ inputs.tag }}
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.x"
          cache-dependency-path: go.sum
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      # Windows: choco install nsis + PATH
      # Linux: libgtk-4-dev libwebkitgtk-6.0-dev
      # Build: wails3 + npm ci + platform package → release/
      - uses: actions/upload-artifact@v4
        with:
          name: ${{ matrix.platform }}
          path: release/*
  publish:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      # 由 tag 生成 changelog → release-notes.md
      - uses: actions/download-artifact@v4
        with:
          path: dist
          merge-multiple: true
      - uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ inputs.tag }}
          target_commitish: master   # 或 main
          name: PRODUCT ${{ inputs.tag }}
          body_path: release-notes.md
          files: dist/**
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

构建段关键命令（替换 `PRODUCT` / `BIN_DIR`）：

```bash
wails3 task windows:package
cp -f BIN_DIR/*-installer.exe release/

wails3 task linux:create:deb && wails3 task linux:create:rpm
cp -f BIN_DIR/*.deb BIN_DIR/*.rpm release/

wails3 task darwin:package:universal
ditto -c -k --keepParent BIN_DIR/PRODUCT.app release/PRODUCT-macos-universal.zip
```

## Gitee → GitHub 模式差异

1. build job **不**用 `actions/checkout`，而用 `GITEE_TOKEN` 浅克隆指定 tag
2. publish changelog 同样从 Gitee 拉 tags
3. Releases 仓可能与源码仓分离；可用 `scripts/release/ensure-release-gha.*` 把 workflow 同步到 Releases 仓
4. Secret：`gh secret set GITEE_TOKEN -R owner/releases-repo`

## NSIS project.nsi 片段

```nsis
!ifndef WAILS_INSTALL_SCOPE
!define WAILS_INSTALL_SCOPE "user"
!endif
!ifndef UNINST_KEY_NAME
!define UNINST_KEY_NAME "oldCompanyoldProduct"  ; 改名时保持稳定
!endif

!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\INSTALL_DIR"
    InstallDirRegKey HKCU "${UNINST_KEY}" "InstallLocation"
!else
    InstallDir "$PROGRAMFILES64\INSTALL_DIR\INSTALL_DIR"
    InstallDirRegKey HKLM "${UNINST_KEY}" "InstallLocation"
!endif

Function .onInit
   !insertmacro wails.checkArchitecture
   StrCpy $LANGUAGE ${LANG_SIMPCHINESE}
   Call RestorePreviousInstallDir
FunctionEnd

; Section 末尾：
SetRegView 64
WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
```

`RestorePreviousInstallDir`：先读 `InstallLocation`，否则从 `UninstallString` 取父目录（见本仓库 / z-git 的 `project.nsi`）。

**完成页中文（易踩坑）：**

```nsis
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "$(ZSB_FINISH_RUN)"
!insertmacro MUI_PAGE_FINISH
; … MUI_LANGUAGE 之后：
LangString ZSB_FINISH_RUN ${LANG_SIMPCHINESE} "立即运行 ${INFO_PRODUCTNAME}"
LangString ZSB_FINISH_RUN ${LANG_ENGLISH} "Launch ${INFO_PRODUCTNAME}"
```

若仅**最后一步** checkbox 中文乱码、前面向导页正常 → 几乎都是 `project.nsi` 里上述 `LangString` 被按 ANSI 误解析，不是 MUI 语言包问题。

**编码（必做两项）：**

1. `project.nsi` 保存为 UTF-8 **带 BOM**（文件头应为 `EF BB BF`）。PowerShell：

```powershell
$path = "build/windows/nsis/project.nsi"
$text = [System.IO.File]::ReadAllText($path, [System.Text.UTF8Encoding]::new($false))
[System.IO.File]::WriteAllText($path, $text, (New-Object System.Text.UTF8Encoding $true))
```

2. `build/windows/Taskfile.yml` → `create:nsis:installer`：

```yaml
makensis /INPUTCHARSET UTF8 ... project.nsi
```

与 BOM 同时使用：BOM 保证脚本字面量正确；`/INPUTCHARSET UTF8` 在 BOM 被 IDE 去掉时仍能编译出正确 Unicode 安装包。

## set-version 后处理

`wails3 task common:update:build-assets` 可能：

- 重写 `info.json` / `nfpm.yaml` / plist / `wails_tools.nsh`
- **不会**覆盖已定制的 `project.nsi`（仍建议每次确认）
- nfpm `contents.src` 常被写成 `./bin/PRODUCT` → 若 `BIN_DIR=dist` 必须改回 `./dist/PRODUCT`

## 跨平台编译坑

磁盘直读相关：

| API | Linux | Darwin |
|-----|-------|--------|
| `unix.O_DIRECT` | 有 | **无** |
| `unix.F_NOCACHE` | **无** | 有 |
| `unix.FcntlInt` | 返回 `(int, error)` 两值 | 同左 |

错误示例：`assignment mismatch: 1 variable but unix.FcntlInt returns 2 values`、`undefined: unix.O_DIRECT`。

## 验收

本地：

```bat
scripts\build-release.bat
dir dist\PRODUCT*
```

CI：

```bat
gh run watch -R owner/repo
gh release view TAG -R owner/repo
```

生产包：安装后点版本号 → 应能检查更新（需已有更高版本 Release，或临时降本地版本验证）。
