<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { message } from "ant-design-vue";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import { BackupConfig as BackupConfigBinding } from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig, LocalPartFile, LocalPartListing } from "../types/backup";
import { formatBytes } from "../utils/size";
import { formatError } from "../types/update";

const props = defineProps<{
  open: boolean;
  config: BackupConfig;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
}>();

const loading = ref(false);
const error = ref("");
const listing = ref<LocalPartListing | null>(null);

const localDir = computed(() => listing.value?.localDir?.trim() || props.config.local_dir?.trim() || "");
const localDirText = computed(() => localDir.value || "（未设置）");

const columns = [
  { title: "分卷", dataIndex: "name", key: "name", ellipsis: true },
  { title: "大小", dataIndex: "sizeBytes", key: "sizeBytes", width: 110 },
  { title: "状态", dataIndex: "state", key: "state", width: 100 },
];

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const cfg = BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(props.config)));
    listing.value = (await BackupService.ListLocalParts(cfg)) as LocalPartListing;
  } catch (e) {
    error.value = formatError(e);
    listing.value = null;
  } finally {
    loading.value = false;
  }
}

async function openFolder() {
  const target = localDir.value;
  if (!target) {
    message.warning("请先设置本机保存目录");
    return;
  }
  try {
    await BackupService.OpenInExplorer(target);
  } catch (err) {
    message.error(formatError(err));
  }
}

function stateLabel(state: LocalPartFile["state"]) {
  return state === "downloading" ? "下载中" : "已下载";
}

function stateColor(state: LocalPartFile["state"]) {
  return state === "downloading" ? "processing" : "success";
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      void load();
    }
  },
);
</script>

<template>
  <a-modal
    :open="open"
    title="任务查看"
    width="640px"
    wrap-class-name="local-parts-modal"
    :footer="null"
    destroy-on-close
    centered
    @update:open="emit('update:open', $event)"
  >
    <div class="local-parts-panel">
      <div class="local-parts-dir">
        <div class="local-parts-dir__text text-xs text-gray-500" :title="localDirText">
          目录：{{ localDirText }}
        </div>
        <a-button size="small" :disabled="!localDir" @click="openFolder">打开文件夹</a-button>
      </div>

      <a-alert v-if="error" type="error" :message="error" show-icon class="text-xs shrink-0" />

      <div class="local-parts-table-wrap">
        <a-table
          size="small"
          :loading="loading"
          :columns="columns"
          :data-source="listing?.files ?? []"
          :pagination="false"
          row-key="path"
          :locale="{ emptyText: loading ? '加载中…' : '暂无分卷文件' }"
          :scroll="{ y: '100%' }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'sizeBytes'">
              <span class="tabular-nums">{{ formatBytes(record.sizeBytes) }}</span>
            </template>
            <template v-else-if="column.key === 'state'">
              <a-tag :color="stateColor(record.state)">{{ stateLabel(record.state) }}</a-tag>
            </template>
          </template>
        </a-table>
      </div>

      <div class="flex justify-end gap-2 shrink-0">
        <a-button @click="emit('update:open', false)">关闭</a-button>
        <a-button type="primary" :loading="loading" @click="load">刷新</a-button>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.local-parts-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.local-parts-dir {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.local-parts-dir__text {
  flex: 1;
  min-width: 0;
  word-break: break-all;
  line-height: 1.4;
}

.local-parts-table-wrap {
  flex: 1 1 0;
  min-height: 0;
  overflow: hidden;
}

.local-parts-table-wrap :deep(.ant-table-wrapper),
.local-parts-table-wrap :deep(.ant-spin-nested-loading),
.local-parts-table-wrap :deep(.ant-spin-container),
.local-parts-table-wrap :deep(.ant-table),
.local-parts-table-wrap :deep(.ant-table-container) {
  height: 100%;
}

.local-parts-table-wrap :deep(.ant-table-body) {
  max-height: calc(80vh - 220px) !important;
  overflow-y: auto !important;
}
</style>
