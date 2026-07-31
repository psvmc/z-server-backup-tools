import { onMounted, onUnmounted, ref, watch, type Ref } from "vue";
import { message } from "ant-design-vue";
import { Events } from "@wailsio/runtime";
import {
  BackupService,
  SingleFileBackupService,
} from "../../bindings/z-server-backup-tools/backend/service";
import type { BackupTask, JobStatus } from "../types/backup";
import { isSingleTask } from "../types/backup";
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

/** Shared across App + SingleFileBackupPanel so status/events are not duplicated. */
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
  status.value = (await SingleFileBackupService.GetStatus()) as JobStatus;
  logs.value = (await SingleFileBackupService.GetLogs()) as string[];
};

const loadTasks = async () => {
  const all = (await BackupService.GetTasks()) as BackupTask[];
  tasks.value = all.filter(isSingleTask);
  activeTaskId.value = await BackupService.GetActiveSingleFileTaskID();
  if (!activeTaskId.value && tasks.value.length > 0) {
    activeTaskId.value = tasks.value[0].id;
    await BackupService.SetActiveSingleFileTaskID(activeTaskId.value);
  }
};

const selectTask = async (taskId: string) => {
  try {
    await BackupService.SetActiveSingleFileTaskID(taskId);
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
      const remainingSingle = next.filter(isSingleTask);
      activeTaskId.value = remainingSingle[0]?.id ?? "";
      await BackupService.SetActiveSingleFileTaskID(activeTaskId.value);
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

const start = async () => {
  try {
    await SingleFileBackupService.StartDownload();
    message.success("单文件下载已启动");
    await refreshLocal();
  } catch (err) {
    message.error(formatError(err));
  }
};

const stop = () => {
  void SingleFileBackupService.StopDownload();
  message.info("已请求停止");
};

function startLifecycle() {
  if (started) return;
  started = true;
  void loadTasks()
    .then(() => refreshLocal())
    .catch((err) => {
      console.warn("加载单文件任务失败:", err);
    });
  unsubs.push(
    Events.On("singlefile-log", (ev) => {
      const line = eventText(ev);
      if (line) logs.value = [...logs.value.slice(-499), line];
    }),
    Events.On("singlefile-done", (ev) => {
      const path = eventText(ev);
      void refreshLocal();
      message.success(path ? `下载完成：${path}` : "下载完成");
    }),
    Events.On("singlefile-error", (ev) => {
      const msg = eventText(ev);
      void refreshLocal();
      if (msg === "任务已取消") {
        message.info(msg);
      } else {
        message.error(msg || "下载失败");
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

export function useSingleFileBackup(options?: { panelActive?: Ref<boolean> }) {
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
