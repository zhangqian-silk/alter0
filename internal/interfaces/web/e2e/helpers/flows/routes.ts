import { expect, type Page } from "@playwright/test";
import { expectComposerReady } from "../asserts/composer";
import { waitForAppReady } from "../guards/app-ready";
import { loginIfNeeded } from "../guards/login";
import { createChatPage } from "../pages/chat";
import { createCronPage } from "../pages/cron";
import { createTerminalPage } from "../pages/terminal";

async function ensureAppReady(page: Page): Promise<void> {
  await loginIfNeeded(page);
  await waitForAppReady(page);
}

export async function openChatRoute(page: Page): Promise<void> {
  await page.goto("/chat");
  await ensureAppReady(page);
  await expectComposerReady(createChatPage(page).composer());
}

export async function ensureChatRouteReady(page: Page): Promise<void> {
  await ensureAppReady(page);
  await expectComposerReady(createChatPage(page).composer());
}

export async function openCronRoute(page: Page): Promise<void> {
  await page.goto("/chat#cron-jobs");
  await ensureAppReady(page);
  if (!page.url().includes("#cron-jobs")) {
    await page.goto("/chat#cron-jobs");
    await ensureAppReady(page);
  }
  await expect(page).toHaveURL(/#cron-jobs$/);
  await expect(createCronPage(page).routeGrid()).toBeVisible();
}

export async function openTerminalRoute(page: Page): Promise<void> {
  await page.goto("/chat");
  await ensureAppReady(page);
  if (!page.url().includes("#terminal")) {
    await page.evaluate(() => {
      if (window.location.hash !== "#terminal") {
        window.location.hash = "#terminal";
        return;
      }
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });
  }
  await expect(page).toHaveURL(/#terminal$/);
  await expect(page.locator("[data-runtime-view='terminal']")).toBeVisible();
  await expect(page.locator("[data-runtime-workspace='terminal']")).toBeVisible();
  await expectComposerReady(createTerminalPage(page).composer());
}
