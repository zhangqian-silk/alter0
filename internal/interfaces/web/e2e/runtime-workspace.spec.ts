import { expect, test, type Page } from "@playwright/test";
import { waitForAppReady } from "./helpers/guards/app-ready";
import { loginIfNeeded } from "./helpers/guards/login";
import { openTerminalRoute } from "./helpers/flows/routes";

async function openRuntimeRoute(page: Page, hash: "#chat" | "#agent-runtime"): Promise<void> {
  await page.goto(`/chat${hash}`);
  await loginIfNeeded(page);
  await waitForAppReady(page);
  await page.waitForSelector(".conversation-chat-form", { timeout: 20000 });
}

async function readConversationViewportGap(page: Page) {
  return page.evaluate(() => {
    const screen = document.querySelector(".conversation-chat-screen");
    const composer = document.querySelector(".conversation-composer-shell");
    if (!(screen instanceof HTMLElement) || !(composer instanceof HTMLElement)) {
      return null;
    }
    const screenRect = screen.getBoundingClientRect();
    const composerRect = composer.getBoundingClientRect();
    return {
      screenBottom: screenRect.bottom,
      composerTop: composerRect.top,
      gap: composerRect.top - screenRect.bottom,
    };
  });
}

async function readTerminalViewportGap(page: Page) {
  return page.evaluate(() => {
    const screen = document.querySelector(".terminal-chat-screen");
    const composer = document.querySelector(".terminal-composer-shell");
    if (!(screen instanceof HTMLElement) || !(composer instanceof HTMLElement)) {
      return null;
    }
    const screenRect = screen.getBoundingClientRect();
    const composerRect = composer.getBoundingClientRect();
    return {
      screenBottom: screenRect.bottom,
      composerTop: composerRect.top,
      gap: composerRect.top - screenRect.bottom,
    };
  });
}

test.describe("Runtime workspace scaffold", () => {
  test("submits chat on the first click and keeps the chat viewport above the composer", async ({ page }) => {
    await openRuntimeRoute(page, "#chat");

    const input = page.locator(".conversation-composer-input");
    const submit = page.locator(".conversation-chat-submit");

    await input.fill("first click submit");
    await submit.click();

    await expect(page.locator(".msg.user .msg-bubble").last()).toContainText("first click submit");
    await expect(input).toHaveValue("");

    const metrics = await readConversationViewportGap(page);
    expect(metrics).not.toBeNull();
    expect(metrics?.gap ?? -1).toBeGreaterThanOrEqual(0);
  });

  test("keeps the agent runtime viewport above the composer", async ({ page }) => {
    await openRuntimeRoute(page, "#agent-runtime");

    const metrics = await readConversationViewportGap(page);
    expect(metrics).not.toBeNull();
    expect(metrics?.gap ?? -1).toBeGreaterThanOrEqual(0);
  });

  test("keeps the terminal viewport above the composer", async ({ page }) => {
    await openTerminalRoute(page);

    const metrics = await readTerminalViewportGap(page);
    expect(metrics).not.toBeNull();
    expect(metrics?.gap ?? -1).toBeGreaterThanOrEqual(0);
  });
});
