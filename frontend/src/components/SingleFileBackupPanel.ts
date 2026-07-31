import { computed, nextTick, ref, watch, type PropType } from "vue";
import { message } from "ant-design-vue";
import { Dialogs } from "@wailsio/runtime";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import type { BackupConfig, Server } from "../types/backup";
import { applyServer, emptyNotifyConfig } from "../types/backup";
import { formatError } from "../types/update";
import { formatSpeed } from "../utils/backupUi";
import { formatBytes } from "../utils/size";
import { useSingleFileBackup } from "../composables/useSingleFileBackup";

export const singleFileBackupPanelProps = {
  sshConfig: {
    type: Object as PropType<BackupConfig>,
    required: true as const,
  },
  servers: {
    type: Array as PropType<Server[]>,
    default: () => [],
  },
  disabledByOtherJob: {
    type: Boolean,
    default: false,
  },
  panelActive: {
    type: Boolean,
    default: true,
  },
};

export type SingleFileBackupPanelProps = {
  sshConfig: BackupConfig;
  servers?: Server[];
  disabledByOtherJob?: boolean;
  panelActive?: boolean;
};

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

export function useSingleFileBackupPanel(props: SingleFileBackupPanelProps) {
  const panelActive = computed(() => props.panelActive !== false);
  const { paths, status, logs, saving, savePaths, start, stop } = useSingleFileBackup({
    panelActive,
  });

  const autoScrollLog = ref(false);
  const logBoxRef = ref<HTMLElement | null>(null);
  const prevLogLen = ref(0);

  const serverOptions = computed(() =>
    (props.servers ?? []).map((s) => ({
      value: s.id,
      label: s.name?.trim() || s.host || s.id,
    })),
  );

  const selectedServer = computed(
    () => (props.servers ?? []).find((s) => s.id === paths.value.server_id) ?? null,
  );

  const progressPercent = computed(() => {
    const total = status.value.downloadBytesTotal ?? 0;
    const done = status.value.downloadBytesDone ?? 0;
    if (total <= 0) return 0;
    return Math.min(100, Math.round((done / total) * 100));
  });

  const progressText = computed(() => {
    const speed = formatSpeed(status.value.downloadSpeedBps ?? 0);
    const done = status.value.downloadBytesDone ?? 0;
    const total = status.value.downloadBytesTotal ?? 0;
    const sizePart =
      total > 0 ? `${formatBytes(done)} / ${formatBytes(total)}` : done > 0 ? formatBytes(done) : "";
    if (speed && sizePart) return `${sizePart} · ${speed}`;
    if (speed) return speed;
    if (sizePart) return sizePart;
    return status.value.running ? "下载中…" : "";
  });

  const statusLabel = computed(() => {
    if (status.value.running) return "下载中";
    if (status.value.done) return "已完成";
    if (status.value.lastError) return "失败";
    return "就绪";
  });

  const statusTagColor = computed(() => {
    if (status.value.running) return "processing";
    if (status.value.done) return "success";
    if (status.value.lastError) return "error";
    return "default";
  });

  const logText = computed(() => logs.value.join("\n") || "等待任务输出…");

  const startDisabled = computed(
    () =>
      !!props.disabledByOtherJob ||
      status.value.running ||
      !paths.value.server_id?.trim() ||
      !paths.value.remote_file?.trim() ||
      !paths.value.local_dir?.trim(),
  );

  const stopDisabled = computed(() => !status.value.running);

  function scrollLogToBottom() {
    const el = logBoxRef.value;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }

  watch(
    () => logs.value.length,
    (len) => {
      if (!autoScrollLog.value) {
        prevLogLen.value = len;
        return;
      }
      if (len > prevLogLen.value) {
        void nextTick(scrollLogToBottom);
      }
      prevLogLen.value = len;
    },
  );

  watch(autoScrollLog, (on) => {
    if (on) {
      prevLogLen.value = logs.value.length;
      void nextTick(scrollLogToBottom);
    }
  });

  function onRemoteBrowse() {
    message.info("请手动输入远程文件完整路径");
  }

  async function pickLocalDir() {
    try {
      const picked = await Dialogs.OpenFile({
        Title: "选择本机保存目录",
        CanChooseDirectories: true,
        CanChooseFiles: false,
        Directory: paths.value.local_dir || undefined,
      });
      const path = Array.isArray(picked) ? picked[0] : picked;
      if (path && typeof path === "string") {
        paths.value.local_dir = path;
      }
    } catch (err) {
      if (isDialogCancelled(err)) return;
      message.error(formatError(err));
    }
  }

  async function openLocalFolder() {
    const target = paths.value.local_dir?.trim();
    if (!target) {
      message.warning("请先填写本机保存目录");
      return;
    }
    try {
      await BackupService.OpenInExplorer(target);
    } catch (err) {
      message.error(formatError(err));
    }
  }

  async function onSave() {
    if (!paths.value.server_id?.trim()) {
      message.warning("请选择服务器");
      return;
    }
    if (!paths.value.remote_file?.trim()) {
      message.warning("请填写远程源文件路径");
      return;
    }
    if (!paths.value.local_dir?.trim()) {
      message.warning("请填写本机保存目录");
      return;
    }
    await savePaths();
  }

  async function onStart() {
    if (props.disabledByOtherJob) {
      message.warning("已有多文件备份任务在运行");
      return;
    }
    if (!paths.value.server_id?.trim()) {
      message.warning("请选择服务器");
      return;
    }
    if (!paths.value.remote_file?.trim()) {
      message.warning("请填写远程源文件路径");
      return;
    }
    if (!paths.value.local_dir?.trim()) {
      message.warning("请填写本机保存目录");
      return;
    }
    // 后端以 store 的 server_id 为准；此处合并所选服务器 SSH 作为入参兜底
    const ssh = applyServer(props.sshConfig ?? emptyNotifyConfig(), selectedServer.value);
    await start(ssh);
  }

  function onStop() {
    stop();
  }

  function progressFormat(pct?: number) {
    const speed = formatSpeed(status.value.downloadSpeedBps ?? 0);
    if (speed) return speed;
    return `${pct ?? progressPercent.value}%`;
  }

  return {
    paths,
    status,
    logs,
    saving,
    autoScrollLog,
    logBoxRef,
    serverOptions,
    progressPercent,
    progressText,
    progressFormat,
    statusLabel,
    statusTagColor,
    logText,
    startDisabled,
    stopDisabled,
    onRemoteBrowse,
    pickLocalDir,
    openLocalFolder,
    onSave,
    onStart,
    onStop,
  };
}
