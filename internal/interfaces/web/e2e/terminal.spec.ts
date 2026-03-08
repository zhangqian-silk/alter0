import { expect, test } from "@playwright/test";
import {
  expectComposerFocusedValue,
  expectComposerReady,
  expectComposerState,
  expectComposerValue,
} from "./helpers/asserts/composer";
import { commitIMEInput, startIMEInput } from "./helpers/interactions/ime";
import { selectTerminalSession } from "./helpers/flows/terminal-session";
import { waitForTerminalPoll, waitForTerminalPollAndRepaint } from "./helpers/flows/terminal-runtime";
import {
  openInterruptedTerminalWorkspace,
  openReadyTerminalWorkspace,
  openTerminalWorkspace,
  openTerminalWorkspaceWithSessions,
} from "./helpers/scenarios/terminal";

test.describe("Terminal route", () => {
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
    await input.fill("Write-Output alter0-playwright");
    await submit.click();

    await expect(terminalPage.chatScreen()).toContainText("alter0-playwright");

    const closeButton = terminalPage.closeButton();
    await expect(closeButton).toBeEnabled();
    await closeButton.click();

    await expect(terminalPage.workspace()).toContainText("Exited");
    await expect(input).toBeDisabled();
    await expect(submit).toBeDisabled();
    await expectComposerState(composer, { disabled: true });
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

  test("marks stored live sessions as interrupted when runtime is unavailable", async ({ page }) => {
    const { terminalPage } = await openInterruptedTerminalWorkspace(page, { scope: "interrupted" });

    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-status", "interrupted");
    await expect(terminalPage.workspace()).toHaveAttribute("data-terminal-workspace-live", "false");
    await expect(terminalPage.workspace()).toContainText("Interrupted");
    await expect(terminalPage.workspace()).toContainText("Terminal session interrupted and cannot be reopened.");
    await expect(terminalPage.composer().input()).toBeDisabled();
    await expect(terminalPage.composer().submitButton()).toBeDisabled();
    await expectComposerState(terminalPage.composer(), { disabled: true });
  });
});
