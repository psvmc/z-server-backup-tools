import { onMounted, onUnmounted, ref, type Ref } from "vue";
import { message } from "ant-design-vue";
import { Events } from "@wailsio/runtime";
import { SingleFileBackupService } from "../../bindings/z-server-backup-tools/backend/service";
import {
  BackupConfig as BackupConfigBinding,
  SingleFileConfig as SingleFileConfigBinding,
} from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig, JobStatus, SingleFileConfig } from "../types/backup";
import { formatError } from "../types/update";

const defaultPaths = (): SingleFileConfig => ({
  remote_file: "",
  local_dir: "",
});

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

function toBindingPaths(cfg: SingleFileConfig) {
  return SingleFileConfigBinding.createFrom(JSON.parse(JSON.stringify(cfg)));
}

function toBindingSSH(cfg: BackupConfig) {
  return BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(cfg)));
}

function bindingToPlainPaths(cfg: SingleFileConfigBinding): SingleFileConfig {
  return JSON.parse(JSON.stringify(cfg)) as SingleFileConfig;
}

function eventText(ev: unknown): string {
  if (typeof ev === "string") return ev;
  return String((ev as { data?: string })?.data ?? "");
}

export function useSingleFileBackup(options?: { panelActive?: Ref<boolean> }) {
  const panelActive = options?.panelActive ?? ref(true);
  const paths = ref<SingleFileConfig>(defaultPaths());
  const status = ref<JobStatus>(defaultStatus());
  const logs = ref<string[]>([]);
  const saving = ref(false);

  let pollTimer: ReturnType<typeof setInterval> | null = null;
  const unsubs: Array<() => void> = [];

  const refreshLocal = async () => {
    status.value = (await SingleFileBackupService.GetStatus()) as JobStatus;
    logs.value = (await SingleFileBackupService.GetLogs()) as string[];
  };

  const load = async () => {
    const stored = SingleFileConfigBinding.createFrom(await SingleFileBackupService.GetConfig());
    paths.value = { ...defaultPaths(), ...bindingToPlainPaths(stored) };
    await refreshLocal();
  };

  const savePaths = async (payload?: SingleFileConfig): Promise<boolean> => {
    const toSave = { ...defaultPaths(), ...(payload ?? paths.value) };
    saving.value = true;
    try {
      await SingleFileBackupService.SaveConfig(toBindingPaths(toSave));
      paths.value = toSave;
      message.success("单文件路径已保存");
      return true;
    } catch (err) {
      message.error(formatError(err));
      return false;
    } finally {
      saving.value = false;
    }
  };

  const start = async (sshCfg: BackupConfig) => {
    try {
      await SingleFileBackupService.StartDownload(toBindingSSH(sshCfg), toBindingPaths(paths.value));
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

  onMounted(async () => {
    try {
      await load();
    } catch (err) {
      console.warn("加载单文件配置失败:", err);
    }
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
  });

  onUnmounted(() => {
    if (pollTimer) clearInterval(pollTimer);
    for (const off of unsubs) off();
  });

  return { paths, status, logs, saving, load, savePaths, start, stop };
}
