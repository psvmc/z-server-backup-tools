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

type TaskTableColumn = {
  title: string;
  dataIndex?: string;
  key: string;
  width?: number;
  minWidth?: number;
  ellipsis?: boolean;
  fixed?: "left" | "right";
};

const ACTIONS_COLUMN_WIDTH = 148;
const TABLE_CELL_FONT = '14px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif';

type ColumnBounds = { min: number; max: number };

type TaskColumnSpec = {
  title: string;
  dataIndex: string;
  key: string;
  bounds: ColumnBounds;
};

const sharedColumnBounds = {
  name: { min: 72, max: 160 },
  remote_source: { min: 120, max: 560 },
  local_dir: { min: 120, max: 420 },
  part_name_prefix: { min: 56, max: 140 },
} as const satisfies Record<string, ColumnBounds>;

const variantColumnSpecs: Record<BackupTaskKind, TaskColumnSpec[]> = {
  single: [
    { title: "任务", dataIndex: "name", key: "name", bounds: sharedColumnBounds.name },
    { title: "远程源文件", dataIndex: "remote_source", key: "remote_source", bounds: sharedColumnBounds.remote_source },
    { title: "本机目录", dataIndex: "local_dir", key: "local_dir", bounds: sharedColumnBounds.local_dir },
  ],
  folder_zip: [
    { title: "任务", dataIndex: "name", key: "name", bounds: sharedColumnBounds.name },
    { title: "远程文件夹", dataIndex: "remote_source", key: "remote_source", bounds: sharedColumnBounds.remote_source },
    { title: "本机目录", dataIndex: "local_dir", key: "local_dir", bounds: sharedColumnBounds.local_dir },
  ],
  multi: [
    { title: "任务", dataIndex: "name", key: "name", bounds: sharedColumnBounds.name },
    { title: "远程源目录", dataIndex: "remote_source", key: "remote_source", bounds: sharedColumnBounds.remote_source },
    { title: "本机目录", dataIndex: "local_dir", key: "local_dir", bounds: sharedColumnBounds.local_dir },
    { title: "前缀", dataIndex: "part_name_prefix", key: "part_name_prefix", bounds: sharedColumnBounds.part_name_prefix },
  ],
};

let measureCanvas: HTMLCanvasElement | null = null;

function measureTextPx(text: string, font = TABLE_CELL_FONT) {
  if (!text) return 0;
  if (typeof document === "undefined") return text.length * 8;
  measureCanvas ??= document.createElement("canvas");
  const ctx = measureCanvas.getContext("2d");
  if (!ctx) return text.length * 8;
  ctx.font = font;
  return ctx.measureText(text).width;
}

function fitColumnWidth(title: string, values: string[], min: number, max: number, cellPadding = 32) {
  const texts = [title, ...values.map((v) => v.trim()).filter(Boolean)];
  const content = texts.length ? Math.max(...texts.map((t) => measureTextPx(t))) : measureTextPx(title);
  return Math.min(max, Math.max(min, Math.ceil(content) + cellPadding));
}

type TaskTableRow = BackupTask & {
  key: string;
  name: string;
  part_name_prefix: string;
};

function buildTaskColumns(variant: BackupTaskKind, rows: TaskTableRow[]): TaskTableColumn[] {
  const specs = variantColumnSpecs[variant];
  const dataColumns = specs.map((spec) => {
    const values = rows.map((row) => String(row[spec.dataIndex as keyof TaskTableRow] ?? ""));
    return {
      title: spec.title,
      dataIndex: spec.dataIndex,
      key: spec.key,
      width: fitColumnWidth(spec.title, values, spec.bounds.min, spec.bounds.max),
      ellipsis: true,
    };
  });

  return [...dataColumns, { title: "操作", key: "actions", width: ACTIONS_COLUMN_WIDTH, fixed: "right" as const }];
}

function tableScrollX(columns: TaskTableColumn[]) {
  return columns.reduce((sum, col) => sum + (col.width ?? col.minWidth ?? 120), 0);
}

export function isEllipsisColumnKey(key: unknown): key is "remote_source" | "local_dir" | "part_name_prefix" | "name" {
  return key === "remote_source" || key === "local_dir" || key === "part_name_prefix" || key === "name";
}

export function ellipsisCellText(record: Record<string, unknown>, key: "remote_source" | "local_dir" | "part_name_prefix" | "name") {
  const value = record[key];
  if (typeof value !== "string") return "";
  const text = value.trim();
  if (!text || text === "-") return "";
  return text;
}

export function useBackupTaskList(props: BackupTaskListProps) {
  const rows = computed(() =>
    props.tasks.map((task) => ({
      ...task,
      key: task.id,
      name: taskDisplayName(task),
      part_name_prefix: task.part_name_prefix?.trim() || "-",
    })),
  );

  const columns = computed(() => buildTaskColumns(props.variant ?? "multi", rows.value));
  const scrollX = computed(() => tableScrollX(columns.value));

  return { columns, rows, scrollX, isEllipsisColumnKey, ellipsisCellText };
}
