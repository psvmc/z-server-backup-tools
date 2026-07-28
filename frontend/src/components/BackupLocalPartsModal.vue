<script setup lang="ts">
import { computed, ref, watch } from "vue";
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

const localDirText = computed(() => listing.value?.localDir?.trim() || props.config.local_dir?.trim() || "（未设置）");

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
    :footer="null"
    destroy-on-close
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-3">
      <div class="text-xs text-gray-500 break-all">目录：{{ localDirText }}</div>
      <a-alert v-if="error" type="error" :message="error" show-icon class="text-xs" />
      <a-table
        size="small"
        :loading="loading"
        :columns="columns"
        :data-source="listing?.files ?? []"
        :pagination="false"
        row-key="path"
        :locale="{ emptyText: loading ? '加载中…' : '暂无分卷文件' }"
        :scroll="{ y: 360 }"
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
      <div class="flex justify-end gap-2">
        <a-button @click="emit('update:open', false)">关闭</a-button>
        <a-button type="primary" :loading="loading" @click="load">刷新</a-button>
      </div>
    </div>
  </a-modal>
</template>
