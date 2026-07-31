<script setup lang="ts">
import { computed } from "vue";
import type { BackupTask } from "../types/backup";
import { taskDisplayName } from "../types/backup";

const props = defineProps<{
  tasks: BackupTask[];
  activeTaskId: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  select: [id: string];
  add: [];
  edit: [task: BackupTask];
  remove: [task: BackupTask];
}>();

const columns = [
  { title: "任务", dataIndex: "name", key: "name", width: 100, ellipsis: true },
  { title: "远程源目录", dataIndex: "remote_source", key: "remote_source", width: 280, ellipsis: true },
  { title: "本机目录", dataIndex: "local_dir", key: "local_dir", ellipsis: true },
  { title: "前缀", dataIndex: "part_name_prefix", key: "part_name_prefix", width: 100, ellipsis: true },
  { title: "操作", key: "actions", width: 160 },
];

const rows = computed(() =>
  props.tasks.map((task) => ({
    ...task,
    key: task.id,
    name: taskDisplayName(task),
    part_name_prefix: task.part_name_prefix?.trim() || "-",
  })),
);
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
      :row-class-name="(record) => (record.id === activeTaskId ? 'backup-task-list__row--active' : '')"
      class="backup-task-list__table"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'actions'">
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
  border: 1px solid #dcebe4;
  border-radius: 10px;
  background: linear-gradient(180deg, #fafdfb 0%, #ffffff 100%);
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
  color: #1a4d42;
}

.backup-task-list__empty {
  margin: 8px 0 4px;
}

.backup-task-list__table :deep(.backup-task-list__row--active > td) {
  background: #eef9f4 !important;
}
</style>
