import { type Locator, type Page } from "@playwright/test";
import { createComposerComponent, type ComposerComponent } from "../components/composer";
import { createSessionListComponent, type SessionListComponent } from "../components/session-list";

export function createTerminalPage(page: Page): {
  createButton(): Locator;
  composer(): ComposerComponent;
  workspace(): Locator;
  chatScreen(): Locator;
  jumpTopButton(): Locator;
  jumpPrevButton(): Locator;
  jumpNextButton(): Locator;
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
  sessionDeleteButton(sessionID: string): Locator;
} {
  const composer = createComposerComponent(page, "terminal");
  const sessionList = createSessionListComponent(page, {
    items: "[data-runtime-session-select]",
    itemByValue: (sessionID: string) => `[data-runtime-session-select="${sessionID}"]`,
  });
  return {
    createButton: () => page.locator("[data-runtime-create-session='terminal']:visible").first(),
    composer: () => composer,
    workspace: () => page.locator("[data-runtime-workspace='terminal']"),
    chatScreen: () => page.locator("[data-runtime-screen='terminal']"),
    jumpTopButton: () => page.locator("[data-terminal-jump-top]"),
    jumpPrevButton: () => page.locator("[data-terminal-jump-prev]"),
    jumpNextButton: () => page.locator("[data-terminal-jump-next]"),
    jumpBottomButton: () => page.locator("[data-terminal-jump-bottom]"),
    sessionPaneToggle: () => page.getByRole("button", { name: "Sessions" }).first(),
    sessionPane: () => page.locator("[data-runtime-session-pane='terminal']"),
    sessionPaneClose: () => page.locator("[data-runtime-session-pane-close='terminal']").first(),
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
    sessionListContainer: () => page.locator("[data-runtime-session-list='terminal']"),
    sessionDeleteButton: (sessionID: string) => page.locator(`[data-runtime-delete-session="${sessionID}"]`),
  };
}
