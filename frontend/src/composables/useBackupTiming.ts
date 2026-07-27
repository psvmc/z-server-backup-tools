import { computed, onUnmounted, ref, watch, type Ref } from "vue";
import type { JobStatus } from "../types/backup";
import { estimateRemainingMs, formatDuration } from "../utils/duration";

export function useBackupTiming(status: Ref<JobStatus>) {
  const backupStartedAt = ref<number | null>(null);
  const packedFilesAtStart = ref(0);
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
        backupStartedAt.value = Date.now();
        packedFilesAtStart.value = status.value.packedFiles;
        startClock();
      } else {
        stopClock();
        backupStartedAt.value = null;
        packedFilesAtStart.value = 0;
      }
    },
  );

  onUnmounted(stopClock);

  const elapsedMs = computed(() => {
    if (!status.value.running || backupStartedAt.value == null) {
      return 0;
    }
    return clockMs.value - backupStartedAt.value;
  });

  const elapsedText = computed(() => {
    if (!status.value.running) {
      return "--:--";
    }
    return formatDuration(elapsedMs.value);
  });

  const remainingText = computed(() => {
    if (!status.value.running) {
      return "--:--";
    }
    const total = status.value.totalFiles;
    const packed = status.value.packedFiles;
    if (total <= 0) {
      return "估算中…";
    }
    const doneInSession = Math.max(0, packed - packedFilesAtStart.value);
    const remainingFiles = Math.max(0, total - packed);
    if (remainingFiles === 0) {
      return "0:00";
    }
    const est = estimateRemainingMs(elapsedMs.value, doneInSession, remainingFiles);
    if (est == null) {
      return "估算中…";
    }
    return formatDuration(est);
  });

  return { elapsedText, remainingText };
}
