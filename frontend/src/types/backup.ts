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
  known_hosts_file?: string;
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
    state: `${base}\\data\\state.db`,
    staging: `${base}\\staging`,
  };
}
