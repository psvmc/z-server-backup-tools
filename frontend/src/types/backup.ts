export interface BackupConfig {
  host: string;
  port: number;
  user: string;
  password: string;
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

export interface BackupTask {
  id: string;
  name?: string;
  remote_source: string;
  local_dir: string;
  part_name_prefix?: string;
}

export interface SingleFileConfig {
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

export function remotePathsFromAppDir(appDir: string) {
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
