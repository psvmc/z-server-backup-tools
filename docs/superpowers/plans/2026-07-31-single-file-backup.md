# 单文件备份 Tab Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 主页拆成「多文件备份 / 单文件备份」两个 Tab；单文件模式不依赖 zipbak-srv、不压缩，直接 SFTP 下载并显示字节进度。

**Architecture:** 新增独立 `SingleFileBackupService`（独立 status/logs/cancel），配置存 `config.json` 的 `singleFileBackup`；与多文件通过包内 `JobGate` 互斥。前端 `App.vue` 用 `a-tabs` 挂载现有面板 + 新 `SingleFileBackupPanel`；SSH/邮件仍走共享设置。

**Tech Stack:** Go + Wails v3、Vue 3、Ant Design Vue、现有 `sftpclient.DownloadWithProgress`、`notify` 邮件。

## Global Constraints

- 单文件：仅一个远程文件路径；本机文件名 = 远程 basename
- 不上传程序、远程不压缩
- SSH + 邮件与多文件共享；`remote_file` / `local_dir` 独立
- Vue 逻辑放同名 `.ts`，不在 `.vue` 写 TypeScript（用户规则）
- 取消下载不发邮件；完成/异常发邮件（有通知邮箱时）
- 多文件与单文件不可同时运行

## File Structure

| File | Responsibility |
|------|----------------|
| `backend/model/single_file_config.go` | `SingleFileConfig` 结构体 |
| `backend/config/store.go` | 持久化 `singleFileBackup` |
| `backend/service/job_gate.go` | 多文件/单文件互斥锁 |
| `backend/service/single_file_service.go` | 下载 API、进度、日志、邮件 |
| `backend/notify/email.go` | `SendSingleFileNotification` |
| `main.go` | 注册 `SingleFileBackupService` |
| `frontend/src/types/backup.ts` | TS 类型 |
| `frontend/src/composables/useSingleFileBackup.ts` | 状态/启停/轮询 |
| `frontend/src/components/SingleFileBackupPanel.vue` + `.ts` | 单文件 UI |
| `frontend/src/App.vue` | Tab 布局 |
| `frontend/src/components/BackupRunPanel.vue` | 运行中禁用切 Tab 所需的 running 暴露（若需要） |

---

### Task 1: SingleFileConfig + Store 持久化

**Files:**
- Create: `backend/model/single_file_config.go`
- Modify: `backend/config/store.go`
- Test: `backend/config/store_test.go`

**Interfaces:**
- Produces: `model.SingleFileConfig{RemoteFile, LocalDir string}`；`Store.GetSingleFileConfig() / SetSingleFileConfig(cfg)`

- [ ] **Step 1: Write failing test**

在 `backend/config/store_test.go` 追加：

```go
func TestStoreSingleFileBackupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &Store{filePath: path}
	s.SingleFile = model.SingleFileConfig{
		RemoteFile: `D:\data\app.bak`,
		LocalDir:   `C:\backup`,
	}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{filePath: path}
	if err := s2.load(); err != nil {
		t.Fatal(err)
	}
	if s2.SingleFile.RemoteFile != `D:\data\app.bak` || s2.SingleFile.LocalDir != `C:\backup` {
		t.Fatalf("got %+v", s2.SingleFile)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./backend/config/ -run TestStoreSingleFileBackupRoundTrip -v`  
Expected: compile error / missing field

- [ ] **Step 3: Implement model + store**

`backend/model/single_file_config.go`:

```go
package model

type SingleFileConfig struct {
	RemoteFile string `json:"remote_file"`
	LocalDir   string `json:"local_dir"`
}
```

在 `diskConfig` / `Store` 增加 `SingleFile model.SingleFileConfig \`json:"singleFileBackup,omitempty"\``；`load`/`save` 读写该字段；增加：

```go
func (s *Store) GetSingleFileConfig() model.SingleFileConfig { /* lock+copy */ }
func (s *Store) SetSingleFileConfig(cfg model.SingleFileConfig) error { /* load, set, save */ }
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./backend/config/ -run TestStoreSingleFileBackupRoundTrip -v`

- [ ] **Step 5: Commit**

```bash
git add backend/model/single_file_config.go backend/config/store.go backend/config/store_test.go
git commit -m "feat: persist single-file backup path config"
```

---

### Task 2: JobGate 互斥

**Files:**
- Create: `backend/service/job_gate.go`
- Modify: `backend/service/backup_service.go`（`StartBackup` / `runPipeline` defer 释放）
- Test: `backend/service/job_gate_test.go`

**Interfaces:**
- Produces: `TryAcquireMulti() error`、`ReleaseMulti()`、`TryAcquireSingle() error`、`ReleaseSingle()`、`MultiRunning() bool`、`SingleRunning() bool`
- Consumes: 无

- [ ] **Step 1: Write failing test**

```go
func TestJobGateMutex(t *testing.T) {
	g := NewJobGate()
	if err := g.TryAcquireMulti(); err != nil {
		t.Fatal(err)
	}
	if err := g.TryAcquireSingle(); err == nil {
		t.Fatal("expected conflict")
	}
	g.ReleaseMulti()
	if err := g.TryAcquireSingle(); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./backend/service/ -run TestJobGateMutex -v`

- [ ] **Step 3: Implement JobGate + wire BackupService**

`job_gate.go`：包级 `var defaultJobGate = NewJobGate()`；用 `sync.Mutex` + `multi`/`single` bool。冲突时返回 `fmt.Errorf("已有备份任务在运行")` 或 `已有单文件下载在运行`。

`StartBackup`：在确认未 running 后 `TryAcquireMulti()`，失败则 return；`runPipeline` 的 `defer` 中 `ReleaseMulti()`（与 `Running=false` 一起）。

- [ ] **Step 4: Run tests**

Run: `go test ./backend/service/ -run TestJobGateMutex -v`  
Run: `go test ./backend/service/ ./backend/zipbak/ ./backend/config/ -count=1`

- [ ] **Step 5: Commit**

```bash
git add backend/service/job_gate.go backend/service/job_gate_test.go backend/service/backup_service.go
git commit -m "feat: add job gate for multi vs single file mutual exclusion"
```

---

### Task 3: SingleFileBackupService 下载核心

**Files:**
- Create: `backend/service/single_file_service.go`
- Modify: `backend/notify/email.go`（`SendSingleFileNotification`）
- Modify: `main.go`（注册服务）
- Test: `backend/service/single_file_path_test.go`（本地路径拼接纯函数，若抽出 helper）

**Interfaces:**
- Consumes: `config.Store`、`JobGate`、`sftpclient.DownloadWithProgress`、`prepareSSH`、共享 `BackupConfig` 邮件字段
- Produces Wails methods:
  - `GetConfig() model.SingleFileConfig`
  - `SaveConfig(cfg model.SingleFileConfig) error`
  - `StartDownload(sshCfg model.BackupConfig, paths model.SingleFileConfig) error`
  - `StopDownload()`
  - `GetStatus() model.JobStatus`
  - `GetLogs() []string`

- [ ] **Step 1: Add notify helper**

在 `email.go` 增加：

```go
func SendSingleFileNotification(cfg model.BackupConfig, success bool, remoteFile, localPath, errMsg string) error {
	to := strings.TrimSpace(cfg.NotifyEmail)
	if to == "" {
		return nil
	}
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "（未知）"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	var subject, body string
	if success {
		subject = fmt.Sprintf("[单文件下载完成] %s", host)
		body = fmt.Sprintf("单文件下载已完成。\n\n时间：%s\n主机：%s\n远程文件：%s\n本机文件：%s\n", now, host, remoteFile, localPath)
	} else {
		subject = fmt.Sprintf("[单文件下载异常] %s", host)
		body = fmt.Sprintf("单文件下载异常停止。\n\n时间：%s\n主机：%s\n远程文件：%s\n本机文件：%s\n错误：%s\n", now, host, remoteFile, localPath, errMsg)
	}
	return sendConfiguredMail(cfg, to, subject, body)
}
```

- [ ] **Step 2: Implement service**

`NewSingleFileBackupService(app *application.App) *SingleFileBackupService`：持有 `store`、`mu`、`cancel`、`status model.JobStatus`、`logs []string`。

`SaveConfig`：trim 字段，`remote_file`/`local_dir` 非空校验，`store.SetSingleFileConfig`。

`StartDownload`：
1. `prepareSSH(sshCfg)`
2. trim paths；空则 error
3. `defaultJobGate.TryAcquireSingle()`；失败 return
4. 若已 running，Release 并 return error
5. `localPath := filepath.Join(localDir, filepath.Base(filepath.FromSlash(strings.ReplaceAll(remoteFile, `\`, `/`))))` — Windows 远程路径用 `path/filepath` 或自写 basename：取最后一个 `\` 或 `/` 后段
6. `ctx, cancel := context.WithCancel`；设 `status.Running=true, Phase=download, LocalFile=localPath`
7. goroutine：`sftpclient.Dial` → `DownloadWithProgress`，进度写 `DownloadBytesDone/Total/SpeedBps`（参考 `pipeline.download`）
8. 成功：emit `singlefile-done`，发成功邮件；失败：若非取消则邮件+`singlefile-error`；defer `ReleaseSingle`、`Running=false`

`StopDownload`：调用 `cancel()`。

事件名：`singlefile-log` / `singlefile-done` / `singlefile-error`。

- [ ] **Step 3: Register in main.go**

```go
app.RegisterService(application.NewService(service.NewSingleFileBackupService(app)))
```

- [ ] **Step 4: Unit test basename helper**

若抽出 `singleFileLocalPath(remoteFile, localDir string) (string, error)`：

```go
func TestSingleFileLocalPath(t *testing.T) {
	p, err := singleFileLocalPath(`D:\data\app.bak`, `C:\out`)
	if err != nil || p != filepath.Join(`C:\out`, `app.bak`) {
		t.Fatalf("%q %v", p, err)
	}
}
```

Run: `go test ./backend/service/ -run TestSingleFileLocalPath -v`

- [ ] **Step 5: Generate bindings**

Run: `wails3 generate bindings -ts`

- [ ] **Step 6: Commit**

```bash
git add backend/service/single_file_service.go backend/service/single_file_path_test.go backend/notify/email.go main.go frontend/bindings
git commit -m "feat: add single-file SFTP download service"
```

---

### Task 4: 前端类型 + useSingleFileBackup

**Files:**
- Modify: `frontend/src/types/backup.ts`
- Create: `frontend/src/composables/useSingleFileBackup.ts`

**Interfaces:**
- Consumes: bindings `SingleFileBackupService`、共享 `BackupConfig`（SSH/邮件来自 `useBackupJob` 的 config 或 GetConfig）
- Produces: `{ paths, status, logs, saving, load, savePaths, start, stop }`

- [ ] **Step 1: Add TS types**

```ts
export interface SingleFileConfig {
  remote_file: string;
  local_dir: string;
}
```

- [ ] **Step 2: Implement composable**

- `load`：`GetConfig` + 轮询 `GetStatus`/`GetLogs`
- `savePaths`：`SaveConfig`
- `start(sshCfg)`：合并当前 paths 调 `StartDownload`
- `stop`：`StopDownload`
- 监听 `Events.On("singlefile-log"|done|error)`
- 1.5s interval 刷新 status（仅当 running 或面板可见时可常刷）

- [ ] **Step 3: Commit**

```bash
git add frontend/src/types/backup.ts frontend/src/composables/useSingleFileBackup.ts
git commit -m "feat: add useSingleFileBackup composable"
```

---

### Task 5: SingleFileBackupPanel UI

**Files:**
- Create: `frontend/src/components/SingleFileBackupPanel.vue`
- Create: `frontend/src/components/SingleFileBackupPanel.ts`（props/emits/helpers 若需；进度文案可放此文件）
- Reuse: `PathPickInput.vue`、`Dialogs.OpenFile` 选本机目录、`BackupService.OpenInExplorer`

**Interfaces:**
- Props: `sshConfig: BackupConfig`、`disabledByOtherJob?: boolean`
- Emits: 无强制；内部调 composable

- [ ] **Step 1: Panel layout**

- 远程源文件：`PathPickInput` `editable`；首版可不做远程选文件，placeholder 提示如 `D:\data\app.bak`
- 本机保存目录：`PathPickInput` + `show-open-folder`
- 按钮：保存路径、开始下载、停止
- 进度：`a-progress` percent = `total>0 ? round(done/total*100) : 0`；文案用速度（可复用/抽 `formatSpeed` from `backupUi.ts`）
- 日志框：样式对齐 `BackupRunPanel` 的 `log-box`

- [ ] **Step 2: Wire open folder / pick local dir**

与任务表单相同的 `Dialogs.OpenFile` + `OpenInExplorer`；取消选择不报错。

- [ ] **Step 3: Manual check**

Run: `scripts\run-dev.bat`，打开单文件面板，保存路径 UI 可用（下载需真实 SSH）。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/SingleFileBackupPanel.vue frontend/src/components/SingleFileBackupPanel.ts frontend/src/utils/backupUi.ts
git commit -m "feat: add single-file backup panel UI"
```

---

### Task 6: App.vue 双 Tab + 互斥 UX

**Files:**
- Modify: `frontend/src/App.vue`
- Possibly: `frontend/src/components/BackupRunPanel.vue`（无需大改，由 App 传 `disabled`）

- [ ] **Step 1: Wrap body in a-tabs**

```vue
<a-tabs v-model:activeKey="mainTab" class="app-main-tabs">
  <a-tab-pane key="multi" tab="多文件备份" :disabled="singleRunning">
    <BackupRunPanel ... />
  </a-tab-pane>
  <a-tab-pane key="single" tab="单文件备份" :disabled="multiRunning">
    <SingleFileBackupPanel :ssh-config="config" :disabled-by-other-job="multiRunning" />
  </a-tab-pane>
</a-tabs>
```

`multiRunning` / `singleRunning` 来自两 composable 的 `status.running`。  
设置按钮仍全局打开 `BackupSettingsDialog`（SSH/邮件/多文件远程应用）。

- [ ] **Step 2: Style tabs**

保证 `app-body` 内 Tab 内容区 `flex:1; min-height:0`，多文件面板仍可撑满高度。

- [ ] **Step 3: Verify**

- 切 Tab 正常；多文件 UI 仍可用
- 一边 running 时另一 Tab `disabled` 或开始按钮 disabled

- [ ] **Step 4: Commit**

```bash
git add frontend/src/App.vue frontend/src/style.css
git commit -m "feat: split homepage into multi and single-file tabs"
```

---

### Task 7: 联调验收 + bindings 收尾

**Files:**
- 按需微调文案/错误提示

- [ ] **Step 1: `wails3 generate bindings -ts` + `go test ./...`**

Expected: all pass

- [ ] **Step 2: 对照验收标准**

1. Tab 切换；多文件回归  
2. 无 zipbak-srv 可下载  
3. 进度百分比+速度  
4. 本机 basename 正确  
5. 邮件完成/异常；取消不发  
6. 互斥

- [ ] **Step 3: Final commit if needed**

```bash
git add -A
git commit -m "chore: finalize single-file backup tab"
```

---

## Spec coverage

| Spec 项 | Task |
|---------|------|
| 双 Tab | 6 |
| 直接 SFTP 下载 | 3 |
| 独立 remote_file/local_dir | 1, 3, 5 |
| 共享 SSH/邮件 | 3, 6 |
| 字节进度 | 3, 5 |
| 取消不发邮件 | 3 |
| 互斥 | 2, 6 |
| 远程选文件增强 | 不做（首版手输，符合 spec「可选」） |

## Placeholder scan

无 TBD/TODO 占位；远程选文件明确延后。
