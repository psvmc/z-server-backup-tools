import { reactive } from "vue";
import type { UpdateCheckResult } from "../types/update";

export type UpdatePromptChoice = "confirm" | "later" | "skip";

type PromptResolver = (choice: UpdatePromptChoice) => void;

const state = reactive({
  show: false,
  result: null as UpdateCheckResult | null,
});

let resolvePrompt: PromptResolver | null = null;

function finish(choice: UpdatePromptChoice) {
  state.show = false;
  state.result = null;
  resolvePrompt?.(choice);
  resolvePrompt = null;
}

export function useUpdateConfirm() {
  const prompt = (result: UpdateCheckResult): Promise<UpdatePromptChoice> =>
    new Promise((resolve) => {
      if (resolvePrompt) resolvePrompt("later");
      state.result = result;
      state.show = true;
      resolvePrompt = resolve;
    });

  return {
    state,
    prompt,
    confirm: () => finish("confirm"),
    cancel: () => finish("later"),
    skip: () => finish("skip"),
  };
}
