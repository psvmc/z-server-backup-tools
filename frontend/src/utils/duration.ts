export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) {
    return "--:--";
  }
  const totalSec = Math.floor(ms / 1000);
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  if (h > 0) {
    return `${h}:${pad(m)}:${pad(s)}`;
  }
  return `${m}:${pad(s)}`;
}

/** 根据当前进度推算整任务总耗时（毫秒）。 */
export function estimateTotalMs(elapsedMs: number, doneCount: number, totalCount: number): number | null {
  if (doneCount <= 0 || totalCount <= 0 || elapsedMs <= 0) {
    return null;
  }
  return (elapsedMs / doneCount) * totalCount;
}

/** 由总耗时估计值减去已用时间得到剩余时间；进度未更新时随时钟递减。 */
export function estimateRemainingMs(
  elapsedMs: number,
  doneCount: number,
  totalCount: number,
): number | null {
  const totalEst = estimateTotalMs(elapsedMs, doneCount, totalCount);
  if (totalEst == null) {
    return null;
  }
  return Math.max(0, totalEst - elapsedMs);
}
