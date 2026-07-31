# 服务器管理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 支持多服务器列表管理；多文件任务与单文件配置各自绑定 `server_id`；原「设置」收窄为仅邮件的「通知设置」。

**Architecture:** 新增 `servers[]` 持久化与 `Server` 模型；运行时将 Server + Task/单文件路径 + 全局邮件合并为现有 `BackupConfig` 形状再调用 pipeline。前端新增服务器管理弹窗；任务表单/单文件面板增加服务器下拉。不迁移旧全局 SSH。

**Tech Stack:** Go + Wails v3、Vue 3、Ant Design Vue、现有 SSH/SFTP/`BackupConfig.Resolved()`。

## Global Constraints

- 方案：独立 `servers[]` + `server_id` 引用（见规格）
- 未勾选多文件的服务器仅可用于单文件；多文件下拉只列 `support_multi_file=true`
- 有引用则禁止删除服务器
- 不自动迁移旧全局 SSH
- Vue 逻辑放同名 `.ts`，不在 `.vue` 写 TypeScript（用户规则）
- JobGate 互斥、取消不发邮件等行为保持不变
- `BackupConfig` 结构体仍作运行时作业形状保留 SSH 字段；全局 `backup` 持久化块只保留邮件

## File Structure

| File | Responsibility |
|------|----------------|
| `backend/model/server.go` | `Server` + `ApplyTo` / 校验辅助 |
| `backend/model/backup_task.go` | 增加 `ServerID` |
| `backend/model/single_file_config.go` | 增加 `ServerID` |
| `backend/config/store.go` | `servers[]` CRUD、删除引用检查 |
| `backend/service/backup_service.go` | Get/Save/Delete Servers、SaveNotifyConfig、合并作业 |
| `backend/service/single_file_service.go` | 按 `server_id` 合并 SSH |
| `backend/service/config_validate.go` | 按需补充服务器相关错误文案 |
| `frontend/src/types/backup.ts` | `Server` 类型与 `mergeJobConfig` |
| `frontend/src/components/ServerManageDialog.vue` + `.ts` | 列表 + 添加/编辑 |
| `frontend/src/components/BackupSettingsDialog.vue` + `BackupConfigPanel` | 仅邮件；标题「通知设置」 |
| `frontend/src/components/BackupTaskFormModal.vue` + `.ts` | 服务器下拉 |
| `frontend/src/components/SingleFileBackupPanel.vue` + `.ts` | 服务器下拉 |
| `frontend/src/App.vue` + `AppTabs.ts` / `useBackupJob.ts` | 按钮与数据流 |

---

### Task 1: Server 模型 + Store 持久化与删除引用校验

**Files:**
- Create: `backend/model/server.go`
- Modify: `backend/model/backup_task.go`
- Modify: `backend/model/single_file_config.go`
- Modify: `backend/config/store.go`
- Test: `backend/config/store_test.go`
- Test: `backend/model/server_test.go`

**Interfaces:**
- Produces:
  - `model.Server{ID, Name, Host, Port, User, Password string; SupportMultiFile bool; RemoteAppDir string; MaxPartGB float64}`
  - `Server.ApplyTo(base BackupConfig) BackupConfig` — 写入 SSH + remote_app + max_part（不覆盖任务路径/邮件）
  - `Store.GetServers() []Server` / `SetServers([]Server) error`
  - `Store.DeleteServer(id string) error` — 若任一 task/`SingleFile.ServerID` 引用则 `fmt.Errorf("服务器仍被引用，无法删除")`
  - `BackupTask.ServerID` / `SingleFileConfig.ServerID` JSON `server_id`

- [ ] **Step 1: Write failing tests**

`backend/model/server_test.go`:

```go
package model

import "testing"

func TestServerApplyTo(t *testing.T) {
	base := BackupConfig{NotifyEmail: "a@b.c", SmtpHost: "smtp", RemoteSource: "keep"}
	srv := Server{
		Host: "10.0.0.1", Port: 22, User: "u", Password: "p",
		SupportMultiFile: true, RemoteAppDir: `D:\zipbak`, MaxPartGB: 3,
	}
	out := srv.ApplyTo(base)
	if out.Host != "10.0.0.1" || out.RemoteAppDir != `D:\zipbak` || out.MaxPartGB != 3 {
		t.Fatalf("ssh/app: %+v", out)
	}
	if out.NotifyEmail != "a@b.c" || out.RemoteSource != "keep" {
		t.Fatalf("should keep notify/task fields: %+v", out)
	}
}
```

`backend/config/store_test.go` 追加：

```go
func TestStoreServersRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.Servers = []model.Server{{
		ID: "s1", Name: "生产", Host: "1.1.1.1", Port: 22, User: "u",
		SupportMultiFile: true, RemoteAppDir: `D:\z`, MaxPartGB: 2,
	}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.Servers) != 1 || s2.Servers[0].ID != "s1" {
		t.Fatalf("got %+v", s2.Servers)
	}
}

func TestStoreDeleteServerBlockedWhenReferenced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.Servers = []model.Server{{ID: "s1", Name: "a", Host: "h", User: "u", Port: 22}}
	s.BackupTasks = []model.BackupTask{{ID: "t1", ServerID: "s1", RemoteSource: `D:\a`, LocalDir: `C:\b`}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteServer("s1"); err == nil {
		t.Fatal("expected delete blocked")
	}
	s.BackupTasks = nil
	s.SingleFile = model.SingleFileConfig{ServerID: "s1", RemoteFile: `D:\f`, LocalDir: `C:\b`}
	_ = s.save()
	if err := s.DeleteServer("s1"); err == nil {
		t.Fatal("expected delete blocked by single file")
	}
	s.SingleFile = model.SingleFileConfig{}
	_ = s.save()
	if err := s.DeleteServer("s1"); err != nil {
		t.Fatal(err)
	}
	if len(s.GetServers()) != 0 {
		t.Fatal("expected empty")
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL**

Run: `go test ./backend/model/ ./backend/config/ -run "TestServerApplyTo|TestStoreServers|TestStoreDeleteServer" -v`  
Expected: compile error / missing types

- [ ] **Step 3: Implement**

`backend/model/server.go`:

```go
package model

type Server struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Host             string  `json:"host"`
	Port             int     `json:"port"`
	User             string  `json:"user"`
	Password         string  `json:"password"`
	SupportMultiFile bool    `json:"support_multi_file"`
	RemoteAppDir     string  `json:"remote_app_dir,omitempty"`
	MaxPartGB        float64 `json:"max_part_gb,omitempty"`
}

func (s Server) ApplyTo(base BackupConfig) BackupConfig {
	out := base
	out.Host = s.Host
	out.Port = s.Port
	out.User = s.User
	out.Password = s.Password
	out.RemoteAppDir = s.RemoteAppDir
	if s.MaxPartGB > 0 {
		out.MaxPartGB = s.MaxPartGB
	}
	return out
}
```

`BackupTask` 增加 `ServerID string \`json:"server_id,omitempty"\``；`SingleFileConfig` 增加同样字段。

`store.go`：`diskConfig`/`Store` 增加 `Servers []model.Server \`json:"servers,omitempty"\``；load/save 读写；实现 `GetServers`/`SetServers`/`DeleteServer`（删除前扫描 tasks 与 SingleFile.ServerID）。

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./backend/model/ ./backend/config/ -run "TestServerApplyTo|TestStoreServers|TestStoreDeleteServer" -v`

- [ ] **Step 5: Commit**

```bash
git add backend/model/server.go backend/model/server_test.go backend/model/backup_task.go backend/model/single_file_config.go backend/config/store.go backend/config/store_test.go
git commit -m "feat: persist servers list with delete reference checks"
```

---

### Task 2: 后端服务 API（Servers CRUD、通知保存、作业合并）

**Files:**
- Modify: `backend/service/backup_service.go`
- Modify: `backend/service/single_file_service.go`
- Modify: `backend/service/config_validate.go`（若需新错误）
- Test: `backend/service/server_merge_test.go`

**Interfaces:**
- Produces（Wails 暴露）:
  - `GetServers() []model.Server`
  - `SaveServer(srv model.Server) (model.Server, error)` — 无 id 则生成；校验名称/SSH；`SupportMultiFile` 时要求 `RemoteAppDir` 与 `MaxPartGB>0`
  - `DeleteServer(id string) error` — 调 store
  - `SaveNotifyConfig(cfg model.BackupConfig) error` — 只写邮件字段到 `backup`
  - `GetConfig()` 仍返回 store 的 `backup`（预期主要为邮件；SSH 可为空）
  - `BuildJobConfig(taskID string) (model.BackupConfig, error)` 或内部：`resolveJobFromTask(task) (BackupConfig, error)` = notify + Server.ApplyTo + task.MergeInto + Resolved
  - `SaveTasks`：校验每任务 `ServerID` 非空，且对应 server 存在且 `SupportMultiFile`
  - `SingleFileBackupService.SaveConfig`：校验 `ServerID`；`StartDownload`：按 paths.ServerID 取 server，再合并邮件后下载（可改为只传 paths，内部读 store；或保留显式 cfg 但前端传入合并结果——**本计划约定：StartDownload 从 store 按 `paths.ServerID` 解析 Server + 邮件，忽略入参 SSH 或要求入参可为空；为少改前端，优先：`StartDownload(paths)` 内部读 store，或保持 `StartDownload(sshCfg, paths)` 但服务端用 store 覆盖 SSH**）
  - **约定（锁定）：** `StartDownload(cfg BackupConfig, paths SingleFileConfig)`：服务端用 `paths.ServerID` 查 Server，用 `store` 邮件覆盖/合并，再 `prepareSSH`；前端仍可传合并后的 cfg 作兜底，但**以 store 为准**
  - `activeTaskMerged` / `StartBackup`/`InitRemote` 等：若调用方传入的 cfg 已含 SSH 则继续用传入 cfg（前端合并后传入）；同时 `SaveTasks` 强制 server_id。前端 Task 3–5 负责合并传入。后端增加 `LookupServer(id) (Server, error)` 供服务内校验。

- [ ] **Step 1: Write failing test**

`backend/service/server_merge_test.go`:

```go
package service

import (
	"testing"

	"z-server-backup-tools/backend/model"
)

func TestMergeServerTaskNotify(t *testing.T) {
	notify := model.BackupConfig{NotifyEmail: "n@e.com", SmtpHost: "smtp"}
	srv := model.Server{
		ID: "s1", Host: "h", Port: 22, User: "u", Password: "p",
		SupportMultiFile: true, RemoteAppDir: `D:\app`, MaxPartGB: 2,
	}
	task := model.BackupTask{
		ID: "t1", ServerID: "s1", RemoteSource: `D:\src`, LocalDir: `C:\out`, PartNamePrefix: "p",
	}
	out := mergeServerTaskNotify(notify, srv, task)
	if out.Host != "h" || out.RemoteSource != `D:\src` || out.NotifyEmail != "n@e.com" || out.RemoteSrv == "" {
		t.Fatalf("got %+v", out)
	}
}
```

导出包内函数：

```go
func mergeServerTaskNotify(notify model.BackupConfig, srv model.Server, task model.BackupTask) model.BackupConfig {
	return task.MergeInto(srv.ApplyTo(notify)).Resolved()
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./backend/service/ -run TestMergeServerTaskNotify -v`

- [ ] **Step 3: Implement service methods**

在 `backup_service.go`：

```go
func (s *BackupService) GetServers() []model.Server { /* store */ }

func (s *BackupService) SaveServer(srv model.Server) (model.Server, error) {
	// normalize + validate; if ID empty generate like task id; upsert into slice; SetServers
}

func (s *BackupService) DeleteServer(id string) error {
	return s.store.DeleteServer(id)
}

func (s *BackupService) SaveNotifyConfig(cfg model.BackupConfig) error {
	stored := s.storedConfig()
	stored.NotifyEmail = strings.TrimSpace(cfg.NotifyEmail)
	stored.SmtpHost = strings.TrimSpace(cfg.SmtpHost)
	stored.SmtpPort = cfg.SmtpPort
	stored.SmtpUser = strings.TrimSpace(cfg.SmtpUser)
	stored.SmtpPassword = cfg.SmtpPassword
	// 剥离全局 SSH/远程应用（写回空），避免 UI 误用
	stored.Host, stored.User, stored.Password, stored.RemoteAppDir = "", "", "", ""
	stored.Port = 22
	stored.MaxPartGB = 0
	return s.store.SetBackupConfig(stored)
}
```

`SaveTasks` 增加对 `ServerID` 与 `SupportMultiFile` 的校验。

`SaveConnectionConfig` / `SavePathsConfig`：可改为内部转调 `SaveNotifyConfig`（若只剩邮件）或保留但不再要求 `RemoteAppDir`——**锁定：前端不再调用 SaveConnection；SavePaths 改为调用 SaveNotifyConfig。后端保留旧方法短期兼容，但 `SavePathsConfig` 改为只存邮件且不校验 remote_app_dir。**

`SingleFileBackupService.SaveConfig`：要求 `ServerID` 非空且 server 存在。

`StartDownload` 开头：

```go
srv, err := lookupServer(s.store, paths.ServerID)
if err != nil { return err }
notify := s.store.GetBackupConfig()
cfg = srv.ApplyTo(notify)
// then existing download with cfg + paths
```

- [ ] **Step 4: Run tests**

Run: `go test ./backend/service/ ./backend/config/ ./backend/model/ -count=1`

- [ ] **Step 5: Commit**

```bash
git add backend/service/backup_service.go backend/service/single_file_service.go backend/service/server_merge_test.go backend/service/config_validate.go
git commit -m "feat: server CRUD and merge job config from server_id"
```

---

### Task 3: 前端类型与合并辅助 + 通知设置 UI

**Files:**
- Modify: `frontend/src/types/backup.ts`
- Modify: `frontend/src/components/BackupConfigPanel.vue`（仅邮件区块；逻辑抽到 `.ts` 若新增）
- Modify: `frontend/src/components/BackupSettingsDialog.vue` — title「通知设置」
- Modify: `frontend/src/composables/useBackupJob.ts` — `saveNotify`；去掉 saveConnection 依赖；加载 servers
- Modify: `frontend/src/App.vue`

**Interfaces:**
- Produces:
  - `export interface Server { id, name, host, port, user, password, support_multi_file, remote_app_dir?, max_part_gb? }`
  - `BackupTask.server_id` / `SingleFileConfig.server_id`
  - `mergeJobConfig(notify: BackupConfig, server: Server | null | undefined, task?: BackupTask | null): BackupConfig`
  - `emptyNotifyConfig(): BackupConfig`

- [ ] **Step 1: Update types**

在 `backup.ts` 增加 `Server`、字段 `server_id`；实现：

```ts
export function applyServer(base: BackupConfig, server?: Server | null): BackupConfig {
  if (!server) return { ...base, host: "", user: "", password: "", remote_app_dir: "", max_part_gb: 0 };
  return {
    ...base,
    host: server.host,
    port: server.port || 22,
    user: server.user,
    password: server.password,
    remote_app_dir: server.remote_app_dir ?? "",
    max_part_gb: server.max_part_gb || 2,
  };
}

export function mergeJobConfig(
  notify: BackupConfig,
  server: Server | null | undefined,
  task?: BackupTask | null,
): BackupConfig {
  return mergeTaskConfig(applyServer(notify, server), task);
}
```

- [ ] **Step 2: Narrow settings UI**

- `BackupSettingsDialog`：`title="通知设置"`；去掉 `@save-connection`
- `BackupConfigPanel`：删除 SSH、远程应用整块；仅保留邮件 +「邮箱测试」；保存触发 `save-paths`（语义改为通知）或新事件 `save-notify`
- `BackupRunPanel`：移除「设置」按钮与 `openSettings` emit
- `useBackupJob`：`savePaths` → 调用 `BackupService.SaveNotifyConfig`（bindings 生成后）；增加 `servers` ref + `loadServers`/`saveServer`/`deleteServer`
- `App.vue`：`jobConfig` 改为 `mergeJobConfig(config, findServer(activeTask?.server_id), activeTask)`

- [ ] **Step 3: Typecheck**

Run: `cd frontend && npx vue-tsc --noEmit`  
Expected: PASS（若 bindings 尚无新方法，可先用 any 临时或先跑 `wails3 generate bindings -ts`）

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/backup.ts frontend/src/components/BackupConfigPanel.vue frontend/src/components/BackupSettingsDialog.vue frontend/src/components/BackupRunPanel.vue frontend/src/composables/useBackupJob.ts frontend/src/App.vue
git commit -m "feat: narrow settings to notify-only and add server merge helpers"
```

---

### Task 4: 服务器管理弹窗

**Files:**
- Create: `frontend/src/components/ServerManageDialog.vue`
- Create: `frontend/src/components/ServerManageDialog.ts`
- Modify: `frontend/src/App.vue` — Tab `rightExtra` 增加「服务器管理」

**Interfaces:**
- Consumes: `BackupService.GetServers/SaveServer/DeleteServer/TestConnection/ListRemoteDirectories`
- Produces: 弹窗 `v-model:open`；`@changed` 通知父级刷新 servers

- [ ] **Step 1: 实现 `ServerManageDialog.ts`**

逻辑要点：
- `servers` 列表加载
- 添加/编辑表单：`name, host, port, user, password, support_multi_file, remote_app_dir, max_part_gb`
- `support_multi_file` 勾选才显示远程应用 + `RemoteDirPickerModal`（connection 用表单当前 SSH 拼 `BackupConfig`）
- 保存调用 `SaveServer`
- 删除调用 `DeleteServer`，错误 `message.error`
- 测试连接：`TestConnection(applyServer(emptyNotify, form))`

- [ ] **Step 2: 实现 `.vue` 模板**

- `a-modal` 宽约 920，高度可滚动
- 标题「服务器管理」；右上角「添加」
- `a-table` 或列表：名称、主机、多文件支持、编辑、删除
- 添加/编辑可用嵌套 Modal 或同一弹窗内切换视图

- [ ] **Step 3: 挂到 App**

```vue
<a-button size="small" @click="serversOpen = true">服务器管理</a-button>
<a-button size="small" @click="settingsOpen = true">通知设置</a-button>
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/ServerManageDialog.vue frontend/src/components/ServerManageDialog.ts frontend/src/App.vue
git commit -m "feat: add server management dialog"
```

---

### Task 5: 任务表单与单文件面板绑定 server_id

**Files:**
- Modify: `frontend/src/components/BackupTaskFormModal.vue`（逻辑迁到 `BackupTaskFormModal.ts`）
- Create: `frontend/src/components/BackupTaskFormModal.ts`（若尚未存在）
- Modify: `frontend/src/components/SingleFileBackupPanel.vue` + `.ts`
- Modify: `frontend/src/App.vue` — 向表单传入 `servers` 列表

**Interfaces:**
- Task form props: `servers: Server[]`（筛选 `support_multi_file` 在组件内）
- 远程目录浏览：用所选 server 的 SSH，不再用全局 `connection.host`
- Single file: 下拉全部 servers；`SaveConfig`/`start` 带 `server_id`

- [ ] **Step 1: BackupTaskFormModal**

- 增加「服务器」`a-select`，options = `servers.filter(s => s.support_multi_file)`
- 保存时 `server_id` 必填
- `ensureConnectionFilled` 改为校验已选服务器
- 打开远程浏览：`connectionForPicker = applyServer(notifyOrEmpty, selectedServer)`

- [ ] **Step 2: SingleFileBackupPanel**

- 增加服务器下拉（全部）
- 持久化 `server_id` 与路径一起 `SaveConfig`
- `onStart`：若无 `server_id` 提示；`start` 前用 store 解析（后端已按 server_id 覆盖）

- [ ] **Step 3: App 传 props**

- `BackupTaskFormModal` 增加 `:servers="servers"`
- 确保 `loadServers` 在 onMounted 与 ServerManageDialog `@changed` 时刷新

- [ ] **Step 4: vue-tsc + go test**

Run:
```bash
go test ./...
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/BackupTaskFormModal.vue frontend/src/components/BackupTaskFormModal.ts frontend/src/components/SingleFileBackupPanel.vue frontend/src/components/SingleFileBackupPanel.ts frontend/src/App.vue frontend/src/composables/useBackupJob.ts frontend/src/composables/useSingleFileBackup.ts
git commit -m "feat: bind backup task and single-file config to server_id"
```

---

### Task 6: Bindings 生成与联调验收

**Files:**
- 本地生成：`frontend/bindings/`（gitignore，不提交）
- 按需小修编译错误

- [ ] **Step 1: 生成 bindings**

Run（在仓库根，按项目惯例）:
```bash
wails3 generate bindings -ts
```
或 `scripts/run-dev` 触发的生成步骤。确认出现 `GetServers`/`SaveServer`/`DeleteServer`/`SaveNotifyConfig`。

- [ ] **Step 2: 全量验证**

```bash
go test ./...
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 3: 手工对照验收（规格）**

1. Tab 右上角「服务器管理」「通知设置」；通知内无 SSH/远程应用  
2. 添加服务器；勾选多文件后出现远程应用并可保存  
3. 多文件任务只能选支持多文件的服务器；单文件可选全部  
4. 被引用不可删；无引用可删  
5. 启动使用所选服务器；JobGate/邮件回归  
6. 旧 SSH 不出现在服务器列表  

- [ ] **Step 4: Commit 残留修复（若有）**

```bash
git commit -m "fix: server management integration polish"
```

---

## Spec coverage checklist

| 规格项 | Task |
|--------|------|
| `servers[]` 模型与存储 | 1 |
| `server_id` 引用 | 1–2, 5 |
| 删除禁止有引用 | 1–2 |
| SaveNotify / 设置仅邮件 / 文案「通知设置」 | 2–3 |
| 服务器管理 UI + 多文件勾选 | 4 |
| 任务/单文件下拉与运行时合并 | 2, 5 |
| 不迁移旧 SSH | 2（SaveNotify 剥离）+ 验收 6 |
| 移除面板内重复设置入口 | 3 |

## Placeholder / consistency notes

- 运行时作业形状统一为 `BackupConfig`；`mergeServerTaskNotify` / `mergeJobConfig` 命名前后端对应
- `SavePathsConfig` 收窄为邮件；前端改调 `SaveNotifyConfig`
- `StartDownload` 以 store 中 `server_id` 为准解析 Server
