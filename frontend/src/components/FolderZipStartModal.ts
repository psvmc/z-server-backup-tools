import { computed, ref, watch, type Ref } from "vue";
import { message } from "ant-design-vue";
import type { BackupTask } from "../types/backup";
import { isFolderZipTaskRunnable, taskDisplayName } from "../types/backup";

export const folderZipStartModalProps = {
  open: {
    type: Boolean,
    required: true as const,
  },
  tasks: {
    type: Array as () => BackupTask[],
    required: true as const,
  },
  activeTaskId: {
    type: String,
    default: "",
  },
};

export type FolderZipStartModalProps = {
  open: boolean;
  tasks: BackupTask[];
  activeTaskId?: string;
};

export type FolderZipStartModalEmit = {
  (e: "update:open", value: boolean): void;
  (e: "confirm", taskIds: string[]): void;
};

export type FolderZipStartRow = BackupTask & {
  key: string;
  label: string;
  runnable: boolean;
};

export function useFolderZipStartModal(
  open: Ref<boolean>,
  props: FolderZipStartModalProps,
  emit: FolderZipStartModalEmit,
) {
  const selectedIds = ref<string[]>([]);

  const rows = computed<FolderZipStartRow[]>(() =>
    props.tasks.map((task) => ({
      ...task,
      key: task.id,
      label: taskDisplayName(task),
      runnable: isFolderZipTaskRunnable(task),
    })),
  );

  const runnableIds = computed(() => rows.value.filter((r) => r.runnable).map((r) => r.id));

  const allChecked = computed(
    () =>
      runnableIds.value.length > 0 &&
      runnableIds.value.every((id) => selectedIds.value.includes(id)),
  );

  const indeterminate = computed(
    () =>
      selectedIds.value.length > 0 &&
      !allChecked.value &&
      runnableIds.value.some((id) => selectedIds.value.includes(id)),
  );

  function resetSelection() {
    const runnable = runnableIds.value;
    if (runnable.length === 0) {
      selectedIds.value = [];
      return;
    }
    const active = props.activeTaskId?.trim();
    if (active && runnable.includes(active)) {
      selectedIds.value = [active];
      return;
    }
    selectedIds.value = [...runnable];
  }

  watch(
    () => open.value,
    (isOpen) => {
      if (isOpen) {
        resetSelection();
      }
    },
  );

  function toggleAll(checked: boolean) {
    selectedIds.value = checked ? [...runnableIds.value] : [];
  }

  function onCancel() {
    emit("update:open", false);
  }

  function onConfirm() {
    const ids = selectedIds.value.filter((id) => runnableIds.value.includes(id));
    if (ids.length === 0) {
      message.warning("请至少选择一个可执行的任务");
      return;
    }
    emit("confirm", ids);
    emit("update:open", false);
  }

  return {
    selectedIds,
    rows,
    runnableIds,
    allChecked,
    indeterminate,
    toggleAll,
    onCancel,
    onConfirm,
  };
}
