<script setup lang="ts">
import type { BackupConfig } from "../types/backup";
import BackupConfigPanel from "./BackupConfigPanel.vue";
import { useBackupSettingsDialog } from "./BackupSettingsDialog";

const open = defineModel<boolean>("open", { required: true });
const config = defineModel<BackupConfig>("config", { required: true });

const props = defineProps<{
  configPath: string;
  saving?: boolean;
}>();

const emit = defineEmits<{
  saveNotify: [cfg: BackupConfig];
}>();

const { onSaveNotify, openConfigFolder } = useBackupSettingsDialog(config, props, emit);
</script>

<template>
  <a-modal
    v-model:open="open"
    title="通知设置"
    :width="720"
    wrap-class-name="backup-settings-modal"
    :mask-closable="false"
    :closable="!props.saving"
    :keyboard="!props.saving"
    centered
    :body-style="{ maxHeight: 'calc(100vh - 200px)', overflowY: 'auto', overflowX: 'hidden' }"
  >
    <a-spin :spinning="!!props.saving" tip="保存中…" class="settings-save-spin">
      <BackupConfigPanel v-model:config="config" :saving="props.saving" />
    </a-spin>

    <template #footer>
      <div class="backup-settings-modal-footer">
        <div
          v-if="props.configPath"
          class="backup-settings-modal-footer-path text-xs text-emerald-700/80 truncate"
          :title="props.configPath"
        >
          配置文件：{{ props.configPath }}
        </div>
        <a-button v-if="props.configPath" @click="openConfigFolder">打开所在文件夹</a-button>
        <a-button type="primary" :loading="props.saving" :disabled="props.saving" @click="onSaveNotify">
          保存配置
        </a-button>
      </div>
    </template>
  </a-modal>
</template>

<style scoped>
.settings-save-spin {
  display: block;
  width: 100%;
}

.settings-save-spin :deep(.ant-spin-container) {
  width: 100%;
}

.backup-settings-modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  width: 100%;
}

.backup-settings-modal-footer-path {
  flex: 1;
  min-width: 0;
  text-align: left;
  line-height: 1.4;
}
</style>
