import { expect, type Page } from "@playwright/test";
import { expectComposerReady } from "../asserts/composer";
import { waitForAppReady } from "../guards/app-ready";
import { loginIfNeeded } from "../guards/login";
import { createChatPage } from "../pages/chat";
import { createCronPage } from "../pages/cron";

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
  await page.goto("/settings");
  await ensureAppReady(page);
  if (!new URL(page.url()).pathname.endsWith("/settings")) {
    await page.goto("/settings");
    await ensureAppReady(page);
  }
  await expect(page).toHaveURL(/\/settings(?:\?.*)?$/);
  await page.getByRole("button", { name: /Schedules|定时任务/ }).click();
  await expect(createCronPage(page).routeGrid()).toBeVisible();
}
