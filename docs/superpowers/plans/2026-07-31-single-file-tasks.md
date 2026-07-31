# 单文件备份多任务 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 单文件备份 Tab 支持多任务列表（添加/编辑/删除/选用），与多文件共用 `backupTasks[]`，用 `kind` 区分；下载读 `activeSingleFileTaskId`。

**Architecture:** `BackupTask` 增加 `kind`（`multi`|`single`，缺省=multi）；store 增加 `activeSingleFileTaskId`；`SaveTasks`/`SetActive*` 按 kind 校验；废弃扁平 `singleFileBackup`（不迁移）。前端两 Tab 各自过滤任务列表；任务弹窗按 `kind` 切换字段；保存/删除前 `GetTasks` 全量合并，避免冲掉另一 kind。

**Tech Stack:** Go + Wails v3、Vue 3、Ant Design Vue、现有 `BackupTaskList` / `BackupTaskFormModal` / `SingleFileBackupService`。

**Spec:** `docs/superpowers/specs/2026-07-31-single-file-tasks-design.md`

## Global Constraints

- 不迁移旧 `singleFileBackup`；无 `kind` 的任务视为 `multi`
- `activeTaskId` 仅 multi；`activeSingleFileTaskId` 仅 single
- Vue 逻辑放同名 `.ts`，不在 `.vue` 写 TypeScript
- JobGate / 取消不发邮件 / 完成异常发邮件行为不变
- 保存任务列表必须保留另一 kind 的条目（`GetTasks` → 改子集 → `SaveTasks`）

## File Structure

| File | Responsibility |
|------|----------------|
| `backend/model/backup_task.go` | `Kind` 字段 + `NormalizedKind()` |
| `backend/config/store.go` | `ActiveSingleFileTaskID`；删除服务器不再查 `SingleFile` |
| `backend/service/backup_service.go` | `SaveTasks` 按 kind 分支；`SetActiveTaskID`/`SetActiveSingleFileTaskID` |
| `backend/service/single_file_service.go` | `StartDownload` 读 active single 任务；停用路径配置 API |
| `frontend/src/types/backup.ts` | `kind`、helpers |
| `frontend/src/composables/useBackupJob.ts` | 仅展示 multi；删除前全量合并 |
| `frontend/src/composables/useSingleFileBackup.ts` | 任务 CRUD + active + 启停 |
| `frontend/src/components/BackupTaskFormModal.*` | `kind` prop；single 表单 |
| `frontend/src/components/BackupTaskList.vue` | 按 kind 列头差异 |
| `frontend/src/components/SingleFileBackupPanel.*` | 任务列表布局，去掉内联路径表单 |
| `frontend/src/App.vue` | 单文件任务弹窗 `kind=single` |

---

### Task 1: 模型 Kind + Store activeSingleFileTaskId

**Files:**
- Modify: `backend/model/backup_task.go`
- Modify: `backend/config/store.go`
- Modify: `backend/config/store_test.go`
- Test: `backend/model/backup_task_kind_test.go`（新建）

**Interfaces:**
- Produces: `BackupTask.Kind string`（json:`kind`）；`NormalizedKind() string` → `"multi"`|`"single"`
- Produces: `Store.ActiveSingleFileTaskID`；`Get/SetActiveSingleFileTaskID`
- Produces: `DeleteServer` 仅检查 `BackupTasks[].ServerID`（含 single）

- [ ] **Step 1: 写失败测试（NormalizedKind + active ID round-trip）**

`backend/model/backup_task_kind_test.go`:

```go
package model

import "testing"

func TestBackupTaskNormalizedKind(t *testing.T) {
	if (BackupTask{}).NormalizedKind() != "multi" {
		t.Fatal("empty kind should be multi")
	}
	if (BackupTask{Kind: "MULTI"}).NormalizedKind() != "multi" {
		t.Fatal("case insensitive multi")
	}
	if (BackupTask{Kind: "single"}).NormalizedKind() != "single" {
		t.Fatal("single")
	}
	if (BackupTask{Kind: "other"}).NormalizedKind() != "multi" {
		t.Fatal("unknown -> multi")
	}
}
```

在 `store_test.go` 追加：

```go
func TestStoreActiveSingleFileTaskIDRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.ActiveSingleFileTaskID = "sf1"
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if s2.GetActiveSingleFileTaskID() != "sf1" {
		t.Fatalf("got %q", s2.GetActiveSingleFileTaskID())
	}
}

func TestStoreDeleteServerBlockedBySingleKindTask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.Servers = []model.Server{{ID: "s1", Name: "a", Host: "h", User: "u", Port: 22}}
	s.BackupTasks = []model.BackupTask{{
		ID: "t1", Kind: "single", ServerID: "s1",
		RemoteSource: `/var/a.bak`, LocalDir: `C:\b`,
	}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteServer("s1"); err == nil {
		t.Fatal("expected delete blocked by single task")
	}
}
```

并修改现有 `TestStoreDeleteServerBlockedWhenReferenced`：去掉「仅靠 `SingleFile` 阻止删除」断言；改为仅任务引用阻止，清空任务后即使 `SingleFile.ServerID` 仍有值也应允许删除。

- [ ] **Step 2: 跑测试 — 期望 FAIL**

Run: `go test ./backend/model/ -run TestBackupTaskNormalizedKind -v`  
Expected: `NormalizedKind` undefined

Run: `go test ./backend/config/ -run "TestStoreActiveSingleFileTaskID|TestStoreDeleteServerBlockedBySingleKind" -v`  
Expected: 缺字段 / 方法

- [ ] **Step 3: 实现**

`backup_task.go` 增加：

```go
Kind string `json:"kind,omitempty"` // "multi" | "single"；空=multi
```

```go
func (t BackupTask) NormalizedKind() string {
	switch strings.ToLower(strings.TrimSpace(t.Kind)) {
	case "single":
		return "single"
	default:
		return "multi"
	}
}
```

`store.go`：

- `diskConfig` / `Store` 增加 `ActiveSingleFileTaskID string \`json:"activeSingleFileTaskId,omitempty"\``
- `load`/`save` 读写该字段
- `GetActiveSingleFileTaskID` / `SetActiveSingleFileTaskID`（镜像 `ActiveTaskID`）
- `DeleteServer`：**删除**对 `s.SingleFile.ServerID` 的检查（保留 `BackupTasks` 循环）

- [ ] **Step 4: 跑测试 — 期望 PASS**

Run: `go test ./backend/model/ ./backend/config/ -count=1`

- [ ] **Step 5: Commit**

```bash
git add backend/model/backup_task.go backend/model/backup_task_kind_test.go backend/config/store.go backend/config/store_test.go
git commit -m "feat: add BackupTask kind and activeSingleFileTaskId"
```

---

### Task 2: SaveTasks / Active ID 按 kind 校验

**Files:**
- Modify: `backend/service/backup_service.go`（`SaveTasks`、`SetActiveTaskID`、新增 active single API）
- Create: `backend/service/backup_task_kind_test.go`
- Consumes: Task 1 `NormalizedKind`、`Get/SetActiveSingleFileTaskID`

**Interfaces:**
- Produces: `SaveTasks` 按 `NormalizedKind()` 分支校验
- Produces: `SetActiveTaskID` 拒绝 non-multi；`Get/SetActiveSingleFileTaskID` 拒绝 non-single
- Produces: 可选内部 `normalizeActiveIDs()`（active 无效则清空）

- [ ] **Step 1: 写失败测试**

`backend/service/backup_task_kind_test.go`（用临时 store 文件构造 `BackupService`，或测纯函数；若难注入 store，可把校验抽成）：

```go
func validateTaskForSave(store *config.Store, t model.BackupTask) error
```

放在 `backup_service.go` 同包，由 `SaveTasks` 调用。测试：

```go
func TestValidateTaskForSave_SingleAllowsAnyServer(t *testing.T) {
	dir := t.TempDir()
	s := &config.Store{ /* 需可测：用 NewStore 改 filePath 或导出测试 helper */ }
	// 更稳妥：在 store_test 同目录已有 pattern；此处用 tempfile + 反射/或直接测 service
}
```

实用写法：在 `backup_service.go` 旁建可测函数：

```go
// validateBackupTask checks one task for SaveTasks.
func validateBackupTask(servers []model.Server, t model.BackupTask) error {
	t.ID = strings.TrimSpace(t.ID)
	// ... normalize fields like SaveTasks
	if t.ID == "" {
		return fmt.Errorf("任务 ID 不能为空")
	}
	if t.ServerID == "" {
		return fmt.Errorf("任务 %s 未选择服务器", t.DisplayName())
	}
	srv, err := findServer(servers, t.ServerID) // 本地查找，不依赖 store
	if err != nil {
		return err
	}
	switch t.NormalizedKind() {
	case "single":
		if t.RemoteSource == "" {
			return fmt.Errorf("任务 %s 的远程源文件不能为空", t.DisplayName())
		}
		if t.LocalDir == "" {
			return fmt.Errorf("任务 %s 的本机保存目录不能为空", t.DisplayName())
		}
		return nil
	default: // multi
		if !srv.SupportMultiFile {
			return fmt.Errorf("任务 %s 所选服务器不支持多文件备份", t.DisplayName())
		}
		if t.RemoteSource == "" {
			return fmt.Errorf("任务 %s 的远程源目录不能为空", t.DisplayName())
		}
		if t.LocalDir == "" {
			return fmt.Errorf("任务 %s 的本机保存目录不能为空", t.DisplayName())
		}
		return nil
	}
}
```

测试文件：

```go
func TestValidateBackupTask_SingleVsMulti(t *testing.T) {
	servers := []model.Server{
		{ID: "s1", SupportMultiFile: false},
		{ID: "s2", SupportMultiFile: true},
	}
	single := model.BackupTask{
		ID: "a", Kind: "single", ServerID: "s1",
		RemoteSource: "/f.bak", LocalDir: `C:\o`,
	}
	if err := validateBackupTask(servers, single); err != nil {
		t.Fatal(err)
	}
	multiBad := model.BackupTask{
		ID: "b", ServerID: "s1",
		RemoteSource: `D:\d`, LocalDir: `C:\o`,
	}
	if err := validateBackupTask(servers, multiBad); err == nil {
		t.Fatal("multi on non-multi server should fail")
	}
	multiOK := model.BackupTask{
		ID: "c", ServerID: "s2",
		RemoteSource: `D:\d`, LocalDir: `C:\o`,
	}
	if err := validateBackupTask(servers, multiOK); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: 跑测试 — 期望 FAIL**

Run: `go test ./backend/service/ -run TestValidateBackupTask_SingleVsMulti -v`

- [ ] **Step 3: 实现校验 + 改造 SaveTasks / Active API**

`SaveTasks`：对每条调用 `validateBackupTask`；规范化时：

```go
kind := t.NormalizedKind()
t.Kind = kind // 持久化为 "multi" 或 "single"
if kind == "single" {
	t.PartNamePrefix = "" // 忽略前缀
} else {
	t.PartNamePrefix = zipbak.SanitizePartPrefix(t.PartNamePrefix)
}
```

`SetActiveTaskID`：若非空，任务必须存在且 `NormalizedKind()=="multi"`。

新增：

```go
func (s *BackupService) GetActiveSingleFileTaskID() string
func (s *BackupService) SetActiveSingleFileTaskID(taskID string) error
```

后者要求任务存在且 kind=single。

保存任务后可调用小型 `reconcileActiveIDs`：若当前 active 指向已删或 kind 不符 → `Set` 为空或同 kind 第一条。

- [ ] **Step 4: 跑测试 — 期望 PASS**

Run: `go test ./backend/service/ -count=1`

- [ ] **Step 5: Commit**

```bash
git add backend/service/backup_service.go backend/service/backup_task_kind_test.go
git commit -m "feat: validate backup tasks by kind and separate active IDs"
```

---

### Task 3: StartDownload 读 active 单文件任务

**Files:**
- Modify: `backend/service/single_file_service.go`
- Modify: `backend/service/single_file_path_test.go`（如需）
- Create: `backend/service/single_file_start_test.go`（尽量测「从任务拼路径」的纯函数）

**Interfaces:**
- Consumes: `store.GetActiveSingleFileTaskID`、`GetBackupTasks`、`lookupServer`
- Produces: `StartDownload()` 无路径入参，或保留旧签名但忽略 paths、以 store 为准
- 废弃前端对 `GetConfig`/`SaveConfig` 的依赖（本 Task 后端可保留方法但不再被下载使用；可在 `StartDownload` 前清空 `SingleFile` 可选）

推荐签名（破坏性更清晰）：

```go
func (s *SingleFileBackupService) StartDownload() error
```

实现要点：

```go
activeID := strings.TrimSpace(s.store.GetActiveSingleFileTaskID())
if activeID == "" {
	return fmt.Errorf("请先添加并选择单文件任务")
}
var task model.BackupTask
found := false
for _, t := range s.store.GetBackupTasks() {
	if t.ID == activeID {
		task = t
		found = true
		break
	}
}
if !found || task.NormalizedKind() != "single" {
	return fmt.Errorf("当前单文件任务无效，请重新选择")
}
srv, err := lookupServer(s.store, task.ServerID)
// ... ApplyTo notify, prepareSSH, singleFileLocalPath(task.RemoteSource, task.LocalDir)
```

- [ ] **Step 1: 写失败测试（路径拼装已有；测「任务→paths」helper）**

```go
func taskToSinglePaths(t model.BackupTask) (model.SingleFileConfig, error) {
	if t.NormalizedKind() != "single" {
		return model.SingleFileConfig{}, fmt.Errorf("不是单文件任务")
	}
	if strings.TrimSpace(t.ServerID) == "" {
		return model.SingleFileConfig{}, fmt.Errorf("请选择服务器")
	}
	rf := strings.TrimSpace(t.RemoteSource)
	ld := strings.TrimSpace(t.LocalDir)
	if rf == "" || ld == "" {
		return model.SingleFileConfig{}, fmt.Errorf("远程文件与本机目录不能为空")
	}
	return model.SingleFileConfig{ServerID: t.ServerID, RemoteFile: rf, LocalDir: ld}, nil
}
```

```go
func TestTaskToSinglePaths(t *testing.T) {
	_, err := taskToSinglePaths(model.BackupTask{Kind: "multi", ServerID: "s", RemoteSource: "a", LocalDir: "b"})
	if err == nil {
		t.Fatal("expected error")
	}
	p, err := taskToSinglePaths(model.BackupTask{
		Kind: "single", ServerID: "s", RemoteSource: `/a.bak`, LocalDir: `C:\o`,
	})
	if err != nil || p.RemoteFile != `/a.bak` {
		t.Fatalf("%+v %v", p, err)
	}
}
```

- [ ] **Step 2: 跑测 FAIL → Step 3 实现 StartDownload + helper → Step 4 PASS**

Run: `go test ./backend/service/ -run "TestTaskToSinglePaths|TestSingleFile" -count=1`

- [ ] **Step 5: Commit**

```bash
git add backend/service/single_file_service.go backend/service/single_file_start_test.go
git commit -m "feat: start single-file download from active task"
```

---

### Task 4: 前端类型 + 多文件侧过滤与安全保存

**Files:**
- Modify: `frontend/src/types/backup.ts`
- Modify: `frontend/src/composables/useBackupJob.ts`
- Modify: `frontend/src/components/BackupTaskFormModal.ts`
- Modify: `frontend/src/components/BackupTaskFormModal.vue`
- Modify: `frontend/src/App.vue`（传 `kind` 给多文件弹窗）

**Interfaces:**
- Produces: `BackupTask.kind?: "multi" | "single"`；`isMultiTask` / `isSingleTask` / `taskKind`
- Produces: `useBackupJob.tasks` 仅 multi；`removeTask`/`SaveTasks` 前 `GetTasks` 全量合并
- Produces: `BackupTaskFormModal` prop `kind: "multi" | "single"`（默认 multi）

- [ ] **Step 1: 更新 types**

```ts
export type BackupTaskKind = "multi" | "single";

export interface BackupTask {
  id: string;
  name?: string;
  kind?: BackupTaskKind;
  server_id?: string;
  remote_source: string;
  local_dir: string;
  part_name_prefix?: string;
}

export function taskKind(task: Pick<BackupTask, "kind">): BackupTaskKind {
  return task.kind === "single" ? "single" : "multi";
}
export function isMultiTask(task: Pick<BackupTask, "kind">) {
  return taskKind(task) === "multi";
}
export function isSingleTask(task: Pick<BackupTask, "kind">) {
  return taskKind(task) === "single";
}
```

- [ ] **Step 2: 改 useBackupJob**

```ts
const allTasks = ref<BackupTask[]>([]);
const tasks = computed(() => allTasks.value.filter(isMultiTask));

const loadTasks = async () => {
  allTasks.value = (await BackupService.GetTasks()) as BackupTask[];
  activeTaskId.value = await BackupService.GetActiveTaskID();
  const multi = tasks.value;
  if (!activeTaskId.value && multi.length > 0) {
    activeTaskId.value = multi[0].id;
    await BackupService.SetActiveTaskID(activeTaskId.value);
  }
};

const removeTask = async (task: BackupTask) => {
  const current = (await BackupService.GetTasks()) as BackupTask[];
  const next = current.filter((t) => t.id !== task.id);
  await BackupService.SaveTasks(next);
  // 再 loadTasks；若删的是 active，SetActiveTaskID 指向剩余 multi 第一条或 ""
};
```

`activeTask` 仍从 `tasks`（multi）查找。

- [ ] **Step 3: 改 BackupTaskFormModal**

Props 增加 `kind`（默认 `"multi"`）。

- 服务器选项：multi → `support_multi_file`；single → 全部 servers
- 远程浏览：multi → `RemoteDirPickerModal` 默认目录模式；single → `mode="file"`
- 文案：single 用「远程源文件」，隐藏「文件名前缀」
- `onSubmit` payload 带 `kind: props.kind`；新增后：
  - multi → `SetActiveTaskID`
  - single → `SetActiveSingleFileTaskID`
- `ensureServerSelected`：single 不要求 `support_multi_file`

Vue 模板按 `kind` 条件渲染标签与前缀字段、picker `mode`。

App 多文件弹窗：`:kind="'multi'"`（或默认即可）。

- [ ] **Step 4: 手动/类型检查**

Run（在 `frontend`）：`npx vue-tsc --noEmit`（若项目有该脚本则用 `npm run` 等价命令）

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types/backup.ts frontend/src/composables/useBackupJob.ts \
  frontend/src/components/BackupTaskFormModal.ts frontend/src/components/BackupTaskFormModal.vue \
  frontend/src/App.vue
git commit -m "feat: filter multi tasks and support kind in task form"
```

---

### Task 5: 单文件面板改为任务列表 + composable

**Files:**
- Modify: `frontend/src/composables/useSingleFileBackup.ts`
- Modify: `frontend/src/components/SingleFileBackupPanel.ts`
- Modify: `frontend/src/components/SingleFileBackupPanel.vue`
- Modify: `frontend/src/components/BackupTaskList.vue`（可选 `kind`/`columns` 以隐藏前缀）
- Modify: `frontend/src/App.vue`

**Interfaces:**
- Consumes: `BackupService.GetTasks/SaveTasks/SetActiveSingleFileTaskID`；`SingleFileBackupService.StartDownload()`（新签名）
- Produces: 单文件 Tab 与多文件同构的任务 UX

- [ ] **Step 1: 改写 useSingleFileBackup**

去掉 `paths` / `savePaths` / `GetConfig`/`SaveConfig`。改为：

```ts
const tasks = ref<BackupTask[]>([]); // 仅 single，或 all + computed
const activeTaskId = ref("");
const status / logs / start / stop 保留

const loadTasks = async () => {
  const all = (await BackupService.GetTasks()) as BackupTask[];
  tasks.value = all.filter(isSingleTask);
  activeTaskId.value = await BackupService.GetActiveSingleFileTaskID();
  if (!activeTaskId.value && tasks.value.length > 0) {
    activeTaskId.value = tasks.value[0].id;
    await BackupService.SetActiveSingleFileTaskID(activeTaskId.value);
  }
};

const selectTask / removeTask：删除时 GetTasks 全量 filter，再 SaveTasks；调整 active

const start = async () => {
  await SingleFileBackupService.StartDownload(); // 无路径入参
};
```

保留事件轮询。`App` 仍 `useSingleFileBackup({ panelActive })`。

- [ ] **Step 2: 改 SingleFileBackupPanel**

布局对齐 `BackupRunPanel` 任务区：

- 嵌入 `BackupTaskList`（`:tasks="singleTasks"`，列：任务 / 远程源文件 / 本机目录 / 操作；无前缀）
- 进度、开始/停止、日志保留
- 去掉内联表单与「保存路径」
- `startDisabled`：无 `activeTaskId` 或任务缺字段 / running / other job
- emit 或直接调 composable：`add`/`edit`/`select`/`remove`

`BackupTaskList.vue` 增加 prop：

```ts
variant?: "multi" | "single" // default multi
```

`single` 时第三列表头为「远程源文件」，隐藏前缀列。

- [ ] **Step 3: App.vue 接线**

- 单文件「添加/编辑」打开同一 `BackupTaskFormModal`，`:kind="'single'"`，`editingTask` 可共用或拆 `editingSingleTask`（避免与 multi 编辑冲突；推荐拆开或打开时设 kind）
- `@saved` 时同时 `loadTasks`（multi）与 `single.loadTasks()`
- `ServerManageDialog @changed` 也刷新单文件任务列表

推荐：

```ts
const taskFormKind = ref<"multi" | "single">("multi");
function openAddTask() {
  taskFormKind.value = "multi";
  editingTask.value = null;
  taskFormOpen.value = true;
}
function openAddSingleTask() {
  taskFormKind.value = "single";
  editingTask.value = null;
  taskFormOpen.value = true;
}
```

弹窗：`:kind="taskFormKind"`。

- [ ] **Step 4: 重新生成 Wails bindings（若 StartDownload 签名变了）**

按仓库惯例跑一次开发构建或 `wails3 generate`（见 `scripts/run-dev.sh`），确保 `frontend/bindings/.../singlefilebackupservice.js` 与 Go 一致。

- [ ] **Step 5: 手工验收清单**

1. 多文件 Tab：旧任务仍在；添加 multi 正常；列表无 single
2. 单文件 Tab：添加任务 → 列表出现 → 选用 → 开始下载（需真机/联调）
3. 删 multi 任务后 single 任务仍在（检查 config.json `backupTasks`）
4. 有 single 任务引用的服务器无法删除
5. 旧 `singleFileBackup` 不出现为任务

- [ ] **Step 6: Commit**

```bash
git add frontend/src/composables/useSingleFileBackup.ts \
  frontend/src/components/SingleFileBackupPanel.ts \
  frontend/src/components/SingleFileBackupPanel.vue \
  frontend/src/components/BackupTaskList.vue \
  frontend/src/App.vue \
  frontend/bindings
git commit -m "feat: single-file tab task list and download from active task"
```

---

## Spec Coverage Checklist

| Spec 项 | Task |
|---------|------|
| `kind` multi/single，缺省 multi | 1 |
| `activeSingleFileTaskId` | 1–2 |
| SaveTasks 按 kind 校验 | 2 |
| 不迁移 singleFileBackup；删除服务器只查 tasks | 1 |
| StartDownload 读 active single | 3 |
| 单文件 UI 任务列表 + 弹窗 | 4–5 |
| 多文件过滤 single；保存不冲掉另一 kind | 4–5 |
| JobGate / 邮件不变 | 3（不改 gate/notify） |

## Placeholder Scan

无 TBD；`StartDownload` 签名以 Task 3 为准改为无参；bindings 在 Task 5 再生。
