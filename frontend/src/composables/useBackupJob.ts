import { onMounted, onUnmounted, ref } from "vue";
import { message } from "ant-design-vue";
import { Events } from "@wailsio/runtime";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import { BackupConfig as BackupConfigBinding } from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig, JobStatus } from "../types/backup";
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
    remotePullInFlight = true;
    try {
      await BackupService.RefreshRemoteStatus(toBindingConfig(config.value));
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

  const loadConfig = async () => {
    const stored = BackupConfigBinding.createFrom(await BackupService.GetConfig());
    config.value = { ...defaultConfig(), ...bindingToPlain(stored) };
    configPath.value = await BackupService.GetConfigPath();
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
        remote_source: toSave.remote_source,
        local_dir: toSave.local_dir,
        max_part_gb: toSave.max_part_gb,
      };
      configPath.value = await BackupService.GetConfigPath();
      message.success("路径配置已保存");
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
    const hide = message.loading("远程 init 扫描中...", 0);
    try {
      await BackupService.InitRemote(toBindingConfig(config.value));
      await refreshStatus();
      hide();
      const s = status.value;
      if (s.remoteInited && s.totalFiles > 0) {
        message.success(`远程 init 完成：共 ${s.totalFiles} 个文件，已打包 ${s.packedFiles} 个`);
      } else if (s.remoteInited) {
        message.success("远程 init 完成（源目录无文件或已全部完成）");
      } else {
        message.success("远程 init 完成");
      }
    } catch (err) {
      hide();
      message.error(formatError(err));
    }
  };

  const resetBackupTask = async () => {
    const hide = message.loading("重置任务中...", 0);
    try {
      await BackupService.ResetBackupTask(toBindingConfig(config.value));
      await refreshStatus();
      hide();
      message.success("任务已重置，下次备份将从第一个文件开始");
    } catch (err) {
      hide();
      message.error(formatError(err));
    }
  };

  const startBackup = async () => {
    try {
      await BackupService.StartBackup(toBindingConfig(config.value));
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
    configPath,
    status,
    logs,
    settingsSaving,
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
