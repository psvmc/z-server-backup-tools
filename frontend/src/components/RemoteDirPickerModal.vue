<script setup lang="ts">
import type { BackupConfig } from "../types/backup";
import { useRemoteDirPicker, type RemotePickerMode } from "./RemoteDirPickerModal";

const open = defineModel<boolean>("open", { required: true });

const props = withDefaults(
  defineProps<{
    connection: BackupConfig;
    initialPath?: string;
    title?: string;
    mode?: RemotePickerMode;
  }>(),
  {
    mode: "directory",
  },
);

const emit = defineEmits<{
  select: [path: string];
}>();

const {
  loading,
  entries,
  jumpPath,
  atRoot,
  isFileMode,
  currentLabel,
  jumpPlaceholder,
  rootButtonLabel,
  modalTitle,
  hintText,
  emptyText,
  onItemActivate,
  goParent,
  goRoot,
  jumpToPath,
  confirmCurrent,
  confirmEntry,
  showSelectButton,
} = useRemoteDirPicker(open, props, emit);
</script>

<template>
  <a-modal
    v-model:open="open"
    :title="modalTitle"
    width="560px"
    wrap-class-name="remote-dir-picker-modal"
    :footer="null"
    destroy-on-close
    centered
  >
    <div class="remote-dir-picker">
      <div class="flex flex-wrap items-center gap-2 text-xs text-gray-600 shrink-0">
        <span class="font-medium text-blue-900">当前：</span>
        <span class="break-all">{{ currentLabel }}</span>
      </div>

      <div class="flex gap-2 shrink-0">
        <a-input
          v-model:value="jumpPath"
          size="small"
          :placeholder="jumpPlaceholder"
          @press-enter="jumpToPath"
        />
        <a-button size="small" :loading="loading" @click="jumpToPath">跳转</a-button>
      </div>

      <div class="flex flex-wrap gap-2 shrink-0">
        <a-button size="small" :disabled="loading || atRoot" @click="goParent">上级目录</a-button>
        <a-button size="small" :disabled="loading || atRoot" @click="goRoot">{{ rootButtonLabel }}</a-button>
        <a-button
          v-if="!isFileMode"
          size="small"
          type="primary"
          :loading="loading"
          @click="confirmCurrent"
        >
          选中当前目录
        </a-button>
      </div>

      <a-spin :spinning="loading" class="remote-dir-picker__spin">
        <div class="remote-dir-list">
          <button
            v-for="item in entries"
            :key="item.path"
            type="button"
            class="remote-dir-list__item"
            @dblclick="onItemActivate(item)"
            @click="onItemActivate(item)"
          >
            <span class="remote-dir-list__name">{{ item.name }}</span>
            <a-button
              v-if="showSelectButton(item)"
              type="link"
              size="small"
              class="shrink-0"
              @click.stop="confirmEntry(item.path)"
            >
              选中
            </a-button>
          </button>
          <div v-if="!loading && entries.length === 0" class="text-xs text-gray-400 py-6 text-center">
            {{ emptyText }}
          </div>
        </div>
      </a-spin>

      <p class="text-xs text-gray-400 mb-0 shrink-0">
        {{ hintText }}
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
  border-bottom: 1px solid var(--app-border-light);
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
