import { expect, test } from "@playwright/test";
import {
  expectComposerCounter,
  expectComposerFocusedValue,
  expectComposerState,
  expectComposerValue,
} from "./helpers/asserts/composer";
import {
  createNewChatSession,
  expectActiveChatSession,
  removeChatSession,
  switchChatSession
} from "./helpers/flows/chat-session";
import { commitIMEInput, startIMEInput } from "./helpers/interactions/ime";
import { clickWithUnsavedDialog } from "./helpers/guards/unsaved";
import {
  openChatWorkspace,
  openChatWorkspaceWithDraft,
  openChatWorkspaceWithTwoDraftSessions,
  reloadChatWorkspace,
} from "./helpers/scenarios/chat";

test.describe("Chat composer", () => {
  test("keeps empty session hint near the session list header", async ({ page }) => {
    await openChatWorkspace(page);

    const historyPanel = page.locator("#sessionHistoryPanel");
    const emptyHint = page.locator("#sessionEmpty");

    await expect(emptyHint).toBeVisible();

    const panelBox = await historyPanel.boundingBox();
    const emptyBox = await emptyHint.boundingBox();

    expect(panelBox).not.toBeNull();
    expect(emptyBox).not.toBeNull();
    expect((emptyBox?.y ?? 0) - (panelBox?.y ?? 0)).toBeLessThan(120);
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
});
