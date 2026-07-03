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
import {
  resolveRuntimeSessionPollPlan,
  useRuntimeSessionController,
  type RuntimeSessionNormalizeContext,
  type RuntimeSessionPayload,
} from "../shell/components/runtimeSessionController";
import {
  RUNTIME_SESSION_HISTORY_PAGE_TURN_LIMIT,
  runtimeSessionDetailEndpoint,
} from "../shell/components/runtimeSessionApi";
import { useRuntimeSessionCatalogs } from "../shell/components/runtimeSessionCatalogs";
import {
  runtimeSessionTurnsToTimelineMessages,
  type RuntimeSessionTurnPaging,
  type RuntimeSessionTimelineMessage,
} from "../shell/components/runtimeSessionViewModel";
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
const RUNTIME_CONFIG_STORAGE_KEY = "alter0.web.runtime.config.v1";
const COMPOSER_DRAFT_PERSIST_DELAY_MS = 160;
const NEW_CHAT_DRAFT_KEY = "__chat_new__";
const MAX_COMPOSER_CHARS = 10000;
const CHAT_SESSION_FAST_COMPENSATION_POLL_INTERVAL_MS = 1000;
const CHAT_SESSION_SLOW_COMPENSATION_POLL_INTERVAL_MS = 5000;
const CHAT_SESSION_FAST_COMPENSATION_ATTEMPTS = 8;
const CHAT_SESSION_EMPTY_UPDATE_DETAIL_REFRESH_BACKOFF_ATTEMPTS = [10, 20, 50] as const;
const CHAT_SESSION_EMPTY_UPDATE_DETAIL_REFRESH_STEADY_ATTEMPTS = 50;
const CHAT_SESSION_UPDATE_POLL_LIMIT = 50;
const CHAT_SESSION_UPDATE_POLL_BYTE_LIMIT = 64 * 1024;
const CHAT_SESSION_UPDATE_ACK_TURN_LIMIT = 16;
const CHAT_SESSION_UPDATE_ACK_EVENT_ID_LIMIT = 128;
const CODEX_RUNTIME_PROVIDER_ID = "alter0-codex";
const CODEX_RUNTIME_MODEL_ID = "codex";
const CANONICAL_CHAT_SESSION_ID = "alter0-chat";
const PAGE_ACTIVE_REFRESH_DEBOUNCE_MS = 400;
export const CHAT_RUNTIME_CACHE_SESSION_TTL_MS = 24 * 60 * 60 * 1000;
const CHAT_LONG_TERM_CACHE_TTL_MS = 24 * 60 * 60 * 1000;
const EMPTY_COMPOSER_ATTACHMENTS: ComposerAttachment[] = [];

type ChatSessionPollPlan = {
  enabled: boolean;
  interval: number;
};

export function resolveChatSessionPollPlan(options: {
  sessionCount: number;
  pageHidden: boolean;
  fallbackAttempt?: number;
}): ChatSessionPollPlan {
  const pollInterval = Math.max(0, options.fallbackAttempt || 0) < CHAT_SESSION_FAST_COMPENSATION_ATTEMPTS
    ? CHAT_SESSION_FAST_COMPENSATION_POLL_INTERVAL_MS
    : CHAT_SESSION_SLOW_COMPENSATION_POLL_INTERVAL_MS;
  const plan = resolveRuntimeSessionPollPlan({
    sessionCount: options.sessionCount,
    status: "busy",
    pageHidden: options.pageHidden,
    pollWhenHidden: false,
    pollInterval,
  });
  return {
    enabled: plan.enabled,
    interval: plan.interval,
  };
}

export type ConversationRoute = "chat";
const CONVERSATION_ROUTES: ConversationRoute[] = ["chat"];

type ChatTarget = {
  type: "model";
  id: string;
  name: string;
};

export type ChatMessage = RuntimeSessionTimelineMessage;

export type ChatSession = {
  id: string;
  sourceRoute?: ConversationRoute;
  status: string;
  title: string;
  titleAuto: boolean;
  titleScore: number;
  createdAt: number;
  updatedAt: number;
  lastOutputAt: number;
  activityAt: number;
  revision: number;
  detailRevision: number;
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
  turnsPaging?: RuntimeSessionTurnPagingPayload;
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
type RuntimeComposerConfig = {
  modelProviderID: string;
  modelID: string;
  toolIDs: string[];
  skillIDs: string[];
  skillIDsExplicit: boolean;
  mcpIDs: string[];
};

type RuntimeComposerConfigState = RuntimeComposerConfig & {
  stored: boolean;
};

type LegacySessionSnapshotLoad = {
  sessionsByRoute: SessionsState;
  migratedLegacySnapshots: boolean;
};

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

type RuntimeSessionDetailPayload = {
  id?: string;
  status?: string;
  title?: string;
  pinned?: boolean;
  created_at?: string | number;
  updated_at?: string | number;
  last_output_at?: string | number;
  activity_at?: string | number;
  revision?: string | number;
  model_provider_id?: string;
  model_id?: string;
  tool_ids?: string[];
  skill_ids?: string[];
  mcp_ids?: string[];
  turns?: RuntimeSessionTurnPayload[];
  turns_paging?: RuntimeSessionTurnPagingPayload;
};

type RuntimeSessionUpdateEventPayload = {
  event_id?: number | string;
  owner_id?: string;
  session_id?: string;
  event_type?: string;
  revision?: number | string;
  created_at?: string | number;
  payload?: {
    session?: RuntimeSessionDetailPayload;
  };
};

type RuntimeSessionUpdatesResponse = {
  owner_id?: string;
  cursor?: number | string;
  resync_required?: boolean;
  has_more?: boolean;
  events?: RuntimeSessionUpdateEventPayload[];
};

type RuntimeSessionUpdatePollResult = {
  appliedAny: boolean;
  appliedRecoverableProgress: boolean;
  appliedRecoverable: boolean;
};

type RuntimeSessionUpdateAckTurn = {
  id: string;
  event_ids?: string[];
  event_seq_ranges?: Array<[number, number]>;
};

type RuntimeSessionUpdateAckSession = {
  id: string;
  turns?: RuntimeSessionUpdateAckTurn[];
};

type RuntimeSessionUpdatesRequest = {
  since_event_id: string;
  limit: number;
  byte_limit: number;
  sessions: RuntimeSessionUpdateAckSession[];
};

type RuntimeSessionTurnPagingPayload = RuntimeSessionTurnPaging;

type RuntimeSessionTurnPayload = {
  id?: string;
  prompt?: string;
  attachments?: RuntimeSessionAttachmentPayload[];
  status?: string;
  started_at?: string | number;
  finished_at?: string | number;
  final_output?: string;
  runtime_trace_events?: RuntimeTraceEvent[];
  runtime_trace_events_partial?: boolean;
};

type RuntimeSessionAttachmentPayload = {
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
  loadEarlierHistory: () => Promise<boolean>;
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
  | "inspectorOpen"
  | "inspectorTab"
  | "inspectorTabOpen"
  | "draft"
  | "draftAttachments"
  | "busy"
  | "selectedProviderId"
  | "selectedModelId"
  | "selectedModelSupportsVision"
  | "providers"
  | "capabilities"
  | "skills"
  | "runtimeEventFilter"
  | "setDraft"
  | "addDraftAttachments"
  | "removeDraftAttachment"
  | "clearDraftAttachments"
  | "sendPrompt"
  | "toggleInspector"
  | "closeInspector"
  | "selectModel"
  | "toggleCapability"
  | "toggleSkill"
  | "toggleRuntimeEventFilter"
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
  if (normalized.length === 1 && normalized[0] === "important_text") {
    return [...DEFAULT_RUNTIME_EVENT_FILTER];
  }
  return normalized.length > 0 ? normalized : [...DEFAULT_RUNTIME_EVENT_FILTER];
}

function activeSessionStorageKey(route: ConversationRoute): string {
  void route;
  return ACTIVE_SESSION_STORAGE_KEY;
}

function longTermSessionSnapshotStorageKey(route: ConversationRoute): string {
  void route;
  return LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY;
}

function sessionInfoSnapshotStorageKey(route: ConversationRoute): string {
  void route;
  return SESSION_INFO_SNAPSHOT_STORAGE_KEY;
}

function composerDraftStorageKey(route: ConversationRoute): string {
  void route;
  return COMPOSER_DRAFT_STORAGE_KEY;
}

function composerAttachmentDraftStorageKey(route: ConversationRoute): string {
  void route;
  return COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY;
}

function runtimeEventFilterStorageKey(route: ConversationRoute): string {
  void route;
  return RUNTIME_EVENT_FILTER_STORAGE_KEY;
}

function runtimeConfigStorageKey(route: ConversationRoute): string {
  void route;
  return RUNTIME_CONFIG_STORAGE_KEY;
}

function emptyRuntimeComposerConfig(stored = false): RuntimeComposerConfigState {
  return {
    modelProviderID: "",
    modelID: "",
    toolIDs: [],
    skillIDs: [],
    skillIDsExplicit: false,
    mcpIDs: [],
    stored,
  };
}

function normalizeRuntimeComposerConfig(value: unknown, stored = false): RuntimeComposerConfigState {
  if (!value || typeof value !== "object") {
    return emptyRuntimeComposerConfig(stored);
  }
  const record = value as Record<string, unknown>;
  return {
    modelProviderID: normalizeText(record.modelProviderID),
    modelID: normalizeText(record.modelID),
    toolIDs: normalizeSelectionIDs(record.toolIDs),
    skillIDs: normalizeSelectionIDs(record.skillIDs),
    skillIDsExplicit: record.skillIDsExplicit === true,
    mcpIDs: normalizeSelectionIDs(record.mcpIDs),
    stored,
  };
}

function loadRuntimeConfig(route: ConversationRoute): RuntimeComposerConfigState {
  if (typeof window === "undefined") {
    return emptyRuntimeComposerConfig(false);
  }
  try {
    const raw = window.localStorage.getItem(runtimeConfigStorageKey(route));
    if (!raw) {
      return emptyRuntimeComposerConfig(false);
    }
    return normalizeRuntimeComposerConfig(JSON.parse(raw), true);
  } catch {
    return emptyRuntimeComposerConfig(false);
  }
}

function persistRuntimeConfig(route: ConversationRoute, config: RuntimeComposerConfigState) {
  if (typeof window === "undefined") {
    return;
  }
  const normalized = normalizeRuntimeComposerConfig(config, true);
  window.localStorage.setItem(runtimeConfigStorageKey(route), JSON.stringify({
    modelProviderID: normalized.modelProviderID,
    modelID: normalized.modelID,
    toolIDs: normalized.toolIDs,
    skillIDs: normalized.skillIDs,
    skillIDsExplicit: normalized.skillIDsExplicit,
    mcpIDs: normalized.mcpIDs,
  }));
}

function loadRuntimeEventFilter(route: ConversationRoute): RuntimeEventFilterID[] {
  if (typeof window === "undefined") {
    return [...DEFAULT_RUNTIME_EVENT_FILTER];
  }
  try {
    return normalizeRuntimeEventFilter(JSON.parse(window.localStorage.getItem(runtimeEventFilterStorageKey(route)) || "null"));
  } catch {
    return [...DEFAULT_RUNTIME_EVENT_FILTER];
  }
}

function persistRuntimeEventFilter(route: ConversationRoute, filter: RuntimeEventFilterID[]) {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(runtimeEventFilterStorageKey(route), JSON.stringify(normalizeRuntimeEventFilter(filter)));
}

function normalizeConversationRoute(route: string): ConversationRoute {
  void route;
  return "chat";
}

function defaultActiveSessionID(route: ConversationRoute): string {
  void route;
  return CANONICAL_CHAT_SESSION_ID;
}

function newDraftKeyForRoute(route: ConversationRoute): string {
  void route;
  return NEW_CHAT_DRAFT_KEY;
}

function emptySessionsState(): SessionsState {
  return {
    chat: [],
  };
}

function emptyActiveSessionState(): ActiveSessionState {
  return {
    chat: CANONICAL_CHAT_SESSION_ID,
  };
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

function runtimeConfigFromSession(session: ChatSession | null | undefined): RuntimeComposerConfigState {
  if (!session) {
    return emptyRuntimeComposerConfig(false);
  }
  return {
    modelProviderID: normalizeText(session.modelProviderID),
    modelID: normalizeText(session.modelID),
    toolIDs: normalizeSelectionIDs(session.toolIDs),
    skillIDs: normalizeSelectionIDs(session.skillIDs),
    skillIDsExplicit: session.skillIDsExplicit === true,
    mcpIDs: normalizeSelectionIDs(session.mcpIDs),
    stored: false,
  };
}

function effectiveRuntimeSkillIDs(config: RuntimeComposerConfigState, availableSkillIDs: string[] | null): string[] {
  if (config.skillIDsExplicit === false) {
    return availableSkillIDs === null ? [] : [...availableSkillIDs];
  }
  return effectiveChatSkillIDs(config.skillIDs, availableSkillIDs);
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
  const activityDelta = resolveSessionActivityAt(right) - resolveSessionActivityAt(left);
  if (activityDelta !== 0) {
    return activityDelta;
  }
  const createdDelta = right.createdAt - left.createdAt;
  if (createdDelta !== 0) {
    return createdDelta;
  }
  return right.id.localeCompare(left.id, undefined, { numeric: true });
}

export function shouldRefreshChatSessionDetailAfterEmptyUpdates(emptyUpdateAttempts: number): boolean {
  const attempts = Math.max(0, Math.trunc(emptyUpdateAttempts));
  if (attempts <= 0) {
    return false;
  }
  if (CHAT_SESSION_EMPTY_UPDATE_DETAIL_REFRESH_BACKOFF_ATTEMPTS.includes(attempts as 10 | 20 | 50)) {
    return true;
  }
  const lastBackoffAttempt = CHAT_SESSION_EMPTY_UPDATE_DETAIL_REFRESH_BACKOFF_ATTEMPTS[
    CHAT_SESSION_EMPTY_UPDATE_DETAIL_REFRESH_BACKOFF_ATTEMPTS.length - 1
  ];
  return attempts > lastBackoffAttempt
    && attempts % CHAT_SESSION_EMPTY_UPDATE_DETAIL_REFRESH_STEADY_ATTEMPTS === 0;
}

function isBlankDraftSession(session: ChatSession): boolean {
  return session.serverBacked !== true && session.messages.length === 0;
}

function normalizeRouteSessions(routeKey: ConversationRoute, sessions: ChatSession[]): ChatSession[] {
  void routeKey;
  const merged = new Map<string, ChatSession>();
  sessions.forEach((session) => {
    merged.set(session.id, {
      ...session,
      status: normalizeRuntimeSessionDerivedStatus(session),
    });
  });
  return Array.from(merged.values()).sort(compareSessions);
}

function normalizeRuntimeSessionDerivedStatus(session: ChatSession): string {
  const status = normalizeText(session.status) || "ready";
  if (
    !isConversationBusyStatus(status)
    && !isChatRuntimeRuntimeFailureStatus(status)
    && hasRecoverableRuntimeState(session)
  ) {
    return "busy";
  }
  return status;
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
    promptAttachments: message.promptAttachments?.map((attachment) => ({ ...attachment })),
    processEvents: message.processEvents.map(cloneRuntimeTraceEvent),
    processEventsPartial: message.processEventsPartial,
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
    chat: normalizeRouteSessions("chat", (sessionsByRoute.chat || []).map(cloneChatSession)),
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

function normalizeRuntimeSessionTurnPagingPayload(value: unknown): RuntimeSessionTurnPagingPayload | undefined {
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

function compactRuntimeEventSeqRanges(events: RuntimeTraceEvent[]): Array<[number, number]> {
  const seqs = Array.from(new Set(events
    .map((event) => Number(event.seq))
    .filter((seq) => Number.isFinite(seq) && seq > 0)
    .map((seq) => Math.floor(seq))))
    .sort((left, right) => left - right);
  const ranges: Array<[number, number]> = [];
  seqs.forEach((seq) => {
    const last = ranges[ranges.length - 1];
    if (last && seq <= last[1] + 1) {
      last[1] = Math.max(last[1], seq);
      return;
    }
    ranges.push([seq, seq]);
  });
  return ranges;
}

function runtimeEventIDsWithoutSeq(events: RuntimeTraceEvent[]): string[] {
  const ids: string[] = [];
  const seen = new Set<string>();
  for (const event of events) {
    if (Number.isFinite(Number(event.seq)) && Number(event.seq) > 0) {
      continue;
    }
    const id = normalizeText(runtimeTraceEventDetailID(event));
    if (!id || seen.has(id)) {
      continue;
    }
    seen.add(id);
    ids.push(id);
    if (ids.length >= CHAT_SESSION_UPDATE_ACK_EVENT_ID_LIMIT) {
      break;
    }
  }
  return ids;
}

function buildRuntimeSessionUpdateAckManifest(
  sessions: ChatSession[],
  sessionIDs: string[],
): RuntimeSessionUpdateAckSession[] {
  const requested = new Set(sessionIDs.map(normalizeText).filter(Boolean));
  return sessions
    .filter((session) => requested.has(session.id))
    .map((session) => {
      const latestTurnIDs: string[] = [];
      const latestTurnIDSet = new Set<string>();
      for (let index = session.messages.length - 1; index >= 0; index -= 1) {
        const turnID = messageTurnID(session.messages[index].id);
        if (!turnID || latestTurnIDSet.has(turnID)) {
          continue;
        }
        latestTurnIDSet.add(turnID);
        latestTurnIDs.push(turnID);
        if (latestTurnIDs.length >= CHAT_SESSION_UPDATE_ACK_TURN_LIMIT) {
          break;
        }
      }
      latestTurnIDs.reverse();
      const turns = latestTurnIDs.map((turnID) => {
        const processEvents = session.messages
          .filter((message) => message.role === "assistant" && messageTurnID(message.id) === turnID)
          .flatMap((message) => message.processEvents);
        const seqRanges = compactRuntimeEventSeqRanges(processEvents);
        const eventIDs = runtimeEventIDsWithoutSeq(processEvents);
        const ack: RuntimeSessionUpdateAckTurn = { id: turnID };
        if (seqRanges.length > 0) {
          ack.event_seq_ranges = seqRanges;
        }
        if (eventIDs.length > 0) {
          ack.event_ids = eventIDs;
        }
        return ack;
      });
      return { id: session.id, turns };
    });
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

function isPagedRuntimeSessionTurnPayload(
  item: RuntimeSessionDetailPayload,
  paging: RuntimeSessionTurnPagingPayload | undefined,
): boolean {
  if (!paging || !Array.isArray(item.turns)) {
    return false;
  }
  if (paging.has_more_before || paging.has_more_after) {
    return true;
  }
  return typeof paging.total === "number" && item.turns.length < paging.total;
}

function mergeRuntimeSessionTurnPagingPayload(
  previous: ChatSession | null | undefined,
  incoming: RuntimeSessionTurnPagingPayload | undefined,
): RuntimeSessionTurnPagingPayload | undefined {
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
  return message.error || isConversationBusyStatus(message.status) || isStreamingPlaceholderText(message.text);
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

function hasBusyRuntimeMessageState(messages: ChatMessage[]): boolean {
  return messages.some((message) => isConversationBusyStatus(message.status));
}

function shouldBlockRuntimeInput(session: ChatSession): boolean {
  if (session.serverBacked !== true) {
    return false;
  }
  if (isChatRuntimeRuntimeFailureStatus(session.status)) {
    return false;
  }
  return isConversationBusyStatus(session.status) || hasRecoverableRuntimeState(session);
}

function shouldPollRuntimeBackedSession(session: ChatSession): boolean {
  if (session.serverBacked !== true) {
    return false;
  }
  return isConversationBusyStatus(session.status) || hasRecoverableRuntimeState(session);
}

export function resolveRuntimeResyncSessionIDs(sessions: ChatSession[]): string[] {
  return sessions
    .filter(shouldPollRuntimeBackedSession)
    .map((session) => session.id);
}

function hasStableLatestRuntimeSessionCache(
  session: ChatSession | null | undefined,
  options: { allowRecoverable?: boolean } = {},
): session is ChatSession {
  return Boolean(
    session
    && session.serverBacked === true
    && session.messagesLoaded === true
    && (session.revision || 0) <= (session.detailRevision || 0)
    && (options.allowRecoverable === true || !hasRecoverableRuntimeState(session)),
  );
}

function hasFullStableRuntimeSessionCache(session: ChatSession | null | undefined): session is ChatSession {
  return Boolean(
    hasStableLatestRuntimeSessionCache(session)
    && session.turnsPaging?.has_more_before === false,
  );
}

function hasPersistedAssistantState(messages: ChatMessage[]): boolean {
  return messages.some((message) => {
    if (message.role !== "assistant") {
      return false;
    }
    return !isConversationBusyStatus(message.status) && !isStreamingPlaceholderText(message.text);
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

function isOptimisticRuntimeUserMessage(message: ChatMessage): boolean {
  return message.role === "user"
    && message.source === "runtime"
    && messageTurnID(message.id) === ""
    && normalizeText(message.status || "queued") === "queued";
}

function compactOptimisticUserMessagesForParsedTurn(previous: ChatMessage[], parsed: ChatMessage[]): ChatMessage[] {
  const parsedHasUserMessage = parsed.some((message) => message.role === "user");
  if (!parsedHasUserMessage && !hasPersistedAssistantState(parsed)) {
    return previous;
  }
  const parsedUserTexts = new Set(
    parsed
      .filter((message) => message.role === "user")
      .map((message) => normalizeText(message.text)),
  );
  return previous.filter((message) => {
    if (!isOptimisticRuntimeUserMessage(message)) {
      return true;
    }
    if (parsedUserTexts.size === 0) {
      return false;
    }
    return !parsedUserTexts.has(normalizeText(message.text));
  });
}

function previousMessageForIncomingRuntimeMessage(previous: ChatMessage[], incoming: ChatMessage): ChatMessage | undefined {
  const exact = previous.find((message) => message.id === incoming.id);
  if (exact) {
    return exact;
  }
  const incomingTurnID = messageTurnID(incoming.id);
  if (!incomingTurnID) {
    return undefined;
  }
  return previous.find((message) =>
    message.role === incoming.role
    && messageTurnID(message.id) === incomingTurnID,
  );
}

function runtimeTraceEventMergeKey(event: RuntimeTraceEvent): string {
  const detailID = normalizeText(runtimeTraceEventDetailID(event));
  if (detailID) {
    return detailID;
  }
  const eventID = normalizeText(event.id);
  if (eventID) {
    return eventID;
  }
  return `${normalizeText(event.turn_id)}:event:${Number(event.seq) || 0}`;
}

function compareRuntimeTraceEvents(left: RuntimeTraceEvent, right: RuntimeTraceEvent): number {
  const leftSeq = Number(left.seq) || 0;
  const rightSeq = Number(right.seq) || 0;
  if (leftSeq !== rightSeq) {
    return leftSeq - rightSeq;
  }
  return runtimeTraceEventMergeKey(left).localeCompare(runtimeTraceEventMergeKey(right), undefined, { numeric: true });
}

function hasLoadedRuntimeTraceEventDetail(event: RuntimeTraceEvent): boolean {
  return event.raw?.has_detail === false && event.blocks.length > 0;
}

function mergeRuntimeProcessEvent(previous: RuntimeTraceEvent, incoming: RuntimeTraceEvent): RuntimeTraceEvent {
  const previousEvent = cloneRuntimeTraceEvent(previous);
  const incomingEvent = cloneRuntimeTraceEvent(incoming);
  const incomingHasLoadedDetail = incomingEvent.raw?.has_detail === false && incomingEvent.blocks.length > 0;
  if (incomingHasLoadedDetail) {
    return incomingEvent;
  }
  if (!hasLoadedRuntimeTraceEventDetail(previousEvent) && (previousEvent.blocks.length === 0 || incomingEvent.blocks.length > 0)) {
    return incomingEvent;
  }
  return {
    ...previousEvent,
    ...incomingEvent,
    blocks: previousEvent.blocks.map((block) => ({ ...block })),
    raw: {
      ...(incomingEvent.raw || {}),
      ...(previousEvent.raw || {}),
      has_detail: false,
    },
  };
}

function mergeRuntimeProcessEvents(previous: RuntimeTraceEvent[], incoming: RuntimeTraceEvent[]): RuntimeTraceEvent[] {
  if (previous.length === 0) {
    return incoming.map(cloneRuntimeTraceEvent).sort(compareRuntimeTraceEvents);
  }
  if (incoming.length === 0) {
    return previous.map(cloneRuntimeTraceEvent).sort(compareRuntimeTraceEvents);
  }
  const merged = new Map<string, RuntimeTraceEvent>();
  previous.forEach((event) => {
    merged.set(runtimeTraceEventMergeKey(event), cloneRuntimeTraceEvent(event));
  });
  incoming.forEach((event) => {
    const key = runtimeTraceEventMergeKey(event);
    const previousEvent = merged.get(key);
    merged.set(key, previousEvent ? mergeRuntimeProcessEvent(previousEvent, event) : cloneRuntimeTraceEvent(event));
  });
  return Array.from(merged.values()).sort(compareRuntimeTraceEvents);
}

function incomingRuntimeMessageCarriesProcessPatch(previous: ChatMessage | undefined, incoming: ChatMessage): previous is ChatMessage {
  if (!previous) {
    return false;
  }
  if (incoming.processEventsPartial === true) {
    return true;
  }
  return previous.processEvents.length > 0;
}

function mergeIncomingRuntimeMessage(previous: ChatMessage[], incoming: ChatMessage): ChatMessage {
  if (incoming.role !== "assistant") {
    return incoming;
  }
  const previousMessage = previousMessageForIncomingRuntimeMessage(previous, incoming);
  const processEvents = incomingRuntimeMessageCarriesProcessPatch(previousMessage, incoming)
    ? mergeRuntimeProcessEvents(previousMessage.processEvents, incoming.processEvents)
    : incoming.processEvents;
  return {
    ...incoming,
    processEvents,
    processEventsPartial: undefined,
    processCollapsed: typeof previousMessage?.processCollapsed === "boolean"
      ? previousMessage.processCollapsed
      : incoming.processCollapsed,
  };
}

function runtimeMessageRoleOrder(message: ChatMessage): number {
  return message.role === "user" ? 0 : 1;
}

function compareRuntimeMessages(left: ChatMessage, right: ChatMessage): number {
  const leftTurnID = messageTurnID(left.id);
  const rightTurnID = messageTurnID(right.id);
  if (leftTurnID && leftTurnID === rightTurnID) {
    const roleDelta = runtimeMessageRoleOrder(left) - runtimeMessageRoleOrder(right);
    if (roleDelta !== 0) {
      return roleDelta;
    }
  }

  const leftOptimisticUser = isOptimisticRuntimeUserMessage(left);
  const rightOptimisticUser = isOptimisticRuntimeUserMessage(right);
  const leftBusyAssistant = left.role === "assistant" && isConversationBusyStatus(left.status);
  const rightBusyAssistant = right.role === "assistant" && isConversationBusyStatus(right.status);
  if (leftOptimisticUser && rightBusyAssistant) {
    return -1;
  }
  if (rightOptimisticUser && leftBusyAssistant) {
    return 1;
  }

  const leftAt = Number(left.at) || 0;
  const rightAt = Number(right.at) || 0;
  if (leftAt !== rightAt) {
    return leftAt - rightAt;
  }
  if (leftTurnID && rightTurnID && leftTurnID !== rightTurnID) {
    return leftTurnID.localeCompare(rightTurnID, undefined, { numeric: true });
  }
  const roleDelta = runtimeMessageRoleOrder(left) - runtimeMessageRoleOrder(right);
  if (roleDelta !== 0) {
    return roleDelta;
  }
  return left.id.localeCompare(right.id, undefined, { numeric: true });
}

function sortRuntimeMessages(messages: ChatMessage[]): ChatMessage[] {
  return [...messages].sort(compareRuntimeMessages);
}

function mergePagedMessages(previous: ChatMessage[], parsed: ChatMessage[]): ChatMessage[] {
  if (previous.length === 0) {
    return sortRuntimeMessages(parsed);
  }
  if (parsed.length === 0) {
    return sortRuntimeMessages(previous);
  }
  const parsedWithUIState = parsed.map((message) => mergeIncomingRuntimeMessage(previous, message));
  const previousIDs = new Set(previous.map((message) => message.id));
  const hasOverlap = parsedWithUIState.some((message) => previousIDs.has(message.id));
  if (!hasOverlap) {
    const previousFirstAt = previous[0]?.at || 0;
    const previousLastAt = previous[previous.length - 1]?.at || 0;
    const parsedFirstAt = parsedWithUIState[0]?.at || 0;
    const parsedLastAt = parsedWithUIState[parsedWithUIState.length - 1]?.at || 0;
    const shouldAppendNonOverlappingParsedTurn = hasUnansweredLatestUserMessage(previous)
      && hasPersistedAssistantState(parsedWithUIState);
    if (
      shouldAppendNonOverlappingParsedTurn
      ||
      (parsedLastAt > 0 && previousFirstAt > 0 && parsedLastAt < previousFirstAt)
      || (parsedFirstAt > 0 && previousLastAt > 0 && parsedFirstAt > previousLastAt)
    ) {
      return sortRuntimeMessages([...compactOptimisticUserMessagesForParsedTurn(previous, parsedWithUIState), ...parsedWithUIState]);
    }
    return sortRuntimeMessages(shouldUseParsedMessages(previous, parsedWithUIState) ? parsedWithUIState : previous);
  }
  const merged = new Map<string, ChatMessage>();
  compactOptimisticUserMessagesForParsedTurn(previous, parsedWithUIState).forEach((message) => merged.set(message.id, message));
  parsedWithUIState.forEach((message) => merged.set(message.id, message));
  return sortRuntimeMessages(Array.from(merged.values()));
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
  const assistantTextDerivedFromPrompt = optionalBooleanField(
    record,
    "assistantTextDerivedFromPrompt",
    "assistant_text_derived_from_prompt",
  );
  const processEventsPartial = optionalBooleanField(
    record,
    "processEventsPartial",
    "runtime_trace_events_partial",
  );
  return {
    id,
    role,
    text: typeof record.text === "string" ? record.text : "",
    attachments: normalizeStoredAttachments(record.attachments),
    promptText: typeof record.promptText === "string"
      ? record.promptText
      : typeof record.prompt_text === "string"
        ? record.prompt_text
        : undefined,
    promptAttachments: normalizeStoredAttachments(record.promptAttachments ?? record.prompt_attachments),
    assistantTextDerivedFromPrompt,
    route: normalizeText(record.route),
    source: normalizeText(record.source),
    error: Boolean(record.error),
    status: normalizeText(record.status) || (role === "assistant" ? "done" : ""),
    at: Number.isFinite(Number(record.at)) ? Number(record.at) : Date.now(),
    processEvents: normalizeRuntimeTraceEvents(record.runtime_trace_events),
    processEventsPartial,
    processCollapsed:
      typeof record.process_collapsed === "boolean"
        ? record.process_collapsed
        : undefined,
  };
}

function optionalBooleanField(record: Record<string, unknown>, ...keys: string[]): boolean | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "boolean") {
      return value;
    }
  }
  return undefined;
}

function resolveProcessCollapsed(message: ChatMessage): boolean {
  if (typeof message.processCollapsed === "boolean") {
    return message.processCollapsed;
  }
  return Boolean(message.text.trim()) && !isConversationBusyStatus(message.status);
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
  const messages = Array.isArray(record.messages)
    ? sortRuntimeMessages(record.messages.map(normalizeStoredMessage).filter((message): message is ChatMessage => message !== null))
    : [];
  const createdAt = normalizeOptionalDateValue(record.createdAt ?? record.created_at) || Date.now();
  const updatedAt = normalizeOptionalDateValue(record.updatedAt ?? record.updated_at);
  const lastOutputAt = normalizeOptionalDateValue(record.lastOutputAt ?? record.last_output_at);
  const latestMessageAt = latestTimestamp(...messages.map((message) => Number(message.at) || 0));
  const activityAt = normalizeOptionalDateValue(record.activityAt ?? record.activity_at)
    || latestTimestamp(lastOutputAt, updatedAt, latestMessageAt, createdAt);
  const revision = normalizeRevisionValue(record.revision, latestTimestamp(activityAt, updatedAt, lastOutputAt, latestMessageAt, createdAt));
  const detailRevision = normalizeRevisionValue(
    record.detailRevision ?? record.detail_revision,
    (record.messagesLoaded === true || messages.length > 0) ? revision : 0,
  );
  return {
    id,
    sourceRoute: normalizeConversationRoute(normalizeText(record.sourceRoute)),
    status: normalizeText(record.status),
    title: normalizeText(record.title) || "New",
    titleAuto: record.titleAuto !== false,
    titleScore: Number.isFinite(Number(record.titleScore)) ? Number(record.titleScore) : 0,
    createdAt,
    updatedAt,
    lastOutputAt,
    activityAt,
    revision,
    detailRevision,
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
    messages,
    messagesLoaded: typeof record.messagesLoaded === "boolean" ? record.messagesLoaded : undefined,
    serverBacked: typeof record.serverBacked === "boolean" ? record.serverBacked : undefined,
    turnsPaging: normalizeRuntimeSessionTurnPagingPayload(record.turnsPaging ?? record.turns_paging),
  };
}

function normalizeCachedSessionsState(value: unknown): SessionsState {
  if (!value || typeof value !== "object") {
    return emptySessionsState();
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
    prompt_text: message.promptText,
    prompt_attachments: message.promptAttachments,
    assistant_text_derived_from_prompt: message.assistantTextDerivedFromPrompt,
    route: message.route,
    source: message.source,
    error: message.error,
    status: message.status,
    at: message.at,
    runtime_trace_events: message.processEvents,
    runtime_trace_events_partial: message.processEventsPartial,
    process_collapsed: message.processCollapsed,
  };
}

function serializeStoredSession(session: ChatSession): Record<string, unknown> {
  const activityAt = resolveSessionActivityAt(session);
  return {
    id: session.id,
    sourceRoute: session.sourceRoute,
    status: session.status,
    title: session.title,
    titleAuto: session.titleAuto,
    titleScore: session.titleScore,
    createdAt: session.createdAt,
    updatedAt: session.updatedAt,
    lastOutputAt: session.lastOutputAt,
    activityAt,
    revision: session.revision || activityAt,
    detailRevision: session.detailRevision || (session.messagesLoaded ? session.revision || activityAt : 0),
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

function hasJSONStorageItem(key: string): boolean {
  try {
    return window.sessionStorage.getItem(key) !== null;
  } catch {
    return false;
  }
}

function removeJSONStorage(key: string) {
  try {
    window.sessionStorage.removeItem(key);
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
  const legacyParsed = readJSONStorage<Record<string, string>>(ACTIVE_SESSION_STORAGE_KEY, {});
  return CONVERSATION_ROUTES.reduce<ActiveSessionState>((acc, routeKey) => {
    acc[routeKey] =
      readWorkbenchRouteSessionID(routeKey)
      || normalizeText(readJSONStorage<unknown>(activeSessionStorageKey(routeKey), ""))
      || normalizeText(legacyParsed[routeKey])
      || normalizeText(fallback?.[routeKey])
      || defaultActiveSessionID(routeKey);
    return acc;
  }, emptyActiveSessionState());
}

function writeActiveSessionState(activeSessionByRoute: ActiveSessionState, route?: ConversationRoute) {
  (route ? [route] : CONVERSATION_ROUTES).forEach((routeKey) => {
    writeJSONStorage(activeSessionStorageKey(routeKey), normalizeText(activeSessionByRoute[routeKey]));
  });
}

function loadLegacySessionSnapshots(fallback?: SessionsState | null): LegacySessionSnapshotLoad {
  const migratedLegacySnapshots =
    hasJSONStorageItem(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY)
    || hasJSONStorageItem(RECENT_SESSION_SNAPSHOT_STORAGE_KEY);
  const parsedActive = readJSONStorage<StoredActiveSessionSnapshotState>(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY, {});
  const parsedRecent = readJSONStorage<StoredRecentSessionSnapshotState>(RECENT_SESSION_SNAPSHOT_STORAGE_KEY, {});
  const mergeStoredRouteSessions = (routeKey: ConversationRoute) => {
    const sessions = new Map<string, ChatSession>();
    (fallback?.[routeKey] || []).forEach((session) => {
      sessions.set(session.id, { ...cloneChatSession(session), sourceRoute: routeKey });
    });
    normalizeStoredSessionList(parsedRecent[routeKey]).forEach((session) => {
      sessions.set(session.id, { ...session, sourceRoute: routeKey });
    });
    const active = normalizeStoredSession(parsedActive[routeKey]);
    if (active) {
      sessions.set(active.id, { ...active, sourceRoute: routeKey });
    }
    return normalizeRouteSessions(
      routeKey,
      Array.from(sessions.values()).sort(compareSessions),
    );
  };
  return {
    sessionsByRoute: {
      chat: mergeStoredRouteSessions("chat"),
    },
    migratedLegacySnapshots,
  };
}

function clearLegacySessionSnapshots() {
  removeJSONStorage(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY);
  removeJSONStorage(RECENT_SESSION_SNAPSHOT_STORAGE_KEY);
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
  const cache = readMergedPersistentConversationRuntimeCache(
    longTermSessionSnapshotStorageKey,
    true,
    [LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY],
  );
  return cache;
}

function readSessionInfoConversationRuntimeCache(): ConversationRuntimeCacheSnapshot | null {
  const cache = readMergedPersistentConversationRuntimeCache(
    sessionInfoSnapshotStorageKey,
    false,
    [SESSION_INFO_SNAPSHOT_STORAGE_KEY],
  );
  return cache;
}

function readRoutePersistentConversationRuntimeCache(
  routeKey: ConversationRoute,
  storageKey: string,
  removeExpired: boolean,
): ConversationRuntimeCacheSnapshot | null {
  const cache = readJSONLocalStorage<ConversationRuntimeCacheSnapshot | null>(storageKey, null);
  if (!cache) {
    return null;
  }
  if (Date.now() - Number(cache.cachedAt || 0) > CHAT_LONG_TERM_CACHE_TTL_MS) {
    if (removeExpired) {
      try {
        window.localStorage.removeItem(storageKey);
      } catch {
      }
    }
    return null;
  }
  const normalizedActiveSessionID = normalizeText(cache.activeSessionByRoute?.[routeKey]);
  const normalizedSessions = normalizeRouteSessions(routeKey, normalizeCachedSessionsState(cache.sessionsByRoute)[routeKey]);
  if (!normalizedActiveSessionID && normalizedSessions.length === 0) {
    return null;
  }
  return {
    cachedAt: cache.cachedAt,
    activeSessionByRoute: {
      ...emptyActiveSessionState(),
      [routeKey]: normalizedActiveSessionID || defaultActiveSessionID(routeKey),
    },
    sessionsByRoute: {
      ...emptySessionsState(),
      [routeKey]: normalizedSessions,
    },
  };
}

function readMergedPersistentConversationRuntimeCache(
  storageKeyForRoute: (route: ConversationRoute) => string,
  removeExpired: boolean,
  legacyStorageKeys: string[],
): ConversationRuntimeCacheSnapshot | null {
  let cachedAt = 0;
  const activeSessionByRoute = emptyActiveSessionState();
  const sessionsByRoute = emptySessionsState();
  let hasRouteCache = false;
  CONVERSATION_ROUTES.forEach((routeKey) => {
    const storageKeys = Array.from(new Set([storageKeyForRoute(routeKey), ...legacyStorageKeys]));
    const routeCache = storageKeys.reduce<ConversationRuntimeCacheSnapshot | null>((match, storageKey) => {
      return match || readRoutePersistentConversationRuntimeCache(routeKey, storageKey, removeExpired && storageKey === storageKeyForRoute(routeKey));
    }, null);
    if (!routeCache) {
      return;
    }
    hasRouteCache = true;
    cachedAt = Math.max(cachedAt, Number(routeCache.cachedAt || 0));
    activeSessionByRoute[routeKey] = routeCache.activeSessionByRoute[routeKey];
    sessionsByRoute[routeKey] = routeCache.sessionsByRoute[routeKey];
  });
  return hasRouteCache
    ? { cachedAt, activeSessionByRoute, sessionsByRoute }
    : null;
}

function hasRouteCacheData(cache: ConversationRuntimeCacheSnapshot | null, route: ConversationRoute): boolean {
  if (!cache) {
    return false;
  }
  const activeSessionID = normalizeText(cache.activeSessionByRoute?.[route]);
  return (cache.sessionsByRoute?.[route] || []).length > 0
    || (!!activeSessionID && activeSessionID !== defaultActiveSessionID(route));
}

function mergeConversationRuntimeCacheSnapshots(
  primary: ConversationRuntimeCacheSnapshot | null,
  fallback: ConversationRuntimeCacheSnapshot | null,
): ConversationRuntimeCacheSnapshot | null {
  if (!primary) {
    return fallback;
  }
  if (!fallback) {
    return primary;
  }
  const activeSessionByRoute = emptyActiveSessionState();
  const sessionsByRoute = emptySessionsState();
  CONVERSATION_ROUTES.forEach((routeKey) => {
    const source = hasRouteCacheData(primary, routeKey) ? primary : fallback;
    activeSessionByRoute[routeKey] = source.activeSessionByRoute[routeKey];
    sessionsByRoute[routeKey] = normalizeRouteSessions(
      routeKey,
      mergeRuntimeSessions(
        primary.sessionsByRoute?.[routeKey] || [],
        fallback.sessionsByRoute?.[routeKey] || [],
      ),
    );
  });
  return {
    cachedAt: Math.max(Number(primary.cachedAt || 0), Number(fallback.cachedAt || 0)),
    activeSessionByRoute,
    sessionsByRoute,
  };
}

function writeConversationRuntimeCache(activeSessionByRoute: ActiveSessionState, sessionsByRoute: SessionsState) {
  conversationRuntimeCache = {
    cachedAt: Date.now(),
    activeSessionByRoute: {
      chat: normalizeText(activeSessionByRoute.chat) || CANONICAL_CHAT_SESSION_ID,
    },
    sessionsByRoute: cloneSessionsState(sessionsByRoute),
  };
}

function writeLongTermConversationRuntimeCache(
  activeSessionByRoute: ActiveSessionState,
  sessionsByRoute: SessionsState,
  route?: ConversationRoute,
) {
  const cachedAt = Date.now();
  (route ? [route] : CONVERSATION_ROUTES).forEach((routeKey) => {
    writeJSONLocalStorage(longTermSessionSnapshotStorageKey(routeKey), {
      cachedAt,
      activeSessionByRoute: {
        ...emptyActiveSessionState(),
        [routeKey]: normalizeText(activeSessionByRoute[routeKey]) || defaultActiveSessionID(routeKey),
      },
      sessionsByRoute: {
        ...emptySessionsState(),
        [routeKey]: normalizeRouteSessions(routeKey, (sessionsByRoute[routeKey] || []).map(cloneChatSession)),
      },
    });
  });
}

function writeSessionInfoConversationRuntimeCache(
  activeSessionByRoute: ActiveSessionState,
  sessionsByRoute: SessionsState,
  route?: ConversationRoute,
) {
  const cachedAt = Date.now();
  (route ? [route] : CONVERSATION_ROUTES).forEach((routeKey) => {
    writeJSONLocalStorage(sessionInfoSnapshotStorageKey(routeKey), {
      cachedAt,
      activeSessionByRoute: {
        ...emptyActiveSessionState(),
        [routeKey]: normalizeText(activeSessionByRoute[routeKey]) || defaultActiveSessionID(routeKey),
      },
      sessionsByRoute: {
        ...emptySessionsState(),
        [routeKey]: normalizeRouteSessions(routeKey, (sessionsByRoute[routeKey] || []).map(trimSessionForInfoCache)),
      },
    });
  });
}

function writeConversationRuntimeCaches(
  activeSessionByRoute: ActiveSessionState,
  sessionsByRoute: SessionsState,
  route?: ConversationRoute,
) {
  writeConversationRuntimeCache(activeSessionByRoute, sessionsByRoute);
  writeLongTermConversationRuntimeCache(activeSessionByRoute, sessionsByRoute, route);
  writeSessionInfoConversationRuntimeCache(activeSessionByRoute, sessionsByRoute, route);
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
  const infoCache = readSessionInfoConversationRuntimeCache();
  const fallbackCache = mergeConversationRuntimeCacheSnapshots(longTermCache, infoCache);
  const activeSessionByRoute = loadActiveSessionState(fallbackCache?.activeSessionByRoute || null);
  const snapshotLoad = loadLegacySessionSnapshots(fallbackCache?.sessionsByRoute || null);
  if (snapshotLoad.migratedLegacySnapshots) {
    writeConversationRuntimeCaches(activeSessionByRoute, snapshotLoad.sessionsByRoute);
    clearLegacySessionSnapshots();
  }
  return {
    activeSessionByRoute,
    sessionsByRoute: snapshotLoad.sessionsByRoute,
  };
}

function loadComposerDrafts(route: ConversationRoute): ComposerDraftMap {
  const parsed = readJSONStorage<Record<string, string>>(composerDraftStorageKey(route), {});
  return Object.entries(parsed).reduce<ComposerDraftMap>((acc, [key, value]) => {
    const normalizedKey = normalizeText(key);
    if (!normalizedKey || typeof value !== "string") {
      return acc;
    }
    acc[normalizedKey] = value;
    return acc;
  }, {});
}

function persistComposerDrafts(route: ConversationRoute, drafts: ComposerDraftMap) {
  writeJSONStorage(composerDraftStorageKey(route), drafts);
}

function loadComposerAttachmentDrafts(route: ConversationRoute): ComposerAttachmentDraftMap {
  const parsed = readJSONStorage<Record<string, unknown>>(composerAttachmentDraftStorageKey(route), {});
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

function persistComposerAttachmentDrafts(route: ConversationRoute, drafts: ComposerAttachmentDraftMap) {
  writeJSONStorage(composerAttachmentDraftStorageKey(route), drafts);
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

function normalizeOptionalDateValue(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return value;
  }
  if (typeof value === "string" && value.trim()) {
    const parsed = Date.parse(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function normalizeRevisionValue(value: unknown, fallback: number): number {
  const parsed = Number(value);
  if (Number.isFinite(parsed) && parsed > 0) {
    return parsed;
  }
  return fallback;
}

function latestTimestamp(...values: number[]): number {
  return Math.max(0, ...values.filter((value) => Number.isFinite(value) && value > 0));
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

function normalizeRuntimeTurnMessages(sessionID: string, turn: RuntimeSessionTurnPayload, route: ConversationRoute): ChatMessage[] {
  const id = normalizeText(turn.id);
  if (!id) {
    return [];
  }
  const messages = runtimeSessionTurnsToTimelineMessages({
    sessionID,
    turns: [turn],
    route,
    source: "runtime",
  });
  return markRuntimeTurnMessagesProcessEventsPartial(messages, turn.runtime_trace_events_partial === true);
}

function markRuntimeTurnMessagesProcessEventsPartial(messages: ChatMessage[], partial: boolean): ChatMessage[] {
  if (!partial) {
    return messages;
  }
  return messages.map((message) => {
    if (message.role !== "assistant") {
      return message;
    }
    return { ...message, processEventsPartial: true };
  });
}

function normalizeRuntimeSession(
  item: RuntimeSessionDetailPayload,
  previous?: ChatSession | null,
  sourceRoute: ConversationRoute = "chat",
  context: RuntimeSessionNormalizeContext = { source: "detail" },
): ChatSession | null {
  const id = normalizeText(item.id);
  if (!id) {
    return null;
  }
  const hasDetailTurns = Array.isArray(item.turns)
    && (context.source !== "summary" || item.turns.length > 0);
  const rawParsedMessages = hasDetailTurns
    ? item.turns.flatMap((turn) => normalizeRuntimeTurnMessages(id, turn, sourceRoute))
    : null;
  const parsedMessages = rawParsedMessages && previous?.messages.length
    ? rawParsedMessages.map((message) => mergeIncomingRuntimeMessage(previous.messages, message))
    : rawParsedMessages;
  const incomingPaging = normalizeRuntimeSessionTurnPagingPayload(item.turns_paging);
  const shouldMergeRuntimeMessages = parsedMessages
    && previous?.messages.length
    && (Boolean(incomingPaging) || isPagedRuntimeSessionTurnPayload(item, incomingPaging) || parsedMessages.length < previous.messages.length);
  const messages = parsedMessages
    ? (shouldMergeRuntimeMessages
      ? mergePagedMessages(previous.messages, parsedMessages)
      : previous?.messages.length && !shouldUseParsedMessages(previous.messages, parsedMessages)
        ? previous.messages
        : parsedMessages)
    : previous?.messages || [];
  const hasExplicitSkillIDs = Array.isArray(item.skill_ids);
  const createdAt = normalizeOptionalDateValue(item.created_at) || previous?.createdAt || Date.now();
  const updatedAt = normalizeOptionalDateValue(item.updated_at);
  const lastOutputAt = normalizeOptionalDateValue(item.last_output_at);
  const latestMessageAt = latestTimestamp(...messages.map((message) => Number(message.at) || 0));
  const activityAt = normalizeOptionalDateValue(item.activity_at)
    || latestTimestamp(lastOutputAt, updatedAt, latestMessageAt, createdAt);
  const explicitRevision = normalizeRevisionValue(item.revision, 0);
  const inferredRevision = latestTimestamp(activityAt, updatedAt, lastOutputAt, createdAt, previous?.revision || 0);
  const revision = explicitRevision
    || inferredRevision;
  const detailRevision = hasDetailTurns
    ? revision
    : previous?.detailRevision || 0;
  return {
    id,
    sourceRoute: previous?.sourceRoute || sourceRoute,
    status: normalizeText(item.status) || previous?.status || "ready",
    title: normalizeText(item.title) || previous?.title || "New",
    titleAuto: previous?.titleAuto ?? true,
    titleScore: previous?.titleScore || 0,
    createdAt,
    updatedAt,
    lastOutputAt,
    activityAt,
    revision,
    detailRevision,
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
    messagesLoaded: hasDetailTurns ? true : previous?.messagesLoaded,
    serverBacked: true,
    turnsPaging: mergeRuntimeSessionTurnPagingPayload(previous, incomingPaging),
  };
}

function isChatRuntimeRuntimeFailureStatus(status: string): boolean {
  return ["error", "failed", "canceled", "cancelled", "interrupted"].includes(normalizeTaskStatus(status));
}

function resolveMergedRuntimeSessionStatus(previous: ChatSession | undefined, incoming: ChatSession, messages: ChatMessage[]): string {
  const incomingStatus = incoming.status || previous?.status || "";
  if (!previous || isConversationBusyStatus(incomingStatus) || isChatRuntimeRuntimeFailureStatus(incomingStatus)) {
    return incomingStatus;
  }
  if (!shouldPollRuntimeBackedSession(previous)) {
    return incomingStatus;
  }
  const incomingHasCompleteRuntimeView = incoming.messagesLoaded === true;
  const incomingStillRecoverable = hasRecoverableRuntimeState({ ...incoming, messages });
  if (incomingStillRecoverable || (!incomingHasCompleteRuntimeView && incoming.messages.length === 0)) {
    return isConversationBusyStatus(previous.status) ? previous.status : "busy";
  }
  return incomingStatus;
}

function runtimeSessionProcessEventCount(session: ChatSession | null | undefined): number {
  if (!session) {
    return 0;
  }
  return session.messages.reduce((count, message) => count + message.processEvents.length, 0);
}

function runtimeSessionUpdateMadeProgress(previous: ChatSession | null | undefined, incoming: ChatSession): boolean {
  if (!previous) {
    return true;
  }
  const previousRevision = previous.revision || resolveSessionActivityAt(previous);
  const incomingRevision = incoming.revision || resolveSessionActivityAt(incoming);
  if (incomingRevision > previousRevision) {
    return true;
  }
  if (resolveSessionActivityAt(incoming) > resolveSessionActivityAt(previous)) {
    return true;
  }
  if (incoming.messages.length > previous.messages.length) {
    return true;
  }
  if (runtimeSessionProcessEventCount(incoming) > runtimeSessionProcessEventCount(previous)) {
    return true;
  }
  return incoming.status !== previous.status && isConversationBusyStatus(incoming.status);
}

function runtimeSessionPayloadFromEventData(value: unknown): RuntimeSessionDetailPayload | null {
  if (!value || typeof value !== "object") {
    return null;
  }
  const record = value as Record<string, unknown>;
  const direct = record.session;
  if (direct && typeof direct === "object") {
    return direct as RuntimeSessionDetailPayload;
  }
  const payload = record.payload;
  if (payload && typeof payload === "object") {
    const nested = (payload as Record<string, unknown>).session;
    if (nested && typeof nested === "object") {
      return nested as RuntimeSessionDetailPayload;
    }
  }
  return null;
}

export function mergeRuntimeSessions(remote: ChatSession[], existing: ChatSession[]): ChatSession[] {
  const merged = new Map<string, ChatSession>();
  const existingByID = new Map(existing.map((session) => [session.id, session]));
  remote.forEach((session) => {
    const previous = existingByID.get(session.id);
    const incomingRevision = session.revision || resolveSessionActivityAt(session);
    const previousRevision = previous ? previous.revision || resolveSessionActivityAt(previous) : 0;
    const incomingOlderThanPrevious = Boolean(previous && incomingRevision > 0 && previousRevision > incomingRevision);
    const incomingStaleSummaryAgainstStableDetail = Boolean(
      previous
      && session.messagesLoaded !== true
      && session.messages.length === 0
      && previous.messagesLoaded === true
      && previous.detailRevision > 0
      && previous.detailRevision >= incomingRevision
      && !isConversationBusyStatus(previous.status)
      && !hasRecoverableRuntimeState(previous)
      && isConversationBusyStatus(session.status),
    );
    const messages = previous && session.messagesLoaded === true
      ? mergePagedMessages(previous.messages, session.messages)
      : session.messages.length > 0
        ? session.messages
        : previous?.messages || [];
    const compactedMessages = session.messages.length > 0
      ? compactOptimisticUserMessagesForParsedTurn(messages, session.messages)
      : messages;
    const mergedSession = {
      ...previous,
      ...session,
      ...(incomingOlderThanPrevious ? {
        status: previous?.status,
        title: previous?.title,
        titleAuto: previous?.titleAuto,
        titleScore: previous?.titleScore,
        updatedAt: previous?.updatedAt,
        lastOutputAt: previous?.lastOutputAt,
        activityAt: previous?.activityAt,
        revision: previous?.revision,
        detailRevision: previous?.detailRevision,
        pinned: previous?.pinned,
        turnsPaging: previous?.turnsPaging,
      } : {}),
      ...(incomingStaleSummaryAgainstStableDetail ? {
        status: previous?.status,
        messagesLoaded: previous?.messagesLoaded,
        detailRevision: previous?.detailRevision,
        turnsPaging: previous?.turnsPaging,
      } : {}),
      status: incomingOlderThanPrevious
        ? previous?.status || session.status
        : incomingStaleSummaryAgainstStableDetail
          ? previous?.status || session.status
          : resolveMergedRuntimeSessionStatus(previous, session, compactedMessages),
      messages: compactedMessages,
      messagesLoaded:
        incomingStaleSummaryAgainstStableDetail
          ? previous?.messagesLoaded
          : typeof session.messagesLoaded === "boolean"
          ? session.messagesLoaded
          : previous?.messagesLoaded,
      serverBacked:
        typeof session.serverBacked === "boolean"
          ? session.serverBacked
          : previous?.serverBacked,
      turnsPaging: incomingOlderThanPrevious || incomingStaleSummaryAgainstStableDetail
        ? previous?.turnsPaging
        : session.turnsPaging || previous?.turnsPaging,
    };
    merged.set(session.id, {
      ...mergedSession,
      activityAt: resolveSessionActivityAt(mergedSession),
      revision: mergedSession.revision || resolveSessionActivityAt(mergedSession),
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
  return latestTimestamp(
    session.activityAt,
    session.lastOutputAt,
    session.updatedAt,
    latestMessageAt,
    session.createdAt,
  );
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
  const provider = available.find((item) => normalizeText(item.id) === CODEX_RUNTIME_PROVIDER_ID)
    || available.find((item) => item.is_default)
    || available[0]
    || null;
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

function resolveModelSelection(selectionConfig: Pick<RuntimeComposerConfigState, "modelProviderID" | "modelID"> | null, providers: ChatProvider[]) {
  const fallback = defaultModelSelection(providers);
  const providerID = normalizeText(selectionConfig?.modelProviderID) || fallback.providerID;
  const provider = enabledProviders(providers).find((item) => normalizeText(item.id) === providerID) || null;
  if (!provider) {
    return fallback;
  }
  const models = enabledModels(provider);
  const preferredModelID = normalizeText(selectionConfig?.modelID);
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

function runtimeSelectionInputPayload(selection: { providerID: string; modelID: string }): Record<string, string> {
  if (normalizeText(selection.providerID) === CODEX_RUNTIME_PROVIDER_ID) {
    return { execution_engine: "codex" };
  }
  return {
    model_provider_id: normalizeText(selection.providerID),
    model_id: normalizeText(selection.modelID),
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
  const initialSessionsByRoute = initialRuntimeStateRef.current?.sessionsByRoute || emptySessionsState();
  const initialActiveSessionByRoute = initialRuntimeStateRef.current?.activeSessionByRoute || emptyActiveSessionState();
  const sessionsByRouteRef = useRef<SessionsState>(initialSessionsByRoute);
  const activeSessionIDRef = useRef<string>(initialActiveSessionByRoute[route] || defaultActiveSessionID(route));
  const [sessionsLoadedByRoute, setSessionsLoadedByRoute] = useState<Record<ConversationRoute, boolean>>({
    chat: false,
  });
  const runtimeCatalogs = useRuntimeSessionCatalogs(apiClient);
  const providers = runtimeCatalogs.providers as ChatProvider[];
  const skills = runtimeCatalogs.skills as ChatCapability[];
  const skillCatalogLoaded = runtimeCatalogs.skillsLoaded;
  const mcps = runtimeCatalogs.mcps as ChatCapability[];
  const [composerDrafts, setComposerDrafts] = useState<ComposerDraftMap>(() => loadComposerDrafts(route));
  const [composerAttachmentDrafts, setComposerAttachmentDrafts] = useState<ComposerAttachmentDraftMap>(() => loadComposerAttachmentDrafts(route));
  const [compact, setCompact] = useState(() => isCompactViewport());
  const [inspectorOpen, setInspectorOpen] = useState(false);
  const [inspectorTab, setInspectorTab] = useState<"model" | "capabilities" | "skills">("model");
  const [inspectorTabOpen, setInspectorTabOpen] = useState(true);
  const [runtimeConfig, setRuntimeConfig] = useState<RuntimeComposerConfigState>(() => loadRuntimeConfig(route));
  const [runtimeEventFilter, setRuntimeEventFilter] = useState<RuntimeEventFilterID[]>(() => loadRuntimeEventFilter(route));
  const [pinningSessionIDs, setPinningSessionIDs] = useState<Record<string, boolean>>({});
  const [pageHidden, setPageHidden] = useState(() => typeof document !== "undefined" && document.hidden);
  const pollTimerRef = useRef<number>(0);
  const updateCursorByRouteRef = useRef<Record<ConversationRoute, string>>({ chat: "0" });
  const recoveryPromisesRef = useRef(new Map<string, Promise<ChatSession | null>>());
  const earlierHistoryLoadKeysRef = useRef(new Set<string>());
  const processEventDetailLoadsRef = useRef(new Set<string>());
  const composerDraftPersistTimerRef = useRef<number>(0);
  const sendPromptRef = useRef<(prompt?: string) => Promise<void>>(async () => undefined);
  const latestComposerDraftsRef = useRef<ComposerDraftMap>(composerDrafts);
  const latestComposerAttachmentDraftsRef = useRef<ComposerAttachmentDraftMap>(composerAttachmentDrafts);
  const [fallbackPollAttempt, setFallbackPollAttempt] = useState(0);
  const runtimeSessionControllerOptions = useMemo(() => ({
    route,
    initialSessions: initialSessionsByRoute[route],
    initialActiveSessionID: initialActiveSessionByRoute[route],
    normalizeSession: (payload: RuntimeSessionPayload, previous, context) => {
      const sessionID = normalizeText(payload.id);
      const localPrevious = sessionID
        ? sessionsByRouteRef.current[route].find((session) => session.id === sessionID) || null
        : null;
      return normalizeRuntimeSession(payload as RuntimeSessionDetailPayload, localPrevious || previous || null, route, context);
    },
    mergeSession: (previous, incoming) => mergeRuntimeSessions([incoming], previous ? [previous] : [])[0] || incoming,
    sortSessions: (items) => normalizeRouteSessions(route, items),
    getProgressiveHistoryPaging: (session) => session.turnsPaging,
    getProgressiveHistoryTurnBefore: (session) =>
      normalizeText(session.turnsPaging?.next_before_turn_id || session.turnsPaging?.oldest_turn_id)
      || oldestTurnIDFromMessages(session.messages),
    canLoadProgressiveHistory: (session) =>
      session.serverBacked === true
      && session.messagesLoaded === true
      && !hasRecoverableRuntimeState(session),
    enableProgressiveHistory: false,
  }), [initialActiveSessionByRoute, initialSessionsByRoute, route]);
  const runtimeSessionController = useRuntimeSessionController<ChatSession>(runtimeSessionControllerOptions);
  const {
    createSession: createRuntimeSession,
    deleteSession: deleteRuntimeSession,
    setSessionPinned: setRuntimeSessionPinned,
    refreshList: refreshRuntimeSessions,
    refreshActiveSession: refreshRuntimeSession,
    sendInput: sendRuntimeInput,
    uploadAttachments: uploadRuntimeAttachments,
    loadEventDetail: loadRuntimeEventDetail,
    setSessions: setRuntimeSessions,
    setActiveSessionID: setRuntimeActiveSessionID,
  } = runtimeSessionController;
  const sessionsByRoute = useMemo<SessionsState>(
    () => ({
      ...initialSessionsByRoute,
      [route]: runtimeSessionController.sessions,
    }),
    [initialSessionsByRoute, route, runtimeSessionController.sessions],
  );
  const activeSessionByRoute = useMemo<ActiveSessionState>(
    () => ({
      ...initialActiveSessionByRoute,
      [route]: runtimeSessionController.activeSessionID || defaultActiveSessionID(route),
    }),
    [initialActiveSessionByRoute, route, runtimeSessionController.activeSessionID],
  );
  const setSessionsByRoute = useCallback((updater: SessionsState | ((current: SessionsState) => SessionsState)) => {
    const currentRouteSessions = normalizeRouteSessions(
      route,
      mergeRuntimeSessions(
        sessionsByRouteRef.current[route] || [],
        runtimeSessionController.sessionsRef.current || [],
      ),
    );
    const current = {
      ...sessionsByRouteRef.current,
      [route]: currentRouteSessions,
    };
    const next = typeof updater === "function" ? updater(current) : updater;
    sessionsByRouteRef.current = next;
    writeConversationRuntimeCaches(activeSessionByRoute, next, route);
    setRuntimeSessions(next[route]);
  }, [activeSessionByRoute, route, runtimeSessionController.sessionsRef, setRuntimeSessions]);
  const setActiveSessionByRoute = useCallback((updater: ActiveSessionState | ((current: ActiveSessionState) => ActiveSessionState)) => {
    const current = {
      ...activeSessionByRoute,
      [route]: runtimeSessionController.activeSessionID || defaultActiveSessionID(route),
    };
    const next = typeof updater === "function" ? updater(current) : updater;
    activeSessionIDRef.current = next[route] || defaultActiveSessionID(route);
    writeConversationRuntimeCaches(next, sessionsByRouteRef.current, route);
    setRuntimeActiveSessionID(next[route]);
  }, [activeSessionByRoute, route, runtimeSessionController.activeSessionID, setRuntimeActiveSessionID]);

  const activeSessions = useMemo(
    () => normalizeRouteSessions(route, sessionsByRoute[route]),
    [route, sessionsByRoute],
  );
  const activeSessionReference = activeSessionByRoute[route];
  const activeSessionID = resolveSessionIDReference(activeSessions, activeSessionReference) || activeSessionReference;
  useEffect(() => {
    activeSessionIDRef.current = activeSessionID;
  }, [activeSessionID]);
  const activeSession = useMemo(() => {
    const session = activeSessions.find((item) => item.id === activeSessionID) || null;
    return session
      ? { ...session, status: normalizeRuntimeSessionDerivedStatus(session) }
      : null;
  }, [activeSessionID, activeSessions]);
  const activeDraftKey = activeSessionID || newDraftKeyForRoute(route);
  const activeDraftAttachments = composerAttachmentDrafts[activeDraftKey] || EMPTY_COMPOSER_ATTACHMENTS;
  const availableProviders = useMemo(() => runtimeProviders(providers), [providers]);
  const availableSkillIDs = useMemo(
    () => skillCatalogLoaded ? defaultChatSkillIDs(skills) : null,
    [skillCatalogLoaded, skills],
  );
  const activeSessionConfigKey = [
    activeSession?.modelProviderID || "",
    activeSession?.modelID || "",
    (activeSession?.toolIDs || []).join("\u0000"),
    (activeSession?.skillIDs || []).join("\u0000"),
    activeSession?.skillIDsExplicit === true ? "skills-explicit" : "skills-default",
    (activeSession?.mcpIDs || []).join("\u0000"),
  ].join("\u0001");
  const activeRuntimeConfig = useMemo(
    () => runtimeConfig.stored ? runtimeConfig : runtimeConfigFromSession(activeSession),
    // activeSessionConfigKey intentionally keeps history/message merges out of Composer context.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [activeSessionConfigKey, runtimeConfig],
  );
  const activeSkillIDs = useMemo(
    () => effectiveRuntimeSkillIDs(activeRuntimeConfig, availableSkillIDs),
    [activeRuntimeConfig, availableSkillIDs],
  );
  useEffect(() => {
    setRuntimeConfig(loadRuntimeConfig(route));
  }, [route]);
  useEffect(() => {
    latestComposerDraftsRef.current = composerDrafts;
    window.clearTimeout(composerDraftPersistTimerRef.current);
    composerDraftPersistTimerRef.current = window.setTimeout(() => {
      persistComposerDrafts(route, latestComposerDraftsRef.current);
      composerDraftPersistTimerRef.current = 0;
    }, COMPOSER_DRAFT_PERSIST_DELAY_MS);
    return () => window.clearTimeout(composerDraftPersistTimerRef.current);
  }, [composerDrafts]);

  useEffect(() => {
    latestComposerAttachmentDraftsRef.current = composerAttachmentDrafts;
  }, [composerAttachmentDrafts]);

  useEffect(() => {
    const flushComposerDrafts = () => {
      window.clearTimeout(composerDraftPersistTimerRef.current);
      composerDraftPersistTimerRef.current = 0;
      persistComposerDrafts(route, latestComposerDraftsRef.current);
      persistComposerAttachmentDrafts(route, latestComposerAttachmentDraftsRef.current);
    };
    window.addEventListener("pagehide", flushComposerDrafts);
    window.addEventListener("beforeunload", flushComposerDrafts);
    return () => {
      window.removeEventListener("pagehide", flushComposerDrafts);
      window.removeEventListener("beforeunload", flushComposerDrafts);
    };
  }, [route]);

  useEffect(() => () => {
    window.clearTimeout(composerDraftPersistTimerRef.current);
    persistComposerDrafts(route, latestComposerDraftsRef.current);
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
  }, [setSessionsByRoute]);

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
      const detail = await loadRuntimeEventDetail(session.id, turnID, detailID);
      const detailedEvent = normalizeRuntimeTraceEventDetail(detail as RuntimeTraceEventDetail | undefined, runtimeEvent);
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
  }, [activeSession, activeSessionByRoute, loadRuntimeEventDetail, patchSession, route]);

  const focusSession = useCallback((sessionID: string) => {
    const resolvedSessionID = sessionID;
    const nextActiveState = { ...activeSessionByRoute, [route]: resolvedSessionID };
    activeSessionIDRef.current = resolvedSessionID;
    setActiveSessionByRoute(nextActiveState);
    writeActiveSessionState(nextActiveState, route);
    writeWorkbenchRouteSessionID(route, resolvedSessionID);
  }, [activeSessionByRoute, route]);

  const removeSession = useCallback(async (sessionID: string) => {
    try {
      await deleteRuntimeSession(sessionID);
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
    persistComposerDrafts(route, nextDrafts);
    persistComposerAttachmentDrafts(route, nextAttachmentDrafts);
    writeActiveSessionState(nextActiveState, route);
  }, [activeSessionByRoute, deleteRuntimeSession, route, sessionsByRoute]);

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
      await setRuntimeSessionPinned(normalizedSessionID, pinned);
      applyPinnedState();
    } catch {
      applyPinnedState();
    } finally {
      setPinningSessionIDs((current) => ({ ...current, [normalizedSessionID]: false }));
    }
  }, [patchSession, route, setRuntimeSessionPinned]);

  const upsertRuntimeSession = useCallback((routeKey: ConversationRoute, nextSession: ChatSession) => {
    const normalizedSession = { ...nextSession, sourceRoute: nextSession.sourceRoute || routeKey };
    setSessionsByRoute((current) => {
      const hasSession = current[routeKey].some((session) => session.id === normalizedSession.id);
      const nextSessions = hasSession
        ? current[routeKey].map((session) =>
            session.id === normalizedSession.id
              ? mergeRuntimeSessions([normalizedSession], [session])[0] || normalizedSession
              : session,
          )
        : [normalizedSession, ...current[routeKey]];
      const nextState = {
        ...current,
        [routeKey]: normalizeRouteSessions(routeKey, nextSessions),
      };
      sessionsByRouteRef.current = nextState;
      return nextState;
    });
  }, [setSessionsByRoute]);

  const createRuntimeBackedSession = useCallback(async (routeKey: ConversationRoute, title: string = ""): Promise<ChatSession | null> => {
    const nextSession = await createRuntimeSession(
      normalizeText(title) ? { title: normalizeText(title).slice(0, 80) } : {},
    );
    if (!nextSession) {
      return null;
    }
    upsertRuntimeSession(routeKey, nextSession);
    setActiveSessionByRoute((current) => {
      const nextActiveState = { ...current, [routeKey]: nextSession.id };
      writeActiveSessionState(nextActiveState, routeKey);
      return nextActiveState;
    });
    writeWorkbenchRouteSessionID(routeKey, nextSession.id);
    return nextSession;
  }, [createRuntimeSession, upsertRuntimeSession]);

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

  const commitRuntimeConfig = useCallback((nextConfig: RuntimeComposerConfigState) => {
    const normalized = normalizeRuntimeComposerConfig(nextConfig, true);
    setRuntimeConfig(normalized);
    persistRuntimeConfig(route, normalized);
  }, [route]);

  const hydrateRuntimeSession = async (
    routeKey: ConversationRoute,
    sessionID: string,
    options: { turnBefore?: string; turnLimit?: number; force?: boolean; allowRecoverableCache?: boolean } = {},
  ): Promise<ChatSession | null> => {
    void routeKey;
    const normalizedSessionID = normalizeText(sessionID);
    const isProgressiveHistoryRequest = Boolean(normalizeText(options.turnBefore));
    const cachedSession = normalizedSessionID
      ? sessionsByRouteRef.current[routeKey].find((session) => session.id === normalizedSessionID) || null
      : null;
    if (
      !options.force
      && !isProgressiveHistoryRequest
      && hasStableLatestRuntimeSessionCache(cachedSession, { allowRecoverable: options.allowRecoverableCache === true })
    ) {
      return cachedSession;
    }
    return refreshRuntimeSession(sessionID, options);
  };

  const refreshActiveSession = useCallback(async () => {
    const sessionID = normalizeText(activeSession?.id);
    if (!sessionID || activeSession?.serverBacked !== true) {
      return;
    }
    try {
      const hydrated = await hydrateRuntimeSession(route, sessionID, { force: true });
      if (hydrated) {
        upsertRuntimeSession(route, hydrated);
      }
    } catch {
    }
  }, [activeSession, route, upsertRuntimeSession]);

  const loadEarlierHistory = useCallback(async () => {
    const sessionID = normalizeText(activeSessionID);
    const session = sessionID
      ? sessionsByRouteRef.current[route].find((item) => item.id === sessionID) || activeSession
      : activeSession;
    if (
      !session?.id
      || session.serverBacked !== true
      || session.messagesLoaded !== true
      || hasRecoverableRuntimeState(session)
      || session.turnsPaging?.has_more_before !== true
    ) {
      return false;
    }
    const beforeTurnID = normalizeText(
      session.turnsPaging?.next_before_turn_id
      || session.turnsPaging?.oldest_turn_id
      || oldestTurnIDFromMessages(session.messages),
    );
    if (!beforeTurnID) {
      return false;
    }
    const requestKey = `${route}:${session.id}:${beforeTurnID}`;
    if (earlierHistoryLoadKeysRef.current.has(requestKey)) {
      return false;
    }
    earlierHistoryLoadKeysRef.current.add(requestKey);
    try {
      const payload = await apiClient.get<{ session?: RuntimeSessionDetailPayload }>(
        runtimeSessionDetailEndpoint(route, session.id, {
          turnBefore: beforeTurnID,
          turnLimit: RUNTIME_SESSION_HISTORY_PAGE_TURN_LIMIT,
        }),
      );
      if (!payload.session) {
        return false;
      }
      const latestSession = sessionsByRouteRef.current[route].find((item) => item.id === session.id) || session;
      const normalized = normalizeRuntimeSession(payload.session, latestSession, route, { source: "detail" });
      if (!normalized) {
        return false;
      }
      const merged = mergeRuntimeSessions([normalized], [latestSession])[0] || normalized;
      const madeProgress = (
        merged.messages.length > latestSession.messages.length
        || merged.turnsPaging?.next_before_turn_id !== latestSession.turnsPaging?.next_before_turn_id
        || merged.turnsPaging?.has_more_before !== latestSession.turnsPaging?.has_more_before
      );
      if (madeProgress) {
        earlierHistoryLoadKeysRef.current.delete(requestKey);
        upsertRuntimeSession(route, merged);
      }
      return madeProgress;
    } catch {
      earlierHistoryLoadKeysRef.current.delete(requestKey);
      return false;
    }
  }, [activeSession, activeSessionID, apiClient, route, upsertRuntimeSession]);

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
          const hydrated = await hydrateRuntimeSession(routeKey, sessionID, { force: true });
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
    if ((remoteSession.revision || 0) > (remoteSession.detailRevision || 0)) {
      return true;
    }
    if (hasRecoverableRuntimeState(remoteSession)) {
      return true;
    }
    return false;
  };

  const sendPromptImpl = async (prompt: string = composerDrafts[activeDraftKey] || "") => {
    const content = prompt.trim().slice(0, MAX_COMPOSER_CHARS);
    let attachments = activeDraftAttachments;
    if (!content && attachments.length === 0) {
      return;
    }
    const currentActiveSession = activeSessionID
      ? sessionsByRouteRef.current[route].find((item) => item.id === activeSessionID) || null
      : null;
    if (currentActiveSession && shouldBlockRuntimeInput(currentActiveSession)) {
      return;
    }
    const session = currentActiveSession?.serverBacked
      ? currentActiveSession
      : await createRuntimeBackedSession(route, content);
    if (!session) {
      return;
    }
    const optimisticUserMessage = createMessage("user", content || (attachments.length > 0 ? "Attached files" : ""), {
      attachments,
      route,
      source: "runtime",
      status: "queued",
    });
    patchSession(route, session.id, (currentSession) => ({
      ...currentSession,
      status: "busy",
      title: currentSession.titleAuto
        ? (optimisticUserMessage.text.slice(0, 32) || currentSession.title)
        : currentSession.title,
      titleAuto: false,
      messagesLoaded: currentSession.messagesLoaded === true,
      messages: [
        ...currentSession.messages,
        optimisticUserMessage,
      ],
    }));
    try {
      attachments = await uploadDraftAttachments(session.id, attachments);
      const hydrated = await sendRuntimeInput(session.id, {
        input: content,
        attachments: attachments.map(serializeMessageAttachment),
        ...runtimeSelectionInputPayload(selection),
        tool_ids: normalizeSelectionIDs(activeRuntimeConfig.toolIDs),
        skill_ids: activeSkillIDs,
        mcp_ids: normalizeSelectionIDs(activeRuntimeConfig.mcpIDs),
      });
      const latestSession = sessionsByRouteRef.current[route].find((item) => item.id === session.id) || session;
      const nextHydrated = hydrated
        ? mergeRuntimeSessions([hydrated], [latestSession])[0] || hydrated
        : null;
      if (nextHydrated) {
        upsertRuntimeSession(route, nextHydrated);
      } else {
        await recoverRuntimeSession(route, session.id, { requireMessages: true }, 1);
      }
      const routeDraftKey = newDraftKeyForRoute(route);
      const nextDrafts = { ...composerDrafts, [session.id]: "", [routeDraftKey]: "" };
      const nextAttachmentDrafts = { ...composerAttachmentDrafts, [session.id]: [], [routeDraftKey]: [] };
      setComposerDrafts(nextDrafts);
      setComposerAttachmentDrafts(nextAttachmentDrafts);
      persistComposerDrafts(route, nextDrafts);
      persistComposerAttachmentDrafts(route, nextAttachmentDrafts);
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

  sendPromptRef.current = sendPromptImpl;

  const sendPrompt = useCallback((prompt?: string) => sendPromptRef.current(prompt), []);

  const uploadDraftAttachments = async (
    sessionID: string,
    attachments: ComposerAttachment[],
  ): Promise<ComposerAttachment[]> => {
    const existing = attachments.filter((attachment) => attachment.assetURL);
    const pending = attachments.filter((attachment) => !attachment.assetURL && attachment.dataURL);
    if (pending.length === 0) {
      return existing;
    }
    const payload = await uploadRuntimeAttachments(sessionID, {
      attachments: pending.map((attachment) => ({
        name: attachment.name,
        content_type: attachment.contentType,
        data_url: attachment.dataURL,
        preview_data_url: attachment.previewDataURL || attachment.dataURL,
      })),
    });
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
    const remoteSessions = await refreshRuntimeSessions();
    const normalizedRemoteSessions = normalizeRouteSessions(routeKey, remoteSessions);
    setSessionsByRoute((current) => ({
      ...current,
      [routeKey]: normalizeRouteSessions(
        routeKey,
        mergeRuntimeSessions(normalizedRemoteSessions, current[routeKey]),
      ),
    }));
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

    const shouldHydrateActiveSession = Boolean(
      activeSession?.id
      && activeSession.serverBacked === true,
    );
    if (activeSession?.id && shouldHydrateActiveSession) {
      try {
        const hydrated = await hydrateRuntimeSession(route, activeSession.id, { force: true, allowRecoverableCache: true });
        if (hydrated) {
          upsertRuntimeSession(route, hydrated);
        }
      } catch {
      }
    }

  }, [activeSession, hydrateRuntimeSession, loadRuntimeSessions, route, upsertRuntimeSession]);

  useEffect(() => {
    writeConversationRuntimeCaches(activeSessionByRoute, sessionsByRoute, route);
  }, [activeSessionByRoute, route, sessionsByRoute]);

  useEffect(() => {
    sessionsByRouteRef.current = {
      chat: normalizeRouteSessions(
        "chat",
        mergeRuntimeSessions(sessionsByRoute.chat, sessionsByRouteRef.current.chat || []),
      ),
    };
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
    let cancelled = false;
    void (async () => {
      try {
        const remoteSessions = await loadRuntimeSessions(route);
        if (cancelled) {
          return;
        }
        const explicitRouteSessionReference = readWorkbenchRouteSessionID(route);
        const preferredActiveReference = explicitRouteSessionReference
          || remoteSessions[0]?.id
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
        const shouldActivatePreferredSession = Boolean(
          preferredActiveID
          && preferredActiveID !== activeSessionByRoute[route]
          && (
            explicitRouteSessionReference
            || remotePreferredSession
            || localPreferredSession
          ),
        );
        if (shouldActivatePreferredSession) {
          const nextActiveState = { ...activeSessionByRoute, [route]: preferredActiveID };
          setActiveSessionByRoute(nextActiveState);
          writeActiveSessionState(nextActiveState, route);
        }
        const shouldRecoverPreferredSession = preferredActiveID && (
          !remotePreferredSession
          || (remotePreferredSession.revision || 0) > (remotePreferredSession.detailRevision || 0)
          || hasRecoverableRuntimeState(remotePreferredSession)
        );
        const recoveredSession = shouldRecoverPreferredSession
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
        const shouldCalibratePreferredSession = Boolean(
          preferredActiveID
          && !recoveredSession
          && (
            localPreferredSession?.serverBacked === true
            || remotePreferredSession?.serverBacked === true
          ),
        );
        if (shouldCalibratePreferredSession) {
          try {
            const hydrated = await hydrateRuntimeSession(route, preferredActiveID, { force: true });
            if (!cancelled && hydrated) {
              upsertRuntimeSession(route, hydrated);
            }
          } catch {
          }
        }
        const nextActiveID = remoteSessions.some((session) => session.id === preferredActiveID) || recoveredSession
          ? preferredActiveID
          : sessionsByRouteRef.current[route][0]?.id || activeSessionByRoute[route];
        if (nextActiveID && nextActiveID !== activeSessionByRoute[route] && !shouldActivatePreferredSession) {
          const nextActiveState = { ...activeSessionByRoute, [route]: nextActiveID };
          setActiveSessionByRoute(nextActiveState);
          writeActiveSessionState(nextActiveState, route);
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
        writeActiveSessionState(nextActiveState, route);
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
      || isConversationBusyStatus(activeSession.status)
      || (activeSession.revision || 0) <= (activeSession.detailRevision || 0)
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
            requireStableAssistant: false,
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

  const pollRuntimeSessionUpdates = useCallback(async (
    routeKey: ConversationRoute,
    recoverableSessionIDs: string[],
  ): Promise<RuntimeSessionUpdatePollResult> => {
    const emptyResult: RuntimeSessionUpdatePollResult = {
      appliedAny: false,
      appliedRecoverableProgress: false,
      appliedRecoverable: false,
    };
    const recoverableIDSet = new Set(recoverableSessionIDs.map(normalizeText).filter(Boolean));
    const cursor = normalizeText(updateCursorByRouteRef.current[routeKey]) || "0";
    const body: RuntimeSessionUpdatesRequest = {
      since_event_id: cursor,
      limit: CHAT_SESSION_UPDATE_POLL_LIMIT,
      byte_limit: CHAT_SESSION_UPDATE_POLL_BYTE_LIMIT,
      sessions: buildRuntimeSessionUpdateAckManifest(sessionsByRouteRef.current[routeKey], recoverableSessionIDs),
    };
    const response = await apiClient.post<RuntimeSessionUpdatesResponse>(`/api/${routeKey}/sessions/updates`, body);
    const nextCursor = normalizeText(response.cursor);
    if (nextCursor) {
      updateCursorByRouteRef.current[routeKey] = nextCursor;
    }
    if (response.resync_required === true) {
      await Promise.all(recoverableSessionIDs.map(async (sessionID) => {
        try {
          const hydrated = await hydrateRuntimeSession(routeKey, sessionID, { force: true });
          if (hydrated) {
            upsertRuntimeSession(routeKey, hydrated);
          }
        } catch {
        }
      }));
      return {
        appliedAny: recoverableIDSet.size > 0,
        appliedRecoverableProgress: recoverableIDSet.size > 0,
        appliedRecoverable: recoverableIDSet.size > 0,
      };
    }
    let result = emptyResult;
    for (const event of Array.isArray(response.events) ? response.events : []) {
      const payload = runtimeSessionPayloadFromEventData(event);
      if (!payload) {
        continue;
      }
      const previous = sessionsByRouteRef.current[routeKey].find((session) => session.id === normalizeText(payload.id)) || null;
      const normalized = normalizeRuntimeSession(payload, previous, routeKey, { source: "event" });
      if (normalized) {
        upsertRuntimeSession(routeKey, normalized);
        const appliedRecoverableProgress = recoverableIDSet.has(normalized.id)
          && runtimeSessionUpdateMadeProgress(previous, normalized);
        result = {
          appliedAny: true,
          appliedRecoverableProgress: result.appliedRecoverableProgress || appliedRecoverableProgress,
          appliedRecoverable:
            result.appliedRecoverable
            || (recoverableIDSet.has(normalized.id) && !shouldPollRuntimeBackedSession(normalized)),
        };
      }
    }
    return result;
  }, [apiClient, hydrateRuntimeSession, upsertRuntimeSession]);

  useEffect(() => {
    window.clearTimeout(pollTimerRef.current);
    const recoverableSessions = sessionsByRoute[route].filter(shouldPollRuntimeBackedSession);
    if (!recoverableSessions.length) {
      return;
    }
    const pollPlan = resolveChatSessionPollPlan({
      sessionCount: recoverableSessions.length,
      pageHidden,
      fallbackAttempt: fallbackPollAttempt,
    });
    if (!pollPlan.enabled) {
      return;
    }
    pollTimerRef.current = window.setTimeout(async () => {
      let appliedRecoverableProgress = false;
      let appliedRecoverable = false;
      const latestSessions = sessionsByRouteRef.current[route];
      const pollSessionIDSet = new Set(
        latestSessions
          .filter(shouldPollRuntimeBackedSession)
          .map((session) => session.id),
      );
      const latestActiveSessionID = normalizeText(activeSessionIDRef.current)
        || normalizeText(activeSessionByRoute[route])
        || normalizeText(activeSessionID);
      const latestActiveSession = latestActiveSessionID
        ? latestSessions.find((session) => session.id === latestActiveSessionID) || null
        : null;
      if (latestActiveSession && shouldBlockRuntimeInput(latestActiveSession)) {
        pollSessionIDSet.add(latestActiveSession.id);
      }
      const pollSessionIDs = Array.from(pollSessionIDSet);
      if (!pollSessionIDs.length) {
        setFallbackPollAttempt(0);
        return;
      }
      try {
        const result = await pollRuntimeSessionUpdates(route, pollSessionIDs);
        appliedRecoverableProgress = result.appliedRecoverableProgress;
        appliedRecoverable = result.appliedRecoverable;
      } catch {
      }
      const nextFallbackPollAttempt = fallbackPollAttempt + 1;
      if (!appliedRecoverableProgress && shouldRefreshChatSessionDetailAfterEmptyUpdates(nextFallbackPollAttempt)) {
        try {
          const refreshed = await Promise.all(pollSessionIDs.map(async (sessionID) => {
            try {
              const hydrated = await hydrateRuntimeSession(route, sessionID, { force: true });
              if (!hydrated) {
                return false;
              }
              upsertRuntimeSession(route, hydrated);
              return !shouldPollRuntimeBackedSession(hydrated);
            } catch {
              return false;
            }
          }));
          appliedRecoverable = refreshed.some(Boolean);
        } catch {
        }
      }
      setFallbackPollAttempt((current) => appliedRecoverableProgress || appliedRecoverable ? 0 : current + 1);
    }, pollPlan.interval);
    return () => window.clearTimeout(pollTimerRef.current);
  }, [activeSessionByRoute, activeSessionID, fallbackPollAttempt, hydrateRuntimeSession, pageHidden, pollRuntimeSessionUpdates, route, sessionsByRoute, upsertRuntimeSession]);

  const selection = resolveModelSelection(activeRuntimeConfig, availableProviders);
  const selectedProvider = enabledProviders(availableProviders).find((provider) => normalizeText(provider.id) === selection.providerID) || null;
  const selectedModel = enabledModels(selectedProvider).find((model) => normalizeText(model.id) === selection.modelID) || null;
  const currentTarget = activeSession?.target || defaultChatTarget();
  const activeSessionBusy = activeSession ? shouldBlockRuntimeInput(activeSession) : false;
  const selectedModelSupportsVision = selectedModel ? selectedModel.supports_vision !== false : true;
  const activeMcpIDKey = activeRuntimeConfig.mcpIDs.join("\u0000");
  const activeSkillIDKey = activeSkillIDs.join("\u0000");
  const availableSkillIDKey = availableSkillIDs === null ? "__loading__" : availableSkillIDs.join("\u0000");
  const runtimeProviderItems = useMemo(() => enabledProviders(availableProviders).map((provider) => ({
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
  })), [availableProviders, selection.modelID, selection.providerID]);
  const runtimeCapabilityItems = useMemo(() => {
    const activeMcpIDs = new Set(activeRuntimeConfig.mcpIDs.map(normalizeText));
    return [
      ...mcps
        .filter((item) => item.enabled !== false)
        .map((item) => ({
          id: normalizeText(item.id),
          name: normalizeText(item.name) || normalizeText(item.id),
          description: normalizeText(item.description) || normalizeText(item.scope) || "MCP",
          kind: "mcp" as const,
          active: activeMcpIDs.has(normalizeText(item.id)),
        }))
        .filter((item) => item.id),
    ];
  // activeMcpIDKey intentionally decouples Composer from activeSession.messages changes.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeMcpIDKey, activeRuntimeConfig.mcpIDs, mcps]);
  const runtimeSkillItems = useMemo(() => {
    const selectedSkillIDs = new Set(activeSkillIDs);
    return [
      ...skills
        .filter((item) => item.enabled !== false && isPublicSkillCapability(item))
        .map((item) => ({
          id: normalizeText(item.id),
          name: normalizeText(item.name) || normalizeText(item.id),
          description: normalizeText(item.description) || normalizeText(item.scope) || "Skill",
          kind: "skill" as const,
          active: selectedSkillIDs.has(normalizeText(item.id)),
          visibility: "public" as const,
          locked: false,
        }))
        .filter((item) => item.id),
    ].filter((item): item is RuntimeSelection => Boolean(item?.id));
  // activeSkillIDKey intentionally decouples Composer from activeSession.messages changes.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeSkillIDKey, skills]);
  const readCurrentActiveSession = useCallback(() => {
    if (!activeSessionID) {
      return null;
    }
    return sessionsByRouteRef.current[route].find((session) => session.id === activeSessionID) || null;
  }, [activeSessionID, route]);
  const toggleInspector = useCallback((tab?: "model" | "capabilities" | "skills") => {
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
  }, [inspectorTab]);
  const closeInspector = useCallback(() => setInspectorOpen(false), []);
  const selectModel = useCallback((providerID: string, modelID: string) => {
    const session = readCurrentActiveSession();
    const nextConfig = normalizeRuntimeComposerConfig({
      ...activeRuntimeConfig,
      modelProviderID: normalizeText(providerID),
      modelID: normalizeText(modelID),
    }, true);
    commitRuntimeConfig(nextConfig);
    if (!session) {
      return;
    }
    const nextSession = {
      ...session,
      modelProviderID: nextConfig.modelProviderID,
      modelID: nextConfig.modelID,
    };
    patchSession(route, session.id, (currentSession) => ({
      ...currentSession,
      modelProviderID: nextSession.modelProviderID,
      modelID: nextSession.modelID,
    }));
    void persistRuntimeSessionConfig(route, nextSession);
  }, [activeRuntimeConfig, commitRuntimeConfig, patchSession, persistRuntimeSessionConfig, readCurrentActiveSession, route]);
  const toggleCapability = useCallback((id: string, kind: "tool" | "mcp", checked: boolean) => {
    const session = readCurrentActiveSession();
    const value = normalizeText(id);
    if (!value) {
      return;
    }
    const mutate = (items: string[]) =>
      checked
        ? normalizeSelectionIDs([...items, value])
        : items.filter((item) => item !== value);
    const nextConfig = kind === "tool"
      ? normalizeRuntimeComposerConfig({ ...activeRuntimeConfig, toolIDs: mutate(activeRuntimeConfig.toolIDs) }, true)
      : normalizeRuntimeComposerConfig({ ...activeRuntimeConfig, mcpIDs: mutate(activeRuntimeConfig.mcpIDs) }, true);
    commitRuntimeConfig(nextConfig);
    if (!session) {
      return;
    }
    const nextSession = kind === "tool"
      ? { ...session, toolIDs: mutate(session.toolIDs) }
      : { ...session, mcpIDs: mutate(session.mcpIDs) };
    patchSession(route, session.id, (currentSession) =>
      kind === "tool"
        ? { ...currentSession, toolIDs: mutate(currentSession.toolIDs) }
        : { ...currentSession, mcpIDs: mutate(currentSession.mcpIDs) },
    );
    void persistRuntimeSessionConfig(route, nextSession);
  }, [activeRuntimeConfig, commitRuntimeConfig, patchSession, persistRuntimeSessionConfig, readCurrentActiveSession, route]);
  const toggleSkill = useCallback((id: string, checked: boolean) => {
    const session = readCurrentActiveSession();
    const value = normalizeText(id);
    if (!value) {
      return;
    }
    if (availableSkillIDs !== null && !availableSkillIDs.includes(value)) {
      return;
    }
    const currentSkillIDs = effectiveRuntimeSkillIDs(activeRuntimeConfig, availableSkillIDs);
    const mutate = () =>
      checked
        ? normalizeSelectionIDs([...currentSkillIDs, value])
        : currentSkillIDs.filter((item) => item !== value);
    const nextConfig = normalizeRuntimeComposerConfig({
      ...activeRuntimeConfig,
      skillIDs: mutate(),
      skillIDsExplicit: true,
    }, true);
    commitRuntimeConfig(nextConfig);
    if (!session) {
      return;
    }
    const nextSession = {
      ...session,
      skillIDs: nextConfig.skillIDs,
      skillIDsExplicit: true,
    };
    patchSession(route, session.id, () => nextSession);
    void persistRuntimeSessionConfig(route, nextSession);
  // availableSkillIDKey intentionally decouples Composer from availableSkillIDs array identity.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeRuntimeConfig, availableSkillIDKey, commitRuntimeConfig, patchSession, persistRuntimeSessionConfig, readCurrentActiveSession, route]);
  const toggleRuntimeEventFilter = useCallback((id: RuntimeEventFilterID, checked: boolean) => {
    const value = normalizeText(id) as RuntimeEventFilterID;
    const allowed = new Set(RUNTIME_EVENT_FILTER_OPTIONS.map((option) => option.id));
    if (!allowed.has(value)) {
      return;
    }
    setRuntimeEventFilter((current) => {
      const next = checked
        ? normalizeRuntimeEventFilter([...current, value])
        : normalizeRuntimeEventFilter(current.filter((item) => item !== value));
      persistRuntimeEventFilter(route, next);
      return next;
    });
  }, []);

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
    selectedModelSupportsVision,
    providers: runtimeProviderItems,
    capabilities: runtimeCapabilityItems,
    skills: runtimeSkillItems,
    runtimeEventFilter,
    toolCount: activeRuntimeConfig.toolIDs.length + activeRuntimeConfig.mcpIDs.length,
    skillCount: activeSkillIDs.length,
    createSession: () => {
      void createRuntimeBackedSession(route);
    },
    focusSession,
    removeSession,
    setSessionPinned,
    refreshActiveSession,
    loadEarlierHistory,
    toggleInspector,
    closeInspector,
    selectModel,
    toggleCapability,
    toggleSkill,
    toggleRuntimeEventFilter,
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
    selectedModelSupportsVision,
    runtimeProviderItems,
    runtimeCapabilityItems,
    runtimeSkillItems,
    runtimeEventFilter,
    activeSkillIDs,
    activeRuntimeConfig.toolIDs.length,
    activeRuntimeConfig.mcpIDs.length,
    createRuntimeBackedSession,
    focusSession,
    patchSession,
    refreshActiveSession,
    loadEarlierHistory,
    loadProcessEventDetail,
    removeSession,
    setSessionPinned,
    toggleInspector,
    closeInspector,
    selectModel,
    toggleCapability,
    toggleSkill,
    toggleRuntimeEventFilter,
  ]);

  const composerValue = useMemo<ConversationRuntimeComposerContextValue>(() => ({
    route,
    inspectorOpen,
    inspectorTab,
    inspectorTabOpen,
    draft: composerDrafts[activeDraftKey] || "",
    draftAttachments: activeDraftAttachments,
    busy: activeSessionBusy,
    selectedProviderId: selection.providerID,
    selectedModelId: selection.modelID,
    selectedModelSupportsVision,
    providers: runtimeProviderItems,
    capabilities: runtimeCapabilityItems,
    skills: runtimeSkillItems,
    runtimeEventFilter,
    setDraft: (value: string) => {
      const nextDrafts = { ...composerDrafts, [activeDraftKey]: value.slice(0, MAX_COMPOSER_CHARS) };
      latestComposerDraftsRef.current = nextDrafts;
      setComposerDrafts(nextDrafts);
    },
    addDraftAttachments: async (attachments: ComposerAttachment[]) => {
      const normalized = normalizeStoredAttachments(attachments);
      if (normalized.length === 0) {
        return;
      }
      const currentActiveSession = readCurrentActiveSession();
      const session = currentActiveSession?.serverBacked
        ? currentActiveSession
        : await createRuntimeBackedSession(route);
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
      const nextDrafts = { ...composerAttachmentDrafts, [session.id]: nextAttachments, [newDraftKeyForRoute(route)]: [] };
      setComposerAttachmentDrafts(nextDrafts);
      persistComposerAttachmentDrafts(route, nextDrafts);
    },
    removeDraftAttachment: (attachmentID: string) => {
      const sessionID = activeSessionID;
      if (!sessionID) {
        return;
      }
      const nextItems = (composerAttachmentDrafts[sessionID] || []).filter((item) => item.id !== attachmentID);
      const nextDrafts = { ...composerAttachmentDrafts, [sessionID]: nextItems };
      setComposerAttachmentDrafts(nextDrafts);
      persistComposerAttachmentDrafts(route, nextDrafts);
    },
    clearDraftAttachments: () => {
      const sessionID = activeSessionID;
      if (!sessionID) {
        return;
      }
      const nextDrafts = { ...composerAttachmentDrafts, [sessionID]: [] };
      setComposerAttachmentDrafts(nextDrafts);
      persistComposerAttachmentDrafts(route, nextDrafts);
    },
    sendPrompt,
    toggleInspector,
    closeInspector,
    selectModel,
    toggleCapability,
    toggleSkill,
    toggleRuntimeEventFilter,
  }), [
    route,
    inspectorOpen,
    inspectorTab,
    inspectorTabOpen,
    activeDraftKey,
    composerDrafts,
    activeDraftAttachments,
    activeSessionBusy,
    selection.providerID,
    selection.modelID,
    selectedModelSupportsVision,
    runtimeProviderItems,
    runtimeCapabilityItems,
    runtimeSkillItems,
    runtimeEventFilter,
    composerAttachmentDrafts,
    activeSessionID,
    readCurrentActiveSession,
    createRuntimeBackedSession,
    sendPrompt,
    toggleInspector,
    closeInspector,
    selectModel,
    toggleCapability,
    toggleSkill,
    toggleRuntimeEventFilter,
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
