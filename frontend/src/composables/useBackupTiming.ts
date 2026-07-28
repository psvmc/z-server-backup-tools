import { computed, onUnmounted, ref, watch, type Ref } from "vue";
import { BackupService } from "../../bindings/z-server-backup-tools/backend/service";
import type { JobStatus } from "../types/backup";
import { estimateTotalMs, formatDuration } from "../utils/duration";

function syncFromStatus(
  status: JobStatus,
  backupStartedAt: Ref<number | null>,
  packedFilesAtStart: Ref<number>,
  estimatedTotalMs: Ref<number | null>,
) {
  const started = status.timingStartedAtMs ?? 0;
  if (started > 0 && !status.done) {
    backupStartedAt.value = started;
    packedFilesAtStart.value = status.timingPackedFilesAtStart ?? 0;
    if (status.timingEstimatedTotalMs && status.timingEstimatedTotalMs > 0) {
      estimatedTotalMs.value = status.timingEstimatedTotalMs;
    }
  } else {
    backupStartedAt.value = null;
    packedFilesAtStart.value = 0;
    estimatedTotalMs.value = null;
  }
}

export function useBackupTiming(status: Ref<JobStatus>) {
  const backupStartedAt = ref<number | null>(null);
  const packedFilesAtStart = ref(0);
  const estimatedTotalMs = ref<number | null>(null);
  const clockMs = ref(Date.now());
  let clockTimer: ReturnType<typeof setInterval> | null = null;
  let persistTimer: ReturnType<typeof setTimeout> | null = null;

  const timingVisible = computed(() => {
    if (status.value.running) return true;
    const started = status.value.timingStartedAtMs ?? backupStartedAt.value ?? 0;
    return started > 0 && !status.value.done;
  });

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
    () => status.value,
    (st) => {
      syncFromStatus(st, backupStartedAt, packedFilesAtStart, estimatedTotalMs);
      if (st.running) {
        startClock();
      } else {
        stopClock();
      }
    },
    { deep: true, immediate: true },
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
        if (persistTimer) {
          clearTimeout(persistTimer);
        }
        persistTimer = setTimeout(() => {
          void BackupService.SetBackupTimingEstimate(Math.round(totalEst));
        }, 800);
      }
    },
  );

  onUnmounted(() => {
    stopClock();
    if (persistTimer) {
      clearTimeout(persistTimer);
    }
  });

  const elapsedMs = computed(() => {
    if (backupStartedAt.value == null || !timingVisible.value) {
      return 0;
    }
    return clockMs.value - backupStartedAt.value;
  });

  const elapsedText = computed(() => {
    if (!timingVisible.value) {
      return "--:--";
    }
    return formatDuration(elapsedMs.value);
  });

  const remainingText = computed(() => {
    if (!timingVisible.value) {
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
