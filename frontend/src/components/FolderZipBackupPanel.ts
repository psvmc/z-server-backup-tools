import { computed, nextTick, ref, watch } from "vue";
import { message } from "ant-design-vue";
import type { BackupTask } from "../types/backup";
import { isFolderZipTaskRunnable } from "../types/backup";
import { formatSpeed } from "../utils/backupUi";
import { formatBytes } from "../utils/size";
import { useFolderZipBackup } from "../composables/useFolderZipBackup";
import { useJobElapsed } from "../composables/useJobElapsed";

export const folderZipBackupPanelProps = {
  disabledByOtherJob: {
    type: Boolean,
    default: false,
  },
  panelActive: {
    type: Boolean,
    default: true,
  },
};

export type FolderZipBackupPanelProps = {
  disabledByOtherJob?: boolean;
  panelActive?: boolean;
};

export type FolderZipBackupPanelEmit = {
  (e: "addTask"): void;
  (e: "editTask", task: BackupTask): void;
};

export function useFolderZipBackupPanel(
  props: FolderZipBackupPanelProps,
  emit: FolderZipBackupPanelEmit,
) {
  const panelActive = computed(() => props.panelActive !== false);
  const { tasks, activeTaskId, status, logs, selectTask, removeTask, start, stop } =
    useFolderZipBackup({
      panelActive,
    });
  const { elapsedText } = useJobElapsed(status);

  const autoScrollLog = ref(false);
  const logBoxRef = ref<HTMLElement | null>(null);
  const prevLogLen = ref(0);
  const startModalOpen = ref(false);

  const runnableTasks = computed(() => tasks.value.filter(isFolderZipTaskRunnable));

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
    return status.value.running ? "压缩中…" : "";
  });

  const statusLabel = computed(() => {
    if (status.value.running) {
      const total = status.value.queueTotal ?? 0;
      const index = status.value.queueIndex ?? 0;
      if (total > 1 && index > 0) {
        return `压缩中 (${index}/${total})`;
      }
      return "压缩中";
    }
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
      tasks.value.length === 0 ||
      runnableTasks.value.length === 0,
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

  function onStart() {
    if (props.disabledByOtherJob) {
      message.warning("已有其他备份任务在运行");
      return;
    }
    if (tasks.value.length === 0) {
      message.warning("请先添加文件夹压缩备份任务");
      return;
    }
    if (runnableTasks.value.length === 0) {
      message.warning("没有可执行的任务，请先完善任务配置");
      return;
    }
    startModalOpen.value = true;
  }

  async function onConfirmStart(taskIds: string[]) {
    await start(taskIds);
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
    elapsedText,
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
    startModalOpen,
    selectTask,
    removeTask,
    onAddTask,
    onEditTask,
    onStart,
    onConfirmStart,
    onStop,
  };
}
