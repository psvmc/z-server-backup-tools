import { ref } from "vue";
import { Events } from "@wailsio/runtime";

export type UpdateProgressPhase = "downloading" | "verifying" | "installing";

export interface UpdateProgressState {
  phase: UpdateProgressPhase;
  written: number;
  total: number;
  rate: number;
}

interface DownloadProgressPayload {
  written?: number;
  total?: number;
  rate?: number;
}

const updateProgress = ref<UpdateProgressState | null>(null);
let unsubscribers: Array<() => void> = [];

function eventData<T>(ev: unknown): T | null {
  if (ev == null) return null;
  if (typeof ev === "object" && "data" in ev) {
    const data = (ev as { data?: T }).data;
    if (data != null) return data;
  }
  return ev as T;
}

function subscribeUpdateEvents() {
  unsubscribers = [
    Events.On("wails:updater:download-started", () => {
      if (!updateProgress.value) return;
      updateProgress.value.phase = "downloading";
      updateProgress.value.written = 0;
      updateProgress.value.total = 0;
      updateProgress.value.rate = 0;
    }),
    Events.On("wails:updater:download-progress", (ev) => {
      if (!updateProgress.value) return;
      const payload = eventData<DownloadProgressPayload>(ev);
      if (!payload) return;
      updateProgress.value.phase = "downloading";
      updateProgress.value.written = payload.written ?? 0;
      updateProgress.value.total = payload.total ?? 0;
      updateProgress.value.rate = payload.rate ?? 0;
    }),
    Events.On("wails:updater:verifying", () => {
      if (updateProgress.value) updateProgress.value.phase = "verifying";
    }),
    Events.On("wails:updater:installing", () => {
      if (updateProgress.value) updateProgress.value.phase = "installing";
    }),
  ];
}

function unsubscribeUpdateEvents() {
  for (const off of unsubscribers) off();
  unsubscribers = [];
}

export function useUpdateProgress() {
  const begin = () => {
    updateProgress.value = { phase: "downloading", written: 0, total: 0, rate: 0 };
    subscribeUpdateEvents();
  };

  const end = () => {
    unsubscribeUpdateEvents();
    updateProgress.value = null;
  };

  const runWithProgress = async <T>(task: () => Promise<T>): Promise<T> => {
    begin();
    try {
      return await task();
    } finally {
      end();
    }
  };

  return { updateProgress, runWithProgress };
}
