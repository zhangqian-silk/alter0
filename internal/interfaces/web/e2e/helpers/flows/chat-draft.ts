import { expect, type Locator, type Page } from "@playwright/test";
import { expectComposerReady } from "../asserts/composer";
import { createChatPage } from "../pages/chat";
import { createNewChatSession } from "./chat-session";

export async function seedChatSession(page: Page, content = "seed message"): Promise<void> {
  const chatPage = createChatPage(page);
  const composer = chatPage.composer();
  const input = composer.input();
  await input.fill(content);
  await composer.submitButton().click();
  await expect(chatPage.latestUserBubble()).toContainText(content);
  await expectComposerReady(composer, 60000);
}

export async function setupTwoChatDraftSessions(
  page: Page,
  options: {
    seedContent?: string;
    firstDraft?: string;
    secondDraft?: string;
  } = {}
): Promise<{ input: Locator; sessionCards: Locator }> {
  const chatPage = createChatPage(page);
  const input = chatPage.composer().input();
  const sessionCards = chatPage.sessionList().items();
  await seedChatSession(page, options.seedContent || "seed message");
  await input.fill(options.firstDraft || "draft-a");
  await createNewChatSession(page);
  await expect(sessionCards).toHaveCount(2);
  await expect(input).toHaveValue("");
  await input.fill(options.secondDraft || "draft-b");
  return { input, sessionCards };
}
