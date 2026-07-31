<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { theme } from "ant-design-vue";
import zhCN from "ant-design-vue/es/locale/zh_CN";
import UpdateConfirmDialog from "./components/UpdateConfirmDialog.vue";
import UpdateProgressDialog from "./components/UpdateProgressDialog.vue";
import BackupSettingsDialog from "./components/BackupSettingsDialog.vue";
import BackupTaskFormModal from "./components/BackupTaskFormModal.vue";
import BackupRunPanel from "./components/BackupRunPanel.vue";
import SingleFileBackupPanel from "./components/SingleFileBackupPanel.vue";
import ServerManageDialog from "./components/ServerManageDialog.vue";
import { useAppUpdate } from "./composables/useAppUpdate";
import { useUpdateProgress } from "./composables/useUpdateProgress";
import { useBackupJob } from "./composables/useBackupJob";
import { useBackupTiming } from "./composables/useBackupTiming";
import { useSingleFileBackup } from "./composables/useSingleFileBackup";
import { useAppMainTabs, useJobExclusion } from "./AppTabs";
import { downloadPhaseLabel } from "./utils/backupUi";
import { mergeJobConfig } from "./types/backup";

import type { BackupConfig, BackupTask } from "./types/backup";

const appTitle = ref("服务器文件备份");
const settingsOpen = ref(false);
const serversOpen = ref(false);
const taskFormOpen = ref(false);
const editingTask = ref<BackupTask | null>(null);
const { checkOnStartup, checkForUpdate, loadCurrentVersion } = useAppUpdate();
const { updateProgress } = useUpdateProgress();
const {
  config,
  servers,
  tasks,
  activeTaskId,
  activeTask,
  findServer,
  configPath,
  status,
  logs,
  settingsSaving,
  remoteInitLoading,
  loadServers,
  loadTasks,
  selectTask,
  removeTask,
  saveNotify,
  initRemote,
  resetBackupTask,
  startBackup,
  stopBackup,
  refreshStatus,
} = useBackupJob();

const jobConfig = computed(() =>
  mergeJobConfig(config.value, findServer(activeTask.value?.server_id), activeTask.value),
);

const { mainTab, singlePanelActive } = useAppMainTabs();
const single = useSingleFileBackup({ panelActive: singlePanelActive });
const { multiRunning, singleRunning } = useJobExclusion(status, single.status);

const { elapsedText, remainingText } = useBackupTiming(status);

const phaseLabel = computed(() => {
  const downloadLabel = downloadPhaseLabel(status.value);
  if (downloadLabel) {
    return downloadLabel;
  }
  const map: Record<string, string> = {
    starting: "启动中",
    pack: "远程打包",
    download: "下载",
    verify: "校验",
    delete: "删除远程包",
    ack: "确认 ack",
  };
  const phase = status.value.phase;
  return map[phase] ?? (phase || "空闲");
});

onMounted(async () => {
  try {
    const ver = await loadCurrentVersion();
    appTitle.value = `服务器文件备份 v${ver}`;
  } catch {
    // keep default
  }
  void checkOnStartup();
});

const appTheme = {
  token: {
    colorPrimary: "#5bb8a8",
    colorInfo: "#5bb8a8",
    colorSuccess: "#52b788",
    borderRadius: 8,
    fontSize: 13,
    controlHeight: 32,
    colorBgContainer: "#ffffff",
    colorBorder: "#dce8e4",
  },
  algorithm: theme.defaultAlgorithm,
};

async function onSaveNotify(cfg: BackupConfig) {
  if (await saveNotify(cfg)) {
    settingsOpen.value = false;
  }
}

function openAddTask() {
  editingTask.value = null;
  taskFormOpen.value = true;
}

function openEditTask(task: BackupTask) {
  editingTask.value = task;
  taskFormOpen.value = true;
}

async function onTaskSaved() {
  await loadTasks();
  await refreshStatus();
}

function onServersChanged() {
  void loadServers();
}
</script>

<template>
  <a-config-provider :locale="zhCN" :theme="appTheme">
    <div class="app-shell">
      <header class="app-titlebar">
        <div class="app-titlebar-main">{{ appTitle }}</div>
        <button type="button" class="app-version-btn" title="检查更新" @click="checkForUpdate">
          检查更新
        </button>
      </header>

      <div class="app-body">
        <a-tabs v-model:activeKey="mainTab" class="app-main-tabs">
          <template #rightExtra>
            <a-space :size="8">
              <a-button type="default" size="small" @click="serversOpen = true">服务器管理</a-button>
              <a-button type="default" size="small" @click="settingsOpen = true">通知设置</a-button>
            </a-space>
          </template>
          <a-tab-pane key="multi" tab="多文件备份" :disabled="singleRunning">
            <BackupRunPanel
              :status="status"
              :config="config"
              :job-config="jobConfig"
              :tasks="tasks"
              :active-task-id="activeTaskId"
              :phase-label="phaseLabel"
              :elapsed-text="elapsedText"
              :remaining-text="remainingText"
              :logs="logs"
              :init-loading="remoteInitLoading"
              @start="startBackup"
              @stop="stopBackup"
              @refresh="refreshStatus"
              @init="initRemote"
              @reset="resetBackupTask"
              @add-task="openAddTask"
              @edit-task="openEditTask"
              @select-task="selectTask"
              @remove-task="removeTask"
            />
          </a-tab-pane>
          <a-tab-pane key="single" tab="单文件备份" :disabled="multiRunning">
            <SingleFileBackupPanel
              :ssh-config="config"
              :servers="servers"
              :disabled-by-other-job="multiRunning"
              :panel-active="singlePanelActive"
            />
          </a-tab-pane>
        </a-tabs>
      </div>

      <BackupSettingsDialog
        v-model:open="settingsOpen"
        v-model:config="config"
        :config-path="configPath"
        :saving="settingsSaving"
        @save-notify="onSaveNotify"
      />

      <ServerManageDialog v-model:open="serversOpen" @changed="onServersChanged" />

      <BackupTaskFormModal
        v-model:open="taskFormOpen"
        :connection="config"
        :servers="servers"
        :task="editingTask"
        @saved="onTaskSaved"
      />

      <UpdateConfirmDialog />
      <UpdateProgressDialog v-model:progress="updateProgress" />
    </div>
  </a-config-provider>
</template>
