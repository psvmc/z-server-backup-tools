import { Modal } from "ant-design-vue";

export function confirmInitRemote(onConfirm: () => void) {
  Modal.confirm({
    title: "远程 init",
    content: "将扫描当前任务的源目录并生成文件清单，是否继续？",
    okText: "开始 init",
    cancelText: "取消",
    centered: true,
    onOk: onConfirm,
  });
}
