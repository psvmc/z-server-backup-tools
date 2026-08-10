import { computed, onMounted, onUnmounted, ref } from "vue";
import { message } from "ant-design-vue";
import { Events } from "@wailsio/runtime";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import {
  BackupConfig as BackupConfigBinding,
  Server as ServerBinding,
} from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig, BackupTask, JobStatus, Server } from "../types/backup";
import { emptyNotifyConfig, isMultiTask, mergeJobConfig } from "../types/backup";
import { formatError } from "../types/update";

function toBindingConfig(cfg: BackupConfig) {
  return BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(cfg)));
}

function toBindingServer(srv: Server) {
  return ServerBinding.createFrom(JSON.parse(JSON.stringify(srv)));
}

function bindingToPlain(cfg: BackupConfigBinding): BackupConfig {
  return JSON.parse(JSON.stringify(cfg)) as BackupConfig;
}

function bindingToPlainServer(srv: ServerBinding): Server {
  return JSON.parse(JSON.stringify(srv)) as Server;
}

function eventText(ev: unknown): string {
  if (typeof ev === "string") return ev;
  return String((ev as { data?: string })?.data ?? "");
}

const MAX_LOG_LINES = 500;

function appendLogLine(lines: string[], line: string) {
  if (lines.length >= MAX_LOG_LINES) {
    lines.shift();
  }
  lines.push(line);
}

export function useBackupJob() {
  const config = ref<BackupConfig>(emptyNotifyConfig());
  const servers = ref<Server[]>([]);
  const allTasks = ref<BackupTask[]>([]);
  const tasks = computed(() => allTasks.value.filter(isMultiTask));
  const activeTaskId = ref("");
  const status = ref<JobStatus>({
    running: false,
    phase: "",
    currentPart: "",
    localFile: "",
    totalFiles: 0,
    packedFiles: 0,
    done: false,
    lastError: "",
    remoteInited: false,
    pendingZip: "",
    remoteHint: "",
    maxFileBytes: 0,
    oversizedFileCount: 0,
  });
  const logs = ref<string[]>([]);
  const configPath = ref("");
  const settingsSaving = ref(false);
  const remoteInitLoading = ref(false);

  const activeTask = computed(() => tasks.value.find((t) => t.id === activeTaskId.value) ?? null);

  const findServer = (id?: string | null) => {
    if (!id?.trim()) return null;
    return servers.value.find((s) => s.id === id) ?? null;
  };

  const jobConfig = computed(() =>
    mergeJobConfig(config.value, findServer(activeTask.value?.server_id), activeTask.value),
  );

  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let remotePollTimer: ReturnType<typeof setInterval> | null = null;
  let remotePullInFlight = false;
  let resetHide: (() => void) | null = null;
  const unsubs: Array<() => void> = [];

  const refreshLocal = async () => {
    status.value = (await BackupService.GetJobStatus()) as JobStatus;
    logs.value = (await BackupService.GetLogs()) as string[];
  };

  const refreshStatusOnly = async () => {
    status.value = (await BackupService.GetJobStatus()) as JobStatus;
  };

  const pullRemoteSnapshot = async () => {
    if (remotePullInFlight || status.value.running) return;
    if (!jobConfig.value.task_id?.trim()) return;
    remotePullInFlight = true;
    try {
      await BackupService.RefreshRemoteStatus(toBindingConfig(jobConfig.value));
    } catch {
      remotePullInFlight = false;
    }
  };

  const refreshStatus = async () => {
    if (!status.value.running) {
      await pullRemoteSnapshot();
      return;
    }
    await refreshStatusOnly();
  };

  const loadServers = async () => {
    const list = await BackupService.GetServers();
    servers.value = (list ?? []).map((s) => bindingToPlainServer(s));
  };

  const loadTasks = async () => {
    allTasks.value = (await BackupService.GetTasks()) as BackupTask[];
    activeTaskId.value = await BackupService.GetActiveTaskID();
    const multi = tasks.value;
    if (!activeTaskId.value && multi.length > 0) {
      activeTaskId.value = multi[0].id;
      await BackupService.SetActiveTaskID(activeTaskId.value);
    }
  };

  const loadConfig = async () => {
    const stored = bindingToPlain(BackupConfigBinding.createFrom(await BackupService.GetConfig()));
    config.value = {
      ...emptyNotifyConfig(),
      notify_email: stored.notify_email,
      smtp_host: stored.smtp_host?.trim() || "smtp.qq.com",
      smtp_port: stored.smtp_port && stored.smtp_port > 0 ? stored.smtp_port : 465,
      smtp_user: stored.smtp_user,
      smtp_password: stored.smtp_password,
    };
    configPath.value = await BackupService.GetConfigPath();
    await loadServers();
    await loadTasks();
  };

  const saveServer = async (srv: Server): Promise<Server | null> => {
    try {
      const saved = await BackupService.SaveServer(toBindingServer(srv));
      await loadServers();
      message.success("服务器已保存");
      return bindingToPlainServer(saved);
    } catch (err) {
      message.error(formatError(err));
      return null;
    }
  };

  const deleteServer = async (id: string): Promise<boolean> => {
    try {
      await BackupService.DeleteServer(id);
      await loadServers();
      message.success("服务器已删除");
      return true;
    } catch (err) {
      message.error(formatError(err));
      return false;
    }
  };

  const selectTask = async (taskId: string) => {
    try {
      await BackupService.SetActiveTaskID(taskId);
      activeTaskId.value = taskId;
      await refreshStatus();
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
        const remainingMulti = next.filter(isMultiTask);
        activeTaskId.value = remainingMulti[0]?.id ?? "";
        await BackupService.SetActiveTaskID(activeTaskId.value);
      }
      await loadTasks();
      await refreshStatus();
      hide();
      message.success("任务已删除");
    } catch (err) {
      hide();
      message.error(formatError(err));
    }
  };

  const saveNotify = async (payload?: BackupConfig): Promise<boolean> => {
    const raw = { ...emptyNotifyConfig(), ...(payload ?? config.value) };
    const toSave = {
      ...raw,
      smtp_host: raw.smtp_host?.trim() || "smtp.qq.com",
      smtp_port: raw.smtp_port && raw.smtp_port > 0 ? raw.smtp_port : 465,
    };
    settingsSaving.value = true;
    try {
      await BackupService.SaveNotifyConfig(toBindingConfig(toSave));
      config.value = {
        ...emptyNotifyConfig(),
        notify_email: toSave.notify_email,
        smtp_host: toSave.smtp_host,
        smtp_port: toSave.smtp_port,
        smtp_user: toSave.smtp_user,
        smtp_password: toSave.smtp_password,
      };
      configPath.value = await BackupService.GetConfigPath();
      message.success("通知设置已保存");
      return true;
    } catch (err) {
      message.error(formatError(err));
      return false;
    } finally {
      settingsSaving.value = false;
    }
  };

  const showInitSuccess = () => {
    const s = status.value;
    if (s.remoteInited && s.totalFiles > 0) {
      message.success(`远程 init 完成：共 ${s.totalFiles} 个文件，已打包 ${s.packedFiles} 个`);
    } else if (s.remoteInited) {
      message.success("远程 init 完成（源目录无文件或已全部完成）");
    } else {
      message.success("远程 init 完成");
    }
  };

  const initRemote = async () => {
    if (remoteInitLoading.value) return;
    if (!jobConfig.value.task_id?.trim()) {
      message.warning("请先添加并选择备份任务");
      return;
    }
    remoteInitLoading.value = true;
    try {
      await BackupService.InitRemote(toBindingConfig(jobConfig.value));
    } catch (err) {
      remoteInitLoading.value = false;
      message.error(formatError(err));
    }
  };

  const resetBackupTask = async () => {
    if (!jobConfig.value.task_id?.trim()) {
      message.warning("请先选择备份任务");
      return;
    }
    resetHide = message.loading("重置任务中...", 0);
    try {
      await BackupService.ResetBackupTask(toBindingConfig(jobConfig.value));
    } catch (err) {
      resetHide?.();
      resetHide = null;
      message.error(formatError(err));
    }
  };

  const startBackup = async () => {
    if (!jobConfig.value.task_id?.trim()) {
      message.warning("请先添加并选择备份任务");
      return;
    }
    try {
      await BackupService.StartBackup(toBindingConfig(jobConfig.value));
      message.success("备份流水线已启动");
      await refreshStatusOnly();
    } catch (err) {
      message.error(formatError(err));
    }
  };

  const stopBackup = () => {
    BackupService.StopBackup();
    message.info("已请求停止");
  };

  const openConfigFolder = async () => {
    const path = configPath.value?.trim();
    if (!path) return;
    try {
      await BackupService.OpenInExplorer(path);
    } catch (err) {
      message.error(formatError(err));
    }
  };

  onMounted(async () => {
    try {
      await loadConfig();
      await refreshStatus();
    } catch (err) {
      console.warn("加载备份配置失败:", err);
    }
    unsubs.push(
      Events.On("backup-log", (ev) => {
        const line = eventText(ev);
        if (line) appendLogLine(logs.value, line);
      }),
      Events.On("backup-done", () => {
        void refreshLocal().then(() => message.success("备份已完成"));
      }),
      Events.On("backup-error", (ev) => {
        const msg = eventText(ev);
        void refreshLocal();
        if (msg.includes("任务已取消")) {
          message.info("备份任务已取消");
        } else {
          message.error(msg || "备份失败");
        }
      }),
      Events.On("remote-init-done", () => {
        remoteInitLoading.value = false;
        void refreshLocal().then(showInitSuccess);
      }),
      Events.On("remote-init-error", (ev) => {
        remoteInitLoading.value = false;
        void refreshLocal();
        message.error(formatError(eventText(ev)));
      }),
      Events.On("remote-status-refreshed", () => {
        remotePullInFlight = false;
        void refreshLocal();
      }),
      Events.On("backup-reset-done", () => {
        resetHide?.();
        resetHide = null;
        void refreshLocal();
        message.success("任务已重置，下次备份将从第一个文件开始");
      }),
      Events.On("backup-reset-error", (ev) => {
        resetHide?.();
        resetHide = null;
        message.error(formatError(eventText(ev)));
      }),
    );
    pollTimer = setInterval(() => {
      if (status.value.running) {
        void refreshStatusOnly();
      } else {
        void refreshLocal();
      }
    }, 2000);
    remotePollTimer = setInterval(() => {
      void pullRemoteSnapshot();
    }, 10000);
  });

  onUnmounted(() => {
    if (pollTimer) clearInterval(pollTimer);
    if (remotePollTimer) clearInterval(remotePollTimer);
    for (const off of unsubs) off();
  });

  return {
    config,
    servers,
    tasks,
    activeTaskId,
    activeTask,
    jobConfig,
    findServer,
    configPath,
    openConfigFolder,
    status,
    logs,
    settingsSaving,
    remoteInitLoading,
    loadServers,
    loadTasks,
    selectTask,
    removeTask,
    saveNotify,
    saveServer,
    deleteServer,
    initRemote,
    resetBackupTask,
    startBackup,
    stopBackup,
    refreshStatus,
  };
}
