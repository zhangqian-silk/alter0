import { expect, type Locator } from "@playwright/test";

export async function startIMEInput(target: Locator, composingValue = "ni"): Promise<void> {
  await target.click();
  await expect(target).toBeFocused();
  await target.dispatchEvent("compositionstart", { data: "n" });
  await target.evaluate((node, value) => {
    const field = node as HTMLInputElement | HTMLTextAreaElement;
    field.value = value;
    field.dispatchEvent(new Event("input", { bubbles: true }));
  }, composingValue);
}

export async function commitIMEInput(target: Locator, committedValue: string): Promise<void> {
  await target.evaluate((node, value) => {
    const field = node as HTMLInputElement | HTMLTextAreaElement;
    field.value = value;
    field.dispatchEvent(new Event("input", { bubbles: true }));
    field.dispatchEvent(new CompositionEvent("compositionend", { bubbles: true, data: value }));
  }, committedValue);
}
