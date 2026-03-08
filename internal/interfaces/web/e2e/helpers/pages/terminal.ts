import { type Locator, type Page } from "@playwright/test";
import { createComposerComponent, type ComposerComponent } from "../components/composer";
import { createSessionListComponent, type SessionListComponent } from "../components/session-list";

export function createTerminalPage(page: Page): {
  createButton(): Locator;
  composer(): ComposerComponent;
  closeButton(): Locator;
  workspace(): Locator;
  chatScreen(): Locator;
  sessionList(): SessionListComponent;
  sessionListContainer(): Locator;
} {
  const composer = createComposerComponent(page, "terminal-runtime");
  const sessionList = createSessionListComponent(page, {
    items: "[data-terminal-session-select]",
    itemByValue: (sessionID: string) => `[data-terminal-session-select="${sessionID}"]`,
  });
  return {
    createButton: () => page.locator("[data-terminal-create]"),
    composer: () => composer,
    closeButton: () => page.locator("[data-terminal-close]"),
    workspace: () => page.locator("[data-terminal-workspace]"),
    chatScreen: () => page.locator("[data-terminal-chat-screen]"),
    sessionList: () => sessionList,
    sessionListContainer: () => page.locator("[data-terminal-session-list]"),
  };
}
