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

export function estimateRemainingMs(elapsedMs: number, doneCount: number, remainingCount: number): number | null {
  if (doneCount <= 0 || remainingCount <= 0 || elapsedMs <= 0) {
    return null;
  }
  return (elapsedMs / doneCount) * remainingCount;
}
