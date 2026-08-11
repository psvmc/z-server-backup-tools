import { onMounted, onUnmounted, ref, watch, type Ref } from "vue";
import { message } from "ant-design-vue";
import { Events } from "@wailsio/runtime";
import {
  BackupService,
  FolderZipBackupService,
} from "../../bindings/z-server-backup-tools/backend/service";
import type { BackupTask, JobStatus } from "../types/backup";
import { isFolderZipTask } from "../types/backup";
import { formatError } from "../types/update";

const defaultStatus = (): JobStatus => ({
  running: false,
  phase: "",
  currentPart: "",
  localFile: "",
  totalFiles: 0,
  packedFiles: 0,
  done: false,
  lastError: "",
  remoteInited: false,
});

function eventText(ev: unknown): string {
  if (typeof ev === "string") return ev;
  return String((ev as { data?: string })?.data ?? "");
}

const tasks = ref<BackupTask[]>([]);
const activeTaskId = ref("");
const status = ref<JobStatus>(defaultStatus());
const logs = ref<string[]>([]);
const panelActive = ref(true);

let started = false;
let mountCount = 0;
let pollTimer: ReturnType<typeof setInterval> | null = null;
const unsubs: Array<() => void> = [];

const refreshLocal = async () => {
  status.value = (await FolderZipBackupService.GetStatus()) as JobStatus;
  logs.value = (await FolderZipBackupService.GetLogs()) as string[];
};

const loadTasks = async () => {
  const all = (await BackupService.GetTasks()) as BackupTask[];
  tasks.value = all.filter(isFolderZipTask);
  activeTaskId.value = await BackupService.GetActiveFolderZipTaskID();
  if (!activeTaskId.value && tasks.value.length > 0) {
    activeTaskId.value = tasks.value[0].id;
    await BackupService.SetActiveFolderZipTaskID(activeTaskId.value);
  }
};

const selectTask = async (taskId: string) => {
  try {
    await BackupService.SetActiveFolderZipTaskID(taskId);
    activeTaskId.value = taskId;
    await refreshLocal();
  } catch (err) {
    message.error(formatError(err));
  }
};

const removeTask = async (task: BackupTask) => {
  const hide = message.loading("删除任务中...", 0);
  try {
    const current = (await BackupService.GetTasks()) as BackupTask[];
    const next = current.filter((t) => t.id !== task.id);
    await BackupService.SaveTasks(next);
    if (activeTaskId.value === task.id) {
      const remaining = next.filter(isFolderZipTask);
      activeTaskId.value = remaining[0]?.id ?? "";
      await BackupService.SetActiveFolderZipTaskID(activeTaskId.value);
    }
    await loadTasks();
    await refreshLocal();
    hide();
    message.success("任务已删除");
  } catch (err) {
    hide();
    message.error(formatError(err));
  }
};

const start = async (taskIds: string[]) => {
  try {
    await FolderZipBackupService.StartBackup(taskIds);
    if (taskIds.length > 1) {
      message.success(`已启动 ${taskIds.length} 个任务的队列备份`);
    } else {
      message.success("文件夹压缩备份已启动");
    }
    await refreshLocal();
  } catch (err) {
    message.error(formatError(err));
  }
};

const stop = () => {
  void FolderZipBackupService.StopBackup();
  message.info("已请求停止");
};

function startLifecycle() {
  if (started) return;
  started = true;
  void loadTasks()
    .then(() => refreshLocal())
    .catch((err) => {
      console.warn("加载文件夹压缩备份任务失败:", err);
    });
  unsubs.push(
    Events.On("folderzip-log", (ev) => {
      const line = eventText(ev);
      if (line) logs.value = [...logs.value.slice(-499), line];
    }),
    Events.On("folderzip-done", (ev) => {
      const path = eventText(ev);
      void refreshLocal().then(() => loadTasks());
      message.success(path ? `压缩备份完成：${path}` : "压缩备份完成");
    }),
    Events.On("folderzip-error", (ev) => {
      const msg = eventText(ev);
      void refreshLocal();
      if (msg === "任务已取消") {
        message.info(msg);
      } else {
        message.error(msg || "压缩备份失败");
      }
    }),
  );
  pollTimer = setInterval(() => {
    if (status.value.running || panelActive.value) {
      void refreshLocal();
    }
  }, 1500);
}

function stopLifecycle() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
  while (unsubs.length) {
    const off = unsubs.pop();
    off?.();
  }
  started = false;
}

export function useFolderZipBackup(options?: { panelActive?: Ref<boolean> }) {
  if (options?.panelActive) {
    watch(
      options.panelActive,
      (v) => {
        panelActive.value = !!v;
      },
      { immediate: true },
    );
  }

  onMounted(() => {
    mountCount += 1;
    startLifecycle();
  });

  onUnmounted(() => {
    mountCount -= 1;
    if (mountCount <= 0) {
      mountCount = 0;
      stopLifecycle();
    }
  });

  return {
    tasks,
    activeTaskId,
    status,
    logs,
    loadTasks,
    selectTask,
    removeTask,
    start,
    stop,
  };
}
