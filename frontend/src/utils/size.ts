/** Human-readable file size (binary SI), aligned with backend util.FormatBytes. */
export function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n < 0) n = 0;
  const unit = 1024;
  if (n < unit) return `${n} B`;
  let div = unit;
  let exp = 0;
  while (n / div >= unit && exp < 4) {
    div *= unit;
    exp++;
  }
  const val = n / div;
  const suffix = ["KiB", "MiB", "GiB", "TiB"][exp];
  return `${val.toFixed(2)} ${suffix}`;
}
