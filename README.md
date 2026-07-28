# ZServerBackup

远程 Windows 服务器**海量小文件**备份桌面客户端：源机按固定大小分卷打 zip，本机通过 **SSH + SFTP** 执行流水线 **pack → 下载 → 删远程包 → ack**，依赖源机 **`state.db`（SQLite）** 保证续传不重不漏。

技术栈：**Wails v3**、Go、`backend/` + `frontend/`（Vue 3 + TypeScript + Ant Design Vue 中文 + Tailwind CSS v4）。

## 架构

| 组件 | 说明 |
|------|------|
| 本应用 | 图形化 **zipbak-cli**，配置 SSH、触发 init、跑完整流水线 |
| `cmd/zipbak-srv` | 部署在源 Windows 服务器，提供 `init` / `pack` / `pack-ahead` / `ack` / `status` |
| 源机 `data/state.db` | SQLite：文件清单与进度（位于远程应用目录下的 `data/`） |

## 环境

- Go 1.25+
- Node.js 18+
- Wails v3：`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- Windows 打包：NSIS（`winget install NSIS.NSIS`）

## 开发

```bat
scripts\run-dev.bat
```

`wails3 dev` 会监听 **Go** 与 **`frontend/src`**（`.vue` / `.ts` / `.css`）。保存后约 1～2 秒会自动 `build DEV` 并**重启窗口**（使用内置 `frontend/dist`，避免 WebView 代理 Vite 白屏）。

| 脚本 | 作用 |
|------|------|
| `scripts\run-dev.bat` | 开发：改代码后自动 rebuild + 重启 |
| `scripts\dev-run-app.bat` | 只运行当前 `dist\ZServerBackup.exe`，无监听 |

在浏览器里调试 UI（可选）：`cd frontend` → `set WAILS_VITE_PORT=10245` → `npm run dev` → http://localhost:10245/

Linux/macOS：

```bash
chmod +x scripts/run-dev.sh
./scripts/run-dev.sh
```

## 打包

产物目录 **`dist/`**，包含 **两个程序**：

| 文件 | 用途 |
|------|------|
| `ZServerBackup.exe` / `*-installer.exe` | 本机图形客户端 |
| `zipbak-srv.exe` | 复制到**远程 Windows 源机**的应用目录 |

```bat
scripts\build-release.bat
```

单独只编源机 CLI：

```bat
scripts\build-zipbak-srv.bat
```

```bash
./scripts/build-release.sh
```

## 版本

根目录 `version.go` 的 `AppName` / `AppVersion` 与 `build/config.yml` 中 `info.version` 保持一致；窗口标题为 **项目名称 v版本号**。

```bat
scripts\set-version.bat 1.0.1
```

## GitHub Actions 发版

1. 修改 `backend/update/constants.go` 中的 `GitHubRepository` 为你的仓库（默认 `psvmc/z-server-backup-tools`）。
2. 提交并推送版本 bump，打 tag 并推送。
3. `scripts\publish-release.bat 1.0.1` 触发 `.github/workflows/release-all.yml`。

Windows 发布资产名：`ZServerBackup-amd64-installer.exe`（与 `ProductAssetName` 一致）。

## 配置说明

配置保存在用户目录 `%APPDATA%\z-server-backup-tools\config.json`（各平台为 `UserConfigDir/z-server-backup-tools/config.json`）。

本机流水线在 **下载当前分卷** 时会并行触发远程 **`pack-ahead`** 预打下一卷；`ack` 后下一卷通常已在 staging 就绪。升级客户端后请同步替换源机上的 **`zipbak-srv.exe`**（需支持 `pack-ahead`）。

主要字段：`host`、`remote_app_dir`（其下使用 `zipbak-srv.exe`、`data/state.db`、`staging/`）、`remote_source`（业务只读源目录）、`local_dir`、`max_part_gb` 等。

## 安全提示

生产环境请为 SSH 配置 **known_hosts** 并使用密钥登录；当前默认跳过主机密钥校验，仅适合内网调试。
