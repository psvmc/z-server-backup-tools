import { computed, type PropType } from "vue";
import type { BackupTask, BackupTaskKind } from "../types/backup";
import { taskDisplayName } from "../types/backup";

export const backupTaskListProps = {
  tasks: {
    type: Array as PropType<BackupTask[]>,
    required: true as const,
  },
  activeTaskId: {
    type: String,
    required: true as const,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  variant: {
    type: String as PropType<BackupTaskKind>,
    default: "multi" as const,
  },
};

export type BackupTaskListProps = {
  tasks: BackupTask[];
  activeTaskId: string;
  disabled?: boolean;
  variant?: BackupTaskKind;
};

export type BackupTaskListEmit = {
  (e: "select", id: string): void;
  (e: "add"): void;
  (e: "edit", task: BackupTask): void;
  (e: "remove", task: BackupTask): void;
};

const multiColumns = [
  { title: "任务", dataIndex: "name", key: "name", width: 100, ellipsis: true },
  { title: "远程源目录", dataIndex: "remote_source", key: "remote_source", width: 280, ellipsis: true },
  { title: "本机目录", dataIndex: "local_dir", key: "local_dir", ellipsis: true },
  { title: "前缀", dataIndex: "part_name_prefix", key: "part_name_prefix", width: 100, ellipsis: true },
  { title: "操作", key: "actions", width: 160 },
];

const singleColumns = [
  { title: "任务", dataIndex: "name", key: "name", width: 100, ellipsis: true },
  { title: "远程源文件", dataIndex: "remote_source", key: "remote_source", width: 280, ellipsis: true },
  { title: "本机目录", dataIndex: "local_dir", key: "local_dir", ellipsis: true },
  { title: "操作", key: "actions", width: 160 },
];

export function useBackupTaskList(props: BackupTaskListProps) {
  const columns = computed(() => (props.variant === "single" ? singleColumns : multiColumns));

  const rows = computed(() =>
    props.tasks.map((task) => ({
      ...task,
      key: task.id,
      name: taskDisplayName(task),
      part_name_prefix: task.part_name_prefix?.trim() || "-",
    })),
  );

  return { columns, rows };
}
