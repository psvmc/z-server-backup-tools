# 单文件备份 Tab 设计

日期：2026-07-31  
状态：已确认

## 背景

主页拆成两个 Tab：保留现有「多文件备份」（依赖远程 `zipbak-srv` 分卷流水线），新增「单文件备份」——不部署远程程序、不压缩，直接 SFTP 下载单个文件并显示进度。

## 目标

- 主页：`多文件备份` | `单文件备份` 两个 Tab
- 单文件：远程选/填单个文件 → 本机目录直接下载 → 字节级进度
- 设置：SSH、邮件与多文件共享；远程源文件路径、本机保存目录独立
- 不上传任何程序到服务器；远程不压缩

## 非目标

- 单文件模式不分卷、不 init/ack、不依赖 `zipbak-srv`
- 不支持选目录（仅单个文件）
- 本阶段不做多任务列表（单文件仅一组路径配置）

## 配置模型

共享（沿用现有 `backup` 中的连接与邮件字段）：

- `host` / `port` / `user` / `password`
- `notify_email` / `smtp_*`

单文件独立（新增 `singleFileBackup` 持久化块）：

```json
{
  "singleFileBackup": {
    "remote_file": "",
    "local_dir": ""
  }
}
```

- `remote_file`：远程单个文件完整路径（如 `D:\data\app.bak`）
- `local_dir`：本机保存目录；本地文件名 = 远程 basename（如 `app.bak`）

多文件的 `backup` / `backupTasks` / `remote_app_dir` / `max_part_gb` 等不变。

设置 UI：

- SSH、邮件：仍在统一「设置」弹窗（两 Tab 共用）
- 单文件路径：放在「单文件备份」Tab 内或该 Tab 的设置区（与多文件任务路径分离）

## 后端

### API（可挂在现有 `BackupService` 或新建 `SingleFileBackupService`）

建议新建服务以免与多文件 `status`/`cancel`/`logs` 互相覆盖：

| 方法 | 说明 |
|------|------|
| `GetSingleFileConfig` / `SaveSingleFileConfig` | 读写 `remote_file`、`local_dir` |
| `StartSingleFileDownload` | 启动下载 |
| `StopSingleFileDownload` | 取消 |
| `GetSingleFileStatus` / `GetSingleFileLogs` | 状态与日志 |
| `ListRemoteFiles`（可选） | 扩展现有目录浏览，支持选文件；或复用路径输入 + 远程目录选择后手输文件名 |

连接测试、打开本机文件夹复用现有能力。

### 下载流程

1. `prepareSSH` + 校验 `remote_file`、`local_dir` 非空
2. 确保本机目录存在
3. `localPath = filepath.Join(local_dir, filepath.Base(remote_file))`
4. `sftpclient.DownloadWithProgress`（已有 `.downloading` 临时后缀与进度回调）
5. 更新状态：`phase=download`，`downloadBytesDone/Total/SpeedBps`，`localFile`
6. 成功 / 异常（非取消）：按共享邮件配置发通知（可复用 `notify.SendBackupNotification`，文案标明单文件）
7. 用户取消：不发邮件（与多文件一致）

并发：多文件流水线与单文件下载互斥（任一运行时拒绝另一个启动，或 Tab 切换时禁用对方操作）。

### 远程文件选择

- 最小可用：路径可手动输入（`editable`）
- 增强：远程浏览弹窗支持「选文件」（扩展 SFTP `ReadDir` 列出非目录项）；双击/选中确认

## 前端

### App.vue

```
a-tabs
  ├─ 多文件备份 → 现有 BackupRunPanel + 任务/设置弹窗（现状）
  └─ 单文件备份 → SingleFileBackupPanel（新）
```

顶部「设置」按钮对两 Tab 共用（SSH + 邮件 + 多文件的远程应用目录等可仍放在同一弹窗；单文件路径在单文件面板内编辑保存）。

### SingleFileBackupPanel

- 远程源文件 + 本机保存目录
- 开始下载 / 停止
- 进度条：按字节（`done/total`），文案含速度
- 日志区
- 可选：打开本机目录、「任务查看」式列表本机该目录文件（非必须，首版可省略）

### 状态与轮询

- `useSingleFileBackup` composable，独立于 `useBackupJob`
- 事件：`singlefile-log` / `singlefile-done` / `singlefile-error`（命名避免与多文件冲突）

## 邮件

- 使用共享 SMTP / 通知邮箱
- 成功主题示例：`[单文件下载完成] {host}`
- 失败主题示例：`[单文件下载异常] {host}`
- 正文含远程路径、本机路径、错误信息（如有）

## 验收标准

1. Tab 切换正常；多文件功能回归无破坏
2. 仅配置 SSH + 单文件路径即可下载，无需 `zipbak-srv`、无需远程 init
3. 下载过程显示百分比与速度；停止可中断
4. 成功后本机文件名为远程 basename；无远程临时压缩产物
5. 邮件配置存在时，完成/异常会发送；取消不发送
6. 与多文件任务不能同时运行

## 实现顺序建议

1. 配置存储 + 后端下载 API + 互斥
2. 前端 Tab + 单文件面板 + 进度
3. 远程选文件（若时间紧可第二迭代）
4. 邮件与联调
