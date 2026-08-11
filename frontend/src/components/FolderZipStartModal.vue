<script setup lang="ts">
import {
  folderZipStartModalProps,
  useFolderZipStartModal,
} from "./FolderZipStartModal";

const open = defineModel<boolean>("open", { required: true });

const props = defineProps(folderZipStartModalProps);

const emit = defineEmits<{
  "update:open": [value: boolean];
  confirm: [taskIds: string[]];
}>();

const {
  selectedIds,
  rows,
  runnableIds,
  allChecked,
  indeterminate,
  toggleAll,
  onCancel,
  onConfirm,
} = useFolderZipStartModal(open, props, emit);
</script>

<template>
  <a-modal
    v-model:open="open"
    title="选择要备份的任务"
    width="640px"
    :mask-closable="false"
    centered
    @cancel="onCancel"
  >
    <div v-if="runnableIds.length === 0" class="folder-zip-start-modal__empty">
      <a-empty description="没有可执行的任务，请先完善任务配置" />
    </div>
    <template v-else>
      <div class="folder-zip-start-modal__toolbar">
        <a-checkbox
          :checked="allChecked"
          :indeterminate="indeterminate"
          @change="(e) => toggleAll(!!e.target.checked)"
        >
          全选
        </a-checkbox>
        <span class="folder-zip-start-modal__hint">已选 {{ selectedIds.length }} / {{ runnableIds.length }}</span>
      </div>
      <a-checkbox-group v-model:value="selectedIds" class="folder-zip-start-modal__list">
        <div v-for="row in rows" :key="row.id" class="folder-zip-start-modal__item">
          <a-checkbox :value="row.id" :disabled="!row.runnable">
            <div class="folder-zip-start-modal__item-main">
              <div class="folder-zip-start-modal__name">{{ row.label }}</div>
              <div class="folder-zip-start-modal__path" :title="row.remote_source">
                {{ row.remote_source || "-" }}
              </div>
            </div>
          </a-checkbox>
          <a-tag v-if="!row.runnable" color="warning" class="folder-zip-start-modal__tag">配置不完整</a-tag>
        </div>
      </a-checkbox-group>
    </template>

    <template #footer>
      <a-button @click="onCancel">取消</a-button>
      <a-button type="primary" :disabled="runnableIds.length === 0" @click="onConfirm">
        开始备份
      </a-button>
    </template>
  </a-modal>
</template>

<style scoped>
.folder-zip-start-modal__empty {
  padding: 12px 0;
}

.folder-zip-start-modal__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.folder-zip-start-modal__hint {
  font-size: 12px;
  color: #6b7280;
}

.folder-zip-start-modal__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  max-height: 360px;
  overflow-y: auto;
}

.folder-zip-start-modal__item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid #dbeafe;
  border-radius: 8px;
  background: #f8fbff;
}

.folder-zip-start-modal__item :deep(.ant-checkbox-wrapper) {
  align-items: flex-start;
  flex: 1;
  min-width: 0;
}

.folder-zip-start-modal__item-main {
  min-width: 0;
}

.folder-zip-start-modal__name {
  font-size: 13px;
  font-weight: 600;
  color: #1f2937;
}

.folder-zip-start-modal__path {
  margin-top: 2px;
  font-size: 12px;
  color: #6b7280;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.folder-zip-start-modal__tag {
  flex: 0 0 auto;
  margin-top: 2px;
}
</style>
