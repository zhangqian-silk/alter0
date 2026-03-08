import { expect, type Page } from "@playwright/test";

export async function waitForAppReady(page: Page, timeout = 20000): Promise<void> {
  await expect(page.locator("body")).toHaveAttribute("data-app-ready", "true", { timeout });
}
