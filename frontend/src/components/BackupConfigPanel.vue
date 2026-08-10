<script setup lang="ts">
import { useBackupConfigPanel } from "./BackupConfigPanel";
import type { BackupConfig } from "../types/backup";

const config = defineModel<BackupConfig>("config", { required: true });

const props = defineProps<{
  saving?: boolean;
}>();

const { mailTesting, testEmail } = useBackupConfigPanel(config, props);
</script>

<template>
  <div class="backup-config-panel" :class="{ 'backup-config-panel--saving': props.saving }">
    <section class="settings-block">
      <div class="settings-block-title">邮件通知</div>
      <a-form layout="vertical" size="small" class="w-full">
        <div class="config-form-grid config-form-grid--mail">
          <a-form-item label="通知邮箱">
            <a-input
              v-model:value="config.notify_email"
              placeholder="备份完成或异常停止时发送通知"
              allow-clear
            />
          </a-form-item>
          <a-form-item label="SMTP 服务器">
            <a-input v-model:value="config.smtp_host" placeholder="smtp.qq.com" allow-clear />
          </a-form-item>
          <a-form-item label="SMTP 端口">
            <a-input-number
              v-model:value="config.smtp_port"
              :min="1"
              :max="65535"
              placeholder="465"
              class="w-full"
            />
          </a-form-item>
          <a-form-item label="SMTP 用户名">
            <a-input
              v-model:value="config.smtp_user"
              placeholder="QQ 邮箱填完整地址，如 xxx@qq.com"
              autocomplete="off"
              allow-clear
            />
          </a-form-item>
          <a-form-item label="SMTP 密码">
            <a-input-password v-model:value="config.smtp_password" autocomplete="off" />
          </a-form-item>
          <a-form-item class="mail-test-item" :colon="false">
            <template #label>
              <span class="mail-test-label-spacer">SMTP 密码</span>
            </template>
            <div class="mail-test-control">
              <a-button
                class="mail-test-btn"
                :loading="mailTesting"
                :disabled="props.saving || mailTesting"
                @click="testEmail"
              >
                邮箱测试
              </a-button>
            </div>
          </a-form-item>
        </div>
      </a-form>
    </section>
  </div>
</template>

<style scoped>
.backup-config-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.settings-block {
  padding: 12px 14px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: linear-gradient(180deg, var(--app-surface-card) 0%, #ffffff 100%);
}

.settings-block-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-primary-dark);
  margin-bottom: 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--app-border-light);
}

.config-form-grid--mail {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0 12px;
  align-items: end;
}

:deep(.ant-form-item) {
  margin-bottom: 8px;
}

.backup-config-panel :deep(.ant-input-number) {
  width: 100%;
}

.mail-test-item :deep(.ant-form-item-label > label) {
  height: auto;
}

.mail-test-label-spacer {
  visibility: hidden;
  user-select: none;
  pointer-events: none;
}

.mail-test-control {
  display: flex;
  align-items: center;
  min-height: 24px;
}

.mail-test-btn {
  height: 24px;
  padding-inline: 12px;
  font-size: 14px;
  line-height: 1;
}

@media (max-width: 720px) {
  .config-form-grid--mail {
    grid-template-columns: 1fr;
  }
}
</style>
