import { computed, onUnmounted, ref, watch, type Ref } from "vue";
import type { JobStatus } from "../types/backup";
import { formatDuration } from "../utils/duration";

/** Live elapsed time for direct SFTP jobs (single file / folder zip). */
export function useJobElapsed(status: Ref<JobStatus>) {
  const clockMs = ref(Date.now());
  let clockTimer: ReturnType<typeof setInterval> | null = null;

  const stopClock = () => {
    if (clockTimer) {
      clearInterval(clockTimer);
      clockTimer = null;
    }
  };

  const startClock = () => {
    stopClock();
    clockMs.value = Date.now();
    clockTimer = setInterval(() => {
      clockMs.value = Date.now();
    }, 1000);
  };

  watch(
    () => status.value.running,
    (running) => {
      if (running) {
        startClock();
      } else {
        stopClock();
      }
    },
    { immediate: true },
  );

  onUnmounted(stopClock);

  const elapsedText = computed(() => {
    const started = status.value.timingStartedAtMs ?? 0;
    if (started <= 0) {
      return "--:--";
    }
    if (status.value.running) {
      return formatDuration(clockMs.value - started);
    }
    const frozen = status.value.timingEstimatedTotalMs ?? 0;
    if (frozen > 0 && (status.value.done || status.value.lastError)) {
      return formatDuration(frozen);
    }
    return "--:--";
  });

  return { elapsedText };
}
