# 服务器管理设计

日期：2026-07-31  
状态：已确认

## 背景

当前 SSH 连接与远程应用目录存在全局 `backup` 配置中，所有多文件任务与单文件备份共用一套连接；「设置」弹窗同时包含 SSH、远程应用、邮件。需要支持多台服务器独立管理，任务/单文件各自绑定服务器，设置收窄为仅通知（邮件）。

## 目标

- Tab 右上角新增「服务器管理」：列表已添加服务器，右上角可添加/编辑
- 添加服务器时可勾选「支持多文件备份」；勾选后配置远程应用目录与分卷上限
- 未勾选的服务器仅可用于单文件备份；多文件任务下拉只列出已勾选服务器
- 多文件任务、单文件配置各自选择 `server_id`
- 原「设置」改为「通知设置」，内容仅保留邮件 SMTP
- 删除服务器：若被任务或单文件配置引用则禁止删除

## 非目标

- 不把旧全局 SSH / 远程应用自动迁移为服务器列表（用户自行重新添加）
- 不支持密钥登录、服务器导入导出、批量操作
- 不改变多文件/单文件流水线本身（仍通过合并后的 `BackupConfig` 形状调用现有逻辑）
- 不改变 JobGate 互斥、取消不发邮件等既有行为

## 配置模型

### Server（新，`config.json` → `servers[]`）

```json
{
  "servers": [
    {
      "id": "srv_xxx",
      "name": "生产机",
      "host": "",
      "port": 22,
      "user": "",
      "password": "",
      "support_multi_file": true,
      "remote_app_dir": "D:\\zipbak",
      "max_part_gb": 2
    }
  ]
}
```

| 字段 | 说明 |
|------|------|
| `id` | 稳定 ID |
| `name` | 显示名（列表与下拉） |
| `host` / `port` / `user` / `password` | SSH |
| `support_multi_file` | 是否支持多文件备份 |
| `remote_app_dir` / `max_part_gb` | 仅当 `support_multi_file=true` 时必填并使用；否则可空 |

运行时仍可按任务 ID 推导 `remote_srv` / `remote_state` / `remote_staging`（逻辑保持现有 `Resolved()`，输入来自所选 Server + Task）。

### 引用

- `BackupTask` 增加 `server_id`
- `SingleFileConfig` 增加 `server_id`（保留 `remote_file` / `local_dir`）

### 全局 `backup`（收窄）

仅保留邮件相关字段：

- `notify_email`
- `smtp_host` / `smtp_port` / `smtp_user` / `smtp_password`

SSH、`remote_app_dir`、`max_part_gb` 不再作为全局「当前连接」使用；对外保存接口改为 `SaveNotifyConfig`（仅邮件字段）。

### 迁移

- **不**自动把旧全局 SSH 建成服务器
- 旧任务若缺少 `server_id`：可展示，但保存/启动前必须选择服务器
- 读配置时：邮件字段继续沿用；`backup` 中残留的 host/port/user/password/`remote_app_dir`/`max_part_gb` 不作为有效连接暴露给 UI，也不写入 `servers[]`

## UI

### Tab 右上角

- 「服务器管理」→ 服务器列表弹窗
- 「通知设置」→ 仅邮件表单（替换原「设置」文案与内容）
- 移除多文件面板内重复的「设置」入口；通知仅从 Tab 右上角「通知设置」进入

### 服务器管理弹窗

- 列表：名称、主机、是否支持多文件、编辑、删除
- 右上角「添加」→ 新增表单；行内编辑同表单
- 表单字段：名称、SSH（主机/端口/用户/密码）、「测试连接」、勾选「支持多文件备份」
  - 勾选后展开：远程应用目录（可浏览）、分卷上限 GB、推导路径只读展示
  - 未勾选：不展示远程应用配置
- 删除：存在 `server_id` 引用（任一 `BackupTask` 或 `SingleFileConfig`）→ 禁止并提示；否则确认后删除

### 多文件任务表单

- 「服务器」下拉：仅 `support_multi_file=true`
- 远程源目录浏览使用所选服务器的 SSH

### 单文件面板

- 「服务器」下拉：全部服务器
- 开始下载前必须已选服务器；SFTP 使用该服务器凭据

## 后端

### Store / API（建议）

| 能力 | 说明 |
|------|------|
| `GetServers` / `SaveServers` 或 CRUD | 读写 `servers[]` |
| `DeleteServer(id)` | 有引用则错误返回 |
| `SaveNotifyConfig` | 仅邮件字段（替代原面向路径/SSH 的全局保存语义） |
| 现有任务 / 单文件 Save | 校验 `server_id` |

测试连接、远程目录列表：入参改为显式 SSH 字段或 `server_id`（与所选服务器一致）。

### 运行时合并

**多文件**（init / start / reset / 状态相关）：

1. 按任务 `server_id` 取 `Server`（必须存在且 `support_multi_file`）
2. 合并：Server SSH + remote_app + max_part + Task 路径/前缀 + 全局邮件 → `BackupConfig`
3. `Resolved()` 推导远程程序/状态/staging 路径后进入现有 pipeline

**单文件**：

1. 按 `server_id` 取 `Server`
2. Server SSH + 全局邮件 + `remote_file`/`local_dir` → 现有 `StartDownload` 入参形状

### 校验

- 保存服务器：名称与 SSH 必填；`support_multi_file` 时 `remote_app_dir`、`max_part_gb` 必填
- 多文件任务：`server_id` 必填且目标服务器 `support_multi_file=true`
- 单文件：`server_id` 必填且服务器存在
- 删除服务器：有引用则拒绝
- 启动时服务器缺失或类型不符：明确中文错误

## 邮件

- 仍使用全局通知设置中的 SMTP / 通知邮箱
- 发信时 host 等展示信息来自**当前作业所选服务器**，而非已废弃的全局 SSH 字段

## 验收标准

1. Tab 右上角可见「服务器管理」「通知设置」；通知设置内无 SSH / 远程应用
2. 可添加多台服务器；勾选多文件后出现远程应用配置并可保存
3. 多文件任务只能选支持多文件的服务器；单文件可选全部
4. 被引用的服务器不可删除；无引用可删
5. 多文件 / 单文件启动使用所选服务器凭据；互斥与邮件行为回归正常
6. 升级后旧 SSH 不会自动出现在服务器列表；需用户重新添加

## 实现顺序建议

1. `Server` 模型 + Store + CRUD/删除引用校验
2. 收窄全局 `backup` 为通知配置；设置 UI 改为「通知设置」
3. 服务器管理弹窗（列表 + 添加/编辑表单 + 多文件勾选）
4. 任务 / 单文件绑定 `server_id` + 运行时合并改造
5. 联调与回归（多文件流水线、单文件下载、邮件、JobGate）
