import { computed, onMounted, onUnmounted, ref } from "vue";
import { message } from "ant-design-vue";
import { Events } from "@wailsio/runtime";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import { BackupConfig as BackupConfigBinding } from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig, BackupTask, JobStatus } from "../types/backup";
import { mergeTaskConfig } from "../types/backup";
import { formatError } from "../types/update";

const defaultConfig = (): BackupConfig => ({
  host: "",
  port: 22,
  user: "",
  password: "",
  remote_app_dir: "D:/Tools/zipbak",
  remote_source: "",
  local_dir: "",
  max_part_gb: 2,
});

function toBindingConfig(cfg: BackupConfig) {
  return BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(cfg)));
}

function bindingToPlain(cfg: BackupConfigBinding): BackupConfig {
  return JSON.parse(JSON.stringify(cfg)) as BackupConfig;
}

export function useBackupJob() {
  const config = ref<BackupConfig>(defaultConfig());
  const tasks = ref<BackupTask[]>([]);
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
  const jobConfig = computed(() => mergeTaskConfig(config.value, activeTask.value));

  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let remotePollTimer: ReturnType<typeof setInterval> | null = null;
  let remotePullInFlight = false;
  const unsubs: Array<() => void> = [];

  const refreshLocal = async () => {
    status.value = (await BackupService.GetJobStatus()) as JobStatus;
    logs.value = (await BackupService.GetLogs()) as string[];
  };

  const pullRemoteSnapshot = async () => {
    if (remotePullInFlight || status.value.running) return;
    if (!jobConfig.value.task_id?.trim()) return;
    remotePullInFlight = true;
    try {
      await BackupService.RefreshRemoteStatus(toBindingConfig(jobConfig.value));
      await refreshLocal();
    } catch {
      // 配置未齐、远程忙或不可达时忽略
    } finally {
      remotePullInFlight = false;
    }
  };

  const refreshStatus = async () => {
    if (!status.value.running) {
      await pullRemoteSnapshot();
      return;
    }
    await refreshLocal();
  };

  const loadTasks = async () => {
    tasks.value = (await BackupService.GetTasks()) as BackupTask[];
    activeTaskId.value = await BackupService.GetActiveTaskID();
    if (!activeTaskId.value && tasks.value.length > 0) {
      activeTaskId.value = tasks.value[0].id;
      await BackupService.SetActiveTaskID(activeTaskId.value);
    }
  };

  const loadConfig = async () => {
    const stored = BackupConfigBinding.createFrom(await BackupService.GetConfig());
    config.value = { ...defaultConfig(), ...bindingToPlain(stored) };
    configPath.value = await BackupService.GetConfigPath();
    await loadTasks();
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
      const next = tasks.value.filter((t) => t.id !== task.id);
      await BackupService.SaveTasks(next);
      tasks.value = next;
      if (activeTaskId.value === task.id) {
        activeTaskId.value = next[0]?.id ?? "";
        if (activeTaskId.value) {
          await BackupService.SetActiveTaskID(activeTaskId.value);
        } else {
          await BackupService.SetActiveTaskID("");
        }
      }
      await refreshStatus();
      hide();
      message.success("任务已删除");
    } catch (err) {
      hide();
      message.error(formatError(err));
    }
  };

  const saveConnectionConfig = async (payload?: BackupConfig): Promise<boolean> => {
    const toSave = { ...defaultConfig(), ...(payload ?? config.value) };
    settingsSaving.value = true;
    try {
      await BackupService.SaveConnectionConfig(toBindingConfig(toSave));
      config.value = { ...config.value, host: toSave.host, user: toSave.user, password: toSave.password, port: toSave.port };
      configPath.value = await BackupService.GetConfigPath();
      message.success("服务器连接已保存");
      return true;
    } catch (err) {
      message.error(formatError(err));
      return false;
    } finally {
      settingsSaving.value = false;
    }
  };

  const savePathsConfig = async (payload?: BackupConfig): Promise<boolean> => {
    const toSave = { ...defaultConfig(), ...(payload ?? config.value) };
    settingsSaving.value = true;
    try {
      await BackupService.SavePathsConfig(toBindingConfig(toSave));
      config.value = {
        ...config.value,
        remote_app_dir: toSave.remote_app_dir,
        max_part_gb: toSave.max_part_gb,
        notify_email: toSave.notify_email,
        smtp_host: toSave.smtp_host,
        smtp_port: toSave.smtp_port,
        smtp_user: toSave.smtp_user,
        smtp_password: toSave.smtp_password,
      };
      configPath.value = await BackupService.GetConfigPath();
      message.success("应用配置已保存");
      return true;
    } catch (err) {
      message.error(formatError(err));
      return false;
    } finally {
      settingsSaving.value = false;
    }
  };

  const testConnection = async () => {
    const hide = message.loading("测试 SSH 连接...", 0);
    try {
      await BackupService.TestConnection(toBindingConfig(config.value));
      hide();
      message.success("连接成功");
    } catch (err) {
      hide();
      message.error(formatError(err));
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
      await refreshStatus();
      const s = status.value;
      if (s.remoteInited && s.totalFiles > 0) {
        message.success(`远程 init 完成：共 ${s.totalFiles} 个文件，已打包 ${s.packedFiles} 个`);
      } else if (s.remoteInited) {
        message.success("远程 init 完成（源目录无文件或已全部完成）");
      } else {
        message.success("远程 init 完成");
      }
    } catch (err) {
      message.error(formatError(err));
    } finally {
      remoteInitLoading.value = false;
    }
  };

  const resetBackupTask = async () => {
    if (!jobConfig.value.task_id?.trim()) {
      message.warning("请先选择备份任务");
      return;
    }
    const hide = message.loading("重置任务中...", 0);
    try {
      await BackupService.ResetBackupTask(toBindingConfig(jobConfig.value));
      await refreshStatus();
      hide();
      message.success("任务已重置，下次备份将从第一个文件开始");
    } catch (err) {
      hide();
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
      await refreshStatus();
    } catch (err) {
      message.error(formatError(err));
    }
  };

  const stopBackup = () => {
    BackupService.StopBackup();
    message.info("已请求停止");
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
        const line = typeof ev === "string" ? ev : String((ev as { data?: string })?.data ?? "");
        if (line) logs.value = [...logs.value.slice(-499), line];
      }),
    );
    pollTimer = setInterval(() => {
      void refreshLocal();
    }, 1500);
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
    tasks,
    activeTaskId,
    activeTask,
    jobConfig,
    configPath,
    status,
    logs,
    settingsSaving,
    remoteInitLoading,
    loadTasks,
    selectTask,
    removeTask,
    saveConnectionConfig,
    savePathsConfig,
    testConnection,
    initRemote,
    resetBackupTask,
    startBackup,
    stopBackup,
    refreshStatus,
  };
}
