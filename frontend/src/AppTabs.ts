import { computed, ref } from "vue";
import type { Ref } from "vue";
import type { JobStatus } from "./types/backup";

export type MainTabKey = "multi" | "single";

/** Homepage dual-tab key + whether the single-file panel is the active tab. */
export function useAppMainTabs() {
  const mainTab = ref<MainTabKey>("multi");
  const singlePanelActive = computed(() => mainTab.value === "single");
  return { mainTab, singlePanelActive };
}

/** Mutual exclusion flags from multi-file vs single-file job status. */
export function useJobExclusion(
  multiStatus: Ref<JobStatus>,
  singleStatus: Ref<JobStatus>,
) {
  const multiRunning = computed(() => !!multiStatus.value.running);
  const singleRunning = computed(() => !!singleStatus.value.running);
  return { multiRunning, singleRunning };
}
