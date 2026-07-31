import { computed, ref, watch, type Ref } from "vue";
import { message } from "ant-design-vue";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import {
  BackupConfig as BackupConfigBinding,
  Server as ServerBinding,
} from "../../bindings/z-server-backup-tools/backend/model/models";
import type { Server } from "../types/backup";
import {
  applyServer,
  emptyNotifyConfig,
  normalizeServerOSType,
  remotePathsFromAppDir,
} from "../types/backup";
import { formatError } from "../types/update";

export type ServerManageDialogEmit = {
  (e: "changed"): void;
};

export type ServerForm = {
  id: string;
  name: string;
  host: string;
  port: number;
  user: string;
  password: string;
  os_type: string;
  support_multi_file: boolean;
  remote_app_dir: string;
  max_part_gb: number;
};

function emptyServerForm(): ServerForm {
  return {
    id: "",
    name: "",
    host: "",
    port: 22,
    user: "",
    password: "",
    os_type: "windows",
    support_multi_file: false,
    remote_app_dir: "",
    max_part_gb: 2,
  };
}

function bindingToPlain(srv: ServerBinding): Server {
  return JSON.parse(JSON.stringify(srv)) as Server;
}

function toBindingServer(srv: Server) {
  return ServerBinding.createFrom(JSON.parse(JSON.stringify(srv)));
}

function toBindingConfig(cfg: ReturnType<typeof applyServer>) {
  return BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(cfg)));
}

export const serverTableColumns = [
  { title: "名称", dataIndex: "name", key: "name", ellipsis: true },
  { title: "主机", key: "host", ellipsis: true },
  { title: "类型", key: "os_type", width: 90 },
  { title: "多文件支持", key: "support_multi_file", width: 110 },
  { title: "操作", key: "actions", width: 140 },
];

export function useServerManageDialog(open: Ref<boolean>, emit: ServerManageDialogEmit) {
  const servers = ref<Server[]>([]);
  const loading = ref(false);
  const formOpen = ref(false);
  const form = ref<ServerForm>(emptyServerForm());
  const saving = ref(false);
  const testing = ref(false);
  const remotePickerOpen = ref(false);

  const isEdit = computed(() => !!form.value.id);
  const isLinux = computed(() => normalizeServerOSType(form.value.os_type) === "linux");
  const derivedPaths = computed(() =>
    remotePathsFromAppDir(form.value.remote_app_dir, form.value.os_type),
  );
  const appDirPlaceholder = computed(() =>
    isLinux.value ? "可手动输入，如 /opt/zipbak" : "可手动输入，如 D:\\Tools\\zipbak",
  );
  const formConnection = computed(() =>
    applyServer(emptyNotifyConfig(), {
      id: form.value.id,
      name: form.value.name,
      host: form.value.host,
      port: form.value.port,
      user: form.value.user,
      password: form.value.password,
      os_type: normalizeServerOSType(form.value.os_type),
      support_multi_file: form.value.support_multi_file,
      remote_app_dir: form.value.remote_app_dir,
      max_part_gb: form.value.max_part_gb,
    }),
  );

  const rows = computed(() =>
    servers.value.map((srv) => ({
      ...srv,
      key: srv.id,
    })),
  );

  async function loadServers() {
    loading.value = true;
    try {
      const list = await BackupService.GetServers();
      servers.value = (list ?? []).map((s) => bindingToPlain(s));
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
        void loadServers();
      } else {
        formOpen.value = false;
        remotePickerOpen.value = false;
      }
    },
  );

  function openAdd() {
    form.value = emptyServerForm();
    formOpen.value = true;
  }

  function openEdit(srv: Server) {
    form.value = {
      id: srv.id,
      name: srv.name ?? "",
      host: srv.host ?? "",
      port: srv.port || 22,
      user: srv.user ?? "",
      password: srv.password ?? "",
      os_type: normalizeServerOSType(srv.os_type),
      support_multi_file: !!srv.support_multi_file,
      remote_app_dir: srv.remote_app_dir ?? "",
      max_part_gb: srv.max_part_gb && srv.max_part_gb > 0 ? srv.max_part_gb : 2,
    };
    formOpen.value = true;
  }

  function ensureSshFilled(): boolean {
    if (!form.value.host?.trim() || !form.value.user?.trim()) {
      message.warning("请先填写 SSH 主机和用户名");
      return false;
    }
    return true;
  }

  function openRemotePicker() {
    if (saving.value) return;
    if (!ensureSshFilled()) return;
    remotePickerOpen.value = true;
  }

  function onRemoteSelect(path: string) {
    form.value.remote_app_dir = path;
  }

  async function testConnection() {
    if (testing.value || saving.value) return;
    if (!ensureSshFilled()) return;
    testing.value = true;
    const hide = message.loading("测试 SSH 连接...", 0);
    try {
      await BackupService.TestConnection(toBindingConfig(formConnection.value));
      hide();
      message.success("连接成功");
    } catch (err) {
      hide();
      message.error(formatError(err));
    } finally {
      testing.value = false;
    }
  }

  function validateForm(): boolean {
    if (!form.value.name?.trim()) {
      message.warning("请填写服务器名称");
      return false;
    }
    if (!form.value.host?.trim()) {
      message.warning("请填写主机");
      return false;
    }
    if (!form.value.user?.trim()) {
      message.warning("请填写用户名");
      return false;
    }
    if (form.value.support_multi_file) {
      if (!form.value.remote_app_dir?.trim()) {
        message.warning("请填写远程应用目录");
        return false;
      }
      if (!form.value.max_part_gb || form.value.max_part_gb <= 0) {
        message.warning("分卷上限必须大于 0");
        return false;
      }
    }
    return true;
  }

  async function onSave() {
    if (saving.value) return Promise.reject();
    if (!validateForm()) return Promise.reject();
    saving.value = true;
    try {
      const payload: Server = {
        id: form.value.id,
        name: form.value.name.trim(),
        host: form.value.host.trim(),
        port: form.value.port && form.value.port > 0 ? form.value.port : 22,
        user: form.value.user.trim(),
        password: form.value.password ?? "",
        os_type: normalizeServerOSType(form.value.os_type),
        support_multi_file: !!form.value.support_multi_file,
        remote_app_dir: form.value.support_multi_file ? form.value.remote_app_dir.trim() : "",
        max_part_gb: form.value.support_multi_file ? form.value.max_part_gb || 0 : 0,
      };
      await BackupService.SaveServer(toBindingServer(payload));
      message.success(isEdit.value ? "服务器已更新" : "服务器已添加");
      formOpen.value = false;
      await loadServers();
      emit("changed");
    } catch (err) {
      message.error(formatError(err));
      return Promise.reject(err);
    } finally {
      saving.value = false;
    }
  }

  async function onDelete(srv: Server) {
    try {
      await BackupService.DeleteServer(srv.id);
      message.success("服务器已删除");
      await loadServers();
      emit("changed");
    } catch (err) {
      message.error(formatError(err));
    }
  }

  return {
    servers,
    loading,
    formOpen,
    form,
    saving,
    testing,
    remotePickerOpen,
    isEdit,
    derivedPaths,
    appDirPlaceholder,
    formConnection,
    rows,
    openAdd,
    openEdit,
    openRemotePicker,
    onRemoteSelect,
    testConnection,
    onSave,
    onDelete,
  };
}
