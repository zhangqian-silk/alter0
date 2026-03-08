import { expect, type Locator } from "@playwright/test";

export async function expectInputReady(target: Locator, timeout = 20000): Promise<void> {
  await expect(target).toBeVisible();
  await expect(target).toBeEnabled({ timeout });
}

export async function expectFocusedValue(target: Locator, value: string | RegExp): Promise<void> {
  await expect(target).toBeFocused();
  await expect(target).toHaveValue(value);
}
