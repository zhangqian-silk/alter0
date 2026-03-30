import { expect, test } from "@playwright/test";
import {
  expectComposerFocusedValue,
  expectComposerReady,
  expectComposerState,
  expectComposerValue,
} from "./helpers/asserts/composer";
import { commitIMEInput, startIMEInput } from "./helpers/interactions/ime";
import { openTerminalRoute } from "./helpers/flows/routes";
import { selectTerminalSession } from "./helpers/flows/terminal-session";
import { waitForTerminalPoll, waitForTerminalPollAndRepaint, waitForTerminalRepaint } from "./helpers/flows/terminal-runtime";
import { bindTerminalClient, closeTrackedTerminalSessions, createTerminalClientID, createTerminalSession, seedTerminalSessions } from "./helpers/flows/terminal-session";
import { createTerminalPage } from "./helpers/pages/terminal";
import {
  openInterruptedTerminalWorkspace,
  openReadyTerminalWorkspace,
  openTerminalWorkspace,
  openTerminalWorkspaceWithSessions,
} from "./helpers/scenarios/terminal";
import { terminalCommandPreview, terminalSessionWorkspace } from "./helpers/support/terminal-env";

test.describe("Terminal route", () => {
  test.beforeEach(async ({ request }) => {
    await closeTrackedTerminalSessions(request);
  });

  test.afterEach(async ({ request }) => {
    await closeTrackedTerminalSessions(request);
  });

  test("creates a terminal session from the page", async ({ page }) => {
    const { terminalPage } = await openTerminalWorkspace(page, { scope: "create" });

    await terminalPage.createButton().click();

    const sessionCard = terminalPage.sessionList().itemAt(0);
    await expect(sessionCard).toBeVisible();
    await expectComposerReady(terminalPage.composer());
    await expect(terminalPage.workspace()).toContainText("Running");
  });

  test("keeps terminal owner identity stable after sessionStorage is cleared and the page reloads", async ({ page, request }) => {
    const clientID = createTerminalClientID("owner-persist");
    const session = await createTerminalSession(request, clientID);

    await bindTerminalClient(page, clientID);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-status", "running");
    await expect(terminalPage.workspace()).toContainText(session.id);
    await expect.poll(async () => {
      return await page.evaluate(() => window.localStorage.getItem("alter0.web.terminal.client.v1"));
    }).toBe(clientID);

    await page.evaluate(() => {
      window.sessionStorage.removeItem("alter0.web.terminal.client.v1");
    });
    await page.reload();

    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-status", "running");
    await expect(terminalPage.workspace()).toContainText(session.id);
    await expect(terminalPage.workspace()).not.toContainText("Codex runtime exited. Send a new input to recover this session.");
    await expect.poll(async () => {
      return await page.evaluate(() => window.localStorage.getItem("alter0.web.terminal.client.v1"));
    }).toBe(clientID);
  });

  test("keeps the terminal composer anchored to the bottom bar", async ({ page, request }) => {
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "layout" });

    const chatScreen = terminalPage.chatScreen();
    const composerForm = terminalPage.composer().form();
    const submitButton = terminalPage.composer().submitButton();

    const chatBox = await chatScreen.boundingBox();
    const formBox = await composerForm.boundingBox();
    const buttonBox = await submitButton.boundingBox();

    expect(chatBox).not.toBeNull();
    expect(formBox).not.toBeNull();
    expect(buttonBox).not.toBeNull();
    expect(formBox?.height ?? 0).toBeLessThan(96);
    expect(buttonBox?.height ?? 0).toBeLessThan(56);
    expect((chatBox?.height ?? 0)).toBeGreaterThan((formBox?.height ?? 0) * 3);
  });

  test("keeps the mobile terminal workspace compact with an inline submit button", async ({ page, request }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "mobile-layout" });

    const routeHead = page.locator(".route-view.terminal-route > .route-head");
    const workspaceHead = page.locator(".terminal-workspace-head");
    const workspaceSubcopy = page.locator(".terminal-workspace-subcopy");
    const mobileNewChat = page.locator("#mobileNewChatButton");
    const headerSessionToggle = page.locator("#sessionToggle");
    const workspaceRow = page.locator(".terminal-workspace-row");
    const composerForm = terminalPage.composer().form();
    const composerInput = terminalPage.composer().input();
    const submitButton = terminalPage.composer().submitButton();

    await expect(routeHead).toBeHidden();
    await expect(workspaceSubcopy).toBeHidden();
    await expect(mobileNewChat).toBeVisible();
    await expect(mobileNewChat).toHaveText("New");
    await expect(headerSessionToggle).toBeVisible();
    await expect(headerSessionToggle).toHaveText("Sessions");

    const headBox = await workspaceHead.boundingBox();
    const rowBox = await workspaceRow.boundingBox();
    const formBox = await composerForm.boundingBox();
    const inputBox = await composerInput.boundingBox();
    const buttonBox = await submitButton.boundingBox();

    expect(headBox).not.toBeNull();
    expect(rowBox).not.toBeNull();
    expect(formBox).not.toBeNull();
    expect(inputBox).not.toBeNull();
    expect(buttonBox).not.toBeNull();
    expect(headBox?.height ?? 0).toBeLessThan(150);
    expect(rowBox?.height ?? 0).toBeLessThan(56);
    expect(formBox?.height ?? 0).toBeLessThan(92);
    expect(buttonBox?.width ?? 0).toBeLessThan(56);
    expect(buttonBox?.x ?? 0).toBeGreaterThan((inputBox?.x ?? 0) + ((inputBox?.width ?? 0) * 0.7));
    expect(Math.abs((buttonBox?.y ?? 0) - (inputBox?.y ?? 0))).toBeLessThan(20);
  });

  test("keeps input focus and draft across polling refresh", async ({ page, request }) => {
    const { session, terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "focus" });

    const composer = terminalPage.composer();
    const input = composer.input();

    await input.click();
    await expect(input).toBeFocused();

    await input.fill("pwd");
    await expectComposerValue(composer, "pwd");
    await expectComposerState(composer, { draft: "dirty" });

    await waitForTerminalPollAndRepaint(page, session.id);

    await expectComposerFocusedValue(composer, "pwd");

    await input.type(" -Path .");
    await expectComposerValue(composer, "pwd -Path .");
  });

  test("keeps the same terminal input node during mobile polling while focused", async ({ page, request }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const { session, terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "mobile-focus" });

    const composer = terminalPage.composer();
    const input = composer.input();
    const inputHandle = await input.elementHandle();

    if (!inputHandle) {
      throw new Error("terminal input handle missing");
    }
    await input.click();
    await expect(input).toBeFocused();
    await input.fill("pwd");

    try {
      await waitForTerminalPoll(page, session.id);

      await expect.poll(async () => {
        return await page.evaluate((node) => {
          const current = document.querySelector('[data-composer-input="terminal-runtime"]');
          return Boolean(node && node.isConnected && current === node && document.activeElement === current);
        }, inputHandle);
      }).toBe(true);

      await expectComposerFocusedValue(composer, "pwd");
    } finally {
      await inputHandle.dispose();
    }
  });

  test("keeps the terminal chat scroller node stable while polling during scroll", async ({ page, request }) => {
    const clientID = createTerminalClientID("scroll-stability");
    const session = await createTerminalSession(request, clientID);
    const now = Date.now();
    const entries = Array.from({ length: 80 }, (_, index) => ({
      id: `entry-${index + 1}`,
      role: "output",
      text: `line ${index + 1} ${"terminal output ".repeat(4).trim()}`,
      at: now - ((80 - index) * 1000),
      kind: "stdout",
      stream: "stdout",
      cursor: index + 1,
    }));

    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [{
      ...session,
      title: session.id,
      status: "running",
      last_output_at: now,
      updated_at: now,
      entry_cursor: entries.length,
      disconnected_notice: false,
      entries,
      chat_scroll_top: 0,
      chat_bottom_offset: 0,
      chat_has_unread_output: false,
      chat_last_seen_output_at: now,
      chat_stick_to_bottom: false,
    }]);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    const chatScreen = terminalPage.chatScreen();
    await expect(chatScreen).toBeVisible();
    await expect.poll(async () => {
      return await chatScreen.evaluate((node) => node.scrollHeight > node.clientHeight + 32);
    }).toBe(true);
    await waitForTerminalPoll(page, session.id);

    await chatScreen.evaluate((node) => {
      node.scrollTop = Math.max(0, (node.scrollHeight - node.clientHeight) * 0.45);
    });
    const beforeScrollTop = await chatScreen.evaluate((node) => Math.round(node.scrollTop));
    const chatHandle = await chatScreen.elementHandle();

    if (!chatHandle) {
      throw new Error("terminal chat handle missing");
    }

    try {
      await waitForTerminalPoll(page, session.id);
      await expect.poll(async () => {
        return await page.evaluate((node) => {
          const current = document.querySelector("[data-terminal-chat-screen]");
          return Boolean(node && node.isConnected && current === node);
        }, chatHandle);
      }).toBe(true);

      const afterScrollTop = await chatScreen.evaluate((node) => Math.round(node.scrollTop));
      expect(afterScrollTop).toBeGreaterThan(0);
      expect(Math.abs(afterScrollTop - beforeScrollTop)).toBeLessThan(8);
    } finally {
      await chatHandle.dispose();
    }
  });

  test("keeps long terminal output constrained within the chat frame", async ({ page }) => {
    const clientID = createTerminalClientID("overflow-guard");
    const now = Date.now();
    const longToken = "terminal-output-overflow-check-".repeat(40);
    const longCodeLine = `const payload = "${"x".repeat(720)}";`;

    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [{
      id: "terminal-overflow-check",
      title: "terminal-overflow-check",
      terminal_session_id: "terminal-overflow-check",
      status: "interrupted",
      shell: "codex exec",
      working_dir: terminalSessionWorkspace("terminal-overflow-check"),
      created_at: now - 10_000,
      updated_at: now,
      last_output_at: now,
      entry_cursor: 0,
      disconnected_notice: false,
      entries: [],
      turns: [{
        id: "turn-overflow-check",
        prompt: "show overflow case",
        status: "completed",
        started_at: now - 3_000,
        finished_at: now - 1_000,
        duration_ms: 2_000,
        final_output: [
          longToken,
          "",
          "```ts",
          longCodeLine,
          "```",
        ].join("\n"),
        steps: [],
      }],
      process_collapsed: {},
      output_collapsed: {},
      expanded_steps: {},
      step_details: {},
      step_errors: {},
      step_loading: {},
      step_search: {},
      meta_expanded: false,
      chat_scroll_top: 0,
      chat_bottom_offset: 0,
      chat_has_unread_output: false,
      chat_last_seen_output_at: now,
      chat_stick_to_bottom: true,
    }]);

    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    await expect(terminalPage.workspace()).toBeVisible();
    await expect(terminalPage.finalOutputs().first()).toBeVisible();

    const widthWithinFrame = async (selector: string) => {
      return await page.locator(selector).first().evaluate((node) => {
        return node.scrollWidth <= node.clientWidth + 2;
      });
    };

    await expect.poll(async () => await widthWithinFrame("[data-terminal-chat-screen]")).toBe(true);
    await expect.poll(async () => await widthWithinFrame("[data-terminal-turn]")).toBe(true);
    await expect.poll(async () => await widthWithinFrame(".terminal-final-output")).toBe(true);
  });

  test("toggles the mobile session sheet without leaving it expanded after selection", async ({ page, request }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const { terminalPage } = await openTerminalWorkspaceWithSessions(page, request, { scope: "mobile-sheet", count: 2 });

    await expect(terminalPage.sessionPane()).not.toHaveClass(/is-open/);
    await expect(terminalPage.sessionPane()).toBeHidden();
    await terminalPage.sessionPaneToggle().click();
    await expect(terminalPage.sessionPane()).toHaveClass(/is-open/);
    await expect(terminalPage.sessionPane()).toBeVisible();

    const secondSession = terminalPage.sessionList().itemAt(1);
    await secondSession.click();
    await expect(terminalPage.sessionPane()).not.toHaveClass(/is-open/);
    await expect(terminalPage.sessionPane()).toBeHidden();
  });

  test("deletes a historical session directly from the session list", async ({ page, request }) => {
    const { sessions, terminalPage } = await openTerminalWorkspaceWithSessions(page, request, { scope: "list-delete-history", count: 2 });

    const activeSessionID = await terminalPage.workspace().getAttribute("data-terminal-session-id");
    expect(activeSessionID).toBeTruthy();
    const historicalSession = sessions.find((session) => session.id !== activeSessionID);
    expect(historicalSession).toBeTruthy();

    page.once("dialog", (dialog) => dialog.accept());
    await terminalPage.sessionDeleteButton(String(historicalSession?.id || "")).click();

    await expect(terminalPage.sessionList().items()).toHaveCount(1);
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-session-id", String(activeSessionID));
    await expect(terminalPage.sessionDeleteButton(String(historicalSession?.id || ""))).toHaveCount(0);
  });

  test("switches to the next session when deleting the active session from the session list", async ({ page, request }) => {
    const { sessions, terminalPage } = await openTerminalWorkspaceWithSessions(page, request, { scope: "list-delete-active", count: 2 });

    const activeSessionID = await terminalPage.workspace().getAttribute("data-terminal-session-id");
    expect(activeSessionID).toBeTruthy();
    const nextSession = sessions.find((session) => session.id !== activeSessionID);
    expect(nextSession).toBeTruthy();

    page.once("dialog", (dialog) => dialog.accept());
    await terminalPage.sessionDeleteButton(String(activeSessionID)).click();

    await expect(terminalPage.sessionList().items()).toHaveCount(1);
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-session-id", String(nextSession?.id || ""));
    await expect(terminalPage.sessionDeleteButton(String(activeSessionID))).toHaveCount(0);
  });

  test("keeps the mobile session sheet closed and interactive after deleting the active session", async ({ page, request }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const { terminalPage } = await openTerminalWorkspaceWithSessions(page, request, { scope: "mobile-delete-sheet", count: 2 });

    const previousSessionID = await terminalPage.workspace().getAttribute("data-terminal-session-id");
    expect(previousSessionID).toBeTruthy();
    await expect(terminalPage.sessionList().items()).toHaveCount(2);

    page.once("dialog", (dialog) => dialog.accept());
    await terminalPage.workspace().locator("[data-terminal-delete]").click();

    await expect.poll(async () => await terminalPage.workspace().getAttribute("data-terminal-session-id")).not.toBe(previousSessionID);
    await expect(terminalPage.sessionList().items()).toHaveCount(1);
    await expect(terminalPage.sessionPane()).not.toHaveClass(/is-open/);
    await expect(terminalPage.sessionPane()).toBeHidden();

    await terminalPage.sessionPaneToggle().click();
    await expect(terminalPage.sessionPane()).toHaveClass(/is-open/);
    await expect(terminalPage.sessionPane()).toBeVisible();
    await terminalPage.sessionPaneClose().click();
    await expect(terminalPage.sessionPane()).not.toHaveClass(/is-open/);
    await expect(terminalPage.sessionPane()).toBeHidden();
  });

  test("keeps the terminal workspace filled to the bottom after deleting the active session on mobile", async ({ page, request }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const { terminalPage } = await openTerminalWorkspaceWithSessions(page, request, { scope: "mobile-delete-height", count: 2 });

    page.once("dialog", (dialog) => dialog.accept());
    await terminalPage.workspace().locator("[data-terminal-delete]").click();

    await expect.poll(async () => await terminalPage.sessionList().items().count()).toBe(1);
    await expect(terminalPage.sessionPane()).toBeHidden();

    const layoutMetrics = await page.evaluate(() => {
      const route = document.querySelector(".route-view.terminal-route");
      const routeBody = document.querySelector(".route-body.terminal-route-body");
      const terminalView = document.querySelector("[data-terminal-view]");
      const terminalWorkspaceShell = document.querySelector(".terminal-workspace");
      const workspace = document.querySelector("[data-terminal-workspace]");
      if (!(route instanceof HTMLElement) || !(workspace instanceof HTMLElement)) {
        return null;
      }
      const rect = (node: Element | null) => {
        if (!(node instanceof HTMLElement)) {
          return null;
        }
        const box = node.getBoundingClientRect();
        return {
          top: Math.round(box.top),
          bottom: Math.round(box.bottom),
          height: Math.round(box.height),
        };
      };
      return {
        gap: Math.round(route.getBoundingClientRect().bottom - workspace.getBoundingClientRect().bottom),
        windowScrollY: Math.round(window.scrollY || 0),
        routeScrollTop: Math.round(route.scrollTop || 0),
        routeBodyScrollTop: routeBody instanceof HTMLElement ? Math.round(routeBody.scrollTop || 0) : null,
        terminalViewScrollTop: terminalView instanceof HTMLElement ? Math.round(terminalView.scrollTop || 0) : null,
        route: rect(route),
        routeBody: rect(routeBody),
        terminalView: rect(terminalView),
        terminalWorkspaceShell: rect(terminalWorkspaceShell),
        workspace: rect(workspace),
      };
    });

    expect(layoutMetrics).not.toBeNull();
    expect(layoutMetrics?.gap ?? 0).toBeLessThanOrEqual(20);
  });

  test("keeps IME composition input across polling refresh", async ({ page, request }) => {
    const { session, terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "ime" });

    const composer = terminalPage.composer();
    const input = composer.input();
    await startIMEInput(input);
    await expectComposerState(composer, { composing: true, draft: "dirty" });

    await waitForTerminalPoll(page, session.id);

    await expectComposerFocusedValue(composer, "ni");

    await commitIMEInput(input, "浣?");

    await expectComposerValue(composer, "浣?");
    await expectComposerState(composer, { composing: false, draft: "dirty" });
  });

  test("defers terminal repaint after mobile IME commit until blur", async ({ page, request }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const { session, terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "mobile-ime" });

    const composer = terminalPage.composer();
    const input = composer.input();
    const inputHandle = await input.elementHandle();

    if (!inputHandle) {
      throw new Error("terminal input handle missing");
    }

    try {
      await startIMEInput(input);
      await expectComposerState(composer, { composing: true, draft: "dirty" });

      await waitForTerminalPoll(page, session.id);
      await commitIMEInput(input, "terminal-ime-mobile");

      await expect.poll(async () => {
        return await page.evaluate((node) => {
          const current = document.querySelector('[data-composer-input="terminal-runtime"]');
          return Boolean(
            node &&
            node.isConnected &&
            current === node &&
            document.activeElement === current &&
            (current instanceof HTMLInputElement ? current.value : "") === "terminal-ime-mobile"
          );
        }, inputHandle);
      }).toBe(true);

      await input.blur();
      await waitForTerminalRepaint(page, 5000);
      await expectComposerValue(composer, "terminal-ime-mobile");
    } finally {
      await inputHandle.dispose();
    }
  });

  test("keeps input available after close for same-session recovery", async ({ page, request }) => {
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "close" });

    const composer = terminalPage.composer();
    const input = composer.input();
    const submit = composer.submitButton();
    await input.fill("Reply with exactly: alter0-playwright");
    await submit.click();
    await expectComposerValue(composer, "");

    await expect(terminalPage.chatScreen()).toContainText("alter0-playwright");

    const closeButton = terminalPage.closeButton();
    await expect(closeButton).toBeEnabled();
    await closeButton.click();

    await expect(terminalPage.workspace()).toContainText("Exited");
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-live", "false");
    await expect(terminalPage.workspace()).toContainText("Codex session exited. Send a new input to continue in this session.");
    await expect(input).toBeEnabled();
    await expect(submit).toBeEnabled();
    await expectComposerState(composer, { disabled: false });
  });

  test("renders process and plain output with lazy-loaded step details", async ({ page, request }) => {
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "structure" });
    const composer = terminalPage.composer();
    let stepDetailRequests = 0;
    page.on("request", (requestEvent) => {
      if (requestEvent.method() === "GET" && requestEvent.url().includes("/steps/")) {
        stepDetailRequests += 1;
      }
    });

    await composer.input().fill("Reply with exactly: alter0-process-structure");
    await composer.submitButton().click();

    await expect(terminalPage.finalOutputs().last()).toContainText("alter0-process-structure");
    const processToggle = terminalPage.processToggle("turn-1");
    await expect(processToggle).toBeVisible();
    await expect.poll(() => stepDetailRequests).toBe(0);

    if ((await processToggle.getAttribute("aria-expanded")) !== "true") {
      await processToggle.click();
    }
    await expect(processToggle).toHaveAttribute("aria-expanded", "true");
    const visibleStepToggles = terminalPage.turnCard("turn-1").locator('.terminal-process-body:not([hidden]) [data-terminal-step-toggle]');
    await expect(visibleStepToggles).toHaveCount(2);
    await expect(visibleStepToggles.nth(0)).toContainText("Inspect workspace");
    await expect(visibleStepToggles.nth(1)).toContainText(terminalCommandPreview);

    const stepDetailResponse = page.waitForResponse(
      (response) => response.request().method() === "GET" && response.ok() && response.url().includes("/steps/"),
    );
    await Promise.all([
      stepDetailResponse,
      visibleStepToggles.nth(1).click(),
    ]);

    await expect.poll(() => stepDetailRequests).toBe(1);
    const currentSessionID = await terminalPage.workspace().getAttribute("data-terminal-session-id");
    expect(currentSessionID).toBeTruthy();
    await expect(terminalPage.workspace()).not.toContainText("Path");
    await terminalPage.metaToggle().click();
    await expect(terminalPage.workspace()).toContainText("Path");
    await expect(terminalPage.workspace()).toContainText(new RegExp(String(currentSessionID).replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "i"));
  });

  test("auto-collapses process after output arrives and preserves manual reopen state", async ({ page, request }) => {
    const clientID = createTerminalClientID("process-auto-collapse");
    const session = await createTerminalSession(request, clientID);
    const prompt = "Reply with exactly: alter0-process-auto-collapse";
    const now = Date.now();

    const inputResponse = await request.post(`/api/terminal/sessions/${encodeURIComponent(session.id)}/input`, {
      headers: {
        "X-Alter0-Terminal-Client": clientID,
      },
      data: {
        input: prompt,
      },
    });
    expect(inputResponse.ok()).toBeTruthy();

    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [
      {
        id: session.id,
        title: session.id,
        terminal_session_id: session.id,
        status: "running",
        shell: "codex exec",
        working_dir: terminalSessionWorkspace(session.id),
        created_at: now - 5000,
        updated_at: now - 4000,
        last_output_at: 0,
        process_collapsed: {
          "turn-1": false,
        },
        output_collapsed: {},
        expanded_steps: {},
        step_details: {},
        step_loading: {},
        step_errors: {},
        step_search: {},
        turns: [
          {
            id: "turn-1",
            prompt,
            status: "running",
            started_at: now - 4500,
            finished_at: 0,
            duration_ms: 0,
            final_output: "",
            steps: [
              {
                id: "step-1",
                title: "Waiting for Codex",
                preview: "running",
                status: "running",
                started_at: now - 4500,
                finished_at: 0,
                duration_ms: 0,
              },
            ],
          },
        ],
        entries: [],
      },
    ]);

    await openTerminalRoute(page);
    await waitForTerminalPollAndRepaint(page, session.id);

    const terminalPage = createTerminalPage(page);
    const processToggle = terminalPage.processToggle("turn-1");

    await expect(terminalPage.finalOutputs().last()).toContainText("alter0-process-auto-collapse");
    await expect(processToggle).toHaveAttribute("aria-expanded", "false");

    await processToggle.click();
    await expect(processToggle).toHaveAttribute("aria-expanded", "true");

    await page.reload();
    await waitForTerminalPollAndRepaint(page, session.id);
    await expect(terminalPage.processToggle("turn-1")).toHaveAttribute("aria-expanded", "true");
  });

  test("renders markdown links in plain terminal output without collapse chrome", async ({ page }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const clientID = createTerminalClientID("collapsed-output");
    const now = Date.now();
    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [
      {
        id: "terminal-rendered-output",
        title: "检查终端输出渲染",
        terminal_session_id: "terminal-rendered-output",
        status: "exited",
        shell: "codex exec",
        working_dir: terminalSessionWorkspace("terminal-rendered-output"),
        created_at: now - 4000,
        updated_at: now - 1000,
        last_output_at: now - 1000,
        turns: [
          {
            id: "turn-rendered",
            prompt: "检查 markdown 和折叠输出",
            status: "completed",
            started_at: now - 3200,
            finished_at: now - 1200,
            duration_ms: 2000,
            final_output: `${Array.from({ length: 18 }, (_, index) => `line ${index + 1}`).join("\n")}\n\n- [requirements.md](/srv/alter0/app/alter0/docs/requirements.md)`,
            steps: [],
          },
        ],
      },
    ]);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    await expect(terminalPage.sessionPane()).not.toHaveClass(/is-open/);
    const outputLink = terminalPage.turnCard("turn-rendered").locator(".terminal-final-rendered a");

    await expect(terminalPage.turnCard("turn-rendered").locator("[data-terminal-output-toggle]")).toHaveCount(0);
    await expect(terminalPage.turnCard("turn-rendered").locator(".terminal-final-head")).toHaveCount(0);
    await expect(terminalPage.finalOutputs().last()).toContainText("line 18");
    await expect(outputLink).toHaveText("requirements.md");
    await expect(outputLink).toHaveAttribute("href", "/srv/alter0/app/alter0/docs/requirements.md");
  });

  test("keeps terminal scroll position when user leaves bottom", async ({ page, request }) => {
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "sticky-scroll" });
    const composer = terminalPage.composer();
    const chatScreen = terminalPage.chatScreen();

    await composer.input().fill("output 120 lines");
    await composer.submitButton().click();
    await expect(terminalPage.finalOutputs().last()).toContainText("line 120");

    await expect.poll(async () => chatScreen.evaluate((node) => node.scrollHeight - node.clientHeight)).toBeGreaterThan(120);

    await chatScreen.evaluate((node) => {
      node.scrollTop = 0;
    });
    await expect.poll(async () => chatScreen.evaluate((node) => node.scrollTop)).toBe(0);

    await composer.input().fill("Reply with exactly: sticky-scroll-result");
    await composer.submitButton().click();
    await expect(terminalPage.finalOutputs().last()).toContainText("sticky-scroll-result");

    await expect.poll(async () => chatScreen.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop)).toBeGreaterThan(64);
  });

  test("shows jump-to-bottom button after manual scroll and returns to latest output", async ({ page, request }) => {
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "jump-bottom" });
    const composer = terminalPage.composer();
    const chatScreen = terminalPage.chatScreen();
    const jumpBottomButton = terminalPage.jumpBottomButton();

    await composer.input().fill("output 120 lines");
    await composer.submitButton().click();
    await expect(terminalPage.finalOutputs().last()).toContainText("line 120");

    await chatScreen.evaluate((node) => {
      node.scrollTop = 0;
    });

    await expect(jumpBottomButton).toHaveClass(/is-visible/);
    await jumpBottomButton.click();

    await expect.poll(async () => chatScreen.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop)).toBeLessThan(12);
    await expect(jumpBottomButton).not.toHaveClass(/is-visible/);
  });

  test("shows jump-to-top button after reading deep output and returns to the beginning", async ({ page, request }) => {
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "jump-top" });
    const composer = terminalPage.composer();
    const chatScreen = terminalPage.chatScreen();
    const jumpTopButton = terminalPage.jumpTopButton();

    await composer.input().fill("output 120 lines");
    await composer.submitButton().click();
    await expect(terminalPage.finalOutputs().last()).toContainText("line 120");

    await chatScreen.evaluate((node) => {
      node.scrollTop = Math.max(node.scrollHeight - node.clientHeight, 0);
      node.dispatchEvent(new Event("scroll"));
    });

    await expect(jumpTopButton).toHaveClass(/is-visible/);
    await jumpTopButton.click();

    await expect.poll(() => chatScreen.evaluate((node) => Math.round(node.scrollTop))).toBeLessThanOrEqual(8);
    await expect(jumpTopButton).not.toHaveClass(/is-visible/);
  });

  test("shows previous and next turn buttons and jumps between turn cards", async ({ page }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const clientID = createTerminalClientID("turn-navigation");
    const now = Date.now();
    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [
      {
        id: "terminal-turn-navigation",
        title: "验证上下条导航",
        terminal_session_id: "terminal-turn-navigation",
        status: "exited",
        shell: "codex exec",
        working_dir: terminalSessionWorkspace("terminal-turn-navigation"),
        created_at: now - 5000,
        updated_at: now - 1000,
        last_output_at: now - 1000,
        process_collapsed: {
          "turn-2": true
        },
        output_collapsed: {},
        expanded_steps: {},
        step_details: {},
        step_loading: {},
        step_errors: {},
        step_search: {},
        turns: Array.from({ length: 3 }, (_, index) => ({
          id: `turn-${index + 1}`,
          prompt: `output block ${index + 1}`,
          status: "completed",
          started_at: now - 4200 + index * 200,
          finished_at: now - 3200 + index * 200,
          duration_ms: 1000,
          final_output: Array.from({ length: 48 }, (_, lineIndex) => `turn ${index + 1} line ${lineIndex + 1}`).join("\n"),
          steps: index === 1 ? Array.from({ length: 8 }, (_value, stepIndex) => ({
            id: `turn-2-step-${stepIndex + 1}`,
            title: `Shell step ${stepIndex + 1}`,
            preview: `pwsh -NoProfile -Command "Get-ChildItem step-${stepIndex + 1}"`,
            status: "completed",
            duration_ms: 1000 + stepIndex * 10,
            started_at: now - 3800 + stepIndex * 20,
            finished_at: now - 3400 + stepIndex * 20,
          })) : [],
        })),
      },
    ]);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    const chatScreen = terminalPage.chatScreen();
    const jumpPrevButton = terminalPage.jumpPrevButton();
    const jumpNextButton = terminalPage.jumpNextButton();
    const alignTurnToTop = async (turnID: string, offset = 18) => {
      await chatScreen.evaluate((node, payload) => {
        const turn = node.querySelector(`[data-terminal-turn="${payload?.turnID || ""}"]`);
        if (!(turn instanceof HTMLElement)) {
          return;
        }
        const nodeRect = node.getBoundingClientRect();
        const turnRect = turn.getBoundingClientRect();
        node.scrollTop = Math.max(node.scrollTop + turnRect.top - nodeRect.top - Number(payload?.offset || 0), 0);
        node.dispatchEvent(new Event("scroll"));
      }, { turnID, offset });
    };

    await expect(terminalPage.turnCard("turn-3")).toContainText("turn 3 line 48");
    await alignTurnToTop("turn-2");
    await expect(jumpPrevButton).toHaveClass(/is-visible/);
    await expect(jumpPrevButton).toHaveAttribute("data-terminal-jump-target", "turn-1");
    await expect(jumpNextButton).toHaveClass(/is-visible/);
    await expect(jumpNextButton).toHaveAttribute("data-terminal-jump-target", "turn-3");

    await terminalPage.processToggle("turn-2").click();
    await expect(jumpPrevButton).toHaveAttribute("data-terminal-jump-target", "turn-1");
    await expect(jumpNextButton).toHaveAttribute("data-terminal-jump-target", "turn-3");

    await jumpNextButton.click();
    await expect.poll(() => chatScreen.evaluate((node) => {
      const turn = node.querySelector('[data-terminal-turn="turn-3"]');
      if (!(turn instanceof HTMLElement)) {
        return null;
      }
      return Math.round(turn.getBoundingClientRect().top - node.getBoundingClientRect().top);
    })).toBeLessThanOrEqual(24);

    await expect(jumpPrevButton).toHaveClass(/is-visible/);
    await expect(jumpPrevButton).toHaveAttribute("data-terminal-jump-target", "turn-2");
    await jumpPrevButton.click();
    await expect.poll(() => chatScreen.evaluate((node) => {
      const turn = node.querySelector('[data-terminal-turn="turn-2"]');
      if (!(turn instanceof HTMLElement)) {
        return null;
      }
      return Math.round(turn.getBoundingClientRect().top - node.getBoundingClientRect().top);
    })).toBeLessThanOrEqual(24);
  });

  test("keeps prompt rows in normal document flow while reading long terminal output", async ({ page }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const clientID = createTerminalClientID("prompt-flow");
    const now = Date.now();
    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [
      {
        id: "terminal-prompt-flow",
        title: "验证用户消息保持自然流",
        terminal_session_id: "terminal-prompt-flow",
        status: "exited",
        shell: "codex exec",
        working_dir: terminalSessionWorkspace("terminal-prompt-flow"),
        created_at: now - 4000,
        updated_at: now - 1000,
        last_output_at: now - 1000,
        process_collapsed: {},
        output_collapsed: { "turn-1": false },
        expanded_steps: {},
        step_details: {},
        step_loading: {},
        step_errors: {},
        step_search: {},
        turns: [
          {
            id: "turn-1",
            prompt: "output 120 lines",
            status: "completed",
            started_at: now - 3200,
            finished_at: now - 1200,
            duration_ms: 2000,
            final_output: Array.from({ length: 120 }, (_, index) => `line ${index + 1}`).join("\n"),
            steps: [],
          },
        ],
      },
    ]);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    const prompt = terminalPage.turnCard("turn-1").locator(".terminal-turn-prompt");
    await expect(terminalPage.turnCard("turn-1")).toContainText("line 120");
    await expect.poll(() => prompt.evaluate((node) => window.getComputedStyle(node).position)).toBe("static");
    await expect.poll(() => terminalPage.chatScreen().evaluate((node) => {
      const promptNode = node.querySelector(".terminal-turn-prompt .terminal-log-main");
      if (!(promptNode instanceof HTMLElement)) {
        return false;
      }
      const promptRect = promptNode.getBoundingClientRect();
      const nodeRect = node.getBoundingClientRect();
      const width = Math.round(promptRect.width);
      const maxWidth = Math.round(nodeRect.width * 0.8);
      const rightGap = Math.round(nodeRect.right - promptRect.right);
      const leftGap = Math.round(promptRect.left - nodeRect.left);
      return width <= maxWidth + 2 && rightGap <= leftGap;
    })).toBe(true);

    await terminalPage.chatScreen().evaluate((node) => {
      node.scrollTop = Math.max((node.scrollHeight - node.clientHeight) / 2, 0);
      node.dispatchEvent(new Event("scroll"));
    });

    await expect(page.locator(".is-terminal-sticky-active")).toHaveCount(0);
    await expect.poll(() => terminalPage.chatScreen().evaluate((node) => {
      const promptNode = node.querySelector(".terminal-turn-prompt");
      if (!(promptNode instanceof HTMLElement)) {
        return null;
      }
      return Math.round(promptNode.getBoundingClientRect().bottom - node.getBoundingClientRect().top);
    })).toBeLessThanOrEqual(0);
  });

  test("keeps process and output content in document flow on mobile", async ({ page }) => {
    await page.setViewportSize({ width: 430, height: 640 });
    const clientID = createTerminalClientID("section-flow");
    const now = Date.now();
    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [
      {
        id: "terminal-section-flow",
        title: "检查区块头部保持自然流",
        terminal_session_id: "terminal-section-flow",
        status: "exited",
        shell: "codex exec",
        working_dir: terminalSessionWorkspace("terminal-section-flow"),
        created_at: now - 4000,
        updated_at: now - 1000,
        last_output_at: now - 1000,
        process_collapsed: {},
        output_collapsed: { "turn-sticky": false },
        expanded_steps: {},
        step_details: {},
        step_loading: {},
        step_errors: {},
        step_search: {},
        turns: [
          {
            id: "turn-sticky",
            prompt: "输出一段很长的过程和最终结果，便于滚动验证吸顶头部",
            status: "completed",
            started_at: now - 3200,
            finished_at: now - 1200,
            duration_ms: 2000,
            final_output: Array.from({ length: 120 }, (_, index) => `line ${index + 1}`).join("\n"),
            steps: Array.from({ length: 12 }, (_, index) => ({
              id: `step-${index + 1}`,
              title: `Step ${index + 1}`,
              preview: `preview ${index + 1}`,
              status: "completed",
              duration_ms: 1000 + index * 10,
              started_at: now - 3000 + index * 20,
              finished_at: now - 2000 + index * 20,
            })),
          },
        ],
      },
    ]);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    const turnID = "turn-sticky";
    const outputNode = page.locator(`[data-terminal-final-output="${turnID}"]`);
    const processToggle = terminalPage.processToggle(turnID);
    const scrollSelectorIntoView = async (selector: string, offset = 18) => {
      await terminalPage.chatScreen().evaluate((node, payload) => {
        const target = node.querySelector(payload?.selector || "");
        if (!(target instanceof HTMLElement)) {
          return;
        }
        const nodeRect = node.getBoundingClientRect();
        const targetRect = target.getBoundingClientRect();
        node.scrollTop += Math.max(targetRect.top - nodeRect.top - Number(payload?.offset || 0), 0);
        node.dispatchEvent(new Event("scroll"));
      }, { selector, offset });
    };

    await expect(terminalPage.turnCard(turnID)).toContainText("line 120");
    await expect.poll(() => processToggle.evaluate((node) => {
      const title = node.querySelector(".terminal-process-title");
      const summary = node.querySelector(".terminal-process-summary");
      if (!(title instanceof HTMLElement) || !(summary instanceof HTMLElement)) {
        return null;
      }
      return Math.abs(title.getBoundingClientRect().top - summary.getBoundingClientRect().top);
    })).toBeLessThanOrEqual(2);
    await expect.poll(() => processToggle.evaluate((node) => window.getComputedStyle(node).position)).toBe("relative");
    await expect.poll(() => outputNode.evaluate((node) => window.getComputedStyle(node).position)).toBe("static");
    await scrollSelectorIntoView(`[data-terminal-process-toggle="${turnID}"]`);
    await expect(page.locator(".is-terminal-sticky-active")).toHaveCount(0);
    await scrollSelectorIntoView(`[data-terminal-final-output="${turnID}"]`);
    await expect(page.locator(".is-terminal-sticky-active")).toHaveCount(0);
    await expect(terminalPage.turnCard(turnID).locator("[data-terminal-output-toggle]")).toHaveCount(0);
  });

  test("keeps plain terminal output unobstructed while scrolled on mobile", async ({ page, request }) => {
    await page.setViewportSize({ width: 430, height: 932 });
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "scrolled-output-toggle" });
    const composer = terminalPage.composer();

    await composer.input().fill("output 120 lines");
    await composer.submitButton().click();
    await expect(terminalPage.finalOutputs().last()).toContainText("line 120");

    await terminalPage.chatScreen().evaluate((node) => {
      const output = node.querySelector('[data-terminal-final-output="turn-1"]');
      if (!(output instanceof HTMLElement)) {
        return;
      }
      const nodeRect = node.getBoundingClientRect();
      const outputRect = output.getBoundingClientRect();
      node.scrollTop += Math.max(outputRect.top - nodeRect.top - 18, 0);
      node.dispatchEvent(new Event("scroll"));
    });

    await expect(page.locator(".is-terminal-sticky-active")).toHaveCount(0);
    await expect(terminalPage.turnCard("turn-1").locator("[data-terminal-output-toggle]")).toHaveCount(0);
    await expect(terminalPage.finalOutputs().last()).toContainText("line 120");
  });

  test("keeps jump-to-bottom button hidden for short offset until new output arrives", async ({ page, request }) => {
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "jump-bottom-unread" });
    const composer = terminalPage.composer();
    const chatScreen = terminalPage.chatScreen();
    const jumpBottomButton = terminalPage.jumpBottomButton();

    await composer.input().fill("output 120 lines");
    await composer.submitButton().click();
    await expect(terminalPage.finalOutputs().last()).toContainText("line 120");

    await chatScreen.evaluate((node) => {
      node.scrollTop = Math.max(node.scrollHeight - node.clientHeight - 120, 0);
    });
    await expect(jumpBottomButton).not.toHaveClass(/is-visible/);

    await composer.input().fill("Reply with exactly: unread-output");
    await composer.submitButton().click();
    await expect(terminalPage.finalOutputs().last()).toContainText("unread-output");

    await expect(jumpBottomButton).toHaveClass(/is-visible/);
    await expect(jumpBottomButton).toHaveClass(/has-unread/);
  });

  test("keeps wheel scrolling active over process header", async ({ page, request }) => {
    const { terminalPage } = await openReadyTerminalWorkspace(page, request, { scope: "wheel-scroll" });
    const composer = terminalPage.composer();
    const chatScreen = terminalPage.chatScreen();

    await composer.input().fill("output 120 lines");
    await composer.submitButton().click();
    await expect(terminalPage.finalOutputs().last()).toContainText("line 120");

    await expect.poll(async () => chatScreen.evaluate((node) => node.scrollHeight - node.clientHeight)).toBeGreaterThan(120);

    const processToggle = terminalPage.processToggles().last();
    await processToggle.hover();
    const beforeScrollTop = await chatScreen.evaluate((node) => node.scrollTop);
    await page.mouse.wheel(0, 480);
    await expect.poll(async () => chatScreen.evaluate((node) => node.scrollTop)).toBeGreaterThan(beforeScrollTop);
  });

  test("keeps drafts isolated between terminal sessions", async ({ page, request }) => {
    const { sessions, terminalPage } = await openTerminalWorkspaceWithSessions(page, request, { scope: "drafts", count: 2 });
    const [sessionA, sessionB] = sessions;

    await selectTerminalSession(page, sessionA.id);
    const composer = terminalPage.composer();
    const input = composer.input();
    await expectComposerReady(composer);
    await waitForTerminalPollAndRepaint(page, sessionA.id);
    await input.fill("draft-for-session-a");
    await expectComposerValue(composer, "draft-for-session-a");

    await selectTerminalSession(page, sessionB.id);
    await waitForTerminalPollAndRepaint(page, sessionB.id);
    await expectComposerValue(composer, "");
    await input.fill("draft-for-session-b");
    await expectComposerValue(composer, "draft-for-session-b");

    await selectTerminalSession(page, sessionA.id);
    await expectComposerValue(composer, "draft-for-session-a");

    await selectTerminalSession(page, sessionB.id);
    await expectComposerValue(composer, "draft-for-session-b");
  });

  test("keeps session order stable when selecting without new activity", async ({ page }) => {
    const clientID = createTerminalClientID("ordering");
    const now = Date.now();
    const olderSession = {
      id: "terminal-ordering-older",
      title: "terminal-ordering-older",
      terminal_session_id: "terminal-ordering-older",
      status: "interrupted",
      shell: "codex exec",
      working_dir: terminalSessionWorkspace("terminal-ordering-older"),
      created_at: now - 5_000,
      last_output_at: now - 1_000,
      updated_at: now - 500,
      entry_cursor: 0,
      disconnected_notice: true,
      entries: [
        {
          id: "terminal-ordering-older-output",
          role: "output",
          text: "older-session-output",
          at: now - 1_000,
          kind: "stdout",
          stream: "stdout",
          cursor: 1,
        },
      ],
    };
    const newerSession = {
      id: "terminal-ordering-newer",
      title: "terminal-ordering-newer",
      terminal_session_id: "terminal-ordering-newer",
      status: "interrupted",
      shell: "codex exec",
      working_dir: terminalSessionWorkspace("terminal-ordering-newer"),
      created_at: now - 4_000,
      last_output_at: now - 2_000,
      updated_at: now - 4_000,
      entry_cursor: 0,
      disconnected_notice: true,
      entries: [
        {
          id: "terminal-ordering-newer-output",
          role: "output",
          text: "newer-session-output",
          at: now - 2_000,
          kind: "stdout",
          stream: "stdout",
          cursor: 1,
        },
      ],
    };

    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [olderSession, newerSession]);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    await expect(terminalPage.sessionList().items()).toHaveCount(2);

    const beforeOrder = await Promise.all(
      [0, 1].map((index) => terminalPage.sessionList().itemAt(index).getAttribute("data-terminal-session-select"))
    );

    await selectTerminalSession(page, olderSession.id);
    await page.waitForTimeout(1500);

    const afterOrder = await Promise.all(
      [0, 1].map((index) => terminalPage.sessionList().itemAt(index).getAttribute("data-terminal-session-select"))
    );

    expect(beforeOrder).toEqual([olderSession.id, newerSession.id]);
    expect(afterOrder).toEqual(beforeOrder);
    await expect(terminalPage.sessionList().itemByValue?.(olderSession.id) || terminalPage.sessionList().itemAt(0)).toHaveClass(/active/);
    await expect(terminalPage.sessionList().itemAt(0)).toContainText("Last output");
  });

  test("preserves session list scroll position when switching sessions", async ({ page }) => {
    const clientID = createTerminalClientID("scroll");
    const now = Date.now();
    const sessions = Array.from({ length: 12 }, (_value, index) => ({
      id: `terminal-scroll-${index}`,
      title: `terminal-scroll-${index}`,
      terminal_session_id: `terminal-scroll-${index}`,
      status: "interrupted",
      shell: "codex exec",
      working_dir: terminalSessionWorkspace(`terminal-scroll-${index}`),
      created_at: now - (index + 1) * 1_000,
      last_output_at: now - (index + 1) * 1_000,
      updated_at: now - index * 500,
      entry_cursor: 1,
      disconnected_notice: true,
      entries: [
        {
          id: `terminal-scroll-${index}-output`,
          role: "output",
          text: `output-${index}`,
          at: now - (index + 1) * 1_000,
          kind: "stdout",
          stream: "stdout",
          cursor: 1,
        },
      ],
    }));

    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, sessions);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    const sessionListContainer = terminalPage.sessionListContainer();
    await expect(terminalPage.sessionList().items()).toHaveCount(12);

    await sessionListContainer.evaluate((node) => {
      node.scrollTop = node.scrollHeight;
    });
    const beforeScrollTop = await sessionListContainer.evaluate((node) => node.scrollTop);

    await selectTerminalSession(page, "terminal-scroll-11");
    const afterScrollTop = await sessionListContainer.evaluate((node) => node.scrollTop);

    expect(beforeScrollTop).toBeGreaterThan(0);
    expect(afterScrollTop).toBeGreaterThan(0);
    expect(Math.abs(afterScrollTop - beforeScrollTop)).toBeLessThan(24);
  });

  test("keeps interrupted sessions recoverable when runtime is unavailable", async ({ page }) => {
    const { terminalPage } = await openInterruptedTerminalWorkspace(page, { scope: "interrupted" });

    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-status", "interrupted");
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-live", "false");
    await expect(terminalPage.workspace()).toContainText("Interrupted");
    await expect(terminalPage.workspace()).toContainText("Codex runtime exited. Send a new input to recover this session.");
    await expect(terminalPage.composer().input()).toBeEnabled();
    await expect(terminalPage.composer().submitButton()).toBeEnabled();
    await expectComposerState(terminalPage.composer(), { disabled: false });
  });

  test("clears the previous session hint immediately when creating a new session", async ({ page }) => {
    const { terminalPage, session } = await openInterruptedTerminalWorkspace(page, { scope: "create-clears-hint" });

    await page.route("**/api/terminal/sessions", async (route) => {
      if (route.request().method() === "POST") {
        await new Promise((resolve) => setTimeout(resolve, 300));
      }
      await route.continue();
    });

    await terminalPage.createButton().click();

    let pendingSessionID = "";
    await expect.poll(async () => {
      pendingSessionID = await terminalPage.workspace().getAttribute("data-terminal-session-id") || "";
      return pendingSessionID;
    }).toMatch(/^terminal-pending-/);
    await expect(page.locator("[data-terminal-runtime-note]")).toHaveCount(0);
    await expect(terminalPage.workspace()).not.toContainText("Codex runtime exited. Send a new input to recover this session.");

    await expect.poll(async () => await terminalPage.workspace().getAttribute("data-terminal-session-id")).not.toBe(session.id);
    await expect.poll(async () => await terminalPage.workspace().getAttribute("data-terminal-session-id")).not.toBe(String(pendingSessionID));
    await expect(page.locator("[data-terminal-runtime-note]")).toHaveCount(0);
  });

  test("recovers stored thread-backed sessions on load and first input", async ({ page }) => {
    const clientID = createTerminalClientID("recover");
    const now = Date.now();
    const sessionID = `terminal-recover-live-${now}`;
    const threadID = `mock-thread-${sessionID}`;
    const session = {
      id: sessionID,
      title: sessionID,
      terminal_session_id: threadID,
      status: "running",
      shell: "codex exec",
      working_dir: terminalSessionWorkspace(sessionID),
      created_at: now - 2_000,
      last_output_at: now - 1_000,
      updated_at: now - 500,
      entry_cursor: 1,
      disconnected_notice: false,
      entries: [
        {
          id: `${sessionID}-output`,
          role: "output",
          text: "mock:before-reload",
          at: now - 1_000,
          kind: "stdout",
          stream: "stdout",
          cursor: 1,
        },
      ],
    };

    await bindTerminalClient(page, clientID);
    await seedTerminalSessions(page, [session]);
    await openTerminalRoute(page);

    const terminalPage = createTerminalPage(page);
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-status", "running");
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-live", "true");
    await expectComposerReady(terminalPage.composer());
    await terminalPage.metaToggle().click();
    await expect(terminalPage.workspace()).toContainText(threadID);
    await expect(terminalPage.composer().input()).toBeEnabled();
    await expect(terminalPage.composer().submitButton()).toBeEnabled();

    await terminalPage.composer().input().fill("Reply with exactly: recovered-after-reload");
    await terminalPage.composer().submitButton().click();

    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-status", "running");
    await expect(terminalPage.finalOutputs().last()).toContainText("recovered-after-reload");
  });
});
