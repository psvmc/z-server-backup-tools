<script setup lang="ts">
const model = defineModel<string>({ required: true });

defineEmits<{
  browse: [];
}>();
</script>

<template>
  <div class="path-pick-row">
    <a-input
      :value="model"
      readonly
      placeholder="点击「选择」打开目录"
      class="path-pick-row__input"
      @click="$emit('browse')"
    />
    <a-button type="default" @click="$emit('browse')">选择</a-button>
  </div>
</template>

<style scoped>
.path-pick-row {
  display: flex;
  gap: 8px;
  width: 100%;
}

.path-pick-row__input {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}
</style>
