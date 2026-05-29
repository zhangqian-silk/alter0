import { expect, test, type Page } from "@playwright/test";
import { waitForAppReady } from "./helpers/guards/app-ready";
import { loginIfNeeded } from "./helpers/guards/login";
import { openTerminalRoute } from "./helpers/flows/routes";
import { installVisualViewportMock, setVisualViewport } from "./helpers/support/visual-viewport";

async function openRuntimeRoute(page: Page, hash: "#chat" | "#agent-runtime"): Promise<void> {
  await page.goto(`/chat${hash}`);
  await loginIfNeeded(page);
  if (!page.url().includes(hash)) {
    await page.goto(`/chat${hash}`);
  }
  await waitForAppReady(page);
  const expectedRoute = hash.slice(1);
  await expect(page.locator("[data-runtime-view='conversation']")).toHaveAttribute("data-runtime-route", expectedRoute);
  await page.waitForSelector("[data-composer-form='conversation']", { timeout: 20000 });
}

async function mockConversationRuntimeSessions(
  page: Page,
  options: {
    route: "chat" | "agent-runtime";
    sessions: Array<Record<string, unknown>>;
    activeSessionID?: string;
  },
): Promise<void> {
  const sessionByID = new Map(
    options.sessions.map((session) => [String(session.id || "").trim(), session]),
  );
  const activeSessionID = String(
    options.activeSessionID
      || options.sessions[0]?.id
      || "",
  ).trim();

  if (!activeSessionID) {
    throw new Error("mockConversationRuntimeSessions requires at least one session");
  }

  await page.context().route("**/api/conversation-runtime/sessions**", async (route) => {
    const url = new URL(route.request().url());
    if (url.searchParams.get("route") !== options.route) {
      await route.fallback();
      return;
    }

    if (url.pathname.endsWith("/api/conversation-runtime/sessions")) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          items: options.sessions,
        }),
      });
      return;
    }

    const requestedSessionID = url.pathname.split("/").pop() || "";
    const session = sessionByID.get(requestedSessionID);
    if (session) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ session }),
      });
      return;
    }

    await route.fallback();
  });

  await page.addInitScript((payload) => {
    window.sessionStorage.setItem(
      "alter0.web.session.active.v1",
      JSON.stringify({ [payload.route]: payload.activeSessionID }),
    );
  }, {
    route: options.route,
    activeSessionID,
  });
}

async function readConversationViewportGap(page: Page) {
  return page.evaluate(() => {
    const screen = document.querySelector("[data-runtime-screen='conversation']");
    const composer = document.querySelector(".runtime-composer-shell");
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
    const screen = document.querySelector("[data-runtime-screen='terminal']");
    const composer = document.querySelector(".runtime-composer-shell");
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
  test("keeps the desktop agent runtime session pane scrollable with a long session list", async ({ page }) => {
    const now = Date.now();
    const sessions = Array.from({ length: 20 }, (_, index) => {
      const sessionID = `agent-scroll-${index + 1}`;
      const createdAt = new Date(now - index * 60_000).toISOString();
      return {
        id: sessionID,
        title: `Desktop scroll session ${index + 1}`,
        created_at: createdAt,
        updated_at: createdAt,
        status: "ready",
        target_type: "agent",
        target_id: "coding",
        target_name: "Coding Agent",
        messages: [
          {
            id: `message-${sessionID}`,
            role: "user",
            text: `session ${index + 1}`,
            at: createdAt,
            status: "done",
          },
        ],
      };
    });

    await mockConversationRuntimeSessions(page, {
      route: "agent-runtime",
      sessions,
      activeSessionID: sessions[0].id as string,
    });
    await page.setViewportSize({ width: 1440, height: 960 });
    await openRuntimeRoute(page, "#agent-runtime");

    const sessionList = page.locator("[data-runtime-session-list='conversation']");
    await expect(sessionList.locator("[role='listitem']")).toHaveCount(20);

    const beforeScroll = await sessionList.evaluate((node) => {
      const element = node as HTMLDivElement;
      element.scrollTop = element.scrollHeight;
      return {
        scrollTop: element.scrollTop,
        scrollHeight: element.scrollHeight,
        clientHeight: element.clientHeight,
      };
    });

    expect(beforeScroll.scrollHeight).toBeGreaterThan(beforeScroll.clientHeight);
    expect(beforeScroll.scrollTop).toBeGreaterThan(0);

    await sessionList.locator("[role='listitem']").last().click();

    const afterScrollTop = await sessionList.evaluate((node) => (node as HTMLDivElement).scrollTop);
    expect(afterScrollTop).toBeGreaterThan(0);
    expect(Math.abs(afterScrollTop - beforeScroll.scrollTop)).toBeLessThan(40);
  });

  test("submits chat on the first click and keeps the chat viewport above the composer", async ({ page }) => {
    await openRuntimeRoute(page, "#chat");

    const input = page.locator("[data-composer-input='conversation']");
    const submit = page.locator("[data-composer-submit='conversation']");

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
    const chatInput = page.locator("[data-composer-input='conversation']");
    const chatSubmit = page.locator("[data-composer-submit='conversation']");
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
    const terminalInput = page.locator("[data-composer-input='terminal']");
    const terminalSubmit = page.locator("[data-runtime-submit='terminal']");
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
    const conversationInput = page.locator("[data-composer-input='conversation']");
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
    const terminalInput = page.locator("[data-composer-input='terminal']");
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
    const conversationInput = page.locator("[data-composer-input='conversation']");
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
    const terminalInput = page.locator("[data-composer-input='terminal']");
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
