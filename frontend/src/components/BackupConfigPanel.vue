<script setup lang="ts">
import { computed, ref } from "vue";
import { message } from "ant-design-vue";
import { Dialogs } from "@wailsio/runtime";
import type { BackupConfig } from "../types/backup";
import { remotePathsFromAppDir } from "../types/backup";
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
const remotePickerTarget = ref<"remote_app_dir" | "remote_source">("remote_app_dir");
const remotePickerTitle = ref("选择远程目录");

const remoteInitialPath = computed(() =>
  remotePickerTarget.value === "remote_app_dir"
    ? config.value.remote_app_dir
    : config.value.remote_source,
);

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

function openRemotePicker(target: "remote_app_dir" | "remote_source", title: string) {
  if (props.saving) return;
  if (!ensureConnectionFilled()) return;
  remotePickerTarget.value = target;
  remotePickerTitle.value = title;
  remotePickerOpen.value = true;
}

function onRemoteSelect(path: string) {
  if (remotePickerTarget.value === "remote_app_dir") {
    config.value.remote_app_dir = path;
  } else {
    config.value.remote_source = path;
  }
}

async function pickLocalDir() {
  if (props.saving) return;
  try {
    const picked = await Dialogs.OpenFile({
      Title: "选择本机保存目录",
      CanChooseDirectories: true,
      CanChooseFiles: false,
      Directory: config.value.local_dir || undefined,
    });
    const path = Array.isArray(picked) ? picked[0] : picked;
    if (path && typeof path === "string") {
      config.value.local_dir = path;
    }
  } catch (err) {
    message.error(String(err));
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
      <div class="settings-block-title">备份路径与分卷</div>
      <a-form layout="vertical" size="small" class="w-full">
        <div class="config-form-grid">
          <a-form-item label="远程应用目录" class="config-form-span-2">
            <PathPickInput
              v-model="config.remote_app_dir"
              @browse="openRemotePicker('remote_app_dir', '选择远程应用目录')"
            />
          </a-form-item>
          <a-form-item label="远程源目录 (--dir)" class="config-form-span-2">
            <PathPickInput
              v-model="config.remote_source"
              @browse="openRemotePicker('remote_source', '选择远程源目录')"
            />
          </a-form-item>

          <div
            v-if="config.remote_app_dir?.trim()"
            class="config-form-span-4 settings-derived-paths"
          >
            <div class="settings-derived-paths-label">自动推导路径</div>
            <dl class="settings-derived-paths-list">
              <div><dt>zipbak-srv</dt><dd>{{ derivedPaths.srv }}</dd></div>
              <div><dt>state</dt><dd>{{ derivedPaths.state }}</dd></div>
              <div><dt>staging</dt><dd>{{ derivedPaths.staging }}</dd></div>
            </dl>
          </div>

          <a-form-item label="本机保存目录" class="config-form-span-2">
            <PathPickInput v-model="config.local_dir" @browse="pickLocalDir" />
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
        </div>
      </a-form>
    </section>

    <RemoteDirPickerModal
      v-model:open="remotePickerOpen"
      :connection="config"
      :initial-path="remoteInitialPath"
      :title="remotePickerTitle"
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

@media (max-width: 720px) {
  .config-form-grid--conn {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .settings-derived-paths-list {
    grid-template-columns: 1fr;
  }
}
</style>
