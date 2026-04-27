import { expect, test } from "@playwright/test";
import {
  expectComposerCounter,
  expectComposerFocusedValue,
  expectComposerReady,
  expectComposerState,
  expectComposerValue,
} from "./helpers/asserts/composer";
import {
  createNewChatSession,
  expectActiveChatSession,
  removeChatSession,
  switchChatSession
} from "./helpers/flows/chat-session";
import { ensureChatRouteReady, openChatRoute } from "./helpers/flows/routes";
import { waitForAppReady } from "./helpers/guards/app-ready";
import { loginIfNeeded } from "./helpers/guards/login";
import { commitIMEInput, startIMEInput } from "./helpers/interactions/ime";
import { clickWithUnsavedDialog } from "./helpers/guards/unsaved";
import {
  openChatWorkspace,
  openChatWorkspaceWithDraft,
  openChatWorkspaceWithTwoDraftSessions,
  reloadChatWorkspace,
} from "./helpers/scenarios/chat";
import { installVisualViewportMock, setVisualViewport } from "./helpers/support/visual-viewport";

async function openChannelsRoute(page: Parameters<typeof loginIfNeeded>[0]): Promise<void> {
  await openChatRoute(page);
  await page.evaluate(() => {
    window.location.hash = "#channels";
    window.dispatchEvent(new HashChangeEvent("hashchange"));
  });
  await waitForAppReady(page);
  await expect.poll(async () => page.locator("#routeView").getAttribute("data-route")).toBe("channels");
  await expect(page.locator("#routeView")).toBeVisible();
}

async function mockRuntimeSession(
  page: Parameters<typeof loginIfNeeded>[0],
  options: {
    route: "chat" | "agent-runtime";
    session: Record<string, unknown>;
    agents?: Array<Record<string, unknown>>;
  },
): Promise<void> {
  const sessionID = String(options.session.id || "").trim();
  if (!sessionID) {
    throw new Error("mockRuntimeSession requires a session id");
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
          items: [options.session],
        }),
      });
      return;
    }
    if (url.pathname.endsWith(`/api/conversation-runtime/sessions/${sessionID}`)) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          session: options.session,
        }),
      });
      return;
    }
    await route.fallback();
  });
  if (options.agents) {
    await page.context().route("**/api/agents", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          items: options.agents,
        }),
      });
    });
  }
}

test.describe("Chat composer", () => {
  test("formats frontend timestamps in Beijing time with a 24-hour clock", async ({ page }) => {
    await openChatRoute(page);

    const formatted = await page.evaluate(() => ({
      dateTime: new Intl.DateTimeFormat("sv-SE", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
        timeZone: "Asia/Shanghai",
      }).format(new Date("2026-04-10T00:05:06Z")).replace(",", ""),
      timeOnly: new Intl.DateTimeFormat("en-GB", {
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
        timeZone: "Asia/Shanghai",
      }).format(new Date("2026-04-10T00:05:00Z"))
    }));

    expect(formatted).toEqual({
      dateTime: "2026-04-10 08:05:06",
      timeOnly: "08:05"
    });

  });

  test("shows a short hash for agent sessions in the list and details panel", async ({ page }) => {
    const sessionID = "db4416b7-452d-44a6-83ca-999e77f47791";
    const createdAt = Date.now();
    await mockRuntimeSession(page, {
      route: "agent-runtime",
      session: {
        id: sessionID,
        title: "修复 Agent 会话标识",
        title_auto: false,
        title_score: 8,
        created_at: new Date(createdAt).toISOString(),
        target_type: "agent",
        target_id: "coding",
        target_name: "Coding Agent",
        model_provider_id: "",
        model_id: "",
        tool_ids: [],
        skill_ids: [],
        mcp_ids: [],
        messages: [{
          id: "message-agent-session-hash",
          role: "user",
          text: "给 Agent 会话加标识",
          at: new Date(createdAt).toISOString(),
          status: "done",
        }],
      },
      agents: [{
        id: "coding",
        name: "Coding Agent",
        enabled: true,
      }],
    });
    await page.addInitScript(() => {
      window.sessionStorage.setItem("alter0.web.session.active.v1", JSON.stringify({
        "agent-runtime": "db4416b7-452d-44a6-83ca-999e77f47791",
      }));
    });

    await page.goto("/chat#agent-runtime");
    await ensureChatRouteReady(page);
    if (!page.url().includes("#agent-runtime")) {
      await page.goto("/chat#agent-runtime");
      await ensureChatRouteReady(page);
    }

    const hashLabel = page.locator(".runtime-session-hash").first();
    const detailsButton = page.getByRole("button", { name: "Details" }).first();
    const shortHash = ((await hashLabel.textContent()) || "").trim();

    expect(shortHash).toMatch(/^[0-9a-f]{8}$/);
    await detailsButton.click();

    await expect(page.locator('[data-runtime-details-panel="conversation"]')).toContainText(shortHash);
  });

  test("keeps empty session hint near the session header", async ({ page }) => {
    await openChatWorkspace(page);

    const heading = page.locator("#sessionHeading");
    const subheading = page.locator("#sessionSubheading");
    const sessionCards = page.locator("#sessionList .session-card");

    await expect(sessionCards).toHaveCount(1);
    await expect(subheading).toContainText("Empty session");

    const headingBox = await heading.boundingBox();
    const subheadingBox = await subheading.boundingBox();

    expect(headingBox).not.toBeNull();
    expect(subheadingBox).not.toBeNull();
    expect((subheadingBox?.y ?? 0) - (headingBox?.y ?? 0)).toBeLessThan(80);
  });

  test("keeps the shell stable while resizing across the desktop and drawer breakpoints", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await openChatWorkspace(page);

    const readShellMetrics = async () =>
      page.evaluate(() => {
        const appShell = document.querySelector(".app-shell");
        const nav = document.querySelector(".primary-nav");
        const sessionPane = document.querySelector(".session-pane");
        const chatPane = document.querySelector(".chat-pane");
        const composerShell = document.querySelector(".composer-shell");
        const navToggle = document.getElementById("navToggle");
        const sessionToggle = document.getElementById("sessionToggle");
        if (
          !(appShell instanceof HTMLElement)
          || !(nav instanceof HTMLElement)
          || !(sessionPane instanceof HTMLElement)
          || !(chatPane instanceof HTMLElement)
          || !(composerShell instanceof HTMLElement)
          || !(navToggle instanceof HTMLElement)
          || !(sessionToggle instanceof HTMLElement)
        ) {
          return null;
        }

        const navRect = nav.getBoundingClientRect();
        const sessionRect = sessionPane.getBoundingClientRect();
        const chatRect = chatPane.getBoundingClientRect();
        const composerRect = composerShell.getBoundingClientRect();
        const navToggleRect = navToggle.getBoundingClientRect();
        const sessionToggleRect = sessionToggle.getBoundingClientRect();
        const shellStyle = getComputedStyle(appShell);
        const navStyle = getComputedStyle(nav);
        const sessionStyle = getComputedStyle(sessionPane);
        const doc = document.documentElement;

        return {
          navPosition: navStyle.position,
          sessionPosition: sessionStyle.position,
          navLeft: navRect.left,
          navRight: navRect.right,
          sessionLeft: sessionRect.left,
          sessionRight: sessionRect.right,
          chatLeft: chatRect.left,
          chatRight: chatRect.right,
          composerBottom: composerRect.bottom,
          navToggleTop: navToggleRect.top,
          sessionToggleTop: sessionToggleRect.top,
          viewportWidth: window.innerWidth,
          viewportHeight: window.innerHeight,
          scrollWidth: doc.scrollWidth,
          gridTemplateColumns: shellStyle.gridTemplateColumns,
          navToggleDisplay: getComputedStyle(navToggle).display,
          sessionToggleDisplay: getComputedStyle(sessionToggle).display,
        };
      });

    const desktop = await readShellMetrics();
    expect(desktop).not.toBeNull();
    expect(desktop?.navPosition).not.toBe("fixed");
    expect(desktop?.sessionPosition).not.toBe("fixed");
    expect(desktop?.appShellClass ?? "").not.toContain("nav-open");
    expect(desktop?.appShellClass ?? "").not.toContain("panel-open");
    expect(desktop?.navRight ?? 0).toBeLessThanOrEqual((desktop?.sessionLeft ?? 0) + 2);
    expect(desktop?.sessionRight ?? 0).toBeLessThanOrEqual((desktop?.chatLeft ?? 0) + 2);
    expect(desktop?.scrollWidth ?? 0).toBeLessThanOrEqual((desktop?.viewportWidth ?? 0) + 1);
    expect(desktop?.gridTemplateColumns.split(" ").length ?? 0).toBeGreaterThanOrEqual(3);
    expect(desktop?.navToggleDisplay).toBe("none");
    expect(desktop?.sessionToggleDisplay).toBe("none");

    await page.setViewportSize({ width: 1024, height: 900 });

    await expect.poll(async () => {
      const metrics = await readShellMetrics();
      return metrics?.navPosition ?? "";
    }).toBe("fixed");

    const drawer = await readShellMetrics();
    expect(drawer).not.toBeNull();
    expect(drawer?.sessionPosition).toBe("fixed");
    expect(drawer?.appShellClass ?? "").not.toContain("nav-open");
    expect(drawer?.appShellClass ?? "").not.toContain("panel-open");
    expect(drawer?.chatLeft ?? 0).toBeLessThanOrEqual(24);
    expect((drawer?.viewportHeight ?? 0) - (drawer?.composerBottom ?? 0)).toBeLessThan(36);
    expect(Math.abs((drawer?.navToggleTop ?? 0) - (drawer?.sessionToggleTop ?? 0))).toBeLessThan(8);
    expect(drawer?.scrollWidth ?? 0).toBeLessThanOrEqual((drawer?.viewportWidth ?? 0) + 1);
    expect(drawer?.navToggleDisplay).not.toBe("none");
    expect(drawer?.sessionToggleDisplay).not.toBe("none");

    await page.setViewportSize({ width: 1180, height: 900 });

    await expect.poll(async () => {
      const metrics = await readShellMetrics();
      return metrics?.navPosition ?? "";
    }).not.toBe("fixed");

    const restoredDesktop = await readShellMetrics();
    expect(restoredDesktop).not.toBeNull();
    expect(restoredDesktop?.appShellClass ?? "").not.toContain("nav-open");
    expect(restoredDesktop?.appShellClass ?? "").not.toContain("panel-open");
    expect(restoredDesktop?.navRight ?? 0).toBeLessThanOrEqual((restoredDesktop?.sessionLeft ?? 0) + 2);
    expect(restoredDesktop?.sessionRight ?? 0).toBeLessThanOrEqual((restoredDesktop?.chatLeft ?? 0) + 2);
    expect(restoredDesktop?.scrollWidth ?? 0).toBeLessThanOrEqual((restoredDesktop?.viewportWidth ?? 0) + 1);
    expect(restoredDesktop?.navToggleDisplay).toBe("none");
    expect(restoredDesktop?.sessionToggleDisplay).toBe("none");
  });

  test("switches exactly at the 1100px shell breakpoint without leaving mixed layout state", async ({ page }) => {
    await openChatWorkspace(page);

    const readBreakpointState = async () =>
      page.evaluate(() => {
        const appShell = document.querySelector(".app-shell");
        const nav = document.querySelector(".primary-nav");
        const sessionPane = document.querySelector(".session-pane");
        const chatPane = document.querySelector(".chat-pane");
        if (
          !(appShell instanceof HTMLElement)
          || !(nav instanceof HTMLElement)
          || !(sessionPane instanceof HTMLElement)
          || !(chatPane instanceof HTMLElement)
        ) {
          return null;
        }

        return {
          viewportWidth: window.innerWidth,
          scrollWidth: document.documentElement.scrollWidth,
          shellClassName: appShell.className,
          gridTemplateColumns: getComputedStyle(appShell).gridTemplateColumns,
          navPosition: getComputedStyle(nav).position,
          sessionPosition: getComputedStyle(sessionPane).position,
          chatLeft: chatPane.getBoundingClientRect().left,
        };
      });

    await page.setViewportSize({ width: 1100, height: 900 });
    await expect.poll(async () => (await readBreakpointState())?.navPosition ?? "").toBe("fixed");

    const mobileEdge = await readBreakpointState();
    expect(mobileEdge).not.toBeNull();
    expect(mobileEdge?.sessionPosition).toBe("fixed");
    expect(mobileEdge?.shellClassName ?? "").not.toContain("nav-open");
    expect(mobileEdge?.shellClassName ?? "").not.toContain("panel-open");
    expect(mobileEdge?.chatLeft ?? 0).toBeLessThanOrEqual(24);
    expect(mobileEdge?.scrollWidth ?? 0).toBeLessThanOrEqual((mobileEdge?.viewportWidth ?? 0) + 1);

    await page.setViewportSize({ width: 1101, height: 900 });
    await expect.poll(async () => (await readBreakpointState())?.navPosition ?? "").not.toBe("fixed");

    const desktopEdge = await readBreakpointState();
    expect(desktopEdge).not.toBeNull();
    expect(desktopEdge?.sessionPosition).not.toBe("fixed");
    expect(desktopEdge?.gridTemplateColumns.split(" ").length ?? 0).toBeGreaterThanOrEqual(3);
    expect(desktopEdge?.shellClassName ?? "").not.toContain("nav-open");
    expect(desktopEdge?.shellClassName ?? "").not.toContain("panel-open");
    expect(desktopEdge?.scrollWidth ?? 0).toBeLessThanOrEqual((desktopEdge?.viewportWidth ?? 0) + 1);
  });

  test("keeps empty chat controls compact on narrow screens", async ({ page }) => {
    await page.setViewportSize({ width: 760, height: 980 });
    await openChatWorkspace(page);

    const header = page.locator(".chat-header");
    const navToggle = page.locator("#navToggle");
    const sessionToggle = page.locator("#sessionToggle");
    const newChatButton = page.locator("#mobileNewChatButton");
    const heading = page.locator("#sessionHeading");
    const welcomeTag = page.locator(".welcome-tag");
    const welcomeHeading = page.locator("#welcomeHeading");
    const promptGrid = page.locator(".prompt-grid");
    const composerShell = page.locator(".composer-shell");
    const runtimeToggles = page.locator("#chatRuntimePanel [data-runtime-toggle]");
    const composerNote = page.locator(".composer-note");
    const composerCounter = page.locator("#charCount");
    const sendButton = page.locator("#sendButton");

    await expect(navToggle).toBeVisible();
    await expect(sessionToggle).toBeVisible();
    await expect(newChatButton).toBeVisible();
    await expect(heading).toBeHidden();
    await expect(runtimeToggles).toHaveCount(1);
    await expect(composerNote).toBeHidden();
    await expect(composerCounter).toBeHidden();
    await expect(runtimeToggles.first()).toContainText(/Session|会话/);
    await expect(runtimeToggles.first()).toContainText(/Tools 0|工具 0/);
    await expect(runtimeToggles.first()).toContainText(/Skills 0|技能 0/);

    const headerBox = await header.boundingBox();
    const navBox = await navToggle.boundingBox();
    const sessionBox = await sessionToggle.boundingBox();
    const newChatBox = await newChatButton.boundingBox();
    const welcomeTagBox = await welcomeTag.boundingBox();
    const welcomeHeadingBox = await welcomeHeading.boundingBox();
    const promptGridBox = await promptGrid.boundingBox();
    const composerBox = await composerShell.boundingBox();
    const runtimeBox = await runtimeToggles.first().boundingBox();
    const sendBox = await sendButton.boundingBox();
    const viewport = page.viewportSize();

    expect(headerBox).not.toBeNull();
    expect(navBox).not.toBeNull();
    expect(sessionBox).not.toBeNull();
    expect(newChatBox).not.toBeNull();
    expect(welcomeTagBox).not.toBeNull();
    expect(welcomeHeadingBox).not.toBeNull();
    expect(promptGridBox).not.toBeNull();
    expect(composerBox).not.toBeNull();
    expect(runtimeBox).not.toBeNull();
    expect(sendBox).not.toBeNull();
    expect(viewport).not.toBeNull();

    expect(Math.abs((navBox?.y ?? 0) - (sessionBox?.y ?? 0))).toBeLessThan(6);
    expect(Math.abs((sessionBox?.y ?? 0) - (newChatBox?.y ?? 0))).toBeLessThan(6);
    expect((welcomeTagBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeGreaterThanOrEqual(10);
    expect((welcomeHeadingBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeLessThan(72);
    expect((composerBox?.y ?? 0) - ((promptGridBox?.y ?? 0) + (promptGridBox?.height ?? 0))).toBeGreaterThanOrEqual(48);
    expect(((viewport?.height ?? 0) - ((composerBox?.y ?? 0) + (composerBox?.height ?? 0)))).toBeLessThan(36);
    expect(Math.abs((runtimeBox?.y ?? 0) - (sendBox?.y ?? 0))).toBeLessThan(6);
    expect(sendBox?.x ?? 0).toBeGreaterThan((runtimeBox?.x ?? 0) + (runtimeBox?.width ?? 0) - 4);

    await runtimeToggles.first().click();
    await expect(page.locator(".composer-runtime-popover-mobile")).toBeVisible();
  });

  test("keeps empty chat controls compact on wide screens", async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 980 });
    await openChatWorkspace(page);

    const welcomeScreen = page.locator(".welcome-screen");
    const header = page.locator(".chat-header");
    const welcomeHeading = page.locator("#welcomeHeading");
    const welcomeDescription = page.locator("#welcomeDescription");
    const promptGrid = page.locator(".prompt-grid");
    const composerShell = page.locator(".composer-shell");

    const welcomeScreenBox = await welcomeScreen.boundingBox();
    const headerBox = await header.boundingBox();
    const welcomeHeadingBox = await welcomeHeading.boundingBox();
    const welcomeDescriptionBox = await welcomeDescription.boundingBox();
    const promptGridBox = await promptGrid.boundingBox();
    const composerBox = await composerShell.boundingBox();

    expect(welcomeScreenBox).not.toBeNull();
    expect(headerBox).not.toBeNull();
    expect(welcomeHeadingBox).not.toBeNull();
    expect(welcomeDescriptionBox).not.toBeNull();
    expect(promptGridBox).not.toBeNull();
    expect(composerBox).not.toBeNull();

    expect(headerBox?.y ?? 0).toBeLessThan(4);
    expect(
      Math.abs(
        ((welcomeScreenBox?.y ?? 0) + (welcomeScreenBox?.height ?? 0) / 2)
          - ((((headerBox?.y ?? 0) + (headerBox?.height ?? 0)) + (composerBox?.y ?? 0)) / 2),
      ),
    ).toBeLessThan(48);
    expect(
      Math.abs(
        ((welcomeHeadingBox?.x ?? 0) + (welcomeHeadingBox?.width ?? 0) / 2)
          - ((welcomeScreenBox?.x ?? 0) + (welcomeScreenBox?.width ?? 0) / 2),
      ),
    ).toBeLessThan(24);
    expect(
      Math.abs(
        ((welcomeDescriptionBox?.x ?? 0) + (welcomeDescriptionBox?.width ?? 0) / 2)
          - ((welcomeScreenBox?.x ?? 0) + (welcomeScreenBox?.width ?? 0) / 2),
      ),
    ).toBeLessThan(24);
    expect(
      Math.abs(
        ((promptGridBox?.x ?? 0) + (promptGridBox?.width ?? 0) / 2)
          - ((welcomeScreenBox?.x ?? 0) + (welcomeScreenBox?.width ?? 0) / 2),
      ),
    ).toBeLessThan(24);
  });

  test("keeps the empty chat header separated from welcome content on medium screens", async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 900 });
    await openChatWorkspace(page);

    const header = page.locator(".chat-header");
    const sessionHeading = page.locator("#sessionHeading");
    const sessionSubheading = page.locator("#sessionSubheading");
    const welcomeScreen = page.locator(".welcome-screen");
    const welcomeTag = page.locator(".welcome-tag");
    const welcomeHeading = page.locator("#welcomeHeading");
    const composerShell = page.locator(".composer-shell");

    await expect(sessionHeading).toBeHidden();
    await expect(sessionSubheading).toBeHidden();

    const headerBox = await header.boundingBox();
    const welcomeScreenBox = await welcomeScreen.boundingBox();
    const welcomeTagBox = await welcomeTag.boundingBox();
    const welcomeHeadingBox = await welcomeHeading.boundingBox();
    const composerBox = await composerShell.boundingBox();
    const viewport = page.viewportSize();

    expect(headerBox).not.toBeNull();
    expect(welcomeScreenBox).not.toBeNull();
    expect(welcomeTagBox).not.toBeNull();
    expect(welcomeHeadingBox).not.toBeNull();
    expect(composerBox).not.toBeNull();
    expect(viewport).not.toBeNull();

    expect((welcomeTagBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeGreaterThanOrEqual(12);
    expect((welcomeHeadingBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeGreaterThanOrEqual(28);
    expect(
      Math.abs(
        ((welcomeScreenBox?.y ?? 0) + (welcomeScreenBox?.height ?? 0) / 2)
          - ((((headerBox?.y ?? 0) + (headerBox?.height ?? 0)) + (composerBox?.y ?? 0)) / 2),
      ),
    ).toBeLessThan(60);
    expect(((viewport?.height ?? 0) - ((composerBox?.y ?? 0) + (composerBox?.height ?? 0)))).toBeLessThan(40);
  });

  test("keeps the empty chat viewport pinned to the top when loading the #chat route", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1024, height: 900 });
    await page.goto("/chat#chat");
    await ensureChatRouteReady(page);

    const header = page.locator(".chat-pane.empty-state .chat-header");
    const welcomeHeading = page.locator("#welcomeHeading");
    const composerShell = page.locator(".chat-pane.empty-state .composer-shell");

    await expect(header).toBeVisible();
    await expect(welcomeHeading).toBeVisible();
    await expect(composerShell).toBeVisible();

    const metrics = await page.evaluate(() => {
      const chatPaneNode = document.querySelector(".chat-pane.empty-state");
      const headerNode = document.querySelector(".chat-pane.empty-state .chat-header");
      const welcomeNode = document.getElementById("welcomeHeading");
      const composerNode = document.querySelector(".chat-pane.empty-state .composer-shell");
      if (
        !(chatPaneNode instanceof HTMLElement)
        || !(headerNode instanceof HTMLElement)
        || !(welcomeNode instanceof HTMLElement)
        || !(composerNode instanceof HTMLElement)
      ) {
        return null;
      }
      const chatPaneRect = chatPaneNode.getBoundingClientRect();
      const headerRect = headerNode.getBoundingClientRect();
      const welcomeRect = welcomeNode.getBoundingClientRect();
      const composerRect = composerNode.getBoundingClientRect();
      return {
        scrollY: window.scrollY,
        chatPaneTop: chatPaneRect.top,
        headerTop: headerRect.top,
        welcomeCenter: welcomeRect.top + welcomeRect.height / 2,
        contentCenter: ((headerRect.top + headerRect.height) + composerRect.top) / 2,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.scrollY ?? 0).toBeLessThanOrEqual(4);
    expect(metrics?.chatPaneTop ?? 999).toBeLessThanOrEqual(4);
    expect(metrics?.headerTop ?? 999).toBeLessThanOrEqual(16);
    expect(Math.abs((metrics?.welcomeCenter ?? 999) - (metrics?.contentCenter ?? 0))).toBeLessThan(96);
  });

  test("keeps shared page routes compact on narrow screens", async ({ page }) => {
    await page.setViewportSize({ width: 760, height: 980 });
    await openChannelsRoute(page);

    const header = page.locator(".chat-header");
    const navToggle = page.locator("#navToggle");
    const sessionToggle = page.locator("#sessionToggle");
    const newChatButton = page.locator("#mobileNewChatButton");
    const routeTitle = page.locator("#routeTitle");

    await expect(navToggle).toBeVisible();
    await expect(sessionToggle).toBeHidden();
    await expect(newChatButton).toBeVisible();
    await expect(routeTitle).not.toHaveText("");

    const headerBox = await header.boundingBox();
    const navBox = await navToggle.boundingBox();
    const routeTitleBox = await routeTitle.boundingBox();

    expect(headerBox).not.toBeNull();
    expect(navBox).not.toBeNull();
    expect(routeTitleBox).not.toBeNull();
    expect((routeTitleBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeLessThan(64);
    expect((routeTitleBox?.y ?? 0) - (navBox?.y ?? 0)).toBeGreaterThan((navBox?.height ?? 0) - 2);
  });

  test("expands the desktop chat reading column on wide screens", async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    const { chatPage, composer } = await openChatWorkspace(page);

    await composer.input().fill("请输出一段稍长的说明，用于验证桌面宽屏聊天布局。");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("请输出一段稍长的说明");
    await expect(page.locator(".session-card-title").first()).toContainText("请输出一段稍长的说明");

    const metrics = await page.evaluate(() => {
      const composerFrame = document.querySelector(".composer-shell .chat-content-frame");
      if (!(composerFrame instanceof HTMLElement)) {
        return null;
      }
      const composerRect = composerFrame.getBoundingClientRect();
      return {
        composerWidth: composerRect.width,
        composerLeft: composerRect.left,
        composerRight: composerRect.right,
        composerCenter: composerRect.left + composerRect.width / 2,
        scrollWidth: document.documentElement.scrollWidth,
        viewportWidth: window.innerWidth,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.composerWidth ?? 0).toBeGreaterThanOrEqual(940);
    expect(metrics?.composerWidth ?? 0).toBeLessThanOrEqual(964);
    expect(metrics?.composerLeft ?? 0).toBeGreaterThan(0);
    expect(metrics?.composerRight ?? 0).toBeLessThanOrEqual((metrics?.viewportWidth ?? 0) + 1);
    expect(metrics?.scrollWidth ?? 0).toBeLessThanOrEqual((metrics?.viewportWidth ?? 0) + 1);
  });

  test("keeps the empty-state welcome copy and composer aligned on wide screens", async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await openChatWorkspace(page);

    const metrics = await page.evaluate(() => {
      const welcomeFrame = document.querySelector(".welcome-screen .chat-content-frame");
      const composerFrame = document.querySelector(".composer-shell .chat-content-frame");
      if (!(welcomeFrame instanceof HTMLElement) || !(composerFrame instanceof HTMLElement)) {
        return null;
      }
      const welcomeRect = welcomeFrame.getBoundingClientRect();
      const composerRect = composerFrame.getBoundingClientRect();
      return {
        welcomeWidth: welcomeRect.width,
        composerWidth: composerRect.width,
        welcomeCenter: welcomeRect.left + welcomeRect.width / 2,
        composerCenter: composerRect.left + composerRect.width / 2,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.welcomeWidth ?? 0).toBeGreaterThanOrEqual(940);
    expect(metrics?.composerWidth ?? 0).toBeGreaterThanOrEqual(940);
    expect(metrics?.welcomeWidth ?? 0).toBeLessThanOrEqual(964);
    expect(metrics?.composerWidth ?? 0).toBeLessThanOrEqual(964);
    expect(Math.abs((metrics?.welcomeWidth ?? 0) - (metrics?.composerWidth ?? 0))).toBeLessThanOrEqual(4);
    expect(Math.abs((metrics?.welcomeCenter ?? 0) - (metrics?.composerCenter ?? 0))).toBeLessThan(4);
  });

  test("shows route jump controls on managed pages and jumps between sections", async ({ page }) => {
    await page.setViewportSize({ width: 760, height: 720 });
    await openChatWorkspace(page);
    await page.goto("/chat#agent");
    await expect(page.locator("#routeView[data-route='agent']")).toBeVisible();

    const routeView = page.locator("#routeView");
    const jumpTopButton = page.locator("[data-scroll-jump-top='route']");
    const jumpPrevButton = page.locator("[data-scroll-jump-prev='route']");
    const jumpNextButton = page.locator("[data-scroll-jump-next='route']");
    const jumpBottomButton = page.locator("[data-scroll-jump-bottom='route']");

    await expect.poll(async () => routeView.evaluate((node) => node.scrollHeight > node.clientHeight + 180)).toBe(true);

    await routeView.evaluate((node) => {
      node.scrollTop = Math.max((node.scrollHeight - node.clientHeight) * 0.35, 0);
      node.dispatchEvent(new Event("scroll"));
    });

    await expect(jumpTopButton).toHaveClass(/is-visible/);
    await expect(jumpBottomButton).toHaveClass(/is-visible/);
    const prevTarget = await jumpPrevButton.getAttribute("data-scroll-jump-target");
    const nextTarget = await jumpNextButton.getAttribute("data-scroll-jump-target");
    const jumpNavigationButton = prevTarget ? jumpPrevButton : jumpNextButton;
    const jumpNavigationTarget = prevTarget || nextTarget;

    expect(jumpNavigationTarget).toBeTruthy();
    await expect(jumpNavigationButton).toHaveClass(/is-visible/);

    await jumpNavigationButton.click();
    await expect.poll(async () => routeView.evaluate((node, targetID) => {
      const target = node.querySelector(`[data-scroll-jump-anchor="${String(targetID || "")}"]`);
      if (!(target instanceof HTMLElement)) {
        return null;
      }
      return Math.round(target.getBoundingClientRect().top - node.getBoundingClientRect().top);
    }, jumpNavigationTarget)).toBeLessThanOrEqual(24);

    await jumpBottomButton.click();
    await expect.poll(async () => routeView.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop)).toBeLessThan(12);

    await jumpTopButton.click();
    await expect.poll(async () => routeView.evaluate((node) => Math.round(node.scrollTop))).toBeLessThanOrEqual(8);
  });

  test("renders mobile session settings as an independent bottom sheet", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    await openChatWorkspace(page);

    const runtimeToggle = page.locator("#chatRuntimePanel [data-runtime-toggle]").first();
    const composerShell = page.locator(".composer-shell");
    const popover = page.locator(".composer-runtime-popover-mobile");
    const backdrop = page.locator(".composer-runtime-sheet-backdrop");
    const closeButton = page.locator(".composer-runtime-popover-mobile-close");

    await runtimeToggle.click();

    await expect(backdrop).toBeVisible();
    await expect(popover).toBeVisible();
    await expect(closeButton).toBeVisible();

    const composerBox = await composerShell.boundingBox();
    const popoverBox = await popover.boundingBox();
    const popoverPosition = await popover.evaluate((node) => getComputedStyle(node).position);
    const sheetDetachedFromComposer = await page.evaluate(() => {
      const popoverNode = document.querySelector(".composer-runtime-popover-mobile");
      const panelNode = document.getElementById("chatRuntimePanel");
      if (!(popoverNode instanceof HTMLElement) || !(panelNode instanceof HTMLElement)) {
        return false;
      }
      return !panelNode.contains(popoverNode);
    });
    const bottomLayerHit = await page.evaluate(() => {
      const popoverNode = document.querySelector(".composer-runtime-popover-mobile");
      if (!(popoverNode instanceof HTMLElement)) {
        return false;
      }
      const rect = popoverNode.getBoundingClientRect();
      const sampleX = Math.min(rect.right - 18, Math.max(rect.left + 18, rect.left + rect.width / 2));
      const sampleY = Math.max(rect.top + 18, rect.bottom - 18);
      const hit = document.elementFromPoint(sampleX, sampleY);
      return popoverNode.contains(hit);
    });

    expect(composerBox).not.toBeNull();
    expect(popoverBox).not.toBeNull();
    expect((popoverBox?.y ?? 0) + (popoverBox?.height ?? 0)).toBeGreaterThan((composerBox?.y ?? 0) - 2);
    expect(popoverPosition).toBe("fixed");
    expect(sheetDetachedFromComposer).toBe(true);
    expect(bottomLayerHit).toBe(true);

    await closeButton.click();
    await expect(popover).toBeHidden();
    await expect(backdrop).toBeHidden();
  });

  test("keeps 393px mobile drawers contained and dismissible without horizontal overflow", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    await openChatWorkspace(page);

    const appShell = page.locator(".app-shell");
    const navToggle = page.locator("#navToggle");
    const sessionToggle = page.locator("#sessionToggle");
    const primaryNav = page.locator(".primary-nav");
    const sessionPane = page.locator(".session-pane");
    const backdrop = page.locator("#mobileBackdrop");

    const readMetrics = async () =>
      page.evaluate(() => {
        const root = document.documentElement;
        const shell = document.querySelector(".app-shell");
        const nav = document.querySelector(".primary-nav");
        const session = document.querySelector(".session-pane");
        if (
          !(shell instanceof HTMLElement)
          || !(nav instanceof HTMLElement)
          || !(session instanceof HTMLElement)
        ) {
          return null;
        }
        const navRect = nav.getBoundingClientRect();
        const sessionRect = session.getBoundingClientRect();
        return {
          viewportWidth: window.innerWidth,
          scrollWidth: root.scrollWidth,
          shellClassName: shell.className,
          navLeft: navRect.left,
          navRight: navRect.right,
          sessionLeft: sessionRect.left,
          sessionRight: sessionRect.right,
        };
      });

    await navToggle.click();
    await expect(appShell).toHaveClass(/nav-open/);
    await expect(backdrop).toBeVisible();
    await expect(primaryNav).toBeVisible();
    await expect.poll(async () => (await readMetrics())?.navLeft ?? Number.NEGATIVE_INFINITY).toBeGreaterThanOrEqual(-1);

    const navOpen = await readMetrics();
    expect(navOpen).not.toBeNull();
    expect(navOpen?.navLeft ?? 0).toBeGreaterThanOrEqual(-1);
    expect(navOpen?.navRight ?? 0).toBeLessThanOrEqual((navOpen?.viewportWidth ?? 0) + 1);
    expect(navOpen?.scrollWidth ?? 0).toBeLessThanOrEqual((navOpen?.viewportWidth ?? 0) + 1);

    await backdrop.dispatchEvent("click");
    await expect(appShell).not.toHaveClass(/nav-open/);
    await expect(appShell).not.toHaveClass(/overlay-open/);

    await sessionToggle.click();
    await expect(appShell).toHaveClass(/panel-open/);
    await expect(sessionPane).toBeVisible();
    await expect
      .poll(async () => {
        const metrics = await readMetrics();
        if (!metrics) {
          return Number.POSITIVE_INFINITY;
        }
        return metrics.sessionRight - metrics.viewportWidth;
      })
      .toBeLessThanOrEqual(1);

    const panelOpen = await readMetrics();
    expect(panelOpen).not.toBeNull();
    expect(panelOpen?.sessionLeft ?? 0).toBeGreaterThanOrEqual(-1);
    expect(panelOpen?.sessionRight ?? 0).toBeLessThanOrEqual((panelOpen?.viewportWidth ?? 0) + 1);
    expect(panelOpen?.scrollWidth ?? 0).toBeLessThanOrEqual((panelOpen?.viewportWidth ?? 0) + 1);

    await backdrop.dispatchEvent("click");
    await expect(appShell).not.toHaveClass(/panel-open/);
    await expect(appShell).not.toHaveClass(/overlay-open/);
  });

  test("keeps agent option copy concise inside session settings", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    await openChatWorkspace(page);
    await page.goto("/chat#agent-runtime");

    const runtimeToggle = page.locator("#chatRuntimePanel [data-runtime-toggle]").first();
    await expect(runtimeToggle).toBeVisible();
    await runtimeToggle.click();

    const codingOption = page.locator("[data-runtime-target-id='coding']").first();
    await expect(codingOption).toBeVisible();
    await expect(codingOption).toContainText("Coding Agent");
    await expect(codingOption).toContainText("Dedicated coding agent");
    await expect(codingOption).not.toContainText("Act as alter0's dedicated coding user proxy");
  });

  test("keeps session settings scroll position while toggling skills", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    await openChatWorkspace(page);
    await page.goto("/chat#agent-runtime");

    const runtimeToggle = page.locator("#chatRuntimePanel [data-runtime-toggle]").first();
    await expect(runtimeToggle).toBeVisible();
    await runtimeToggle.click();

    const body = page.locator(".composer-runtime-popover-mobile-body");
    await expect(body).toBeVisible();

    const before = await body.evaluate((node) => {
      node.scrollTop = Math.max(node.scrollHeight - node.clientHeight - 48, 0);
      return node.scrollTop;
    });

    expect(before).toBeGreaterThan(120);

    const toggled = await page.evaluate(() => {
      const input = document.querySelector(".composer-runtime-popover-mobile-body input[data-runtime-toggle-item='skills'][value='memory']");
      if (!(input instanceof HTMLInputElement)) {
        return false;
      }
      input.click();
      return true;
    });

    expect(toggled).toBe(true);

    await expect.poll(async () => body.evaluate((node) => node.scrollTop)).toBeGreaterThan(120);
    const after = await body.evaluate((node) => node.scrollTop);
    expect(Math.abs(after - before)).toBeLessThan(80);
  });

  test("dismisses the mobile keyboard before opening the session settings sheet", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 393, height: 852 });
    const { composer } = await openChatWorkspace(page);

    const runtimeToggle = page.locator("#chatRuntimePanel [data-runtime-toggle]").first();
    const popover = page.locator(".composer-runtime-popover-mobile");
    const backdrop = page.locator(".composer-runtime-sheet-backdrop");
    const input = composer.input();

    await input.click();
    await setVisualViewport(page, { width: 393, height: 520, offsetTop: 0 });

    await expect(input).toBeFocused();
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("332px");

    await runtimeToggle.click();
    await setVisualViewport(page, { width: 393, height: 852, offsetTop: 0 });

    await expect(input).not.toBeFocused();
    await expect(popover).toBeVisible();
    await expect(backdrop).toBeVisible();
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");

    const metrics = await page.evaluate(() => {
      const sheet = document.querySelector(".composer-runtime-popover-mobile");
      const viewport = window.visualViewport;
      if (!(sheet instanceof HTMLElement) || !viewport) {
        return null;
      }
      return {
        viewportBottom: viewport.height + viewport.offsetTop,
        sheetBottom: sheet.getBoundingClientRect().bottom,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.sheetBottom ?? 0).toBeLessThanOrEqual((metrics?.viewportBottom ?? 0) + 2);
  });

  test("keeps the chat composer visible while the mobile keyboard changes the visual viewport", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 760, height: 980 });
    const { composer } = await openChatWorkspace(page);
    const input = composer.input();

    await input.click();
    await setVisualViewport(page, { width: 760, height: 620, offsetTop: 0 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("360px");

    const opened = await page.evaluate(() => {
      const shell = document.querySelector(".composer-shell");
      const inputNode = document.getElementById("composerInput");
      const viewport = window.visualViewport;
      if (!(shell instanceof HTMLElement) || !(inputNode instanceof HTMLElement) || !viewport) {
        return null;
      }
      const shellRect = shell.getBoundingClientRect();
      const inputRect = inputNode.getBoundingClientRect();
      return {
        viewportBottom: viewport.height + viewport.offsetTop,
        shellBottom: shellRect.bottom,
        inputBottom: inputRect.bottom,
      };
    });

    expect(opened).not.toBeNull();
    expect(opened?.shellBottom ?? 0).toBeLessThanOrEqual((opened?.viewportBottom ?? 0) + 2);
    expect(opened?.inputBottom ?? 0).toBeLessThanOrEqual((opened?.viewportBottom ?? 0) - 8);

    await setVisualViewport(page, { width: 760, height: 980, offsetTop: 0 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--mobile-viewport-height").trim()
    )).toBe("980px");

    const closed = await page.evaluate(() => {
      const shell = document.querySelector(".composer-shell");
      const viewport = window.visualViewport;
      if (!(shell instanceof HTMLElement) || !viewport) {
        return null;
      }
      const shellRect = shell.getBoundingClientRect();
      return {
        viewportBottom: viewport.height + viewport.offsetTop,
        shellBottom: shellRect.bottom,
      };
    });

    expect(closed).not.toBeNull();
    expect(Math.abs((closed?.viewportBottom ?? 0) - (closed?.shellBottom ?? 0))).toBeLessThan(20);
  });

  test("keeps chat chrome fixed while only the composer follows the mobile keyboard", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 430, height: 932 });
    await page.goto("/chat#chat");
    await loginIfNeeded(page);
    await waitForAppReady(page);
    await page.waitForSelector("[data-composer-form='conversation']", { timeout: 20000 });
    const input = page.locator("[data-composer-input='conversation']");

    const readMetrics = async () => page.evaluate(() => {
      const mobileHeader = document.querySelector("[data-runtime-mobile-variant='conversation']");
      const workspaceHeader = document.querySelector("[data-runtime-workspace-header='true']");
      const composerShell = document.querySelector(".runtime-composer-shell");
      const viewport = window.visualViewport;
      if (
        !(mobileHeader instanceof HTMLElement)
        || !(workspaceHeader instanceof HTMLElement)
        || !(composerShell instanceof HTMLElement)
        || !viewport
      ) {
        return null;
      }

      const mobileHeaderRect = mobileHeader.getBoundingClientRect();
      const workspaceHeaderRect = workspaceHeader.getBoundingClientRect();
      const composerRect = composerShell.getBoundingClientRect();
      return {
        keyboardOffset: getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim(),
        viewportBottom: viewport.height + viewport.offsetTop,
        mobileHeaderTop: mobileHeaderRect.top,
        mobileHeaderBottom: mobileHeaderRect.bottom,
        workspaceHeaderTop: workspaceHeaderRect.top,
        workspaceHeaderBottom: workspaceHeaderRect.bottom,
        composerTop: composerRect.top,
        composerBottom: composerRect.bottom,
      };
    });

    const baseline = await readMetrics();
    expect(baseline).not.toBeNull();
    expect(baseline?.keyboardOffset).toBe("0px");

    await input.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");
    await expect.poll(async () => {
      const current = await readMetrics();
      if (!current) {
        return Number.POSITIVE_INFINITY;
      }
      return current.composerBottom - current.viewportBottom;
    }).toBeLessThanOrEqual(2);

    const opened = await readMetrics();
    expect(opened).not.toBeNull();
    expect(opened?.keyboardOffset).toBe("312px");
    expect(Math.abs((opened?.mobileHeaderTop ?? 0) - (baseline?.mobileHeaderTop ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.mobileHeaderBottom ?? 0) - (baseline?.mobileHeaderBottom ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.workspaceHeaderTop ?? 0) - (baseline?.workspaceHeaderTop ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.workspaceHeaderBottom ?? 0) - (baseline?.workspaceHeaderBottom ?? 0))).toBeLessThanOrEqual(2);
    expect((baseline?.composerTop ?? 0) - (opened?.composerTop ?? 0)).toBeGreaterThan(120);

    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    });
    await setVisualViewport(page, { width: 430, height: 932, offsetTop: 0 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");
    await expect.poll(async () => {
      const current = await readMetrics();
      if (!current) {
        return Number.POSITIVE_INFINITY;
      }
      return Math.abs(current.composerBottom - current.viewportBottom);
    }).toBeLessThanOrEqual(20);

    const closed = await readMetrics();
    expect(closed).not.toBeNull();
    expect(Math.abs((closed?.mobileHeaderTop ?? 0) - (baseline?.mobileHeaderTop ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((closed?.mobileHeaderBottom ?? 0) - (baseline?.mobileHeaderBottom ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((closed?.workspaceHeaderTop ?? 0) - (baseline?.workspaceHeaderTop ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((closed?.workspaceHeaderBottom ?? 0) - (baseline?.workspaceHeaderBottom ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((closed?.composerTop ?? 0) - (baseline?.composerTop ?? 0))).toBeLessThanOrEqual(20);
  });

  test("keeps mobile route pages aligned to the visual viewport bottom", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 760, height: 980 });
    await openChannelsRoute(page);

    await setVisualViewport(page, { width: 760, height: 760, offsetTop: 0 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--mobile-viewport-height").trim()
    )).toBe("760px");

    const metrics = await page.evaluate(() => {
      const pane = document.querySelector(".chat-pane");
      const route = document.getElementById("routeView");
      const viewport = window.visualViewport;
      if (!(pane instanceof HTMLElement) || !(route instanceof HTMLElement) || !viewport) {
        return null;
      }
      return {
        viewportBottom: viewport.height + viewport.offsetTop,
        paneBottom: pane.getBoundingClientRect().bottom,
        routeBottom: route.getBoundingClientRect().bottom,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.paneBottom ?? 0).toBeLessThanOrEqual((metrics?.viewportBottom ?? 0) + 2);
    expect(metrics?.routeBottom ?? 0).toBeLessThanOrEqual((metrics?.paneBottom ?? 0) + 10);
  });

  test("keeps the mobile navigation fully reachable on short viewports", async ({ page }) => {
    await page.setViewportSize({ width: 760, height: 680 });
    await openChatWorkspace(page);

    const navToggle = page.locator("#navToggle");
    const primaryNav = page.locator(".primary-nav");
    const localeButton = page.locator(".nav-locale-button");

    await navToggle.click();
    await expect(primaryNav).toBeVisible();

    const before = await primaryNav.evaluate((node) => ({
      scrollHeight: node.scrollHeight,
      clientHeight: node.clientHeight,
      scrollTop: node.scrollTop,
    }));

    expect(before.scrollHeight).toBeGreaterThan(before.clientHeight);

    await primaryNav.evaluate((node) => {
      node.scrollTop = node.scrollHeight;
    });

    const after = await primaryNav.evaluate((node) => ({
      scrollTop: node.scrollTop,
      top: node.getBoundingClientRect().top,
      bottom: node.getBoundingClientRect().bottom,
    }));
    const localeBox = await localeButton.boundingBox();

    expect(after.scrollTop).toBeGreaterThan(0);
    expect(localeBox).not.toBeNull();
    expect(localeBox?.y ?? 0).toBeGreaterThanOrEqual((after.top ?? 0) - 1);
    expect((localeBox?.y ?? 0) + (localeBox?.height ?? 0)).toBeLessThanOrEqual((after.bottom ?? 0) + 1);
  });

  test("shows detailed explanations for environment variables", async ({ page }) => {
    const { appShellPage } = await openChatWorkspace(page);

    await appShellPage.routeMenuItem("environments").click();

    const webAddrCard = page.locator(".environment-item").filter({ hasText: "web_addr" }).first();
    const llmTemperatureCard = page.locator(".environment-item").filter({ hasText: "llm_temperature" }).first();

    await expect(webAddrCard).toContainText("控制 HTTP 服务监听的 host:port");
    await expect(webAddrCard).toContainText("浏览器和反向代理都会连接到这里");
    await expect(llmTemperatureCard).toContainText("控制模型采样温度");
    await expect(llmTemperatureCard).toContainText("值越低，输出越稳定和保守");
  });

  test("keeps environment details collapsed until expanded", async ({ page }) => {
    const { appShellPage } = await openChatWorkspace(page);

    await appShellPage.routeMenuItem("environments").click();

    const webAddrCard = page.locator(".environment-item").filter({ hasText: "web_addr" }).first();
    const details = webAddrCard.locator("details.environment-details");
    const summary = details.locator("summary");
    const valueTypeRow = details.locator(".route-field-row").filter({ hasText: "Value Type" });
    const effectiveRow = details.locator(".route-field-row").filter({ hasText: "Effective" });

    await expect(details).not.toHaveAttribute("open", "");
    await expect(valueTypeRow).toBeHidden();

    await summary.click();

    await expect(details).toHaveAttribute("open", "");
    await expect(valueTypeRow).toBeVisible();
    await expect(effectiveRow).toBeVisible();
  });

  test("prompts before leaving with unsent content", async ({ page }) => {
    const { appShellPage, composer } = await openChatWorkspaceWithDraft(page, "unsent draft");
    await expectComposerState(composer, { draft: "dirty" });

    await clickWithUnsavedDialog(page, appShellPage.routeMenuItem("terminal"), "dismiss");
    await expect(page).toHaveURL(/\/chat(?:#chat)?$/);

    await clickWithUnsavedDialog(page, appShellPage.routeMenuItem("terminal"), "accept");
    await expect(page).toHaveURL(/#terminal$/);
  });

  test("restores draft and char count after reload", async ({ page }) => {
    const { composer } = await openChatWorkspace(page);
    const input = composer.input();

    await input.click();
    await input.pressSequentially("draft message");
    await expectComposerValue(composer, "draft message");
    await expectComposerCounter(composer, "13/10000");
    await expectComposerState(composer, { draft: "dirty" });

    const { composer: reloadedComposer } = await reloadChatWorkspace(page);

    await expectComposerValue(reloadedComposer, "draft message");
    await expectComposerCounter(reloadedComposer, "13/10000");
    await expectComposerState(reloadedComposer, { draft: "dirty" });
  });

  test("isolates drafts across chat sessions", async ({ page }) => {
    const { composer } = await openChatWorkspaceWithTwoDraftSessions(page);

    await switchChatSession(page, 1);
    await expectComposerValue(composer, "draft-a");

    await switchChatSession(page, 0);
    await expectComposerValue(composer, "draft-b");
  });

  test("cleans deleted session draft and restores remaining session draft", async ({ page }) => {
    const { composer, sessionCards } = await openChatWorkspaceWithTwoDraftSessions(page);

    await removeChatSession(page, 0);

    await expect(sessionCards).toHaveCount(1);
    await expectComposerValue(composer, "draft-a");

    await createNewChatSession(page);

    await expect(sessionCards).toHaveCount(2);
    await expectComposerValue(composer, "");
    await expectComposerState(composer, { draft: "empty" });
  });

  test("restores session scoped drafts after reload", async ({ page }) => {
    await openChatWorkspaceWithTwoDraftSessions(page);

    const { composer } = await reloadChatWorkspace(page);
    await expectComposerValue(composer, "draft-b");

    await switchChatSession(page, 1);
    await expectComposerValue(composer, "draft-a");
  });

  test("keeps current draft when session switch is cancelled or inactive session is removed", async ({ page }) => {
    const { composer, sessionCards } = await openChatWorkspaceWithTwoDraftSessions(page);

    await switchChatSession(page, 1, "dismiss");

    await expectComposerValue(composer, "draft-b");
    await expectActiveChatSession(page, 0);

    await removeChatSession(page, 1, false);
    await expect(sessionCards).toHaveCount(1);
    await expectComposerValue(composer, "draft-b");
    await expectActiveChatSession(page, 0);
  });

  test("keeps IME composition text when pressing Enter during composition", async ({ page }) => {
    const { composer } = await openChatWorkspace(page);
    const input = composer.input();
    await startIMEInput(input);
    await expectComposerState(composer, { composing: true, draft: "dirty" });

    await input.press("Enter");

    await expectComposerFocusedValue(composer, "ni");

    await commitIMEInput(input, "浣?");

    await expectComposerValue(composer, "浣?");
    await expectComposerState(composer, { composing: false, draft: "dirty" });
  });
  test("clears composer value and draft after sending", async ({ page }) => {
    const { chatPage, composer } = await openChatWorkspace(page);
    const input = composer.input();

    await input.fill("clear-after-send");
    await composer.submitButton().click();

    await expect(chatPage.latestUserBubble()).toContainText("clear-after-send");
    await expectComposerValue(composer, "");
    await expectComposerState(composer, { draft: "empty" });
  });

  test("keeps the first auto title stable after later user prompts", async ({ page }) => {
    const { chatPage, composer } = await openChatWorkspace(page);
    const input = composer.input();

    await input.fill("先拉取仓库");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("先拉取仓库");
    await expectComposerReady(composer);
    await expect(page.locator(".runtime-session-title").first()).toContainText("先拉取仓库");

    await input.fill("修改 terminal 和 agent 的会话标题");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("修改 terminal 和 agent 的会话标题");
    await expectComposerReady(composer);
    await expect(page.locator(".runtime-session-title").first()).toContainText("先拉取仓库");
  });

  test("keeps a generated title unchanged after later follow-up prompts", async ({ page }) => {
    const { chatPage, composer } = await openChatWorkspace(page);
    const input = composer.input();

    await input.fill("排查会话标题逻辑");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("排查会话标题逻辑");
    await expectComposerReady(composer);
    await expect(page.locator(".runtime-session-title").first()).toContainText("排查会话标题逻辑");

    await input.fill("修复多轮沟通后会话标题不刷新");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("修复多轮沟通后会话标题不刷新");
    await expectComposerReady(composer);
    await expect(page.locator(".runtime-session-title").first()).toContainText("排查会话标题逻辑");
  });

  test("keeps user bubbles right-aligned and within eighty percent width", async ({ page }) => {
    const { chatPage, composer } = await openChatWorkspace(page);
    const input = composer.input();

    await input.fill("请继续从产品视角介绍下，并说明 README 与 requirements 的关系");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("请继续从产品视角介绍下");

    await expect.poll(() => page.evaluate(() => {
      const bubbles = [...document.querySelectorAll(".msg.user .msg-bubble")];
      const bubble = bubbles[bubbles.length - 1] || null;
      const message = bubble ? bubble.closest(".msg.user") : null;
      const list = document.querySelector(".message-list");
      if (!(bubble instanceof HTMLElement) || !(message instanceof HTMLElement) || !(list instanceof HTMLElement)) {
        return false;
      }
      const bubbleRect = bubble.getBoundingClientRect();
      const messageRect = message.getBoundingClientRect();
      const listRect = list.getBoundingClientRect();
      const maxWidth = listRect.width * 0.8;
      const rightGap = Math.round(listRect.right - bubbleRect.right);
      const leftGap = Math.round(bubbleRect.left - listRect.left);
      return (
        bubbleRect.width <= maxWidth + 2 &&
        rightGap <= leftGap &&
        Math.round(listRect.right - messageRect.right) <= Math.round(messageRect.left - listRect.left)
      );
    })).toBe(true);
  });

  test("keeps streamed partial output visible when the stream is interrupted", async ({ page }) => {
    await page.addInitScript(() => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (!url.includes("/api/agent/messages/stream")) {
          return originalFetch(input, init);
        }
        const encoder = new TextEncoder();
        const body = new ReadableStream({
          start(controller) {
            controller.enqueue(encoder.encode(`event: start
data: {"message_id":"message-stream","session_id":"session-stream","channel_id":"web-default","trace_id":"trace-stream"}

`));
            controller.enqueue(encoder.encode(`event: delta
data: {"delta":"Partial answer from stream"}

`));
            controller.close();
          }
        });
        return new Response(body, {
          status: 200,
          headers: {
            "Content-Type": "text/event-stream"
          }
        });
      };
    });

    const { composer } = await openChatWorkspace(page);
    await composer.input().fill("继续输出");
    await composer.submitButton().click();

    const assistantMessage = page.locator(".msg.assistant").last();
    await expect(assistantMessage.locator(".msg-bubble")).toContainText("Partial answer from stream");
    await expect(assistantMessage.locator(".msg-bubble")).not.toContainText("Stream failed");
    await expect(assistantMessage.locator(".status-pill")).toContainText("Failed");
  });

  test("shows structured process steps before the agent stream is done", async ({ page }) => {
    await page.addInitScript(() => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (!url.includes("/api/agent/messages/stream")) {
          return originalFetch(input, init);
        }
        const encoder = new TextEncoder();
        const body = new ReadableStream({
          start(controller) {
            controller.enqueue(encoder.encode(`event: start
data: {"message_id":"message-process-live","session_id":"session-process-live","channel_id":"web-default","trace_id":"trace-process-live"}

`));
            controller.enqueue(encoder.encode(`event: process
data: {"process_step":{"id":"step-1","kind":"action","title":"codex_exec","status":"running"}}

`));
            window.setTimeout(() => {
              controller.enqueue(encoder.encode(`event: done
data: {"result":{"route":"nl","output":"任务已完成","process_steps":[{"id":"step-1","kind":"action","title":"codex_exec","detail":"检查仓库状态","status":"completed"}]}}

`));
              controller.close();
            }, 300);
          }
        });
        return new Response(body, {
          status: 200,
          headers: {
            "Content-Type": "text/event-stream"
          }
        });
      };
    });

    const { composer } = await openChatWorkspace(page);
    await composer.input().fill("帮我检查仓库状态");
    await composer.submitButton().click();

    const assistantMessage = page.locator(".msg.assistant").last();
    await expect(assistantMessage.locator(".agent-process-shell")).toBeVisible();
    await expect(assistantMessage.locator(".agent-process-step-title")).toContainText("codex_exec");
    await expect(assistantMessage.locator(".assistant-message-body")).toHaveCount(0);

    await expect(assistantMessage.locator(".assistant-message-body")).toContainText("任务已完成");
  });

  test("shows explicit load failures after a chat stream aborts", async ({ page }) => {
    const seededAt = Date.now();
    await page.addInitScript(() => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (url.includes("/api/messages/stream")) {
          const encoder = new TextEncoder();
          let sentStart = false;
          const body = new ReadableStream({
            pull(controller) {
              if (!sentStart) {
                sentStart = true;
                controller.enqueue(encoder.encode(`event: start
data: {"message_id":"message-stream-refresh","session_id":"session-refresh-recover","channel_id":"web-default","trace_id":"trace-refresh"}

`));
                return;
              }
              controller.error(new TypeError("Load failed"));
            }
          });
          return new Response(body, {
            status: 200,
            headers: {
              "Content-Type": "text/event-stream"
            }
          });
        }
        return originalFetch(input, init);
      };
    });

    const { composer } = await openChatWorkspace(page);
    await composer.input().fill("帮我确认仓库状态");
    await composer.submitButton().click();

    const failedMessage = page.locator(".msg.assistant").last();
    await expect(failedMessage.locator(".msg-bubble")).toContainText("Load failed");
    await expect(failedMessage.locator(".status-pill")).toContainText("Failed");
  });

  test("renders structured process steps from chat stream results", async ({ page }) => {
    const seededAt = Date.now();
    await page.addInitScript(() => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (url.includes("/api/messages/stream")) {
          const encoder = new TextEncoder();
          const body = new ReadableStream({
            start(controller) {
              controller.enqueue(encoder.encode(`event: start
data: {"message_id":"message-stream-steps","session_id":"session-agent-steps","channel_id":"web-default","trace_id":"trace-steps"}

`));
              controller.enqueue(encoder.encode(`event: delta
data: {"delta":"[agent] action: codex_exec\\n"}

`));
              controller.enqueue(encoder.encode(`event: done
data: {"result":{"route":"nl","output":"任务已完成","process_steps":[{"kind":"action","title":"codex_exec","detail":"检查仓库状态","status":"completed"}]}}

`));
              controller.close();
            }
          });
          return new Response(body, {
            status: 200,
            headers: {
              "Content-Type": "text/event-stream"
            }
          });
        }
        return originalFetch(input, init);
      };
    });

    const { composer } = await openChatWorkspace(page);
    await composer.input().fill("帮我检查仓库状态");
    await composer.submitButton().click();

    const streamedMessage = page.locator(".msg.assistant").last();
    await expect(streamedMessage.locator(".agent-process-shell")).toBeVisible();
    await streamedMessage.locator(".agent-process-body").evaluate((node) => {
      if (node instanceof HTMLElement) {
        node.hidden = false;
        node.removeAttribute("hidden");
      }
    });
    await expect(streamedMessage.locator(".agent-process-toggle")).toContainText("Process");
    await expect(streamedMessage.locator(".agent-process-step-body").first()).toContainText("检查仓库状态");
    await expect(streamedMessage.locator(".agent-process-answer-shell")).toContainText("任务已完成");
    await expect(streamedMessage.locator(".agent-process-answer-shell")).not.toContainText("[agent] action:");
  });

  test("keeps structured agent process detail readable on mobile", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.addInitScript(() => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (url.includes("/api/agents")) {
          return new Response(JSON.stringify({
            items: [
              {
                id: "coding",
                name: "Coding Agent",
                description: "Coding workflow",
                session_profile_fields: [],
              },
            ],
          }), {
            status: 200,
            headers: {
              "Content-Type": "application/json",
            },
          });
        }
        if (url.includes("/api/agent/messages/stream")) {
          const encoder = new TextEncoder();
          const body = new ReadableStream({
            start(controller) {
              controller.enqueue(encoder.encode(`event: start
data: {"message_id":"message-stream-steps-mobile","session_id":"session-agent-steps-mobile","channel_id":"web-default","trace_id":"trace-steps-mobile"}

`));
              controller.enqueue(encoder.encode(`event: delta
data: {"delta":"[agent] action: codex_exec\\n"}

`));
              controller.enqueue(encoder.encode(`event: done
data: {"result":{"route":"nl","output":"任务已完成","process_steps":[{"kind":"action","title":"codex_exec","detail":"需要把远端最新的 alter0 项目克隆到当前会话的单独工作区中，并检查工作区结构、远端分支和当前 HEAD 是否对齐。","status":"completed"}]}}

`));
              controller.close();
            }
          });
          return new Response(body, {
            status: 200,
            headers: {
              "Content-Type": "text/event-stream"
            }
          });
        }
        return originalFetch(input, init);
      };
    });

    await page.goto("/chat#agent-runtime");
    await loginIfNeeded(page);
    await waitForAppReady(page);
    const input = page.locator("[data-composer-input='conversation']");
    const submit = page.locator("[data-composer-submit='conversation']");
    await expect(input).toBeVisible();
    await expect(submit).toBeVisible();
    await input.fill("帮我检查仓库同步情况");
    await submit.click();
    const assistantMessage = page.locator(".msg.assistant").last();
    await expect(assistantMessage.locator(".agent-process-shell")).toBeVisible();
    await assistantMessage.locator(".agent-process-body").evaluate((node) => {
      if (node instanceof HTMLElement) {
        node.hidden = false;
        node.removeAttribute("hidden");
      }
    });

    const processBody = assistantMessage.locator(".agent-process-step-body").first();
    await expect(processBody).toContainText("需要把远端最新的 alter0 项目克隆到当前会话的单独工作区中");

    const metrics = await processBody.evaluate((node) => {
      const detail = node instanceof HTMLElement ? node : null;
      const rendered = detail?.querySelector(".runtime-markdown-rendered");
      const step = detail?.closest(".agent-process-step");
      const shell = detail?.closest(".agent-process-shell");
      if (!detail || !(rendered instanceof HTMLElement) || !(step instanceof HTMLElement) || !(shell instanceof HTMLElement)) {
        return null;
      }
      const detailRect = detail.getBoundingClientRect();
      const renderedRect = rendered.getBoundingClientRect();
      const stepRect = step.getBoundingClientRect();
      const shellRect = shell.getBoundingClientRect();
      return {
        detailWidth: detailRect.width,
        renderedWidth: renderedRect.width,
        stepWidth: stepRect.width,
        shellWidth: shellRect.width,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.detailWidth ?? 0).toBeGreaterThan(200);
    expect(metrics?.renderedWidth ?? 0).toBeGreaterThan(200);
    expect(metrics?.detailWidth ?? 0).toBeGreaterThan((metrics?.stepWidth ?? 0) * 0.7);
    expect(metrics?.renderedWidth ?? 0).toBeGreaterThan((metrics?.shellWidth ?? 0) * 0.65);
  });

  test("keeps sparse mobile chat messages packed with their timestamps", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.addInitScript(() => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (url.includes("/api/agents")) {
          return new Response(JSON.stringify({
            items: [
              {
                id: "coding",
                name: "Coding Agent",
                description: "Coding workflow",
                session_profile_fields: [],
              },
            ],
          }), {
            status: 200,
            headers: {
              "Content-Type": "application/json",
            },
          });
        }
        if (url.includes("/api/agent/messages/stream")) {
          const encoder = new TextEncoder();
          const body = new ReadableStream({
            start(controller) {
              controller.enqueue(encoder.encode(`event: start
data: {"message_id":"message-stream-sparse-mobile","session_id":"session-agent-sparse-mobile","channel_id":"web-default","trace_id":"trace-sparse-mobile"}

`));
              controller.enqueue(encoder.encode(`event: delta
data: {"delta":"[agent] action: codex_exec\\n"}

`));
              controller.enqueue(encoder.encode(`event: done
data: {"result":{"route":"nl","output":"Node 更偏应用层与生态速度，Go 更偏并发效率与部署稳定性。","process_steps":[{"kind":"action","title":"codex_exec","detail":"整理 Node 与 Go 在运行时模型、并发方式、构建发布和工程适配上的主要差异。","status":"completed"}]}}

`));
              controller.close();
            }
          });
          return new Response(body, {
            status: 200,
            headers: {
              "Content-Type": "text/event-stream"
            }
          });
        }
        return originalFetch(input, init);
      };
    });

    await page.goto("/chat#agent-runtime");
    await loginIfNeeded(page);
    await waitForAppReady(page);
    const input = page.locator("[data-composer-input='conversation']");
    const submit = page.locator("[data-composer-submit='conversation']");
    await expect(input).toBeVisible();
    await expect(submit).toBeVisible();
    await input.fill("详细介绍下 node 和 go 的差异");
    await submit.click();

    const assistantMessage = page.locator(".msg.assistant").last();
    await expect(assistantMessage.locator(".agent-process-shell")).toBeVisible();

    const metrics = await page.evaluate(() => {
      const timeline = document.querySelector(".runtime-timeline");
      const userMessages = Array.from(document.querySelectorAll(".msg.user"));
      const assistantMessages = Array.from(document.querySelectorAll(".msg.assistant"));
      const user = userMessages[userMessages.length - 1];
      const assistant = assistantMessages[assistantMessages.length - 1];
      const userBubble = user?.querySelector(".msg-bubble");
      const userMeta = user?.querySelector(".msg-meta");
      const assistantProcess = assistant?.querySelector(".agent-process-shell");
      const assistantAnswer = assistant?.querySelector(".agent-process-answer-shell");
      const assistantMeta = assistant?.querySelector(".msg-meta");
      if (
        !(timeline instanceof HTMLElement)
        || !(user instanceof HTMLElement)
        || !(assistant instanceof HTMLElement)
        || !(userBubble instanceof HTMLElement)
        || !(userMeta instanceof HTMLElement)
        || !(assistantProcess instanceof HTMLElement)
        || !(assistantAnswer instanceof HTMLElement)
        || !(assistantMeta instanceof HTMLElement)
      ) {
        return null;
      }
      const timelineStyle = getComputedStyle(timeline);
      const userBubbleRect = userBubble.getBoundingClientRect();
      const userMetaRect = userMeta.getBoundingClientRect();
      const assistantProcessRect = assistantProcess.getBoundingClientRect();
      const assistantAnswerRect = assistantAnswer.getBoundingClientRect();
      const assistantMetaRect = assistantMeta.getBoundingClientRect();
      const userRect = user.getBoundingClientRect();
      const assistantRect = assistant.getBoundingClientRect();
      return {
        alignContent: timelineStyle.alignContent,
        gridAutoRows: timelineStyle.gridAutoRows,
        userGap: userMetaRect.top - userBubbleRect.bottom,
        assistantGap: assistantMetaRect.top - assistantAnswerRect.bottom,
        userExtraHeight: userRect.height - userBubbleRect.height,
        assistantExtraHeight: assistantRect.height - assistantProcessRect.height - assistantAnswerRect.height,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.alignContent).toBe("start");
    expect(metrics?.gridAutoRows).toBe("max-content");
    expect(metrics?.userGap ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(12);
    expect(metrics?.assistantGap ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(12);
    expect(metrics?.userExtraHeight ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(40);
    expect(metrics?.assistantExtraHeight ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(56);
  });
});
