import { expect, type Page } from "@playwright/test";

export async function waitForAppReady(page: Page, timeout = 20000): Promise<void> {
  await page.waitForFunction(() => {
    const body = document.body;
    if (!body) {
      return false;
    }
    if (body.getAttribute("data-app-ready") === "true") {
      return true;
    }
    const appShell = document.getElementById("appShell");
    const routeBody = document.getElementById("routeBody");
    const routeVisible = routeBody instanceof HTMLElement && routeBody.childElementCount >= 0;
    const navigationReady = document.querySelectorAll(".menu-item[data-route]").length > 0;
    return Boolean(appShell && routeBody && routeVisible && navigationReady);
  }, { timeout });
  await expect(page.locator("#appShell")).toBeVisible({ timeout });
}
