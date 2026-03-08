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
import { waitForTerminalPoll, waitForTerminalPollAndRepaint } from "./helpers/flows/terminal-runtime";
import { bindTerminalClient, closeTrackedTerminalSessions, createTerminalClientID, seedTerminalSessions } from "./helpers/flows/terminal-session";
import { createTerminalPage } from "./helpers/pages/terminal";
import {
  openInterruptedTerminalWorkspace,
  openReadyTerminalWorkspace,
  openTerminalWorkspace,
  openTerminalWorkspaceWithSessions,
} from "./helpers/scenarios/terminal";
import { terminalCommandPreview, terminalRepoRoot } from "./helpers/support/terminal-env";

const seededTerminalWorkingDir = terminalRepoRoot;

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

  test("sends a command and disables input after close", async ({ page, request }) => {
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
    await expect(input).toBeDisabled();
    await expect(submit).toBeDisabled();
    await expectComposerState(composer, { disabled: true });
  });

  test("renders process and final output with lazy-loaded step details", async ({ page, request }) => {
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
    await expect(terminalPage.workspace()).toContainText("WorkingDirectory");
    await expect(terminalPage.workspace()).toContainText(seededTerminalWorkingDir);
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
      working_dir: seededTerminalWorkingDir,
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
      working_dir: seededTerminalWorkingDir,
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
      working_dir: seededTerminalWorkingDir,
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

  test("marks stored live sessions as interrupted when runtime is unavailable", async ({ page }) => {
    const { terminalPage } = await openInterruptedTerminalWorkspace(page, { scope: "interrupted" });

    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-status", "interrupted");
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-live", "false");
    await expect(terminalPage.workspace()).toContainText("Interrupted");
    await expect(terminalPage.workspace()).toContainText("Codex session interrupted and cannot be resumed here.");
    await expect(terminalPage.composer().input()).toBeDisabled();
    await expect(terminalPage.composer().submitButton()).toBeDisabled();
    await expectComposerState(terminalPage.composer(), { disabled: true });
  });
});
