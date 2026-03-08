import { type Locator, type Page } from "@playwright/test";
import type { ComposerComponent } from "../components/composer";
import { ensureChatRouteReady, openChatRoute } from "../flows/routes";
import { setupTwoChatDraftSessions } from "../flows/chat-draft";
import { createAppShellPage } from "../pages/app-shell";
import { createChatPage } from "../pages/chat";

export async function openChatWorkspace(page: Page): Promise<{
  appShellPage: ReturnType<typeof createAppShellPage>;
  chatPage: ReturnType<typeof createChatPage>;
  composer: ComposerComponent;
}> {
  await openChatRoute(page);
  const appShellPage = createAppShellPage(page);
  const chatPage = createChatPage(page);
  return {
    appShellPage,
    chatPage,
    composer: chatPage.composer(),
  };
}

export async function openChatWorkspaceWithDraft(
  page: Page,
  draft: string
): Promise<{
  appShellPage: ReturnType<typeof createAppShellPage>;
  chatPage: ReturnType<typeof createChatPage>;
  composer: ComposerComponent;
}> {
  const workspace = await openChatWorkspace(page);
  await workspace.composer.input().fill(draft);
  return workspace;
}

export async function openChatWorkspaceWithTwoDraftSessions(
  page: Page,
  options: {
    seedContent?: string;
    firstDraft?: string;
    secondDraft?: string;
  } = {}
): Promise<{
  chatPage: ReturnType<typeof createChatPage>;
  composer: ComposerComponent;
  sessionCards: Locator;
}> {
  const { chatPage } = await openChatWorkspace(page);
  const { sessionCards } = await setupTwoChatDraftSessions(page, options);
  return { chatPage, composer: chatPage.composer(), sessionCards };
}

export async function reloadChatWorkspace(page: Page): Promise<{
  chatPage: ReturnType<typeof createChatPage>;
  composer: ComposerComponent;
}> {
  await page.reload();
  await ensureChatRouteReady(page);
  const chatPage = createChatPage(page);
  return {
    chatPage,
    composer: chatPage.composer(),
  };
}
