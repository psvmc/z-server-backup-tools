# Linux 多文件备份（zipbak-srv）设计

日期：2026-07-31  
状态：已确认

## 背景

多文件备份依赖远程 `zipbak-srv`。当前构建固定 `GOOS=windows` 产出 `zipbak-srv.exe`；客户端路径与 SSH 调用亦按 Windows 假设。服务器已有 `os_type`（Windows/Linux），但 Linux 被禁止勾选多文件。`zipbak` 核心逻辑可跨平台，需补齐 Linux amd64 构建与客户端按 OS 适配。

## 目标

- 同时构建 Windows / Linux（amd64）两套 `zipbak-srv`
- Linux 服务器可启用多文件备份（远程应用目录 + 分卷）
- 按 `os_type` 选择程序名、路径分隔符、SSH 命令拼法
- 用户仍手动将对应二进制放到远程应用目录（首版不自动上传）

## 非目标

- 自动上传二进制或自动 `chmod +x`
- Linux arm64
- 另起一套与 zipbak 无关的远程协议 / 容器方案
- 改变单文件备份行为（已按 `os_type` 浏览路径）

## 构建与产物

| 目标 | 命令要点 | 产出 |
|------|----------|------|
| Windows（现状） | `GOOS=windows GOARCH=amd64` | `dist/zipbak-srv.exe` |
| Linux（新增） | `GOOS=linux GOARCH=amd64` | `dist/zipbak-srv` |

- Taskfile：新增 Linux 构建任务，或扩展现有任务一次产出两套
- `build-release` / CI：release 目录同时包含两套文件
- 发布说明标明源机 OS 与文件对应关系

## 运行时约定

`BackupConfig` 已有 `os_type`（由所选 `Server` 合并而来）。

### Resolved 路径

| OS | RemoteSrv | State / Staging |
|----|-----------|-----------------|
| Windows | `{app}\zipbak-srv.exe` | `{app}\data\state-{id}.db`、`{app}\staging-{id}`（`\`） |
| Linux | `{app}/zipbak-srv` | `{app}/data/state-{id}.db`、`{app}/staging-{id}`（`/`） |

### 路径规范化

多文件流水线中所有远程路径规范化改为 `NormalizeRemotePathForOS(path, cfg.OSType)`，覆盖：

- init / pack / pack-ahead / ack / status / reset / oversized
- SFTP 下载后删除远程分卷等

### SSH `RunRemote`

| OS | 行为 |
|----|------|
| Windows | 保持 `QuoteWindowsArg` 拼整行 |
| Linux | 对程序路径与参数做 shell 安全引号后拼接执行；可执行权限由用户保证 |

找不到程序或无执行权限时：保留远程错误原文；日志可提示检查是否已放置匹配 OS 的二进制（Linux 注意 `chmod +x`）。

## 配置与 UI

- 取消「Linux 禁止多文件」：Linux 可勾选「支持多文件备份」并配置 `remote_app_dir` / `max_part_gb`
- 保存校验：多文件时仍要求远程应用目录与分卷上限；不再因 Linux 拒绝
- 服务器表单中推导路径文案按 `os_type` 显示 `zipbak-srv` 或 `zipbak-srv.exe` 及分隔符
- 远程目录浏览继续使用现有 `os_type` 逻辑

## 验收标准

1. 可构建出 `zipbak-srv.exe` 与 linux/amd64 的 `zipbak-srv`
2. Linux 服务器可勾选多文件并保存远程应用配置
3. 手动放置 Linux 二进制后，init / 备份 / 分卷下载可用（路径为 POSIX 风格）
4. Windows 多文件回归正常
5. 单文件备份行为不变

## 实现顺序建议

1. `Resolved` + 路径规范化 + `RunRemote` 按 OS 分支；放开 Linux 多文件校验/UI
2. Taskfile / release / CI 增加 Linux 构建产物
3. 前端推导路径与文案
4. 联调与 Windows 回归
