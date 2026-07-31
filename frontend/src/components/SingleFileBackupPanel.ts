import { computed, nextTick, onMounted, ref, watch, type PropType } from "vue";
import { message } from "ant-design-vue";
import { Dialogs } from "@wailsio/runtime";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import type { BackupConfig } from "../types/backup";
import { formatError } from "../types/update";
import { formatSpeed } from "../utils/backupUi";
import { formatBytes } from "../utils/size";
import { useSingleFileBackup } from "../composables/useSingleFileBackup";

export const singleFileBackupPanelProps = {
  sshConfig: {
    type: Object as PropType<BackupConfig>,
    required: true as const,
  },
  disabledByOtherJob: {
    type: Boolean,
    default: false,
  },
};

export type SingleFileBackupPanelProps = {
  sshConfig: BackupConfig;
  disabledByOtherJob?: boolean;
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
  const panelActive = ref(true);
  const { paths, status, logs, saving, load, savePaths, start, stop } = useSingleFileBackup({
    panelActive,
  });

  const autoScrollLog = ref(false);
  const logBoxRef = ref<HTMLElement | null>(null);
  const prevLogLen = ref(0);

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

  onMounted(() => {
    panelActive.value = true;
    void load().catch((err) => {
      console.warn("加载单文件配置失败:", err);
    });
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
    if (!paths.value.remote_file?.trim()) {
      message.warning("请填写远程源文件路径");
      return;
    }
    if (!paths.value.local_dir?.trim()) {
      message.warning("请填写本机保存目录");
      return;
    }
    await start(props.sshConfig);
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
