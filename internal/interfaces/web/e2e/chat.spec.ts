import { expect, test, type Page } from "@playwright/test";
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
import { commitIMEInput, startIMEInput } from "./helpers/interactions/ime";
import { clickWithUnsavedDialog } from "./helpers/guards/unsaved";
import {
  openChatWorkspace,
  openChatWorkspaceWithDraft,
  openChatWorkspaceWithTwoDraftSessions,
  reloadChatWorkspace,
} from "./helpers/scenarios/chat";

type VisualViewportShape = {
  width?: number;
  height?: number;
  offsetTop?: number;
  offsetLeft?: number;
};

async function installVisualViewportMock(page: Page): Promise<void> {
  await page.addInitScript(() => {
    class MockVisualViewport extends EventTarget {
      width: number;
      height: number;
      offsetTop: number;
      offsetLeft: number;
      pageTop: number;
      pageLeft: number;
      scale: number;

      constructor() {
        super();
        this.width = window.innerWidth;
        this.height = window.innerHeight;
        this.offsetTop = 0;
        this.offsetLeft = 0;
        this.pageTop = 0;
        this.pageLeft = 0;
        this.scale = 1;
      }
    }

    const mock = new MockVisualViewport();
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: mock,
    });
    Object.defineProperty(window, "__alter0SetVisualViewport", {
      configurable: true,
      value: (next: VisualViewportShape) => {
        if (!next || typeof next !== "object") {
          return;
        }
        if (typeof next.width === "number") {
          mock.width = next.width;
        }
        if (typeof next.height === "number") {
          mock.height = next.height;
        }
        if (typeof next.offsetTop === "number") {
          mock.offsetTop = next.offsetTop;
          mock.pageTop = next.offsetTop;
        }
        if (typeof next.offsetLeft === "number") {
          mock.offsetLeft = next.offsetLeft;
          mock.pageLeft = next.offsetLeft;
        }
        mock.dispatchEvent(new Event("resize"));
        mock.dispatchEvent(new Event("scroll"));
      },
    });
  });
}

async function setVisualViewport(
  page: Page,
  next: VisualViewportShape
): Promise<void> {
  await page.evaluate((value) => {
    const setter = (window as typeof window & {
      __alter0SetVisualViewport?: (payload: typeof value) => void;
    }).__alter0SetVisualViewport;
    setter?.(value);
  }, next);
}

test.describe("Chat composer", () => {
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
    const sendButton = page.locator("#sendButton");

    await expect(navToggle).toBeVisible();
    await expect(sessionToggle).toBeVisible();
    await expect(newChatButton).toBeVisible();
    await expect(runtimeToggles).toHaveCount(1);
    await expect(composerNote).toBeHidden();
    await expect(composerCounter).toBeHidden();
    await expect(runtimeToggles.first()).toContainText("Session");
    await expect(runtimeToggles.first()).toContainText("Tools 0");
    await expect(runtimeToggles.first()).toContainText("Skills 0");

    const navBox = await navToggle.boundingBox();
    const sessionBox = await sessionToggle.boundingBox();
    const newChatBox = await newChatButton.boundingBox();
    const headingBox = await heading.boundingBox();
    const composerBox = await composerShell.boundingBox();
    const runtimeBox = await runtimeToggles.first().boundingBox();
    const sendBox = await sendButton.boundingBox();
    const viewport = page.viewportSize();

    expect(navBox).not.toBeNull();
    expect(sessionBox).not.toBeNull();
    expect(newChatBox).not.toBeNull();
    expect(headingBox).not.toBeNull();
    expect(composerBox).not.toBeNull();
    expect(runtimeBox).not.toBeNull();
    expect(sendBox).not.toBeNull();
    expect(viewport).not.toBeNull();

    expect(Math.abs((navBox?.y ?? 0) - (sessionBox?.y ?? 0))).toBeLessThan(6);
    expect(Math.abs((sessionBox?.y ?? 0) - (newChatBox?.y ?? 0))).toBeLessThan(6);
    expect(headingBox?.y ?? 0).toBeGreaterThan((navBox?.y ?? 0) + (navBox?.height ?? 0) - 2);
    expect((viewport?.height ?? 0) - ((composerBox?.y ?? 0) + (composerBox?.height ?? 0))).toBeLessThan(20);
    expect(Math.abs((runtimeBox?.y ?? 0) - (sendBox?.y ?? 0))).toBeLessThan(6);
    expect(sendBox?.x ?? 0).toBeGreaterThan((runtimeBox?.x ?? 0) + (runtimeBox?.width ?? 0) - 4);

    await runtimeToggles.first().click();
    await expect(page.locator(".composer-runtime-popover-mobile")).toBeVisible();
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

  test("upgrades the session title after a later user message becomes specific", async ({ page }) => {
    const seededAt = Date.now();
    await page.addInitScript(({ createdAt }) => {
      const sessionID = "session-seeded-title";
      window.localStorage.setItem("alter0.web.sessions.v3", JSON.stringify([{
        id: sessionID,
        title: "先拉取仓库",
        titleAuto: true,
        titleScore: 0,
        createdAt,
        messages: [{
          id: "message-seeded-title",
          role: "user",
          text: "先拉取仓库",
          at: createdAt,
          status: "done"
        }],
        historyBucket: "agent:main",
        targetType: "model",
        targetID: "raw-model",
        targetName: "Raw Model",
        modelProviderID: "",
        modelID: "",
        toolIDs: [],
        skillIDs: [],
        mcpIDs: []
      }]));
      window.localStorage.setItem("alter0.web.session.active.v1", JSON.stringify({
        "agent:main": sessionID
      }));
    }, { createdAt: seededAt });

    const { chatPage, composer } = await openChatWorkspace(page);
    const input = composer.input();
    await expect(page.locator(".session-card-title").first()).toContainText("先拉取仓库");

    await input.fill("修改 terminal 和 agent 的会话标题");
    await composer.submitButton().click();
    await expect(chatPage.latestUserBubble()).toContainText("修改 terminal 和 agent 的会话标题");
    await expectComposerReady(composer);
    await expect(page.locator(".session-card-title").first()).toContainText("修改 terminal");
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
});
