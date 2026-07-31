# Linux 多文件 zipbak-srv Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 同时构建 Windows/Linux(amd64) 两套 `zipbak-srv`；多文件备份按服务器 `os_type` 适配程序名、路径与 SSH 调用，使 Linux 源机可端到端使用多文件备份（二进制仍手动放置）。

**Architecture:** 共用现有 `zipbak` / `zipbak-srv` 源码；`BackupConfig.Resolved` 与远程路径规范化、`remote.RunRemote` 按 `OSType` 分支；放开 Linux 多文件 UI/校验；构建与 CI 增加 linux/amd64 产物。

**Tech Stack:** Go、Taskfile、现有 SSH/SFTP 流水线、Vue 服务器管理表单。

## Global Constraints

- 方案：同一套流水线按 `os_type` 分支（见规格）
- Linux 产物：`GOOS=linux GOARCH=amd64` → `dist/zipbak-srv`（无后缀）
- Windows 产物：保持 `dist/zipbak-srv.exe`
- 不自动上传二进制、不自动 `chmod +x`
- 不做 Linux arm64
- 不改变单文件备份语义
- Vue 逻辑放同名 `.ts`（用户规则）

## File Structure

| File | Responsibility |
|------|----------------|
| `backend/util/path.go` | `JoinRemoteForOS`、`QuoteShellArg`；路径/引号按 OS |
| `backend/model/backup_config.go` | `Resolved()` 按 OSType 拼 srv/state/staging |
| `backend/remote/ssh.go` | `RunRemote` Windows vs Linux 拼命令 |
| `backend/zipbak/pipeline.go` 等 | 远程路径改用 `NormalizeRemotePathForOS(..., cfg.OSType)` |
| `backend/service/*.go` | init/reset/status/oversized 同上 |
| `backend/service/config_validate.go` | 去掉 Linux 禁多文件 |
| `frontend/src/types/backup.ts` | `remotePathsFromAppDir(appDir, osType)` |
| `frontend/src/components/ServerManageDialog.*` | 放开 Linux 多文件勾选与文案 |
| `Taskfile.yml` / scripts / `.github/workflows/release-all.yml` | Linux 构建与发布 |

---

### Task 1: 路径 Join / Resolved 按 OSType

**Files:**
- Modify: `backend/util/path.go`
- Modify: `backend/model/backup_config.go`
- Test: `backend/util/path_test.go`（新建）
- Test: `backend/model/backup_config_resolve_test.go`（新建，或并入已有 model 测试）

**Interfaces:**
- Produces:
  - `util.JoinRemoteForOS(osType, base string, parts ...string) string`
  - `util.QuoteShellArg(s string) string`（可本 Task 先写测试占位，或 Task 2 再加；**本 Task 只做 Join + Resolved**）
  - `BackupConfig.Resolved()` 使用 `NormalizeOSType(c.OSType)` + `JoinRemoteForOS`

- [ ] **Step 1: Write failing tests**

`backend/util/path_test.go`:

```go
package util

import "testing"

func TestJoinRemoteForOS(t *testing.T) {
	if got := JoinRemoteForOS("linux", "/opt/zipbak", "zipbak-srv"); got != "/opt/zipbak/zipbak-srv" {
		t.Fatalf("linux join srv: %q", got)
	}
	if got := JoinRemoteForOS("linux", "/opt/zipbak", "data", "state-t1.db"); got != "/opt/zipbak/data/state-t1.db" {
		t.Fatalf("linux join state: %q", got)
	}
	if got := JoinRemoteForOS("windows", `D:\Tools\zipbak`, "zipbak-srv.exe"); got != `D:\Tools\zipbak\zipbak-srv.exe` {
		t.Fatalf("windows join: %q", got)
	}
}
```

`backend/model/backup_config_resolve_test.go`:

```go
package model

import "testing"

func TestResolvedLinuxSrvName(t *testing.T) {
	cfg := BackupConfig{RemoteAppDir: "/opt/zipbak", OSType: OSTypeLinux, TaskID: "abc"}
	out := cfg.Resolved()
	if out.RemoteSrv != "/opt/zipbak/zipbak-srv" {
		t.Fatalf("RemoteSrv=%q", out.RemoteSrv)
	}
	if out.RemoteState != "/opt/zipbak/data/state-abc.db" {
		t.Fatalf("RemoteState=%q", out.RemoteState)
	}
	if out.RemoteStaging != "/opt/zipbak/staging-abc" {
		t.Fatalf("RemoteStaging=%q", out.RemoteStaging)
	}
}

func TestResolvedWindowsSrvName(t *testing.T) {
	cfg := BackupConfig{RemoteAppDir: `D:\Tools\zipbak`, OSType: OSTypeWindows, TaskID: "abc"}
	out := cfg.Resolved()
	if out.RemoteSrv != `D:\Tools\zipbak\zipbak-srv.exe` {
		t.Fatalf("RemoteSrv=%q", out.RemoteSrv)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./backend/util/ ./backend/model/ -run "TestJoinRemoteForOS|TestResolved" -v`

- [ ] **Step 3: Implement**

`JoinRemoteForOS`：先 `NormalizeRemotePathForOS(base, osType)`，再按 OS 用 `/` 或 `\` 拼接 parts（不要用宿主机 `filepath.Join` 决定远程分隔符）。

`Resolved()`：

```go
osType := NormalizeOSType(c.OSType)
base := util.NormalizeRemotePathForOS(app, osType)
exe := "zipbak-srv.exe"
if IsLinuxOS(osType) {
	exe = "zipbak-srv"
}
out.RemoteSrv = util.JoinRemoteForOS(osType, base, exe)
out.RemoteState = util.JoinRemoteForOS(osType, base, "data", "state"+suffix+".db")
out.RemoteStaging = util.JoinRemoteForOS(osType, base, "staging"+suffix)
```

保留旧 `JoinRemote` 作为 Windows 默认包装（调用 `JoinRemoteForOS("", ...)`）以免无关调用方立刻全崩；新逻辑一律走 `JoinRemoteForOS`。

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./backend/util/ ./backend/model/ -run "TestJoinRemoteForOS|TestResolved" -v`

- [ ] **Step 5: Commit**

```bash
git add backend/util/path.go backend/util/path_test.go backend/model/backup_config.go backend/model/backup_config_resolve_test.go
git commit -m "feat: resolve zipbak remote paths by server os_type"
```

---

### Task 2: SSH RunRemote 与流水线路径规范化

**Files:**
- Modify: `backend/util/path.go`（`QuoteShellArg`）
- Modify: `backend/remote/ssh.go`
- Modify: `backend/zipbak/pipeline.go`
- Modify: `backend/service/backup_service.go`（InitRemote/Reset 等）
- Modify: `backend/service/remote_status.go`
- Modify: `backend/service/oversized_warnings.go`
- Test: `backend/util/path_test.go`（追加 QuoteShellArg）
- Test: `backend/remote/ssh_cmd_test.go`（纯函数抽离拼命令更好测）

**Interfaces:**
- Produces: `util.QuoteShellArg(s string) string`
- Produces: `remote.buildRemoteCommand(cfg model.BackupConfig, argv ...string) string`（包内或同文件未导出函数，供测试）
- Consumes: Task 1 的 `JoinRemoteForOS` / `Resolved`

- [ ] **Step 1: Failing tests for QuoteShellArg + command build**

```go
func TestQuoteShellArg(t *testing.T) {
	if got := QuoteShellArg(`hello`); got != `hello` {
		t.Fatalf("%q", got)
	}
	if got := QuoteShellArg(`a b`); got != `'a b'` {
		t.Fatalf("%q", got)
	}
	if got := QuoteShellArg(`a'b`); !strings.Contains(got, `'"'"'`) && got != `'a'"'"'b'` {
		// 标准 POSIX：用单引号包裹，内部 ' 拆成 '\'' 
		t.Fatalf("%q", got)
	}
}
```

拼命令测试（可把拼装抽到 `remote/cmd.go` 的 `BuildRemoteCommand(osType, srv string, argv ...string) string`）：

```go
func TestBuildRemoteCommandLinux(t *testing.T) {
	cmd := BuildRemoteCommand("linux", "/opt/zipbak/zipbak-srv", "status", "--state", "/opt/zipbak/data/state.db")
	if !strings.Contains(cmd, "/opt/zipbak/zipbak-srv") || !strings.Contains(cmd, "status") {
		t.Fatalf("%q", cmd)
	}
	if strings.Contains(cmd, `"`) { // Linux 用单引号风格，不应是 Windows 双引号为主
		// 允许路径无空格时无引号；有空格则单引号
	}
}
```

实现约定 `QuoteShellArg`：无特殊字符原样返回；否则用单引号包裹，内部 `'` → `'\''`。

- [ ] **Step 2: Run FAIL**

- [ ] **Step 3: Implement RunRemote + 全链路 NormalizeRemotePathForOS**

`RunRemoteWithStderr`:

```go
osType := model.NormalizeOSType(c.cfg.OSType)
srv := util.NormalizeRemotePathForOS(c.cfg.RemoteSrv, osType)
quote := util.QuoteWindowsArg
if model.IsLinuxOS(osType) {
	quote = util.QuoteShellArg
}
parts := []string{quote(srv)}
for _, a := range argv {
	parts = append(parts, quote(a))
}
cmd := strings.Join(parts, " ")
```

凡 `util.NormalizeRemotePath(x)` 且 `x` 来自远程作业配置处，改为 `NormalizeRemotePathForOS(x, cfg.OSType)`（pipeline / backup_service init&reset / remote_status / oversized_warnings）。SFTP 本地路径不受影响。

可选：在远程命令失败且 Linux 时，`appendLog` 追加一句提示「请确认已放置 zipbak-srv 并 chmod +x」（仅 init 失败或首次 pack 失败处一处即可，避免刷屏）。

- [ ] **Step 4: `go test ./backend/... -count=1` PASS**

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: OS-aware SSH remote command and path normalization"
```

---

### Task 3: 放开 Linux 多文件校验与 UI

**Files:**
- Modify: `backend/service/config_validate.go`（删除 Linux + SupportMultiFile 错误）
- Modify: `frontend/src/types/backup.ts` — `remotePathsFromAppDir(appDir, osType?)`
- Modify: `frontend/src/components/ServerManageDialog.ts` / `.vue`

**Interfaces:**
- Produces: `remotePathsFromAppDir(appDir: string, osType?: string)` 返回 linux/windows 风格推导路径
- UI：Linux 可勾选多文件；去掉「仅支持 Windows」提示与 `disabled`/`watch` 强制取消

- [ ] **Step 1: Update `remotePathsFromAppDir`**

```ts
export function remotePathsFromAppDir(appDir: string, osType?: string) {
  const linux = normalizeServerOSType(osType) === "linux";
  if (linux) {
    const base = appDir.trim().replace(/\\/g, "/").replace(/\/+$/, "") || "";
    if (!base) return { srv: "", state: "", staging: "" };
    return {
      srv: `${base}/zipbak-srv`,
      state: `${base}/data/state-{任务ID}.db`,
      staging: `${base}/staging-{任务ID}`,
    };
  }
  const base = appDir.trim().replace(/\//g, "\\").replace(/\\+$/, "");
  if (!base) return { srv: "", state: "", staging: "" };
  return {
    srv: `${base}\\zipbak-srv.exe`,
    state: `${base}\\data\\state-{任务ID}.db`,
    staging: `${base}\\staging-{任务ID}`,
  };
}
```

- [ ] **Step 2: ServerManageDialog**

- 去掉 `isLinux` 时 checkbox `disabled`
- 去掉 `watch(os_type)` 里强制清空多文件
- 去掉校验「多文件备份仅支持 Windows 服务器」
- `derivedPaths = remotePathsFromAppDir(form.remote_app_dir, form.os_type)`
- 文案：Linux 也可「勾选后需配置远程应用目录」；占位符已按 OS 切换则保持

- [ ] **Step 3: 后端 `normalizeServer` 删除 Linux 禁多文件分支**

- [ ] **Step 4: `npx vue-tsc --noEmit` + `go test ./backend/service/ -count=1`**

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: allow multi-file backup on Linux servers"
```

---

### Task 4: 构建与发布 Linux zipbak-srv

**Files:**
- Modify: `Taskfile.yml`
- Modify: `scripts/build-zipbak-srv.bat` / `.sh`
- Modify: `scripts/build-release.bat` / `.sh`
- Modify: `.github/workflows/release-all.yml`

**Interfaces:**
- Produces: `wails3 task build:zipbak-srv` 构建 Windows；`build:zipbak-srv-linux` 构建 Linux；或 `build:zipbak-srv-all` 两者都构建

- [ ] **Step 1: Taskfile**

```yaml
  build:zipbak-srv:
    summary: Builds zipbak-srv.exe for remote Windows (GOOS=windows/amd64)
    cmds:
      - '{{if eq OS "windows"}}cmd /c if not exist {{.BIN_DIR}} mkdir {{.BIN_DIR}}{{else}}mkdir -p {{.BIN_DIR}}{{end}}'
      - GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "{{.BIN_DIR}}/zipbak-srv.exe" ./cmd/zipbak-srv

  build:zipbak-srv-linux:
    summary: Builds zipbak-srv for remote Linux (GOOS=linux/amd64)
    cmds:
      - '{{if eq OS "windows"}}cmd /c if not exist {{.BIN_DIR}} mkdir {{.BIN_DIR}}{{else}}mkdir -p {{.BIN_DIR}}{{end}}'
      - GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "{{.BIN_DIR}}/zipbak-srv" ./cmd/zipbak-srv

  build:zipbak-srv-all:
    cmds:
      - task: build:zipbak-srv
      - task: build:zipbak-srv-linux
```

- [ ] **Step 2: Scripts / CI**

- `build-zipbak-srv.*`：改为调用 `build:zipbak-srv-all`，echo 两套输出路径
- `build-release.*`：同样打两套；提示 Linux 拷贝 `zipbak-srv`
- `release-all.yml`：windows job 中 `wails3 task build:zipbak-srv-all`，并 `cp dist/zipbak-srv` 到 `release/`（与 exe 并列）；缺文件则 fail

- [ ] **Step 3: 本地验证构建**

Run: `wails3 task build:zipbak-srv-linux`  
Expected: `dist/zipbak-srv` 存在（Windows 上交叉编译应成功）

- [ ] **Step 4: Commit**

```bash
git commit -m "build: add linux amd64 zipbak-srv artifact"
```

---

### Task 5: 验收核对

**Files:** 按需小修

- [ ] **Step 1:** `go test ./... -count=1`
- [ ] **Step 2:** `cd frontend && npx vue-tsc --noEmit`
- [ ] **Step 3:** 对照规格验收 1–5（代码级：Resolved/RunRemote/UI/Taskfile；真实 Linux SFTP 可选手测）
- [ ] **Step 4:** 若有 polish，commit `fix: linux multifile zipbak integration polish`

---

## Spec coverage

| 规格项 | Task |
|--------|------|
| Resolved 程序名与路径分隔符 | 1 |
| NormalizeRemotePathForOS 全链路 | 2 |
| RunRemote Linux 引号 | 2 |
| 放开 Linux 多文件 UI/校验 | 3 |
| 双产物构建与 CI | 4 |
| 验收 | 5 |
| 不自动上传 / 无 arm64 | 全局约束，不做 |
