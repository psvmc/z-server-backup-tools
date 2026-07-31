<script setup lang="ts">
import PathPickInput from "./PathPickInput.vue";
import { singleFileBackupPanelProps, useSingleFileBackupPanel } from "./SingleFileBackupPanel";

const props = defineProps(singleFileBackupPanelProps);
const {
  paths,
  status,
  saving,
  autoScrollLog,
  logBoxRef,
  serverOptions,
  progressPercent,
  progressText,
  progressFormat,
  statusLabel,
  statusTagColor,
  logText,
  startDisabled,
  stopDisabled,
  onRemoteBrowse,
  pickLocalDir,
  openLocalFolder,
  onSave,
  onStart,
  onStop,
} = useSingleFileBackupPanel(props);
</script>

<template>
  <section class="panel-card panel-card--grow single-file-backup-panel">
    <div class="single-file-backup-panel__content">
      <div class="panel-card-header panel-card-header--with-actions">
        <span>单文件备份</span>
      </div>

      <div class="panel-card-body single-file-backup-panel__body">
        <div class="single-file-backup-panel__main">
          <a-form layout="vertical" size="small" class="single-file-backup-panel__form">
            <a-form-item label="服务器" required>
              <a-select
                v-model:value="paths.server_id"
                :options="serverOptions"
                placeholder="选择服务器"
                allow-clear
                show-search
                option-filter-prop="label"
                :disabled="status.running"
              />
            </a-form-item>
            <a-form-item label="远程源文件">
              <PathPickInput
                v-model="paths.remote_file"
                editable
                placeholder="D:\data\app.bak"
                @browse="onRemoteBrowse"
              />
            </a-form-item>
            <a-form-item label="本机保存目录">
              <PathPickInput
                v-model="paths.local_dir"
                show-open-folder
                @browse="pickLocalDir"
                @open-folder="openLocalFolder"
              />
            </a-form-item>
          </a-form>

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
            :percent="progressPercent"
            size="small"
            :status="status.running ? 'active' : undefined"
            :format="progressFormat"
          />

          <div class="flex flex-wrap gap-2 items-center">
            <a-button type="default" :loading="saving" :disabled="status.running" @click="onSave">
              保存路径
            </a-button>
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

.single-file-backup-panel__form {
  margin-bottom: 0;
}

.single-file-backup-panel__form :deep(.ant-form-item) {
  margin-bottom: 12px;
}

.single-file-backup-panel__form :deep(.ant-form-item:last-child) {
  margin-bottom: 0;
}
</style>
