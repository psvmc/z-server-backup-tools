<script setup lang="ts">
import type { BackupConfig } from "../types/backup";
import BackupConfigPanel from "./BackupConfigPanel.vue";
import { useBackupSettingsDialog } from "./BackupSettingsDialog";

const open = defineModel<boolean>("open", { required: true });
const config = defineModel<BackupConfig>("config", { required: true });

const props = defineProps<{
  saving?: boolean;
}>();

const emit = defineEmits<{
  saveNotify: [cfg: BackupConfig];
}>();

const { onSaveNotify } = useBackupSettingsDialog(config, props, emit);
</script>

<template>
  <a-modal
    v-model:open="open"
    title="通知设置"
    width="80%"
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
      <a-button type="primary" :loading="props.saving" :disabled="props.saving" @click="onSaveNotify">
        保存配置
      </a-button>
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
</style>
