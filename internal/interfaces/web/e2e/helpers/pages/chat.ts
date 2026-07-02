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
    items: "[data-runtime-session-card]:visible",
    deleteButtons: ".runtime-session-delete",
  });
  return {
    composer: () => composer,
    newChatButton: () => page.getByRole("button", { name: "New" }).first(),
    latestUserBubble: () => page.locator(".runtime-message-user .runtime-message-bubble").last(),
    sessionList: () => sessionList,
  };
}
