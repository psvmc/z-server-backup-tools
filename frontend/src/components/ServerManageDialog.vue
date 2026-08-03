<script setup lang="ts">
import PathPickInput from "./PathPickInput.vue";
import RemoteDirPickerModal from "./RemoteDirPickerModal.vue";
import { serverTableColumns, useServerManageDialog } from "./ServerManageDialog";

const open = defineModel<boolean>("open", { required: true });

const emit = defineEmits<{
  changed: [];
}>();

const {
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
} = useServerManageDialog(open, emit);
</script>

<template>
  <a-modal
    v-model:open="open"
    :width="920"
    wrap-class-name="server-manage-modal"
    :mask-closable="false"
    centered
    :footer="null"
    title="服务器管理"
  >
    <a-spin :spinning="loading">
      <div class="server-manage-body">
        <div class="server-manage-body__header">
          <a-button type="primary" size="small" @click="openAdd">添加</a-button>
        </div>

        <a-table
        v-if="rows.length > 0"
        size="small"
        :columns="serverTableColumns"
        :data-source="rows"
        :pagination="false"
        class="server-manage-table"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'host'">
            {{ record.host }}{{ record.port ? `:${record.port}` : "" }}
          </template>
          <template v-else-if="column.key === 'os_type'">
            {{ record.os_type === "linux" ? "Linux" : "Windows" }}
          </template>
          <template v-else-if="column.key === 'support_multi_file'">
            {{ record.support_multi_file ? "是" : "否" }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space :size="4">
              <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
              <a-popconfirm
                title="确定删除该服务器？"
                ok-text="删除"
                cancel-text="取消"
                @confirm="onDelete(record)"
              >
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
      <a-empty v-else description="暂无服务器，点击「添加」创建" />
      </div>
    </a-spin>
  </a-modal>

  <a-modal
    v-model:open="formOpen"
    :title="isEdit ? '编辑服务器' : '添加服务器'"
    width="90vw"
    wrap-class-name="server-form-modal"
    :mask-closable="false"
    :confirm-loading="saving"
    ok-text="保存"
    cancel-text="取消"
    centered
    @ok="onSave"
  >
    <a-form layout="vertical" size="small" class="server-form">
      <a-form-item label="名称" required>
        <a-input v-model:value="form.name" placeholder="便于识别的名称" allow-clear />
      </a-form-item>

      <a-form-item label="服务器类型" required>
        <a-radio-group v-model:value="form.os_type">
          <a-radio value="windows">Windows</a-radio>
          <a-radio value="linux">Linux</a-radio>
        </a-radio-group>
      </a-form-item>

      <section class="settings-block">
        <div class="settings-block-title">SSH 连接</div>
        <div class="config-form-grid config-form-grid--conn">
          <a-form-item label="主机" required>
            <a-input v-model:value="form.host" placeholder="192.168.1.10" />
          </a-form-item>
          <a-form-item label="端口">
            <a-input-number v-model:value="form.port" :min="1" :max="65535" class="w-full" />
          </a-form-item>
          <a-form-item label="用户名" required>
            <a-input v-model:value="form.user" />
          </a-form-item>
          <a-form-item label="密码">
            <a-input-password v-model:value="form.password" autocomplete="off" />
          </a-form-item>
        </div>
        <div class="settings-block-actions">
          <a-button :loading="testing" :disabled="saving || testing" @click="testConnection">
            测试连接
          </a-button>
        </div>
      </section>

      <a-form-item class="server-form-check">
        <a-checkbox v-model:checked="form.support_multi_file">支持多文件备份</a-checkbox>
        <div class="text-xs text-gray-500 mt-1">
          未勾选时仅可用于单文件备份；勾选后需配置远程应用目录
        </div>
      </a-form-item>

      <section v-if="form.support_multi_file" class="settings-block">
        <div class="settings-block-title">远程应用</div>
        <div class="config-form-grid">
          <a-form-item label="远程应用目录" class="config-form-span-2" required>
            <PathPickInput
              v-model="form.remote_app_dir"
              editable
              :placeholder="appDirPlaceholder"
              @browse="openRemotePicker"
            />
          </a-form-item>
          <a-form-item label="分卷上限 (GB)" class="config-form-span-2" required>
            <a-input-number
              v-model:value="form.max_part_gb"
              :min="0.1"
              :step="0.5"
              class="w-full"
            />
            <div class="text-xs text-gray-500 mt-1">单文件超过上限时仍会单独打成一卷</div>
          </a-form-item>

          <div
            v-if="form.remote_app_dir.trim()"
            class="config-form-span-4 settings-derived-paths"
          >
            <div class="settings-derived-paths-label">自动推导路径（按任务 ID 区分）</div>
            <dl class="settings-derived-paths-list">
              <div>
                <dt>zipbak-srv</dt>
                <dd>{{ derivedPaths.srv }}</dd>
              </div>
              <div>
                <dt>state</dt>
                <dd>{{ derivedPaths.state }}</dd>
              </div>
              <div>
                <dt>staging</dt>
                <dd>{{ derivedPaths.staging }}</dd>
              </div>
            </dl>
          </div>
        </div>
      </section>
    </a-form>

    <RemoteDirPickerModal
      v-model:open="remotePickerOpen"
      :connection="formConnection"
      :initial-path="form.remote_app_dir"
      title="选择远程应用目录"
      @select="onRemoteSelect"
    />
  </a-modal>
</template>

<style scoped>
.server-manage-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 0;
}

.server-manage-body__header {
  display: flex;
  justify-content: flex-end;
}

.server-manage-table {
  width: 100%;
}

.server-form {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.server-form-check {
  margin-bottom: 8px;
}

.settings-block {
  padding: 12px 14px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: linear-gradient(180deg, var(--app-surface-card) 0%, #ffffff 100%);
  margin-bottom: 12px;
}

.settings-block-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-primary-dark);
  margin-bottom: 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--app-border-light);
}

.settings-block-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
  padding-top: 4px;
}

.config-form-grid--conn {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.settings-derived-paths {
  font-size: 12px;
  color: #4b635c;
  border-radius: 8px;
  border: 1px solid var(--app-border);
  background: var(--app-surface-muted);
  padding: 10px 12px;
}

.settings-derived-paths-label {
  font-weight: 500;
  color: var(--app-primary);
  margin-bottom: 8px;
}

.settings-derived-paths-list {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px 16px;
}

.settings-derived-paths-list > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.settings-derived-paths-list dt {
  margin: 0;
  font-weight: 500;
  color: var(--app-primary-muted);
}

.settings-derived-paths-list dd {
  margin: 0;
  word-break: break-all;
  font-family: Consolas, "Courier New", monospace;
  font-size: 11px;
  line-height: 1.4;
}

:deep(.ant-form-item) {
  margin-bottom: 8px;
}

:deep(.ant-input-number) {
  width: 100%;
}

@media (max-width: 720px) {
  .config-form-grid--conn {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .settings-derived-paths-list {
    grid-template-columns: 1fr;
  }
}
</style>
