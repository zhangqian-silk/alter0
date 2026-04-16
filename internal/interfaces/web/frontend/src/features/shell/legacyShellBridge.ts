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

declare global {
  interface Window {
    __alter0LegacyRuntime?: {
      createSession?: () => boolean | void;
      focusSession?: (sessionId: string) => boolean | void;
      removeSession?: (sessionId: string) => boolean | void;
      toggleChatRuntimePopover?: (popover: string) => boolean | void;
      closeChatRuntimePopover?: () => boolean | void;
      selectChatRuntimeTarget?: (target: LegacyShellChatRuntimeTargetActionDetail) => boolean | void;
      selectChatRuntimeModel?: (selection: LegacyShellChatRuntimeModelActionDetail) => boolean | void;
      toggleChatRuntimeItem?: (selection: LegacyShellChatRuntimeItemActionDetail) => boolean | void;
    };
  }
}

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

export type LegacyShellChatRuntimeTargetActionDetail = {
  type: string;
  id: string;
  name: string;
};

export type LegacyShellChatRuntimeModelActionDetail = {
  providerId: string;
  modelId: string;
};

export type LegacyShellChatRuntimeItemActionDetail = {
  group: "capabilities" | "skills";
  id: string;
  checked: boolean;
  kind?: "tool" | "mcp";
};

export type LegacyShellChatWorkspaceDetail = {
  route: string;
  heading: string;
  subheading: string;
  welcomeHeading: string;
  welcomeDescription: string;
  welcomeTargets?: LegacyShellChatWorkspaceTargetDetail[];
  welcomeTargetError?: string;
};

export type LegacyShellChatWorkspaceTargetDetail = {
  type?: string;
  id?: string;
  name?: string;
  active?: boolean;
  interactive?: boolean;
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
  messages?: LegacyShellMessageSnapshotDetail[];
};

export type LegacyShellMessageSnapshotDetail = {
  id?: string;
  role?: string;
  text?: string;
  route?: string;
  source?: string;
  error?: boolean;
  status?: string;
  at?: number;
  process_steps?: LegacyShellMessageProcessStepDetail[];
  agent_process_collapsed?: boolean;
};

export type LegacyShellMessageProcessStepDetail = {
  id?: string;
  kind?: string;
  title?: string;
  detail?: string;
  status?: string;
};

export type LegacyShellChatRuntimeTargetDetail = {
  type?: string;
  id?: string;
  name?: string;
};

export type LegacyShellChatRuntimeTargetOptionDetail = {
  type?: string;
  id?: string;
  name?: string;
  subtitle?: string;
  active?: boolean;
};

export type LegacyShellChatRuntimeModelDetail = {
  id?: string;
  name?: string;
  active?: boolean;
};

export type LegacyShellChatRuntimeProviderDetail = {
  id?: string;
  name?: string;
  models?: LegacyShellChatRuntimeModelDetail[];
};

export type LegacyShellChatRuntimeSelectionDetail = {
  id?: string;
  name?: string;
  description?: string;
  kind?: "tool" | "mcp" | "skill";
  active?: boolean;
};

export type LegacyShellChatRuntimeDetail = {
  route: string;
  compact?: boolean;
  openPopover?: string;
  note?: string;
  agentRuntime?: boolean;
  locked?: boolean;
  target?: LegacyShellChatRuntimeTargetDetail;
  targetOptions?: LegacyShellChatRuntimeTargetOptionDetail[];
  selectedProviderId?: string;
  selectedModelId?: string;
  selectedModelLabel?: string;
  toolCount?: number;
  skillCount?: number;
  providers?: LegacyShellChatRuntimeProviderDetail[];
  capabilities?: LegacyShellChatRuntimeSelectionDetail[];
  skills?: LegacyShellChatRuntimeSelectionDetail[];
};

function callLegacyRuntimeAction<TArgs extends unknown[]>(
  action: ((...args: TArgs) => boolean | void) | undefined,
  args: TArgs,
): boolean | null {
  if (typeof action !== "function") {
    return null;
  }
  return action(...args) !== false;
}

export function requestLegacyShellRouteNavigation(route: string): boolean {
  return document.dispatchEvent(
    new CustomEvent<LegacyShellNavigateDetail>(LEGACY_SHELL_NAVIGATE_EVENT, {
      bubbles: true,
      detail: { route },
    }),
  );
}

export function requestLegacyShellSessionCreation(): boolean {
  const handled = callLegacyRuntimeAction(window.__alter0LegacyRuntime?.createSession, []);
  if (handled !== null) {
    return handled;
  }
  return document.dispatchEvent(
    new CustomEvent(LEGACY_SHELL_CREATE_SESSION_EVENT, {
      bubbles: true,
    }),
  );
}

export function requestLegacyShellSessionFocus(sessionId: string): boolean {
  const handled = callLegacyRuntimeAction(window.__alter0LegacyRuntime?.focusSession, [sessionId]);
  if (handled !== null) {
    return handled;
  }
  return document.dispatchEvent(
    new CustomEvent<LegacyShellSessionActionDetail>(LEGACY_SHELL_FOCUS_SESSION_EVENT, {
      bubbles: true,
      detail: { sessionId },
    }),
  );
}

export function requestLegacyShellSessionRemoval(sessionId: string): boolean {
  const handled = callLegacyRuntimeAction(window.__alter0LegacyRuntime?.removeSession, [sessionId]);
  if (handled !== null) {
    return handled;
  }
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

export function requestLegacyChatRuntimePopover(popover: string): boolean {
  return callLegacyRuntimeAction(window.__alter0LegacyRuntime?.toggleChatRuntimePopover, [popover]) ?? false;
}

export function requestLegacyChatRuntimeClose(): boolean {
  return callLegacyRuntimeAction(window.__alter0LegacyRuntime?.closeChatRuntimePopover, []) ?? false;
}

export function requestLegacyChatRuntimeTarget(
  target: LegacyShellChatRuntimeTargetActionDetail,
): boolean {
  return callLegacyRuntimeAction(window.__alter0LegacyRuntime?.selectChatRuntimeTarget, [target]) ?? false;
}

export function requestLegacyChatRuntimeModel(
  selection: LegacyShellChatRuntimeModelActionDetail,
): boolean {
  return callLegacyRuntimeAction(window.__alter0LegacyRuntime?.selectChatRuntimeModel, [selection]) ?? false;
}

export function requestLegacyChatRuntimeItem(
  selection: LegacyShellChatRuntimeItemActionDetail,
): boolean {
  return callLegacyRuntimeAction(window.__alter0LegacyRuntime?.toggleChatRuntimeItem, [selection]) ?? false;
}
