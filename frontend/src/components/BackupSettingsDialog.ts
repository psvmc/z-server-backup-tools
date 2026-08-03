import type { BackupConfig } from "../types/backup";

export type BackupSettingsDialogProps = {
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

  return {
    onSaveNotify,
  };
}
