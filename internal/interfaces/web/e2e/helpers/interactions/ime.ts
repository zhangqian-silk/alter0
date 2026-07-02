import { expect, type Locator } from "@playwright/test";

export async function startIMEInput(target: Locator, composingValue = "ni"): Promise<void> {
  await target.click();
  await expect(target).toBeFocused();
  await target.dispatchEvent("compositionstart", { data: "n" });
  await target.fill(composingValue);
}

export async function pressEnterDuringIMEInput(target: Locator): Promise<void> {
  await target.evaluate((node) => {
    const keydown = new KeyboardEvent("keydown", {
      key: "Enter",
      code: "Enter",
      bubbles: true,
      cancelable: true,
    });
    Object.defineProperty(keydown, "isComposing", { value: true });
    node.dispatchEvent(keydown);
    const keyup = new KeyboardEvent("keyup", {
      key: "Enter",
      code: "Enter",
      bubbles: true,
      cancelable: true,
    });
    Object.defineProperty(keyup, "isComposing", { value: true });
    node.dispatchEvent(keyup);
  });
}

export async function commitIMEInput(target: Locator, committedValue: string): Promise<void> {
  await target.fill(committedValue);
  await target.dispatchEvent("compositionend", { data: committedValue });
}
