import { LEGACY_SHELL_IDS } from "./legacyDomContract";
import { isLegacyShellChatRoute } from "./legacyShellState";

export const LEGACY_TRANSIENT_SHELL_CLASSES = [
  "nav-open",
  "panel-open",
  "overlay-open",
  "runtime-sheet-open",
] as const;

export type LegacyTransientShellClass = (typeof LEGACY_TRANSIENT_SHELL_CLASSES)[number];

export function readLegacyTransientShellClasses(root: HTMLElement | null): LegacyTransientShellClass[] {
  if (!root) {
    return [];
  }

  return LEGACY_TRANSIENT_SHELL_CLASSES.filter((className) => root.classList.contains(className));
}

function syncLegacyOverlayState(root: HTMLElement): void {
  const opened = root.classList.contains("nav-open") || root.classList.contains("panel-open");
  root.classList.toggle("overlay-open", opened);
}

function dispatchLegacyShellDismiss(): void {
  document.dispatchEvent(
    new KeyboardEvent("keydown", {
      bubbles: true,
      key: "Escape",
    }),
  );
}

export function dismissLegacyTransientPanels(root: HTMLElement): void {
  dispatchLegacyShellDismiss();
  root.classList.remove("nav-open", "panel-open");
  syncLegacyOverlayState(root);
}

export function toggleLegacyNavDrawer(root: HTMLElement): void {
  const shouldOpen = !root.classList.contains("nav-open");
  dismissLegacyTransientPanels(root);
  if (!shouldOpen) {
    return;
  }

  root.classList.add("nav-open");
  syncLegacyOverlayState(root);
}

export function toggleLegacySessionPane(root: HTMLElement, currentRoute: string): void {
  if (currentRoute === "terminal") {
    const terminalToggle = document
      .getElementById(LEGACY_SHELL_IDS.routeBody)
      ?.querySelector("[data-terminal-session-pane-toggle]");
    if (terminalToggle instanceof HTMLElement) {
      terminalToggle.click();
    }
    return;
  }

  if (!isLegacyShellChatRoute(currentRoute)) {
    return;
  }

  const shouldOpen = !root.classList.contains("panel-open");
  dismissLegacyTransientPanels(root);
  if (!shouldOpen) {
    return;
  }

  root.classList.add("panel-open");
  syncLegacyOverlayState(root);
}
