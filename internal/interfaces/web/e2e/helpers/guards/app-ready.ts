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
    const appShell = document.getElementById("appShell") || document.querySelector(".app-shell");
    const routeBody =
      document.getElementById("routeBody") ||
      document.querySelector(".route-body, .terminal-route-body, [data-conversation-workspace], [data-terminal-workspace]");
    const routeVisible = routeBody instanceof HTMLElement;
    const navigationReady = document.querySelectorAll(".menu-item[data-route]").length > 0;
    return Boolean(appShell && routeBody && routeVisible && navigationReady);
  }, { timeout });
  await expect(page.locator(".app-shell")).toBeVisible({ timeout });
}
