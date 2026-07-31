<script setup lang="ts">
import { computed, ref } from "vue";
import { message } from "ant-design-vue";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import { BackupConfig as BackupConfigBinding } from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig } from "../types/backup";
import { remotePathsFromAppDir } from "../types/backup";
import { formatError } from "../types/update";
import PathPickInput from "./PathPickInput.vue";
import RemoteDirPickerModal from "./RemoteDirPickerModal.vue";

const config = defineModel<BackupConfig>("config", { required: true });

const props = defineProps<{
  saving?: boolean;
}>();

const emit = defineEmits<{
  saveConnection: [cfg: BackupConfig];
  test: [];
}>();

const remotePickerOpen = ref(false);
const mailTesting = ref(false);

const derivedPaths = computed(() => remotePathsFromAppDir(config.value.remote_app_dir ?? ""));

const snapshot = () => ({ ...config.value });

const onSaveConnection = () => {
  if (props.saving) return;
  emit("saveConnection", snapshot());
};

function ensureConnectionFilled(): boolean {
  if (!config.value.host?.trim() || !config.value.user?.trim()) {
    message.warning("请先填写 SSH 主机和用户名");
    return false;
  }
  return true;
}

function openRemotePicker() {
  if (props.saving) return;
  if (!ensureConnectionFilled()) return;
  remotePickerOpen.value = true;
}

function onRemoteSelect(path: string) {
  config.value.remote_app_dir = path;
}

async function testEmail() {
  if (mailTesting.value || props.saving) return;
  if (!config.value.notify_email?.trim()) {
    message.warning("请先填写通知邮箱");
    return;
  }
  if (!config.value.smtp_host?.trim()) {
    message.warning("请先填写 SMTP 服务器");
    return;
  }
  mailTesting.value = true;
  const hide = message.loading("正在发送测试邮件…", 0);
  try {
    const payload = {
      ...config.value,
      smtp_port: config.value.smtp_port && config.value.smtp_port > 0 ? config.value.smtp_port : 587,
    };
    const cfg = BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(payload)));
    await BackupService.TestEmailNotification(cfg);
    hide();
    message.success("测试邮件已发送，请查收收件箱（含垃圾箱）");
  } catch (err) {
    hide();
    message.error(formatError(err));
  } finally {
    mailTesting.value = false;
  }
}
</script>

<template>
  <div class="backup-config-panel" :class="{ 'backup-config-panel--saving': props.saving }">
    <section class="settings-block">
      <div class="settings-block-title">SSH 连接</div>
      <a-form layout="vertical" size="small" class="w-full">
        <div class="config-form-grid config-form-grid--conn">
          <a-form-item label="主机">
            <a-input v-model:value="config.host" placeholder="192.168.1.10" />
          </a-form-item>
          <a-form-item label="端口">
            <a-input-number v-model:value="config.port" :min="1" :max="65535" class="w-full" />
          </a-form-item>
          <a-form-item label="用户名">
            <a-input v-model:value="config.user" />
          </a-form-item>
          <a-form-item label="密码">
            <a-input-password v-model:value="config.password" autocomplete="off" />
          </a-form-item>
        </div>
        <div class="settings-block-actions">
          <a-button type="primary" :loading="props.saving" :disabled="props.saving" @click="onSaveConnection">
            保存连接
          </a-button>
          <a-button :disabled="props.saving" @click="$emit('test')">测试连接</a-button>
        </div>
      </a-form>
    </section>

    <section class="settings-block">
      <div class="settings-block-title">远程应用</div>
      <a-form layout="vertical" size="small" class="w-full">
        <div class="config-form-grid">
          <a-form-item label="远程应用目录" class="config-form-span-2">
            <PathPickInput
              v-model="config.remote_app_dir"
              editable
              placeholder="可手动输入，如 D:\Tools\zipbak"
              @browse="openRemotePicker"
            />
          </a-form-item>
          <a-form-item label="分卷上限 (GB)" class="config-form-span-2">
            <a-input-number
              v-model:value="config.max_part_gb"
              :min="0.1"
              :step="0.5"
              class="w-full"
            />
            <div class="text-xs text-gray-500 mt-1">单文件超过上限时仍会单独打成一卷</div>
          </a-form-item>

          <div
            v-if="config.remote_app_dir?.trim()"
            class="config-form-span-4 settings-derived-paths"
          >
            <div class="settings-derived-paths-label">自动推导路径（按任务 ID 区分）</div>
            <dl class="settings-derived-paths-list">
              <div><dt>zipbak-srv</dt><dd>{{ derivedPaths.srv }}</dd></div>
              <div><dt>state</dt><dd>{{ derivedPaths.state }}</dd></div>
              <div><dt>staging</dt><dd>{{ derivedPaths.staging }}</dd></div>
            </dl>
          </div>
        </div>
      </a-form>
    </section>

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
            <a-input v-model:value="config.smtp_host" placeholder="smtp.example.com" allow-clear />
          </a-form-item>
          <a-form-item label="SMTP 端口">
            <a-input-number
              v-model:value="config.smtp_port"
              :min="1"
              :max="65535"
              placeholder="587"
              class="w-full"
            />
          </a-form-item>
          <a-form-item label="SMTP 用户名">
            <a-input v-model:value="config.smtp_user" autocomplete="off" allow-clear />
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

    <RemoteDirPickerModal
      v-model:open="remotePickerOpen"
      :connection="config"
      :initial-path="config.remote_app_dir"
      title="选择远程应用目录"
      @select="onRemoteSelect"
    />
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
  border: 1px solid #dcebe4;
  border-radius: 10px;
  background: linear-gradient(180deg, #fafdfb 0%, #ffffff 100%);
}

.settings-block-title {
  font-size: 13px;
  font-weight: 600;
  color: #1a4d42;
  margin-bottom: 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid #e8f2ee;
}

.settings-block-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
  padding-top: 4px;
}

.config-form-grid--conn {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.config-form-grid--mail {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  align-items: end;
}

.settings-derived-paths {
  font-size: 12px;
  color: #4b635c;
  border-radius: 8px;
  border: 1px solid #dcebe4;
  background: #f3faf7;
  padding: 10px 12px;
}

.settings-derived-paths-label {
  font-weight: 500;
  color: #2d6a5a;
  margin-bottom: 8px;
}

.settings-derived-paths-list {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px 16px;
}

.settings-derived-paths-list > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.settings-derived-paths-list dt {
  margin: 0;
  font-weight: 500;
  color: #5a7a72;
}

.settings-derived-paths-list dd {
  margin: 0;
  word-break: break-all;
  font-family: Consolas, "Courier New", monospace;
  font-size: 11px;
  line-height: 1.4;
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
  .config-form-grid--conn {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .config-form-grid--mail {
    grid-template-columns: 1fr;
  }
  .settings-derived-paths-list {
    grid-template-columns: 1fr;
  }
}
</style>
