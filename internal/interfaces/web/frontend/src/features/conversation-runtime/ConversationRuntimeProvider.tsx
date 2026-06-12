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
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  isComposerImageAttachment,
  type ComposerAttachment,
} from "./composerImageAttachments";

const ACTIVE_SESSION_STORAGE_KEY = "alter0.web.session.active.v1";
const ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.snapshot.v1";
const RECENT_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.recent.v1";
const COMPOSER_DRAFT_STORAGE_KEY = "alter0.web.composer.drafts.v1";
const COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY = "alter0.web.composer.attachments.v1";
const COMPOSER_DRAFT_PERSIST_DELAY_MS = 160;
const STREAM_ENDPOINT = "/api/messages/stream";
const FALLBACK_ENDPOINT = "/api/messages";
const RUNTIME_SESSION_COLLECTION_ENDPOINT = "/api/conversation-runtime/sessions";
const MAX_COMPOSER_CHARS = 10000;
const CHAT_TASK_POLL_INTERVAL_MS = 3000;
const STREAM_DELTA_FLUSH_INTERVAL_MS = 50;
const LOCAL_STREAM_PROCESS_STEP_ID = "local-stream-start";
const EXECUTION_ENGINE_METADATA_KEY = "alter0.execution.engine";
const EXECUTION_ENGINE_CODEX = "codex";
const LLM_PROVIDER_METADATA_KEY = "alter0.llm.provider_id";
const LLM_MODEL_METADATA_KEY = "alter0.llm.model";
const CODEX_RUNTIME_PROVIDER_ID = "alter0-codex";
const CODEX_RUNTIME_MODEL_ID = "codex";
const CANONICAL_CHAT_SESSION_ID = "alter0-chat";
const MAX_RECENT_SESSION_SNAPSHOTS = 12;
const PAGE_ACTIVE_REFRESH_DEBOUNCE_MS = 400;

type ChatTaskPollPlan = {
  enabled: boolean;
  interval: number;
};

export function resolveChatTaskPollPlan(options: { pendingCount: number; pageHidden: boolean }): ChatTaskPollPlan {
  if (options.pendingCount <= 0 || options.pageHidden) {
    return {
      enabled: false,
      interval: 0,
    };
  }
  return {
    enabled: true,
    interval: CHAT_TASK_POLL_INTERVAL_MS,
  };
}

export type ConversationRoute = "chat";

type ChatTarget = {
  type: "model";
  id: string;
  name: string;
};

type ChatProcessStep = {
  id: string;
  kind: string;
  title: string;
  detail: string;
  status: string;
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
  processSteps: ChatProcessStep[];
  processCollapsed?: boolean;
  taskID: string;
  taskStatus: string;
  taskPending: boolean;
  taskResultDelivered: boolean;
  taskResultFor: string;
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
  mcpIDs: string[];
  messages: ChatMessage[];
  messagesLoaded?: boolean;
  serverBacked?: boolean;
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

type ChatTaskResponse = {
  id?: string;
  status?: string;
  summary?: string;
  result?: {
    route?: string;
    process_steps?: Array<{
      id?: string;
      kind?: string;
      title?: string;
      detail?: string;
      status?: string;
    }>;
  };
};

type ActiveSessionState = Record<ConversationRoute, string>;
type SessionsState = Record<ConversationRoute, ChatSession[]>;
type ComposerDraftMap = Record<string, string>;
type ComposerAttachmentDraftMap = Record<string, ComposerAttachment[]>;
type StoredActiveSessionSnapshotState = Record<string, unknown>;
type StoredRecentSessionSnapshotState = Record<string, unknown>;

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

type RuntimeSessionPayload = {
  id?: string;
  status?: string;
  title?: string;
  title_auto?: boolean;
  title_score?: number;
  created_at?: string | number;
  pinned?: boolean;
  target_type?: string;
  target_id?: string;
  target_name?: string;
  model_provider_id?: string;
  model_id?: string;
  tool_ids?: string[];
  skill_ids?: string[];
  mcp_ids?: string[];
  messages?: RuntimeMessagePayload[];
};

type RuntimeSessionConfigPatch = {
  title: string;
  model_provider_id: string;
  model_id: string;
  tool_ids: string[];
  skill_ids: string[];
  mcp_ids: string[];
};

type RuntimeMessagePayload = {
  id?: string;
  role?: string;
  text?: string;
  attachments?: Array<{
    id?: string;
    name?: string;
    content_type?: string;
    asset_url?: string;
    preview_url?: string;
  }>;
  route?: string;
  source?: string;
  error?: boolean;
  status?: string;
  at?: string | number;
  process_steps?: Array<{
    id?: string;
    kind?: string;
    title?: string;
    detail?: string;
    status?: string;
  }>;
  task_id?: string;
  task_status?: string;
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
  toolCount: number;
  skillCount: number;
  createSession: () => void;
  focusSession: (sessionID: string) => void;
  removeSession: (sessionID: string) => Promise<void>;
  setSessionPinned: (sessionID: string, pinned: boolean) => Promise<void>;
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
  toggleProcess: (messageID: string) => void;
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

type StreamResult = {
  ok: boolean;
  canFallback: boolean;
  canRecover: boolean;
  error: string;
};

type RuntimeRecoveryRequirement = {
  requireMessages?: boolean;
  requireStableAssistant?: boolean;
};

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
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

function isCodexRuntimeSelection(providerID: string, modelID: string): boolean {
  return normalizeText(providerID) === CODEX_RUNTIME_PROVIDER_ID && normalizeText(modelID) === CODEX_RUNTIME_MODEL_ID;
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

function normalizeProcessSteps(values: unknown): ChatProcessStep[] {
  if (!Array.isArray(values)) {
    return [];
  }
  return values
    .map((item) => {
      if (!item || typeof item !== "object") {
        return null;
      }
      const detail = item as Record<string, unknown>;
      const title = normalizeText(detail.title);
      const body = normalizeText(detail.detail);
      if (!title && !body) {
        return null;
      }
      return {
        id: normalizeText(detail.id),
        kind: normalizeText(detail.kind),
        title,
        detail: body,
        status: normalizeText(detail.status),
      };
    })
    .filter((item): item is ChatProcessStep => item !== null);
}

function isStreamingPlaceholderText(text: string): boolean {
  const normalized = normalizeText(text).toLowerCase();
  return normalized === "" || normalized === "thinking...";
}

function createLocalStreamProcessStep(): ChatProcessStep {
  return {
    id: LOCAL_STREAM_PROCESS_STEP_ID,
    kind: "analysis",
    title: "Thinking",
    detail: "",
    status: "running",
  };
}

function removeLocalStreamProcessStep(steps: ChatProcessStep[]): ChatProcessStep[] {
  return steps.filter((step) => step.id !== LOCAL_STREAM_PROCESS_STEP_ID);
}

function isRecoverableAssistantMessage(message: ChatMessage): boolean {
  if (message.role !== "assistant") {
    return false;
  }
  return message.taskPending || message.error || normalizeTaskStatus(message.status) === "streaming" || isStreamingPlaceholderText(message.text);
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

function hasPersistedAssistantState(messages: ChatMessage[]): boolean {
  return messages.some((message) => {
    if (message.role !== "assistant") {
      return false;
    }
    if (message.taskID) {
      return true;
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
    processSteps: normalizeProcessSteps(record.process_steps),
    processCollapsed:
      typeof record.process_collapsed === "boolean"
        ? record.process_collapsed
        : undefined,
    taskID: normalizeText(record.task_id),
    taskStatus: normalizeText(record.task_status),
    taskPending: Boolean(record.task_pending),
    taskResultDelivered: Boolean(record.task_result_delivered),
    taskResultFor: normalizeText(record.task_result_for),
  };
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
      id: normalizeText(record.targetID),
      name: normalizeText(record.targetName),
    }),
    modelProviderID: normalizeText(record.modelProviderID),
    modelID: normalizeText(record.modelID),
    toolIDs: normalizeSelectionIDs(record.toolIDs),
    skillIDs: normalizeSelectionIDs(record.skillIDs),
    mcpIDs: normalizeSelectionIDs(record.mcpIDs),
    messages: Array.isArray(record.messages)
      ? record.messages.map(normalizeStoredMessage).filter((message): message is ChatMessage => message !== null)
      : [],
    messagesLoaded: typeof record.messagesLoaded === "boolean" ? record.messagesLoaded : undefined,
    serverBacked: typeof record.serverBacked === "boolean" ? record.serverBacked : undefined,
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
    process_steps: message.processSteps,
    process_collapsed: message.processCollapsed,
    task_id: message.taskID,
    task_status: message.taskStatus,
    task_pending: message.taskPending,
    task_result_delivered: message.taskResultDelivered,
    task_result_for: message.taskResultFor,
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
    mcpIDs: session.mcpIDs,
    messages: session.messages.map(serializeStoredMessage),
    messagesLoaded: session.messagesLoaded,
    serverBacked: session.serverBacked,
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

function loadActiveSessionState(): ActiveSessionState {
  const parsed = readJSONStorage<Record<string, string>>(ACTIVE_SESSION_STORAGE_KEY, {});
  return {
    chat:
      readWorkbenchRouteSessionID("chat")
      || normalizeText(parsed.chat)
      || normalizeText(parsed["chat"])
      || CANONICAL_CHAT_SESSION_ID,
  };
}

function loadActiveSessionSnapshots(): SessionsState {
  const parsedActive = readJSONStorage<StoredActiveSessionSnapshotState>(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY, {});
  const parsedRecent = readJSONStorage<StoredRecentSessionSnapshotState>(RECENT_SESSION_SNAPSHOT_STORAGE_KEY, {});
  const mergeStoredRouteSessions = (routeKey: string) => {
    const sessions = new Map<string, ChatSession>();
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

function normalizeRuntimeMessage(item: RuntimeMessagePayload): ChatMessage | null {
  const id = normalizeText(item.id);
  if (!id) {
    return null;
  }
  const role = normalizeText(item.role) === "assistant" ? "assistant" : "user";
  return {
    id,
    role,
    text: typeof item.text === "string" ? item.text : "",
    attachments: Array.isArray(item.attachments)
      ? item.attachments
        .map((attachment) => {
          const attachmentID = normalizeText(attachment.id);
          const contentType = normalizeText(attachment.content_type);
          const assetURL = normalizeText(attachment.asset_url);
          const previewURL = normalizeText(attachment.preview_url);
          if (!attachmentID || !contentType || !assetURL) {
            return null;
          }
          return {
            id: attachmentID,
            kind: contentType.startsWith("image/") ? "image" as const : "file" as const,
            name: normalizeText(attachment.name) || (contentType.startsWith("image/") ? "image" : "file"),
            contentType,
            size: 0,
            assetURL,
            previewURL: contentType.startsWith("image/") ? previewURL || assetURL : undefined,
          };
        })
        .filter((attachment): attachment is ComposerAttachment => attachment !== null)
      : [],
    route: normalizeText(item.route),
    source: normalizeText(item.source),
    error: item.error === true,
    status: normalizeText(item.status) || (role === "assistant" ? "done" : ""),
    at: normalizeDateValue(item.at),
    processSteps: normalizeProcessSteps(item.process_steps),
    taskID: normalizeText(item.task_id),
    taskStatus: normalizeText(item.task_status),
    taskPending: false,
    taskResultDelivered: false,
    taskResultFor: "",
  };
}

function normalizeRuntimeSession(
  item: RuntimeSessionPayload,
  previous?: ChatSession | null,
  sourceRoute: ConversationRoute = "chat",
): ChatSession | null {
  const id = normalizeText(item.id);
  if (!id) {
    return null;
  }
  const parsedMessages = Array.isArray(item.messages)
    ? item.messages.map(normalizeRuntimeMessage).filter((message): message is ChatMessage => message !== null)
    : null;
  const messages = parsedMessages
    ? (previous?.messages.length && !shouldUseParsedMessages(previous.messages, parsedMessages)
      ? previous.messages
      : parsedMessages)
    : previous?.messages || [];
  return {
    id,
    sourceRoute: previous?.sourceRoute || sourceRoute,
    status: normalizeText(item.status) || previous?.status || "",
    title: normalizeText(item.title) || previous?.title || "New",
    titleAuto: item.title_auto !== false,
    titleScore: Number.isFinite(Number(item.title_score)) ? Number(item.title_score) : previous?.titleScore || 0,
    createdAt: normalizeDateValue(item.created_at),
    pinned: typeof item.pinned === "boolean" ? item.pinned : previous?.pinned || false,
    target: normalizeChatTarget({
      type: "model",
      id: normalizeText(item.target_id),
      name: normalizeText(item.target_name),
    }),
    modelProviderID: normalizeText(item.model_provider_id) || previous?.modelProviderID || "",
    modelID: normalizeText(item.model_id) || previous?.modelID || "",
    toolIDs: normalizeSelectionIDs(item.tool_ids || previous?.toolIDs || []),
    skillIDs: normalizeSelectionIDs(item.skill_ids || previous?.skillIDs || []),
    mcpIDs: normalizeSelectionIDs(item.mcp_ids || previous?.mcpIDs || []),
    messages,
    messagesLoaded: Array.isArray(item.messages),
    serverBacked: true,
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

function buildMessageMetadata(
  session: ChatSession | null,
  selection: { providerID: string; modelID: string },
  skillIDs: string[] = session?.skillIDs || [],
): Record<string, string> {
  const metadata: Record<string, string> = {
    "alter0.skills.include": JSON.stringify(skillIDs),
    "alter0.mcp.request.enable": JSON.stringify(session?.mcpIDs || []),
  };
  if (isCodexRuntimeSelection(selection.providerID, selection.modelID)) {
    metadata[EXECUTION_ENGINE_METADATA_KEY] = EXECUTION_ENGINE_CODEX;
    return metadata;
  }
  metadata[LLM_PROVIDER_METADATA_KEY] = selection.providerID;
  metadata[LLM_MODEL_METADATA_KEY] = selection.modelID;
  return metadata;
}

function normalizeTaskStatus(status: string): string {
  return normalizeText(status).toLowerCase() || "queued";
}

function isTerminalTaskStatus(status: string): boolean {
  return ["success", "failed", "canceled"].includes(normalizeTaskStatus(status));
}

function readResponsePayload(response: Response): Promise<unknown> {
  return response.text().then((text) => {
    if (!text) {
      return {};
    }
    try {
      return JSON.parse(text);
    } catch {
      return {};
    }
  });
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

function parseSSEBlock(block: string) {
  const lines = block.split("\n");
  let event = "message";
  const dataLines: string[] = [];
  lines.forEach((line) => {
    if (!line || line.startsWith(":")) {
      return;
    }
    if (line.startsWith("event:")) {
      event = line.slice(6).trim();
      return;
    }
    if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  });
  if (!dataLines.length) {
    return null;
  }
  try {
    return { event, data: JSON.parse(dataLines.join("\n")) as Record<string, unknown> };
  } catch {
    return null;
  }
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
  const [sessionsByRoute, setSessionsByRoute] = useState<SessionsState>(() => loadActiveSessionSnapshots());
  const [sessionsLoadedByRoute, setSessionsLoadedByRoute] = useState<Record<ConversationRoute, boolean>>({
    chat: false,
  });
  const [activeSessionByRoute, setActiveSessionByRoute] = useState<ActiveSessionState>(() =>
    loadActiveSessionState(),
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
  const [pendingTasksVersion, setPendingTasksVersion] = useState(0);
  const [pinningSessionIDs, setPinningSessionIDs] = useState<Record<string, boolean>>({});
  const [pageHidden, setPageHidden] = useState(() => typeof document !== "undefined" && document.hidden);
  const pollTimerRef = useRef<number>(0);
  const sessionsByRouteRef = useRef(sessionsByRoute);
  const recoveryPromisesRef = useRef(new Map<string, Promise<ChatSession | null>>());
  const composerDraftPersistTimerRef = useRef<number>(0);
  const latestComposerDraftsRef = useRef<ComposerDraftMap>(composerDrafts);
  const latestComposerAttachmentDraftsRef = useRef<ComposerAttachmentDraftMap>(composerAttachmentDrafts);

  const activeSessions = sessionsByRoute[route];
  const activeSessionReference = activeSessionByRoute[route];
  const activeSessionID = resolveSessionIDReference(activeSessions, activeSessionReference) || activeSessionReference;
  const activeSession = activeSessions.find((session) => session.id === activeSessionID) || null;
  const activeDraftAttachments = activeSessionID ? composerAttachmentDrafts[activeSessionID] || [] : [];
  const availableProviders = useMemo(() => runtimeProviders(providers), [providers]);
  const availableSkillIDs = useMemo(
    () => skillCatalogLoaded ? defaultChatSkillIDs(skills) : null,
    [skillCatalogLoaded, skills],
  );
  const activeSkillIDs = useMemo(
    () => effectiveChatSkillIDs(activeSession?.skillIDs, availableSkillIDs),
    [activeSession?.skillIDs, availableSkillIDs],
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

  const ensureSession = useCallback((
    target?: Partial<ChatTarget> | null,
    preferredActiveState: ActiveSessionState = activeSessionByRoute,
    currentSessions: SessionsState = sessionsByRoute,
  ) => {
    const targetValue = normalizeChatTarget(target || defaultChatTarget());
    const routeSessions = normalizeRouteSessions(route, currentSessions[route]);
    const existing = routeSessions.find((session) => session.id === preferredActiveState[route]) || null;
    if (existing) {
      return existing;
    }
    const created: ChatSession = {
      id: makeID("session"),
      sourceRoute: route,
      status: "ready",
      title: "New",
      titleAuto: true,
      titleScore: 0,
      createdAt: Date.now(),
      pinned: false,
      target: targetValue,
      modelProviderID: "",
      modelID: "",
      toolIDs: [],
      skillIDs: defaultChatSkillIDs(skills),
      mcpIDs: [],
      messages: [],
      messagesLoaded: true,
      serverBacked: false,
    };
    const nextSessionsByRoute: SessionsState = {
      ...currentSessions,
      [route]: [created, ...currentSessions[route]],
    };
    const nextActiveState = { ...preferredActiveState, [route]: created.id };
    setSessionsByRoute(nextSessionsByRoute);
    setActiveSessionByRoute(nextActiveState);
    writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
    return created;
  }, [activeSessionByRoute, route, sessionsByRoute, skills]);

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
    processSteps: patch.processSteps || [],
    processCollapsed: patch.processCollapsed,
    taskID: patch.taskID || "",
    taskStatus: patch.taskStatus || "",
    taskPending: Boolean(patch.taskPending),
    taskResultDelivered: Boolean(patch.taskResultDelivered),
    taskResultFor: patch.taskResultFor || "",
  });

  const appendMessage = useCallback((routeKey: ConversationRoute, sessionID: string, message: ChatMessage) => {
    patchSession(routeKey, sessionID, (session) => ({
      ...session,
      status: message.role === "assistant" && message.error
        ? "failed"
        : message.role === "assistant" && (message.taskPending || normalizeTaskStatus(message.status) === "running")
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
        : patch.taskPending || normalizeTaskStatus(patch.taskStatus || patch.status || "") === "running"
          ? "busy"
          : normalizeText(patch.status) === "done"
            ? "ready"
            : session.status,
      messages: session.messages.map((message) =>
        message.id === messageID ? { ...message, ...patch } : message,
      ),
    }));
  }, [patchSession]);

  const focusSession = useCallback((sessionID: string) => {
    const resolvedSessionID = sessionID;
    const nextActiveState = { ...activeSessionByRoute, [route]: resolvedSessionID };
    setActiveSessionByRoute(nextActiveState);
    writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
    writeWorkbenchRouteSessionID(route, resolvedSessionID);
  }, [activeSessionByRoute, route]);

  const removeSession = useCallback(async (sessionID: string) => {
    try {
      await apiClient.delete(`/api/sessions/${encodeURIComponent(sessionID)}`);
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
      await apiClient.post(`/api/sessions/${encodeURIComponent(normalizedSessionID)}/pin`, { pinned });
      applyPinnedState();
    } catch {
      applyPinnedState();
    } finally {
      setPinningSessionIDs((current) => ({ ...current, [normalizedSessionID]: false }));
    }
  }, [apiClient, patchSession, route]);

  const sendMessageFallback = async (
    routeKey: ConversationRoute,
    sessionID: string,
    assistantMessageID: string,
    content: string,
    attachments: ComposerAttachment[],
  ) => {
    const session = sessionsByRoute[routeKey].find((item) => item.id === sessionID) || null;
    const selection = resolveModelSelection(session, runtimeProviders(providers));
    const body = await apiClient.post<{
      result?: {
        output?: string;
        route?: string;
        metadata?: Record<string, string>;
        process_steps?: Array<Record<string, unknown>>;
      };
      task_id?: string;
      task_status?: string;
    }>(FALLBACK_ENDPOINT, {
      session_id: sessionID,
      channel_id: "web-default",
      content,
      attachments: attachments.map(serializeMessageAttachment),
      metadata: buildMessageMetadata(session, selection, effectiveChatSkillIDs(session?.skillIDs, availableSkillIDs)),
    });
    setAssistantMessage(routeKey, sessionID, assistantMessageID, {
      text: normalizeText(body?.result?.output) || "No response",
      route: normalizeText(body?.result?.route),
      source: normalizeText(body?.result?.metadata?.["alter0.execution.source"]),
      processSteps: normalizeProcessSteps(body?.result?.process_steps),
      taskID: normalizeText(body?.task_id),
      taskStatus: normalizeText(body?.task_status),
      taskPending: Boolean(body?.task_id && !isTerminalTaskStatus(normalizeText(body?.task_status))),
      status: normalizeText(body?.task_status) || "done",
      error: false,
    });
    if (body?.task_id) {
      setPendingTasksVersion((value) => value + 1);
    }
  };

  const hydrateRuntimeSessionResponse = (
    routeKey: ConversationRoute,
    sourceRoute: ConversationRoute,
    sessionID: string,
    payload: { session?: RuntimeSessionPayload },
  ): ChatSession | null => {
    return normalizeRuntimeSession(
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

  const buildRuntimeSessionConfigPatch = (session: ChatSession): RuntimeSessionConfigPatch => ({
    title: session.title,
    model_provider_id: session.modelProviderID,
    model_id: session.modelID,
    tool_ids: normalizeSelectionIDs(session.toolIDs),
    skill_ids: normalizeSelectionIDs(session.skillIDs),
    mcp_ids: normalizeSelectionIDs(session.mcpIDs),
  });

  const persistRuntimeSessionConfig = useCallback(async (routeKey: ConversationRoute, session: ChatSession) => {
    try {
      const payload = await apiClient.patch<{ session?: RuntimeSessionPayload }>(
        `${RUNTIME_SESSION_COLLECTION_ENDPOINT}/${encodeURIComponent(session.id)}?route=${encodeURIComponent(routeKey)}`,
        buildRuntimeSessionConfigPatch(session),
      );
      const hydrated = normalizeRuntimeSession(
        payload.session || {},
        sessionsByRouteRef.current[routeKey].find((item) => item.id === session.id) || session,
        routeKey,
      );
      if (hydrated) {
        upsertRuntimeSession(routeKey, hydrated);
      }
    } catch {
    }
  }, [apiClient, upsertRuntimeSession]);

  const hydrateRuntimeSession = async (routeKey: ConversationRoute, sessionID: string): Promise<ChatSession | null> => {
    const payload = await apiClient.get<{ session?: RuntimeSessionPayload }>(
      `${RUNTIME_SESSION_COLLECTION_ENDPOINT}/${encodeURIComponent(sessionID)}?route=${encodeURIComponent(routeKey)}`,
    );
    return hydrateRuntimeSessionResponse(routeKey, routeKey, sessionID, payload);
  };

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

  const recoverInterruptedStream = async (
    routeKey: ConversationRoute,
    sessionID: string,
  ): Promise<boolean> => {
    for (let attempt = 0; attempt < 3; attempt += 1) {
      try {
        const hydrated = await hydrateRuntimeSession(routeKey, sessionID);
        const recovered = hydrated?.messages.some((message) => {
          const text = normalizeText(message.text);
          if (message.role !== "assistant") {
            return false;
          }
          if (message.taskID) {
            return true;
          }
          return message.status !== "streaming" && text !== "" && text.toLowerCase() !== "thinking...";
        });
        if (recovered) {
          upsertRuntimeSession(routeKey, hydrated);
          return true;
        }
      } catch {
      }
      if (attempt < 2) {
        await new Promise((resolve) => window.setTimeout(resolve, 1000));
      }
    }
    return false;
  };

  const sendMessageStream = async (
    routeKey: ConversationRoute,
    sessionID: string,
    assistantMessageID: string,
    content: string,
    attachments: ComposerAttachment[],
  ): Promise<StreamResult> => {
    const session = sessionsByRoute[routeKey].find((item) => item.id === sessionID) || null;
    const selection = resolveModelSelection(session, runtimeProviders(providers));
    let sawEvent = false;
    let sawDone = false;
    let output = "";
    let routeHint = "";
    let deltaFlushTimer = 0;
    const response = await fetch(STREAM_ENDPOINT, {
      method: "POST",
      headers: {
        Accept: "text/event-stream",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        session_id: sessionID,
        channel_id: "web-default",
        content,
        attachments: attachments.map(serializeMessageAttachment),
        metadata: buildMessageMetadata(session, selection, effectiveChatSkillIDs(session?.skillIDs, availableSkillIDs)),
      }),
    });
    if (!response.ok || !response.body) {
      const failure = await readResponsePayload(response);
      return {
        ok: false,
        canFallback: true,
        canRecover: false,
        error: normalizeText((failure as { error?: string } | null)?.error) || `HTTP ${response.status}`,
      };
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";
    const clearDeltaFlushTimer = () => {
      if (!deltaFlushTimer) {
        return;
      }
      window.clearTimeout(deltaFlushTimer);
      deltaFlushTimer = 0;
    };
    const flushAssistantDelta = () => {
      clearDeltaFlushTimer();
      setAssistantMessage(routeKey, sessionID, assistantMessageID, {
        text: output,
        route: routeHint,
        status: "streaming",
      });
    };
    const scheduleAssistantDeltaFlush = () => {
      if (deltaFlushTimer) {
        return;
      }
      deltaFlushTimer = window.setTimeout(flushAssistantDelta, STREAM_DELTA_FLUSH_INTERVAL_MS);
    };
    try {
      while (true) {
        const { value, done } = await reader.read();
        buffer += done ? decoder.decode() : decoder.decode(value, { stream: true });
        if (done) {
          buffer += "\n\n";
        }
        let splitIndex = buffer.indexOf("\n\n");
        while (splitIndex >= 0) {
          const parsed = parseSSEBlock(buffer.slice(0, splitIndex).replace(/\r/g, ""));
          buffer = buffer.slice(splitIndex + 2);
          if (parsed) {
            sawEvent = true;
            if (sawDone && parsed.event !== "done") {
              splitIndex = buffer.indexOf("\n\n");
              continue;
            }
            if (parsed.event === "process") {
              patchSession(routeKey, sessionID, (currentSession) => {
                const nextMessages = currentSession.messages.map((message) =>
                  message.id === assistantMessageID
	                    ? {
	                        ...message,
	                        processSteps: normalizeProcessSteps([
	                          ...removeLocalStreamProcessStep(message.processSteps),
	                          parsed.data.process_step as Record<string, unknown>,
	                        ]),
	                        status: "streaming",
	                      }
                    : message,
                );
                return { ...currentSession, messages: nextMessages };
              });
            }
            if (parsed.event === "delta") {
              const delta = typeof parsed.data.delta === "string" ? parsed.data.delta : "";
              const nextRouteHint = normalizeText(parsed.data.route);
              if (nextRouteHint) {
                routeHint = nextRouteHint;
              }
              if (delta) {
                output += delta;
                scheduleAssistantDeltaFlush();
              }
            }
            if (parsed.event === "done") {
              clearDeltaFlushTimer();
              const result = (parsed.data.result as Record<string, unknown>) || {};
              const taskID = normalizeText(parsed.data.task_id);
              const taskStatus = normalizeText(parsed.data.task_status) || "done";
              setAssistantMessage(routeKey, sessionID, assistantMessageID, {
                text: normalizeText(result.output) || output || "No response",
                route: normalizeText(result.route) || routeHint,
                source: normalizeText((result.metadata as Record<string, string> | undefined)?.["alter0.execution.source"]),
                processSteps: normalizeProcessSteps(result.process_steps),
                taskID,
                taskStatus,
                taskPending: Boolean(taskID),
                status: taskID ? taskStatus : "done",
                error: false,
              });
              if (taskID) {
                setPendingTasksVersion((value) => value + 1);
              }
              sawDone = true;
            }
            if (parsed.event === "error") {
              clearDeltaFlushTimer();
              setAssistantMessage(routeKey, sessionID, assistantMessageID, {
                text: normalizeText(parsed.data.error) || "Request failed",
                status: "error",
                error: true,
              });
              return {
                ok: false,
                canFallback: false,
                canRecover: false,
                error: normalizeText(parsed.data.error) || "request failed",
              };
            }
          }
          splitIndex = buffer.indexOf("\n\n");
        }
        if (done) {
          break;
        }
      }
    } catch (error) {
      if (output) {
        flushAssistantDelta();
      } else {
        clearDeltaFlushTimer();
      }
      return {
        ok: false,
        canFallback: !sawEvent,
        canRecover: sawEvent,
        error: error instanceof Error ? error.message : "stream interrupted",
      };
    }
    if (!sawDone && output) {
      flushAssistantDelta();
    } else {
      clearDeltaFlushTimer();
    }
    return {
      ok: sawDone,
      canFallback: !sawEvent,
      canRecover: sawEvent && !sawDone,
      error: sawDone ? "" : "stream interrupted",
    };
  };

  const sendPrompt = async (prompt: string = activeSessionID ? composerDrafts[activeSessionID] || "" : "") => {
    const content = prompt.trim().slice(0, MAX_COMPOSER_CHARS);
    const attachments = activeDraftAttachments;
    if (!content && attachments.length === 0) {
      return;
    }
    const session = ensureSession(defaultChatTarget());
    const userMessage = createMessage("user", content, { at: Date.now(), attachments });
	    const assistantMessage = createMessage("assistant", "", {
	      status: "streaming",
	      at: Date.now(),
	      processSteps: [createLocalStreamProcessStep()],
	    });
    appendMessage(route, session.id, userMessage);
    appendMessage(route, session.id, assistantMessage);
    const nextDrafts = { ...composerDrafts, [session.id]: "" };
    const nextAttachmentDrafts = { ...composerAttachmentDrafts, [session.id]: [] };
    setComposerDrafts(nextDrafts);
    setComposerAttachmentDrafts(nextAttachmentDrafts);
    persistComposerDrafts(nextDrafts);
    persistComposerAttachmentDrafts(nextAttachmentDrafts);
    try {
      const streamResult = await sendMessageStream(route, session.id, assistantMessage.id, content, attachments);
      if (!streamResult.ok && streamResult.canRecover) {
        const recovered = await recoverInterruptedStream(route, session.id);
        if (recovered) {
          return;
        }
      }
      if (!streamResult.ok && streamResult.canFallback) {
        await sendMessageFallback(route, session.id, assistantMessage.id, content, attachments);
      }
      if (!streamResult.ok && !streamResult.canFallback) {
        setAssistantMessage(route, session.id, assistantMessage.id, {
          text: streamResult.error || "Request failed",
          status: "error",
          error: true,
        });
      }
    } catch (error) {
      setAssistantMessage(route, session.id, assistantMessage.id, {
        text: error instanceof Error ? error.message : "Request failed",
        status: "error",
        error: true,
      });
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
      `/api/sessions/${encodeURIComponent(sessionID)}/attachments`,
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
    const payload = await apiClient.get<{ items?: RuntimeSessionPayload[] }>(
      `${RUNTIME_SESSION_COLLECTION_ENDPOINT}?route=${encodeURIComponent(routeKey)}`,
    );
    const remoteSessions = (Array.isArray(payload.items) ? payload.items : [])
      .map((item) => normalizeRuntimeSession(item, undefined, routeKey))
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

    setPendingTasksVersion((value) => value + 1);
  }, [activeSession, hydrateRuntimeSession, loadRuntimeSessions, route, upsertRuntimeSession]);

  useEffect(() => {
    persistActiveSessionSnapshots(activeSessionByRoute, sessionsByRoute);
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
    if (!sessionsLoadedByRoute[route] || !skillCatalogLoaded || sessionsByRoute[route].length > 0) {
      return;
    }
    ensureSession(defaultChatTarget());
    // Keep an active session available for the current runtime route.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [route, sessionsByRoute, sessionsLoadedByRoute, skillCatalogLoaded]);

  useEffect(() => {
    window.clearTimeout(pollTimerRef.current);
    const pending = Object.entries(sessionsByRoute).flatMap(([routeKey, sessions]) =>
      sessions.flatMap((session) =>
        session.messages
          .filter((message) => message.taskID && message.taskPending && !message.taskResultDelivered)
          .map((message) => ({
            route: routeKey as ConversationRoute,
            sessionID: session.id,
            messageID: message.id,
            taskID: message.taskID,
          })),
      ),
    );
    if (!pending.length) {
      return;
    }
    const pollPlan = resolveChatTaskPollPlan({
      pendingCount: pending.length,
      pageHidden,
    });
    if (!pollPlan.enabled) {
      return;
    }
    pollTimerRef.current = window.setTimeout(async () => {
      for (const item of pending) {
        try {
          const task = await apiClient.get<ChatTaskResponse>(`/api/tasks/${encodeURIComponent(item.taskID)}`);
          const status = normalizeTaskStatus(task.status || "");
          setAssistantMessage(item.route, item.sessionID, item.messageID, {
            taskStatus: status,
            taskPending: !isTerminalTaskStatus(status),
            taskResultDelivered: isTerminalTaskStatus(status),
            status,
          });
          if (isTerminalTaskStatus(status)) {
            appendMessage(item.route, item.sessionID, createMessage("assistant", normalizeText(task.summary) || "Task completed", {
              route: normalizeText(task.result?.route),
              processSteps: normalizeProcessSteps(task.result?.process_steps),
              error: status !== "success",
              status: status === "success" ? "done" : "error",
              taskResultFor: item.taskID,
            }));
          }
        } catch {
        }
      }
      setPendingTasksVersion((value) => value + 1);
    }, pollPlan.interval);
    return () => window.clearTimeout(pollTimerRef.current);
  }, [apiClient, pageHidden, pendingTasksVersion, sessionsByRoute]);

  const selection = resolveModelSelection(activeSession, availableProviders);
  const selectedProvider = enabledProviders(availableProviders).find((provider) => normalizeText(provider.id) === selection.providerID) || null;
  const selectedModel = enabledModels(selectedProvider).find((model) => normalizeText(model.id) === selection.modelID) || null;
  const currentTarget = activeSession?.target || defaultChatTarget();

  const workspaceValue = useMemo<ConversationRuntimeWorkspaceContextValue>(() => ({
    route,
    compact,
    inspectorOpen,
    inspectorTab,
    inspectorTabOpen,
    sessions: activeSessions,
    activeSession,
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
    toolCount: (activeSession?.toolIDs.length || 0) + (activeSession?.mcpIDs.length || 0),
    skillCount: activeSkillIDs.length,
    createSession: () => {
      const existingBlankDraft = activeSessions.find(isBlankDraftSession) || null;
      if (existingBlankDraft) {
        focusSession(existingBlankDraft.id);
        return;
      }
      ensureSession(null, { ...activeSessionByRoute, [route]: "" });
    },
    focusSession,
    removeSession,
    setSessionPinned,
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
      const session = activeSession || ensureSession();
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
      const session = activeSession || ensureSession();
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
      const session = activeSession || ensureSession();
      const value = normalizeText(id);
      if (!value) {
        return;
      }
      if (availableSkillIDs !== null && !availableSkillIDs.includes(value)) {
        return;
      }
      const mutate = (items: string[]) =>
        checked
          ? normalizeSelectionIDs([...effectiveChatSkillIDs(items, availableSkillIDs), value])
          : effectiveChatSkillIDs(items, availableSkillIDs).filter((item) => item !== value);
      const nextSession = {
        ...session,
        skillIDs: mutate(session.skillIDs),
      };
      patchSession(route, session.id, (currentSession) => ({
        ...currentSession,
        skillIDs: mutate(currentSession.skillIDs),
      }));
      void persistRuntimeSessionConfig(route, nextSession);
    },
    toggleProcess: (messageID: string) => {
      if (!activeSession) {
        return;
      }
      patchSession(route, activeSession.id, (session) => ({
        ...session,
        messages: session.messages.map((message) =>
          message.id === messageID
            ? { ...message, processCollapsed: !message.processCollapsed }
            : message,
        ),
      }));
    },
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
    selection.providerID,
    selection.modelID,
    selectedModel?.name,
    selectedModel?.id,
    availableProviders,
    mcps,
    skills,
    activeSessionByRoute,
    focusSession,
    persistRuntimeSessionConfig,
    removeSession,
    setSessionPinned,
  ]);

  const composerValue = useMemo<ConversationRuntimeComposerContextValue>(() => ({
    route,
    draft: activeSessionID ? composerDrafts[activeSessionID] || "" : "",
    draftAttachments: activeDraftAttachments,
    selectedModelSupportsVision: selectedModel ? selectedModel.supports_vision !== false : true,
    setDraft: (value: string) => {
      const session = ensureSession();
      const nextDrafts = { ...composerDrafts, [session.id]: value.slice(0, MAX_COMPOSER_CHARS) };
      setComposerDrafts(nextDrafts);
    },
    addDraftAttachments: async (attachments: ComposerAttachment[]) => {
      const normalized = normalizeStoredAttachments(attachments);
      if (normalized.length === 0) {
        return;
      }
      const session = ensureSession();
      const uploaded = await uploadDraftAttachments(session.id, normalized);
      const existing = composerAttachmentDrafts[session.id] || [];
      const deduped = new Map<string, ComposerAttachment>();
      [...existing, ...uploaded].forEach((item) => {
        deduped.set(item.id, item);
      });
      const nextAttachments = Array.from(deduped.values()).slice(0, MAX_COMPOSER_IMAGE_ATTACHMENTS);
      const nextDrafts = { ...composerAttachmentDrafts, [session.id]: nextAttachments };
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
    activeSessionID,
    composerDrafts,
    activeDraftAttachments,
    selectedModel,
    ensureSession,
    composerAttachmentDrafts,
    activeSession,
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
