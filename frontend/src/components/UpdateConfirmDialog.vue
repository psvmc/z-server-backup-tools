<script setup lang="ts">
import { computed } from "vue";
import { useUpdateConfirm } from "../composables/useUpdateConfirm";

const { state, confirm, cancel, skip } = useUpdateConfirm();
const result = computed(() => state.result);

const versionSummary = computed(() => {
  const item = result.value;
  if (!item) return "";
  return `v${item.currentVersion} → v${item.latestVersion || "新版本"}`;
});
</script>

<template>
  <a-modal
    v-model:open="state.show"
    title="发现新版本"
    :width="520"
    :mask-closable="false"
    @cancel="cancel"
  >
    <div v-if="result" class="space-y-3">
      <div class="font-medium">{{ versionSummary }}</div>
      <pre v-if="result.notes" class="text-xs bg-gray-50 p-2 rounded max-h-40 overflow-auto">{{ result.notes }}</pre>
      <p class="text-sm text-gray-600 m-0">是否立即下载并安装？</p>
    </div>
    <template #footer>
      <a-button @click="skip">跳过该版本</a-button>
      <a-button @click="cancel">稍后</a-button>
      <a-button type="primary" @click="confirm">立即更新</a-button>
    </template>
  </a-modal>
</template>
