import { expect } from "@playwright/test";
import type { ComposerComponent } from "../components/composer";
import { expectFocusedValue, expectInputReady } from "./input";

export async function expectComposerReady(composer: ComposerComponent, timeout = 20000): Promise<void> {
  const input = composer.input();
  await expect(input).toHaveAttribute("data-composer-ready", "true", { timeout });
  await expectInputReady(input, timeout);
}

export async function expectComposerValue(composer: ComposerComponent, value: string | RegExp): Promise<void> {
  await expect(composer.input()).toHaveValue(value);
}

export async function expectComposerFocusedValue(composer: ComposerComponent, value: string | RegExp): Promise<void> {
  await expectFocusedValue(composer.input(), value);
}

export async function expectComposerCounter(composer: ComposerComponent, value: string | RegExp): Promise<void> {
  await expect(composer.counter()).toHaveText(value);
}

export async function expectComposerState(
  composer: ComposerComponent,
  state: {
    draft?: "dirty" | "empty";
    composing?: boolean;
    pending?: boolean;
    disabled?: boolean;
  }
): Promise<void> {
  const input = composer.input();
  if (state.draft) {
    await expect(input).toHaveAttribute("data-composer-draft-state", state.draft);
  }
  if (typeof state.composing === "boolean") {
    await expect(input).toHaveAttribute("data-composer-composing", state.composing ? "true" : "false");
  }
  if (typeof state.pending === "boolean") {
    await expect(input).toHaveAttribute("data-composer-pending", state.pending ? "true" : "false");
  }
  if (typeof state.disabled === "boolean") {
    await expect(input).toHaveAttribute("data-composer-disabled", state.disabled ? "true" : "false");
  }
}
