<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { message } from "ant-design-vue";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import { BackupConfig as BackupConfigBinding } from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig } from "../types/backup";
import { formatError } from "../types/update";

const open = defineModel<boolean>("open", { required: true });

const props = defineProps<{
  connection: BackupConfig;
  initialPath?: string;
  title?: string;
}>();

const emit = defineEmits<{
  select: [path: string];
}>();

const loading = ref(false);
const currentPath = ref("");
const parentPath = ref("");
const entries = ref<{ name: string; path: string }[]>([]);
const jumpPath = ref("");

const atDriveList = computed(() => !currentPath.value.trim());

function toBinding(cfg: BackupConfig) {
  return BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(cfg)));
}

async function load(pathHint: string) {
  loading.value = true;
  try {
    const listing = await BackupService.ListRemoteDirectories(toBinding(props.connection), pathHint);
    currentPath.value = listing.current_path ?? "";
    parentPath.value = listing.parent_path ?? "";
    entries.value = (listing.entries ?? []).map((e) => ({
      name: e.name ?? "",
      path: e.path ?? "",
    }));
    jumpPath.value = currentPath.value;
  } catch (err) {
    message.error(formatError(err));
  } finally {
    loading.value = false;
  }
}

watch(
  () => open.value,
  (visible) => {
    if (visible) {
      void load(props.initialPath?.trim() || "");
    }
  },
);

function enterDir(path: string) {
  void load(path);
}

function goParent() {
  if (atDriveList.value) return;
  void load(parentPath.value || "");
}

function goDriveList() {
  void load("");
}

function jumpToPath() {
  const target = jumpPath.value.trim();
  if (!target) {
    void load("");
    return;
  }
  void load(target);
}

function confirmCurrent() {
  const picked = currentPath.value.trim();
  if (!picked) {
    message.warning(atDriveList.value ? "请先选择盘符并进入目录" : "请先进入要选中的目录");
    return;
  }
  emit("select", picked);
  open.value = false;
}

function confirmEntry(path: string) {
  emit("select", path);
  open.value = false;
}
</script>

<template>
  <a-modal
    v-model:open="open"
    :title="title ?? '选择远程目录'"
    width="560px"
    wrap-class-name="remote-dir-picker-modal"
    :footer="null"
    destroy-on-close
    centered
  >
    <div class="remote-dir-picker">
      <div class="flex flex-wrap items-center gap-2 text-xs text-gray-600 shrink-0">
        <span class="font-medium text-emerald-900">当前：</span>
        <span class="break-all">{{ atDriveList ? "（盘符列表）" : currentPath }}</span>
      </div>

      <div class="flex gap-2 shrink-0">
        <a-input
          v-model:value="jumpPath"
          size="small"
          placeholder="手动输入路径，如 D:\Tools\zipbak"
          @press-enter="jumpToPath"
        />
        <a-button size="small" :loading="loading" @click="jumpToPath">跳转</a-button>
      </div>

      <div class="flex flex-wrap gap-2 shrink-0">
        <a-button size="small" :disabled="loading || atDriveList" @click="goParent">上级目录</a-button>
        <a-button size="small" :disabled="loading || atDriveList" @click="goDriveList">盘符列表</a-button>
        <a-button size="small" type="primary" :loading="loading" @click="confirmCurrent">选中当前目录</a-button>
      </div>

      <a-spin :spinning="loading" class="remote-dir-picker__spin">
        <div class="remote-dir-list">
          <button
            v-for="item in entries"
            :key="item.path"
            type="button"
            class="remote-dir-list__item"
            @dblclick="enterDir(item.path)"
            @click="enterDir(item.path)"
          >
            <span class="remote-dir-list__name">{{ item.name }}</span>
            <a-button
              v-if="!atDriveList"
              type="link"
              size="small"
              class="shrink-0"
              @click.stop="confirmEntry(item.path)"
            >
              选中
            </a-button>
          </button>
          <div v-if="!loading && entries.length === 0" class="text-xs text-gray-400 py-6 text-center">
            {{ atDriveList ? "未发现可用盘符" : "此目录下没有子文件夹" }}
          </div>
        </div>
      </a-spin>

      <p class="text-xs text-gray-400 mb-0 shrink-0">
        {{ atDriveList ? "点击盘符进入；也可在上方手动输入完整路径。" : "单击文件夹进入；点「选中」确认路径。" }}
      </p>
    </div>
  </a-modal>
</template>

<style scoped>
.remote-dir-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.remote-dir-picker__spin {
  flex: 1 1 0;
  min-height: 0;
  display: flex !important;
  flex-direction: column;
}

.remote-dir-picker__spin :deep(.ant-spin-container) {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.remote-dir-list {
  flex: 1 1 0;
  min-height: 0;
  overflow: auto;
  border: 1px solid #e2eee8;
  border-radius: 8px;
  background: #f7fbf9;
}

.remote-dir-list__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 8px 12px;
  border: none;
  border-bottom: 1px solid #eaf3ef;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.remote-dir-list__item:hover {
  background: rgba(91, 184, 168, 0.08);
}

.remote-dir-list__name {
  font-size: 13px;
  color: #2c3e50;
  word-break: break-all;
}
</style>
