import { computed, ref } from "vue";
import type { Ref } from "vue";
import type { JobStatus } from "./types/backup";

export type MainTabKey = "multi" | "single" | "folder_zip";

/** Homepage tab keys + whether each direct-backup panel is active. */
export function useAppMainTabs() {
  const mainTab = ref<MainTabKey>("multi");
  const singlePanelActive = computed(() => mainTab.value === "single");
  const folderZipPanelActive = computed(() => mainTab.value === "folder_zip");
  return { mainTab, singlePanelActive, folderZipPanelActive };
}

/** Mutual exclusion flags from multi-file vs direct SFTP jobs. */
export function useJobExclusion(
  multiStatus: Ref<JobStatus>,
  singleStatus: Ref<JobStatus>,
  folderZipStatus: Ref<JobStatus>,
) {
  const multiRunning = computed(() => !!multiStatus.value.running);
  const singleRunning = computed(() => !!singleStatus.value.running);
  const folderZipRunning = computed(() => !!folderZipStatus.value.running);
  const directRunning = computed(() => singleRunning.value || folderZipRunning.value);
  return { multiRunning, singleRunning, folderZipRunning, directRunning };
}
