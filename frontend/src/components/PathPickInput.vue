<script setup lang="ts">
const model = defineModel<string>({ required: true });

const props = defineProps<{
  showOpenFolder?: boolean;
  editable?: boolean;
  placeholder?: string;
}>();

defineEmits<{
  browse: [];
  openFolder: [];
}>();

function onInput(value: string) {
  if (props.editable) {
    model.value = value;
  }
}
</script>

<template>
  <div class="path-pick-row">
    <a-input
      :value="model"
      :readonly="!props.editable"
      :placeholder="props.placeholder || (props.editable ? '可手动输入，或点「选择」浏览' : '点击「选择」打开目录')"
      class="path-pick-row__input"
      :class="{ 'path-pick-row__input--editable': props.editable }"
      @update:value="onInput"
      @click="!props.editable && $emit('browse')"
    />
    <a-button v-if="showOpenFolder" :disabled="!model?.trim()" @click="$emit('openFolder')">
      打开文件夹
    </a-button>
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

.path-pick-row__input--editable {
  cursor: text;
}
</style>
