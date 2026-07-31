import { computed, nextTick, ref, watch } from "vue";
import { message } from "ant-design-vue";
import type { BackupTask } from "../types/backup";
import { formatSpeed } from "../utils/backupUi";
import { formatBytes } from "../utils/size";
import { useSingleFileBackup } from "../composables/useSingleFileBackup";

export const singleFileBackupPanelProps = {
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
  disabledByOtherJob?: boolean;
  panelActive?: boolean;
};

export type SingleFileBackupPanelEmit = {
  (e: "addTask"): void;
  (e: "editTask", task: BackupTask): void;
};

export function useSingleFileBackupPanel(
  props: SingleFileBackupPanelProps,
  emit: SingleFileBackupPanelEmit,
) {
  const panelActive = computed(() => props.panelActive !== false);
  const { tasks, activeTaskId, status, logs, selectTask, removeTask, start, stop } =
    useSingleFileBackup({
      panelActive,
    });

  const autoScrollLog = ref(false);
  const logBoxRef = ref<HTMLElement | null>(null);
  const prevLogLen = ref(0);

  const activeTask = computed(
    () => tasks.value.find((t) => t.id === activeTaskId.value) ?? null,
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
      !activeTaskId.value?.trim() ||
      !activeTask.value?.server_id?.trim() ||
      !activeTask.value?.remote_source?.trim() ||
      !activeTask.value?.local_dir?.trim(),
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

  async function onStart() {
    if (props.disabledByOtherJob) {
      message.warning("已有多文件备份任务在运行");
      return;
    }
    if (!activeTaskId.value?.trim()) {
      message.warning("请先添加并选择单文件任务");
      return;
    }
    if (
      !activeTask.value?.server_id?.trim() ||
      !activeTask.value?.remote_source?.trim() ||
      !activeTask.value?.local_dir?.trim()
    ) {
      message.warning("当前任务配置不完整，请编辑后重试");
      return;
    }
    await start();
  }

  function onStop() {
    stop();
  }

  function onAddTask() {
    emit("addTask");
  }

  function onEditTask(task: BackupTask) {
    emit("editTask", task);
  }

  function progressFormat(pct?: number) {
    return `${pct ?? progressPercent.value}%`;
  }

  return {
    tasks,
    activeTaskId,
    status,
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
    selectTask,
    removeTask,
    onAddTask,
    onEditTask,
    onStart,
    onStop,
  };
}
