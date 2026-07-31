# 单文件备份多任务设计

日期：2026-07-31  
状态：已确认  
关联：`2026-07-31-single-file-backup-design.md`（本设计覆盖其中「本阶段不做多任务列表」）

## 背景

单文件备份 Tab 目前只有一组扁平配置（`singleFileBackup`：`server_id` / `remote_file` / `local_dir`），面板内联编辑路径。多文件 Tab 已支持任务列表 +「添加任务」弹窗。用户需要单文件侧同样可添加多条任务，选中后下载。

## 目标

- 单文件 Tab：任务列表 +「添加任务」；路径仅在弹窗中编辑
- 任务字段：名称（可选）、服务器、远程源文件、本机保存目录
- 列表选中当前任务后「开始下载」使用该任务
- 与多文件共用 `backupTasks[]`，通过 `kind` 区分

## 非目标

- 不为旧 `singleFileBackup` 自动建任务（用户自行添加）
- 不做单文件定时/调度
- 不做 multi ↔ single 任务互转
- 不改变 JobGate 互斥、取消不发邮件、SFTP 下载与邮件文案的既有行为

## 配置模型

### BackupTask 增加 `kind`

```json
{
  "backupTasks": [
    {
      "id": "task_xxx",
      "name": "日志备份",
      "kind": "single",
      "server_id": "srv_xxx",
      "remote_source": "/var/log/app.bak",
      "local_dir": "C:\\backup",
      "part_name_prefix": ""
    }
  ],
  "activeTaskId": "…",
  "activeSingleFileTaskId": "…"
}
```

| 字段 | multi | single |
|------|-------|--------|
| `kind` | `"multi"`（缺省/空视为 multi，兼容旧任务） | `"single"` |
| `server_id` | 必填且 `support_multi_file=true` | 必填，任意已存在服务器 |
| `remote_source` | 远程源**目录** | 远程源**文件**完整路径 |
| `local_dir` | 本机目录 | 本机目录 |
| `part_name_prefix` | 可选 | 忽略（可不写） |

### 当前任务 ID（分离）

- `activeTaskId`：仅指向 `kind=multi`（或未设 kind）的任务
- `activeSingleFileTaskId`：仅指向 `kind=single` 的任务
- 两 Tab 互不影响选中项

### 废弃 `singleFileBackup`

- 读配置时：若仍存在该块，**忽略**，不迁移为任务
- 写配置时：不再依赖；可清空或不写出
- 删除服务器时：仅检查 `backupTasks[].server_id` 引用（不再检查 `SingleFileConfig`）

## 后端

### 校验（`SaveTasks`）

按每条任务的 `kind` 分支：

- **multi**（含缺省）：与现有一致——服务器须支持多文件；`remote_source`、`local_dir` 必填
- **single**：服务器须存在（不要求 `support_multi_file`）；`remote_source`、`local_dir` 必填；`part_name_prefix` 可清空

整表保存：前端/后端合并时须保留另一 `kind` 的任务（避免单文件 Tab 保存时冲掉 multi 列表，反之亦然）。推荐：`SaveTasks` 仍接收完整列表；UI 保存时 `GetTasks` 后只替换本 kind 子集再写回。

### Active ID

- `SetActiveTaskID`：目标须为 multi（或空）
- `SetActiveSingleFileTaskID`（新）：目标须为 single（或空）
- 加载后若 active 指向错误 kind / 已删任务 → 清空或改指同 kind 第一条

### 下载

- `StartDownload`：从 store 取 `activeSingleFileTaskId` 对应任务，合并服务器 SSH + 全局邮件，再按现有单文件下载流程执行
- 也可保留入参校验，但**路径与 server 以 store 中当前单文件任务为准**（与多文件 `StartBackup` 读 active 任务一致）
- JobGate、进度、日志、邮件主题不变

### 服务器删除

- 引用检查：任一 `BackupTask.server_id`（含 single）→ 禁止删除

## 前端

### 单文件 Tab（对齐多文件布局）

```
SingleFileBackupPanel
  ├─ 任务列表（filter kind=single）+「添加任务」
  ├─ 进度 / 状态 / 本机文件
  ├─ 开始下载 / 停止
  └─ 日志
```

- 去掉面板内联「服务器 / 远程文件 / 本机目录 / 保存路径」表单
- 无任务或未选中：开始按钮禁用，提示先添加/选择任务

### 任务弹窗

- 复用或扩展 `BackupTaskFormModal`：按 `kind` 切换字段文案与校验
  - single：远程源**文件**（`PathPickInput` + 远程浏览 `mode=file`）；服务器下拉为**全部**服务器；无分卷前缀
  - multi：保持现有（仅支持多文件的服务器、源目录、前缀）
- 保存：`GetTasks` → 替换/追加本条 → `SaveTasks`（完整列表）

### 多文件 Tab

- `BackupTaskList` / `useBackupJob` 只展示与操作 `kind=multi`（含缺省）
- 添加任务强制 `kind=multi`

### 列表列（单文件）

- 任务名、远程源文件、本机目录、操作（选用 / 编辑 / 删除）；无「前缀」列

## 迁移

1. 旧任务无 `kind` → 视为 `"multi"`
2. 不从 `singleFileBackup` 生成任务
3. 读入后规范化 active：`activeTaskId` 只认 multi；`activeSingleFileTaskId` 只认 single

## 验收标准

1. 单文件 Tab 可添加 / 编辑 / 删除 / 选中任务；下载使用选中任务的服务器与路径
2. 多文件 Tab 列表不含 single 任务；旧 multi 任务行为不变
3. 旧 `singleFileBackup` 配置不自动出现为任务
4. JobGate 互斥、取消不发邮件、完成/异常发邮件与现有一致
5. 无可用单文件任务时无法启动下载
6. 被任一任务（含 single）引用的服务器不可删除

## 实现顺序建议

1. 模型 `kind` + store `activeSingleFileTaskId` + 校验分支 + 废弃 singleFile 引用检查
2. 单文件下载改为读 active single 任务
3. 前端类型 / 列表筛选 / 弹窗 kind / 单文件面板改任务列表布局
4. 联调与回归多文件
