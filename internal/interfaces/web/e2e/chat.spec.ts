import { expect, test, type APIRequestContext } from "@playwright/test";
import {
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
import { commitIMEInput, pressEnterDuringIMEInput, startIMEInput } from "./helpers/interactions/ime";
import { authenticateWebRequest } from "./helpers/flows/auth";
import {
  openChatWorkspace,
  openChatWorkspaceWithDraft,
  openChatWorkspaceWithTwoDraftSessions,
  reloadChatWorkspace,
} from "./helpers/scenarios/chat";
import { installVisualViewportMock, setVisualViewport } from "./helpers/support/visual-viewport";

const CHAT_BROWSER_STORAGE_KEYS = [
  "alter0.web.session.active.v1",
  "alter0.web.session.snapshot.v1",
  "alter0.web.session.recent.v1",
  "alter0.web.session.long_term_snapshot.v1",
  "alter0.web.session.info_snapshot.v1",
  "alter0.web.composer.drafts.v1",
  "alter0.web.composer.attachments.v1",
  "alter0.web.runtime.event_filter.v1",
];

async function clearChatBrowserStorage(page: Parameters<typeof loginIfNeeded>[0]): Promise<void> {
  const clearFlag = `__alter0_e2e_chat_storage_cleared_${Date.now()}_${Math.random().toString(36).slice(2)}__`;
  await page.addInitScript(({ keys, flag }) => {
    const clearFlag = flag;
    if (window.sessionStorage.getItem(clearFlag) === "1") {
      return;
    }
    keys.forEach((key) => {
      window.localStorage.removeItem(key);
      window.sessionStorage.removeItem(key);
    });
    window.sessionStorage.setItem(clearFlag, "1");
  }, { keys: CHAT_BROWSER_STORAGE_KEYS, flag: clearFlag });
}

async function clearChatServerSessions(request: APIRequestContext): Promise<void> {
  await authenticateWebRequest(request);
  const deadline = Date.now() + 5000;
  let emptyReads = 0;
  while (Date.now() < deadline && emptyReads < 10) {
    const listResponse = await request.get("/api/chat/sessions");
    expect(listResponse.ok()).toBeTruthy();
    const payload = await listResponse.json();
    const items = Array.isArray(payload?.items) ? payload.items : [];
    if (items.length === 0) {
      emptyReads += 1;
      await new Promise((resolve) => setTimeout(resolve, 200));
      continue;
    }
    emptyReads = 0;
    for (const item of items) {
      const sessionID = typeof item?.id === "string" ? item.id : "";
      if (!sessionID) {
        continue;
      }
      const deleteResponse = await request.delete(`/api/chat/sessions/${encodeURIComponent(sessionID)}`);
      expect(deleteResponse.ok()).toBeTruthy();
    }
  }
  await expect.poll(async () => {
    const response = await request.get("/api/chat/sessions");
    if (!response.ok()) {
      return -1;
    }
    const nextPayload = await response.json();
    return Array.isArray(nextPayload?.items) ? nextPayload.items.length : 0;
  }).toBe(0);
}

async function expectChatDraftPersisted(page: Parameters<typeof loginIfNeeded>[0], value: string): Promise<void> {
  await expect.poll(async () => page.evaluate((expected) => {
    const raw = window.sessionStorage.getItem("alter0.web.composer.drafts.v1") || "{}";
    try {
      const parsed = JSON.parse(raw);
      return Object.values(parsed).includes(expected);
    } catch {
      return false;
    }
  }, value)).toBe(true);
}

function workbenchRouteView(page: Parameters<typeof loginIfNeeded>[0], route: string) {
  return page.locator(`.route-view[data-route='${route}']`).first();
}

function workbenchRouteBody(page: Parameters<typeof loginIfNeeded>[0], route: string) {
  return page.locator(`.route-body[data-route='${route}']`).first();
}

function mobileMenuButton(page: Parameters<typeof loginIfNeeded>[0]) {
  return page.getByRole("button", { name: "Menu" }).first();
}

function mobileSessionButton(page: Parameters<typeof loginIfNeeded>[0]) {
  return page.getByRole("button", { name: "Session" }).last();
}

function runtimeSettingsToggle(page: Parameters<typeof loginIfNeeded>[0]) {
  return page.locator("[data-runtime-composer-utility='session']").first();
}

function runtimeSettingsPanel(page: Parameters<typeof loginIfNeeded>[0]) {
  return page.locator("[data-runtime-config-surface='conversation']").first();
}

function latestAssistantMessage(page: Parameters<typeof loginIfNeeded>[0]) {
  return page.locator(".runtime-message-assistant[data-message-id]").last();
}

async function openSettingsRoute(page: Parameters<typeof loginIfNeeded>[0]): Promise<void> {
  await page.goto("/settings");
  await loginIfNeeded(page);
  await waitForAppReady(page);
  await expect(workbenchRouteView(page, "settings")).toBeVisible();
}

async function mockRuntimeSession(
  page: Parameters<typeof loginIfNeeded>[0],
  options: {
    route: "chat" | "chat";
    session: Record<string, unknown>;
  },
): Promise<void> {
  const sessionID = String(options.session.id || "").trim();
  if (!sessionID) {
    throw new Error("mockRuntimeSession requires a session id");
  }
  await page.context().route("**/api/chat/sessions**", async (route) => {
    const url = new URL(route.request().url());
    void options;
    if (url.pathname.endsWith("/api/chat/sessions")) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          items: [options.session],
        }),
      });
      return;
    }
    if (url.pathname.endsWith(`/api/chat/sessions/${sessionID}`)) {
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
}

async function mockControlSkills(page: Parameters<typeof loginIfNeeded>[0]): Promise<void> {
  await page.route("**/api/control/skills", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        items: [
          { id: "memory", name: "Memory", description: "Use workspace memory", enabled: true },
          {
            id: "implementation",
            name: "Implementation Skill",
            description: "Implementation guidance skill",
            enabled: true,
          },
        ],
      }),
    });
  });
}

function compactChatSessionID(value: string): string {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  const token = (hash >>> 0).toString(16).padStart(8, "0").slice(0, 8);
  return `c_${token}${token}`;
}

async function mockChatRuntimeSessions(
  page: Parameters<typeof loginIfNeeded>[0],
  sessions: Array<Record<string, unknown>>,
): Promise<void> {
  await page.context().route("**/api/chat/sessions**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname.endsWith("/api/chat/sessions")) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: sessions }),
      });
      return;
    }
    const session = sessions.find((item) => url.pathname.endsWith(
      `/api/chat/sessions/${encodeURIComponent(String(item.id || ""))}`,
    ));
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
}

async function mockEmptyChatRuntimeSessions(page: Parameters<typeof loginIfNeeded>[0]): Promise<void> {
  await page.addInitScript(() => {
    window.localStorage.clear();
    window.sessionStorage.clear();
  });
  await page.context().route("**/api/chat/sessions**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname.endsWith("/api/chat/sessions")) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [] }),
      });
      return;
    }
    await route.fallback();
  });
}

test.describe("Chat composer", () => {
  test.beforeEach(async ({ page, request }) => {
    await clearChatServerSessions(request);
    await clearChatBrowserStorage(page);
  });

  test.afterEach(async ({ page, request }) => {
    await page.close();
    await clearChatServerSessions(request);
  });

  for (const scenario of [
    { name: "desktop", width: 1280, height: 900, mobile: false },
    { name: "mobile", width: 390, height: 844, mobile: true },
  ]) {
    test(`opens the latest Chat session after stale query re-entry on ${scenario.name}`, async ({ page }) => {
      const latestSession = {
        id: compactChatSessionID("latest-chat-session"),
        title: "Latest chat",
        title_auto: false,
        title_score: 10,
        created_at: "2026-06-11T05:40:00Z",
        target_type: "model",
        target_id: "raw-model",
        target_name: "Raw Model",
        messages: [],
      };
      const olderSession = {
        id: compactChatSessionID("older-chat-session"),
        title: "Older chat",
        title_auto: false,
        title_score: 8,
        created_at: "2026-06-10T05:40:00Z",
        target_type: "model",
        target_id: "raw-model",
        target_name: "Raw Model",
        messages: [],
      };

      await page.setViewportSize({ width: scenario.width, height: scenario.height });
      await mockChatRuntimeSessions(page, [latestSession, olderSession]);
      await page.addInitScript((sessionID) => {
        window.sessionStorage.setItem("alter0.web.session.active.v1", JSON.stringify({ chat: sessionID }));
      }, olderSession.id);

      await page.goto("/chat");
      await ensureChatRouteReady(page);
      await page.goto(`/chat?session_id=${olderSession.id}`);
      await ensureChatRouteReady(page);
      const navRail = page.locator('[data-nav-session-rail="chat"]');
      if (scenario.mobile) {
        await page.getByRole("button", { name: "Menu" }).click();
      }
      await expect(navRail.locator(`[data-runtime-session-card="${olderSession.id}"]`)).toHaveClass(/is-active/);

      await page.locator('button[data-route="chat"]').click();

      await expect(page).toHaveURL(/\/chat$/);
      if (scenario.mobile) {
        await page.getByRole("button", { name: "Menu" }).click();
      }
      await expect(navRail.locator(`[data-runtime-session-card="${latestSession.id}"]`)).toHaveClass(/is-active/);
      await expect(page.locator(".runtime-workspace-head h4")).toContainText(String(latestSession.title));
    });
  }

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

  test("keeps chat session short hashes in the details panel without crowding the list", async ({ page }) => {
    const sessionID = "db4416b7-452d-44a6-83ca-999e77f47791";
    const createdAt = Date.now();
    await mockRuntimeSession(page, {
      route: "chat",
      session: {
        id: sessionID,
        title: "修复 Skill 会话标识",
        title_auto: false,
        title_score: 8,
        created_at: new Date(createdAt).toISOString(),
        target_type: "skill",
        target_id: "implementation",
        target_name: "Implementation Skill",
        model_provider_id: "",
        model_id: "",
        tool_ids: [],
        mcp_ids: [],
        messages: [{
          id: "message-skill-session-hash",
          role: "user",
          text: "给 Skill 会话加标识",
          at: new Date(createdAt).toISOString(),
          status: "done",
        }],
      },
    });
    await page.addInitScript(() => {
      window.sessionStorage.setItem("alter0.web.session.active.v1", JSON.stringify({
        "chat": "db4416b7-452d-44a6-83ca-999e77f47791",
      }));
    });

    await page.goto("/chat");
    await ensureChatRouteReady(page);
    if (!new URL(page.url()).pathname.endsWith("/chat")) {
      await page.goto("/chat");
      await ensureChatRouteReady(page);
    }

    const detailsButton = page.getByRole("button", { name: "Details" }).first();
    expect(sessionID).toMatch(/^c_[a-z0-9]{16}$/);
    await expect(page.locator(".runtime-session-hash")).toHaveCount(0);
    await detailsButton.click();

    await expect(page.locator('[data-runtime-details-panel="conversation"]')).not.toContainText("Short hash");
    await expect(page.locator('[data-runtime-details-panel="conversation"]')).not.toContainText(sessionID.slice(2, 10));
  });

  test("keeps empty session hint near the session header", async ({ page }) => {
    await mockEmptyChatRuntimeSessions(page);
    await openChatWorkspace(page);

    const sessionPane = page.locator('[data-runtime-session-pane="conversation"]');
    const heading = page.getByText("Sessions").first();
    const emptyHint = page.getByText("No sessions yet. Click New to start.").first();
    const sessionCards = page.locator('[data-runtime-session-card]');

    await expect(sessionCards).toHaveCount(0);
    await expect(emptyHint).toBeVisible();

    const headingBox = await heading.boundingBox();
    const emptyHintBox = await emptyHint.boundingBox();

    expect(headingBox).not.toBeNull();
    expect(emptyHintBox).not.toBeNull();
    expect((emptyHintBox?.y ?? 0) - (headingBox?.y ?? 0)).toBeLessThan(120);
  });

  test("keeps the shell stable while resizing across the desktop and drawer breakpoints", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await openChatWorkspace(page);

    const readShellMetrics = async () =>
      page.evaluate(() => {
        const appShell = document.querySelector(".app-shell");
        const nav = document.querySelector("nav[aria-label='Primary workspace navigation']");
        const sessionPane = document.querySelector("[data-runtime-session-pane='conversation']");
        const chatPane = document.querySelector(".chat-pane");
        const composerShell = document.querySelector("[data-composer-form='conversation']");
        const mobileMenu = Array.from(document.querySelectorAll("button")).find((button) =>
          (button.textContent || "").trim() === "Menu",
        );
        const mobileNew = Array.from(document.querySelectorAll("button")).find((button) =>
          (button.textContent || "").trim() === "New",
        );
        if (
          !(appShell instanceof HTMLElement || document.querySelector("main") instanceof HTMLElement)
          || !(sessionPane instanceof HTMLElement)
          || !(chatPane instanceof HTMLElement)
          || !(composerShell instanceof HTMLElement)
        ) {
          return null;
        }

        const navRect = nav instanceof HTMLElement ? nav.getBoundingClientRect() : null;
        const sessionRect = sessionPane.getBoundingClientRect();
        const chatRect = chatPane.getBoundingClientRect();
        const composerRect = composerShell.getBoundingClientRect();
        const mobileMenuRect = mobileMenu instanceof HTMLElement ? mobileMenu.getBoundingClientRect() : null;
        const mobileNewRect = mobileNew instanceof HTMLElement ? mobileNew.getBoundingClientRect() : null;
        const shellStyle = appShell instanceof HTMLElement ? getComputedStyle(appShell) : null;
        const navStyle = nav instanceof HTMLElement ? getComputedStyle(nav) : null;
        const sessionStyle = getComputedStyle(sessionPane);
        const doc = document.documentElement;

        return {
          navPosition: navStyle?.position || "",
          sessionPosition: sessionStyle.position,
          navVisible: nav instanceof HTMLElement && navRect !== null && navRect.width > 0 && navStyle?.display !== "none",
          mobileMenuVisible: mobileMenuRect !== null && mobileMenuRect.width > 0,
          mobileNewVisible: mobileNewRect !== null && mobileNewRect.width > 0,
          navLeft: navRect?.left ?? 0,
          navRight: navRect?.right ?? 0,
          sessionLeft: sessionRect.left,
          sessionRight: sessionRect.right,
          chatLeft: chatRect.left,
          chatRight: chatRect.right,
          composerBottom: composerRect.bottom,
          mobileMenuTop: mobileMenuRect?.top ?? 0,
          mobileNewTop: mobileNewRect?.top ?? 0,
          viewportWidth: window.innerWidth,
          viewportHeight: window.innerHeight,
          scrollWidth: doc.scrollWidth,
          gridTemplateColumns: shellStyle?.gridTemplateColumns || "",
        };
      });

    const desktop = await readShellMetrics();
    expect(desktop).not.toBeNull();
    expect(desktop?.navPosition).not.toBe("fixed");
    expect(desktop?.sessionPosition).not.toBe("fixed");
    expect(desktop?.navVisible).toBe(true);
    expect(desktop?.mobileMenuVisible).toBe(false);
    expect(desktop?.sessionRight ?? 0).toBeLessThanOrEqual((desktop?.chatLeft ?? 0) + 4);
    expect(desktop?.scrollWidth ?? 0).toBeLessThanOrEqual((desktop?.viewportWidth ?? 0) + 1);
    expect(desktop?.gridTemplateColumns.split(" ").length ?? 0).toBeGreaterThanOrEqual(2);

    await page.setViewportSize({ width: 760, height: 900 });

    await expect.poll(async () => {
      const metrics = await readShellMetrics();
      return metrics?.mobileMenuVisible ?? false;
    }).toBe(true);

    const drawer = await readShellMetrics();
    expect(drawer).not.toBeNull();
    expect(drawer?.navVisible).toBe(false);
    expect(drawer?.chatLeft ?? 0).toBeLessThanOrEqual(24);
    expect((drawer?.viewportHeight ?? 0) - (drawer?.composerBottom ?? 0)).toBeLessThan(36);
    expect(drawer?.scrollWidth ?? 0).toBeLessThanOrEqual((drawer?.viewportWidth ?? 0) + 1);

    await page.setViewportSize({ width: 1180, height: 900 });

    await expect.poll(async () => {
      const metrics = await readShellMetrics();
      return metrics?.navVisible ?? false;
    }).toBe(true);

    const restoredDesktop = await readShellMetrics();
    expect(restoredDesktop).not.toBeNull();
    expect(restoredDesktop?.mobileMenuVisible).toBe(false);
    expect(restoredDesktop?.sessionRight ?? 0).toBeLessThanOrEqual((restoredDesktop?.chatLeft ?? 0) + 4);
    expect(restoredDesktop?.scrollWidth ?? 0).toBeLessThanOrEqual((restoredDesktop?.viewportWidth ?? 0) + 1);
  });

  test("keeps mobile and desktop shell states from mixing across breakpoints", async ({ page }) => {
    await openChatWorkspace(page);

    const readBreakpointState = async () =>
      page.evaluate(() => {
        const appShell = document.querySelector(".app-shell");
        const nav = document.querySelector("nav[aria-label='Primary workspace navigation']");
        const sessionPane = document.querySelector("[data-runtime-session-pane='conversation']");
        const chatPane = document.querySelector(".chat-pane");
        const mobileMenu = Array.from(document.querySelectorAll("button")).find((button) =>
          (button.textContent || "").trim() === "Menu",
        );
        if (
          !(appShell instanceof HTMLElement || document.querySelector("main") instanceof HTMLElement)
          || !(sessionPane instanceof HTMLElement)
          || !(chatPane instanceof HTMLElement)
        ) {
          return null;
        }

        return {
          viewportWidth: window.innerWidth,
          scrollWidth: document.documentElement.scrollWidth,
          shellClassName: appShell instanceof HTMLElement ? appShell.className : "",
          gridTemplateColumns: appShell instanceof HTMLElement ? getComputedStyle(appShell).gridTemplateColumns : "",
          navVisible: nav instanceof HTMLElement && getComputedStyle(nav).display !== "none" && nav.getBoundingClientRect().width > 0,
          mobileMenuVisible: mobileMenu instanceof HTMLElement && mobileMenu.getBoundingClientRect().width > 0,
          sessionPosition: getComputedStyle(sessionPane).position,
          chatLeft: chatPane.getBoundingClientRect().left,
        };
      });

    await page.setViewportSize({ width: 760, height: 900 });
    await expect.poll(async () => (await readBreakpointState())?.mobileMenuVisible ?? false).toBe(true);

    const mobileEdge = await readBreakpointState();
    expect(mobileEdge).not.toBeNull();
    expect(mobileEdge?.navVisible).toBe(false);
    expect(mobileEdge?.chatLeft ?? 0).toBeLessThanOrEqual(24);
    expect(mobileEdge?.scrollWidth ?? 0).toBeLessThanOrEqual((mobileEdge?.viewportWidth ?? 0) + 1);

    await page.setViewportSize({ width: 1180, height: 900 });
    await expect.poll(async () => (await readBreakpointState())?.navVisible ?? false).toBe(true);

    const desktopEdge = await readBreakpointState();
    expect(desktopEdge).not.toBeNull();
    expect(desktopEdge?.gridTemplateColumns.split(" ").length ?? 0).toBeGreaterThanOrEqual(2);
    expect(desktopEdge?.scrollWidth ?? 0).toBeLessThanOrEqual((desktopEdge?.viewportWidth ?? 0) + 1);
  });

  test("keeps empty chat controls compact on narrow screens", async ({ page }) => {
    await page.setViewportSize({ width: 760, height: 980 });
    await mockEmptyChatRuntimeSessions(page);
    await openChatWorkspace(page);

    const mobileHeader = page.locator("[data-route-mobile-head], .runtime-workspace-mobile-header").first();
    const navToggle = page.getByRole("button", { name: "Menu" }).first();
    const newChatButton = page.getByRole("button", { name: "New" }).first();
    const emptyHeading = page.getByText("Start a new workspace flow").last();
    const emptyDescription = page.getByText("Conversation, process, and delivery stay in a single timeline.").last();
    const composerShell = page.locator('[data-composer-form="conversation"]');
    const sessionButton = page.getByRole("button", { name: "Session" }).last();
    const sendButton = page.locator('[data-composer-submit="conversation"]');

    await expect(navToggle).toBeVisible();
    await expect(newChatButton).toBeVisible();
    await expect(mobileHeader).toBeVisible();
    await expect(emptyHeading).toBeVisible();
    await expect(sessionButton).toBeVisible();

    const headerBox = await mobileHeader.boundingBox();
    const navBox = await navToggle.boundingBox();
    const newChatBox = await newChatButton.boundingBox();
    const emptyHeadingBox = await emptyHeading.boundingBox();
    const emptyDescriptionBox = await emptyDescription.boundingBox();
    const composerBox = await composerShell.boundingBox();
    const runtimeBox = await sessionButton.boundingBox();
    const sendBox = await sendButton.boundingBox();
    const viewport = page.viewportSize();

    expect(headerBox).not.toBeNull();
    expect(navBox).not.toBeNull();
    expect(newChatBox).not.toBeNull();
    expect(emptyHeadingBox).not.toBeNull();
    expect(emptyDescriptionBox).not.toBeNull();
    expect(composerBox).not.toBeNull();
    expect(runtimeBox).not.toBeNull();
    expect(sendBox).not.toBeNull();
    expect(viewport).not.toBeNull();

    expect(Math.abs((navBox?.y ?? 0) - (newChatBox?.y ?? 0))).toBeLessThan(8);
    expect((emptyHeadingBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeGreaterThanOrEqual(16);
    expect((emptyDescriptionBox?.y ?? 0) - (emptyHeadingBox?.y ?? 0)).toBeGreaterThan(10);
    expect((composerBox?.y ?? 0) - ((emptyDescriptionBox?.y ?? 0) + (emptyDescriptionBox?.height ?? 0))).toBeGreaterThanOrEqual(48);
    expect(((viewport?.height ?? 0) - ((composerBox?.y ?? 0) + (composerBox?.height ?? 0)))).toBeLessThan(36);
    expect(Math.abs((runtimeBox?.y ?? 0) - (sendBox?.y ?? 0))).toBeLessThan(6);
    expect(sendBox?.x ?? 0).toBeGreaterThan((runtimeBox?.x ?? 0) + (runtimeBox?.width ?? 0) - 4);

    await sessionButton.click();
    await expect(page.locator(".runtime-composer-config-panel").first()).toBeVisible();
  });

  test("keeps empty chat controls compact on wide screens", async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 980 });
    await openChatWorkspace(page);

    const boxes = await page.evaluate(() => {
      const workspace = document.querySelector("[data-runtime-workspace='conversation']");
      const header = document.querySelector(".runtime-workspace-head");
      const composer = document.querySelector("[data-composer-form='conversation']");
      if (
        !(workspace instanceof HTMLElement)
        || !(header instanceof HTMLElement)
        || !(composer instanceof HTMLElement)
      ) {
        return null;
      }
      const rect = (node: HTMLElement) => {
        const box = node.getBoundingClientRect();
        return { x: box.x, y: box.y, width: box.width, height: box.height };
      };
      return {
        workspace: rect(workspace),
        header: rect(header),
        composer: rect(composer),
      };
    });

    expect(boxes).not.toBeNull();
    const workspaceBox = boxes?.workspace;
    const headerBox = boxes?.header;
    const composerBox = boxes?.composer;

    expect((headerBox?.y ?? 0) - (workspaceBox?.y ?? 0)).toBeLessThanOrEqual(8);
    expect((composerBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeGreaterThan(120);
    expect((composerBox?.x ?? 0)).toBeGreaterThanOrEqual((workspaceBox?.x ?? 0) - 1);
    expect((composerBox?.x ?? 0) + (composerBox?.width ?? 0)).toBeLessThanOrEqual((workspaceBox?.x ?? 0) + (workspaceBox?.width ?? 0) + 1);
  });

  test("keeps the empty chat header separated from welcome content on medium screens", async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 900 });
    await openChatWorkspace(page);

    const boxes = await page.evaluate(() => {
      const header = document.querySelector(".runtime-workspace-head");
      const workspace = document.querySelector("[data-runtime-workspace='conversation']");
      const composer = document.querySelector("[data-composer-form='conversation']");
      if (
        !(header instanceof HTMLElement)
        || !(workspace instanceof HTMLElement)
        || !(composer instanceof HTMLElement)
      ) {
        return null;
      }
      const rect = (node: HTMLElement) => {
        const box = node.getBoundingClientRect();
        return { x: box.x, y: box.y, width: box.width, height: box.height };
      };
      return {
        header: rect(header),
        workspace: rect(workspace),
        composer: rect(composer),
      };
    });
    const headerBox = boxes?.header;
    const workspaceBox = boxes?.workspace;
    const composerBox = boxes?.composer;
    const viewport = page.viewportSize();

    expect(boxes).not.toBeNull();
    expect(headerBox).not.toBeNull();
    expect(workspaceBox).not.toBeNull();
    expect(composerBox).not.toBeNull();
    expect(viewport).not.toBeNull();

    expect((composerBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeGreaterThan(160);
    expect((composerBox?.x ?? 0)).toBeGreaterThanOrEqual((workspaceBox?.x ?? 0) - 1);
    expect((composerBox?.x ?? 0) + (composerBox?.width ?? 0)).toBeLessThanOrEqual((workspaceBox?.x ?? 0) + (workspaceBox?.width ?? 0) + 1);
    expect(((viewport?.height ?? 0) - ((composerBox?.y ?? 0) + (composerBox?.height ?? 0)))).toBeLessThan(40);
  });

  test("keeps the empty chat viewport pinned to the top when loading the chat route", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1024, height: 900 });
    await page.goto("/chat");
    await ensureChatRouteReady(page);

    const header = page.locator(".runtime-workspace-head").first();
    const composerShell = page.locator('[data-composer-form="conversation"]');

    await expect(header).toBeVisible();
    await expect(composerShell).toBeVisible();

    const metrics = await page.evaluate(() => {
      const chatPaneNode = document.querySelector(".chat-pane");
      const headerNode = document.querySelector(".runtime-workspace-head");
      const composerNode = document.querySelector("[data-composer-form='conversation']");
      if (
        !(chatPaneNode instanceof HTMLElement)
        || !(headerNode instanceof HTMLElement)
        || !(composerNode instanceof HTMLElement)
      ) {
        return null;
      }
      const chatPaneRect = chatPaneNode.getBoundingClientRect();
      const headerRect = headerNode.getBoundingClientRect();
      const composerRect = composerNode.getBoundingClientRect();
      return {
        scrollY: window.scrollY,
        chatPaneTop: chatPaneRect.top,
        headerTop: headerRect.top,
        headerBottom: headerRect.top + headerRect.height,
        contentCenter: ((headerRect.top + headerRect.height) + composerRect.top) / 2,
        composerTop: composerRect.top,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.scrollY ?? 0).toBeLessThanOrEqual(4);
    expect(metrics?.chatPaneTop ?? 999).toBeLessThanOrEqual(8);
    expect(metrics?.headerTop ?? 999).toBeLessThanOrEqual(24);
    expect((metrics?.composerTop ?? 0) - (metrics?.headerBottom ?? 0)).toBeGreaterThan(160);
  });

  test("keeps shared page routes compact on narrow screens", async ({ page }) => {
    await page.setViewportSize({ width: 760, height: 980 });
    await openSettingsRoute(page);

    const header = page.locator("[data-route-mobile-head]").first();
    const navToggle = mobileMenuButton(page);
    const sessionToggle = mobileSessionButton(page);
    const newChatButton = page.getByRole("button", { name: "New" }).first();
    const routeTitle = workbenchRouteView(page, "settings").locator(".route-mobile-title h3").first();

    await expect(navToggle).toBeVisible();
    await expect(sessionToggle).toHaveCount(0);
    await expect(newChatButton).toHaveCount(0);
    await expect(routeTitle).not.toHaveText("");

    const headerBox = await header.boundingBox();
    const navBox = await navToggle.boundingBox();
    const routeTitleBox = await routeTitle.boundingBox();

    expect(headerBox).not.toBeNull();
    expect(navBox).not.toBeNull();
    expect(routeTitleBox).not.toBeNull();
    expect((routeTitleBox?.y ?? 0) - ((headerBox?.y ?? 0) + (headerBox?.height ?? 0))).toBeLessThan(64);
    expect(Math.abs((routeTitleBox?.y ?? 0) - (navBox?.y ?? 0))).toBeLessThanOrEqual(16);
  });

  test("expands the desktop chat reading column on wide screens", async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    const { chatPage, composer } = await openChatWorkspace(page);

    await composer.input().fill("请输出一段稍长的说明，用于验证桌面宽屏聊天布局。");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("请输出一段稍长的说明");
    await expect(page.locator(".runtime-session-title").first()).toContainText("请输出一段稍长的说明");

    const metrics = await page.evaluate(() => {
      const composerFrame = document.querySelector("[data-composer-form='conversation']");
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
    expect(metrics?.composerWidth ?? 0).toBeGreaterThanOrEqual(860);
    expect(metrics?.composerLeft ?? 0).toBeGreaterThan(0);
    expect(metrics?.composerRight ?? 0).toBeLessThanOrEqual((metrics?.viewportWidth ?? 0) + 1);
    expect(metrics?.scrollWidth ?? 0).toBeLessThanOrEqual((metrics?.viewportWidth ?? 0) + 1);
  });

  test("keeps the empty-state welcome copy and composer aligned on wide screens", async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await openChatWorkspace(page);

    const metrics = await page.evaluate(() => {
      const workspaceFrame = document.querySelector("[data-runtime-workspace='conversation']");
      const composerFrame = document.querySelector("[data-composer-form='conversation']");
      if (!(workspaceFrame instanceof HTMLElement) || !(composerFrame instanceof HTMLElement)) {
        return null;
      }
      const workspaceRect = workspaceFrame.getBoundingClientRect();
      const composerRect = composerFrame.getBoundingClientRect();
      return {
        workspaceWidth: workspaceRect.width,
        composerWidth: composerRect.width,
        workspaceCenter: workspaceRect.left + workspaceRect.width / 2,
        composerCenter: composerRect.left + composerRect.width / 2,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.workspaceWidth ?? 0).toBeGreaterThanOrEqual(900);
    expect(metrics?.composerWidth ?? 0).toBeGreaterThanOrEqual(860);
    expect(Math.abs((metrics?.workspaceCenter ?? 0) - (metrics?.composerCenter ?? 0))).toBeLessThan(220);
  });

  test("shows chat jump controls and jumps between visible messages", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    const turns = Array.from({ length: 18 }, (_value, index) => ({
      id: `turn-jump-${index + 1}`,
      prompt: `Jump prompt ${index + 1}`,
      status: "success",
      started_at: `2026-04-23T03:${String(index).padStart(2, "0")}:00Z`,
      finished_at: `2026-04-23T03:${String(index).padStart(2, "0")}:01Z`,
      final_output: (`Jump answer ${index + 1} `.repeat(240)).trim(),
    }));
    await mockRuntimeSession(page, {
      route: "chat",
      session: {
        id: compactChatSessionID("chat-jump-controls"),
        title: "Jump controls",
        status: "ready",
        created_at: "2026-04-23T03:00:00Z",
        turns,
        turns_paging: { has_more_before: false },
      },
    });
    await page.goto(`/chat?session_id=${compactChatSessionID("chat-jump-controls")}`);
    await loginIfNeeded(page);
    await waitForAppReady(page);
    await expect(page.locator(".runtime-message-user[data-message-id]").first()).toBeVisible();

    const routeView = page.locator("[data-runtime-screen='conversation']").first();
    const jumpTopButton = page.locator("[data-scroll-jump-top='chat']");
    const jumpPrevButton = page.locator("[data-scroll-jump-prev='chat']");
    const jumpNextButton = page.locator("[data-scroll-jump-next='chat']");
    const jumpBottomButton = page.locator("[data-scroll-jump-bottom='chat']");

    await expect.poll(async () => routeView.evaluate((node) => node.scrollHeight > node.clientHeight + 180)).toBe(true);

    const expectAdjacentJumpControls = async () => {
      const anchorGap = await routeView.evaluate((node) => {
        const containerRect = node.getBoundingClientRect();
        const anchors = Array.from(node.querySelectorAll<HTMLElement>(".runtime-message-user[data-message-id]"))
          .map((anchor) => {
            const rect = anchor.getBoundingClientRect();
            return {
              id: anchor.getAttribute("data-message-id") || "",
              top: node.scrollTop + rect.top - containerRect.top,
              bottom: node.scrollTop + rect.bottom - containerRect.top,
            };
          });
        for (let index = 0; index < anchors.length - 1; index += 1) {
          const previous = anchors[index];
          const next = anchors[index + 1];
          if (!previous || !next || next.top - previous.bottom <= node.clientHeight + 40) {
            continue;
          }
          node.scrollTop = previous.bottom + Math.floor((next.top - previous.bottom - node.clientHeight) / 2);
          node.dispatchEvent(new Event("scroll"));
          return { previousID: previous.id, nextID: next.id };
        }
        return null;
      });

      expect(anchorGap).not.toBeNull();
      await expect(jumpPrevButton).toHaveAttribute("data-scroll-jump-target", anchorGap?.previousID || "");
      await expect(jumpNextButton).toHaveAttribute("data-scroll-jump-target", anchorGap?.nextID || "");
      await expect(jumpPrevButton).toHaveClass(/is-visible/);
      await expect(jumpNextButton).toHaveClass(/is-visible/);
    };

    await expectAdjacentJumpControls();
    await page.setViewportSize({ width: 1280, height: 800 });
    await expectAdjacentJumpControls();
    await page.setViewportSize({ width: 760, height: 720 });

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
      const target = node.querySelector(`[data-message-id="${String(targetID || "")}"]`);
      if (!(target instanceof HTMLElement)) {
        return null;
      }
      return Math.round(target.getBoundingClientRect().top - node.getBoundingClientRect().top);
    }, jumpNavigationTarget)).toBeLessThanOrEqual(24);

    await jumpBottomButton.click();
    await expect.poll(async () => routeView.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop)).toBeLessThan(12);

    await expect(jumpTopButton).toHaveClass(/is-visible/);
  });

  test("renders mobile session settings as an independent bottom sheet", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    await openChatWorkspace(page);

    const runtimeToggle = runtimeSettingsToggle(page);
    const composerShell = page.locator(".runtime-composer-shell").first();
    const panel = runtimeSettingsPanel(page);
    const closeButton = panel.locator(".runtime-composer-panel-close");

    await runtimeToggle.click();

    await expect(panel).toBeVisible();
    await expect(closeButton).toBeVisible();

    const composerBox = await composerShell.boundingBox();
    const panelBox = await panel.boundingBox();
    const panelPosition = await panel.evaluate((node) => getComputedStyle(node).position);
    const bottomLayerHit = await page.evaluate(() => {
      const panelNode = document.querySelector("[data-runtime-config-surface='conversation']");
      if (!(panelNode instanceof HTMLElement)) {
        return false;
      }
      const rect = panelNode.getBoundingClientRect();
      const sampleX = Math.min(rect.right - 18, Math.max(rect.left + 18, rect.left + rect.width / 2));
      const sampleY = Math.max(rect.top + 18, rect.bottom - 18);
      const hit = document.elementFromPoint(sampleX, sampleY);
      return panelNode.contains(hit);
    });

    expect(composerBox).not.toBeNull();
    expect(panelBox).not.toBeNull();
    expect(panelBox?.y ?? 0).toBeGreaterThan(0);
    expect((panelBox?.y ?? 0) + (panelBox?.height ?? 0)).toBeLessThanOrEqual(page.viewportSize()?.height || 852);
    expect(panelPosition).toBe("fixed");
    expect(bottomLayerHit).toBe(true);

    await closeButton.click();
    await expect(panel).toHaveCount(0);
  });

  test("keeps 393px mobile drawers contained and dismissible without horizontal overflow", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    await openChatWorkspace(page);

    const appShell = page.locator(".app-shell");
    const navToggle = mobileMenuButton(page);
    const primaryNav = page.locator(".primary-nav");
    const backdrop = page.locator(".mobile-backdrop").first();

    const readMetrics = async () =>
      page.evaluate(() => {
        const root = document.documentElement;
        const shell = document.querySelector(".app-shell");
        const nav = document.querySelector(".primary-nav");
        const session = document.querySelector("[data-runtime-session-card]");
        if (
          !(shell instanceof HTMLElement)
          || !(nav instanceof HTMLElement)
        ) {
          return null;
        }
        const navRect = nav.getBoundingClientRect();
        const sessionRect = session instanceof HTMLElement
          ? session.getBoundingClientRect()
          : new DOMRect(0, 0, 0, 0);
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

    await navToggle.click();
    await expect(appShell).toHaveClass(/nav-open/);
    await expect(primaryNav).toBeVisible();
    await backdrop.dispatchEvent("click");
    await expect(appShell).not.toHaveClass(/nav-open/);
    await expect(appShell).not.toHaveClass(/overlay-open/);
  });

  test("keeps skill option copy concise inside session settings", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    await mockControlSkills(page);
    await openChatWorkspace(page);
    await page.goto("/chat");

    const runtimeToggle = runtimeSettingsToggle(page);
    await expect(runtimeToggle).toBeVisible();
    await runtimeToggle.click();

    const panel = runtimeSettingsPanel(page);
    await expect(panel).toBeVisible();
    await panel.getByRole("tab", { name: "Skills" }).click();
    const codingOption = panel.locator(".conversation-check-item").filter({ hasText: "Implementation Skill" }).first();
    await expect(codingOption).toBeVisible();
    await expect(codingOption).toContainText("Implementation Skill");
    await expect(codingOption).toContainText("Implementation guidance skill");
    await expect(codingOption).not.toContainText("Act as alter0's dedicated implementation user proxy");
  });

  test("keeps session settings scroll position while toggling skills", async ({ page }) => {
    await page.setViewportSize({ width: 393, height: 852 });
    await mockControlSkills(page);
    await openChatWorkspace(page);
    await page.goto("/chat");

    const runtimeToggle = runtimeSettingsToggle(page);
    await expect(runtimeToggle).toBeVisible();
    await runtimeToggle.click();

    const panel = runtimeSettingsPanel(page);
    await expect(panel).toBeVisible();
    await panel.getByRole("tab", { name: "Skills" }).click();
    const body = panel.locator(".conversation-inspector-sections").first();
    await expect(body).toBeVisible();

    const before = await body.evaluate((node) => {
      node.scrollTop = Math.max(node.scrollHeight - node.clientHeight - 48, 0);
      return node.scrollTop;
    });

    expect(before).toBeGreaterThanOrEqual(0);

    const toggled = await page.evaluate(() => {
      const memoryLabel = Array.from(document.querySelectorAll("[data-runtime-config-surface='conversation'] .conversation-check-item"))
        .find((item) => item.textContent?.toLowerCase().includes("memory"));
      const input = memoryLabel?.querySelector("input[type='checkbox']");
      if (!(input instanceof HTMLInputElement)) {
        return false;
      }
      input.click();
      return true;
    });

    expect(toggled).toBe(true);

    await expect.poll(async () => body.evaluate((node) => node.scrollTop)).toBeGreaterThanOrEqual(0);
    const after = await body.evaluate((node) => node.scrollTop);
    expect(Math.abs(after - before)).toBeLessThan(80);
  });

  test("dismisses the mobile keyboard before opening the session settings sheet", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 393, height: 852 });
    const { composer } = await openChatWorkspace(page);

    const runtimeToggle = runtimeSettingsToggle(page);
    const panel = runtimeSettingsPanel(page);
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
    await expect(panel).toBeVisible();
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");

    const metrics = await page.evaluate(() => {
      const sheet = document.querySelector("[data-runtime-config-surface='conversation']");
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
    const readComposerMetrics = async () => page.evaluate(() => {
      const shell = document.querySelector("[data-runtime-composer-kind='chat']");
      const inputNode = document.querySelector("[data-composer-input='conversation']");
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

    await input.click();
    await setVisualViewport(page, { width: 760, height: 620, offsetTop: 0 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("360px");

    const opened = await readComposerMetrics();
    expect(opened).not.toBeNull();
    expect(opened?.shellBottom ?? 0).toBeLessThanOrEqual((opened?.viewportBottom ?? 0) + 2);
    expect(opened?.inputBottom ?? 0).toBeLessThanOrEqual((opened?.viewportBottom ?? 0) - 8);

    await setVisualViewport(page, { width: 760, height: 700, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("280px");

    const tallerViewportKeyboard = await readComposerMetrics();
    expect(tallerViewportKeyboard).not.toBeNull();
    expect(tallerViewportKeyboard?.shellBottom ?? 0).toBeLessThanOrEqual((tallerViewportKeyboard?.viewportBottom ?? 0) + 2);
    expect(tallerViewportKeyboard?.inputBottom ?? 0).toBeLessThanOrEqual((tallerViewportKeyboard?.viewportBottom ?? 0) - 8);

    await setVisualViewport(page, { width: 760, height: 560, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("420px");

    const shorterViewportKeyboard = await readComposerMetrics();
    expect(shorterViewportKeyboard).not.toBeNull();
    expect(shorterViewportKeyboard?.shellBottom ?? 0).toBeLessThanOrEqual((shorterViewportKeyboard?.viewportBottom ?? 0) + 2);
    expect(shorterViewportKeyboard?.inputBottom ?? 0).toBeLessThanOrEqual((shorterViewportKeyboard?.viewportBottom ?? 0) - 8);

    await setVisualViewport(page, { width: 760, height: 980, offsetTop: 0 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--mobile-viewport-height").trim()
    )).toBe("980px");

    const closed = await page.evaluate(() => {
      const shell = document.querySelector("[data-runtime-composer-kind='chat']");
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

  test("keeps chat chrome in the workspace grid while the composer follows the dynamic viewport", async ({ page }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    await page.goto("/chat");
    await loginIfNeeded(page);
    await waitForAppReady(page);
    await page.waitForSelector("[data-composer-form='conversation']", { timeout: 20000 });
    const input = page.locator("[data-composer-input='conversation']");
    const screen = page.locator("[data-runtime-screen='conversation']");

    const readMetrics = async () => page.evaluate(() => {
      const appShell = document.querySelector(".app-shell");
      const chatPane = document.querySelector(".chat-pane");
      const mobileHeader = document.querySelector("[data-runtime-mobile-variant='conversation']");
      const workspaceHeader = document.querySelector("[data-runtime-workspace-header='true']");
      const workspaceBody = document.querySelector("[data-runtime-view='conversation'] .runtime-workspace-body");
      const workspaceScreen = document.querySelector("[data-runtime-view='conversation'] .runtime-workspace-screen");
      const composerShell = document.querySelector(".runtime-composer-shell");
      const viewport = window.visualViewport;
      if (
        !(appShell instanceof HTMLElement)
        || !(chatPane instanceof HTMLElement)
        || !(mobileHeader instanceof HTMLElement)
        || !(workspaceHeader instanceof HTMLElement)
        || !(workspaceBody instanceof HTMLElement)
        || !(workspaceScreen instanceof HTMLElement)
        || !(composerShell instanceof HTMLElement)
        || !viewport
      ) {
        return null;
      }

      const appShellRect = appShell.getBoundingClientRect();
      const chatPaneRect = chatPane.getBoundingClientRect();
      const mobileHeaderRect = mobileHeader.getBoundingClientRect();
      const workspaceHeaderRect = workspaceHeader.getBoundingClientRect();
      const workspaceScreenRect = workspaceScreen.getBoundingClientRect();
      const composerRect = composerShell.getBoundingClientRect();
      const mobileHeaderStyle = getComputedStyle(mobileHeader);
      return {
        keyboardOffset: getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim(),
        mobileViewportOffsetTop: getComputedStyle(document.documentElement).getPropertyValue("--mobile-viewport-offset-top").trim(),
        viewportHeight: viewport.height,
        viewportOffsetTop: viewport.offsetTop,
        viewportBottom: viewport.height + viewport.offsetTop,
        windowScrollY: window.scrollY,
        appShellTop: appShellRect.top,
        appShellScreenTop: appShellRect.top - viewport.offsetTop,
        appShellHeight: appShellRect.height,
        chatPaneHeight: chatPaneRect.height,
        mobileHeaderPosition: mobileHeaderStyle.position,
        mobileHeaderTop: mobileHeaderRect.top,
        mobileHeaderScreenTop: mobileHeaderRect.top - viewport.offsetTop,
        mobileHeaderBottom: mobileHeaderRect.bottom,
        mobileHeaderScreenBottom: mobileHeaderRect.bottom - viewport.offsetTop,
        workspaceHeaderTop: workspaceHeaderRect.top,
        workspaceHeaderScreenTop: workspaceHeaderRect.top - viewport.offsetTop,
        workspaceHeaderBottom: workspaceHeaderRect.bottom,
        workspaceHeaderScreenBottom: workspaceHeaderRect.bottom - viewport.offsetTop,
        workspaceBodyScrollTop: workspaceBody.scrollTop,
        workspaceScreenTop: workspaceScreenRect.top,
        workspaceScreenScreenTop: workspaceScreenRect.top - viewport.offsetTop,
        workspaceScreenHeight: workspaceScreenRect.height,
        workspaceScreenScrollTop: workspaceScreen.scrollTop,
        workspaceScreenBottomDistance: workspaceScreen.scrollHeight - workspaceScreen.clientHeight - workspaceScreen.scrollTop,
        composerTop: composerRect.top,
        composerBottom: composerRect.bottom,
        composerScreenBottom: composerRect.bottom - viewport.offsetTop,
      };
    });

    const baseline = await readMetrics();
    expect(baseline).not.toBeNull();
    expect(baseline?.keyboardOffset).toBe("0px");
    await screen.evaluate((node) => {
      node.scrollTop = node.scrollHeight;
      node.dispatchEvent(new Event("scroll"));
    });
    await expect.poll(async () => screen.evaluate((node) =>
      Math.round(node.scrollHeight - node.clientHeight - node.scrollTop)
    )).toBeLessThanOrEqual(24);

    await input.click();
    await page.setViewportSize({ width: 430, height: 620 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--mobile-viewport-height").trim()
    )).toBe("620px");
    const opened = await readMetrics();
    expect(opened).not.toBeNull();
    expect(opened?.keyboardOffset).toBe("312px");
    expect(Math.abs((opened?.composerBottom ?? 0) - (opened?.viewportBottom ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.windowScrollY ?? 0) - (baseline?.windowScrollY ?? 0))).toBeLessThanOrEqual(1);
    expect(Math.abs((opened?.appShellTop ?? 0) - (baseline?.appShellTop ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.appShellHeight ?? 0) - (opened?.viewportHeight ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.chatPaneHeight ?? 0) - (opened?.viewportHeight ?? 0))).toBeLessThanOrEqual(2);
    expect(opened?.mobileHeaderPosition).toBe("fixed");
    expect(Math.abs(opened?.mobileHeaderTop ?? 0)).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.mobileHeaderTop ?? 0) - (baseline?.mobileHeaderTop ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.mobileHeaderBottom ?? 0) - (baseline?.mobileHeaderBottom ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.workspaceHeaderTop ?? 0) - (baseline?.workspaceHeaderTop ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.workspaceHeaderBottom ?? 0) - (baseline?.workspaceHeaderBottom ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs((opened?.workspaceBodyScrollTop ?? 0) - (baseline?.workspaceBodyScrollTop ?? 0))).toBeLessThanOrEqual(1);
    expect(Math.abs((opened?.workspaceScreenTop ?? 0) - (baseline?.workspaceScreenTop ?? 0))).toBeLessThanOrEqual(2);
    expect(opened?.workspaceScreenHeight ?? 0).toBeLessThanOrEqual((baseline?.workspaceScreenHeight ?? Number.POSITIVE_INFINITY) + 2);
    expect(opened?.workspaceScreenBottomDistance ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(24);
    expect((baseline?.composerTop ?? 0) - (opened?.composerTop ?? 0)).toBeGreaterThan(120);

    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    });
    await page.setViewportSize({ width: 430, height: 932 });

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

  test("keeps the mobile composer visible under the nav drawer after dismissing the keyboard", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 430, height: 932 });
    const { composer } = await openChatWorkspace(page);
    const input = composer.input();

    await input.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");

    await page.getByRole("button", { name: "Menu" }).first().click();
    await setVisualViewport(page, { width: 430, height: 932, offsetTop: 0 });

    await expect(input).not.toBeFocused();
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("0px");

    const metrics = await page.evaluate(() => {
      const composerShell = document.querySelector("[data-runtime-composer-kind='chat']");
      const workspaceBody = document.querySelector(".runtime-workspace-body");
      const backdrop = document.querySelector(".workbench-mobile-overlay-portal .mobile-backdrop");
      const drawer = document.querySelector(".workbench-mobile-overlay-portal .primary-nav");
      if (
        !(composerShell instanceof HTMLElement) ||
        !(workspaceBody instanceof HTMLElement) ||
        !(backdrop instanceof HTMLElement) ||
        !(drawer instanceof HTMLElement)
      ) {
        return null;
      }
      const composerStyle = getComputedStyle(composerShell);
      const backdropStyle = getComputedStyle(backdrop);
      const drawerStyle = getComputedStyle(drawer);
      return {
        composerVisibility: composerStyle.visibility,
        composerOpacity: composerStyle.opacity,
        bodyInteractive: workspaceBody.getAttribute("data-runtime-composer-interactive"),
        composerInWorkspace: workspaceBody.contains(composerShell),
        portalHostExists: Boolean(document.querySelector("[data-runtime-composer-portal-host='chat']")),
        composerPointerEvents: composerStyle.pointerEvents,
        backdropPointerEvents: backdropStyle.pointerEvents,
        composerZIndex: Number(composerStyle.zIndex),
        backdropZIndex: Number(backdropStyle.zIndex),
        drawerZIndex: Number(drawerStyle.zIndex),
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.composerVisibility).toBe("visible");
    expect(metrics?.composerOpacity).toBe("1");
    expect(metrics?.bodyInteractive).toBe("false");
    expect(metrics?.composerInWorkspace).toBe(true);
    expect(metrics?.portalHostExists).toBe(false);
    expect(metrics?.composerPointerEvents).toBe("none");
    expect(metrics?.backdropPointerEvents).toBe("auto");
    expect(metrics?.backdropZIndex ?? 0).toBeGreaterThan(metrics?.composerZIndex ?? 0);
    expect(metrics?.drawerZIndex ?? 0).toBeGreaterThan(metrics?.composerZIndex ?? 0);
  });

  test("lets the mobile chat transcript scroll back to the top while the keyboard is open", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 430, height: 932 });
    const sessionID = compactChatSessionID("mobile-keyboard-scroll-top-chat");
    await mockChatRuntimeSessions(page, [{
      id: sessionID,
      title: "成都旅游攻略",
      title_auto: false,
      title_score: 8,
      created_at: "2026-06-28T10:00:00Z",
      target_type: "model",
      target_id: "codex",
      target_name: "Codex",
      status: "done",
      turns: [
        {
          id: "mobile-keyboard-scroll-top-turn",
          prompt: "成都旅游攻略",
          started_at: "2026-06-28T10:00:00Z",
          finished_at: "2026-06-28T10:01:00Z",
          status: "success",
          final_output: [
            "按第一次来成都、3天2晚/4天3晚、不自驾来排。",
            "",
            "## 推荐池",
            "",
            "景点优先级：",
            "",
            "- 必去：成都大熊猫繁育研究基地、武侯祠/锦里、杜甫草堂、成都博物馆、人民公园/鹤鸣茶社、宽窄巷子、春熙路/太古里/IFS。",
            "- 博物馆线：成都博物馆、金沙遗址博物馆、三星堆博物馆。成都博物馆周一闭馆，周五周六夜间开放到20:30；金沙9:00-18:00，17:00停止入馆。",
            "- 体验线：盖碗茶、川剧变脸、采耳、夜游九眼桥。",
            "- 周边加一天：三星堆；或都江堰+青城山。",
            "",
            "美食推荐池：",
            "",
            "- 火锅：蜀大侠、小龙坎、青年火锅。",
            "- 串串：钢管厂五区小郡肝、冒椒火辣。",
            "- 小吃：钟水饺、龙抄手、赖汤圆、蛋烘糕、甜水面、担担面。",
            "- 茶馆：鹤鸣茶社、望江楼公园茶馆。",
            "",
            "行程建议：",
            "",
            "D1：抵达后人民公园喝茶，晚上春熙路、太古里、IFS。D2：熊猫基地早起，下午武侯祠和锦里，晚上川剧。D3：杜甫草堂、成都博物馆，晚上九眼桥。D4：三星堆或都江堰青城山。",
            "",
            "交通建议：市区优先地铁，熊猫基地建议早出发。热门博物馆和三星堆提前预约。熊猫基地尽量上午去，避开中午高温和人流。",
            "",
            "预算参考：",
            "",
            "- 市区公共交通和打车结合，人均交通按每天60-100元预留。",
            "- 餐饮按小吃、茶馆、火锅搭配，人均每天160-260元更稳妥。",
            "- 博物馆和景区按预约规则执行，临时改线优先保留熊猫基地和成都博物馆。",
            "",
            "避坑提醒：",
            "",
            "- 熊猫基地不要中午到，上午入园更容易看到活跃状态。",
            "- 宽窄巷子适合顺路扫街，不建议把整晚都押在这里。",
            "- 三星堆和都江堰青城山都适合单独占一天，不建议同一天硬拼。",
            "",
            "补充路线：",
            "",
            "- 慢节奏版：人民公园、杜甫草堂、武侯祠、夜游锦江。",
            "- 博物馆版：成都博物馆、金沙遗址、三星堆。",
            "- 城市夜游版：太古里、IFS、九眼桥、玉林路。",
            "",
            "如果只有2天，第一天熊猫基地、武侯祠、锦里、川剧；第二天成都博物馆、人民公园、春熙路和太古里。这样不会太赶，也能覆盖第一次来成都的核心体验。",
          ].join("\n"),
        },
      ],
    }]);
    await page.addInitScript((activeSessionID) => {
      window.sessionStorage.setItem("alter0.web.session.active.v1", JSON.stringify({
        chat: activeSessionID,
      }));
    }, sessionID);

    await page.goto(`/chat?session_id=${sessionID}`);
    await loginIfNeeded(page);
    await waitForAppReady(page);
    await page.waitForSelector("[data-message-id='mobile-keyboard-scroll-top-turn:assistant']", { timeout: 20000 });
    const screen = page.locator("[data-runtime-screen='conversation']");
    const input = page.locator("[data-composer-input='conversation']");

    await expect.poll(async () => screen.evaluate((node) => node.scrollHeight > node.clientHeight + 240)).toBe(true);
    await screen.evaluate((node) => {
      node.scrollTop = node.scrollHeight;
      node.dispatchEvent(new Event("scroll"));
    });
    await input.click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");
    await expect.poll(async () => screen.evaluate((node) =>
      Math.round(node.scrollHeight - node.clientHeight - node.scrollTop)
    )).toBeLessThanOrEqual(24);
    await expect.poll(async () => page.evaluate(() => {
      const screenNode = document.querySelector("[data-runtime-screen='conversation']");
      const workspaceBody = screenNode?.closest(".runtime-workspace-body");
      const composer = document.querySelector("[data-runtime-composer-kind='chat']");
      if (!(workspaceBody instanceof HTMLElement) || !(screenNode instanceof HTMLElement)) {
        return null;
      }
      if (!(composer instanceof HTMLElement)) {
        return null;
      }
      const screenRect = screenNode.getBoundingClientRect();
      const composerRect = composer.getBoundingClientRect();
      return {
        screenPaddingBottom: getComputedStyle(screenNode).paddingBottom,
        screenEndsBeforeComposer: screenRect.bottom <= composerRect.top + 2,
      };
    })).toEqual({
      screenPaddingBottom: "20px",
      screenEndsBeforeComposer: true,
    });

    const screenBox = await screen.boundingBox();
    expect(screenBox).not.toBeNull();
    await page.mouse.move((screenBox?.x ?? 0) + (screenBox?.width ?? 0) / 2, (screenBox?.y ?? 0) + 96);
    await page.mouse.wheel(0, -2600);

    await expect.poll(async () => screen.evaluate((node) => Math.round(node.scrollTop))).toBeLessThanOrEqual(8);
    const metrics = await page.evaluate(() => {
      const header = document.querySelector("[data-runtime-mobile-variant='conversation']");
      const screenNode = document.querySelector("[data-runtime-screen='conversation']");
      const firstMessage = document.querySelector("[data-message-id='mobile-keyboard-scroll-top-turn:assistant']");
      if (!(header instanceof HTMLElement) || !(screenNode instanceof HTMLElement) || !(firstMessage instanceof HTMLElement)) {
        return null;
      }
      return {
        headerBottom: header.getBoundingClientRect().bottom,
        screenTop: screenNode.getBoundingClientRect().top,
        firstMessageTop: firstMessage.getBoundingClientRect().top,
      };
    });
    expect(metrics).not.toBeNull();
    expect(metrics?.screenTop ?? 0).toBeGreaterThanOrEqual((metrics?.headerBottom ?? 0) - 1);
    expect(metrics?.firstMessageTop ?? 0).toBeGreaterThanOrEqual((metrics?.screenTop ?? 0) - 1);
  });

  test("keeps the mobile runtime header in the workspace grid when visual viewport offset changes", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 430, height: 932 });
    await page.goto("/chat");
    await loginIfNeeded(page);
    await waitForAppReady(page);
    await page.waitForSelector("[data-composer-form='conversation']", { timeout: 20000 });

    await page.locator("[data-composer-input='conversation']").click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 312 });

    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--mobile-viewport-offset-top").trim()
    )).toBe("312px");

    const metrics = await page.evaluate(() => {
      const mobileHeader = document.querySelector("[data-runtime-mobile-variant='conversation']");
      const viewport = window.visualViewport;
      if (!(mobileHeader instanceof HTMLElement) || !viewport) {
        return null;
      }
      const rect = mobileHeader.getBoundingClientRect();
      return {
        mobileViewportHeight: getComputedStyle(document.documentElement).getPropertyValue("--mobile-viewport-height").trim(),
        keyboardOffset: getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim(),
        offsetTop: viewport.offsetTop,
        viewportBottom: viewport.height + viewport.offsetTop,
        headerTop: rect.top,
        headerScreenTop: rect.top - viewport.offsetTop,
        headerPosition: getComputedStyle(mobileHeader).position,
        headerTopStyle: getComputedStyle(mobileHeader).top,
        headerTransform: getComputedStyle(mobileHeader).transform,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.mobileViewportHeight).toBe(`${metrics?.viewportBottom}px`);
    expect(metrics?.keyboardOffset).toBe("0px");
    expect(metrics?.headerPosition).toBe("fixed");
    expect(metrics?.headerTopStyle).toBe(`${metrics?.offsetTop}px`);
    expect(metrics?.headerTransform).toBe("none");
    expect(Math.abs((metrics?.headerTop ?? 0) - (metrics?.offsetTop ?? 0))).toBeLessThanOrEqual(2);
    expect(Math.abs(metrics?.headerScreenTop ?? 0)).toBeLessThanOrEqual(2);
  });

  test("keeps mobile route pages aligned to the visual viewport bottom", async ({ page }) => {
    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 393, height: 980 });
    await openSettingsRoute(page);

    await setVisualViewport(page, { width: 393, height: 760, offsetTop: 0 });

    const metrics = await page.evaluate(() => {
      const pane = document.querySelector(".chat-pane");
      const route = document.querySelector(".route-view[data-route='settings']");
      const viewport = window.visualViewport;
      if (!(pane instanceof HTMLElement) || !(route instanceof HTMLElement) || !viewport) {
        return null;
      }
      return {
        viewportBottom: viewport.height + viewport.offsetTop,
        windowBottom: window.innerHeight,
        paneBottom: pane.getBoundingClientRect().bottom,
        routeBottom: route.getBoundingClientRect().bottom,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.paneBottom ?? 0).toBeLessThanOrEqual((metrics?.windowBottom ?? 0) + 2);
    expect(metrics?.routeBottom ?? 0).toBeLessThanOrEqual((metrics?.paneBottom ?? 0) + 10);
  });

  test("keeps the mobile navigation fully reachable on short viewports", async ({ page }) => {
    await page.setViewportSize({ width: 760, height: 680 });
    await openChatWorkspace(page);

    const navToggle = mobileMenuButton(page);
    const primaryNav = page.locator(".primary-nav");
    const localeButton = page.locator(".nav-locale-button");

    await navToggle.click();
    await expect(primaryNav).toBeVisible();

    const before = await primaryNav.evaluate((node) => ({
      scrollHeight: node.scrollHeight,
      clientHeight: node.clientHeight,
      scrollTop: node.scrollTop,
    }));

    expect(before.scrollHeight).toBeGreaterThanOrEqual(before.clientHeight);

    await primaryNav.evaluate((node) => {
      node.scrollTop = node.scrollHeight;
    });

    const after = await primaryNav.evaluate((node) => ({
      scrollTop: node.scrollTop,
      top: node.getBoundingClientRect().top,
      bottom: node.getBoundingClientRect().bottom,
    }));
    const localeBox = await localeButton.boundingBox();

    expect(after.scrollTop).toBeGreaterThanOrEqual(0);
    expect(localeBox).not.toBeNull();
    expect(localeBox?.y ?? 0).toBeGreaterThanOrEqual((after.top ?? 0) - 1);
    expect((localeBox?.y ?? 0) + (localeBox?.height ?? 0)).toBeLessThanOrEqual((after.bottom ?? 0) + 1);
  });

  test("keeps unsent content when re-entering Chat", async ({ page }) => {
    const { appShellPage, composer } = await openChatWorkspaceWithDraft(page, "unsent draft");
    await expectComposerState(composer, { draft: "dirty" });
    page.on("dialog", async (dialog) => {
      throw new Error(`unexpected unsent draft dialog: ${dialog.message()}`);
    });

    await appShellPage.routeMenuItem("chat").click();
    await expect(page).toHaveURL(/\/chat(?:\?.*)?$/);
    await expectComposerValue(composer, "unsent draft");
  });

  test("restores draft after reload", async ({ page }) => {
    const { composer } = await openChatWorkspace(page);
    const input = composer.input();

    await input.click();
    await input.pressSequentially("draft message");
    await expectComposerValue(composer, "draft message");
    await expectComposerState(composer, { draft: "dirty" });
    await expectChatDraftPersisted(page, "draft message");

    const { composer: reloadedComposer } = await reloadChatWorkspace(page);

    await expectComposerValue(reloadedComposer, "draft message");
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
    const { composer } = await openChatWorkspaceWithTwoDraftSessions(page);

    await removeChatSession(page, 0);

    await expectComposerValue(composer, "draft-a");

    await createNewChatSession(page);

    await expectComposerValue(composer, "");
    await expectComposerState(composer, { draft: "empty" });
  });

  test("restores session scoped drafts after reload", async ({ page }) => {
    await openChatWorkspaceWithTwoDraftSessions(page);
    await expectChatDraftPersisted(page, "draft-b");

    const { composer } = await reloadChatWorkspace(page);
    await expectComposerValue(composer, "draft-b");

    await switchChatSession(page, 1);
    await expectComposerValue(composer, "draft-a");
  });

  test("keeps current draft when an inactive session is removed", async ({ page }) => {
    const { composer } = await openChatWorkspaceWithTwoDraftSessions(page);

    await removeChatSession(page, 1);
    await expectComposerValue(composer, "draft-b");
    await expectActiveChatSession(page, 0);
  });

  test("keeps IME composition text when pressing Enter during composition", async ({ page }) => {
    const { composer } = await openChatWorkspace(page);
    const input = composer.input();
    await startIMEInput(input);
    await expectComposerState(composer, { composing: true, draft: "dirty" });

    await pressEnterDuringIMEInput(input);

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

    await input.fill("修改 chatRuntime 和 skill 的会话标题");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("修改 chatRuntime 和 skill 的会话标题");
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
      const bubbles = [...document.querySelectorAll(".runtime-message-user .runtime-message-bubble")];
      const bubble = bubbles[bubbles.length - 1] || null;
      const message = bubble ? bubble.closest(".runtime-message-user") : null;
      const list = document.querySelector(".runtime-timeline");
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

  test("renders structured process events from chat message results", async ({ page }) => {
    const sessionID = compactChatSessionID("chat-process-events");
    await page.addInitScript((chatSessionID) => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (new URL(url, window.location.href).pathname.endsWith("/input")) {
          return new Response(JSON.stringify({
            session: {
              id: chatSessionID,
              title: "检查仓库状态",
              status: "ready",
              created_at: "2026-06-18T00:00:00Z",
              turns: [{
                id: "turn-process",
                prompt: "帮我检查仓库状态",
                status: "success",
                final_output: "任务已完成",
                runtime_trace_events: [
                  ["event-1", "读取运行状态", "检查仓库状态、当前分支和工作区清洁度。"],
                  ["event-2", "定位 Thinking 样式", "确认移动端展开逻辑来自 .runtime-thinking-shell .chatRuntime-process-body。"],
                  ["event-3", "调整展开方式", "将过程详情保持在当前消息内联展开，不再脱离消息流。"],
                  ["event-4", "回归验证", "补充样式断言并确认最终回复仍独立展示。"],
                ].map(([id, title, summary], index) => ({
                  id,
                  turn_id: "turn-process",
                  seq: index + 1,
                  source: "adapter",
                  provider: { engine: "codex", adapter: "codex_cli_json", event_type: "message", item_id: id },
                  role: "assistant",
                  kind: "assistant_commentary",
                  lifecycle: "completed",
                  status: "completed",
                  title,
                  summary,
                  blocks: [{ type: "markdown", text: summary }],
                  visibility: "collapsed",
                  raw: { ref: id, type: "message", has_detail: true },
                }))
              }]
            },
          }), {
            status: 200,
            headers: { "Content-Type": "application/json" }
          });
        }
        return originalFetch(input, init);
      };
    }, sessionID);

    await mockRuntimeSession(page, {
      route: "chat",
      session: {
        id: sessionID,
        title: "检查仓库状态",
        status: "ready",
        created_at: "2026-06-18T00:00:00Z",
        turns: [{
          id: "turn-process",
          prompt: "帮我检查仓库状态",
          status: "success",
          final_output: "任务已完成",
          runtime_trace_events: [
            ["event-1", "读取运行状态", "检查仓库状态、当前分支和工作区清洁度。"],
            ["event-2", "定位 Thinking 样式", "确认移动端展开逻辑来自 .runtime-thinking-shell .chatRuntime-process-body。"],
            ["event-3", "调整展开方式", "将过程详情保持在当前消息内联展开，不再脱离消息流。"],
            ["event-4", "回归验证", "补充样式断言并确认最终回复仍独立展示。"],
          ].map(([id, title, summary], index) => ({
            id,
            turn_id: "turn-process",
            seq: index + 1,
            source: "adapter",
            provider: { engine: "codex", adapter: "codex_cli_json", event_type: "message", item_id: id },
            role: "assistant",
            kind: "assistant_commentary",
            lifecycle: "completed",
            status: "completed",
            title,
            summary,
            blocks: [{ type: "markdown", text: summary }],
            visibility: "collapsed",
            raw: { ref: id, type: "message", has_detail: true },
          })),
        }],
      },
    });
    await page.goto(`/chat?session_id=${sessionID}`);
    await loginIfNeeded(page);
    await waitForAppReady(page);

    const assistantMessage = latestAssistantMessage(page);
    await expect(assistantMessage.locator("[data-conversation-process-shell]")).toBeVisible();
    await assistantMessage.locator(".chatRuntime-process-body").evaluate((node) => {
      if (node instanceof HTMLElement) {
        node.hidden = false;
        node.removeAttribute("hidden");
      }
    });
    await expect(assistantMessage.locator("[data-conversation-process-toggle]")).toContainText("Thinking");
    await expect(assistantMessage.locator("[data-conversation-process-toggle]")).toContainText("4 steps");
    await expect(assistantMessage.locator("[data-conversation-process-step]")).toHaveCount(4);
    await expect(assistantMessage.locator(".chatRuntime-step-body").first()).toContainText("检查仓库状态");
    await expect(assistantMessage.locator(".chatRuntime-step-body").nth(2)).toContainText("当前消息内联展开");
    await expect(assistantMessage.locator(".conversation-process-answer-shell")).toContainText("任务已完成");
    await expect(assistantMessage.locator(".conversation-process-answer-shell")).not.toContainText("[process] action:");
  });

  test("keeps structured skill process detail readable on mobile", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    const sessionID = compactChatSessionID("chat-mobile-process");
    await page.addInitScript((chatSessionID) => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (new URL(url, window.location.href).pathname.endsWith("/input")) {
          return new Response(JSON.stringify({
            session: {
              id: chatSessionID,
              title: "检查仓库同步情况",
              status: "ready",
              created_at: "2026-06-18T00:00:00Z",
              turns: [{
                id: "turn-mobile-process",
                prompt: "帮我检查仓库同步情况",
                status: "success",
                final_output: "任务已完成",
                runtime_trace_events: [
                  ["mobile-event-1", "确认目标工作区", "需要把远端最新的 alter0 项目克隆到当前会话的单独工作区中，并检查工作区结构、远端分支和当前 HEAD 是否对齐。"],
                  ["mobile-event-2", "读取前端契约", "检查 Chat 与 ChatRuntime 共享的 RuntimeTimeline process block，确认 Thinking 披露入口复用同一 DOM 契约。"],
                  ["mobile-event-3", "调整移动端展开", "移动端 Process 展开体保持在当前 assistant 消息内，避免独立 fixed 面板遮挡 Composer 或脱离上下文。"],
                  ["mobile-event-4", "同步静态产物", "重新构建前端产物，使部署子域名加载新的哈希 CSS 和 JS。"],
                  ["mobile-event-5", "部署预览服务", "通过 session scoped web 服务注册到短哈希子域名，并使用 /readyz 完成健康检查。"],
                  ["mobile-event-6", "补充测试数据", "增加多步骤思考过程 fixture，覆盖长过程在窄屏同页展开时的宽度、换行和滚动表现。"],
                ].map(([id, title, summary], index) => ({
                  id,
                  turn_id: "turn-mobile-process",
                  seq: index + 1,
                  source: "adapter",
                  provider: { engine: "codex", adapter: "codex_cli_json", event_type: "message", item_id: id },
                  role: "assistant",
                  kind: "assistant_commentary",
                  lifecycle: "completed",
                  status: "completed",
                  title,
                  summary,
                  blocks: [{ type: "markdown", text: summary }],
                  visibility: "collapsed",
                  raw: { ref: id, type: "message", has_detail: true },
                }))
              }]
            },
          }), {
            status: 200,
            headers: { "Content-Type": "application/json" }
          });
        }
        return originalFetch(input, init);
      };
    }, sessionID);

    await mockRuntimeSession(page, {
      route: "chat",
      session: {
        id: sessionID,
        title: "检查仓库同步情况",
        status: "ready",
        created_at: "2026-06-18T00:00:00Z",
        turns: [{
          id: "turn-mobile-process",
          prompt: "帮我检查仓库同步情况",
          status: "success",
          final_output: "任务已完成",
          runtime_trace_events: [
            ["mobile-event-1", "确认目标工作区", "需要把远端最新的 alter0 项目克隆到当前会话的单独工作区中，并检查工作区结构、远端分支和当前 HEAD 是否对齐。"],
            ["mobile-event-2", "读取前端契约", "检查 Chat 与 ChatRuntime 共享的 RuntimeTimeline process block，确认 Thinking 披露入口复用同一 DOM 契约。"],
            ["mobile-event-3", "调整移动端展开", "移动端 Process 展开体保持在当前 assistant 消息内，避免独立 fixed 面板遮挡 Composer 或脱离上下文。"],
            ["mobile-event-4", "同步静态产物", "重新构建前端产物，使部署子域名加载新的哈希 CSS 和 JS。"],
            ["mobile-event-5", "部署预览服务", "通过 session scoped web 服务注册到短哈希子域名，并使用 /readyz 完成健康检查。"],
            ["mobile-event-6", "补充测试数据", "增加多步骤思考过程 fixture，覆盖长过程在窄屏同页展开时的宽度、换行和滚动表现。"],
          ].map(([id, title, summary], index) => ({
            id,
            turn_id: "turn-mobile-process",
            seq: index + 1,
            source: "adapter",
            provider: { engine: "codex", adapter: "codex_cli_json", event_type: "message", item_id: id },
            role: "assistant",
            kind: "assistant_commentary",
            lifecycle: "completed",
            status: "completed",
            title,
            summary,
            blocks: [{ type: "markdown", text: summary }],
            visibility: "collapsed",
            raw: { ref: id, type: "message", has_detail: true },
          })),
        }],
      },
    });
    await page.goto(`/chat?session_id=${sessionID}`);
    await loginIfNeeded(page);
    await waitForAppReady(page);
    const assistantMessage = latestAssistantMessage(page);
    await expect(assistantMessage.locator("[data-conversation-process-shell]")).toBeVisible();
    await assistantMessage.locator(".chatRuntime-process-body").evaluate((node) => {
      if (node instanceof HTMLElement) {
        node.hidden = false;
        node.removeAttribute("hidden");
      }
    });

    const processBody = assistantMessage.locator(".chatRuntime-step-body").first();
    await expect(processBody).toContainText("需要把远端最新的 alter0 项目克隆到当前会话的单独工作区中");
    await expect(assistantMessage.locator("[data-conversation-process-toggle]")).toContainText("6 steps");
    await expect(assistantMessage.locator("[data-conversation-process-step]")).toHaveCount(6);
    await assistantMessage.locator("[data-conversation-process-step-toggle]").first().click();
    await expect(assistantMessage.locator(".chatRuntime-step-body").nth(5)).toContainText("增加多步骤思考过程 fixture");

    const metrics = await processBody.evaluate((node) => {
      const detail = node instanceof HTMLElement ? node : null;
      const rendered = detail?.querySelector(".chatRuntime-step-richtext");
      const step = detail?.closest("[data-conversation-process-step]");
      const shell = detail?.closest("[data-conversation-process-shell]");
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
    const sessionID = compactChatSessionID("chat-node-go-process");
    await page.addInitScript((chatSessionID) => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const url = typeof input === "string"
          ? input
          : input instanceof Request
            ? input.url
            : String(input || "");
        if (new URL(url, window.location.href).pathname.endsWith("/input")) {
          return new Response(JSON.stringify({
            session: {
              id: chatSessionID,
              title: "Node 和 Go 的差异",
              status: "ready",
              created_at: "2026-06-18T00:00:00Z",
              turns: [{
                id: "turn-node-go",
                prompt: "详细介绍下 node 和 go 的差异",
                status: "success",
                final_output: "Node 更偏应用层与生态速度，Go 更偏并发效率与部署稳定性。",
                runtime_trace_events: [{
                  id: "event-node-go",
                  turn_id: "turn-node-go",
                  seq: 1,
                  source: "adapter",
                  provider: { engine: "codex", adapter: "codex_cli_json", event_type: "message", item_id: "event-node-go" },
                  role: "assistant",
                  kind: "assistant_commentary",
                  lifecycle: "completed",
                  status: "completed",
                  title: "codex_exec",
                  summary: "整理 Node 与 Go 在运行时模型、并发方式、构建发布和工程适配上的主要差异。",
                  blocks: [{ type: "markdown", text: "整理 Node 与 Go 在运行时模型、并发方式、构建发布和工程适配上的主要差异。" }],
                  visibility: "collapsed",
                  raw: { ref: "event-node-go", type: "message", has_detail: true },
                }]
              }]
            },
          }), {
            status: 200,
            headers: { "Content-Type": "application/json" }
          });
        }
        return originalFetch(input, init);
      };
    }, sessionID);

    await mockRuntimeSession(page, {
      route: "chat",
      session: {
        id: sessionID,
        title: "Node 和 Go 的差异",
        status: "ready",
        created_at: "2026-06-18T00:00:00Z",
        turns: [{
          id: "turn-node-go",
          prompt: "详细介绍下 node 和 go 的差异",
          status: "success",
          final_output: "Node 更偏应用层与生态速度，Go 更偏并发效率与部署稳定性。",
          runtime_trace_events: [{
            id: "event-node-go",
            turn_id: "turn-node-go",
            seq: 1,
            source: "adapter",
            provider: { engine: "codex", adapter: "codex_cli_json", event_type: "message", item_id: "event-node-go" },
            role: "assistant",
            kind: "assistant_commentary",
            lifecycle: "completed",
            status: "completed",
            title: "codex_exec",
            summary: "整理 Node 与 Go 在运行时模型、并发方式、构建发布和工程适配上的主要差异。",
            blocks: [{ type: "markdown", text: "整理 Node 与 Go 在运行时模型、并发方式、构建发布和工程适配上的主要差异。" }],
            visibility: "collapsed",
            raw: { ref: "event-node-go", type: "message", has_detail: true },
          }],
        }],
      },
    });
    await page.goto(`/chat?session_id=${sessionID}`);
    await loginIfNeeded(page);
    await waitForAppReady(page);

    const assistantMessage = latestAssistantMessage(page);
    await expect(assistantMessage.locator("[data-conversation-process-shell]")).toBeVisible();

    const metrics = await page.evaluate(() => {
      const timeline = document.querySelector(".runtime-timeline");
      const userMessages = Array.from(document.querySelectorAll(".msg.user"));
      const assistantMessages = Array.from(document.querySelectorAll(".msg.assistant"));
      const user = userMessages[userMessages.length - 1];
      const assistant = assistantMessages[assistantMessages.length - 1];
      const userBubble = user?.querySelector(".msg-bubble");
      const assistantProcess = assistant?.querySelector("[data-conversation-process-shell]");
      const assistantAnswer = assistant?.querySelector(".conversation-process-answer-shell");
      if (
        !(timeline instanceof HTMLElement)
        || !(user instanceof HTMLElement)
        || !(assistant instanceof HTMLElement)
        || !(userBubble instanceof HTMLElement)
        || !(assistantProcess instanceof HTMLElement)
        || !(assistantAnswer instanceof HTMLElement)
      ) {
        return null;
      }
      const timelineStyle = getComputedStyle(timeline);
      const userBubbleRect = userBubble.getBoundingClientRect();
      const assistantProcessRect = assistantProcess.getBoundingClientRect();
      const assistantAnswerRect = assistantAnswer.getBoundingClientRect();
      const userRect = user.getBoundingClientRect();
      const assistantRect = assistant.getBoundingClientRect();
      return {
        alignContent: timelineStyle.alignContent,
        gridAutoRows: timelineStyle.gridAutoRows,
        processAnswerGap: assistantAnswerRect.top - assistantProcessRect.bottom,
        userExtraHeight: userRect.height - userBubbleRect.height,
        assistantExtraHeight: assistantRect.height - assistantProcessRect.height - assistantAnswerRect.height,
      };
    });

    expect(metrics).not.toBeNull();
    expect(metrics?.alignContent).toBe("start");
    expect(metrics?.gridAutoRows).toBe("max-content");
    expect(metrics?.processAnswerGap ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(16);
    expect(metrics?.userExtraHeight ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(40);
    expect(metrics?.assistantExtraHeight ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(56);
  });
});
