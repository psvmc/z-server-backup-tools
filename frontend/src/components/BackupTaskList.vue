<script setup lang="ts">
import type { BackupTask } from "../types/backup";
import { backupTaskListProps, useBackupTaskList } from "./BackupTaskList";

const props = defineProps(backupTaskListProps);
defineEmits<{
  select: [id: string];
  add: [];
  edit: [task: BackupTask];
  remove: [task: BackupTask];
}>();

const { columns, rows, scrollX, isEllipsisColumnKey, ellipsisCellText } = useBackupTaskList(props);
</script>

<template>
  <div class="backup-task-list">
    <div class="backup-task-list__header">
      <span class="backup-task-list__title">备份任务</span>
      <a-button type="primary" size="small" :disabled="disabled" @click="$emit('add')">添加任务</a-button>
    </div>

    <a-table
      v-if="tasks.length > 0"
      size="small"
      :columns="columns"
      :data-source="rows"
      :pagination="false"
      :scroll="{ x: scrollX }"
      :row-class-name="(record) => (record.id === activeTaskId ? 'backup-task-list__row--active' : '')"
      class="backup-task-list__table"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="isEllipsisColumnKey(column.key)">
          <a-tooltip
            v-if="ellipsisCellText(record, column.key)"
            :title="ellipsisCellText(record, column.key)"
            placement="topLeft"
          >
            <span class="backup-task-list__path">{{ ellipsisCellText(record, column.key) }}</span>
          </a-tooltip>
          <span v-else class="backup-task-list__path backup-task-list__path--empty">-</span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <a-space :size="4">
            <a-button
              type="link"
              size="small"
              :disabled="disabled || record.id === activeTaskId"
              @click="$emit('select', record.id)"
            >
              {{ record.id === activeTaskId ? "当前" : "选择" }}
            </a-button>
            <a-button type="link" size="small" :disabled="disabled" @click="$emit('edit', record)">编辑</a-button>
            <a-popconfirm
              title="确定删除该任务？不会删除已下载文件。"
              ok-text="删除"
              cancel-text="取消"
              :disabled="disabled"
              @confirm="$emit('remove', record)"
            >
              <a-button type="link" size="small" danger :disabled="disabled">删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-empty v-else description="暂无任务，点击「添加任务」创建" class="backup-task-list__empty" />
  </div>
</template>

<style scoped>
.backup-task-list {
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: linear-gradient(180deg, var(--app-surface-card) 0%, #ffffff 100%);
  padding: 10px 12px;
}

.backup-task-list__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.backup-task-list__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-primary-dark);
}

.backup-task-list__empty {
  margin: 8px 0 4px;
}

.backup-task-list__table :deep(.backup-task-list__row--active > td) {
  background: #eef4fc !important;
}

.backup-task-list__path {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  cursor: default;
}

.backup-task-list__path--empty {
  color: var(--app-text-muted, #9ca3af);
}
</style>
