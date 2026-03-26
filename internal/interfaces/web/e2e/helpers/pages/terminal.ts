import { type Locator, type Page } from "@playwright/test";
import { createComposerComponent, type ComposerComponent } from "../components/composer";
import { createSessionListComponent, type SessionListComponent } from "../components/session-list";

export function createTerminalPage(page: Page): {
  createButton(): Locator;
  composer(): ComposerComponent;
  closeButton(): Locator;
  workspace(): Locator;
  chatScreen(): Locator;
  jumpBottomButton(): Locator;
  sessionPaneToggle(): Locator;
  sessionPane(): Locator;
  sessionPaneClose(): Locator;
  metaToggle(): Locator;
  turnCards(): Locator;
  turnCard(turnID: string): Locator;
  processToggles(): Locator;
  processToggle(turnID: string): Locator;
  outputToggle(turnID: string): Locator;
  stepItems(): Locator;
  stepItem(stepID: string): Locator;
  stepToggles(): Locator;
  stepToggle(stepID: string): Locator;
  finalOutputs(): Locator;
  sessionList(): SessionListComponent;
  sessionListContainer(): Locator;
} {
  const composer = createComposerComponent(page, "terminal-runtime");
  const sessionList = createSessionListComponent(page, {
    items: "[data-terminal-session-select]",
    itemByValue: (sessionID: string) => `[data-terminal-session-select="${sessionID}"]`,
  });
  return {
    createButton: () => page.locator("[data-terminal-create]").first(),
    composer: () => composer,
    closeButton: () => page.locator("[data-terminal-close]"),
    workspace: () => page.locator("[data-terminal-workspace]"),
    chatScreen: () => page.locator("[data-terminal-chat-screen]"),
    jumpBottomButton: () => page.locator("[data-terminal-jump-bottom]"),
    sessionPaneToggle: () => page.locator("[data-terminal-session-pane-toggle]"),
    sessionPane: () => page.locator("[data-terminal-session-pane]"),
    sessionPaneClose: () => page.locator("[data-terminal-session-pane-close]").first(),
    metaToggle: () => page.locator("[data-terminal-meta-toggle]"),
    turnCards: () => page.locator("[data-terminal-turn]"),
    turnCard: (turnID: string) => page.locator(`[data-terminal-turn="${turnID}"]`),
    processToggles: () => page.locator("[data-terminal-process-toggle]"),
    processToggle: (turnID: string) => page.locator(`[data-terminal-process-toggle="${turnID}"]`),
    outputToggle: (turnID: string) => page.locator(`[data-terminal-output-toggle="${turnID}"]`),
    stepItems: () => page.locator("[data-terminal-step-item]"),
    stepItem: (stepID: string) => page.locator(`[data-terminal-step-item="${stepID}"]`),
    stepToggles: () => page.locator("[data-terminal-step-toggle]"),
    stepToggle: (stepID: string) => page.locator(`[data-terminal-step-toggle="${stepID}"]`),
    finalOutputs: () => page.locator(".terminal-final-output"),
    sessionList: () => sessionList,
    sessionListContainer: () => page.locator("[data-terminal-session-list]"),
  };
}
