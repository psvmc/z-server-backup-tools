import { computed, ref, watch, type PropType, type Ref } from "vue";
import { message } from "ant-design-vue";
import { Dialogs } from "@wailsio/runtime";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import type { BackupConfig, BackupTask, BackupTaskKind, Server } from "../types/backup";
import { applyServer, emptyNotifyConfig, newTaskId } from "../types/backup";
import { formatError } from "../types/update";
import { validateIgnorePatterns } from "../utils/ignorePatterns";
import type { RemotePickerMode } from "./RemoteDirPickerModal";

export const backupTaskFormModalProps = {
  connection: {
    type: Object as PropType<BackupConfig>,
    required: true as const,
  },
  servers: {
    type: Array as PropType<Server[]>,
    default: () => [],
  },
  task: {
    type: Object as PropType<BackupTask | null>,
    default: null,
  },
  kind: {
    type: String as PropType<BackupTaskKind>,
    default: "multi" as const,
  },
};

export type BackupTaskFormModalProps = {
  connection: BackupConfig;
  servers?: Server[];
  task?: BackupTask | null;
  kind?: BackupTaskKind;
};

export type BackupTaskFormModalEmit = {
  (e: "saved"): void;
};

function normalizeIgnorePatterns(patterns?: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of patterns ?? []) {
    const p = raw.trim();
    if (!p || seen.has(p)) continue;
    seen.add(p);
    out.push(p);
  }
  return out;
}

function isDialogCancelled(err: unknown): boolean {
  const text = formatError(err).toLowerCase();
  return (
    text.includes("cancel") ||
    text.includes("cancelled") ||
    text.includes("canceled") ||
    text.includes("abort") ||
    text.includes("取消")
  );
}

export function useBackupTaskFormModal(
  open: Ref<boolean>,
  props: BackupTaskFormModalProps,
  emit: BackupTaskFormModalEmit,
) {
  const form = ref<BackupTask>({
    id: "",
    name: "",
    server_id: "",
    remote_source: "",
    local_dir: "",
    part_name_prefix: "",
    ignore_patterns: [],
  });

  const remotePickerOpen = ref(false);
  const saving = ref(false);

  const isEdit = computed(() => !!props.task?.id);
  const isSingle = computed(() => props.kind === "single");
  const isFolderZip = computed(() => props.kind === "folder_zip");
  const isMulti = computed(() => !isSingle.value && !isFolderZip.value);
  const isDirectBackup = computed(() => isSingle.value || isFolderZip.value);

  const eligibleServers = computed(() => {
    const list = props.servers ?? [];
    return isDirectBackup.value ? list : list.filter((s) => s.support_multi_file);
  });

  const serverOptions = computed(() =>
    eligibleServers.value.map((s) => ({
      value: s.id,
      label: s.name?.trim() || s.host || s.id,
    })),
  );

  const selectedServer = computed(
    () => eligibleServers.value.find((s) => s.id === form.value.server_id) ?? null,
  );

  const connectionForPicker = computed(() =>
    applyServer(props.connection ?? emptyNotifyConfig(), selectedServer.value),
  );

  const remotePickerMode = computed<RemotePickerMode>(() =>
    isSingle.value ? "file" : "directory",
  );

  const remoteSourceLabel = computed(() => {
    if (isSingle.value) return "远程源文件";
    if (isFolderZip.value) return "远程文件夹";
    return "远程源目录 (--dir)";
  });

  const remotePickerTitle = computed(() => {
    if (isSingle.value) return "选择远程源文件";
    if (isFolderZip.value) return "选择远程文件夹";
    return "选择远程源目录";
  });

  const serverPlaceholder = computed(() =>
    isDirectBackup.value ? "选择服务器" : "选择支持多文件备份的服务器",
  );

  watch(
    () => [open.value, props.task] as const,
    ([visible, task]) => {
      if (!visible) return;
      if (task) {
        form.value = {
          id: task.id,
          name: task.name ?? "",
          server_id: task.server_id ?? "",
          remote_source: task.remote_source ?? "",
          local_dir: task.local_dir ?? "",
          part_name_prefix: task.part_name_prefix ?? "",
          ignore_patterns: [...(task.ignore_patterns ?? [])],
        };
      } else {
        form.value = {
          id: "",
          name: "",
          server_id: "",
          remote_source: "",
          local_dir: "",
          part_name_prefix: "",
          ignore_patterns: [],
        };
      }
    },
    { immediate: true },
  );

  function ensureServerSelected(): boolean {
    if (!form.value.server_id?.trim()) {
      message.warning("请选择服务器");
      return false;
    }
    if (!selectedServer.value) {
      if (isDirectBackup.value) {
        message.warning("所选服务器不可用");
      } else {
        message.warning("所选服务器不可用或不支持多文件备份");
      }
      return false;
    }
    return true;
  }

  function openRemotePicker() {
    if (!ensureServerSelected()) return;
    remotePickerOpen.value = true;
  }

  async function pickLocalDir() {
    try {
      const picked = await Dialogs.OpenFile({
        Title: "选择本机保存目录",
        CanChooseDirectories: true,
        CanChooseFiles: false,
      });
      const path = Array.isArray(picked) ? picked[0] : picked;
      if (path && typeof path === "string") {
        form.value.local_dir = path;
      }
    } catch (err) {
      if (isDialogCancelled(err)) return;
      message.error(formatError(err));
    }
  }

  async function openFolder(path: string) {
    const target = path?.trim();
    if (!target) {
      message.warning("请先填写路径");
      return;
    }
    try {
      await BackupService.OpenInExplorer(target);
    } catch (err) {
      message.error(formatError(err));
    }
  }

  async function onSubmit() {
    if (!ensureServerSelected()) return;
    if (!form.value.remote_source?.trim()) {
      const hint = isSingle.value ? "远程源文件" : isFolderZip.value ? "远程文件夹" : "远程源目录";
      message.warning(`请填写${hint}`);
      return;
    }
    if (!form.value.local_dir?.trim()) {
      message.warning("请填写本机保存目录");
      return;
    }
    const ignorePatterns = isFolderZip.value
      ? normalizeIgnorePatterns(form.value.ignore_patterns)
      : [];
    if (isFolderZip.value) {
      const invalid = validateIgnorePatterns(ignorePatterns);
      if (invalid) {
        message.warning(invalid);
        return;
      }
    }
    saving.value = true;
    try {
      const current = (await BackupService.GetTasks()) as BackupTask[];
      const payload: BackupTask = {
        id: isEdit.value ? form.value.id : newTaskId(),
        kind: props.kind ?? "multi",
        name: form.value.name?.trim() || undefined,
        server_id: form.value.server_id!.trim(),
        remote_source: form.value.remote_source.trim(),
        local_dir: form.value.local_dir.trim(),
        part_name_prefix: isMulti.value
          ? form.value.part_name_prefix?.trim() || undefined
          : undefined,
        ignore_patterns: isFolderZip.value && ignorePatterns.length > 0 ? ignorePatterns : undefined,
      };
      let next: BackupTask[];
      if (isEdit.value) {
        next = current.map((t) => (t.id === payload.id ? payload : t));
      } else {
        next = [...current, payload];
      }
      await BackupService.SaveTasks(next);
      if (!isEdit.value) {
        if (isSingle.value) {
          await BackupService.SetActiveSingleFileTaskID(payload.id);
        } else if (isFolderZip.value) {
          await BackupService.SetActiveFolderZipTaskID(payload.id);
        } else {
          await BackupService.SetActiveTaskID(payload.id);
        }
      }
      message.success(isEdit.value ? "任务已更新" : "任务已添加");
      open.value = false;
      emit("saved");
    } catch (err) {
      message.error(formatError(err));
    } finally {
      saving.value = false;
    }
  }

  return {
    form,
    remotePickerOpen,
    saving,
    isEdit,
    isSingle,
    isFolderZip,
    isMulti,
    ignorePatterns: computed({
      get: () => form.value.ignore_patterns ?? [],
      set: (v: string[]) => {
        form.value.ignore_patterns = v;
      },
    }),
    serverOptions,
    connectionForPicker,
    remotePickerMode,
    remoteSourceLabel,
    remotePickerTitle,
    serverPlaceholder,
    openRemotePicker,
    pickLocalDir,
    openFolder,
    onSubmit,
  };
}
