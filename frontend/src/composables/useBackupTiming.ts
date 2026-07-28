import { computed, onUnmounted, ref, watch, type Ref } from "vue";
import type { JobStatus } from "../types/backup";
import { estimateTotalMs, formatDuration } from "../utils/duration";

export function useBackupTiming(status: Ref<JobStatus>) {
  const backupStartedAt = ref<number | null>(null);
  const packedFilesAtStart = ref(0);
  /** 最近一次文件进度变化时推算的整任务总耗时；两次进度之间随时钟递减剩余时间 */
  const estimatedTotalMs = ref<number | null>(null);
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
        estimatedTotalMs.value = null;
        startClock();
      } else {
        stopClock();
        backupStartedAt.value = null;
        packedFilesAtStart.value = 0;
        estimatedTotalMs.value = null;
      }
    },
  );

  watch(
    () => status.value.packedFiles,
    (packed) => {
      if (!status.value.running || backupStartedAt.value == null) {
        return;
      }
      const total = status.value.totalFiles;
      if (total <= 0) {
        return;
      }
      const doneInSession = Math.max(0, packed - packedFilesAtStart.value);
      if (doneInSession <= 0) {
        return;
      }
      const elapsed = Date.now() - backupStartedAt.value;
      const totalEst = estimateTotalMs(elapsed, doneInSession, total);
      if (totalEst != null) {
        estimatedTotalMs.value = totalEst;
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
    const remainingFiles = Math.max(0, total - packed);
    if (remainingFiles === 0) {
      return "0:00";
    }
    if (estimatedTotalMs.value == null) {
      return "估算中…";
    }
    const est = Math.max(0, estimatedTotalMs.value - elapsedMs.value);
    return formatDuration(est);
  });

  return { elapsedText, remainingText };
}
