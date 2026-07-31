<script setup lang="ts">
import type { BackupTask } from "../types/backup";
import BackupTaskList from "./BackupTaskList.vue";
import { singleFileBackupPanelProps, useSingleFileBackupPanel } from "./SingleFileBackupPanel";

const props = defineProps(singleFileBackupPanelProps);
const emit = defineEmits<{
  addTask: [];
  editTask: [task: BackupTask];
}>();

const {
  tasks,
  activeTaskId,
  status,
  autoScrollLog,
  logBoxRef,
  progressPercent,
  progressText,
  progressFormat,
  statusLabel,
  statusTagColor,
  logText,
  startDisabled,
  stopDisabled,
  selectTask,
  removeTask,
  onAddTask,
  onEditTask,
  onStart,
  onStop,
} = useSingleFileBackupPanel(props, emit);
</script>

<template>
  <section class="panel-card panel-card--grow single-file-backup-panel">
    <div class="single-file-backup-panel__content">
      <div class="panel-card-header panel-card-header--with-actions">
        <span>单文件备份</span>
      </div>

      <div class="panel-card-body single-file-backup-panel__body">
        <BackupTaskList
          variant="single"
          :tasks="tasks"
          :active-task-id="activeTaskId"
          :disabled="status.running"
          @add="onAddTask"
          @edit="onEditTask"
          @select="selectTask"
          @remove="removeTask"
        />

        <div class="single-file-backup-panel__main">
          <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
            <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2">
              <div class="text-xs text-gray-500 mb-1">状态</div>
              <a-tag :color="statusTagColor">{{ statusLabel }}</a-tag>
            </div>
            <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2 min-w-0">
              <div class="text-xs text-gray-500 mb-1">本机文件</div>
              <div class="text-sm font-medium truncate" :title="status.localFile || '-'">
                {{ status.localFile || "-" }}
              </div>
            </div>
            <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2 col-span-2 min-w-0">
              <div class="text-xs text-gray-500 mb-1">传输</div>
              <div class="text-sm font-medium tabular-nums truncate" :title="progressText || '-'">
                {{ progressText || "-" }}
              </div>
            </div>
          </div>

          <a-progress
            class="panel-progress"
            :percent="progressPercent"
            :status="status.running ? 'active' : undefined"
            :format="progressFormat"
          />

          <div class="flex flex-wrap gap-2 items-center">
            <a-button type="primary" :disabled="startDisabled" @click="onStart">开始下载</a-button>
            <a-button danger :disabled="stopDisabled" @click="onStop">停止</a-button>
            <div class="backup-toolbar-trailing">
              <a-checkbox v-model:checked="autoScrollLog">自动滚动</a-checkbox>
            </div>
          </div>

          <a-alert
            v-if="disabledByOtherJob && !status.running"
            type="warning"
            message="多文件备份正在运行，请等待结束后再开始单文件下载"
            show-icon
            class="text-xs"
          />
          <a-alert v-if="status.lastError" type="error" :message="status.lastError" show-icon class="text-xs" />
        </div>

        <div class="log-box-wrap">
          <div ref="logBoxRef" class="log-box">{{ logText }}</div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.single-file-backup-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.single-file-backup-panel__content {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.single-file-backup-panel__content > .panel-card-header {
  flex-shrink: 0;
}

.single-file-backup-panel__body {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
}

.single-file-backup-panel__main {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
