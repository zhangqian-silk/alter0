import { type Locator, type Page } from "@playwright/test";
import { createComposerComponent, type ComposerComponent } from "../components/composer";
import { createSessionListComponent, type SessionListComponent } from "../components/session-list";

export function createChatPage(page: Page): {
  composer(): ComposerComponent;
  newChatButton(): Locator;
  latestUserBubble(): Locator;
  sessionList(): SessionListComponent;
} {
  const composer = createComposerComponent(page, "conversation", {
    hasCounter: true,
  });
  const sessionList = createSessionListComponent(page, {
    items: ".session-card",
    deleteButtons: ".session-card-delete",
  });
  return {
    composer: () => composer,
    newChatButton: () => page.locator("#newChatButton"),
    latestUserBubble: () => page.locator(".msg.user .msg-bubble").last(),
    sessionList: () => sessionList,
  };
}
