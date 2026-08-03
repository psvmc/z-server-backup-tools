import { ref, type Ref } from "vue";
import { message } from "ant-design-vue";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import { BackupConfig as BackupConfigBinding } from "../../bindings/z-server-backup-tools/backend/model/models";
import type { BackupConfig } from "../types/backup";
import { formatError } from "../types/update";

export type BackupConfigPanelProps = {
  saving?: boolean;
};

export function useBackupConfigPanel(
  config: Ref<BackupConfig>,
  props: BackupConfigPanelProps,
) {
  const mailTesting = ref(false);

  async function testEmail() {
    if (mailTesting.value || props.saving) return;
    if (!config.value.notify_email?.trim()) {
      message.warning("请先填写通知邮箱");
      return;
    }
    if (!config.value.smtp_host?.trim()) {
      message.warning("请先填写 SMTP 服务器");
      return;
    }
    mailTesting.value = true;
    const hide = message.loading("正在发送测试邮件…", 0);
    try {
      const payload = {
        ...config.value,
        smtp_port: config.value.smtp_port && config.value.smtp_port > 0 ? config.value.smtp_port : 465,
      };
      const cfg = BackupConfigBinding.createFrom(JSON.parse(JSON.stringify(payload)));
      await BackupService.TestEmailNotification(cfg);
      hide();
      message.success("测试邮件已发送，请查收收件箱（含垃圾箱）");
    } catch (err) {
      hide();
      message.error(formatError(err));
    } finally {
      mailTesting.value = false;
    }
  }

  return {
    mailTesting,
    testEmail,
  };
}
