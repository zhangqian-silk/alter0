import { useEffect, useMemo, useRef, useState } from "react";
import { createAPIClient } from "../shared/api/client";
import {
  LEGACY_SHELL_CREATE_SESSION_EVENT,
  LEGACY_SHELL_FOCUS_SESSION_EVENT,
  LEGACY_SHELL_NAVIGATE_EVENT,
  LEGACY_SHELL_QUICK_PROMPT_EVENT,
  LEGACY_SHELL_REMOVE_SESSION_EVENT,
  LEGACY_SHELL_TOGGLE_LANGUAGE_EVENT,
  type LegacyShellChatRuntimeDetail,
  type LegacyShellChatRuntimeItemActionDetail,
  type LegacyShellChatRuntimeModelActionDetail,
  type LegacyShellChatRuntimeTargetActionDetail,
  type LegacyShellMessageProcessStepDetail,
  type LegacyShellMessageRegionDetail,
  type LegacyShellSessionPaneDetail,
} from "../features/shell/legacyShellBridge";
import {
  ensureLegacyRuntimeSnapshotBridge,
  publishLegacyRuntimeSnapshot,
} from "../features/shell/legacyRuntimeSnapshotStore";
import {
  LEGACY_SHELL_DEFAULT_ROUTE,
  isLegacyShellChatRoute,
  isLegacyShellMobileViewport,
  parseLegacyShellHashRoute,
} from "../features/shell/legacyShellState";
import {
  getLegacyRouteHeadingCopy,
  normalizeLegacyShellLanguage,
  type LegacyShellLanguage,
} from "../features/shell/legacyShellCopy";
import { LEGACY_SHELL_IDS } from "../features/shell/legacyDomContract";
import { createMobileViewportSyncController } from "../shared/viewport/mobileViewportSync";

const SESSION_STORAGE_KEY = "alter0.web.sessions.v3";
const ACTIVE_SESSION_STORAGE_KEY = "alter0.web.session.active.v1";
const COMPOSER_DRAFT_STORAGE_KEY = "alter0.web.composer.drafts.v1";
const STREAM_ENDPOINT = "/api/messages/stream";
const AGENT_STREAM_ENDPOINT = "/api/agent/messages/stream";
const FALLBACK_ENDPOINT = "/api/messages";
const AGENT_FALLBACK_ENDPOINT = "/api/agent/messages";
const MAX_COMPOSER_CHARS = 10000;
const CHAT_TASK_POLL_INTERVAL_MS = 3000;

type ChatRoute = "chat" | "agent-runtime";

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

type ChatMessage = {
  id: string;
  role: "user" | "assistant";
  text: string;
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
};

type ChatProviderModel = {
  id: string;
  name: string;
  is_enabled?: boolean;
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
};

type ChatAgent = {
  id: string;
  name: string;
  description?: string;
  enabled?: boolean;
  ui_route?: string;
  tools?: string[];
  skills?: string[];
  mcps?: string[];
};

type ChatTaskResponse = {
  id?: string;
  status?: string;
  summary?: string;
  result?: {
    route?: string;
    process_steps?: LegacyShellMessageProcessStepDetail[];
    metadata?: Record<string, string>;
  };
};

type StreamResult = {
  ok: boolean;
  canFallback: boolean;
  error: string;
};

type ActiveSessionState = Record<ChatRoute, string>;
type SessionsState = Record<ChatRoute, ChatSession[]>;
type ComposerDraftMap = Record<string, string>;

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function makeID(prefix: string): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

function hashShort(value: string): string {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(16).padStart(8, "0").slice(0, 8);
}

function routeConversationMode(route: string): ChatRoute {
  return route === "agent-runtime" ? "agent-runtime" : "chat";
}

function normalizeChatTarget(target?: Partial<ChatTarget> | null): ChatTarget {
  const type = target?.type === "agent" ? "agent" : "model";
  const id = normalizeText(target?.id) || (type === "agent" ? "" : "raw-model");
  const name = normalizeText(target?.name) || (type === "agent" ? id : "Raw Model");
  return { type, id, name };
}

function defaultChatTarget(): ChatTarget {
  return normalizeChatTarget({ type: "model", id: "raw-model", name: "Raw Model" });
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
      const detail = item as LegacyShellMessageProcessStepDetail | undefined;
      const title = normalizeText(detail?.title);
      const body = normalizeText(detail?.detail);
      if (!title && !body) {
        return null;
      }
      return {
        id: normalizeText(detail?.id),
        kind: normalizeText(detail?.kind),
        title,
        detail: body,
        status: normalizeText(detail?.status),
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
    title: normalizeText(record.title) || "New Chat",
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

function loadStoredSessions(): ChatSession[] {
  const parsed = readJSONStorage<unknown[]>(SESSION_STORAGE_KEY, []);
  if (!Array.isArray(parsed)) {
    return [];
  }
  return parsed
    .map(normalizeStoredSession)
    .filter((session): session is ChatSession => session !== null)
    .sort((left, right) => right.createdAt - left.createdAt);
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

function splitSessionsByRoute(sessions: ChatSession[]): SessionsState {
  return {
    chat: sessions.filter((session) => session.target.type !== "agent"),
    "agent-runtime": sessions.filter((session) => session.target.type === "agent"),
  };
}

function toPersistedSessions(sessionsByRoute: SessionsState): unknown[] {
  const deduped = new Map<string, ChatSession>();
  [...sessionsByRoute.chat, ...sessionsByRoute["agent-runtime"]].forEach((session) => {
    const existing = deduped.get(session.id);
    if (!existing || existing.createdAt <= session.createdAt) {
      deduped.set(session.id, session);
    }
  });
  return Array.from(deduped.values()).sort((left, right) => right.createdAt - left.createdAt).map((session) => ({
    id: session.id,
    title: session.title,
    titleAuto: session.titleAuto,
    titleScore: session.titleScore,
    createdAt: session.createdAt,
    targetType: session.target.type,
    targetID: session.target.id,
    targetName: session.target.name,
    modelProviderID: session.modelProviderID,
    modelID: session.modelID,
    toolIDs: session.toolIDs,
    skillIDs: session.skillIDs,
    mcpIDs: session.mcpIDs,
    messages: session.messages.map((message) => ({
      id: message.id,
      role: message.role,
      text: message.text,
      route: message.route,
      source: message.source,
      error: message.error,
      status: message.status,
      at: message.at,
      process_steps: message.processSteps,
      agent_process_collapsed: message.agentProcessCollapsed,
      task_id: message.taskID,
      task_status: message.taskStatus,
      task_pending: message.taskPending,
      task_result_delivered: message.taskResultDelivered,
      task_result_for: message.taskResultFor,
    })),
  }));
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

function buildSessionPaneSnapshot(
  route: string,
  language: LegacyShellLanguage,
  sessions: ChatSession[],
  activeSessionID: string,
): LegacyShellSessionPaneDetail {
  return {
    route,
    hasSessions: sessions.length > 0,
    loadError: "",
    items: sessions.map((session) => ({
      id: session.id,
      title: session.title,
      meta: buildSessionMeta(session, language),
      active: session.id === activeSessionID,
      shortHash: hashShort(session.id),
      copyValue: session.id,
      copyLabel: language === "zh" ? "复制会话 ID" : "Copy session id",
      deleteLabel: language === "zh" ? "删除会话" : "Delete session",
    })),
  };
}

function toMessageRegionSnapshot(route: string, session: ChatSession | null): LegacyShellMessageRegionDetail {
  return {
    route,
    hasMessages: Boolean(session?.messages.length),
    sessionId: session?.id || "",
    messages: (session?.messages || []).map((message) => ({
      id: message.id,
      role: message.role,
      text: message.text,
      route: message.route,
      source: message.source,
      error: message.error,
      status: message.status,
      at: message.at,
      process_steps: message.processSteps,
      agent_process_collapsed: message.agentProcessCollapsed,
    })),
  };
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

function isTerminalTaskStatus(status: string): boolean {
  return ["success", "failed", "canceled"].includes(normalizeTaskStatus(status));
}

function activeViewportInput(): HTMLInputElement | HTMLTextAreaElement | null {
  const active = document.activeElement;
  if (active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement) {
    return active;
  }
  return null;
}

function createRuntimeSnapshot(params: {
  route: string;
  session: ChatSession | null;
  selectedAgentID: string;
  openPopover: string;
  compact: boolean;
  providers: ChatProvider[];
  agents: ChatAgent[];
  skills: ChatCapability[];
  mcps: ChatCapability[];
}): LegacyShellChatRuntimeDetail {
  const { route, session, selectedAgentID, openPopover, compact, providers, agents, skills, mcps } = params;
  const selection = resolveModelSelection(session, providers);
  const selectedProvider = enabledProviders(providers).find((provider) => normalizeText(provider.id) === selection.providerID) || null;
  const selectedModel = enabledModels(selectedProvider).find((model) => normalizeText(model.id) === selection.modelID) || null;
  const agentTarget = normalizeChatTarget({
    type: "agent",
    id: selectedAgentID,
    name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
  });
  const currentTarget = session?.target || (route === "agent-runtime" ? agentTarget : defaultChatTarget());
  return {
    route,
    compact,
    openPopover,
    note: "",
    agentRuntime: route === "agent-runtime",
    locked: Boolean(session?.messages.length),
    target: currentTarget,
    targetOptions: route === "agent-runtime"
      ? agents
          .filter((agent) => normalizeText(agent.id))
          .map((agent) => ({
            type: "agent",
            id: normalizeText(agent.id),
            name: normalizeText(agent.name) || normalizeText(agent.id),
            subtitle: normalizeText(agent.description) || "Agent",
            active: normalizeText(agent.id) === currentTarget.id,
          }))
      : [],
    selectedProviderId: selection.providerID,
    selectedModelId: selection.modelID,
    selectedModelLabel: selectedModel?.name || selectedModel?.id || "Default",
    toolCount: (session?.toolIDs.length || 0) + (session?.mcpIDs.length || 0),
    skillCount: session?.skillIDs.length || 0,
    providers: enabledProviders(providers).map((provider) => ({
      id: normalizeText(provider.id),
      name: normalizeText(provider.name) || normalizeText(provider.id),
      models: enabledModels(provider).map((model) => ({
        id: normalizeText(model.id),
        name: normalizeText(model.name) || normalizeText(model.id),
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
        kind: "tool",
        active: Boolean(session?.toolIDs.includes("memory")),
      },
      ...mcps
        .filter((item) => item.enabled !== false)
        .map((item) => ({
          id: normalizeText(item.id),
          name: normalizeText(item.name) || normalizeText(item.id),
          description: normalizeText(item.description) || normalizeText(item.scope) || "MCP",
          kind: "mcp" as const,
          active: Boolean(session?.mcpIDs.includes(normalizeText(item.id))),
        }))
        .filter((item) => item.id),
    ],
    skills: skills
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        id: normalizeText(item.id),
        name: normalizeText(item.name) || normalizeText(item.id),
        description: normalizeText(item.description) || normalizeText(item.scope) || "Skill",
        kind: "skill" as const,
        active: Boolean(session?.skillIDs.includes(normalizeText(item.id))),
      }))
      .filter((item) => item.id),
  };
}

async function readResponsePayload(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) {
    return {};
  }
  try {
    return JSON.parse(text);
  } catch {
    return {};
  }
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

function updateDocumentLanguage(language: LegacyShellLanguage) {
  document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
}

export function ReactRuntimeFacade() {
  const apiClient = useMemo(() => createAPIClient(), []);
  const [currentRoute, setCurrentRoute] = useState(() => parseLegacyShellHashRoute());
  const [language, setLanguage] = useState<LegacyShellLanguage>(() =>
    normalizeLegacyShellLanguage(document.documentElement.lang),
  );
  const [sessionsByRoute, setSessionsByRoute] = useState<SessionsState>(() =>
    splitSessionsByRoute(loadStoredSessions()),
  );
  const [activeSessionByRoute, setActiveSessionByRoute] = useState<ActiveSessionState>(() =>
    loadActiveSessionState(),
  );
  const [selectedAgentID, setSelectedAgentID] = useState("");
  const [compactRuntime, setCompactRuntime] = useState(() => isLegacyShellMobileViewport());
  const [runtimeOpenPopover, setRuntimeOpenPopover] = useState("");
  const [providers, setProviders] = useState<ChatProvider[]>([]);
  const [skills, setSkills] = useState<ChatCapability[]>([]);
  const [mcps, setMcps] = useState<ChatCapability[]>([]);
  const [agents, setAgents] = useState<ChatAgent[]>([]);
  const [composerDrafts, setComposerDrafts] = useState<ComposerDraftMap>(() => loadComposerDrafts());
  const [pendingTasksVersion, setPendingTasksVersion] = useState(0);
  const pollTimerRef = useRef<number>(0);

  const activeChatRoute = routeConversationMode(currentRoute);
  const activeSessions = sessionsByRoute[activeChatRoute];
  const activeSession = activeSessions.find((session) => session.id === activeSessionByRoute[activeChatRoute]) || null;

  const persistSessionsState = (nextSessionsByRoute: SessionsState, nextActiveState: ActiveSessionState) => {
    writeJSONStorage(SESSION_STORAGE_KEY, toPersistedSessions(nextSessionsByRoute));
    writeJSONStorage(ACTIVE_SESSION_STORAGE_KEY, nextActiveState);
  };

  const updateComposerDOM = (drafts: ComposerDraftMap, route: ChatRoute, activeState: ActiveSessionState) => {
    const inputNode = document.getElementById("composerInput");
    const counterNode = document.getElementById("charCount");
    const sessionID = activeState[route];
    const value = sessionID ? drafts[sessionID] || "" : "";
    if (inputNode instanceof HTMLTextAreaElement && inputNode.value !== value) {
      inputNode.value = value;
    }
    if (counterNode) {
      counterNode.textContent = `${value.length}/${MAX_COMPOSER_CHARS}`;
    }
  };

  const ensureSession = (route: ChatRoute, target?: Partial<ChatTarget> | null) => {
    const targetValue = normalizeChatTarget(
      target || (route === "agent-runtime"
        ? {
            type: "agent",
            id: selectedAgentID,
            name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
          }
        : defaultChatTarget()),
    );
    const existing = sessionsByRoute[route].find((session) => session.id === activeSessionByRoute[route]) || null;
    if (existing) {
      return existing;
    }
    const created: ChatSession = {
      id: makeID("session"),
      title: route === "agent-runtime" ? "New Agent Session" : "New Chat",
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
    };
    const nextSessionsByRoute: SessionsState = {
      ...sessionsByRoute,
      [route]: [created, ...sessionsByRoute[route]],
    };
    const nextActiveState = { ...activeSessionByRoute, [route]: created.id };
    setSessionsByRoute(nextSessionsByRoute);
    setActiveSessionByRoute(nextActiveState);
    persistSessionsState(nextSessionsByRoute, nextActiveState);
    updateComposerDOM(composerDrafts, route, nextActiveState);
    return created;
  };

  const patchSession = (
    route: ChatRoute,
    sessionID: string,
    updater: (session: ChatSession) => ChatSession,
  ) => {
    setSessionsByRoute((current) => {
      const nextSessions = current[route].map((session) =>
        session.id === sessionID ? updater(session) : session,
      );
      const nextState = { ...current, [route]: nextSessions };
      persistSessionsState(nextState, activeSessionByRoute);
      return nextState;
    });
  };

  const createMessage = (
    role: "user" | "assistant",
    text: string,
    patch: Partial<ChatMessage> = {},
  ): ChatMessage => ({
    id: makeID("msg"),
    role,
    text,
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

  const appendMessage = (route: ChatRoute, sessionID: string, message: ChatMessage) => {
    patchSession(route, sessionID, (session) => ({
      ...session,
      title: session.titleAuto && message.role === "user"
        ? (message.text.slice(0, 32) || session.title)
        : session.title,
      titleAuto: session.titleAuto && message.role !== "user",
      messages: [...session.messages, message],
    }));
  };

  const setAssistantMessage = (
    route: ChatRoute,
    sessionID: string,
    messageID: string,
    patch: Partial<ChatMessage>,
  ) => {
    patchSession(route, sessionID, (session) => ({
      ...session,
      messages: session.messages.map((message) =>
        message.id === messageID ? { ...message, ...patch } : message,
      ),
    }));
  };

  const focusSession = (sessionID: string) => {
    const route = sessionsByRoute["agent-runtime"].some((session) => session.id === sessionID)
      ? "agent-runtime"
      : "chat";
    const nextActiveState = { ...activeSessionByRoute, [route]: sessionID };
    setActiveSessionByRoute(nextActiveState);
    persistSessionsState(sessionsByRoute, nextActiveState);
    updateComposerDOM(composerDrafts, route, nextActiveState);
    return true;
  };

  const removeSession = async (sessionID: string) => {
    const route = sessionsByRoute["agent-runtime"].some((session) => session.id === sessionID)
      ? "agent-runtime"
      : "chat";
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
    delete nextDrafts[sessionID];
    setSessionsByRoute(nextSessionsByRoute);
    setActiveSessionByRoute(nextActiveState);
    setComposerDrafts(nextDrafts);
    persistComposerDrafts(nextDrafts);
    persistSessionsState(nextSessionsByRoute, nextActiveState);
    updateComposerDOM(nextDrafts, route, nextActiveState);
    return true;
  };

  const sendMessageFallback = async (
    route: ChatRoute,
    sessionID: string,
    target: ChatTarget,
    assistantMessageID: string,
    content: string,
  ) => {
    const session = sessionsByRoute[route].find((item) => item.id === sessionID) || null;
    const selection = resolveModelSelection(session, providers);
    const body = await apiClient.post<{
      result?: {
        output?: string;
        route?: string;
        metadata?: Record<string, string>;
        process_steps?: LegacyShellMessageProcessStepDetail[];
      };
      task_id?: string;
      task_status?: string;
    }>(
      route === "agent-runtime" ? AGENT_FALLBACK_ENDPOINT : FALLBACK_ENDPOINT,
      {
        session_id: sessionID,
        channel_id: "web-default",
        content,
        metadata: {
          "alter0.llm.provider_id": selection.providerID,
          "alter0.llm.model": selection.modelID,
          "alter0.agent.tools": JSON.stringify(session?.toolIDs || []),
          "alter0.skills.include": JSON.stringify(session?.skillIDs || []),
          "alter0.mcp.request.enable": JSON.stringify(session?.mcpIDs || []),
        },
        ...(route === "agent-runtime" ? { agent_id: target.id } : {}),
      },
    );
    setAssistantMessage(route, sessionID, assistantMessageID, {
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
    route: ChatRoute,
    sessionID: string,
    target: ChatTarget,
    assistantMessageID: string,
    content: string,
  ): Promise<StreamResult> => {
    const session = sessionsByRoute[route].find((item) => item.id === sessionID) || null;
    const selection = resolveModelSelection(session, providers);
    let sawEvent = false;
    let sawDone = false;
    let output = "";
    let routeHint = "";
    const response = await fetch(route === "agent-runtime" ? AGENT_STREAM_ENDPOINT : STREAM_ENDPOINT, {
      method: "POST",
      headers: {
        Accept: "text/event-stream",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        session_id: sessionID,
        channel_id: "web-default",
        content,
        metadata: {
          "alter0.llm.provider_id": selection.providerID,
          "alter0.llm.model": selection.modelID,
          "alter0.agent.tools": JSON.stringify(session?.toolIDs || []),
          "alter0.skills.include": JSON.stringify(session?.skillIDs || []),
          "alter0.mcp.request.enable": JSON.stringify(session?.mcpIDs || []),
        },
        ...(route === "agent-runtime" ? { agent_id: target.id } : {}),
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
          if (parsed.event === "process") {
            patchSession(route, sessionID, (currentSession) => {
              const nextMessages = currentSession.messages.map((message) =>
                message.id === assistantMessageID
                  ? {
                      ...message,
                      processSteps: normalizeProcessSteps([
                        ...message.processSteps,
                        parsed.data.process_step as LegacyShellMessageProcessStepDetail,
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
              setAssistantMessage(route, sessionID, assistantMessageID, {
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
            setAssistantMessage(route, sessionID, assistantMessageID, {
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
            setAssistantMessage(route, sessionID, assistantMessageID, {
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

  const sendPrompt = async (prompt: string) => {
    const route = activeChatRoute;
    const content = prompt.trim();
    if (!content || (route === "agent-runtime" && !selectedAgentID)) {
      return;
    }
    const session = ensureSession(route, route === "agent-runtime"
      ? {
          type: "agent",
          id: selectedAgentID,
          name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
        }
      : defaultChatTarget());
    const userMessage = createMessage("user", content, { at: Date.now() });
    const assistantMessage = createMessage("assistant", "Thinking...", {
      status: "streaming",
      at: Date.now(),
    });
    appendMessage(route, session.id, userMessage);
    appendMessage(route, session.id, assistantMessage);
    const nextDrafts = { ...composerDrafts, [session.id]: "" };
    setComposerDrafts(nextDrafts);
    persistComposerDrafts(nextDrafts);
    updateComposerDOM(nextDrafts, route, { ...activeSessionByRoute, [route]: session.id });
    try {
      const streamResult = await sendMessageStream(route, session.id, session.target, assistantMessage.id, content);
      if (!streamResult.ok && streamResult.canFallback) {
        await sendMessageFallback(route, session.id, session.target, assistantMessage.id, content);
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

  useEffect(() => {
    ensureLegacyRuntimeSnapshotBridge();
  }, []);

  useEffect(() => {
    const syncViewport = () => setCompactRuntime(isLegacyShellMobileViewport());
    window.addEventListener("resize", syncViewport);
    return () => window.removeEventListener("resize", syncViewport);
  }, []);

  useEffect(() => {
    const syncRoute = () => setCurrentRoute(parseLegacyShellHashRoute());
    const observer = new MutationObserver(() => {
      setLanguage(normalizeLegacyShellLanguage(document.documentElement.lang));
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["lang"] });
    window.addEventListener("hashchange", syncRoute);
    return () => {
      observer.disconnect();
      window.removeEventListener("hashchange", syncRoute);
    };
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
          ? agentPayload.items.filter((agent) => agent.enabled !== false)
          : [];
        setAgents(nextAgents);
        if (!selectedAgentID) {
          setSelectedAgentID(normalizeText(nextAgents[0]?.id));
        }
      } catch {
      }
    };
    void loadCatalogs();
  }, [apiClient, selectedAgentID]);

  useEffect(() => {
    if (!isLegacyShellChatRoute(currentRoute)) {
      return;
    }
    ensureSession(activeChatRoute, activeChatRoute === "agent-runtime"
      ? {
          type: "agent",
          id: selectedAgentID,
          name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
        }
      : defaultChatTarget());
  }, [activeChatRoute, agents, currentRoute, selectedAgentID]);

  useEffect(() => {
    if (!isLegacyShellChatRoute(currentRoute)) {
      return;
    }
    const routeCopy = getLegacyRouteHeadingCopy(language, currentRoute);
    publishLegacyRuntimeSnapshot(
      "alter0:legacy-shell:sync-session-pane",
      buildSessionPaneSnapshot(currentRoute, language, activeSessions, activeSession?.id || ""),
    );
    publishLegacyRuntimeSnapshot(
      "alter0:legacy-shell:sync-message-region",
      toMessageRegionSnapshot(currentRoute, activeSession),
    );
    publishLegacyRuntimeSnapshot("alter0:legacy-shell:sync-chat-workspace", {
      route: currentRoute,
      heading: routeCopy.title,
      subheading: routeCopy.subtitle,
      welcomeHeading: language === "zh" ? "你好，今天想一起完成什么？" : "Hello, what should we build today?",
      welcomeDescription:
        language === "zh"
          ? "我会结合当前会话、目标和运行时配置继续推进。"
          : "I will keep moving with the current session, target, and runtime configuration.",
      welcomeTargets: currentRoute === "agent-runtime"
        ? agents.map((agent) => ({
            type: "agent",
            id: normalizeText(agent.id),
            name: normalizeText(agent.name) || normalizeText(agent.id),
            active: normalizeText(agent.id) === normalizeText(activeSession?.target.id || selectedAgentID),
            interactive: true,
          }))
        : [],
      welcomeTargetError: "",
    });
    publishLegacyRuntimeSnapshot(
      "alter0:legacy-shell:sync-chat-runtime",
      createRuntimeSnapshot({
        route: currentRoute,
        session: activeSession,
        selectedAgentID,
        openPopover: runtimeOpenPopover,
        compact: compactRuntime,
        providers,
        agents,
        skills,
        mcps,
      }),
    );
    updateComposerDOM(composerDrafts, activeChatRoute, activeSessionByRoute);
  }, [
    activeChatRoute,
    activeSession,
    activeSessionByRoute,
    activeSessions,
    agents,
    composerDrafts,
    currentRoute,
    language,
    mcps,
    providers,
    runtimeOpenPopover,
    selectedAgentID,
    skills,
    compactRuntime,
  ]);

  useEffect(() => {
    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    if (!appShell) {
      return;
    }
    appShell.classList.toggle(
      "runtime-sheet-open",
      compactRuntime && runtimeOpenPopover === "mobile",
    );
    return () => {
      appShell.classList.remove("runtime-sheet-open");
    };
  }, [compactRuntime, runtimeOpenPopover]);

  useEffect(() => {
    const runtime = window.__alter0LegacyRuntime || {};
    window.__alter0LegacyRuntime = Object.assign(runtime, {
      createSession: () => {
        if (currentRoute === "terminal") {
          const button = document.querySelector("[data-terminal-create]");
          if (button instanceof HTMLElement) {
            button.click();
            return true;
          }
          return false;
        }
        ensureSession(activeChatRoute, activeChatRoute === "agent-runtime"
          ? {
              type: "agent",
              id: selectedAgentID,
              name: agents.find((agent) => normalizeText(agent.id) === selectedAgentID)?.name || selectedAgentID,
            }
          : defaultChatTarget());
        return true;
      },
      focusSession: (sessionID: string) => focusSession(sessionID),
      removeSession: (sessionID: string) => {
        void removeSession(sessionID);
        return true;
      },
      toggleChatRuntimePopover: (popover: string) => {
        setRuntimeOpenPopover((current) => {
          const nextPopover = current === popover ? "" : popover;
          if (
            compactRuntime
            && nextPopover === "mobile"
            && current !== nextPopover
          ) {
            const activeInput = activeViewportInput();
            window.requestAnimationFrame(() => {
              activeInput?.blur();
            });
          }
          return nextPopover;
        });
        return true;
      },
      closeChatRuntimePopover: () => {
        setRuntimeOpenPopover("");
        return true;
      },
      selectChatRuntimeTarget: (target: LegacyShellChatRuntimeTargetActionDetail) => {
        if (activeChatRoute !== "agent-runtime") {
          return false;
        }
        const normalizedTarget = normalizeChatTarget(target);
        setSelectedAgentID(normalizedTarget.id);
        patchSession(activeChatRoute, activeSessionByRoute[activeChatRoute], (session) =>
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
        setRuntimeOpenPopover("");
        return true;
      },
      selectChatRuntimeModel: (selection: LegacyShellChatRuntimeModelActionDetail) => {
        if (!activeSession) {
          return false;
        }
        patchSession(activeChatRoute, activeSession.id, (session) => ({
          ...session,
          modelProviderID: normalizeText(selection.providerId),
          modelID: normalizeText(selection.modelId),
        }));
        setRuntimeOpenPopover("");
        return true;
      },
      toggleChatRuntimeItem: (selection: LegacyShellChatRuntimeItemActionDetail) => {
        if (!activeSession) {
          return false;
        }
        const value = normalizeText(selection.id);
        if (!value) {
          return false;
        }
        const mutate = (items: string[]) =>
          selection.checked
            ? normalizeSelectionIDs([...items, value])
            : items.filter((item) => item !== value);
        patchSession(activeChatRoute, activeSession.id, (session) => {
          if (selection.group === "skills") {
            return { ...session, skillIDs: mutate(session.skillIDs) };
          }
          if (selection.kind === "tool") {
            return { ...session, toolIDs: mutate(session.toolIDs) };
          }
          return { ...session, mcpIDs: mutate(session.mcpIDs) };
        });
        return true;
      },
    });
  }, [activeChatRoute, activeSession, activeSessionByRoute, agents, compactRuntime, currentRoute, selectedAgentID]);

  useEffect(() => {
    const controller = createMobileViewportSyncController({
      hasActiveInput: () => Boolean(activeViewportInput()),
    });
    return () => controller.destroy();
  }, []);

  useEffect(() => {
    const handleNavigate = (event: Event) => {
      const route = normalizeText((event as CustomEvent<{ route?: string }>).detail?.route) || LEGACY_SHELL_DEFAULT_ROUTE;
      window.location.hash = `#${route}`;
    };
    const handleLanguageToggle = () => {
      updateDocumentLanguage(language === "zh" ? "en" : "zh");
    };
    const handleQuickPrompt = (event: Event) => {
      const prompt = normalizeText((event as CustomEvent<{ prompt?: string }>).detail?.prompt);
      if (prompt) {
        void sendPrompt(prompt);
      }
    };
    const handleCreateSession = () => {
      window.__alter0LegacyRuntime?.createSession?.();
    };
    const handleFocusSession = (event: Event) => {
      const sessionID = normalizeText((event as CustomEvent<{ sessionId?: string }>).detail?.sessionId);
      if (sessionID) {
        window.__alter0LegacyRuntime?.focusSession?.(sessionID);
      }
    };
    const handleRemoveSession = (event: Event) => {
      const sessionID = normalizeText((event as CustomEvent<{ sessionId?: string }>).detail?.sessionId);
      if (sessionID) {
        window.__alter0LegacyRuntime?.removeSession?.(sessionID);
      }
    };
    document.addEventListener(LEGACY_SHELL_NAVIGATE_EVENT, handleNavigate);
    document.addEventListener(LEGACY_SHELL_TOGGLE_LANGUAGE_EVENT, handleLanguageToggle);
    document.addEventListener(LEGACY_SHELL_QUICK_PROMPT_EVENT, handleQuickPrompt);
    document.addEventListener(LEGACY_SHELL_CREATE_SESSION_EVENT, handleCreateSession);
    document.addEventListener(LEGACY_SHELL_FOCUS_SESSION_EVENT, handleFocusSession);
    document.addEventListener(LEGACY_SHELL_REMOVE_SESSION_EVENT, handleRemoveSession);
    return () => {
      document.removeEventListener(LEGACY_SHELL_NAVIGATE_EVENT, handleNavigate);
      document.removeEventListener(LEGACY_SHELL_TOGGLE_LANGUAGE_EVENT, handleLanguageToggle);
      document.removeEventListener(LEGACY_SHELL_QUICK_PROMPT_EVENT, handleQuickPrompt);
      document.removeEventListener(LEGACY_SHELL_CREATE_SESSION_EVENT, handleCreateSession);
      document.removeEventListener(LEGACY_SHELL_FOCUS_SESSION_EVENT, handleFocusSession);
      document.removeEventListener(LEGACY_SHELL_REMOVE_SESSION_EVENT, handleRemoveSession);
    };
  }, [language]);

  useEffect(() => {
    const inputNode = document.getElementById("composerInput");
    const formNode = document.getElementById("chatForm");
    const counterNode = document.getElementById("charCount");
    if (!(inputNode instanceof HTMLTextAreaElement) || !(formNode instanceof HTMLFormElement)) {
      return;
    }
    const handleInput = () => {
      const value = inputNode.value.slice(0, MAX_COMPOSER_CHARS);
      if (value !== inputNode.value) {
        inputNode.value = value;
      }
      if (counterNode) {
        counterNode.textContent = `${value.length}/${MAX_COMPOSER_CHARS}`;
      }
      const sessionID = activeSessionByRoute[activeChatRoute];
      if (!sessionID) {
        return;
      }
      const nextDrafts = { ...composerDrafts, [sessionID]: value };
      setComposerDrafts(nextDrafts);
      persistComposerDrafts(nextDrafts);
    };
    const handleFocus = () => {
      if (compactRuntime && runtimeOpenPopover) {
        setRuntimeOpenPopover("");
      }
    };
    const handleSubmit = (event: Event) => {
      event.preventDefault();
      void sendPrompt(inputNode.value);
    };
    inputNode.addEventListener("input", handleInput);
    inputNode.addEventListener("focus", handleFocus);
    formNode.addEventListener("submit", handleSubmit);
    updateComposerDOM(composerDrafts, activeChatRoute, activeSessionByRoute);
    return () => {
      inputNode.removeEventListener("input", handleInput);
      inputNode.removeEventListener("focus", handleFocus);
      formNode.removeEventListener("submit", handleSubmit);
    };
  }, [activeChatRoute, activeSessionByRoute, composerDrafts, compactRuntime, runtimeOpenPopover]);

  useEffect(() => {
    window.clearTimeout(pollTimerRef.current);
    const pending = Object.entries(sessionsByRoute).flatMap(([route, sessions]) =>
      sessions.flatMap((session) =>
        session.messages
          .filter((message) => message.taskID && message.taskPending && !message.taskResultDelivered)
          .map((message) => ({
            route: route as ChatRoute,
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

  return null;
}
