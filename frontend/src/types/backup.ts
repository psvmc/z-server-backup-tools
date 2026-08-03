export interface BackupConfig {
  host: string;
  port: number;
  user: string;
  password: string;
  os_type?: string;
  remote_app_dir: string;
  remote_srv?: string;
  remote_state?: string;
  remote_source: string;
  remote_staging?: string;
  local_dir: string;
  max_part_gb: number;
  part_name_prefix?: string;
  task_id?: string;
  notify_email?: string;
  smtp_host?: string;
  smtp_port?: number;
  smtp_user?: string;
  smtp_password?: string;
  known_hosts_file?: string;
}

export interface Server {
  id: string;
  name: string;
  host: string;
  port: number;
  user: string;
  password: string;
  os_type: string;
  support_multi_file: boolean;
  remote_app_dir?: string;
  max_part_gb?: number;
}

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

export function isMultiTask(task: Pick<BackupTask, "kind">): boolean {
  return taskKind(task) === "multi";
}

export function isSingleTask(task: Pick<BackupTask, "kind">): boolean {
  return taskKind(task) === "single";
}

export interface SingleFileConfig {
  server_id?: string;
  remote_file: string;
  local_dir: string;
}

export interface JobStatus {
  running: boolean;
  phase: string;
  currentPart: string;
  localFile: string;
  totalFiles: number;
  packedFiles: number;
  done: boolean;
  lastError: string;
  remoteInited: boolean;
  pendingZip?: string;
  prefetchZip?: string;
  remoteHint?: string;
  maxFileBytes?: number;
  oversizedFileCount?: number;
  timingStartedAtMs?: number;
  timingPackedFilesAtStart?: number;
  timingEstimatedTotalMs?: number;
  downloadBytesDone?: number;
  downloadBytesTotal?: number;
  downloadSpeedBps?: number;
}

export type LocalPartState = "downloaded" | "downloading";

export interface LocalPartFile {
  name: string;
  path: string;
  sizeBytes: number;
  state: LocalPartState;
}

export interface LocalPartListing {
  localDir: string;
  files: LocalPartFile[];
}

export function normalizeServerOSType(osType?: string): "windows" | "linux" {
  return (osType ?? "").trim().toLowerCase() === "linux" ? "linux" : "windows";
}

export function emptyNotifyConfig(): BackupConfig {
  return {
    host: "",
    port: 22,
    user: "",
    password: "",
    remote_app_dir: "",
    remote_source: "",
    local_dir: "",
    max_part_gb: 0,
    smtp_host: "smtp.qq.com",
    smtp_port: 465,
  };
}

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
  if (!base) {
    return { srv: "", state: "", staging: "" };
  }
  return {
    srv: `${base}\\zipbak-srv.exe`,
    state: `${base}\\data\\state-{任务ID}.db`,
    staging: `${base}\\staging-{任务ID}`,
  };
}

export function applyServer(base: BackupConfig, server?: Server | null): BackupConfig {
  if (!server) {
    return {
      ...base,
      host: "",
      user: "",
      password: "",
      os_type: undefined,
      remote_app_dir: "",
      max_part_gb: 0,
    };
  }
  return {
    ...base,
    host: server.host,
    port: server.port || 22,
    user: server.user,
    password: server.password,
    os_type: normalizeServerOSType(server.os_type),
    remote_app_dir: server.remote_app_dir ?? "",
    max_part_gb: server.max_part_gb || 2,
  };
}

export function mergeTaskConfig(base: BackupConfig, task?: BackupTask | null): BackupConfig {
  if (!task) {
    return { ...base, remote_source: "", local_dir: "", part_name_prefix: "", task_id: "" };
  }
  return {
    ...base,
    task_id: task.id,
    remote_source: task.remote_source ?? "",
    local_dir: task.local_dir ?? "",
    part_name_prefix: task.part_name_prefix ?? "",
  };
}

export function mergeJobConfig(
  notify: BackupConfig,
  server: Server | null | undefined,
  task?: BackupTask | null,
): BackupConfig {
  return mergeTaskConfig(applyServer(notify, server), task);
}

export function taskDisplayName(task: BackupTask): string {
  const name = task.name?.trim();
  if (name) return name;
  const src = task.remote_source?.trim();
  if (src) return src;
  return task.id;
}

export function newTaskId(): string {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID().replace(/-/g, "").slice(0, 12);
  }
  return `task-${Date.now().toString(36)}`;
}
