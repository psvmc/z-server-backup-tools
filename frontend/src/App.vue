<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { theme } from "ant-design-vue";
import zhCN from "ant-design-vue/es/locale/zh_CN";
import UpdateConfirmDialog from "./components/UpdateConfirmDialog.vue";
import UpdateProgressDialog from "./components/UpdateProgressDialog.vue";
import BackupSettingsDialog from "./components/BackupSettingsDialog.vue";
import BackupRunPanel from "./components/BackupRunPanel.vue";
import { useAppUpdate } from "./composables/useAppUpdate";
import { useUpdateProgress } from "./composables/useUpdateProgress";
import { useBackupJob } from "./composables/useBackupJob";
import { useBackupTiming } from "./composables/useBackupTiming";

import type { BackupConfig } from "./types/backup";

const appTitle = ref("ZServerBackup v1.0.0");
const settingsOpen = ref(false);
const { checkOnStartup, checkForUpdate, loadCurrentVersion } = useAppUpdate();
const { updateProgress } = useUpdateProgress();
const {
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
} = useBackupJob();

const { elapsedText, remainingText } = useBackupTiming(status);

const phaseLabel = computed(() => {
  const map: Record<string, string> = {
    starting: "启动中",
    pack: "远程打包",
    download: "下载",
    delete: "删除远程包",
    ack: "确认 ack",
  };
  const phase = status.value.phase;
  return map[phase] ?? (phase || "空闲");
});

onMounted(async () => {
  try {
    const ver = await loadCurrentVersion();
    appTitle.value = `ZServerBackup v${ver}`;
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

async function onSaveConnection(cfg: BackupConfig) {
  await saveConnectionConfig(cfg);
}

async function onSavePaths(cfg: BackupConfig) {
  if (await savePathsConfig(cfg)) {
    settingsOpen.value = false;
  }
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
        <BackupRunPanel
          :status="status"
          :phase-label="phaseLabel"
          :elapsed-text="elapsedText"
          :remaining-text="remainingText"
          :logs="logs"
          @start="startBackup"
          @stop="stopBackup"
          @refresh="refreshStatus"
          @init="initRemote"
          @reset="resetBackupTask"
          @open-settings="settingsOpen = true"
        />
      </div>

      <BackupSettingsDialog
        v-model:open="settingsOpen"
        v-model:config="config"
        :config-path="configPath"
        :saving="settingsSaving"
        @save-connection="onSaveConnection"
        @save-paths="onSavePaths"
        @test="testConnection"
      />

      <UpdateConfirmDialog />
      <UpdateProgressDialog v-model:progress="updateProgress" />
    </div>
  </a-config-provider>
</template>
