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

  test("keeps empty chat controls tidy and composer docked on narrow screens", async ({ page }) => {
    await page.setViewportSize({ width: 760, height: 980 });
    await openChatWorkspace(page);

    const navToggle = page.locator("#navToggle");
    const sessionToggle = page.locator("#sessionToggle");
    const newChatButton = page.locator("#mobileNewChatButton");
    const heading = page.locator("#sessionHeading");
    const composerShell = page.locator(".composer-shell");
    const runtimeToggles = page.locator("#chatRuntimePanel [data-runtime-toggle]");
    const composerNote = page.locator(".composer-note");
    const composerCounter = page.locator("#charCount");

    await expect(navToggle).toBeVisible();
    await expect(sessionToggle).toBeVisible();
    await expect(newChatButton).toBeVisible();
    await expect(runtimeToggles).toHaveCount(1);
    await expect(composerNote).toBeHidden();
    await expect(composerCounter).toBeHidden();

    const navBox = await navToggle.boundingBox();
    const sessionBox = await sessionToggle.boundingBox();
    const newChatBox = await newChatButton.boundingBox();
    const headingBox = await heading.boundingBox();
    const composerBox = await composerShell.boundingBox();
    const viewport = page.viewportSize();

    expect(navBox).not.toBeNull();
    expect(sessionBox).not.toBeNull();
    expect(newChatBox).not.toBeNull();
    expect(headingBox).not.toBeNull();
    expect(composerBox).not.toBeNull();
    expect(viewport).not.toBeNull();

    expect(Math.abs((navBox?.y ?? 0) - (sessionBox?.y ?? 0))).toBeLessThan(6);
    expect(Math.abs((sessionBox?.y ?? 0) - (newChatBox?.y ?? 0))).toBeLessThan(6);
    expect(headingBox?.y ?? 0).toBeGreaterThan((navBox?.y ?? 0) + (navBox?.height ?? 0) - 2);
    expect((viewport?.height ?? 0) - ((composerBox?.y ?? 0) + (composerBox?.height ?? 0))).toBeLessThan(20);

    await runtimeToggles.first().click();
    await expect(page.locator(".composer-runtime-popover-mobile")).toBeVisible();
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

  test("keeps user bubbles right-aligned and within seventy percent width", async ({ page }) => {
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
      const maxWidth = listRect.width * 0.7;
      const rightGap = Math.round(listRect.right - bubbleRect.right);
      const leftGap = Math.round(bubbleRect.left - listRect.left);
      return (
        bubbleRect.width <= maxWidth + 2 &&
        rightGap <= leftGap &&
        Math.round(listRect.right - messageRect.right) <= Math.round(messageRect.left - listRect.left)
      );
    })).toBe(true);
  });
});
