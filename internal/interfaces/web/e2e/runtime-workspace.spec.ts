import { expect, test, type Page } from "@playwright/test";
import { waitForAppReady } from "./helpers/guards/app-ready";
import { loginIfNeeded } from "./helpers/guards/login";
import { openTerminalRoute } from "./helpers/flows/routes";
import { installVisualViewportMock, setVisualViewport } from "./helpers/support/visual-viewport";

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

  test("submits chat and terminal directly from the mobile send button while the keyboard is open", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 430, height: 932 });

    await openRuntimeRoute(page, "#chat");
    const chatInput = page.locator(".conversation-composer-input");
    const chatSubmit = page.locator(".conversation-chat-submit");
    await chatInput.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");
    await chatInput.fill("tap send with keyboard open");
    await chatSubmit.dispatchEvent("touchstart");
    await expect(page.locator(".msg.user .msg-bubble").last()).toContainText("tap send with keyboard open");
    await expect(chatInput).toHaveValue("");

    await openTerminalRoute(page);
    const terminalInput = page.locator(".terminal-composer-input");
    const terminalSubmit = page.locator("[data-terminal-submit]");
    await terminalInput.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");
    await terminalInput.fill("pwd");
    await terminalSubmit.dispatchEvent("touchstart");
    await expect(terminalInput).toHaveValue("");
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

  test("keeps chat, agent runtime, and terminal viewports above the composer on mobile", async ({ page }) => {
    await page.setViewportSize({ width: 430, height: 932 });

    await openRuntimeRoute(page, "#chat");
    let metrics = await readConversationViewportGap(page);
    expect(metrics).not.toBeNull();
    expect(metrics?.gap ?? -1).toBeGreaterThanOrEqual(0);

    await openRuntimeRoute(page, "#agent-runtime");
    metrics = await readConversationViewportGap(page);
    expect(metrics).not.toBeNull();
    expect(metrics?.gap ?? -1).toBeGreaterThanOrEqual(0);

    await openTerminalRoute(page);
    const terminalMetrics = await readTerminalViewportGap(page);
    expect(terminalMetrics).not.toBeNull();
    expect(terminalMetrics?.gap ?? -1).toBeGreaterThanOrEqual(0);
  });

  test("restores chat and terminal viewport height after the mobile keyboard closes", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 430, height: 932 });

    await openRuntimeRoute(page, "#chat");
    const conversationInput = page.locator(".conversation-composer-input");
    await conversationInput.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");

    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    });
    await setVisualViewport(page, { width: 430, height: 932, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");
    await expect.poll(async () => (await readConversationViewportGap(page))?.gap ?? Number.POSITIVE_INFINITY)
      .toBeLessThanOrEqual(20);

    await openTerminalRoute(page);
    const terminalInput = page.locator(".terminal-composer-input");
    await terminalInput.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");

    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    });
    await setVisualViewport(page, { width: 430, height: 932, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");
    await expect.poll(async () => (await readTerminalViewportGap(page))?.gap ?? Number.POSITIVE_INFINITY)
      .toBeLessThanOrEqual(20);
  });

  test("holds keyboard offset until chat and terminal viewports actually recover after blur", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 430, height: 932 });

    await openRuntimeRoute(page, "#chat");
    const conversationInput = page.locator(".conversation-composer-input");
    await conversationInput.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");

    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");

    await setVisualViewport(page, { width: 430, height: 760, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("172px");

    await setVisualViewport(page, { width: 430, height: 932, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");

    await openTerminalRoute(page);
    const terminalInput = page.locator(".terminal-composer-input");
    await terminalInput.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");

    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");

    await setVisualViewport(page, { width: 430, height: 760, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("172px");

    await setVisualViewport(page, { width: 430, height: 932, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");
  });
});
