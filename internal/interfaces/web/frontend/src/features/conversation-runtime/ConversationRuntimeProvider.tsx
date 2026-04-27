import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { createAPIClient } from "../../shared/api/client";
import { hashSessionIDShort } from "../../shared/session/sessionHash";
import type { LegacyShellLanguage } from "../shell/legacyShellCopy";
import { MOBILE_VIEWPORT_BREAKPOINT_PX } from "../../shared/viewport/mobileViewport";
import {
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  isComposerImageAttachment,
  type ComposerAttachment,
} from "./composerImageAttachments";

const ACTIVE_SESSION_STORAGE_KEY = "alter0.web.session.active.v1";
const COMPOSER_DRAFT_STORAGE_KEY = "alter0.web.composer.drafts.v1";
const COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY = "alter0.web.composer.attachments.v1";
const STREAM_ENDPOINT = "/api/messages/stream";
const AGENT_STREAM_ENDPOINT = "/api/agent/messages/stream";
const FALLBACK_ENDPOINT = "/api/messages";
const AGENT_FALLBACK_ENDPOINT = "/api/agent/messages";
const RUNTIME_SESSION_COLLECTION_ENDPOINT = "/api/conversation-runtime/sessions";
const MAX_COMPOSER_CHARS = 10000;
const CHAT_TASK_POLL_INTERVAL_MS = 3000;
const EXECUTION_ENGINE_METADATA_KEY = "alter0.execution.engine";
const EXECUTION_ENGINE_CODEX = "codex";
const LLM_PROVIDER_METADATA_KEY = "alter0.llm.provider_id";
const LLM_MODEL_METADATA_KEY = "alter0.llm.model";
const CODEX_RUNTIME_PROVIDER_ID = "alter0-codex";
const CODEX_RUNTIME_MODEL_ID = "codex";

export type ConversationRoute = "chat" | "agent-runtime";

type ChatTarget = {
  type: "model" | "agent";
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
  agentProcessCollapsed?: boolean;
  taskID: string;
  taskStatus: string;
  taskPending: boolean;
  taskResultDelivered: boolean;
  taskResultFor: string;
};

type ChatSession = {
  id: string;
  title: string;
  titleAuto: boolean;
  titleScore: number;
  createdAt: number;
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

type AgentSessionProfileField = {
  key: string;
  label: string;
  description?: string;
  readonly?: boolean;
};

type ChatAgent = {
  id: string;
  name: string;
  description?: string;
  enabled?: boolean;
  tools?: string[];
  skills?: string[];
  mcps?: string[];
  capabilities?: string[];
  session_profile_fields?: AgentSessionProfileField[];
};

type ChatAgentSessionProfile = {
  agent_id: string;
  session_id: string;
  path: string;
  exists: boolean;
  fields: AgentSessionProfileField[];
  attributes: Record<string, string>;
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

type RuntimeSelection = {
  id: string;
  name: string;
  description: string;
  kind: "tool" | "mcp" | "skill";
  active: boolean;
  visibility?: "public" | "agent-private";
  locked?: boolean;
  ownerAgentID?: string;
};

type RuntimeTargetOption = {
  type: "agent";
  id: string;
  name: string;
  subtitle: string;
  active: boolean;
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
  title?: string;
  title_auto?: boolean;
  title_score?: number;
  created_at?: string | number;
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
  inspectorTab: "target" | "model" | "capabilities" | "skills" | "session-profile";
  inspectorTabOpen: boolean;
  sessions: ChatSession[];
  activeSession: ChatSession | null;
  sessionItems: Array<{
    id: string;
    title: string;
    meta: string;
    shortHash: string;
    createdAt: number;
    active: boolean;
  }>;
  draft: string;
  target: ChatTarget;
  activeAgent: ChatAgent | null;
  activeSessionProfile: ChatAgentSessionProfile | null;
  lockedTarget: boolean;
  targetOptions: RuntimeTargetOption[];
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
  setDraft: (value: string) => void;
  addDraftAttachments: (attachments: ComposerAttachment[]) => Promise<void>;
  removeDraftAttachment: (attachmentID: string) => void;
  clearDraftAttachments: () => void;
  sendPrompt: (prompt?: string) => Promise<void>;
  toggleInspector: (tab?: "target" | "model" | "capabilities" | "skills" | "session-profile") => void;
  closeInspector: () => void;
  selectTarget: (targetID: string) => void;
  selectModel: (providerID: string, modelID: string) => void;
  toggleCapability: (id: string, kind: "tool" | "mcp", checked: boolean) => void;
  toggleSkill: (id: string, checked: boolean) => void;
  toggleAgentProcess: (messageID: string) => void;
};

const ConversationRuntimeContext = createContext<ConversationRuntimeContextValue | null>(null);

type ProviderProps = {
  route: ConversationRoute;
  language: LegacyShellLanguage;
  children: ReactNode;
};

type StreamResult = {
  ok: boolean;
  canFallback: boolean;
  error: string;
};

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeRuntimeAgentID(value: unknown): string {
  const normalized = normalizeText(value)
    .toLowerCase()
    .replace(/[^a-z0-9._-]+/g, "-")
    .replace(/^[-.]+|[-.]+$/g, "");
  return normalized || "unknown";
}

function agentHasCapability(agent: ChatAgent | null, capability: string): boolean {
  const lookup = capability.toLowerCase();
  return Boolean(agent?.capabilities?.some((item) => normalizeText(item).toLowerCase() === lookup));
}

function isTravelRuntimeAgent(agent: ChatAgent | null): boolean {
  const id = normalizeText(agent?.id).toLowerCase();
  const name = normalizeText(agent?.name).toLowerCase();
  return id === "travel" || name.includes("travel") || agentHasCapability(agent, "travel");
}

function buildAgentPrivateSkill(route: ConversationRoute, agent: ChatAgent | null): RuntimeSelection | null {
  if (route !== "agent-runtime" || !agent) {
    return null;
  }
  const normalizedAgentID = normalizeRuntimeAgentID(agent.id);
  const name = normalizeText(agent.name) || normalizeText(agent.id) || "Agent";
  const description = isTravelRuntimeAgent(agent)
    ? "Private reusable rulebook for the current travel agent's city-page structure, itinerary composition, rendering conventions, and stable travel preferences."
    : "Private reusable rulebook for the current agent's execution patterns, output structure, domain heuristics, and stable preferences.";
  return {
    id: `agent-skill-${normalizedAgentID}`,
    name: `${name} Skill`,
    description,
    kind: "skill",
    active: true,
    visibility: "agent-private",
    locked: true,
    ownerAgentID: normalizeText(agent.id),
  };
}

function isPublicSkillCapability(skill: ChatCapability): boolean {
  const metadata = skill.metadata || {};
  const visibility = normalizeText(metadata["alter0.skill.visibility"] || metadata["skill.visibility"]).toLowerCase();
  return visibility !== "agent-private" && visibility !== "private";
}

function makeID(prefix: string): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

function normalizeChatTarget(target?: { type?: string; id?: string; name?: string } | null): ChatTarget {
  const type = target?.type === "agent" ? "agent" : "model";
  const id = normalizeText(target?.id) || (type === "agent" ? "" : "raw-model");
  const name = normalizeText(target?.name) || (type === "agent" ? id : "Raw Model");
  return { type, id, name };
}

function normalizeAgentSessionProfileField(item: unknown): AgentSessionProfileField | null {
  if (!item || typeof item !== "object") {
    return null;
  }
  const record = item as Record<string, unknown>;
  const key = normalizeText(record.key);
  const label = normalizeText(record.label);
  if (!key || !label) {
    return null;
  }
  return {
    key,
    label,
    description: normalizeText(record.description) || undefined,
    readonly: record.readonly === true,
  };
}

function normalizeAgentSessionProfileFields(items: unknown): AgentSessionProfileField[] {
  if (!Array.isArray(items)) {
    return [];
  }
  const deduped = new Map<string, AgentSessionProfileField>();
  items.forEach((item) => {
    const field = normalizeAgentSessionProfileField(item);
    if (!field || deduped.has(field.key.toLowerCase())) {
      return;
    }
    deduped.set(field.key.toLowerCase(), field);
  });
  return Array.from(deduped.values());
}

function normalizeAgentSessionProfileAttributes(items: unknown): Record<string, string> {
  if (!items || typeof items !== "object") {
    return {};
  }
  return Object.entries(items as Record<string, unknown>).reduce<Record<string, string>>((acc, [key, value]) => {
    const normalizedKey = normalizeText(key);
    const normalizedValue = normalizeText(value);
    if (!normalizedKey || !normalizedValue) {
      return acc;
    }
    acc[normalizedKey] = normalizedValue;
    return acc;
  }, {});
}

function normalizeAgentSessionProfile(
  payload: unknown,
  fallbackAgentID: string,
  fallbackSessionID: string,
  fallbackFields: AgentSessionProfileField[],
): ChatAgentSessionProfile {
  const record = payload && typeof payload === "object" ? payload as Record<string, unknown> : {};
  const fields = normalizeAgentSessionProfileFields(record.fields);
  return {
    agent_id: normalizeText(record.agent_id) || fallbackAgentID,
    session_id: normalizeText(record.session_id) || fallbackSessionID,
    path: normalizeText(record.path),
    exists: record.exists === true,
    fields: fields.length > 0 ? fields : fallbackFields,
    attributes: normalizeAgentSessionProfileAttributes(record.attributes),
  };
}

function buildFallbackAgentSessionProfile(agent: ChatAgent | null, sessionID: string): ChatAgentSessionProfile | null {
  if (!agent || !sessionID) {
    return null;
  }
  return {
    agent_id: normalizeText(agent.id),
    session_id: sessionID,
    path: "",
    exists: false,
    fields: normalizeAgentSessionProfileFields(agent.session_profile_fields),
    attributes: {},
  };
}

function defaultChatTarget(): ChatTarget {
  return normalizeChatTarget({ type: "model", id: "raw-model", name: "Raw Model" });
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

function isSelectableRuntimeAgent(agent: ChatAgent | null): agent is ChatAgent {
  if (!agent || agent.enabled === false) {
    return false;
  }
  const id = normalizeText(agent.id).toLowerCase();
  const name = normalizeText(agent.name).toLowerCase();
  return id !== "main" && id !== "alter0" && name !== "alter0";
}

function normalizeChatAgent(item: unknown): ChatAgent | null {
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
    name: normalizeText(record.name) || id,
    description: normalizeText(record.description) || undefined,
    enabled: record.enabled !== false,
    tools: normalizeSelectionIDs(record.tools),
    skills: normalizeSelectionIDs(record.skills),
    mcps: normalizeSelectionIDs(record.mcps),
    session_profile_fields: normalizeAgentSessionProfileFields(record.session_profile_fields),
  };
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
    agentProcessCollapsed:
      typeof record.agent_process_collapsed === "boolean"
        ? record.agent_process_collapsed
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
    title: normalizeText(record.title) || "New",
    titleAuto: record.titleAuto !== false,
    titleScore: Number.isFinite(Number(record.titleScore)) ? Number(record.titleScore) : 0,
    createdAt: Number.isFinite(Number(record.createdAt)) ? Number(record.createdAt) : Date.now(),
    target: normalizeChatTarget({
      type: normalizeText(record.targetType) === "agent" ? "agent" : "model",
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
    chat: normalizeText(parsed.chat),
    "agent-runtime": normalizeText(parsed["agent-runtime"]),
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

function normalizeRuntimeSession(item: RuntimeSessionPayload, previous?: ChatSession | null): ChatSession | null {
  const id = normalizeText(item.id);
  if (!id) {
    return null;
  }
  const parsedMessages = Array.isArray(item.messages)
    ? item.messages.map(normalizeRuntimeMessage).filter((message): message is ChatMessage => message !== null)
    : null;
  const messages = parsedMessages
    ? (previous?.messages.length && parsedMessages.length < previous.messages.length
      ? previous.messages
      : parsedMessages)
    : previous?.messages || [];
  return {
    id,
    title: normalizeText(item.title) || previous?.title || "New",
    titleAuto: item.title_auto !== false,
    titleScore: Number.isFinite(Number(item.title_score)) ? Number(item.title_score) : previous?.titleScore || 0,
    createdAt: normalizeDateValue(item.created_at),
    target: normalizeChatTarget({
      type: normalizeText(item.target_type) === "agent" ? "agent" : "model",
      id: normalizeText(item.target_id),
      name: normalizeText(item.target_name),
    }),
    modelProviderID: normalizeText(item.model_provider_id) || previous?.modelProviderID || "",
    modelID: normalizeText(item.model_id) || previous?.modelID || "",
    toolIDs: normalizeSelectionIDs(item.tool_ids || previous?.toolIDs || []),
    skillIDs: normalizeSelectionIDs(item.skill_ids || previous?.skillIDs || []),
    mcpIDs: normalizeSelectionIDs(item.mcp_ids || previous?.mcpIDs || []),
    messages,
    messagesLoaded: Array.isArray(item.messages) ? true : previous?.messagesLoaded,
    serverBacked: true,
  };
}

function mergeRuntimeSessions(remote: ChatSession[], existing: ChatSession[]): ChatSession[] {
  const merged = new Map<string, ChatSession>();
  remote.forEach((session) => {
    merged.set(session.id, session);
  });
  existing
    .filter((session) => session.serverBacked !== true)
    .forEach((session) => {
      if (!merged.has(session.id)) {
        merged.set(session.id, session);
      }
    });
  return Array.from(merged.values()).sort((left, right) => right.createdAt - left.createdAt);
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

function buildSessionMeta(session: ChatSession, language: LegacyShellLanguage): string {
  const targetLabel = session.target.type === "agent" ? `Agent · ${session.target.name}` : "Chat";
  const countLabel = language === "zh"
    ? `${session.messages.length} 条消息`
    : `${session.messages.length} messages`;
  return `${targetLabel} · ${countLabel} · ${formatRelativeTime(session.createdAt, language)}`;
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
): Record<string, string> {
  const metadata: Record<string, string> = {
    "alter0.agent.tools": JSON.stringify(session?.toolIDs || []),
    "alter0.skills.include": JSON.stringify(session?.skillIDs || []),
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
  route,
  language,
  children,
}: ProviderProps) {
  const apiClient = useMemo(() => createAPIClient(), []);
  const [sessionsByRoute, setSessionsByRoute] = useState<SessionsState>({ chat: [], "agent-runtime": [] });
  const [sessionsLoadedByRoute, setSessionsLoadedByRoute] = useState<Record<ConversationRoute, boolean>>({
    chat: false,
    "agent-runtime": false,
  });
  const [activeSessionByRoute, setActiveSessionByRoute] = useState<ActiveSessionState>(() =>
    loadActiveSessionState(),
  );
  const [selectedAgentID, setSelectedAgentID] = useState("");
  const [providers, setProviders] = useState<ChatProvider[]>([]);
  const [skills, setSkills] = useState<ChatCapability[]>([]);
  const [mcps, setMcps] = useState<ChatCapability[]>([]);
  const [agents, setAgents] = useState<ChatAgent[]>([]);
  const [agentSessionProfiles, setAgentSessionProfiles] = useState<Record<string, ChatAgentSessionProfile>>({});
  const [composerDrafts, setComposerDrafts] = useState<ComposerDraftMap>(() => loadComposerDrafts());
  const [composerAttachmentDrafts, setComposerAttachmentDrafts] = useState<ComposerAttachmentDraftMap>(() => loadComposerAttachmentDrafts());
  const [compact, setCompact] = useState(() => isCompactViewport());
  const [inspectorOpen, setInspectorOpen] = useState(false);
  const [inspectorTab, setInspectorTab] = useState<"target" | "model" | "capabilities" | "skills" | "session-profile">("model");
  const [inspectorTabOpen, setInspectorTabOpen] = useState(true);
  const [pendingTasksVersion, setPendingTasksVersion] = useState(0);
  const pollTimerRef = useRef<number>(0);

  const activeSessions = sessionsByRoute[route];
  const activeSessionID = activeSessionByRoute[route];
  const activeSession = activeSessions.find((session) => session.id === activeSessionID) || null;
  const activeDraftAttachments = activeSessionID ? composerAttachmentDrafts[activeSessionID] || [] : [];
  const availableProviders = runtimeProviders(providers);
  const activeAgent = activeSession?.target.type === "agent"
    ? agents.find((agent) => normalizeText(agent.id) === normalizeText(activeSession.target.id)) || null
    : null;
  const activeSessionProfileKey = activeAgent && activeSession
    ? `${normalizeText(activeAgent.id)}:${activeSession.id}`
    : "";
  const activeSessionProfile = activeSessionProfileKey
    ? agentSessionProfiles[activeSessionProfileKey] || buildFallbackAgentSessionProfile(activeAgent, activeSession?.id || "")
    : null;

  const ensureSession = (
    target?: Partial<ChatTarget> | null,
    preferredActiveState: ActiveSessionState = activeSessionByRoute,
    currentSessions: SessionsState = sessionsByRoute,
  ) => {
    const targetValue = normalizeChatTarget(
      target || (route === "agent-runtime"
        ? {
            type: "agent",
            id: selectedAgentID,
            name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
          }
        : defaultChatTarget()),
    );
    const existing = currentSessions[route].find((session) => session.id === preferredActiveState[route]) || null;
    if (existing) {
      if (
        route === "agent-runtime"
        && targetValue.type === "agent"
        && targetValue.id
        && existing.target.id !== targetValue.id
        && existing.messages.length === 0
      ) {
        const nextSession = {
          ...existing,
          target: targetValue,
          toolIDs: normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === targetValue.id)?.tools),
          skillIDs: normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === targetValue.id)?.skills),
          mcpIDs: normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === targetValue.id)?.mcps),
        };
        const nextSessionsByRoute = {
          ...currentSessions,
          [route]: currentSessions[route].map((session) => session.id === existing.id ? nextSession : session),
        };
        setSessionsByRoute(nextSessionsByRoute);
        return nextSession;
      }
      return existing;
    }
    const created: ChatSession = {
      id: makeID("session"),
      title: "New",
      titleAuto: true,
      titleScore: 0,
      createdAt: Date.now(),
      target: targetValue,
      modelProviderID: "",
      modelID: "",
      toolIDs: targetValue.type === "agent"
        ? normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === targetValue.id)?.tools)
        : [],
      skillIDs: targetValue.type === "agent"
        ? normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === targetValue.id)?.skills)
        : [],
      mcpIDs: targetValue.type === "agent"
        ? normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === targetValue.id)?.mcps)
        : [],
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
  };

  const patchSession = (
    routeKey: ConversationRoute,
    sessionID: string,
    updater: (session: ChatSession) => ChatSession,
  ) => {
    setSessionsByRoute((current) => ({
      ...current,
      [routeKey]: current[routeKey].map((session) =>
        session.id === sessionID ? updater(session) : session,
      ),
    }));
  };

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
    agentProcessCollapsed: patch.agentProcessCollapsed,
    taskID: patch.taskID || "",
    taskStatus: patch.taskStatus || "",
    taskPending: Boolean(patch.taskPending),
    taskResultDelivered: Boolean(patch.taskResultDelivered),
    taskResultFor: patch.taskResultFor || "",
  });

  const appendMessage = (routeKey: ConversationRoute, sessionID: string, message: ChatMessage) => {
    patchSession(routeKey, sessionID, (session) => ({
      ...session,
      title: session.titleAuto && message.role === "user"
        ? (message.text.slice(0, 32) || session.title)
        : session.title,
      titleAuto: session.titleAuto && message.role !== "user",
      serverBacked: true,
      messages: [...session.messages, message],
    }));
  };

  const setAssistantMessage = (
    routeKey: ConversationRoute,
    sessionID: string,
    messageID: string,
    patch: Partial<ChatMessage>,
  ) => {
    patchSession(routeKey, sessionID, (session) => ({
      ...session,
      messages: session.messages.map((message) =>
        message.id === messageID ? { ...message, ...patch } : message,
      ),
    }));
  };

  const focusSession = (sessionID: string) => {
    const nextActiveState = { ...activeSessionByRoute, [route]: sessionID };
    setActiveSessionByRoute(nextActiveState);
    writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
  };

  const removeSession = async (sessionID: string) => {
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
    const nextDrafts = { ...composerDrafts };
    const nextAttachmentDrafts = { ...composerAttachmentDrafts };
    delete nextDrafts[sessionID];
    delete nextAttachmentDrafts[sessionID];
    setSessionsByRoute(nextSessionsByRoute);
    setActiveSessionByRoute(nextActiveState);
    setComposerDrafts(nextDrafts);
    setComposerAttachmentDrafts(nextAttachmentDrafts);
    persistComposerDrafts(nextDrafts);
    persistComposerAttachmentDrafts(nextAttachmentDrafts);
    writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
  };

  const sendMessageFallback = async (
    routeKey: ConversationRoute,
    sessionID: string,
    target: ChatTarget,
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
    }>(
      routeKey === "agent-runtime" ? AGENT_FALLBACK_ENDPOINT : FALLBACK_ENDPOINT,
      {
        session_id: sessionID,
        channel_id: "web-default",
        content,
        attachments: attachments.map(serializeMessageAttachment),
        metadata: buildMessageMetadata(session, selection),
        ...(routeKey === "agent-runtime" ? { agent_id: target.id } : {}),
      },
    );
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

  const sendMessageStream = async (
    routeKey: ConversationRoute,
    sessionID: string,
    target: ChatTarget,
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
    const response = await fetch(routeKey === "agent-runtime" ? AGENT_STREAM_ENDPOINT : STREAM_ENDPOINT, {
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
        metadata: buildMessageMetadata(session, selection),
        ...(routeKey === "agent-runtime" ? { agent_id: target.id } : {}),
      }),
    });
    if (!response.ok || !response.body) {
      const failure = await readResponsePayload(response);
      return {
        ok: false,
        canFallback: true,
        error: normalizeText((failure as { error?: string } | null)?.error) || `HTTP ${response.status}`,
      };
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";
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
                        ...message.processSteps,
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
              setAssistantMessage(routeKey, sessionID, assistantMessageID, {
                text: output,
                route: routeHint,
                status: "streaming",
              });
            }
          }
          if (parsed.event === "done") {
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
            setAssistantMessage(routeKey, sessionID, assistantMessageID, {
              text: normalizeText(parsed.data.error) || "Request failed",
              status: "error",
              error: true,
            });
            return { ok: false, canFallback: false, error: normalizeText(parsed.data.error) || "request failed" };
          }
        }
        splitIndex = buffer.indexOf("\n\n");
      }
      if (done) {
        break;
      }
    }
    return {
      ok: sawDone,
      canFallback: !sawEvent,
      error: sawDone ? "" : "stream interrupted",
    };
  };

  const sendPrompt = async (prompt: string = activeSessionID ? composerDrafts[activeSessionID] || "" : "") => {
    const content = prompt.trim().slice(0, MAX_COMPOSER_CHARS);
    const attachments = activeDraftAttachments;
    if ((!content && attachments.length === 0) || (route === "agent-runtime" && !selectedAgentID)) {
      return;
    }
    const session = ensureSession(route === "agent-runtime"
      ? {
          type: "agent",
          id: selectedAgentID,
          name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
        }
      : defaultChatTarget());
    const userMessage = createMessage("user", content, { at: Date.now(), attachments });
    const assistantMessage = createMessage("assistant", "Thinking...", {
      status: "streaming",
      at: Date.now(),
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
      const streamResult = await sendMessageStream(route, session.id, session.target, assistantMessage.id, content, attachments);
      if (!streamResult.ok && streamResult.canFallback) {
        await sendMessageFallback(route, session.id, session.target, assistantMessage.id, content, attachments);
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
      .map((item) => normalizeRuntimeSession(item))
      .filter((session): session is ChatSession => session !== null);
    setSessionsByRoute((current) => ({
      ...current,
      [routeKey]: mergeRuntimeSessions(remoteSessions, current[routeKey]),
    }));
    setSessionsLoadedByRoute((current) => ({ ...current, [routeKey]: true }));
    return remoteSessions;
  };

  useEffect(() => {
    const syncViewport = () => setCompact(isCompactViewport());
    window.addEventListener("resize", syncViewport);
    return () => window.removeEventListener("resize", syncViewport);
  }, []);

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
      }
      try {
        const agentPayload = await apiClient.get<{ items?: ChatAgent[] }>("/api/agents");
        const nextAgents = Array.isArray(agentPayload.items)
          ? agentPayload.items.map(normalizeChatAgent).filter(isSelectableRuntimeAgent)
          : [];
        setAgents(nextAgents);
        setSelectedAgentID((current) => current || normalizeText(nextAgents[0]?.id));
      } catch {
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
        const preferredActiveID = normalizeText(activeSessionByRoute[route]);
        const nextActiveID = remoteSessions.some((session) => session.id === preferredActiveID)
          ? preferredActiveID
          : remoteSessions[0]?.id || activeSessionByRoute[route];
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
    if (!activeSession?.id || activeSession.serverBacked !== true || activeSession.messagesLoaded) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const payload = await apiClient.get<{ session?: RuntimeSessionPayload }>(
          `${RUNTIME_SESSION_COLLECTION_ENDPOINT}/${encodeURIComponent(activeSession.id)}?route=${encodeURIComponent(route)}`,
        );
        if (cancelled) {
          return;
        }
        const hydrated = normalizeRuntimeSession(
          payload.session || {},
          sessionsByRoute[route].find((item) => item.id === activeSession.id) || null,
        );
        if (!hydrated) {
          return;
        }
        setSessionsByRoute((current) => ({
          ...current,
          [route]: current[route]
            .map((session) => (session.id === activeSession.id ? hydrated : session))
            .sort((left, right) => right.createdAt - left.createdAt),
        }));
      } catch {
        if (cancelled) {
          return;
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [activeSession?.id, activeSession?.messagesLoaded, activeSession?.serverBacked, apiClient, route, sessionsByRoute]);

  useEffect(() => {
    if (route !== "agent-runtime" || !activeSession || activeSession.target.type !== "agent") {
      return;
    }
    const agentID = normalizeText(activeSession.target.id);
    if (!agentID) {
      return;
    }
    const profileKey = `${agentID}:${activeSession.id}`;
    if (agentSessionProfiles[profileKey]) {
      return;
    }
    const fallbackFields = normalizeAgentSessionProfileFields(
      agents.find((agent) => normalizeText(agent.id) === agentID)?.session_profile_fields,
    );
    let cancelled = false;
    void (async () => {
      try {
        const payload = await apiClient.get<ChatAgentSessionProfile>(
          `/api/agent/session-profile?agent_id=${encodeURIComponent(agentID)}&session_id=${encodeURIComponent(activeSession.id)}`,
        );
        if (cancelled) {
          return;
        }
        setAgentSessionProfiles((current) => ({
          ...current,
          [profileKey]: normalizeAgentSessionProfile(payload, agentID, activeSession.id, fallbackFields),
        }));
      } catch {
        if (cancelled) {
          return;
        }
        setAgentSessionProfiles((current) => ({
          ...current,
          [profileKey]: normalizeAgentSessionProfile({}, agentID, activeSession.id, fallbackFields),
        }));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [activeSession, agentSessionProfiles, agents, apiClient, route]);

  useEffect(() => {
    if (!sessionsLoadedByRoute[route] || sessionsByRoute[route].length > 0) {
      return;
    }
    ensureSession(route === "agent-runtime"
      ? {
          type: "agent",
          id: selectedAgentID,
          name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
        }
      : defaultChatTarget());
    // Keep an active session available for the current runtime route.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [route, selectedAgentID, agents, sessionsByRoute, sessionsLoadedByRoute]);

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
    }, CHAT_TASK_POLL_INTERVAL_MS);
    return () => window.clearTimeout(pollTimerRef.current);
  }, [apiClient, pendingTasksVersion, sessionsByRoute]);

  const selection = resolveModelSelection(activeSession, availableProviders);
  const selectedProvider = enabledProviders(availableProviders).find((provider) => normalizeText(provider.id) === selection.providerID) || null;
  const selectedModel = enabledModels(selectedProvider).find((model) => normalizeText(model.id) === selection.modelID) || null;
  const currentTarget = activeSession?.target || (route === "agent-runtime"
    ? normalizeChatTarget({
        type: "agent",
        id: selectedAgentID,
        name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
      })
    : defaultChatTarget());
  const currentAgent = currentTarget.type === "agent"
    ? agents.find((agent) => normalizeText(agent.id) === currentTarget.id) || null
    : null;

  const contextValue = useMemo<ConversationRuntimeContextValue>(() => ({
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
      meta: buildSessionMeta(session, language),
      shortHash: hashSessionIDShort(session.id),
      createdAt: session.createdAt,
      active: session.id === activeSessionID,
    })),
    draft: activeSessionID ? composerDrafts[activeSessionID] || "" : "",
    draftAttachments: activeDraftAttachments,
    target: currentTarget,
    activeAgent: currentAgent,
    activeSessionProfile,
    lockedTarget: Boolean(activeSession?.messages.length),
    targetOptions: route === "agent-runtime"
      ? agents
          .filter((agent) => normalizeText(agent.id))
          .map((agent) => ({
            type: "agent" as const,
            id: normalizeText(agent.id),
            name: normalizeText(agent.name) || normalizeText(agent.id),
            subtitle: normalizeText(agent.description) || "Agent",
            active: normalizeText(agent.id) === currentTarget.id,
          }))
      : [],
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
      {
        id: "memory",
        name: "Memory",
        description: "Search memory files",
        kind: "tool" as const,
        active: Boolean(activeSession?.toolIDs.includes("memory")),
      },
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
      buildAgentPrivateSkill(route, currentAgent),
      ...skills
        .filter((item) => item.enabled !== false && isPublicSkillCapability(item))
        .map((item) => ({
          id: normalizeText(item.id),
          name: normalizeText(item.name) || normalizeText(item.id),
          description: normalizeText(item.description) || normalizeText(item.scope) || "Skill",
          kind: "skill" as const,
          active: Boolean(activeSession?.skillIDs.includes(normalizeText(item.id))),
          visibility: "public" as const,
          locked: false,
        }))
        .filter((item) => item.id),
    ].filter((item): item is RuntimeSelection => Boolean(item?.id)),
    toolCount: (activeSession?.toolIDs.length || 0) + (activeSession?.mcpIDs.length || 0),
    skillCount: (activeSession?.skillIDs.length || 0) + (buildAgentPrivateSkill(route, currentAgent) ? 1 : 0),
    createSession: () => {
      ensureSession(null, { ...activeSessionByRoute, [route]: "" });
    },
    focusSession,
    removeSession,
    setDraft: (value: string) => {
      const session = ensureSession();
      const nextDrafts = { ...composerDrafts, [session.id]: value.slice(0, MAX_COMPOSER_CHARS) };
      setComposerDrafts(nextDrafts);
      persistComposerDrafts(nextDrafts);
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
    selectTarget: (targetID: string) => {
      if (route !== "agent-runtime") {
        return;
      }
      const normalizedTarget = normalizeChatTarget({
        type: "agent",
        id: targetID,
        name: agents.find((agent) => normalizeText(agent.id) === targetID)?.name || targetID,
      });
      setSelectedAgentID(normalizedTarget.id);
      if (activeSessionID) {
        patchSession(route, activeSessionID, (session) =>
          session.messages.length > 0
            ? session
            : {
                ...session,
                target: normalizedTarget,
                toolIDs: normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === normalizedTarget.id)?.tools),
                skillIDs: normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === normalizedTarget.id)?.skills),
                mcpIDs: normalizeSelectionIDs(agents.find((agent) => normalizeText(agent.id) === normalizedTarget.id)?.mcps),
              },
        );
      }
    },
    selectModel: (providerID: string, modelID: string) => {
      const session = activeSession || ensureSession();
      patchSession(route, session.id, (currentSession) => ({
        ...currentSession,
        modelProviderID: normalizeText(providerID),
        modelID: normalizeText(modelID),
      }));
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
      patchSession(route, session.id, (currentSession) =>
        kind === "tool"
          ? { ...currentSession, toolIDs: mutate(currentSession.toolIDs) }
          : { ...currentSession, mcpIDs: mutate(currentSession.mcpIDs) },
      );
    },
    toggleSkill: (id: string, checked: boolean) => {
      const session = activeSession || ensureSession();
      const value = normalizeText(id);
      if (!value) {
        return;
      }
      if (value === buildAgentPrivateSkill(route, currentAgent)?.id) {
        return;
      }
      const mutate = (items: string[]) =>
        checked
          ? normalizeSelectionIDs([...items, value])
          : items.filter((item) => item !== value);
      patchSession(route, session.id, (currentSession) => ({
        ...currentSession,
        skillIDs: mutate(currentSession.skillIDs),
      }));
    },
    toggleAgentProcess: (messageID: string) => {
      if (!activeSession) {
        return;
      }
      patchSession(route, activeSession.id, (session) => ({
        ...session,
        messages: session.messages.map((message) =>
          message.id === messageID
            ? { ...message, agentProcessCollapsed: !message.agentProcessCollapsed }
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
    composerDrafts,
    composerAttachmentDrafts,
    activeDraftAttachments,
    currentTarget,
    currentAgent,
    activeSessionProfile,
    agents,
    selection.providerID,
    selection.modelID,
    selectedModel?.name,
    selectedModel?.id,
    availableProviders,
    mcps,
    skills,
    activeSessionByRoute,
    removeSession,
    sendPrompt,
  ]);

  return (
    <ConversationRuntimeContext.Provider value={contextValue}>
      {children}
    </ConversationRuntimeContext.Provider>
  );
}

export function useConversationRuntime() {
  const value = useContext(ConversationRuntimeContext);
  if (!value) {
    throw new Error("ConversationRuntimeContext is not available");
  }
  return value;
}
