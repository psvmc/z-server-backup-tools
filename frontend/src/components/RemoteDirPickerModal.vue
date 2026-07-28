<script setup lang="ts">
import { ref, watch } from "vue";
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
      void load(props.initialPath?.trim() || "D:/");
    }
  },
);

function enterDir(path: string) {
  void load(path);
}

function goParent() {
  if (parentPath.value !== undefined) {
    void load(parentPath.value);
  }
}

function confirmCurrent() {
  const picked = currentPath.value.trim();
  if (!picked) {
    message.warning("请先进入要选中的目录");
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
    width="520px"
    :footer="null"
    destroy-on-close
  >
    <div class="space-y-3">
      <div class="flex flex-wrap items-center gap-2 text-xs text-gray-600">
        <span class="font-medium text-emerald-900">当前：</span>
        <span class="break-all">{{ currentPath || "（根目录）" }}</span>
      </div>

      <div class="flex flex-wrap gap-2">
        <a-button size="small" :disabled="loading || !parentPath" @click="goParent">上级目录</a-button>
        <a-button size="small" type="primary" :loading="loading" @click="confirmCurrent">选中当前目录</a-button>
      </div>

      <a-spin :spinning="loading">
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
            <a-button type="link" size="small" class="shrink-0" @click.stop="confirmEntry(item.path)">选中</a-button>
          </button>
          <div v-if="!loading && entries.length === 0" class="text-xs text-gray-400 py-6 text-center">
            此目录下没有子文件夹
          </div>
        </div>
      </a-spin>

      <p class="text-xs text-gray-400 mb-0">单击文件夹进入；双击或点「选中」确认路径。</p>
    </div>
  </a-modal>
</template>

<style scoped>
.remote-dir-list {
  max-height: 280px;
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
