export const LEGACY_SHELL_NAVIGATE_EVENT = "alter0:legacy-shell:navigate";
export const LEGACY_SHELL_CREATE_SESSION_EVENT = "alter0:legacy-shell:create-session";
export const LEGACY_SHELL_FOCUS_SESSION_EVENT = "alter0:legacy-shell:focus-session";
export const LEGACY_SHELL_REMOVE_SESSION_EVENT = "alter0:legacy-shell:remove-session";
export const LEGACY_SHELL_TOGGLE_LANGUAGE_EVENT = "alter0:legacy-shell:toggle-language";
export const LEGACY_SHELL_SYNC_NAV_COLLAPSED_EVENT = "alter0:legacy-shell:sync-nav-collapsed";
export const LEGACY_SHELL_SYNC_SESSION_HISTORY_EVENT = "alter0:legacy-shell:sync-session-history-collapsed";
export const LEGACY_SHELL_QUICK_PROMPT_EVENT = "alter0:legacy-shell:quick-prompt";
export const LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT = "alter0:legacy-shell:sync-chat-workspace";
export const LEGACY_SHELL_SYNC_SESSION_PANE_EVENT = "alter0:legacy-shell:sync-session-pane";
export const LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT = "alter0:legacy-shell:sync-message-region";
export const LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT = "alter0:legacy-shell:sync-chat-runtime";

type LegacyShellNavigateDetail = {
  route: string;
};

type LegacyShellNavCollapsedDetail = {
  collapsed: boolean;
};

type LegacyShellSessionHistoryDetail = {
  collapsed: boolean;
};

type LegacyShellQuickPromptDetail = {
  prompt: string;
};

type LegacyShellSessionActionDetail = {
  sessionId: string;
};

export type LegacyShellChatWorkspaceDetail = {
  route: string;
  heading: string;
  subheading: string;
  welcomeHeading: string;
  welcomeDescription: string;
  welcomeTargetHTML?: string;
};

export type LegacyShellSessionPaneDetail = {
  route: string;
  hasSessions: boolean;
  loadError: string;
  items?: Array<{
    id: string;
    title: string;
    meta: string;
    active: boolean;
    shortHash: string;
    copyValue: string;
    copyLabel: string;
    deleteLabel: string;
  }>;
};

export type LegacyShellMessageRegionDetail = {
  route: string;
  hasMessages: boolean;
  sessionId?: string;
  html?: string;
};

export type LegacyShellChatRuntimeDetail = {
  route: string;
  controlsHTML?: string;
  noteHTML?: string;
  sheetHTML?: string;
  scrollPopover?: string;
  scrollTop?: number;
};

export function requestLegacyShellRouteNavigation(route: string): boolean {
  return document.dispatchEvent(
    new CustomEvent<LegacyShellNavigateDetail>(LEGACY_SHELL_NAVIGATE_EVENT, {
      bubbles: true,
      detail: { route },
    }),
  );
}

export function requestLegacyShellSessionCreation(): boolean {
  return document.dispatchEvent(
    new CustomEvent(LEGACY_SHELL_CREATE_SESSION_EVENT, {
      bubbles: true,
    }),
  );
}

export function requestLegacyShellSessionFocus(sessionId: string): boolean {
  return document.dispatchEvent(
    new CustomEvent<LegacyShellSessionActionDetail>(LEGACY_SHELL_FOCUS_SESSION_EVENT, {
      bubbles: true,
      detail: { sessionId },
    }),
  );
}

export function requestLegacyShellSessionRemoval(sessionId: string): boolean {
  return document.dispatchEvent(
    new CustomEvent<LegacyShellSessionActionDetail>(LEGACY_SHELL_REMOVE_SESSION_EVENT, {
      bubbles: true,
      detail: { sessionId },
    }),
  );
}

export function requestLegacyShellLanguageToggle(): boolean {
  return document.dispatchEvent(
    new CustomEvent(LEGACY_SHELL_TOGGLE_LANGUAGE_EVENT, {
      bubbles: true,
    }),
  );
}

export function syncLegacyShellNavCollapsed(collapsed: boolean): boolean {
  return document.dispatchEvent(
    new CustomEvent<LegacyShellNavCollapsedDetail>(LEGACY_SHELL_SYNC_NAV_COLLAPSED_EVENT, {
      bubbles: true,
      detail: { collapsed },
    }),
  );
}

export function requestLegacyShellQuickPrompt(prompt: string): boolean {
  return document.dispatchEvent(
    new CustomEvent<LegacyShellQuickPromptDetail>(LEGACY_SHELL_QUICK_PROMPT_EVENT, {
      bubbles: true,
      detail: { prompt },
    }),
  );
}

export function syncLegacyShellSessionHistoryCollapsed(collapsed: boolean): boolean {
  return document.dispatchEvent(
    new CustomEvent<LegacyShellSessionHistoryDetail>(LEGACY_SHELL_SYNC_SESSION_HISTORY_EVENT, {
      bubbles: true,
      detail: { collapsed },
    }),
  );
}
