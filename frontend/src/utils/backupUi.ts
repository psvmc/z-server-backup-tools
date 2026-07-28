import type { JobStatus } from "../types/backup";

export function formatSpeed(bps: number): string {
  if (!Number.isFinite(bps) || bps <= 0) {
    return "";
  }
  const unit = 1024;
  if (bps < unit) {
    return `${Math.round(bps)} B/s`;
  }
  let div = unit;
  let exp = 0;
  while (bps / div >= unit && exp < 4) {
    div *= unit;
    exp++;
  }
  const val = bps / div;
  const suffix = ["KiB/s", "MiB/s", "GiB/s", "TiB/s"][exp];
  return `${val.toFixed(2)} ${suffix}`;
}

export function downloadPhaseLabel(status: JobStatus): string {
  if (status.phase !== "download") {
    return "";
  }
  const speed = formatSpeed(status.downloadSpeedBps ?? 0);
  if (speed) {
    return `下载 · ${speed}`;
  }
  const done = status.downloadBytesDone ?? 0;
  const total = status.downloadBytesTotal ?? 0;
  if (total > 0 && done > 0) {
    const pct = Math.min(100, Math.round((done / total) * 100));
    return `下载 · ${pct}%`;
  }
  return "下载";
}
