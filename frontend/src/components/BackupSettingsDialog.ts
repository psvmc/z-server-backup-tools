import { message } from "ant-design-vue";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import type { BackupConfig } from "../types/backup";
import { formatError } from "../types/update";

export type BackupSettingsDialogProps = {
  configPath: string;
  saving?: boolean;
};

export type BackupSettingsDialogEmit = {
  (e: "saveNotify", cfg: BackupConfig): void;
};

export function useBackupSettingsDialog(
  config: { value: BackupConfig },
  props: BackupSettingsDialogProps,
  emit: BackupSettingsDialogEmit,
) {
  function onSaveNotify() {
    if (props.saving) return;
    emit("saveNotify", { ...config.value });
  }

  async function openConfigFolder() {
    const path = props.configPath?.trim();
    if (!path) return;
    try {
      await BackupService.OpenInExplorer(path);
    } catch (err) {
      message.error(formatError(err));
    }
  }

  return {
    onSaveNotify,
    openConfigFolder,
  };
}
