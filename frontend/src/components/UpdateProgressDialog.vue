<script setup lang="ts">
import { computed } from "vue";
import type { UpdateProgressState } from "../composables/useUpdateProgress";

const progress = defineModel<UpdateProgressState | null>("progress", { default: null });

const show = computed({
  get: () => progress.value !== null,
  set: (visible) => {
    if (!visible) progress.value = null;
  },
});

const percentage = computed(() => {
  const state = progress.value;
  if (!state || state.total <= 0) return state?.phase === "verifying" ? 100 : 0;
  return Math.min(100, Math.round((state.written / state.total) * 100));
});
</script>

<template>
  <a-modal
    v-model:open="show"
    title="正在下载更新"
    :width="360"
    :mask-closable="false"
    :closable="false"
    :footer="null"
  >
    <a-progress :percent="percentage" />
  </a-modal>
</template>
