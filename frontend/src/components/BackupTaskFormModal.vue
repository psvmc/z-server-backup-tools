<script setup lang="ts">
import PathPickInput from "./PathPickInput.vue";
import RemoteDirPickerModal from "./RemoteDirPickerModal.vue";
import {
  backupTaskFormModalProps,
  useBackupTaskFormModal,
} from "./BackupTaskFormModal";

const open = defineModel<boolean>("open", { required: true });
const props = defineProps(backupTaskFormModalProps);
const emit = defineEmits<{
  saved: [];
}>();

const {
  form,
  remotePickerOpen,
  saving,
  isEdit,
  isMulti,
  isFolderZip,
  ignorePatterns,
  serverOptions,
  connectionForPicker,
  remotePickerMode,
  remoteSourceLabel,
  remotePickerTitle,
  serverPlaceholder,
  openRemotePicker,
  pickLocalDir,
  openFolder,
  onSubmit,
} = useBackupTaskFormModal(open, props, emit);
</script>

<template>
  <a-modal
    v-model:open="open"
    :title="isEdit ? '编辑任务' : '添加任务'"
    :width="720"
    :mask-closable="false"
    :confirm-loading="saving"
    ok-text="保存"
    cancel-text="取消"
    @ok="onSubmit"
  >
    <a-form layout="vertical" size="small">
      <a-form-item label="服务器" required>
        <a-select
          v-model:value="form.server_id"
          :options="serverOptions"
          :placeholder="serverPlaceholder"
          allow-clear
          show-search
          option-filter-prop="label"
        />
      </a-form-item>
      <a-form-item label="任务名称（可选）">
        <a-input v-model:value="form.name" placeholder="便于识别的名称" allow-clear />
      </a-form-item>
      <a-form-item :label="remoteSourceLabel">
        <PathPickInput
          v-model="form.remote_source"
          editable
          placeholder="可手动输入，如 D:\data"
          @browse="openRemotePicker"
        />
      </a-form-item>
      <a-form-item v-if="isMulti" label="文件名前缀">
        <a-input
          v-model:value="form.part_name_prefix"
          placeholder="如 srv1-，压缩包将命名为 srv1-part-000001.zip"
          allow-clear
        />
        <div class="text-xs text-gray-500 mt-1">修改前缀后需重新对该任务执行远程 init</div>
      </a-form-item>
      <a-form-item v-if="isFolderZip" label="忽略文件名（可选）">
        <a-select
          v-model:value="ignorePatterns"
          mode="tags"
          :token-separators="[',']"
          placeholder="输入后按回车确认，如 *log.txt*"
          allow-clear
        />
        <div class="text-xs text-gray-500 mt-1">
          支持通配符 * ?（*log.txt* 匹配含 log.txt 的文件名，不区分大小写）或正则；输入后请按回车添加规则。
        </div>
      </a-form-item>
      <a-form-item label="本机保存目录">
        <PathPickInput
          v-model="form.local_dir"
          show-open-folder
          @browse="pickLocalDir"
          @open-folder="openFolder(form.local_dir)"
        />
      </a-form-item>
    </a-form>

    <RemoteDirPickerModal
      v-model:open="remotePickerOpen"
      :connection="connectionForPicker"
      :initial-path="form.remote_source"
      :mode="remotePickerMode"
      :title="remotePickerTitle"
      @select="(path) => (form.remote_source = path)"
    />
  </a-modal>
</template>
