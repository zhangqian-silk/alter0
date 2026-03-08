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
  await expectComposerReady(createCronPage(page).composer());
}

export async function openTerminalRoute(page: Page): Promise<void> {
  await page.goto("/chat#terminal");
  await ensureAppReady(page);
  if (!page.url().includes("#terminal")) {
    await page.goto("/chat#terminal");
    await ensureAppReady(page);
  }
  await expect(createTerminalPage(page).createButton()).toBeVisible();
}
