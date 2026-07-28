import { message } from "ant-design-vue";
import { UpdateService } from "../../bindings/z-server-backup-tools/backend/service";
import type { UpdateCheckResult } from "../types/update";
import { formatError } from "../types/update";
import { useUpdateConfirm } from "./useUpdateConfirm";
import { useUpdateProgress } from "./useUpdateProgress";

function toUpdateResult(raw: unknown): UpdateCheckResult {
  const item = (raw ?? {}) as Partial<UpdateCheckResult>;
  return {
    available: Boolean(item.available),
    enabled: Boolean(item.enabled),
    currentVersion: item.currentVersion ?? "",
    latestVersion: item.latestVersion ?? "",
    releaseName: item.releaseName ?? "",
    notes: item.notes ?? "",
    releaseURL: item.releaseURL ?? "",
  };
}

export function useAppUpdate() {
  const { prompt: promptUpdateConfirm } = useUpdateConfirm();
  const { runWithProgress } = useUpdateProgress();

  const promptAndApply = async (result: UpdateCheckResult) => {
    const choice = await promptUpdateConfirm(result);
    if (choice === "skip") {
      const version = result.latestVersion?.trim();
      if (!version) return;
      try {
        await UpdateService.SkipUpdateVersion(version);
        message.info(`已跳过 v${version}`);
      } catch (err) {
        message.error(formatError(err));
      }
      return;
    }
    if (choice !== "confirm") return;
    try {
      await runWithProgress(() => UpdateService.ApplyUpdate());
    } catch (err) {
      message.error(formatError(err));
    }
  };

  const handleCheckResult = async (result: UpdateCheckResult, manual: boolean) => {
    if (!result.enabled) {
      if (manual) message.info("当前为开发版本，不支持自动更新");
      return;
    }
    if (!result.available) {
      if (manual) message.success(`已是最新版本（v${result.currentVersion}）`);
      return;
    }
    await promptAndApply(result);
  };

  const checkOnStartup = async () => {
    try {
      const result = toUpdateResult(await UpdateService.CheckForUpdate());
      await handleCheckResult(result, false);
    } catch (err) {
      console.warn("检查更新失败:", err);
    }
  };

  const checkForUpdate = async () => {
    const hide = message.loading("正在检查更新...", 0);
    try {
      const result = toUpdateResult(await UpdateService.CheckForUpdate());
      hide();
      await handleCheckResult(result, true);
    } catch (err) {
      hide();
      message.error(formatError(err));
    }
  };

  const loadCurrentVersion = () => UpdateService.GetCurrentVersion();

  return { checkOnStartup, checkForUpdate, loadCurrentVersion };
}
