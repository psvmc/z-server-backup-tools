<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { message } from "ant-design-vue";
import { Dialogs } from "@wailsio/runtime";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import type { BackupConfig, BackupTask } from "../types/backup";
import { formatError } from "../types/update";
import { newTaskId } from "../types/backup";
import PathPickInput from "./PathPickInput.vue";
import RemoteDirPickerModal from "./RemoteDirPickerModal.vue";

const open = defineModel<boolean>("open", { required: true });

const props = defineProps<{
  connection: BackupConfig;
  task?: BackupTask | null;
}>();

const emit = defineEmits<{
  saved: [];
}>();

const form = ref<BackupTask>({
  id: "",
  name: "",
  remote_source: "",
  local_dir: "",
  part_name_prefix: "",
});

const remotePickerOpen = ref(false);
const saving = ref(false);

const isEdit = computed(() => !!props.task?.id);

watch(
  () => [open.value, props.task] as const,
  ([visible, task]) => {
    if (!visible) return;
    if (task) {
      form.value = {
        id: task.id,
        name: task.name ?? "",
        remote_source: task.remote_source ?? "",
        local_dir: task.local_dir ?? "",
        part_name_prefix: task.part_name_prefix ?? "",
      };
    } else {
      form.value = {
        id: "",
        name: "",
        remote_source: "",
        local_dir: "",
        part_name_prefix: "",
      };
    }
  },
  { immediate: true },
);

function ensureConnectionFilled(): boolean {
  if (!props.connection.host?.trim() || !props.connection.user?.trim()) {
    message.warning("请先在设置中填写 SSH 主机和用户名");
    return false;
  }
  return true;
}

function isDialogCancelled(err: unknown): boolean {
  const text = formatError(err).toLowerCase();
  return (
    text.includes("cancel") ||
    text.includes("cancelled") ||
    text.includes("canceled") ||
    text.includes("abort") ||
    text.includes("取消")
  );
}

async function pickLocalDir() {
  try {
    const picked = await Dialogs.OpenFile({
      Title: "选择本机保存目录",
      CanChooseDirectories: true,
      CanChooseFiles: false,
      Directory: form.value.local_dir || undefined,
    });
    const path = Array.isArray(picked) ? picked[0] : picked;
    if (path && typeof path === "string") {
      form.value.local_dir = path;
    }
  } catch (err) {
    if (isDialogCancelled(err)) return;
    message.error(formatError(err));
  }
}

async function openFolder(path: string) {
  const target = path?.trim();
  if (!target) {
    message.warning("请先填写路径");
    return;
  }
  try {
    await BackupService.OpenInExplorer(target);
  } catch (err) {
    message.error(formatError(err));
  }
}

async function onSubmit() {
  if (!form.value.remote_source?.trim()) {
    message.warning("请填写远程源目录");
    return;
  }
  if (!form.value.local_dir?.trim()) {
    message.warning("请填写本机保存目录");
    return;
  }
  saving.value = true;
  try {
    const current = (await BackupService.GetTasks()) as BackupTask[];
    const payload: BackupTask = {
      id: isEdit.value ? form.value.id : newTaskId(),
      name: form.value.name?.trim() || undefined,
      remote_source: form.value.remote_source.trim(),
      local_dir: form.value.local_dir.trim(),
      part_name_prefix: form.value.part_name_prefix?.trim() || undefined,
    };
    let next: BackupTask[];
    if (isEdit.value) {
      next = current.map((t) => (t.id === payload.id ? payload : t));
    } else {
      next = [...current, payload];
    }
    await BackupService.SaveTasks(next);
    if (!isEdit.value) {
      await BackupService.SetActiveTaskID(payload.id);
    }
    message.success(isEdit.value ? "任务已更新" : "任务已添加");
    open.value = false;
    emit("saved");
  } catch (err) {
    message.error(formatError(err));
  } finally {
    saving.value = false;
  }
}
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
      <a-form-item label="任务名称（可选）">
        <a-input v-model:value="form.name" placeholder="便于识别的名称" allow-clear />
      </a-form-item>
      <a-form-item label="远程源目录 (--dir)">
        <PathPickInput
          v-model="form.remote_source"
          editable
          placeholder="可手动输入，如 D:\data"
          @browse="
            () => {
              if (!ensureConnectionFilled()) return;
              remotePickerOpen = true;
            }
          "
        />
      </a-form-item>
      <a-form-item label="文件名前缀">
        <a-input
          v-model:value="form.part_name_prefix"
          placeholder="如 srv1-，压缩包将命名为 srv1-part-000001.zip"
          allow-clear
        />
        <div class="text-xs text-gray-500 mt-1">修改前缀后需重新对该任务执行远程 init</div>
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
      :connection="connection"
      :initial-path="form.remote_source"
      title="选择远程源目录"
      @select="(path) => (form.remote_source = path)"
    />
  </a-modal>
</template>
