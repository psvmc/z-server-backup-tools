export interface UpdateCheckResult {
  available: boolean;
  enabled: boolean;
  currentVersion: string;
  latestVersion: string;
  releaseName: string;
  notes: string;
  releaseURL: string;
}

export function formatError(err: unknown): string {
  if (err instanceof Error) return err.message;
  if (typeof err === "string") return err;
  return "操作失败";
}
