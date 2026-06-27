import {
  useCallback,
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { createAPIClient } from "../../shared/api/client";
import { hashSessionIDShort, resolveSessionIDReference } from "../../shared/session/sessionHash";
import { formatDateTimeMinute } from "../../shared/time/format";
import type { LegacyShellLanguage } from "../shell/legacyShellCopy";
import { readWorkbenchRouteSessionID, writeWorkbenchRouteSessionID } from "../../app/routeState";
import { MOBILE_VIEWPORT_BREAKPOINT_PX } from "../../shared/viewport/mobileViewport";
import { usePageActivation } from "../../shared/visibility/usePageActivation";
import { runtimeSessionEndpoint } from "../shell/components/runtimeSessionApi";
import {
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  isComposerImageAttachment,
  type ComposerAttachment,
} from "./composerImageAttachments";
import {
  DEFAULT_RUNTIME_EVENT_FILTER,
  RUNTIME_EVENT_FILTER_OPTIONS,
  normalizeRuntimeTraceEvents,
  runtimeTraceEventDetailID,
  type RuntimeEventFilterID,
  type RuntimeBlock,
  type RuntimeTraceEvent,
} from "../shell/components/runtimeTraceEvents";

const ACTIVE_SESSION_STORAGE_KEY = "alter0.web.session.active.v1";
const ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.snapshot.v1";
const RECENT_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.recent.v1";
const LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.long_term_snapshot.v1";
const SESSION_INFO_SNAPSHOT_STORAGE_KEY = "alter0.web.session.info_snapshot.v1";
const COMPOSER_DRAFT_STORAGE_KEY = "alter0.web.composer.drafts.v1";
const COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY = "alter0.web.composer.attachments.v1";
const RUNTIME_EVENT_FILTER_STORAGE_KEY = "alter0.web.runtime.event_filter.v1";
const COMPOSER_DRAFT_PERSIST_DELAY_MS = 160;
const NEW_CHAT_DRAFT_KEY = "__chat_new__";
const MAX_COMPOSER_CHARS = 10000;
const CHAT_SESSION_RECOVERY_POLL_INTERVAL_MS = 3000;
const CODEX_RUNTIME_PROVIDER_ID = "alter0-codex";
const CODEX_RUNTIME_MODEL_ID = "codex";
const CANONICAL_CHAT_SESSION_ID = "alter0-chat";
const MAX_RECENT_SESSION_SNAPSHOTS = 12;
const PAGE_ACTIVE_REFRESH_DEBOUNCE_MS = 400;
const CHAT_HISTORY_PAGE_TURN_LIMIT = 20;
export const CHAT_RUNTIME_CACHE_SESSION_TTL_MS = 24 * 60 * 60 * 1000;
const CHAT_LONG_TERM_CACHE_TTL_MS = 24 * 60 * 60 * 1000;

function chatTerminalSessionEndpoint(
  path: string = "",
  query: Record<string, string | number | boolean | undefined> = {},
): string {
  return runtimeSessionEndpoint("chat", path, query);
}

type ChatSessionPollPlan = {
  enabled: boolean;
  interval: number;
};

export function resolveChatSessionPollPlan(options: { sessionCount: number; pageHidden: boolean }): ChatSessionPollPlan {
  if (options.sessionCount <= 0 || options.pageHidden) {
    return {
      enabled: false,
      interval: 0,
    };
  }
  return {
    enabled: true,
    interval: CHAT_SESSION_RECOVERY_POLL_INTERVAL_MS,
  };
}

export type ConversationRoute = "chat";

type ChatTarget = {
  type: "model";
  id: string;
  name: string;
};

export type ChatMessage = {
  id: string;
  role: "user" | "assistant";
  text: string;
  attachments: ComposerAttachment[];
  route: string;
  source: string;
  error: boolean;
  status: string;
  at: number;
  processEvents: RuntimeTraceEvent[];
  processCollapsed?: boolean;
};

type ChatSession = {
  id: string;
  sourceRoute?: ConversationRoute;
  status: string;
  title: string;
  titleAuto: boolean;
  titleScore: number;
  createdAt: number;
  pinned: boolean;
  target: ChatTarget;
  modelProviderID: string;
  modelID: string;
  toolIDs: string[];
  skillIDs: string[];
  skillIDsExplicit?: boolean;
  mcpIDs: string[];
  messages: ChatMessage[];
  messagesLoaded?: boolean;
  serverBacked?: boolean;
  turnsPaging?: TerminalTurnPaging;
};

type ChatProviderModel = {
  id: string;
  name: string;
  is_enabled?: boolean;
  supports_vision?: boolean;
};

type ChatProvider = {
  id: string;
  name: string;
  is_enabled?: boolean;
  is_default?: boolean;
  default_model?: string;
  models?: ChatProviderModel[];
};

type ChatCapability = {
  id: string;
  name: string;
  description?: string;
  scope?: string;
  enabled?: boolean;
  metadata?: Record<string, string>;
};

type ActiveSessionState = Record<ConversationRoute, string>;
type SessionsState = Record<ConversationRoute, ChatSession[]>;
type ComposerDraftMap = Record<string, string>;
type ComposerAttachmentDraftMap = Record<string, ComposerAttachment[]>;
type StoredActiveSessionSnapshotState = Record<string, unknown>;
type StoredRecentSessionSnapshotState = Record<string, unknown>;

type ConversationRuntimeCacheSnapshot = {
  cachedAt: number;
  activeSessionByRoute: ActiveSessionState;
  sessionsByRoute: SessionsState;
};

type ConversationRuntimeInitialState = {
  activeSessionByRoute: ActiveSessionState;
  sessionsByRoute: SessionsState;
};

let conversationRuntimeCache: ConversationRuntimeCacheSnapshot | null = null;

export function resetConversationRuntimeCache() {
  conversationRuntimeCache = null;
}

type RuntimeSelection = {
  id: string;
  name: string;
  description: string;
  kind: "tool" | "mcp" | "skill";
  active: boolean;
  visibility?: "public";
  locked?: boolean;
};

type RuntimeModel = {
  id: string;
  name: string;
  active: boolean;
  supportsVision: boolean;
};

type RuntimeProvider = {
  id: string;
  name: string;
  models: RuntimeModel[];
};

type SessionAttachmentUploadResponse = {
  items?: Array<{
    id?: string;
    name?: string;
    content_type?: string;
    size?: number;
    asset_url?: string;
    preview_url?: string;
  }>;
};

type TerminalSessionPayload = {
  id?: string;
  status?: string;
  title?: string;
  pinned?: boolean;
  created_at?: string | number;
  updated_at?: string | number;
  last_output_at?: string | number;
  model_provider_id?: string;
  model_id?: string;
  tool_ids?: string[];
  skill_ids?: string[];
  mcp_ids?: string[];
  turns?: TerminalTurnPayload[];
  turns_paging?: TerminalTurnPaging;
};

type TerminalTurnPaging = {
  limit?: number;
  total?: number;
  byte_limit?: number;
  approx_bytes?: number;
  has_more_before?: boolean;
  has_more_after?: boolean;
  oldest_turn_id?: string;
  newest_turn_id?: string;
  next_before_turn_id?: string;
};

type TerminalTurnPayload = {
  id?: string;
  prompt?: string;
  attachments?: TerminalAttachmentPayload[];
  status?: string;
  started_at?: string | number;
  finished_at?: string | number;
  final_output?: string;
  runtime_trace_events?: RuntimeTraceEvent[];
};

type TerminalAttachmentPayload = {
  id?: string;
  name?: string;
  content_type?: string;
  asset_url?: string;
  preview_url?: string;
};

type RuntimeTraceEventDetail = {
  turn_id?: string;
  event?: RuntimeTraceEvent;
  blocks?: RuntimeBlock[];
};

type RuntimeTraceEventDetailResponse = {
  event?: RuntimeTraceEventDetail;
};

type ConversationRuntimeContextValue = {
  route: ConversationRoute;
  compact: boolean;
  inspectorOpen: boolean;
  inspectorTab: "model" | "capabilities" | "skills";
  inspectorTabOpen: boolean;
  sessions: ChatSession[];
  activeSession: ChatSession | null;
  sessionItems: Array<{
    id: string;
    title: string;
    contextLabel?: string;
    meta: string;
    shortHash: string;
    createdAt: number;
    active: boolean;
    draft: boolean;
    pinned: boolean;
    pinning: boolean;
  }>;
  busy: boolean;
  draft: string;
  target: ChatTarget;
  lockedTarget: boolean;
  selectedProviderId: string;
  selectedModelId: string;
  selectedModelLabel: string;
  selectedModelSupportsVision: boolean;
  providers: RuntimeProvider[];
  draftAttachments: ComposerAttachment[];
  capabilities: RuntimeSelection[];
  skills: RuntimeSelection[];
  runtimeEventFilter: RuntimeEventFilterID[];
  toolCount: number;
  skillCount: number;
  createSession: () => void;
  focusSession: (sessionID: string) => void;
  removeSession: (sessionID: string) => Promise<void>;
  setSessionPinned: (sessionID: string, pinned: boolean) => Promise<void>;
  refreshActiveSession: () => Promise<void>;
  setDraft: (value: string) => void;
  addDraftAttachments: (attachments: ComposerAttachment[]) => Promise<void>;
  removeDraftAttachment: (attachmentID: string) => void;
  clearDraftAttachments: () => void;
  sendPrompt: (prompt?: string) => Promise<void>;
  toggleInspector: (tab?: "model" | "capabilities" | "skills") => void;
  closeInspector: () => void;
  selectModel: (providerID: string, modelID: string) => void;
  toggleCapability: (id: string, kind: "tool" | "mcp", checked: boolean) => void;
  toggleSkill: (id: string, checked: boolean) => void;
  toggleRuntimeEventFilter: (id: RuntimeEventFilterID, checked: boolean) => void;
  toggleProcess: (messageID: string) => void;
  loadProcessEventDetail: (messageID: string, eventID: string) => Promise<void>;
};

type ConversationRuntimeWorkspaceContextValue = Omit<
  ConversationRuntimeContextValue,
  "draft"
  | "draftAttachments"
  | "setDraft"
  | "addDraftAttachments"
  | "removeDraftAttachment"
  | "clearDraftAttachments"
  | "sendPrompt"
>;

type ConversationRuntimeComposerContextValue = Pick<
  ConversationRuntimeContextValue,
  "route"
  | "draft"
  | "draftAttachments"
  | "busy"
  | "selectedModelSupportsVision"
  | "setDraft"
  | "addDraftAttachments"
  | "removeDraftAttachment"
  | "clearDraftAttachments"
  | "sendPrompt"
>;

const ConversationRuntimeWorkspaceContext = createContext<ConversationRuntimeWorkspaceContextValue | null>(null);
const ConversationRuntimeComposerContext = createContext<ConversationRuntimeComposerContextValue | null>(null);

type ProviderProps = {
  route: string;
  language: LegacyShellLanguage;
  children: ReactNode;
};

type RuntimeRecoveryRequirement = {
  requireMessages?: boolean;
  requireStableAssistant?: boolean;
};

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeRuntimeEventFilter(value: unknown): RuntimeEventFilterID[] {
  const allowed = new Set(RUNTIME_EVENT_FILTER_OPTIONS.map((option) => option.id));
  const items = Array.isArray(value) ? value : [];
  const normalized = items.filter((item): item is RuntimeEventFilterID =>
    typeof item === "string" && allowed.has(item as RuntimeEventFilterID),
  );
  return normalized.length > 0 ? normalized : [...DEFAULT_RUNTIME_EVENT_FILTER];
}

function loadRuntimeEventFilter(): RuntimeEventFilterID[] {
  if (typeof window === "undefined") {
    return [...DEFAULT_RUNTIME_EVENT_FILTER];
  }
  try {
    return normalizeRuntimeEventFilter(JSON.parse(window.localStorage.getItem(RUNTIME_EVENT_FILTER_STORAGE_KEY) || "null"));
  } catch {
    return [...DEFAULT_RUNTIME_EVENT_FILTER];
  }
}

function persistRuntimeEventFilter(filter: RuntimeEventFilterID[]) {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(RUNTIME_EVENT_FILTER_STORAGE_KEY, JSON.stringify(normalizeRuntimeEventFilter(filter)));
}

function normalizeConversationRoute(_route: string): ConversationRoute {
  return "chat";
}

function isPublicSkillCapability(skill: ChatCapability): boolean {
  const metadata = skill.metadata || {};
  const visibility = normalizeText(metadata["alter0.skill.visibility"] || metadata["skill.visibility"]).toLowerCase();
  return visibility !== "private" && visibility !== "private";
}

function defaultChatSkillIDs(skills: ChatCapability[]): string[] {
  return normalizeSelectionIDs(
    skills
      .filter((item) => item.enabled !== false && isPublicSkillCapability(item))
      .map((item) => normalizeText(item.id)),
  );
}

function effectiveChatSkillIDs(selectedIDs: string[] | undefined, availableSkillIDs: string[] | null): string[] {
  const normalized = normalizeSelectionIDs(selectedIDs || []);
  if (availableSkillIDs === null) {
    return normalized;
  }
  const available = new Set(availableSkillIDs);
  return normalized.filter((item) => available.has(item));
}

function effectiveSessionSkillIDs(session: ChatSession | null | undefined, availableSkillIDs: string[] | null): string[] {
  if (!session) {
    return availableSkillIDs === null ? [] : [...availableSkillIDs];
  }
  if (session.skillIDsExplicit === false && availableSkillIDs !== null) {
    return [...availableSkillIDs];
  }
  return effectiveChatSkillIDs(session.skillIDs, availableSkillIDs);
}

function makeID(prefix: string): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

function normalizeChatTarget(target?: { type?: string; id?: string; name?: string } | null): ChatTarget {
  return {
    type: "model",
    id: normalizeText(target?.id) || "raw-model",
    name: normalizeText(target?.name) || "Raw Model",
  };
}

function defaultChatTarget(): ChatTarget {
  return normalizeChatTarget({ type: "model", id: "raw-model", name: "Raw Model" });
}

function compareSessions(left: ChatSession, right: ChatSession): number {
  if (left.pinned !== right.pinned) {
    return left.pinned ? -1 : 1;
  }
  return right.createdAt - left.createdAt;
}

function isBlankDraftSession(session: ChatSession): boolean {
  return session.serverBacked !== true && session.messages.length === 0;
}

function normalizeRouteSessions(routeKey: ConversationRoute, sessions: ChatSession[]): ChatSession[] {
  void routeKey;
  const merged = new Map<string, ChatSession>();
  sessions.forEach((session) => {
    merged.set(session.id, session);
  });
  return Array.from(merged.values()).sort(compareSessions);
}

function cloneRuntimeTraceEvent(event: RuntimeTraceEvent): RuntimeTraceEvent {
  return {
    ...event,
    provider: { ...event.provider },
    blocks: event.blocks.map((block) => ({ ...block })),
    action: event.action
      ? {
          ...event.action,
          permission: event.action.permission ? { ...event.action.permission } : undefined,
        }
      : undefined,
    error: event.error ? { ...event.error } : undefined,
    raw: event.raw ? { ...event.raw } : undefined,
  };
}

function cloneChatMessage(message: ChatMessage): ChatMessage {
  return {
    ...message,
    attachments: message.attachments.map((attachment) => ({ ...attachment })),
    processEvents: message.processEvents.map(cloneRuntimeTraceEvent),
  };
}

function cloneChatSession(session: ChatSession): ChatSession {
  return {
    ...session,
    target: { ...session.target },
    toolIDs: [...session.toolIDs],
    skillIDs: [...session.skillIDs],
    mcpIDs: [...session.mcpIDs],
    messages: session.messages.map(cloneChatMessage),
    turnsPaging: session.turnsPaging ? { ...session.turnsPaging } : undefined,
  };
}

function cloneSessionsState(sessionsByRoute: SessionsState): SessionsState {
  return {
    chat: normalizeRouteSessions("chat", sessionsByRoute.chat.map(cloneChatSession)),
  };
}

function codexRuntimeProvider(): ChatProvider {
  return {
    id: CODEX_RUNTIME_PROVIDER_ID,
    name: "Codex",
    default_model: CODEX_RUNTIME_MODEL_ID,
    models: [
      {
        id: CODEX_RUNTIME_MODEL_ID,
        name: "Codex",
        is_enabled: true,
        supports_vision: true,
      },
    ],
  };
}

function runtimeProviders(providers: ChatProvider[]): ChatProvider[] {
  if (providers.some((provider) => normalizeText(provider.id) === CODEX_RUNTIME_PROVIDER_ID)) {
    return providers;
  }
  return [...providers, codexRuntimeProvider()];
}

function normalizeSelectionIDs(values: unknown): string[] {
  if (!Array.isArray(values)) {
    return [];
  }
  return Array.from(new Set(values.map((item) => normalizeText(item)).filter(Boolean)));
}

function normalizeOptionalNumber(value: unknown): number | undefined {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function normalizeTerminalTurnPaging(value: unknown): TerminalTurnPaging | undefined {
  if (!value || typeof value !== "object") {
    return undefined;
  }
  const record = value as Record<string, unknown>;
  return {
    limit: normalizeOptionalNumber(record.limit),
    total: normalizeOptionalNumber(record.total),
    byte_limit: normalizeOptionalNumber(record.byte_limit),
    approx_bytes: normalizeOptionalNumber(record.approx_bytes),
    has_more_before: typeof record.has_more_before === "boolean" ? record.has_more_before : undefined,
    has_more_after: typeof record.has_more_after === "boolean" ? record.has_more_after : undefined,
    oldest_turn_id: normalizeText(record.oldest_turn_id),
    newest_turn_id: normalizeText(record.newest_turn_id),
    next_before_turn_id: normalizeText(record.next_before_turn_id),
  };
}

function messageTurnID(messageID: string): string {
  const normalized = normalizeText(messageID);
  const separatorIndex = normalized.lastIndexOf(":");
  return separatorIndex > 0 ? normalized.slice(0, separatorIndex) : "";
}

function sessionHasTurn(session: ChatSession | null | undefined, turnID: string): boolean {
  const normalizedTurnID = normalizeText(turnID);
  return Boolean(normalizedTurnID && session?.messages.some((message) => messageTurnID(message.id) === normalizedTurnID));
}

function oldestTurnIDFromMessages(messages: ChatMessage[]): string {
  for (const message of messages) {
    const turnID = messageTurnID(message.id);
    if (turnID) {
      return turnID;
    }
  }
  return "";
}

function isPagedTerminalTurnPayload(
  item: TerminalSessionPayload,
  paging: TerminalTurnPaging | undefined,
): boolean {
  if (!paging || !Array.isArray(item.turns)) {
    return false;
  }
  if (paging.has_more_before || paging.has_more_after) {
    return true;
  }
  return typeof paging.total === "number" && item.turns.length < paging.total;
}

function mergeTerminalTurnPaging(
  previous: ChatSession | null | undefined,
  incoming: TerminalTurnPaging | undefined,
): TerminalTurnPaging | undefined {
  if (!incoming) {
    return previous?.turnsPaging ? { ...previous.turnsPaging } : undefined;
  }
  const next = { ...incoming };
  const boundaryTurnID = normalizeText(next.next_before_turn_id || next.oldest_turn_id);
  if (
    previous?.turnsPaging?.has_more_before === false
    && next.has_more_before === true
    && sessionHasTurn(previous, boundaryTurnID)
  ) {
    next.has_more_before = false;
  }
  if (!next.next_before_turn_id) {
    next.next_before_turn_id = next.oldest_turn_id || oldestTurnIDFromMessages(previous?.messages || []);
  }
  return next;
}

function trimSessionForInfoCache(session: ChatSession): ChatSession {
  return {
    ...cloneChatSession(session),
    messages: [],
    messagesLoaded: false,
  };
}

function isStreamingPlaceholderText(text: string): boolean {
  const normalized = normalizeText(text).toLowerCase();
  return normalized === "" || normalized === "thinking...";
}

function isRecoverableAssistantMessage(message: ChatMessage): boolean {
  if (message.role !== "assistant") {
    return false;
  }
  return message.error || normalizeTaskStatus(message.status) === "streaming" || isStreamingPlaceholderText(message.text);
}

function hasRecoverableAssistantState(messages: ChatMessage[]): boolean {
  return messages.some((message) => isRecoverableAssistantMessage(message));
}

function hasUnansweredLatestUserMessage(messages: ChatMessage[]): boolean {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message.role === "assistant") {
      return false;
    }
    if (message.role === "user") {
      return true;
    }
  }
  return false;
}

function hasRecoverableRuntimeState(session: ChatSession | null | undefined): boolean {
  if (!session) {
    return false;
  }
  return hasRecoverableAssistantState(session.messages) || hasUnansweredLatestUserMessage(session.messages);
}

function shouldPollTerminalBackedSession(session: ChatSession): boolean {
  if (session.serverBacked !== true) {
    return false;
  }
  return isConversationBusyStatus(session.status) || hasRecoverableRuntimeState(session);
}

function hasPersistedAssistantState(messages: ChatMessage[]): boolean {
  return messages.some((message) => {
    if (message.role !== "assistant") {
      return false;
    }
    return normalizeTaskStatus(message.status) !== "streaming" && !isStreamingPlaceholderText(message.text);
  });
}

function shouldUseParsedMessages(previous: ChatMessage[], parsed: ChatMessage[]): boolean {
  if (parsed.length >= previous.length) {
    return true;
  }
  if (parsed.length < Math.max(0, previous.length - 1)) {
    return false;
  }
  return hasRecoverableAssistantState(previous) && hasPersistedAssistantState(parsed);
}

function mergePagedMessages(previous: ChatMessage[], parsed: ChatMessage[]): ChatMessage[] {
  if (previous.length === 0) {
    return parsed;
  }
  if (parsed.length === 0) {
    return previous;
  }
  const previousIDs = new Set(previous.map((message) => message.id));
  const hasOverlap = parsed.some((message) => previousIDs.has(message.id));
  if (!hasOverlap) {
    const previousFirstAt = previous[0]?.at || 0;
    const previousLastAt = previous[previous.length - 1]?.at || 0;
    const parsedFirstAt = parsed[0]?.at || 0;
    const parsedLastAt = parsed[parsed.length - 1]?.at || 0;
    if (
      (parsedLastAt > 0 && previousFirstAt > 0 && parsedLastAt < previousFirstAt)
      || (parsedFirstAt > 0 && previousLastAt > 0 && parsedFirstAt > previousLastAt)
    ) {
      return [...previous, ...parsed].sort((left, right) => left.at - right.at);
    }
    return shouldUseParsedMessages(previous, parsed) ? parsed : previous;
  }
  const merged = new Map<string, ChatMessage>();
  previous.forEach((message) => merged.set(message.id, message));
  parsed.forEach((message) => merged.set(message.id, message));
  return Array.from(merged.values()).sort((left, right) => left.at - right.at);
}

function normalizeStoredMessage(item: unknown): ChatMessage | null {
  if (!item || typeof item !== "object") {
    return null;
  }
  const record = item as Record<string, unknown>;
  const id = normalizeText(record.id);
  if (!id) {
    return null;
  }
  const role = normalizeText(record.role) === "assistant" ? "assistant" : "user";
  return {
    id,
    role,
    text: typeof record.text === "string" ? record.text : "",
    attachments: normalizeStoredAttachments(record.attachments),
    route: normalizeText(record.route),
    source: normalizeText(record.source),
    error: Boolean(record.error),
    status: normalizeText(record.status) || (role === "assistant" ? "done" : ""),
    at: Number.isFinite(Number(record.at)) ? Number(record.at) : Date.now(),
    processEvents: normalizeRuntimeTraceEvents(record.runtime_trace_events),
    processCollapsed:
      typeof record.process_collapsed === "boolean"
        ? record.process_collapsed
        : undefined,
  };
}

function resolveProcessCollapsed(message: ChatMessage): boolean {
  if (typeof message.processCollapsed === "boolean") {
    return message.processCollapsed;
  }
  return Boolean(message.text.trim()) && normalizeTaskStatus(message.status) !== "streaming";
}

function normalizeStoredAttachments(value: unknown): ComposerAttachment[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .map((item) => {
      if (!item || typeof item !== "object") {
        return null;
      }
      const record = item as Record<string, unknown>;
      const id = normalizeText(record.id);
      const dataURL = normalizeText(record.data_url ?? record.dataURL);
      const previewDataURL = normalizeText(record.preview_data_url ?? record.previewDataURL);
      const assetURL = normalizeText(record.asset_url ?? record.assetURL);
      const previewURL = normalizeText(record.preview_url ?? record.previewURL);
      const contentType = normalizeText(record.content_type ?? record.contentType);
      if (!id || !contentType || (!dataURL && !assetURL && !previewURL)) {
        return null;
      }
      const kind = normalizeText(record.kind) === "file" || !contentType.startsWith("image/")
        ? "file"
        : "image";
      return {
        id,
        kind,
        name: normalizeText(record.name) || (kind === "image" ? "image" : "file"),
        contentType,
        size: Number.isFinite(Number(record.size)) ? Number(record.size) : 0,
        dataURL: dataURL || undefined,
        previewDataURL: kind === "image" ? previewDataURL || undefined : undefined,
        assetURL: assetURL || undefined,
        previewURL: kind === "image" ? previewURL || undefined : undefined,
      };
    })
    .filter((item): item is ComposerAttachment => item !== null);
}

function normalizeStoredSession(item: unknown): ChatSession | null {
  if (!item || typeof item !== "object") {
    return null;
  }
  const record = item as Record<string, unknown>;
  const id = normalizeText(record.id);
  if (!id) {
    return null;
  }
  const targetRecord = record.target && typeof record.target === "object"
    ? record.target as Record<string, unknown>
    : {};
  return {
    id,
    sourceRoute: "chat",
    status: normalizeText(record.status),
    title: normalizeText(record.title) || "New",
    titleAuto: record.titleAuto !== false,
    titleScore: Number.isFinite(Number(record.titleScore)) ? Number(record.titleScore) : 0,
    createdAt: Number.isFinite(Number(record.createdAt)) ? Number(record.createdAt) : Date.now(),
    pinned: record.pinned === true,
    target: normalizeChatTarget({
      type: "model",
      id: normalizeText(record.targetID ?? targetRecord.id),
      name: normalizeText(record.targetName ?? targetRecord.name),
    }),
    modelProviderID: normalizeText(record.modelProviderID),
    modelID: normalizeText(record.modelID),
    toolIDs: normalizeSelectionIDs(record.toolIDs),
    skillIDs: normalizeSelectionIDs(record.skillIDs),
    skillIDsExplicit: record.skillIDsExplicit === true,
    mcpIDs: normalizeSelectionIDs(record.mcpIDs),
    messages: Array.isArray(record.messages)
      ? record.messages.map(normalizeStoredMessage).filter((message): message is ChatMessage => message !== null)
      : [],
    messagesLoaded: typeof record.messagesLoaded === "boolean" ? record.messagesLoaded : undefined,
    serverBacked: typeof record.serverBacked === "boolean" ? record.serverBacked : undefined,
    turnsPaging: normalizeTerminalTurnPaging(record.turnsPaging ?? record.turns_paging),
  };
}

function normalizeCachedSessionsState(value: unknown): SessionsState {
  if (!value || typeof value !== "object") {
    return { chat: [] };
  }
  const record = value as Record<string, unknown>;
  return {
    chat: normalizeRouteSessions("chat", normalizeStoredSessionList(record.chat)),
  };
}

function normalizeStoredSessionList(value: unknown): ChatSession[] {
  if (!Array.isArray(value)) {
    return [];
  }
  const merged = new Map<string, ChatSession>();
  value.forEach((item) => {
    const session = normalizeStoredSession(item);
    if (!session) {
      return;
    }
    merged.set(session.id, session);
  });
  return Array.from(merged.values()).sort(compareSessions);
}

function serializeStoredMessage(message: ChatMessage): Record<string, unknown> {
  return {
    id: message.id,
    role: message.role,
    text: message.text,
    attachments: message.attachments,
    route: message.route,
    source: message.source,
    error: message.error,
    status: message.status,
    at: message.at,
    runtime_trace_events: message.processEvents,
    process_collapsed: message.processCollapsed,
  };
}

function serializeStoredSession(session: ChatSession): Record<string, unknown> {
  return {
    id: session.id,
    sourceRoute: session.sourceRoute,
    status: session.status,
    title: session.title,
    titleAuto: session.titleAuto,
    titleScore: session.titleScore,
    createdAt: session.createdAt,
    pinned: session.pinned,
    targetType: session.target.type,
    targetID: session.target.id,
    targetName: session.target.name,
    modelProviderID: session.modelProviderID,
    modelID: session.modelID,
    toolIDs: session.toolIDs,
    skillIDs: session.skillIDs,
    skillIDsExplicit: session.skillIDsExplicit === true,
    mcpIDs: session.mcpIDs,
    messages: session.messages.map(serializeStoredMessage),
    messagesLoaded: session.messagesLoaded,
    serverBacked: session.serverBacked,
    turnsPaging: session.turnsPaging,
  };
}

function readJSONStorage<T>(key: string, fallback: T): T {
  try {
    const raw = window.sessionStorage.getItem(key);
    if (!raw) {
      return fallback;
    }
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function writeJSONStorage(key: string, value: unknown) {
  try {
    window.sessionStorage.setItem(key, JSON.stringify(value));
  } catch {
  }
}

function readJSONLocalStorage<T>(key: string, fallback: T): T {
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) {
      return fallback;
    }
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function writeJSONLocalStorage(key: string, value: unknown) {
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
  }
}

function loadActiveSessionState(fallback?: ActiveSessionState | null): ActiveSessionState {
  const parsed = readJSONStorage<Record<string, string>>(ACTIVE_SESSION_STORAGE_KEY, {});
  return {
    chat:
      readWorkbenchRouteSessionID("chat")
      || normalizeText(parsed.chat)
      || normalizeText(parsed["chat"])
      || normalizeText(fallback?.chat)
      || CANONICAL_CHAT_SESSION_ID,
  };
}

function loadActiveSessionSnapshots(fallback?: SessionsState | null): SessionsState {
  const parsedActive = readJSONStorage<StoredActiveSessionSnapshotState>(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY, {});
  const parsedRecent = readJSONStorage<StoredRecentSessionSnapshotState>(RECENT_SESSION_SNAPSHOT_STORAGE_KEY, {});
  const mergeStoredRouteSessions = (routeKey: string) => {
    const sessions = new Map<string, ChatSession>();
    (fallback?.chat || []).forEach((session) => {
      sessions.set(session.id, { ...cloneChatSession(session), sourceRoute: "chat" });
    });
    normalizeStoredSessionList(parsedRecent[routeKey]).forEach((session) => {
      sessions.set(session.id, { ...session, sourceRoute: "chat" });
    });
    const active = normalizeStoredSession(parsedActive[routeKey]);
    if (active) {
      sessions.set(active.id, { ...active, sourceRoute: "chat" });
    }
    return normalizeRouteSessions(
      "chat",
      Array.from(sessions.values()).sort(compareSessions),
    );
  };
  return {
    chat: normalizeRouteSessions("chat", [
      ...mergeStoredRouteSessions("chat"),
      ...mergeStoredRouteSessions("chat"),
    ]),
  };
}

function persistActiveSessionSnapshots(activeState: ActiveSessionState, sessions: SessionsState) {
  const payload: StoredActiveSessionSnapshotState = {};
  const recentPayload: StoredRecentSessionSnapshotState = {};
  (["chat"] as ConversationRoute[]).forEach((routeKey) => {
    const activeID = normalizeText(activeState[routeKey]);
    recentPayload[routeKey] = sessions[routeKey]
      .slice(0, MAX_RECENT_SESSION_SNAPSHOTS)
      .map((session) => serializeStoredSession(session));
    if (!activeID) {
      return;
    }
    const session = sessions[routeKey].find((item) => item.id === activeID);
    if (!session) {
      return;
    }
    payload[routeKey] = serializeStoredSession(session);
  });
  writeJSONStorage(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY, payload);
  writeJSONStorage(RECENT_SESSION_SNAPSHOT_STORAGE_KEY, recentPayload);
}

function readConversationRuntimeCache(): ConversationRuntimeCacheSnapshot | null {
  if (!conversationRuntimeCache) {
    return null;
  }
  if (Date.now() - conversationRuntimeCache.cachedAt > CHAT_RUNTIME_CACHE_SESSION_TTL_MS) {
    conversationRuntimeCache = null;
    return null;
  }
  return {
    cachedAt: conversationRuntimeCache.cachedAt,
    activeSessionByRoute: { ...conversationRuntimeCache.activeSessionByRoute },
    sessionsByRoute: cloneSessionsState(conversationRuntimeCache.sessionsByRoute),
  };
}

function readLongTermConversationRuntimeCache(): ConversationRuntimeCacheSnapshot | null {
  const cache = readJSONLocalStorage<ConversationRuntimeCacheSnapshot | null>(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY, null);
  if (!cache) {
    return null;
  }
  if (Date.now() - Number(cache.cachedAt || 0) > CHAT_LONG_TERM_CACHE_TTL_MS) {
    try {
      window.localStorage.removeItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY);
    } catch {
    }
    return null;
  }
  return {
    cachedAt: cache.cachedAt,
    activeSessionByRoute: { chat: normalizeText(cache.activeSessionByRoute?.chat) },
    sessionsByRoute: normalizeCachedSessionsState(cache.sessionsByRoute),
  };
}

function readSessionInfoConversationRuntimeCache(): ConversationRuntimeCacheSnapshot | null {
  const cache = readJSONLocalStorage<ConversationRuntimeCacheSnapshot | null>(SESSION_INFO_SNAPSHOT_STORAGE_KEY, null);
  if (!cache) {
    return null;
  }
  if (Date.now() - Number(cache.cachedAt || 0) > CHAT_LONG_TERM_CACHE_TTL_MS) {
    try {
      window.localStorage.removeItem(SESSION_INFO_SNAPSHOT_STORAGE_KEY);
    } catch {
    }
    return null;
  }
  return {
    cachedAt: cache.cachedAt,
    activeSessionByRoute: { chat: normalizeText(cache.activeSessionByRoute?.chat) },
    sessionsByRoute: normalizeCachedSessionsState(cache.sessionsByRoute),
  };
}

function writeConversationRuntimeCache(activeSessionByRoute: ActiveSessionState, sessionsByRoute: SessionsState) {
  conversationRuntimeCache = {
    cachedAt: Date.now(),
    activeSessionByRoute: { chat: normalizeText(activeSessionByRoute.chat) },
    sessionsByRoute: cloneSessionsState(sessionsByRoute),
  };
}

function writeLongTermConversationRuntimeCache(activeSessionByRoute: ActiveSessionState, sessionsByRoute: SessionsState) {
  writeJSONLocalStorage(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY, {
    cachedAt: Date.now(),
    activeSessionByRoute: { chat: normalizeText(activeSessionByRoute.chat) },
    sessionsByRoute: cloneSessionsState(sessionsByRoute),
  });
}

function writeSessionInfoConversationRuntimeCache(activeSessionByRoute: ActiveSessionState, sessionsByRoute: SessionsState) {
  writeJSONLocalStorage(SESSION_INFO_SNAPSHOT_STORAGE_KEY, {
    cachedAt: Date.now(),
    activeSessionByRoute: { chat: normalizeText(activeSessionByRoute.chat) },
    sessionsByRoute: {
      chat: normalizeRouteSessions("chat", sessionsByRoute.chat.map(trimSessionForInfoCache)),
    },
  });
}

function resolveInitialConversationRuntimeState(): ConversationRuntimeInitialState {
  const cache = readConversationRuntimeCache();
  if (cache) {
    return {
      activeSessionByRoute: cache.activeSessionByRoute,
      sessionsByRoute: cache.sessionsByRoute,
    };
  }
  const longTermCache = readLongTermConversationRuntimeCache();
  const infoCache = longTermCache ? null : readSessionInfoConversationRuntimeCache();
  const fallbackCache = longTermCache || infoCache;
  return {
    activeSessionByRoute: loadActiveSessionState(fallbackCache?.activeSessionByRoute || null),
    sessionsByRoute: loadActiveSessionSnapshots(fallbackCache?.sessionsByRoute || null),
  };
}

function loadComposerDrafts(): ComposerDraftMap {
  const parsed = readJSONStorage<Record<string, string>>(COMPOSER_DRAFT_STORAGE_KEY, {});
  return Object.entries(parsed).reduce<ComposerDraftMap>((acc, [key, value]) => {
    const normalizedKey = normalizeText(key);
    if (!normalizedKey || typeof value !== "string") {
      return acc;
    }
    acc[normalizedKey] = value;
    return acc;
  }, {});
}

function persistComposerDrafts(drafts: ComposerDraftMap) {
  writeJSONStorage(COMPOSER_DRAFT_STORAGE_KEY, drafts);
}

function loadComposerAttachmentDrafts(): ComposerAttachmentDraftMap {
  const parsed = readJSONStorage<Record<string, unknown>>(COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY, {});
  return Object.entries(parsed).reduce<ComposerAttachmentDraftMap>((acc, [key, value]) => {
    const normalizedKey = normalizeText(key);
    if (!normalizedKey) {
      return acc;
    }
    const attachments = normalizeStoredAttachments(value);
    if (attachments.length > 0) {
      acc[normalizedKey] = attachments;
    }
    return acc;
  }, {});
}

function persistComposerAttachmentDrafts(drafts: ComposerAttachmentDraftMap) {
  writeJSONStorage(COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY, drafts);
}

function normalizeDateValue(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim()) {
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return Date.now();
}

function normalizeTerminalAttachment(item: TerminalAttachmentPayload): ComposerAttachment | null {
  const id = normalizeText(item.id);
  const contentType = normalizeText(item.content_type);
  const assetURL = normalizeText(item.asset_url);
  const previewURL = normalizeText(item.preview_url);
  if (!id || !contentType || !assetURL) {
    return null;
  }
  const kind = contentType.startsWith("image/") ? "image" as const : "file" as const;
  return {
    id,
    kind,
    name: normalizeText(item.name) || (kind === "image" ? "image" : "file"),
    contentType,
    size: 0,
    assetURL,
    previewURL: kind === "image" ? previewURL || assetURL : undefined,
  };
}

function normalizeTerminalProcessEvents(sessionID: string, turnID: string, turn: TerminalTurnPayload): RuntimeTraceEvent[] {
  return normalizeRuntimeTraceEvents(turn.runtime_trace_events, { sessionID, turnID });
}

function normalizeRuntimeTraceEventDetail(
  detail: RuntimeTraceEventDetail | undefined,
  fallback: RuntimeTraceEvent,
): RuntimeTraceEvent | null {
  if (!detail?.event) {
    return null;
  }
  const normalized = normalizeRuntimeTraceEvents([detail.event], {
    sessionID: fallback.session_id,
    turnID: fallback.turn_id,
  })[0];
  if (!normalized) {
    return null;
  }
  const blocks = normalized.blocks.length > 0
    ? normalized.blocks
    : Array.isArray(detail.blocks)
      ? detail.blocks
      : fallback.blocks;
  return {
    ...fallback,
    ...normalized,
    blocks,
    raw: normalized.raw ? { ...normalized.raw, has_detail: false } : fallback.raw ? { ...fallback.raw, has_detail: false } : undefined,
  };
}

function normalizeTerminalTurnMessages(sessionID: string, turn: TerminalTurnPayload): ChatMessage[] {
  const id = normalizeText(turn.id);
  if (!id) {
    return [];
  }
  const at = normalizeDateValue(turn.started_at);
  const prompt = typeof turn.prompt === "string" ? turn.prompt : "";
  const attachments = Array.isArray(turn.attachments)
    ? turn.attachments.map(normalizeTerminalAttachment).filter((item): item is ComposerAttachment => item !== null)
    : [];
  const messages: ChatMessage[] = [];
  if (prompt || attachments.length > 0) {
    messages.push({
      id: `${id}:prompt`,
      role: "user",
      text: prompt,
      attachments,
      route: "",
      source: "",
      error: false,
      status: "",
      at,
      processEvents: [],
    });
  }
  const status = normalizeTaskStatus(turn.status || "");
  const finalOutput = typeof turn.final_output === "string" ? turn.final_output : "";
  const processEvents = normalizeTerminalProcessEvents(sessionID, id, turn);
  if (finalOutput || processEvents.length > 0 || status === "running" || status === "queued") {
    messages.push({
      id: `${id}:response`,
      role: "assistant",
      text: finalOutput,
      attachments: [],
      route: "terminal",
      source: "terminal",
      error: status === "failed" || status === "canceled",
      status: status === "success" ? "done" : status || "done",
      at: normalizeDateValue(turn.finished_at || turn.started_at),
      processEvents,
      processCollapsed: finalOutput ? undefined : false,
    });
  }
  return messages;
}

function normalizeTerminalSession(
  item: TerminalSessionPayload,
  previous?: ChatSession | null,
  sourceRoute: ConversationRoute = "chat",
): ChatSession | null {
  const id = normalizeText(item.id);
  if (!id) {
    return null;
  }
  const parsedMessages = Array.isArray(item.turns)
    ? item.turns.flatMap((turn) => normalizeTerminalTurnMessages(id, turn))
    : null;
  const incomingPaging = normalizeTerminalTurnPaging(item.turns_paging);
  const shouldMergeTerminalMessages = parsedMessages
    && previous?.messages.length
    && (Boolean(incomingPaging) || isPagedTerminalTurnPayload(item, incomingPaging) || parsedMessages.length < previous.messages.length);
  const messages = parsedMessages
    ? (shouldMergeTerminalMessages
      ? mergePagedMessages(previous.messages, parsedMessages)
      : previous?.messages.length && !shouldUseParsedMessages(previous.messages, parsedMessages)
        ? previous.messages
        : parsedMessages)
    : previous?.messages || [];
  const hasExplicitSkillIDs = Array.isArray(item.skill_ids);
  return {
    id,
    sourceRoute: previous?.sourceRoute || sourceRoute,
    status: normalizeText(item.status) || previous?.status || "ready",
    title: normalizeText(item.title) || previous?.title || "New",
    titleAuto: previous?.titleAuto ?? true,
    titleScore: previous?.titleScore || 0,
    createdAt: normalizeDateValue(item.created_at),
    pinned: typeof item.pinned === "boolean" ? item.pinned : previous?.pinned || false,
    target: previous?.target || defaultChatTarget(),
    modelProviderID: normalizeText(item.model_provider_id) || previous?.modelProviderID || "",
    modelID: normalizeText(item.model_id) || previous?.modelID || "",
    toolIDs: normalizeSelectionIDs(item.tool_ids || previous?.toolIDs || []),
    skillIDs: hasExplicitSkillIDs
      ? normalizeSelectionIDs(item.skill_ids)
      : normalizeSelectionIDs(previous?.skillIDs || []),
    skillIDsExplicit: hasExplicitSkillIDs ? true : previous?.skillIDsExplicit === true,
    mcpIDs: normalizeSelectionIDs(item.mcp_ids || previous?.mcpIDs || []),
    messages,
    messagesLoaded: Array.isArray(item.turns),
    serverBacked: true,
    turnsPaging: mergeTerminalTurnPaging(previous, incomingPaging),
  };
}

function mergeRuntimeSessions(remote: ChatSession[], existing: ChatSession[]): ChatSession[] {
  const merged = new Map<string, ChatSession>();
  const existingByID = new Map(existing.map((session) => [session.id, session]));
  remote.forEach((session) => {
    const previous = existingByID.get(session.id);
    const messages = previous && session.messagesLoaded === true && !shouldUseParsedMessages(previous.messages, session.messages)
      ? previous.messages
      : session.messages.length > 0 || session.messagesLoaded === true
        ? session.messages
        : previous?.messages || [];
    merged.set(session.id, {
      ...previous,
      ...session,
      status: session.status || previous?.status || "",
      messages,
      messagesLoaded:
        typeof session.messagesLoaded === "boolean"
          ? session.messagesLoaded
          : previous?.messagesLoaded,
      serverBacked:
        typeof session.serverBacked === "boolean"
          ? session.serverBacked
          : previous?.serverBacked,
      turnsPaging: session.turnsPaging || previous?.turnsPaging,
    });
  });
  existing
    .filter((session) => !merged.has(session.id))
    .forEach((session) => {
      merged.set(session.id, session);
    });
  return Array.from(merged.values()).sort(compareSessions);
}

function formatRelativeTime(at: number, language: LegacyShellLanguage): string {
  const delta = Math.max(0, Date.now() - at);
  const minutes = Math.floor(delta / 60000);
  if (minutes <= 0) {
    return language === "zh" ? "刚刚" : "just now";
  }
  if (minutes < 60) {
    return language === "zh" ? `${minutes} 分钟前` : `${minutes} min ago`;
  }
  const hours = Math.floor(minutes / 60);
  return language === "zh" ? `${hours} 小时前` : `${hours} hr ago`;
}

function resolveSessionActivityAt(session: ChatSession): number {
  const latestMessageAt = session.messages.reduce((latest, message) => {
    return Math.max(latest, Number(message.at) || 0);
  }, 0);
  return Math.max(session.createdAt, latestMessageAt);
}

function buildSessionMeta(session: ChatSession, _language: LegacyShellLanguage): string {
  return formatDateTimeMinute(resolveSessionActivityAt(session));
}

function buildSessionContextLabel(session: ChatSession): string | undefined {
  
  const name = normalizeText(session.target.name) || normalizeText(session.target.id);
  return name || undefined;
}

function enabledModels(provider: ChatProvider | null | undefined): ChatProviderModel[] {
  return Array.isArray(provider?.models)
    ? provider.models.filter((model) => model && model.is_enabled !== false)
    : [];
}

function enabledProviders(providers: ChatProvider[]): ChatProvider[] {
  return providers.filter((provider) => provider.is_enabled !== false && enabledModels(provider).length > 0);
}

function defaultModelSelection(providers: ChatProvider[]) {
  const available = enabledProviders(providers);
  const provider = available.find((item) => item.is_default) || available[0] || null;
  if (!provider) {
    return { providerID: "", modelID: "" };
  }
  const models = enabledModels(provider);
  const preferredModelID = normalizeText(provider.default_model);
  const model = models.find((item) => normalizeText(item.id) === preferredModelID) || models[0] || null;
  return {
    providerID: normalizeText(provider.id),
    modelID: model ? normalizeText(model.id) : "",
  };
}

function resolveModelSelection(session: ChatSession | null, providers: ChatProvider[]) {
  const fallback = defaultModelSelection(providers);
  const providerID = normalizeText(session?.modelProviderID) || fallback.providerID;
  const provider = enabledProviders(providers).find((item) => normalizeText(item.id) === providerID) || null;
  if (!provider) {
    return fallback;
  }
  const models = enabledModels(provider);
  const preferredModelID = normalizeText(session?.modelID);
  const model = models.find((item) => normalizeText(item.id) === preferredModelID)
    || models.find((item) => normalizeText(item.id) === normalizeText(provider.default_model))
    || models[0]
    || null;
  return {
    providerID: normalizeText(provider.id),
    modelID: model ? normalizeText(model.id) : "",
  };
}

function normalizeTaskStatus(status: string): string {
  return normalizeText(status).toLowerCase() || "queued";
}

function isConversationBusyStatus(status: string): boolean {
  return ["streaming", "queued", "running", "in_progress", "inprogress", "busy"].includes(normalizeTaskStatus(status));
}

function serializeMessageAttachment(attachment: ComposerAttachment) {
  if (attachment.assetURL) {
    return {
      id: attachment.id,
      name: attachment.name,
      content_type: attachment.contentType,
      asset_url: attachment.assetURL,
      preview_url: attachment.previewURL,
    };
  }
  return {
    id: attachment.id,
    name: attachment.name,
    content_type: attachment.contentType,
    data_url: attachment.dataURL,
    preview_data_url: isComposerImageAttachment(attachment) ? attachment.previewDataURL : undefined,
  };
}

function isCompactViewport(): boolean {
  if (typeof window.matchMedia !== "function") {
    return false;
  }
  return window.matchMedia(`(max-width: ${MOBILE_VIEWPORT_BREAKPOINT_PX}px)`).matches;
}

export function ConversationRuntimeProvider({
  route: rawRoute,
  language,
  children,
}: ProviderProps) {
  const route = normalizeConversationRoute(rawRoute);
  const apiClient = useMemo(() => createAPIClient(), []);
  const initialRuntimeStateRef = useRef<ConversationRuntimeInitialState | null>(null);
  if (!initialRuntimeStateRef.current) {
    initialRuntimeStateRef.current = resolveInitialConversationRuntimeState();
  }
  const [sessionsByRoute, setSessionsByRoute] = useState<SessionsState>(() =>
    initialRuntimeStateRef.current?.sessionsByRoute || { chat: [] },
  );
  const [sessionsLoadedByRoute, setSessionsLoadedByRoute] = useState<Record<ConversationRoute, boolean>>({
    chat: false,
  });
  const [activeSessionByRoute, setActiveSessionByRoute] = useState<ActiveSessionState>(() =>
    initialRuntimeStateRef.current?.activeSessionByRoute || { chat: CANONICAL_CHAT_SESSION_ID },
  );
  const [providers, setProviders] = useState<ChatProvider[]>([]);
  const [skills, setSkills] = useState<ChatCapability[]>([]);
  const [skillCatalogLoaded, setSkillCatalogLoaded] = useState(false);
  const [mcps, setMcps] = useState<ChatCapability[]>([]);
  const [composerDrafts, setComposerDrafts] = useState<ComposerDraftMap>(() => loadComposerDrafts());
  const [composerAttachmentDrafts, setComposerAttachmentDrafts] = useState<ComposerAttachmentDraftMap>(() => loadComposerAttachmentDrafts());
  const [compact, setCompact] = useState(() => isCompactViewport());
  const [inspectorOpen, setInspectorOpen] = useState(false);
  const [inspectorTab, setInspectorTab] = useState<"model" | "capabilities" | "skills">("model");
  const [inspectorTabOpen, setInspectorTabOpen] = useState(true);
  const [runtimeEventFilter, setRuntimeEventFilter] = useState<RuntimeEventFilterID[]>(() => loadRuntimeEventFilter());
  const [pinningSessionIDs, setPinningSessionIDs] = useState<Record<string, boolean>>({});
  const [pageHidden, setPageHidden] = useState(() => typeof document !== "undefined" && document.hidden);
  const pollTimerRef = useRef<number>(0);
  const sessionsByRouteRef = useRef(sessionsByRoute);
  const recoveryPromisesRef = useRef(new Map<string, Promise<ChatSession | null>>());
  const progressiveHistoryLoadsRef = useRef(new Set<string>());
  const processEventDetailLoadsRef = useRef(new Set<string>());
  const composerDraftPersistTimerRef = useRef<number>(0);
  const latestComposerDraftsRef = useRef<ComposerDraftMap>(composerDrafts);
  const latestComposerAttachmentDraftsRef = useRef<ComposerAttachmentDraftMap>(composerAttachmentDrafts);

  const activeSessions = sessionsByRoute[route];
  const activeSessionReference = activeSessionByRoute[route];
  const activeSessionID = resolveSessionIDReference(activeSessions, activeSessionReference) || activeSessionReference;
  const activeSession = activeSessions.find((session) => session.id === activeSessionID) || null;
  const activeDraftKey = activeSessionID || NEW_CHAT_DRAFT_KEY;
  const activeDraftAttachments = composerAttachmentDrafts[activeDraftKey] || [];
  const availableProviders = useMemo(() => runtimeProviders(providers), [providers]);
  const availableSkillIDs = useMemo(
    () => skillCatalogLoaded ? defaultChatSkillIDs(skills) : null,
    [skillCatalogLoaded, skills],
  );
  const activeSkillIDs = useMemo(
    () => effectiveSessionSkillIDs(activeSession, availableSkillIDs),
    [activeSession, availableSkillIDs],
  );
  useEffect(() => {
    latestComposerDraftsRef.current = composerDrafts;
    window.clearTimeout(composerDraftPersistTimerRef.current);
    composerDraftPersistTimerRef.current = window.setTimeout(() => {
      persistComposerDrafts(latestComposerDraftsRef.current);
      composerDraftPersistTimerRef.current = 0;
    }, COMPOSER_DRAFT_PERSIST_DELAY_MS);
    return () => window.clearTimeout(composerDraftPersistTimerRef.current);
  }, [composerDrafts]);

  useEffect(() => {
    latestComposerAttachmentDraftsRef.current = composerAttachmentDrafts;
  }, [composerAttachmentDrafts]);

  useEffect(() => () => {
    window.clearTimeout(composerDraftPersistTimerRef.current);
    persistComposerDrafts(latestComposerDraftsRef.current);
  }, []);

  const patchSession = useCallback((
    routeKey: ConversationRoute,
    sessionID: string,
    updater: (session: ChatSession) => ChatSession,
  ) => {
    setSessionsByRoute((current) => {
      const nextState = {
        ...current,
        [routeKey]: normalizeRouteSessions(
          routeKey,
          current[routeKey].map((session) =>
            session.id === sessionID ? updater(session) : session,
          ),
        ),
      };
      sessionsByRouteRef.current = nextState;
      return nextState;
    });
  }, []);

  const createMessage = (
    role: "user" | "assistant",
    text: string,
    patch: Partial<ChatMessage> = {},
  ): ChatMessage => ({
    id: makeID("msg"),
    role,
    text,
    attachments: patch.attachments || [],
    route: patch.route || "",
    source: patch.source || "",
    error: Boolean(patch.error),
    status: patch.status || (role === "assistant" ? "done" : ""),
    at: patch.at || Date.now(),
    processEvents: patch.processEvents || [],
    processCollapsed: patch.processCollapsed,
  });

  const appendMessage = useCallback((routeKey: ConversationRoute, sessionID: string, message: ChatMessage) => {
    patchSession(routeKey, sessionID, (session) => ({
      ...session,
      status: message.role === "assistant" && message.error
        ? "failed"
        : message.role === "assistant" && isConversationBusyStatus(message.status)
          ? "busy"
          : session.status,
      title: session.titleAuto && message.role === "user"
        ? (message.text.slice(0, 32) || session.title)
        : session.title,
      titleAuto: session.titleAuto && message.role !== "user",
      messages: [...session.messages, message],
    }));
  }, [patchSession]);

  const setAssistantMessage = useCallback((
    routeKey: ConversationRoute,
    sessionID: string,
    messageID: string,
    patch: Partial<ChatMessage>,
  ) => {
    patchSession(routeKey, sessionID, (session) => ({
      ...session,
      status: normalizeText(patch.status) === "error"
        ? "failed"
        : normalizeText(patch.status) === "interrupted"
          ? "interrupted"
          : isConversationBusyStatus(patch.status || "")
          ? "busy"
          : normalizeText(patch.status) === "done"
            ? "ready"
            : session.status,
      messages: session.messages.map((message) =>
        message.id === messageID ? { ...message, ...patch } : message,
      ),
    }));
  }, [patchSession]);

  const loadProcessEventDetail = useCallback(async (messageID: string, eventID: string) => {
    const session = sessionsByRouteRef.current[route].find((item) => item.id === activeSessionByRoute[route]) || activeSession;
    const message = session?.messages.find((item) => item.id === messageID);
    const runtimeEvent = message?.processEvents.find((event) =>
      runtimeTraceEventDetailID(event) === eventID || normalizeText(event.id) === normalizeText(eventID),
    );
    const turnID = normalizeText(runtimeEvent?.turn_id);
    const detailID = runtimeEvent ? runtimeTraceEventDetailID(runtimeEvent) : normalizeText(eventID);
    if (!session?.id || !message || !runtimeEvent || !turnID || !detailID || runtimeEvent.raw?.has_detail !== true) {
      return;
    }
    const requestKey = `${session.id}:${turnID}:${detailID}`;
    if (processEventDetailLoadsRef.current.has(requestKey)) {
      return;
    }
    processEventDetailLoadsRef.current.add(requestKey);
    try {
      const payload = await apiClient.get<RuntimeTraceEventDetailResponse>(
        chatTerminalSessionEndpoint(`${encodeURIComponent(session.id)}/turns/${encodeURIComponent(turnID)}/events/${encodeURIComponent(detailID)}`),
      );
      const detailedEvent = normalizeRuntimeTraceEventDetail(payload.event, runtimeEvent);
      if (!detailedEvent) {
        return;
      }
      patchSession(route, session.id, (currentSession) => ({
        ...currentSession,
        messages: currentSession.messages.map((currentMessage) => {
          if (currentMessage.id !== messageID) {
            return currentMessage;
          }
          return {
            ...currentMessage,
            processEvents: currentMessage.processEvents.map((event) =>
              runtimeTraceEventDetailID(event) === detailID || normalizeText(event.id) === detailID
                ? detailedEvent
                : event,
            ),
          };
        }),
      }));
    } catch {
    } finally {
      processEventDetailLoadsRef.current.delete(requestKey);
    }
  }, [activeSession, activeSessionByRoute, apiClient, patchSession, route]);

  const focusSession = useCallback((sessionID: string) => {
    const resolvedSessionID = sessionID;
    const nextActiveState = { ...activeSessionByRoute, [route]: resolvedSessionID };
    setActiveSessionByRoute(nextActiveState);
    writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
    writeWorkbenchRouteSessionID(route, resolvedSessionID);
  }, [activeSessionByRoute, route]);

  const removeSession = useCallback(async (sessionID: string) => {
    try {
      await apiClient.delete(chatTerminalSessionEndpoint(encodeURIComponent(sessionID)));
    } catch {
    }
    const nextSessionsByRoute: SessionsState = {
      ...sessionsByRoute,
      [route]: sessionsByRoute[route].filter((session) => session.id !== sessionID),
    };
    const nextActiveState = {
      ...activeSessionByRoute,
      [route]:
        activeSessionByRoute[route] === sessionID
          ? nextSessionsByRoute[route][0]?.id || ""
          : activeSessionByRoute[route],
    };
    const nextDrafts = { ...latestComposerDraftsRef.current };
    const nextAttachmentDrafts = { ...latestComposerAttachmentDraftsRef.current };
    delete nextDrafts[sessionID];
    delete nextAttachmentDrafts[sessionID];
    setSessionsByRoute(nextSessionsByRoute);
    setActiveSessionByRoute(nextActiveState);
    setComposerDrafts(nextDrafts);
    setComposerAttachmentDrafts(nextAttachmentDrafts);
    persistComposerDrafts(nextDrafts);
    persistComposerAttachmentDrafts(nextAttachmentDrafts);
    writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
  }, [activeSessionByRoute, apiClient, route, sessionsByRoute]);

  const setSessionPinned = useCallback(async (sessionID: string, pinned: boolean) => {
    const normalizedSessionID = normalizeText(sessionID);
    if (!normalizedSessionID) {
      return;
    }
    const applyPinnedState = () => {
      patchSession(route, normalizedSessionID, (session) => ({
        ...session,
        pinned,
      }));
    };
    setPinningSessionIDs((current) => ({ ...current, [normalizedSessionID]: true }));
    try {
      const payload = await apiClient.post<{ session?: TerminalSessionPayload }>(
        chatTerminalSessionEndpoint(`${encodeURIComponent(normalizedSessionID)}/pin`),
        { pinned },
      );
      void payload;
      applyPinnedState();
    } catch {
      applyPinnedState();
    } finally {
      setPinningSessionIDs((current) => ({ ...current, [normalizedSessionID]: false }));
    }
  }, [apiClient, patchSession, route]);

  const hydrateRuntimeSessionResponse = (
    routeKey: ConversationRoute,
    sourceRoute: ConversationRoute,
    sessionID: string,
    payload: { session?: TerminalSessionPayload },
  ): ChatSession | null => {
    return normalizeTerminalSession(
      payload.session || {},
      sessionsByRouteRef.current[routeKey].find((item) => item.id === sessionID) || null,
      sourceRoute,
    );
  };

  const upsertRuntimeSession = useCallback((routeKey: ConversationRoute, nextSession: ChatSession) => {
    const normalizedSession = { ...nextSession, sourceRoute: nextSession.sourceRoute || routeKey };
    setSessionsByRoute((current) => {
      const hasSession = current[routeKey].some((session) => session.id === normalizedSession.id);
      const nextSessions = hasSession
        ? current[routeKey].map((session) => (session.id === normalizedSession.id ? normalizedSession : session))
        : [normalizedSession, ...current[routeKey]];
      const nextState = {
        ...current,
        [routeKey]: normalizeRouteSessions(routeKey, nextSessions),
      };
      sessionsByRouteRef.current = nextState;
      return nextState;
    });
  }, []);

  const createTerminalRuntimeSession = useCallback(async (routeKey: ConversationRoute, title: string = ""): Promise<ChatSession | null> => {
    const payload = await apiClient.post<{ session?: TerminalSessionPayload }>(
      chatTerminalSessionEndpoint(),
      normalizeText(title) ? { title: normalizeText(title).slice(0, 80) } : {},
    );
    const nextSession = normalizeTerminalSession(payload.session || {}, null, routeKey);
    if (!nextSession) {
      return null;
    }
    upsertRuntimeSession(routeKey, nextSession);
    setActiveSessionByRoute((current) => {
      const nextActiveState = { ...current, [routeKey]: nextSession.id };
      writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
      return nextActiveState;
    });
    writeWorkbenchRouteSessionID(routeKey, nextSession.id);
    return nextSession;
  }, [apiClient, upsertRuntimeSession]);

  const persistRuntimeSessionConfig = useCallback(async (routeKey: ConversationRoute, session: ChatSession) => {
    void routeKey;
    patchSession(route, session.id, (currentSession) => ({
      ...currentSession,
      modelProviderID: session.modelProviderID,
      modelID: session.modelID,
      toolIDs: normalizeSelectionIDs(session.toolIDs),
      skillIDs: normalizeSelectionIDs(session.skillIDs),
      skillIDsExplicit: session.skillIDsExplicit === true,
      mcpIDs: normalizeSelectionIDs(session.mcpIDs),
    }));
  }, [patchSession, route]);

  const hydrateRuntimeSession = async (
    routeKey: ConversationRoute,
    sessionID: string,
    options: { turnBefore?: string; turnLimit?: number } = {},
  ): Promise<ChatSession | null> => {
    const payload = await apiClient.get<{ session?: TerminalSessionPayload }>(
      chatTerminalSessionEndpoint(encodeURIComponent(sessionID), {
        turn_before: normalizeText(options.turnBefore),
        turn_limit: options.turnLimit,
      }),
    );
    return hydrateRuntimeSessionResponse(routeKey, routeKey, sessionID, payload);
  };

  const refreshActiveSession = useCallback(async () => {
    const sessionID = normalizeText(activeSession?.id);
    if (!sessionID || activeSession?.serverBacked !== true) {
      return;
    }
    try {
      const hydrated = await hydrateRuntimeSession(route, sessionID);
      if (hydrated) {
        upsertRuntimeSession(route, hydrated);
      }
    } catch {
    }
  }, [activeSession, route, upsertRuntimeSession]);

  const recoverRuntimeSession = async (
    routeKey: ConversationRoute,
    sessionID: string,
    requirements: RuntimeRecoveryRequirement = {},
    attempts: number = 3,
  ): Promise<ChatSession | null> => {
    const requirementKey = [
      routeKey,
      sessionID,
      requirements.requireMessages === true ? "messages" : "any-messages",
      requirements.requireStableAssistant === true ? "stable" : "any-state",
      String(attempts),
    ].join(":");
    const inFlight = recoveryPromisesRef.current.get(requirementKey);
    if (inFlight) {
      return inFlight;
    }
    const recoveryPromise = (async () => {
      for (let attempt = 0; attempt < attempts; attempt += 1) {
        try {
          const hydrated = await hydrateRuntimeSession(routeKey, sessionID);
          if (
            hydrated
            && (!requirements.requireMessages || hydrated.messages.length > 0)
            && (!requirements.requireStableAssistant || hasPersistedAssistantState(hydrated.messages))
          ) {
            upsertRuntimeSession(routeKey, hydrated);
            return hydrated;
          }
        } catch {
        }
        if (attempt < attempts - 1) {
          await new Promise((resolve) => window.setTimeout(resolve, 1000));
        }
      }
      return null;
    })();
    recoveryPromisesRef.current.set(requirementKey, recoveryPromise);
    return recoveryPromise.finally(() => {
      recoveryPromisesRef.current.delete(requirementKey);
    });
  };

  const shouldAttemptServerRecovery = (
    remoteSession: ChatSession | null,
    localSession: ChatSession | null,
  ): boolean => {
    if (!remoteSession) {
      return true;
    }
    if (remoteSession.messagesLoaded !== true) {
      return true;
    }
    if (hasRecoverableRuntimeState(localSession)) {
      return true;
    }
    if (hasRecoverableRuntimeState(remoteSession)) {
      return true;
    }
    return false;
  };

  const sendPrompt = async (prompt: string = composerDrafts[activeDraftKey] || "") => {
    const content = prompt.trim().slice(0, MAX_COMPOSER_CHARS);
    let attachments = activeDraftAttachments;
    if (!content && attachments.length === 0) {
      return;
    }
    if (activeSession && shouldPollTerminalBackedSession(activeSession)) {
      return;
    }
    const session = activeSession?.serverBacked
      ? activeSession
      : await createTerminalRuntimeSession(route, content);
    if (!session) {
      return;
    }
    patchSession(route, session.id, (currentSession) => ({ ...currentSession, status: "busy" }));
    try {
      attachments = await uploadDraftAttachments(session.id, attachments);
      const payload = await apiClient.post<{ session?: TerminalSessionPayload }>(
        chatTerminalSessionEndpoint(`${encodeURIComponent(session.id)}/input`),
        {
          input: content,
          attachments: attachments.map(serializeMessageAttachment),
          skill_ids: activeSkillIDs,
        },
      );
      const latestSession = sessionsByRouteRef.current[route].find((item) => item.id === session.id) || session;
      const hydrated = normalizeTerminalSession(payload.session || {}, latestSession, route);
      if (hydrated) {
        upsertRuntimeSession(route, hydrated);
      } else {
        await recoverRuntimeSession(route, session.id, { requireMessages: true }, 1);
      }
      const nextDrafts = { ...composerDrafts, [session.id]: "", [NEW_CHAT_DRAFT_KEY]: "" };
      const nextAttachmentDrafts = { ...composerAttachmentDrafts, [session.id]: [], [NEW_CHAT_DRAFT_KEY]: [] };
      setComposerDrafts(nextDrafts);
      setComposerAttachmentDrafts(nextAttachmentDrafts);
      persistComposerDrafts(nextDrafts);
      persistComposerAttachmentDrafts(nextAttachmentDrafts);
    } catch (error) {
      patchSession(route, session.id, (currentSession) => ({
        ...currentSession,
        status: "failed",
        messages: [
          ...currentSession.messages,
          createMessage("assistant", error instanceof Error ? error.message : "Request failed", {
            status: "error",
            error: true,
          }),
        ],
      }));
    }
  };

  const uploadDraftAttachments = async (
    sessionID: string,
    attachments: ComposerAttachment[],
  ): Promise<ComposerAttachment[]> => {
    const existing = attachments.filter((attachment) => attachment.assetURL);
    const pending = attachments.filter((attachment) => !attachment.assetURL && attachment.dataURL);
    if (pending.length === 0) {
      return existing;
    }
    const payload = await apiClient.post<SessionAttachmentUploadResponse>(
      chatTerminalSessionEndpoint(`${encodeURIComponent(sessionID)}/attachments`),
      {
        attachments: pending.map((attachment) => ({
          name: attachment.name,
          content_type: attachment.contentType,
          data_url: attachment.dataURL,
          preview_data_url: attachment.previewDataURL || attachment.dataURL,
        })),
      },
    );
    const items = Array.isArray(payload.items) ? payload.items : [];
    if (items.length !== pending.length) {
      throw new Error("Failed to store attachments.");
    }
    return [
      ...existing,
      ...items.map((item, index) => {
        const fallback = pending[index];
        const id = normalizeText(item.id);
        const assetURL = normalizeText(item.asset_url);
        const previewURL = normalizeText(item.preview_url);
        if (!id || !assetURL) {
          throw new Error("Failed to store attachments.");
        }
        const contentType = normalizeText(item.content_type) || fallback.contentType;
        const kind = isComposerImageAttachment(fallback) || contentType.startsWith("image/") ? "image" : "file";
        return {
          id,
          kind,
          name: normalizeText(item.name) || fallback.name,
          contentType,
          size: Number.isFinite(Number(item.size)) ? Number(item.size) : fallback.size,
          assetURL,
          previewURL: kind === "image" ? previewURL || assetURL : undefined,
        };
      }),
    ];
  };

  const loadRuntimeSessions = async (routeKey: ConversationRoute) => {
    const payload = await apiClient.get<{ items?: TerminalSessionPayload[] }>(chatTerminalSessionEndpoint());
    const remoteSessions = (Array.isArray(payload.items) ? payload.items : [])
      .map((item) => normalizeTerminalSession(item, undefined, routeKey))
      .filter((session): session is ChatSession => session !== null);
    const normalizedRemoteSessions = normalizeRouteSessions(routeKey, remoteSessions);
    const nextSessions = normalizeRouteSessions(
      routeKey,
      mergeRuntimeSessions(normalizedRemoteSessions, sessionsByRouteRef.current[routeKey]),
    );
    const nextState = {
      ...sessionsByRouteRef.current,
      [routeKey]: nextSessions,
    };
    sessionsByRouteRef.current = nextState;
    setSessionsByRoute(nextState);
    setSessionsLoadedByRoute((current) => ({ ...current, [routeKey]: true }));
    return normalizedRemoteSessions;
  };

  const refreshCurrentRouteOnPageActive = useCallback(async () => {
    if (typeof document !== "undefined" && document.visibilityState === "hidden") {
      return;
    }

    try {
      await loadRuntimeSessions(route);
    } catch {
    }

    if (activeSession?.id) {
      try {
        const hydrated = await hydrateRuntimeSession(route, activeSession.id);
        if (hydrated) {
          upsertRuntimeSession(route, hydrated);
        }
      } catch {
      }
    }

  }, [activeSession, hydrateRuntimeSession, loadRuntimeSessions, route, upsertRuntimeSession]);

  useEffect(() => {
    persistActiveSessionSnapshots(activeSessionByRoute, sessionsByRoute);
  }, [activeSessionByRoute, sessionsByRoute]);

  useEffect(() => {
    writeConversationRuntimeCache(activeSessionByRoute, sessionsByRoute);
    writeLongTermConversationRuntimeCache(activeSessionByRoute, sessionsByRoute);
    writeSessionInfoConversationRuntimeCache(activeSessionByRoute, sessionsByRoute);
  }, [activeSessionByRoute, sessionsByRoute]);

  useEffect(() => {
    sessionsByRouteRef.current = sessionsByRoute;
  }, [sessionsByRoute]);

  useEffect(() => {
    const syncViewport = () => setCompact(isCompactViewport());
    window.addEventListener("resize", syncViewport);
    return () => window.removeEventListener("resize", syncViewport);
  }, []);

  usePageActivation({
    debounceMs: PAGE_ACTIVE_REFRESH_DEBOUNCE_MS,
    onVisibilityChange: setPageHidden,
    onActive: refreshCurrentRouteOnPageActive,
  });

  useEffect(() => {
    const sessionID = normalizeText(activeSession?.id);
    const paging = activeSession?.turnsPaging;
    const beforeTurnID = normalizeText(paging?.next_before_turn_id || paging?.oldest_turn_id)
      || oldestTurnIDFromMessages(activeSession?.messages || []);
    if (
      !sessionID
      || activeSession?.serverBacked !== true
      || activeSession.messagesLoaded !== true
      || paging?.has_more_before !== true
      || !beforeTurnID
      || hasRecoverableRuntimeState(activeSession)
    ) {
      return;
    }
    const requestKey = `${route}:${sessionID}:${beforeTurnID}`;
    if (progressiveHistoryLoadsRef.current.has(requestKey)) {
      return;
    }
    let cancelled = false;
    progressiveHistoryLoadsRef.current.add(requestKey);
    void (async () => {
      try {
        const hydrated = await hydrateRuntimeSession(route, sessionID, {
          turnBefore: beforeTurnID,
          turnLimit: CHAT_HISTORY_PAGE_TURN_LIMIT,
        });
        if (!cancelled && hydrated) {
          upsertRuntimeSession(route, hydrated);
        }
      } catch {
      } finally {
        progressiveHistoryLoadsRef.current.delete(requestKey);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [
    activeSession?.id,
    activeSession?.messages,
    activeSession?.messagesLoaded,
    activeSession?.serverBacked,
    activeSession?.turnsPaging?.has_more_before,
    activeSession?.turnsPaging?.next_before_turn_id,
    activeSession?.turnsPaging?.oldest_turn_id,
    route,
    upsertRuntimeSession,
  ]);

  useEffect(() => {
    const loadCatalogs = async () => {
      try {
        const providerPayload = await apiClient.get<{ items?: ChatProvider[] }>("/api/control/llm/providers");
        setProviders(Array.isArray(providerPayload.items) ? providerPayload.items : []);
      } catch {
      }
      try {
        const [skillPayload, mcpPayload] = await Promise.all([
          apiClient.get<{ items?: ChatCapability[] }>("/api/control/skills"),
          apiClient.get<{ items?: ChatCapability[] }>("/api/control/mcps"),
        ]);
        setSkills(Array.isArray(skillPayload.items) ? skillPayload.items : []);
        setMcps(Array.isArray(mcpPayload.items) ? mcpPayload.items : []);
      } catch {
      } finally {
        setSkillCatalogLoaded(true);
      }
    };
    void loadCatalogs();
  }, [apiClient]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const remoteSessions = await loadRuntimeSessions(route);
        if (cancelled) {
          return;
        }
        const explicitRouteSessionReference = readWorkbenchRouteSessionID(route);
        const preferredActiveReference = explicitRouteSessionReference
          || sessionsByRouteRef.current[route][0]?.id
          || normalizeText(activeSessionByRoute[route]);
        const preferredActiveID = resolveSessionIDReference(
          [...remoteSessions, ...sessionsByRouteRef.current[route]],
          preferredActiveReference,
        ) || preferredActiveReference;
        const localPreferredSession = preferredActiveID
          ? sessionsByRouteRef.current[route].find((session) => session.id === preferredActiveID) || null
          : null;
        const remotePreferredSession = preferredActiveID
          ? remoteSessions.find((session) => session.id === preferredActiveID) || null
          : null;
        const recoveredSession = preferredActiveID && shouldAttemptServerRecovery(remotePreferredSession, localPreferredSession)
          ? await recoverRuntimeSession(
              route,
              preferredActiveID,
              {
                requireMessages: remotePreferredSession?.messagesLoaded !== true,
                requireStableAssistant: hasRecoverableRuntimeState(localPreferredSession) || hasRecoverableRuntimeState(remotePreferredSession),
              },
            )
          : null;
        if (cancelled) {
          return;
        }
        if (recoveredSession) {
          upsertRuntimeSession(route, recoveredSession);
        }
        const nextActiveID = remoteSessions.some((session) => session.id === preferredActiveID) || recoveredSession
          ? preferredActiveID
          : sessionsByRouteRef.current[route][0]?.id || activeSessionByRoute[route];
        if (nextActiveID && nextActiveID !== activeSessionByRoute[route]) {
          const nextActiveState = { ...activeSessionByRoute, [route]: nextActiveID };
          setActiveSessionByRoute(nextActiveState);
          writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
        }
      } catch {
        if (cancelled) {
          return;
        }
        setSessionsLoadedByRoute((current) => ({ ...current, [route]: true }));
      }
    })();
    return () => {
      cancelled = true;
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiClient, route]);

  useEffect(() => {
    const syncActiveSessionFromRoute = () => {
      const explicitRouteSessionReference = readWorkbenchRouteSessionID(route);
      const nextActiveID = explicitRouteSessionReference
        ? resolveSessionIDReference(sessionsByRouteRef.current[route], explicitRouteSessionReference) || explicitRouteSessionReference
        : sessionsByRouteRef.current[route][0]?.id || "";
      if (!nextActiveID) {
        return;
      }
      setActiveSessionByRoute((current) => {
        if (current[route] === nextActiveID) {
          return current;
        }
        const nextActiveState = { ...current, [route]: nextActiveID };
        writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
        return nextActiveState;
      });
    };
    window.addEventListener("popstate", syncActiveSessionFromRoute);
    return () => window.removeEventListener("popstate", syncActiveSessionFromRoute);
  }, [route]);

  useEffect(() => {
    if (
      !activeSession?.id
      || activeSession.serverBacked !== true
      || (activeSession.messagesLoaded && !hasRecoverableRuntimeState(activeSession))
    ) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const hydrated = await recoverRuntimeSession(
          route,
          activeSession.id,
          {
            requireMessages: !activeSession.messagesLoaded,
            requireStableAssistant: hasRecoverableRuntimeState(activeSession),
          },
          3,
        );
        if (cancelled || !hydrated) {
          return;
        }
        upsertRuntimeSession(route, hydrated);
      } catch {
        if (cancelled) {
          return;
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [activeSession, apiClient, route]);

  useEffect(() => {
    window.clearTimeout(pollTimerRef.current);
    const recoverableSessions = Object.entries(sessionsByRoute).flatMap(([routeKey, sessions]) =>
      sessions
        .filter(shouldPollTerminalBackedSession)
        .map((session) => ({
          route: routeKey as ConversationRoute,
          sessionID: session.id,
        })),
    );
    if (!recoverableSessions.length) {
      return;
    }
    const pollPlan = resolveChatSessionPollPlan({
      sessionCount: recoverableSessions.length,
      pageHidden,
    });
    if (!pollPlan.enabled) {
      return;
    }
    pollTimerRef.current = window.setTimeout(async () => {
      for (const item of recoverableSessions) {
        try {
          const hydrated = await hydrateRuntimeSession(item.route, item.sessionID);
          if (hydrated) {
            upsertRuntimeSession(item.route, hydrated);
          }
        } catch {
        }
      }
    }, pollPlan.interval);
    return () => window.clearTimeout(pollTimerRef.current);
  }, [pageHidden, sessionsByRoute, upsertRuntimeSession]);

  const selection = resolveModelSelection(activeSession, availableProviders);
  const selectedProvider = enabledProviders(availableProviders).find((provider) => normalizeText(provider.id) === selection.providerID) || null;
  const selectedModel = enabledModels(selectedProvider).find((model) => normalizeText(model.id) === selection.modelID) || null;
  const currentTarget = activeSession?.target || defaultChatTarget();
  const activeSessionBusy = activeSession ? shouldPollTerminalBackedSession(activeSession) : false;

  const workspaceValue = useMemo<ConversationRuntimeWorkspaceContextValue>(() => ({
    route,
    compact,
    inspectorOpen,
    inspectorTab,
    inspectorTabOpen,
    sessions: activeSessions,
    activeSession,
    busy: activeSessionBusy,
    sessionItems: activeSessions.map((session) => ({
      id: session.id,
      title: session.title,
      contextLabel: buildSessionContextLabel(session),
      meta: buildSessionMeta(session, language),
      shortHash: hashSessionIDShort(session.id),
      createdAt: session.createdAt,
      active: session.id === activeSessionID,
      draft: isBlankDraftSession(session),
      pinned: session.pinned,
      pinning: Boolean(pinningSessionIDs[session.id]),
    })),
    target: currentTarget,
    lockedTarget: Boolean(activeSession?.messages.length),
    selectedProviderId: selection.providerID,
    selectedModelId: selection.modelID,
    selectedModelLabel: selectedModel?.name || selectedModel?.id || "Default",
    selectedModelSupportsVision: selectedModel ? selectedModel.supports_vision !== false : true,
    providers: enabledProviders(availableProviders).map((provider) => ({
      id: normalizeText(provider.id),
      name: normalizeText(provider.name) || normalizeText(provider.id),
      models: enabledModels(provider).map((model) => ({
        id: normalizeText(model.id),
        name: normalizeText(model.name) || normalizeText(model.id),
        supportsVision: model.supports_vision !== false,
        active:
          normalizeText(provider.id) === selection.providerID
          && normalizeText(model.id) === selection.modelID,
      })),
    })),
    capabilities: [
      ...mcps
        .filter((item) => item.enabled !== false)
        .map((item) => ({
          id: normalizeText(item.id),
          name: normalizeText(item.name) || normalizeText(item.id),
          description: normalizeText(item.description) || normalizeText(item.scope) || "MCP",
          kind: "mcp" as const,
          active: Boolean(activeSession?.mcpIDs.includes(normalizeText(item.id))),
        }))
        .filter((item) => item.id),
    ],
    skills: [
      ...skills
        .filter((item) => item.enabled !== false && isPublicSkillCapability(item))
        .map((item) => ({
          id: normalizeText(item.id),
          name: normalizeText(item.name) || normalizeText(item.id),
          description: normalizeText(item.description) || normalizeText(item.scope) || "Skill",
          kind: "skill" as const,
          active: activeSkillIDs.includes(normalizeText(item.id)),
          visibility: "public" as const,
          locked: false,
        }))
        .filter((item) => item.id),
    ].filter((item): item is RuntimeSelection => Boolean(item?.id)),
    runtimeEventFilter,
    toolCount: (activeSession?.toolIDs.length || 0) + (activeSession?.mcpIDs.length || 0),
    skillCount: activeSkillIDs.length,
    createSession: () => {
      void createTerminalRuntimeSession(route);
    },
    focusSession,
    removeSession,
    setSessionPinned,
    refreshActiveSession,
    toggleInspector: (tab) => {
      if (!tab) {
        setInspectorOpen((current) => {
          const nextOpen = !current;
          if (nextOpen) {
            setInspectorTabOpen(true);
          }
          return nextOpen;
        });
        return;
      }
      if (tab === inspectorTab) {
        setInspectorOpen(true);
        setInspectorTabOpen((current) => !current);
        return;
      }
      setInspectorTab(tab);
      setInspectorTabOpen(true);
      setInspectorOpen(true);
    },
    closeInspector: () => setInspectorOpen(false),
    selectModel: (providerID: string, modelID: string) => {
      const session = activeSession;
      if (!session) {
        return;
      }
      const nextSession = {
        ...session,
        modelProviderID: normalizeText(providerID),
        modelID: normalizeText(modelID),
      };
      patchSession(route, session.id, (currentSession) => ({
        ...currentSession,
        modelProviderID: nextSession.modelProviderID,
        modelID: nextSession.modelID,
      }));
      void persistRuntimeSessionConfig(route, nextSession);
    },
    toggleCapability: (id: string, kind: "tool" | "mcp", checked: boolean) => {
      const session = activeSession;
      if (!session) {
        return;
      }
      const value = normalizeText(id);
      if (!value) {
        return;
      }
      const mutate = (items: string[]) =>
        checked
          ? normalizeSelectionIDs([...items, value])
          : items.filter((item) => item !== value);
      const nextSession = kind === "tool"
        ? { ...session, toolIDs: mutate(session.toolIDs) }
        : { ...session, mcpIDs: mutate(session.mcpIDs) };
      patchSession(route, session.id, (currentSession) =>
        kind === "tool"
          ? { ...currentSession, toolIDs: mutate(currentSession.toolIDs) }
          : { ...currentSession, mcpIDs: mutate(currentSession.mcpIDs) },
      );
      void persistRuntimeSessionConfig(route, nextSession);
    },
    toggleSkill: (id: string, checked: boolean) => {
      const session = activeSession;
      if (!session) {
        return;
      }
      const value = normalizeText(id);
      if (!value) {
        return;
      }
      if (availableSkillIDs !== null && !availableSkillIDs.includes(value)) {
        return;
      }
      const mutate = (items: string[]) =>
        checked
          ? normalizeSelectionIDs([...effectiveSessionSkillIDs(session, availableSkillIDs), value])
          : effectiveSessionSkillIDs(session, availableSkillIDs).filter((item) => item !== value);
      const nextSession = {
        ...session,
        skillIDs: mutate(session.skillIDs),
        skillIDsExplicit: true,
      };
      patchSession(route, session.id, (currentSession) => ({
        ...currentSession,
        skillIDs: mutate(currentSession.skillIDs),
        skillIDsExplicit: true,
      }));
      void persistRuntimeSessionConfig(route, nextSession);
    },
    toggleRuntimeEventFilter: (id: RuntimeEventFilterID, checked: boolean) => {
      const value = normalizeText(id) as RuntimeEventFilterID;
      const allowed = new Set(RUNTIME_EVENT_FILTER_OPTIONS.map((option) => option.id));
      if (!allowed.has(value)) {
        return;
      }
      setRuntimeEventFilter((current) => {
        const next = checked
          ? normalizeRuntimeEventFilter([...current, value])
          : normalizeRuntimeEventFilter(current.filter((item) => item !== value));
        persistRuntimeEventFilter(next);
        return next;
      });
    },
    toggleProcess: (messageID: string) => {
      if (!activeSession) {
        return;
      }
      patchSession(route, activeSession.id, (session) => ({
        ...session,
        messages: session.messages.map((message) =>
          message.id === messageID
            ? { ...message, processCollapsed: !resolveProcessCollapsed(message) }
            : message,
          ),
      }));
    },
    loadProcessEventDetail,
  }), [
    route,
    compact,
    inspectorOpen,
    inspectorTab,
    inspectorTabOpen,
    activeSessions,
    activeSession,
    language,
    activeSessionID,
    pinningSessionIDs,
    currentTarget,
    activeSessionBusy,
    selection.providerID,
    selection.modelID,
    selectedModel?.name,
    selectedModel?.id,
    availableProviders,
    mcps,
    skills,
    runtimeEventFilter,
    activeSessionByRoute,
    activeSkillIDs,
    availableSkillIDs,
    createTerminalRuntimeSession,
    focusSession,
    patchSession,
    persistRuntimeSessionConfig,
    refreshActiveSession,
    loadProcessEventDetail,
    removeSession,
    setSessionPinned,
  ]);

  const composerValue = useMemo<ConversationRuntimeComposerContextValue>(() => ({
    route,
    draft: composerDrafts[activeDraftKey] || "",
    draftAttachments: activeDraftAttachments,
    busy: activeSessionBusy,
    selectedModelSupportsVision: selectedModel ? selectedModel.supports_vision !== false : true,
    setDraft: (value: string) => {
      const nextDrafts = { ...composerDrafts, [activeDraftKey]: value.slice(0, MAX_COMPOSER_CHARS) };
      setComposerDrafts(nextDrafts);
    },
    addDraftAttachments: async (attachments: ComposerAttachment[]) => {
      const normalized = normalizeStoredAttachments(attachments);
      if (normalized.length === 0) {
        return;
      }
      const session = activeSession?.serverBacked
        ? activeSession
        : await createTerminalRuntimeSession(route);
      if (!session) {
        return;
      }
      const uploaded = await uploadDraftAttachments(session.id, normalized);
      const existing = composerAttachmentDrafts[session.id] || composerAttachmentDrafts[activeDraftKey] || [];
      const deduped = new Map<string, ComposerAttachment>();
      [...existing, ...uploaded].forEach((item) => {
        deduped.set(item.id, item);
      });
      const nextAttachments = Array.from(deduped.values()).slice(0, MAX_COMPOSER_IMAGE_ATTACHMENTS);
      const nextDrafts = { ...composerAttachmentDrafts, [session.id]: nextAttachments, [NEW_CHAT_DRAFT_KEY]: [] };
      setComposerAttachmentDrafts(nextDrafts);
      persistComposerAttachmentDrafts(nextDrafts);
    },
    removeDraftAttachment: (attachmentID: string) => {
      const sessionID = activeSession?.id;
      if (!sessionID) {
        return;
      }
      const nextItems = (composerAttachmentDrafts[sessionID] || []).filter((item) => item.id !== attachmentID);
      const nextDrafts = { ...composerAttachmentDrafts, [sessionID]: nextItems };
      setComposerAttachmentDrafts(nextDrafts);
      persistComposerAttachmentDrafts(nextDrafts);
    },
    clearDraftAttachments: () => {
      const sessionID = activeSession?.id;
      if (!sessionID) {
        return;
      }
      const nextDrafts = { ...composerAttachmentDrafts, [sessionID]: [] };
      setComposerAttachmentDrafts(nextDrafts);
      persistComposerAttachmentDrafts(nextDrafts);
    },
    sendPrompt,
  }), [
    route,
    activeDraftKey,
    composerDrafts,
    activeDraftAttachments,
    activeSessionBusy,
    selectedModel,
    composerAttachmentDrafts,
    activeSession,
    createTerminalRuntimeSession,
    sendPrompt,
  ]);

  return (
    <ConversationRuntimeWorkspaceContext.Provider value={workspaceValue}>
      <ConversationRuntimeComposerContext.Provider value={composerValue}>
        {children}
      </ConversationRuntimeComposerContext.Provider>
    </ConversationRuntimeWorkspaceContext.Provider>
  );
}

export function useConversationRuntimeWorkspace() {
  const value = useContext(ConversationRuntimeWorkspaceContext);
  if (!value) {
    throw new Error("ConversationRuntimeWorkspaceContext is not available");
  }
  return value;
}

export function useConversationRuntimeComposer() {
  const value = useContext(ConversationRuntimeComposerContext);
  if (!value) {
    throw new Error("ConversationRuntimeComposerContext is not available");
  }
  return value;
}

export function useConversationRuntime() {
  const workspace = useConversationRuntimeWorkspace();
  const composer = useConversationRuntimeComposer();
  return useMemo<ConversationRuntimeContextValue>(() => ({
    ...workspace,
    ...composer,
  }), [workspace, composer]);
}
