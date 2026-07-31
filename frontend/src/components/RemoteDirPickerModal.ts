import { computed, ref, watch, type Ref } from "vue";
import { message } from "ant-design-vue";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import { BackupConfig as BackupConfigBinding } from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig } from "../types/backup";
import { normalizeServerOSType } from "../types/backup";
import { formatError } from "../types/update";

export type RemotePickerMode = "directory" | "file";

export type RemoteDirPickerProps = {
  connection: BackupConfig;
  initialPath?: string;
  title?: string;
  mode?: RemotePickerMode;
};

export type RemoteDirPickerEmit = {
  (e: "select", path: string): void;
};

export type RemoteListEntry = {
  name: string;
  path: string;
  is_dir: boolean;
};

export function useRemoteDirPicker(
  open: Ref<boolean>,
  props: RemoteDirPickerProps,
  emit: RemoteDirPickerEmit,
) {
  const loading = ref(false);
  const currentPath = ref("");
  const parentPath = ref("");
  const entries = ref<RemoteListEntry[]>([]);
  const jumpPath = ref("");

  const mode = computed<RemotePickerMode>(() => props.mode ?? "directory");
  const isLinux = computed(() => normalizeServerOSType(props.connection?.os_type) === "linux");
  const isFileMode = computed(() => mode.value === "file");

  /** Windows: empty = drive list. Linux: "/" = filesystem root. */
  const atRoot = computed(() => {
    const cur = currentPath.value.trim();
    if (isLinux.value) return !cur || cur === "/";
    return !cur;
  });

  const currentLabel = computed(() => {
    if (isLinux.value) {
      return currentPath.value.trim() || "/";
    }
    return atRoot.value ? "（盘符列表）" : currentPath.value;
  });

  const jumpPlaceholder = computed(() => {
    if (isLinux.value) {
      return isFileMode.value ? "手动输入路径，如 /var/log/app.log" : "手动输入路径，如 /opt/data";
    }
    return isFileMode.value ? "手动输入路径，如 D:\\data\\app.bak" : "手动输入路径，如 D:\\Tools\\zipbak";
  });

  const rootButtonLabel = computed(() => (isLinux.value ? "根目录" : "盘符列表"));

  const modalTitle = computed(() => {
    if (props.title?.trim()) return props.title.trim();
    return isFileMode.value ? "选择远程文件" : "选择远程目录";
  });

  const hintText = computed(() => {
    if (atRoot.value) {
      return isLinux.value
        ? "当前为根目录；也可在上方手动输入完整路径。"
        : "点击盘符进入；也可在上方手动输入完整路径。";
    }
    if (isFileMode.value) {
      return "单击文件夹进入；点文件旁「选中」确认文件路径。";
    }
    return "单击文件夹进入；点「选中」确认路径。";
  });

  const emptyText = computed(() => {
    if (atRoot.value && !isLinux.value) return "未发现可用盘符";
    if (isFileMode.value) return "此目录下没有可显示的项目";
    return "此目录下没有子文件夹";
  });

  function toBinding(cfg: BackupConfig) {
    return BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(cfg)));
  }

  async function load(pathHint: string) {
    loading.value = true;
    try {
      const listing = isFileMode.value
        ? await BackupService.ListRemoteEntries(toBinding(props.connection), pathHint)
        : await BackupService.ListRemoteDirectories(toBinding(props.connection), pathHint);
      currentPath.value = listing.current_path ?? "";
      parentPath.value = listing.parent_path ?? "";
      entries.value = (listing.entries ?? []).map((e) => ({
        name: e.name ?? "",
        path: e.path ?? "",
        is_dir: e.is_dir ?? true,
      }));
      jumpPath.value = currentPath.value;
    } catch (err) {
      message.error(formatError(err));
    } finally {
      loading.value = false;
    }
  }

  watch(
    () => open.value,
    (visible) => {
      if (visible) {
        const initial = props.initialPath?.trim() || "";
        void load(initial || (isLinux.value ? "/" : ""));
      }
    },
  );

  function enterDir(path: string) {
    void load(path);
  }

  function onItemActivate(item: RemoteListEntry) {
    if (item.is_dir) {
      enterDir(item.path);
      return;
    }
    if (isFileMode.value) {
      confirmEntry(item.path);
    }
  }

  function goParent() {
    if (atRoot.value) return;
    if (isLinux.value) {
      void load(parentPath.value || "/");
      return;
    }
    void load(parentPath.value || "");
  }

  function goRoot() {
    void load(isLinux.value ? "/" : "");
  }

  function jumpToPath() {
    const target = jumpPath.value.trim();
    if (!target) {
      goRoot();
      return;
    }
    void load(target);
  }

  function confirmCurrent() {
    if (isFileMode.value) {
      message.warning("请选择一个文件");
      return;
    }
    const picked = currentPath.value.trim() || (isLinux.value ? "/" : "");
    if (!picked) {
      message.warning("请先选择盘符并进入目录");
      return;
    }
    emit("select", picked);
    open.value = false;
  }

  function confirmEntry(path: string) {
    emit("select", path);
    open.value = false;
  }

  function showSelectButton(item: RemoteListEntry): boolean {
    if (item.is_dir) {
      if (isFileMode.value) return false;
      // Windows drive list: hide select on drives (same as before: !atRoot)
      return !atRoot.value || isLinux.value;
    }
    return isFileMode.value;
  }

  return {
    loading,
    currentPath,
    entries,
    jumpPath,
    atRoot,
    isLinux,
    isFileMode,
    currentLabel,
    jumpPlaceholder,
    rootButtonLabel,
    modalTitle,
    hintText,
    emptyText,
    enterDir,
    onItemActivate,
    goParent,
    goRoot,
    jumpToPath,
    confirmCurrent,
    confirmEntry,
    showSelectButton,
  };
}
