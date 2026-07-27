<script setup lang="ts">
import { computed } from "vue";
import type { JobStatus } from "../types/backup";
import { formatBytes } from "../utils/size";

const props = defineProps<{
  status: JobStatus;
  phaseLabel: string;
  elapsedText: string;
  remainingText: string;
  logs: string[];
}>();

defineEmits<{
  start: [];
  stop: [];
  refresh: [];
  init: [];
  reset: [];
  openSettings: [];
}>();

const progressPercent = computed(() => {
  if (!props.status.totalFiles) return 0;
  return Math.min(100, Math.round((props.status.packedFiles / props.status.totalFiles) * 100));
});

const statusLabel = computed(() => {
  if (props.status.running) return "运行中";
  if (props.status.done) return "已完成";
  if (props.status.remoteInited) return "已 init";
  return "就绪";
});

const statusTagColor = computed(() => {
  if (props.status.running) return "processing";
  if (props.status.done) return "success";
  if (props.status.remoteInited) return "cyan";
  return "default";
});

const logText = computed(() => props.logs.join("\n") || "等待任务输出…");

const maxFileText = computed(() => {
  if (!props.status.remoteInited) return "-";
  const n = props.status.maxFileBytes ?? 0;
  if (props.status.totalFiles > 0 && n <= 0) return "未知";
  if (n <= 0) return "-";
  return formatBytes(n);
});

const oversizedCountText = computed(() => {
  if (!props.status.remoteInited) return "-";
  const maxB = props.status.maxFileBytes ?? 0;
  if (props.status.totalFiles > 0 && maxB <= 0) return "未知";
  const c = props.status.oversizedFileCount ?? 0;
  return String(c);
});

const startBackupLabel = computed(() => {
  if (props.status.running) return "开始备份";
  if (!props.status.remoteInited || props.status.done) return "开始备份";
  if (props.status.pendingZip?.trim()) return "继续备份";
  if (props.status.totalFiles > 0 && props.status.packedFiles > 0) return "继续备份";
  return "开始备份";
});
</script>

<template>
  <section class="panel-card panel-card--grow">
    <div class="panel-card-header panel-card-header--with-actions">
      <span>备份任务</span>
      <div class="panel-card-header-actions">
        <a-button type="default" size="small" @click="$emit('openSettings')">设置</a-button>
        <a-button type="default" size="small" :disabled="status.running" @click="$emit('init')">远程 init</a-button>
      </div>
    </div>
    <div class="panel-card-body space-y-3">
      <div class="grid grid-cols-2 lg:grid-cols-4 xl:grid-cols-8 gap-3">
        <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2">
          <div class="text-xs text-gray-500 mb-1">状态</div>
          <a-tag :color="statusTagColor">
            {{ statusLabel }}
          </a-tag>
        </div>
        <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2">
          <div class="text-xs text-gray-500 mb-1">阶段</div>
          <div class="text-sm font-medium text-emerald-900">{{ phaseLabel }}</div>
        </div>
        <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2">
          <div class="text-xs text-gray-500 mb-1">文件进度</div>
          <div class="text-sm font-medium">{{ status.packedFiles }} / {{ status.totalFiles || "?" }}</div>
        </div>
        <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2">
          <div class="text-xs text-gray-500 mb-1">已用时</div>
          <div class="text-sm font-medium tabular-nums">{{ elapsedText }}</div>
        </div>
        <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2">
          <div class="text-xs text-gray-500 mb-1">剩余用时（估）</div>
          <div class="text-sm font-medium tabular-nums">{{ remainingText }}</div>
        </div>
        <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2 min-w-0">
          <div class="text-xs text-gray-500 mb-1">最大文件</div>
          <div class="text-sm font-medium tabular-nums truncate" :title="maxFileText">{{ maxFileText }}</div>
        </div>
        <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2">
          <div class="text-xs text-gray-500 mb-1">超分卷数</div>
          <div
            class="text-sm font-medium tabular-nums"
            :class="(status.oversizedFileCount ?? 0) > 0 ? 'text-amber-700' : ''"
          >
            {{ oversizedCountText }}
          </div>
        </div>
        <div class="rounded-lg border border-emerald-100/80 bg-white/60 px-3 py-2">
          <div class="text-xs text-gray-500 mb-1">当前分卷</div>
          <div class="text-sm font-medium truncate" :title="status.currentPart">{{ status.currentPart || "-" }}</div>
        </div>
      </div>

      <a-alert
        v-if="status.remoteHint && !status.running"
        type="warning"
        :message="status.remoteHint"
        show-icon
        class="text-xs"
      />
      <a-alert
        v-if="status.pendingZip?.trim()"
        type="warning"
        :message="`远程待处理分卷：${status.pendingZip}`"
        show-icon
        class="text-xs"
      />

      <a-progress :percent="progressPercent" size="small" status="active" />

      <div class="flex flex-wrap gap-2">
        <a-button type="primary" :disabled="status.running" @click="$emit('start')">{{ startBackupLabel }}</a-button>
        <a-button danger :disabled="!status.running" @click="$emit('stop')">停止</a-button>
        <a-button @click="$emit('refresh')">刷新状态</a-button>
        <a-popconfirm
          title="重置远程备份进度？保留文件清单，下次将从第一个文件重新打包。不会删除本机已下载文件。"
          ok-text="重置"
          cancel-text="取消"
          :disabled="status.running || !status.remoteInited"
          @confirm="$emit('reset')"
        >
          <a-button :disabled="status.running || !status.remoteInited">重置任务</a-button>
        </a-popconfirm>
      </div>

      <a-alert v-if="status.lastError" type="error" :message="status.lastError" show-icon class="text-xs" />

      <div class="log-box">{{ logText }}</div>
    </div>
  </section>
</template>
