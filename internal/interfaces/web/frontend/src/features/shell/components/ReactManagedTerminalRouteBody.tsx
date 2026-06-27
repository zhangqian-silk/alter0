import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ClipboardEvent, type PointerEvent, type TouchEvent } from "react";
import { useWorkbenchContext } from "../../../app/WorkbenchContext";
import { readWorkbenchRouteSessionID, writeWorkbenchRouteSessionID } from "../../../app/routeState";
import { createAPIClient } from "../../../shared/api/client";
import { hashSessionIDShort, resolveSessionIDReference } from "../../../shared/session/sessionHash";
import { groupSessionListItems } from "../../../shared/time/sessionListGroups";
import { formatDateTime, formatDateTimeMinute } from "../../../shared/time/format";
import { usePageActivation } from "../../../shared/visibility/usePageActivation";
import {
  canPreviewComposerAttachment,
  getPastedComposerImageFiles,
  readComposerFiles,
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  resolveComposerAttachmentPreviewURL,
  type ComposerAttachment,
} from "../../conversation-runtime/composerImageAttachments";
import { getLegacyShellCopy } from "../legacyShellCopy";
import {
  buildDraftWithCodexSlashCommand,
  CODEX_SLASH_COMMANDS,
  codexSlashCommandQuery,
  isCodexShellSession,
} from "./codexSlashCommands";
import { RuntimeWorkspacePage, type RuntimeWorkspacePageController } from "./RuntimeWorkspacePage";
import type { RuntimeTimelineItem, RuntimeTimelineProcessEvent } from "./RuntimeTimeline";
import { normalizeText, RouteMarkdownContent } from "./RouteBodyPrimitives";
import {
  RuntimeProcessDetailBlocks,
  runtimeTraceEventToProcessDetailBlocks,
} from "./RuntimeProcessDetailBlocks";
import { RuntimeProcessStepMeta } from "./RuntimeProcessStepMeta";
import { ScrollJumpStrip } from "./ScrollJumpStrip";
import { useRuntimeComposerViewportSync } from "./useRuntimeComposerViewportSync";
import {
  normalizeRuntimeTraceEvents,
  runtimeTraceEventDisclosureCategory,
  runtimeTraceEventDetailID,
  type RuntimeBlock,
  type RuntimeTraceEvent,
} from "./runtimeTraceEvents";

type TerminalStatus = "ready" | "busy" | "exited" | "failed" | "interrupted";

type TerminalTurn = {
  id: string;
  prompt: string;
  attachments?: TerminalAttachment[];
  status: string;
  started_at?: string | number;
  finished_at?: string | number;
  duration_ms?: number;
  final_output?: string;
  runtime_trace_events?: RuntimeTraceEvent[];
};

type TerminalTurnPaging = {
  limit?: number;
  total?: number;
  has_more_before?: boolean;
  has_more_after?: boolean;
  oldest_turn_id?: string;
  newest_turn_id?: string;
  next_before_turn_id?: string;
};

type TerminalAttachment = {
  id?: string;
  name: string;
  content_type: string;
  data_url?: string;
  asset_url?: string;
  preview_url?: string;
};

type TerminalSession = {
  id: string;
  terminal_session_id?: string;
  title?: string;
  shell?: string;
  working_dir?: string;
  status?: string;
  pinned?: boolean;
  created_at?: string | number;
  last_output_at?: string | number;
  updated_at?: string | number;
  error_message?: string;
  turns?: TerminalTurn[];
  turns_paging?: TerminalTurnPaging;
};

type TerminalSessionsResponse = {
  items?: TerminalSession[];
};

type TerminalSessionResponse = {
  session?: TerminalSession;
};

function RuntimeSessionControlIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <circle cx="6" cy="6" r="2.25" stroke="currentColor" strokeWidth="1.7" />
      <circle cx="14" cy="6" r="2.25" stroke="currentColor" strokeWidth="1.7" />
      <circle cx="6" cy="14" r="2.25" stroke="currentColor" strokeWidth="1.7" />
      <circle cx="14" cy="14" r="2.25" stroke="currentColor" strokeWidth="1.7" />
    </svg>
  );
}

type RuntimeTraceEventDetail = {
  turn_id?: string;
  event?: RuntimeTraceEvent;
  blocks?: RuntimeBlock[];
  searchable?: boolean;
};

type RuntimeTraceEventDetailResponse = {
  event?: RuntimeTraceEventDetail;
};

type TerminalSkill = {
  id?: string;
  name?: string;
  enabled?: boolean;
  scope?: string;
  metadata?: Record<string, string>;
};

type TerminalSkillSelection = {
  id: string;
  name: string;
  description: string;
  active: boolean;
};

type TerminalCopy = {
  sessions: string;
  sessionCount: (count: number) => string;
  current: string;
  sessionLabel: string;
  newShort: string;
  hideSessions: string;
  empty: string;
  ready: string;
  busy: string;
  exited: string;
  failed: string;
  interrupted: string;
  pin: string;
  unpin: string;
  pinSession: string;
  unpinSession: string;
  sessionActions: string;
  delete: string;
  deleteConfirm: string;
  inputPlaceholder: string;
  send: string;
  sending: string;
  addAttachment: string;
  addAttachmentAccept: string;
  fileAttachmentLabel: string;
  closePreview: string;
  shell: string;
  path: string;
  session: string;
  status: string;
  details: string;
  process: string;
  processEvents: (count: number) => string;
  noProcess: string;
  noOutput: string;
  noOutputMeta: string;
  top: string;
  prev: string;
  next: string;
  bottom: string;
  sessionRuntime: string;
  updatedAt: string;
  terminalNoteExited: string;
  terminalNoteInterrupted: string;
  terminalNoteFailed: string;
  preview: string;
  loading: string;
  noSession: string;
  metadata: string;
  skills: string;
  activeSkills: string;
  noSkills: string;
  copy: string;
};

const TERMINAL_COPY: Record<"en" | "zh", TerminalCopy> = {
  en: {
    sessions: "Sessions",
    sessionCount: (count) => `${count} sessions`,
    current: "Current",
    sessionLabel: "Session",
    newShort: "New",
    hideSessions: "Hide",
    empty: "No terminal sessions yet.",
    ready: "Ready",
    busy: "Busy",
    exited: "Exited",
    failed: "Failed",
    interrupted: "Interrupted",
    pin: "Pin",
    unpin: "Unpin",
    pinSession: "Pin session",
    unpinSession: "Unpin session",
    sessionActions: "Session actions",
    delete: "Delete",
    deleteConfirm: "Delete this terminal session?",
    inputPlaceholder: "Type command or prompt...",
    send: "Send",
    sending: "Sending...",
    addAttachment: "Add attachment",
    addAttachmentAccept: "image/*,.txt,.md,.json,.yaml,.yml,.csv,.log,.pdf",
    fileAttachmentLabel: "File",
    closePreview: "Close preview",
    shell: "Shell",
    path: "Path",
    session: "Session",
    status: "Status",
    details: "Details",
    process: "Thinking",
    processEvents: (count) => `${count} steps`,
    noProcess: "No execution details.",
    noOutput: "No output yet.",
    noOutputMeta: "No output yet.",
    top: "Top",
    prev: "Prev",
    next: "Next",
    bottom: "Bottom",
    sessionRuntime: "Runtime",
    updatedAt: "Updated",
    terminalNoteExited: "Codex session exited. Send a new input to continue in this session.",
    terminalNoteInterrupted: "Codex session interrupted. Send a new input to restart this session runtime.",
    terminalNoteFailed: "The last runtime failed. Send a new input to continue in this session.",
    preview: "Preview",
    loading: "Loading...",
    noSession: "Create a terminal session to begin.",
    metadata: "Metadata",
    skills: "Skills",
    activeSkills: "Active skills",
    noSkills: "No skills selected",
    copy: "Copy output",
  },
  zh: {
    sessions: "会话列表",
    sessionCount: (count) => `${count} 个会话`,
    current: "当前",
    sessionLabel: "会话",
    newShort: "新建",
    hideSessions: "收起",
    empty: "暂时还没有终端会话。",
    ready: "就绪",
    busy: "执行中",
    exited: "已退出",
    failed: "失败",
    interrupted: "已中断",
    pin: "置顶",
    unpin: "取消置顶",
    pinSession: "置顶会话",
    unpinSession: "取消置顶会话",
    sessionActions: "会话操作",
    delete: "删除",
    deleteConfirm: "确认删除这个终端会话？",
    inputPlaceholder: "输入命令或继续追问...",
    send: "发送",
    sending: "发送中...",
    addAttachment: "添加附件",
    addAttachmentAccept: "image/*,.txt,.md,.json,.yaml,.yml,.csv,.log,.pdf",
    fileAttachmentLabel: "文件",
    closePreview: "关闭预览",
    shell: "Shell",
    path: "路径",
    session: "会话",
    status: "状态",
    details: "详情",
    process: "已思考",
    processEvents: (count) => `${count} 步`,
    noProcess: "暂无执行细节。",
    noOutput: "暂时还没有输出。",
    noOutputMeta: "暂无输出。",
    top: "顶部",
    prev: "上一个",
    next: "下一个",
    bottom: "底部",
    sessionRuntime: "运行态",
    updatedAt: "更新时间",
    terminalNoteExited: "Codex 会话已退出。继续发送输入即可在当前会话中恢复执行。",
    terminalNoteInterrupted: "Codex 会话已中断。继续发送输入即可在当前会话中重新启动运行态。",
    terminalNoteFailed: "上一次运行失败。继续发送输入即可在当前会话中继续。",
    preview: "预览",
    loading: "加载中...",
    noSession: "先创建一个终端会话再开始。",
    metadata: "元数据",
    skills: "技能",
    activeSkills: "已启用技能",
    noSkills: "未选择技能",
    copy: "复制输出",
  },
};

const POLL_INTERVAL_MS = 2000;
const INTERACTION_POLL_INTERVAL_MS = 6000;
const HIDDEN_POLL_INTERVAL_MS = 12000;
const SCROLL_IDLE_MS = 1200;
const SCROLL_BOTTOM_ANCHOR_THRESHOLD = 24;
const PAGE_ACTIVE_REFRESH_DEBOUNCE_MS = 400;
const TERMINAL_NEW_SESSION_PLACEHOLDER_ID = "terminal-new-placeholder";
const TERMINAL_ATTACHMENT_DRAFT_STORAGE_KEY = "alter0.web.terminal.attachments.v1";
const TERMINAL_PENDING_DRAFT_KEY = "__pending__";
const TERMINAL_HISTORY_PAGE_TURN_LIMIT = 20;
export const TERMINAL_RUNTIME_CACHE_SESSION_TTL_MS = 24 * 60 * 60 * 1000;

type TerminalPollPlan = {
  enabled: boolean;
  interval: number;
  refreshActiveSession: boolean;
};

type TerminalRuntimeCacheSnapshot = {
  cachedAt: number;
  activeSessionID: string;
  sessions: TerminalSession[];
};

type TerminalRuntimeInitialState = {
  sessions: TerminalSession[];
  activeSessionID: string;
  hydratedFromCache: boolean;
};

type TerminalSessionRefreshOptions = {
  turnBefore?: string;
  turnLimit?: number;
};

let terminalRuntimeCache: TerminalRuntimeCacheSnapshot | null = null;

export function resetTerminalRuntimeCache() {
  terminalRuntimeCache = null;
}

function resolveLanguage(): "en" | "zh" {
  return document.documentElement.lang.toLowerCase().startsWith("zh") ? "zh" : "en";
}

function normalizeTerminalDraftAttachments(value: unknown): ComposerAttachment[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.flatMap((item) => {
    if (!item || typeof item !== "object") {
      return [];
    }
    const record = item as Record<string, unknown>;
    const id = typeof record.id === "string" ? record.id.trim() : "";
    const name = typeof record.name === "string" ? record.name.trim() : "";
    const kind = record.kind === "file" ? "file" : "image";
    const contentType = typeof record.contentType === "string" ? record.contentType.trim() : "";
    const dataURL = typeof record.dataURL === "string" ? record.dataURL.trim() : "";
    const previewDataURL = typeof record.previewDataURL === "string" ? record.previewDataURL.trim() : "";
    const assetURL = typeof record.assetURL === "string" ? record.assetURL.trim() : "";
    const previewURL = typeof record.previewURL === "string" ? record.previewURL.trim() : "";
    const size = typeof record.size === "number" && Number.isFinite(record.size) ? record.size : 0;
    if (!id || !contentType || (!dataURL && !assetURL && !previewURL)) {
      return [];
    }
    return [{
      id,
      kind,
      name,
      contentType,
      dataURL: dataURL || undefined,
      previewDataURL: previewDataURL || undefined,
      assetURL: assetURL || undefined,
      previewURL: previewURL || undefined,
      size,
    }];
  });
}

function normalizeAttachmentText(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeSelectionIDs(values: unknown): string[] {
  if (!Array.isArray(values)) {
    return [];
  }
  return Array.from(new Set(values.map((item) => normalizeAttachmentText(item)).filter(Boolean)));
}

function isPublicTerminalSkill(skill: TerminalSkill): boolean {
  const metadata = skill.metadata || {};
  const visibility = normalizeAttachmentText(metadata["alter0.skill.visibility"] || metadata["skill.visibility"]).toLowerCase();
  return visibility !== "private" && visibility !== "private";
}

function normalizeTerminalSkills(values: unknown): TerminalSkillSelection[] {
  if (!Array.isArray(values)) {
    return [];
  }
  const deduped = new Map<string, TerminalSkillSelection>();
  values.forEach((item) => {
    if (!item || typeof item !== "object") {
      return;
    }
    const skill = item as TerminalSkill;
    if (skill.enabled === false || !isPublicTerminalSkill(skill)) {
      return;
    }
    const id = normalizeAttachmentText(skill.id);
    if (!id || deduped.has(id)) {
      return;
    }
    const metadata = skill.metadata || {};
    deduped.set(id, {
      id,
      name: normalizeAttachmentText(skill.name) || id,
      description:
        normalizeAttachmentText(metadata["skill.description"])
        || normalizeAttachmentText(skill.scope)
        || "Skill",
      active: false,
    });
  });
  return Array.from(deduped.values());
}

function resolveDefaultTerminalSkillIDs(skills: TerminalSkillSelection[]): string[] {
  return skills.map((skill) => skill.id);
}

function serializeTerminalComposerAttachment(attachment: ComposerAttachment) {
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
    preview_data_url: attachment.previewDataURL,
  };
}

async function uploadTerminalSessionAttachments(
  apiClient: ReturnType<typeof createAPIClient>,
  sessionID: string,
  attachments: ComposerAttachment[],
): Promise<ComposerAttachment[]> {
  const existing = attachments.filter((attachment) => attachment.assetURL);
  const pending = attachments.filter((attachment) => !attachment.assetURL && attachment.dataURL);
  if (pending.length === 0) {
    return existing;
  }
  const payload = await apiClient.post<{
    items?: Array<{
      id?: string;
      name?: string;
      content_type?: string;
      size?: number;
      asset_url?: string;
      preview_url?: string;
    }>;
  }>(
    `/api/sessions/${encodeURIComponent(sessionID)}/attachments`,
    {
      attachments: pending.map((attachment) => ({
        name: attachment.name,
        content_type: attachment.contentType,
        data_url: attachment.dataURL,
        preview_data_url: attachment.previewDataURL,
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
      const id = normalizeAttachmentText(item.id);
      const assetURL = normalizeAttachmentText(item.asset_url);
      const previewURL = normalizeAttachmentText(item.preview_url) || assetURL;
      if (!id || !assetURL) {
        throw new Error("Failed to store attachments.");
      }
      return {
        id,
        kind: fallback.kind,
        name: normalizeAttachmentText(item.name) || fallback.name,
        contentType: normalizeAttachmentText(item.content_type) || fallback.contentType,
        size: typeof item.size === "number" && Number.isFinite(item.size) ? item.size : fallback.size,
        assetURL,
        previewURL: fallback.kind === "image" ? previewURL : undefined,
      };
    }),
  ];
}

function loadTerminalAttachmentDrafts(): Record<string, ComposerAttachment[]> {
  if (typeof window === "undefined") {
    return {};
  }
  try {
    const raw = window.localStorage.getItem(TERMINAL_ATTACHMENT_DRAFT_STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    return Object.entries(parsed).reduce<Record<string, ComposerAttachment[]>>((acc, [key, value]) => {
      const normalized = normalizeTerminalDraftAttachments(value);
      if (normalized.length > 0) {
        acc[key] = normalized;
      }
      return acc;
    }, {});
  } catch {
    return {};
  }
}

function parseTimestamp(value: string | number | undefined): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim()) {
    const parsed = Date.parse(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function normalizeStatus(status: string): TerminalStatus {
  switch (normalizeText(status).toLowerCase()) {
    case "busy":
    case "starting":
      return "busy";
    case "exited":
      return "exited";
    case "failed":
    case "error":
      return "failed";
    case "interrupted":
      return "interrupted";
    default:
      return "ready";
  }
}

function renderStatus(status: string, copy: TerminalCopy): string {
  switch (normalizeStatus(status)) {
    case "busy":
      return copy.busy;
    case "exited":
      return copy.exited;
    case "failed":
      return copy.failed;
    case "interrupted":
      return copy.interrupted;
    default:
      return copy.ready;
  }
}

function isLiveStatus(status: string): boolean {
  const normalized = normalizeStatus(status);
  return normalized === "ready" || normalized === "busy";
}

export function resolveTerminalPollPlan(options: {
  status: string;
  pageHidden: boolean;
  scrollingActive: boolean;
  inputFocused: boolean;
}): TerminalPollPlan {
  const normalized = normalizeStatus(options.status);

  if (normalized === "busy") {
    if (options.scrollingActive) {
      return {
        enabled: false,
        interval: 0,
        refreshActiveSession: true,
      };
    }
    return {
      enabled: true,
      interval: options.pageHidden
        ? HIDDEN_POLL_INTERVAL_MS
        : options.inputFocused
          ? INTERACTION_POLL_INTERVAL_MS
          : POLL_INTERVAL_MS,
      refreshActiveSession: true,
    };
  }

  if (normalized === "ready") {
    return {
      enabled: false,
      interval: 0,
      refreshActiveSession: false,
    };
  }

  return {
    enabled: false,
    interval: 0,
    refreshActiveSession: false,
  };
}

function sortSessions(items: TerminalSession[]): TerminalSession[] {
  return [...items].sort((left, right) => {
    if (Boolean(left.pinned) !== Boolean(right.pinned)) {
      return left.pinned ? -1 : 1;
    }
    const leftAt = Math.max(
      parseTimestamp(left.last_output_at),
      parseTimestamp(left.updated_at),
      parseTimestamp(left.created_at),
    );
    const rightAt = Math.max(
      parseTimestamp(right.last_output_at),
      parseTimestamp(right.updated_at),
      parseTimestamp(right.created_at),
    );
    return rightAt - leftAt;
  });
}

function terminalTurnOrdinal(id: string): number {
  const match = normalizeText(id).match(/(\d+)$/);
  return match ? Number(match[1]) : Number.NaN;
}

function compareTerminalTurns(left: TerminalTurn, right: TerminalTurn): number {
  const leftAt = Math.max(parseTimestamp(left.started_at), parseTimestamp(left.finished_at));
  const rightAt = Math.max(parseTimestamp(right.started_at), parseTimestamp(right.finished_at));
  if (leftAt > 0 && rightAt > 0 && leftAt !== rightAt) {
    return leftAt - rightAt;
  }
  const leftOrdinal = terminalTurnOrdinal(left.id);
  const rightOrdinal = terminalTurnOrdinal(right.id);
  if (Number.isFinite(leftOrdinal) && Number.isFinite(rightOrdinal) && leftOrdinal !== rightOrdinal) {
    return leftOrdinal - rightOrdinal;
  }
  return normalizeText(left.id).localeCompare(normalizeText(right.id));
}

function cloneTerminalTurn(turn: TerminalTurn): TerminalTurn {
  return {
    ...turn,
    attachments: Array.isArray(turn.attachments)
      ? turn.attachments.map((attachment) => ({ ...attachment }))
      : undefined,
    runtime_trace_events: Array.isArray(turn.runtime_trace_events)
      ? turn.runtime_trace_events.map((event) => ({
          ...event,
          blocks: event.blocks.map((block) => ({ ...block })),
        }))
      : undefined,
  };
}

function cloneTerminalSession(session: TerminalSession): TerminalSession {
  return {
    ...session,
    turns: Array.isArray(session.turns) ? session.turns.map(cloneTerminalTurn) : undefined,
    turns_paging: session.turns_paging ? { ...session.turns_paging } : undefined,
  };
}

function trimTerminalSessionForRuntimeCache(session: TerminalSession): TerminalSession {
  return cloneTerminalSession(session);
}

function readTerminalRuntimeCache(): TerminalRuntimeCacheSnapshot | null {
  if (!terminalRuntimeCache) {
    return null;
  }
  if (Date.now() - terminalRuntimeCache.cachedAt > TERMINAL_RUNTIME_CACHE_SESSION_TTL_MS) {
    terminalRuntimeCache = null;
    return null;
  }
  return {
    cachedAt: terminalRuntimeCache.cachedAt,
    activeSessionID: terminalRuntimeCache.activeSessionID,
    sessions: terminalRuntimeCache.sessions.map(cloneTerminalSession),
  };
}

function writeTerminalRuntimeCache(sessions: TerminalSession[], activeSessionID: string) {
  terminalRuntimeCache = {
    cachedAt: Date.now(),
    activeSessionID,
    sessions: sortSessions(sessions).map(trimTerminalSessionForRuntimeCache),
  };
}

function resolveInitialTerminalRuntimeState(): TerminalRuntimeInitialState {
  const routeSessionID = readWorkbenchRouteSessionID("terminal");
  const cache = readTerminalRuntimeCache();
  const sessions = cache?.sessions || [];
  const activeSessionID = resolveSessionIDReference(sessions, routeSessionID)
    || resolveSessionIDReference(sessions, cache?.activeSessionID || "")
    || routeSessionID
    || sessions[0]?.id
    || "";
  return {
    sessions,
    activeSessionID,
    hydratedFromCache: sessions.length > 0,
  };
}

function mergeTerminalTurns(current: TerminalTurn[] | undefined, incoming: TerminalTurn[] | undefined): TerminalTurn[] | undefined {
  if (!Array.isArray(incoming)) {
    return current;
  }
  if (!Array.isArray(current) || current.length === 0) {
    return incoming;
  }
  const merged = new Map<string, TerminalTurn>();
  current.forEach((turn) => {
    if (normalizeText(turn.id)) {
      merged.set(turn.id, turn);
    }
  });
  incoming.forEach((turn) => {
    if (normalizeText(turn.id)) {
      merged.set(turn.id, turn);
    }
  });
  return Array.from(merged.values()).sort(compareTerminalTurns);
}

function oldestTerminalTurnID(turns: TerminalTurn[] | undefined): string {
  if (!Array.isArray(turns) || turns.length === 0) {
    return "";
  }
  return normalizeText([...turns].sort(compareTerminalTurns)[0]?.id);
}

function newestTerminalTurnID(turns: TerminalTurn[] | undefined): string {
  if (!Array.isArray(turns) || turns.length === 0) {
    return "";
  }
  return normalizeText([...turns].sort(compareTerminalTurns)[turns.length - 1]?.id);
}

function hasTerminalTurn(turns: TerminalTurn[] | undefined, turnID: string): boolean {
  const normalized = normalizeText(turnID);
  return Boolean(normalized && Array.isArray(turns) && turns.some((turn) => turn.id === normalized));
}

function mergeTerminalTurnPaging(
  current: TerminalTurnPaging | undefined,
  incoming: TerminalTurnPaging | undefined,
  turns: TerminalTurn[] | undefined,
): TerminalTurnPaging | undefined {
  if (!current && !incoming) {
    return undefined;
  }
  const next: TerminalTurnPaging = {
    ...(current || {}),
    ...(incoming || {}),
  };
  const oldestTurnID = oldestTerminalTurnID(turns);
  const newestTurnID = newestTerminalTurnID(turns);
  if (oldestTurnID) {
    next.oldest_turn_id = oldestTurnID;
  }
  if (newestTurnID) {
    next.newest_turn_id = newestTurnID;
  }
  if (incoming?.has_more_before === false) {
    next.has_more_before = false;
    delete next.next_before_turn_id;
    return next;
  }
  if (current?.has_more_before === false && incoming?.has_more_before === true) {
    const incomingBeforeTurnID = normalizeAttachmentText(incoming.next_before_turn_id || incoming.oldest_turn_id);
    if (!incomingBeforeTurnID || hasTerminalTurn(turns, incomingBeforeTurnID)) {
      next.has_more_before = false;
      delete next.next_before_turn_id;
      return next;
    }
  }
  if (next.has_more_before === true) {
    const beforeTurnID = normalizeAttachmentText(next.next_before_turn_id || next.oldest_turn_id || oldestTurnID);
    if (beforeTurnID) {
      next.next_before_turn_id = beforeTurnID;
    }
  } else {
    delete next.next_before_turn_id;
  }
  return next;
}

function mergeSessionSnapshot(
  current: TerminalSession | undefined,
  incoming: TerminalSession,
): TerminalSession {
  if (!current) {
    return incoming;
  }
  const merged = { ...current } as Record<string, unknown>;
  (Object.keys(incoming) as Array<keyof TerminalSession>).forEach((key) => {
    const value = incoming[key];
    if (typeof value !== "undefined") {
      merged[key as string] = value;
    }
  });
  if (Array.isArray(incoming.turns)) {
    merged.turns = mergeTerminalTurns(current.turns, incoming.turns);
  }
  merged.turns_paging = mergeTerminalTurnPaging(
    current.turns_paging,
    incoming.turns_paging,
    merged.turns as TerminalTurn[] | undefined,
  );
  return merged as TerminalSession;
}

function eventKey(turnID: string, eventID: string) {
  return `${turnID}:${eventID}`;
}

function runtimeNote(status: string, copy: TerminalCopy): string {
  switch (normalizeStatus(status)) {
    case "exited":
      return copy.terminalNoteExited;
    case "interrupted":
      return copy.terminalNoteInterrupted;
    case "failed":
      return copy.terminalNoteFailed;
    default:
      return "";
  }
}

function sessionStatusClassName(status: string) {
  switch (normalizeStatus(status)) {
    case "busy":
      return "status-pending";
    case "failed":
      return "status-failed";
    case "exited":
    case "interrupted":
      return "status-neutral";
    default:
      return "status-success";
  }
}

function sessionSignalTone(status: string): "ready" | "busy" | "failed" {
  switch (normalizeStatus(status)) {
    case "busy":
      return "busy";
    case "ready":
      return "ready";
    default:
      return "failed";
  }
}

function sessionLastOutputLabel(
  session: TerminalSession | null,
  copy: TerminalCopy,
): string {
  const outputAt = parseTimestamp(session?.last_output_at);
  const fallbackAt = Math.max(
    parseTimestamp(session?.updated_at),
    parseTimestamp(session?.created_at),
  );
  const labelAt = outputAt > 0 ? outputAt : fallbackAt;
  if (labelAt <= 0) {
    return copy.noOutputMeta;
  }
  return formatDateTimeMinute(labelAt);
}

export function useTerminalRuntimeController(): RuntimeWorkspacePageController {
  /* Source contract markers:
     workbench.toggleMobileNav();
     workbench.closeMobileNav();
  */
  const workbench = useWorkbenchContext();
  const apiClient = useMemo(() => createAPIClient(), []);
  const [language, setLanguage] = useState<"en" | "zh">(() => resolveLanguage());
  const copy = TERMINAL_COPY[language];
  const shellCopy = getLegacyShellCopy(workbench.language);
  const initialRuntimeStateRef = useRef<TerminalRuntimeInitialState | null>(null);
  if (!initialRuntimeStateRef.current) {
    initialRuntimeStateRef.current = resolveInitialTerminalRuntimeState();
  }
  const [sessions, setSessions] = useState<TerminalSession[]>(() => initialRuntimeStateRef.current?.sessions || []);
  const [activeSessionID, setActiveSessionID] = useState(() => initialRuntimeStateRef.current?.activeSessionID || "");
  const [metaOpen, setMetaOpen] = useState(false);
  const [sessionDetailsOpen, setSessionDetailsOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [deletingSessionID, setDeletingSessionID] = useState("");
  const [pinningSessionID, setPinningSessionID] = useState("");
  const [loading, setLoading] = useState(() => !initialRuntimeStateRef.current?.hydratedFromCache);
  const [loadError, setLoadError] = useState("");
  const [inputValue, setInputValue] = useState("");
  const [attachmentDrafts, setAttachmentDrafts] = useState<Record<string, ComposerAttachment[]>>(() => loadTerminalAttachmentDrafts());
  const [skills, setSkills] = useState<TerminalSkillSelection[]>([]);
  const [selectedSkillIDs, setSelectedSkillIDs] = useState<string[]>([]);
  const attachmentDraftsRef = useRef<Record<string, ComposerAttachment[]>>(attachmentDrafts);
  const attachmentUploadPromisesRef = useRef<Record<string, {
    pendingIDs: string[];
    promise: Promise<ComposerAttachment[]>;
  }>>({});
  const [composerAttachmentError, setComposerAttachmentError] = useState("");
  const [previewAttachment, setPreviewAttachment] = useState<ComposerAttachment | null>(null);
  const [scrollingActive, setScrollingActive] = useState(false);
  const [inputFocused, setInputFocused] = useState(false);
  const [pageHidden, setPageHidden] = useState(() => document.hidden);
  const [expandedTurns, setExpandedTurns] = useState<Record<string, boolean>>({});
  const [expandedEvents, setExpandedEvents] = useState<Record<string, boolean>>({});
  const [eventDetails, setEventDetails] = useState<Record<string, RuntimeTraceEventDetail>>({});
  const [eventErrors, setEventErrors] = useState<Record<string, string>>({});
  const chatScreenRef = useRef<HTMLDivElement | null>(null);
  const composerInputRef = useRef<HTMLTextAreaElement | null>(null);
  const composerFileInputRef = useRef<HTMLInputElement | null>(null);
  const composerShellRef = useRef<HTMLElement | null>(null);
  const workspaceBodyRef = useRef<HTMLDivElement | null>(null);
  const activeTimelineSessionRef = useRef("");
  const timelineBottomPinnedSessionRef = useRef("");
  const scrollIdleTimerRef = useRef<number | null>(null);
  const scrollRestoreSnapshotRef = useRef<{
    top: number;
    anchoredToBottom: boolean;
  } | null>(null);
  const draftPersistTimerRef = useRef<number | null>(null);
  const deletedSessionIDsRef = useRef<Set<string>>(new Set());
  const progressiveHistoryLoadsRef = useRef<Set<string>>(new Set());
  const restoreMobileSessionPaneRef = useRef(false);
  const mobileSubmitGestureLockRef = useRef(false);
  const mobileSessionGestureLockRef = useRef(false);
  const groupedSessions = useMemo(
    () => groupSessionListItems(sessions, {
      language,
      getTimestamp: (session) =>
        parseTimestamp(session.updated_at) || parseTimestamp(session.last_output_at) || parseTimestamp(session.created_at),
      getPinned: (session) => Boolean(session.pinned),
    }),
    [language, sessions],
  );

  const activeSession = sessions.find((session) => session.id === activeSessionID) || null;
  const activeSessionResolvedID = activeSession?.id || "";
  const turns = Array.isArray(activeSession?.turns) ? activeSession.turns : [];
  const activeDraftKey = activeSessionID || TERMINAL_PENDING_DRAFT_KEY;
  const draftAttachments = attachmentDrafts[activeDraftKey] || [];
  const activeStatus = normalizeStatus(activeSession?.status || "");
  const selectedSkillSet = useMemo(() => new Set(selectedSkillIDs), [selectedSkillIDs]);
  const skillOptions = useMemo(
    () => skills.map((skill) => ({ ...skill, active: selectedSkillSet.has(skill.id) })),
    [selectedSkillSet, skills],
  );
  const pollPlan = resolveTerminalPollPlan({
    status: activeSession?.status || "",
    pageHidden,
    scrollingActive,
    inputFocused,
  });

  const focusComposerInputWithoutScroll = () => {
    const node = composerInputRef.current;
    if (!node) {
      return;
    }
    try {
      node.focus({ preventScroll: true });
    } catch {
      node.focus();
    }
  };

  const blurComposerInput = () => {
    const node = composerInputRef.current;
    if (!node || document.activeElement !== node) {
      return;
    }
    node.blur();
  };

  const updateDraftAttachments = (
    key: string,
    updater: (current: ComposerAttachment[]) => ComposerAttachment[],
  ) => {
    setAttachmentDrafts((current) => {
      const next = { ...current };
      const resolved = updater(current[key] || []);
      if (resolved.length > 0) {
        next[key] = resolved;
      } else {
        delete next[key];
      }
      attachmentDraftsRef.current = next;
      return next;
    });
  };

  const clearDraftAttachments = (key: string) => {
    setAttachmentDrafts((current) => {
      if (!current[key]) {
        return current;
      }
      const next = { ...current };
      delete next[key];
      attachmentDraftsRef.current = next;
      return next;
    });
  };

  const toggleSkill = (id: string, checked: boolean) => {
    const value = normalizeAttachmentText(id);
    if (!value) {
      return;
    }
    setSelectedSkillIDs((current) =>
      checked
        ? normalizeSelectionIDs([...current, value])
        : current.filter((item) => item !== value),
    );
  };

  const releaseMobileSubmitGestureLock = () => {
    window.setTimeout(() => {
      mobileSubmitGestureLockRef.current = false;
    }, 0);
  };

  const submitMobileInputOnPress = () => {
    mobileSubmitGestureLockRef.current = true;
    releaseMobileSubmitGestureLock();
    void submitInput();
  };

  const releaseMobileSessionGestureLock = () => {
    window.setTimeout(() => {
      mobileSessionGestureLockRef.current = false;
    }, 0);
  };

  const toggleSessionPanel = () => {
    setMetaOpen((current) => !current);
  };

  const openMobileSessionPanelOnPress = () => {
    mobileSessionGestureLockRef.current = true;
    releaseMobileSessionGestureLock();
    blurComposerInput();
    toggleSessionPanel();
  };

  const handleSubmitPointerDownCapture = (event: PointerEvent<HTMLButtonElement>) => {
    if (
      !workbench.isMobileViewport
      || event.pointerType === "mouse"
      || submitting
      || !canInput
      || mobileSubmitGestureLockRef.current
    ) {
      return;
    }
    event.preventDefault();
    submitMobileInputOnPress();
  };

  const handleSubmitTouchStartCapture = (event: TouchEvent<HTMLButtonElement>) => {
    if (!workbench.isMobileViewport || submitting || !canInput || mobileSubmitGestureLockRef.current) {
      return;
    }
    event.preventDefault();
    submitMobileInputOnPress();
  };

  const handleSessionUtilityPointerDownCapture = (event: PointerEvent<HTMLButtonElement>) => {
    if (!workbench.isMobileViewport || event.pointerType === "mouse" || mobileSessionGestureLockRef.current) {
      return;
    }
    event.preventDefault();
    openMobileSessionPanelOnPress();
  };

  const handleSessionUtilityTouchStartCapture = (event: TouchEvent<HTMLButtonElement>) => {
    if (!workbench.isMobileViewport || mobileSessionGestureLockRef.current) {
      return;
    }
    event.preventDefault();
    openMobileSessionPanelOnPress();
  };

  useRuntimeComposerViewportSync({
    isMobileViewport: workbench.isMobileViewport,
    inputFocused,
    workspaceBodyRef,
    composerShellRef,
  });

  const captureScrollSnapshot = () => {
    const node = chatScreenRef.current;
    if (!node) {
      scrollRestoreSnapshotRef.current = null;
      return;
    }
    const remaining = Math.max(node.scrollHeight - node.clientHeight - node.scrollTop, 0);
    scrollRestoreSnapshotRef.current = {
      top: node.scrollTop,
      anchoredToBottom: remaining <= SCROLL_BOTTOM_ANCHOR_THRESHOLD,
    };
  };

  const restoreScrollSnapshot = () => {
    const snapshot = scrollRestoreSnapshotRef.current;
    const node = chatScreenRef.current;
    if (!snapshot || !node) {
      return;
    }
    node.scrollTop = snapshot.anchoredToBottom ? node.scrollHeight : snapshot.top;
    scrollRestoreSnapshotRef.current = null;
  };

  const refreshList = async () => {
    const payload = await apiClient.get<TerminalSessionsResponse>("/api/terminal/sessions");
    const nextSessions = (Array.isArray(payload.items) ? payload.items : []).filter(
      (session) => !deletedSessionIDsRef.current.has(session.id),
    );
    setSessions((current) => {
      const currentMap = new Map(current.map((session) => [session.id, session]));
      return sortSessions(
        nextSessions.map((session) => mergeSessionSnapshot(currentMap.get(session.id), session)),
      );
    });
    setActiveSessionID((current) => {
      const resolvedCurrent = resolveSessionIDReference(nextSessions, current);
      if (resolvedCurrent) {
        return resolvedCurrent;
      }
      return nextSessions[0]?.id || "";
    });
    return nextSessions;
  };

  const refreshActiveSession = async (sessionID: string, options: TerminalSessionRefreshOptions = {}) => {
    if (!sessionID || deletedSessionIDsRef.current.has(sessionID)) {
      return null;
    }
    const query = new URLSearchParams();
    const beforeTurnID = normalizeAttachmentText(options.turnBefore);
    if (beforeTurnID) {
      query.set("turn_before", beforeTurnID);
    }
    if (typeof options.turnLimit === "number" && Number.isFinite(options.turnLimit) && options.turnLimit > 0) {
      query.set("turn_limit", String(Math.floor(options.turnLimit)));
    }
    const queryString = query.toString();
    const payload = await apiClient.get<TerminalSessionResponse>(
      `/api/terminal/sessions/${encodeURIComponent(sessionID)}${queryString ? `?${queryString}` : ""}`,
    );
    const nextSession = payload.session || null;
    if (!nextSession || deletedSessionIDsRef.current.has(nextSession.id)) {
      return null;
    }
    setSessions((current) => {
      const existing = current.some((session) => session.id === sessionID);
      const merged = existing
        ? current.map((session) =>
            session.id === sessionID
              ? mergeSessionSnapshot(session, nextSession)
              : session,
          )
        : [nextSession, ...current];
      return sortSessions(merged);
    });
    return nextSession;
  };

  const refreshTerminalOnPageActive = async () => {
    await refreshList().catch(() => null);
    if (!activeSession) {
      return;
    }
    captureScrollSnapshot();
    await refreshActiveSession(activeSession.id).catch(() => null);
  };

  const createSession = async () => {
    const payload = await apiClient.post<TerminalSessionResponse>("/api/terminal/sessions", {});
    if (!payload.session) {
      return null;
    }
    const nextSession = payload.session;
    setSessions((current) =>
      sortSessions([nextSession, ...current.filter((session) => session.id !== nextSession.id)]),
    );
    setActiveSessionID(nextSession.id);
    closeMobileSessionPane();
    setMetaOpen(false);
    setExpandedTurns({});
    setExpandedEvents({});
    setEventDetails({});
    setEventErrors({});
    return nextSession;
  };

  const focusNewSessionPlaceholder = () => {
    setActiveSessionID("");
    closeMobileSessionPane();
    setMetaOpen(false);
    setExpandedTurns({});
    setExpandedEvents({});
    setEventDetails({});
    setEventErrors({});
    requestAnimationFrame(() => focusComposerInputWithoutScroll());
  };

  useEffect(() => {
    const observer = new MutationObserver(() => setLanguage(resolveLanguage()));
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["lang"],
    });
    return () => observer.disconnect();
  }, []);

  usePageActivation({
    debounceMs: PAGE_ACTIVE_REFRESH_DEBOUNCE_MS,
    onVisibilityChange: setPageHidden,
    onActive: refreshTerminalOnPageActive,
  });

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const payload = await apiClient.get<{ items?: TerminalSkill[] }>("/api/control/skills");
        if (cancelled) {
          return;
        }
        const nextSkills = normalizeTerminalSkills(payload.items);
        setSkills(nextSkills);
        setSelectedSkillIDs((current) => {
          const available = new Set(nextSkills.map((skill) => skill.id));
          const nextSelected = current.filter((id) => available.has(id));
          if (nextSelected.length > 0) {
            return nextSelected;
          }
          return resolveDefaultTerminalSkillIDs(nextSkills);
        });
      } catch {
        if (!cancelled) {
          setSkills([]);
          setSelectedSkillIDs([]);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [apiClient]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      setLoadError("");
      try {
        const nextSessions = await refreshList();
        if (!cancelled && nextSessions.length === 0) {
          setActiveSessionID("");
        }
      } catch (error) {
        if (!cancelled) {
          setLoadError(error instanceof Error ? error.message : "load failed");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    writeWorkbenchRouteSessionID("terminal", activeSessionID);
  }, [activeSessionID]);

  useEffect(() => {
    writeTerminalRuntimeCache(sessions, activeSessionID);
  }, [activeSessionID, sessions]);

  useEffect(() => {
    if (!activeSessionID) {
      setInputValue("");
      return;
    }
    setInputValue(window.localStorage.getItem(`terminal:${activeSessionID}`) || "");
  }, [activeSessionID]);

  useEffect(() => {
    setComposerAttachmentError("");
    setPreviewAttachment(null);
  }, [activeDraftKey]);

  useEffect(() => {
    if (!activeSessionID) {
      return;
    }
    if (draftPersistTimerRef.current !== null) {
      window.clearTimeout(draftPersistTimerRef.current);
    }
    const persistDelay = scrollingActive ? SCROLL_IDLE_MS : 0;
    draftPersistTimerRef.current = window.setTimeout(() => {
      window.localStorage.setItem(`terminal:${activeSessionID}`, inputValue);
      draftPersistTimerRef.current = null;
    }, persistDelay);
    return () => {
      if (draftPersistTimerRef.current !== null) {
        window.clearTimeout(draftPersistTimerRef.current);
        draftPersistTimerRef.current = null;
      }
    };
  }, [activeSessionID, inputValue, scrollingActive]);

  useEffect(() => {
    try {
      window.localStorage.setItem(TERMINAL_ATTACHMENT_DRAFT_STORAGE_KEY, JSON.stringify(attachmentDrafts));
    } catch {
      // Ignore localStorage persistence errors for attachment drafts.
    }
  }, [attachmentDrafts]);

  useEffect(() => {
    if (!activeSessionResolvedID) {
      return;
    }
    void refreshActiveSession(activeSessionResolvedID);
  }, [activeSessionResolvedID]);

  useEffect(() => {
    const sessionID = normalizeAttachmentText(activeSession?.id);
    const paging = activeSession?.turns_paging;
    const beforeTurnID = normalizeAttachmentText(paging?.next_before_turn_id || paging?.oldest_turn_id)
      || oldestTerminalTurnID(turns);
    if (
      !sessionID
      || paging?.has_more_before !== true
      || !beforeTurnID
    ) {
      return;
    }
    const requestKey = `${sessionID}:${beforeTurnID}`;
    if (progressiveHistoryLoadsRef.current.has(requestKey)) {
      return;
    }
    let cancelled = false;
    progressiveHistoryLoadsRef.current.add(requestKey);
    void (async () => {
      try {
        if (!cancelled) {
          await refreshActiveSession(sessionID, {
            turnBefore: beforeTurnID,
            turnLimit: TERMINAL_HISTORY_PAGE_TURN_LIMIT,
          });
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
    activeSession?.turns_paging?.has_more_before,
    activeSession?.turns_paging?.next_before_turn_id,
    activeSession?.turns_paging?.oldest_turn_id,
    turns,
  ]);

  useEffect(() => {
    if (!activeSessionID || !activeSession || !pollPlan.enabled) {
      return;
    }
    const timer = window.setTimeout(() => {
      if (pollPlan.refreshActiveSession) {
        captureScrollSnapshot();
      }
      void refreshList();
      if (pollPlan.refreshActiveSession) {
        void refreshActiveSession(activeSession.id);
      }
    }, pollPlan.interval);
    return () => window.clearTimeout(timer);
  }, [activeSession, activeSessionID, pollPlan.enabled, pollPlan.interval, pollPlan.refreshActiveSession, sessions]);

  useLayoutEffect(() => {
    if (activeTimelineSessionRef.current !== activeSessionID) {
      activeTimelineSessionRef.current = activeSessionID;
      timelineBottomPinnedSessionRef.current = "";
    }
    if (!activeSessionID) {
      timelineBottomPinnedSessionRef.current = "";
      return;
    }
    if (
      timelineBottomPinnedSessionRef.current === activeSessionID
      || turns.length === 0
      || scrollRestoreSnapshotRef.current
    ) {
      return;
    }

    const node = chatScreenRef.current;
    if (!node) {
      return;
    }
    const pinToBottom = () => {
      node.scrollTop = node.scrollHeight;
    };
    pinToBottom();
    const pinnedTop = node.scrollTop;
    const frame = window.requestAnimationFrame(() => {
      if (node.scrollTop === pinnedTop) {
        pinToBottom();
      }
    });
    timelineBottomPinnedSessionRef.current = activeSessionID;
    return () => {
      window.cancelAnimationFrame(frame);
    };
  }, [activeSessionID, turns.length]);

  useEffect(() => {
    setExpandedTurns((current) => {
      let changed = false;
      const next = { ...current };
      const valid = new Set(turns.map((turn) => turn.id));
      Object.keys(next).forEach((turnID) => {
        if (!valid.has(turnID)) {
          delete next[turnID];
          changed = true;
        }
      });
      turns.forEach((turn) => {
        if (typeof next[turn.id] === "undefined") {
          next[turn.id] = !normalizeText(turn.final_output);
          changed = true;
        }
      });
      return changed ? next : current;
    });
  }, [turns]);

  useLayoutEffect(() => {
    restoreScrollSnapshot();
  }, [activeSessionID, turns, expandedTurns, expandedEvents, eventDetails, metaOpen]);

  useEffect(() => {
    return () => {
      if (scrollIdleTimerRef.current !== null) {
        window.clearTimeout(scrollIdleTimerRef.current);
      }
      if (draftPersistTimerRef.current !== null) {
        window.clearTimeout(draftPersistTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (
      !restoreMobileSessionPaneRef.current
      || !workbench.isMobileViewport
      || workbench.mobileSessionPaneOpen
    ) {
      return;
    }
    restoreMobileSessionPaneRef.current = false;
    workbench.openMobileSessionPane();
  }, [
    activeSessionID,
    sessions.length,
    workbench,
  ]);

  useEffect(() => {
    if (
      restoreMobileSessionPaneRef.current
      && workbench.isMobileViewport
      && workbench.mobileSessionPaneOpen
      && !deletingSessionID
    ) {
      restoreMobileSessionPaneRef.current = false;
    }
  }, [
    deletingSessionID,
    sessions.length,
    workbench.isMobileViewport,
    workbench.mobileSessionPaneOpen,
  ]);

  const closeMobileSessionPane = () => {
    restoreMobileSessionPaneRef.current = false;
    workbench.closeMobileSessionPane();
  };

  const selectSession = async (sessionID: string) => {
    setActiveSessionID(sessionID);
    closeMobileSessionPane();
    setMetaOpen(false);
    setExpandedTurns({});
    setExpandedEvents({});
    setEventDetails({});
    setEventErrors({});
    await refreshActiveSession(sessionID);
  };

  const viewSessionDetails = async (sessionID: string) => {
    setActiveSessionID(sessionID);
    setMetaOpen(false);
    setExpandedTurns({});
    setExpandedEvents({});
    setEventDetails({});
    setEventErrors({});
    await refreshActiveSession(sessionID);
    setSessionDetailsOpen(true);
  };

  const deleteSession = async (sessionID: string) => {
    const keepMobileSessionPaneOpen = workbench.isMobileViewport && workbench.mobileSessionPaneOpen;
    restoreMobileSessionPaneRef.current = keepMobileSessionPaneOpen;
    setDeletingSessionID(sessionID);
    try {
      await apiClient.delete(`/api/terminal/sessions/${encodeURIComponent(sessionID)}`);
      deletedSessionIDsRef.current.add(sessionID);
      setSessions((current) => {
        const next = current.filter((session) => session.id !== sessionID);
        setActiveSessionID((currentActiveSessionID) =>
          currentActiveSessionID === sessionID ? next[0]?.id || "" : currentActiveSessionID,
        );
        return next;
      });
      window.localStorage.removeItem(`terminal:${sessionID}`);
      clearDraftAttachments(sessionID);
    } finally {
      setDeletingSessionID("");
    }
  };

  const setSessionPinned = async (sessionID: string, pinned: boolean) => {
    setPinningSessionID(sessionID);
    try {
      const payload = await apiClient.post<TerminalSessionResponse>(
        `/api/terminal/sessions/${encodeURIComponent(sessionID)}/pin`,
        { pinned },
      );
      setSessions((current) =>
        sortSessions(
          current.map((session) =>
            session.id === sessionID
              ? mergeSessionSnapshot(session, payload.session || { id: sessionID, pinned })
              : session,
          ),
        ),
      );
    } finally {
      setPinningSessionID("");
    }
  };

  const handleComposerAttachmentPicker = () => {
    composerFileInputRef.current?.click();
  };

  const handleComposerAttachmentSelection = async (files: FileList | File[] | null) => {
    if (!files || files.length === 0) {
      return;
    }
    if ((draftAttachments.length + files.length) > MAX_COMPOSER_IMAGE_ATTACHMENTS) {
      setComposerAttachmentError(
        language === "zh"
          ? `最多可暂存 ${MAX_COMPOSER_IMAGE_ATTACHMENTS} 个附件。`
          : `You can attach up to ${MAX_COMPOSER_IMAGE_ATTACHMENTS} files.`,
      );
      return;
    }
    let uploadSessionID = "";
    let uploadPromise: Promise<ComposerAttachment[]> | null = null;
    try {
      const attachments = await readComposerFiles(files);
      let session = activeSession;
      if (!session) {
        session = await createSession();
      }
      if (!session) {
        return;
      }
      updateDraftAttachments(session.id, (current) => [...current, ...attachments]);
      uploadSessionID = session.id;
      uploadPromise = uploadTerminalSessionAttachments(apiClient, session.id, attachments);
      attachmentUploadPromisesRef.current[session.id] = {
        pendingIDs: attachments.map((attachment) => attachment.id),
        promise: uploadPromise,
      };
      const uploaded = await uploadPromise;
      if (activeDraftKey !== session.id) {
        clearDraftAttachments(activeDraftKey);
      }
      updateDraftAttachments(session.id, (current) => [
        ...current.filter((item) => !attachments.some((pending) => pending.id === item.id)),
        ...uploaded,
      ]);
      setComposerAttachmentError("");
    } catch (error) {
      setComposerAttachmentError(error instanceof Error ? error.message : "Failed to add attachment.");
    } finally {
      if (
        uploadSessionID
        && uploadPromise
        && attachmentUploadPromisesRef.current[uploadSessionID]?.promise === uploadPromise
      ) {
        delete attachmentUploadPromisesRef.current[uploadSessionID];
      }
      if (composerFileInputRef.current) {
        composerFileInputRef.current.value = "";
      }
    }
  };

  const handleComposerInputPaste = (event: ClipboardEvent<HTMLTextAreaElement>) => {
    const imageFiles = getPastedComposerImageFiles(event.clipboardData);
    if (imageFiles.length === 0) {
      return;
    }
    event.preventDefault();
    void handleComposerAttachmentSelection(imageFiles);
  };

  const submitInput = async () => {
    const content = inputValue.trim();
    const draftKey = activeDraftKey;
    if (submitting) {
      return;
    }
    setSubmitting(true);
    let session = activeSession;
    try {
      if (!session) {
        session = await createSession();
      }
      if (!session) {
        return;
      }
      let nextDraftAttachments = attachmentDraftsRef.current[session.id] || attachmentDraftsRef.current[draftKey] || [];
      const pendingUpload = attachmentUploadPromisesRef.current[session.id];
      if (pendingUpload) {
        const uploaded = await pendingUpload.promise.catch(() => null);
        if (Array.isArray(uploaded) && uploaded.length > 0) {
          nextDraftAttachments = [
            ...nextDraftAttachments.filter((item) => !pendingUpload.pendingIDs.includes(item.id)),
            ...uploaded,
          ];
        }
      }
      if (nextDraftAttachments.some((attachment) => !attachment.assetURL && attachment.dataURL)) {
        nextDraftAttachments = await uploadTerminalSessionAttachments(apiClient, session.id, nextDraftAttachments);
        updateDraftAttachments(session.id, () => nextDraftAttachments);
        if (draftKey !== session.id) {
          clearDraftAttachments(draftKey);
        }
      }
      const attachments = nextDraftAttachments.map(serializeTerminalComposerAttachment);
      if (content === "" && attachments.length === 0) {
        return;
      }
      const payload = await apiClient.post<TerminalSessionResponse>(
        `/api/terminal/sessions/${encodeURIComponent(session.id)}/input`,
        { input: content, attachments, skill_ids: selectedSkillIDs },
      );
      window.localStorage.removeItem(`terminal:${session.id}`);
      setInputValue("");
      clearDraftAttachments(draftKey);
      if (draftKey !== session.id) {
        clearDraftAttachments(session.id);
      }
      setComposerAttachmentError("");
      if (payload.session) {
        setSessions((current) =>
          sortSessions(
            current.map((item) =>
              item.id === session!.id
                ? mergeSessionSnapshot(item, payload.session as TerminalSession)
                : item,
            ),
          ),
        );
      }
      await refreshActiveSession(session.id);
      window.requestAnimationFrame(() => {
        const node = chatScreenRef.current;
        if (node) {
          node.scrollTop = node.scrollHeight;
        }
      });
    } finally {
      setSubmitting(false);
    }
  };

  const toggleTurn = useCallback((turnID: string) => {
    setExpandedTurns((current) => ({ ...current, [turnID]: !current[turnID] }));
  }, []);

  const toggleEvent = useCallback(async (turnID: string, eventID: string, hasDetail: boolean) => {
    const key = eventKey(turnID, eventID);
    const currentlyExpanded = Boolean(expandedEvents[key]);
    if (currentlyExpanded) {
      setExpandedEvents((current) => ({ ...current, [key]: false }));
      return;
    }
    if (!hasDetail || eventDetails[key] || !activeSession) {
      setExpandedEvents((current) => ({ ...current, [key]: true }));
      return;
    }
    try {
      const payload = await apiClient.get<RuntimeTraceEventDetailResponse>(
        `/api/terminal/sessions/${encodeURIComponent(activeSession.id)}/turns/${encodeURIComponent(turnID)}/events/${encodeURIComponent(eventID)}`,
      );
      if (payload.event) {
        setEventDetails((current) => ({ ...current, [key]: payload.event as RuntimeTraceEventDetail }));
      }
      setExpandedEvents((current) => ({ ...current, [key]: true }));
    } catch (error) {
      setEventErrors((current) => ({
        ...current,
        [key]: error instanceof Error ? error.message : "unknown error",
      }));
      setExpandedEvents((current) => ({ ...current, [key]: true }));
    }
  }, [activeSession, apiClient, expandedEvents, eventDetails]);

  const handleScroll = () => {
    setScrollingActive(true);
    if (scrollIdleTimerRef.current !== null) {
      window.clearTimeout(scrollIdleTimerRef.current);
    }
    scrollIdleTimerRef.current = window.setTimeout(() => {
      setScrollingActive(false);
      scrollIdleTimerRef.current = null;
    }, SCROLL_IDLE_MS);
  };

  const handleJumpControlPointerDown = (event: PointerEvent<HTMLButtonElement>) => {
    if (event.pointerType === "mouse") {
      event.preventDefault();
    }
  };

  const activeNote = runtimeNote(activeSession?.status || "", copy);
  const runtimeDetail = String(activeSession?.error_message || "").trim();
  const composerNote = [activeNote, runtimeDetail, composerAttachmentError].filter(Boolean).join(" | ");
  const canInput = !activeSession || activeStatus !== "busy";
  const isWorkspaceLive = activeSession && isLiveStatus(activeSession.status || "") ? "true" : "false";
  const inputPlaceholder = canInput ? copy.inputPlaceholder : copy.busy;
  const showNewSessionPlaceholder = !loadError && sessions.length === 0;
  const visibleSessionCount = showNewSessionPlaceholder ? 1 : sessions.length;
  const startNewSession = () => {
    if (showNewSessionPlaceholder) {
      focusNewSessionPlaceholder();
      return;
    }
    void createSession();
  };
  const activeSessionTitle = activeSession
    ? normalizeText(activeSession.title || activeSession.id)
    : showNewSessionPlaceholder
      ? copy.newShort
      : copy.noSession;
  const terminalDetailsSummary = activeSession ? [
    { label: copy.session, value: activeSession.id, copyLabel: copy.session, mono: true },
    { label: copy.shell, value: activeSession.shell, copyLabel: copy.shell, mono: true },
    { label: copy.path, value: activeSession.working_dir, copyLabel: copy.path, mono: true, multiline: true },
    { label: copy.status, value: renderStatus(activeSession.status || "", copy), copyLabel: copy.status },
    {
      label: copy.updatedAt,
      value: formatDateTime(activeSession.updated_at || activeSession.created_at),
      copyLabel: copy.updatedAt,
    },
  ] : [];
  const codexSlashCommandsLabel = language === "zh" ? "Codex 斜线命令" : "Codex slash commands";
  const terminalCodexSlashQuery = activeSession && isCodexShellSession(activeSession) && canInput && !submitting
    ? codexSlashCommandQuery(inputValue)
    : "";
  const terminalCodexSlashCommandCandidates = terminalCodexSlashQuery
    ? CODEX_SLASH_COMMANDS.filter((item) => item.command.startsWith(terminalCodexSlashQuery))
    : [];
  const applyTerminalCodexSlashCommand = (command: string) => {
    setInputValue(buildDraftWithCodexSlashCommand(inputValue, command));
    focusComposerInputWithoutScroll();
  };
  const terminalCodexSlashCommandAssist = terminalCodexSlashCommandCandidates.length > 0 ? (
    <div
      className="runtime-composer-command-list"
      role="listbox"
      aria-label={codexSlashCommandsLabel}
      data-runtime-composer-command-list="codex"
    >
      {terminalCodexSlashCommandCandidates.map((item) => (
        <button
          key={item.command}
          type="button"
          role="option"
          className="runtime-composer-command-option"
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => applyTerminalCodexSlashCommand(item.command)}
        >
          <strong>{item.command}</strong>
          <span>{item.label[language]}</span>
        </button>
      ))}
    </div>
  ) : null;
  const terminalConfigPanel = metaOpen ? (
    <div
      className="conversation-inspector runtime-composer-config-panel"
      data-runtime-config-panel="terminal"
      data-runtime-config-tab="skills"
    >
      <div className="runtime-composer-panel-head">
        <strong>{shellCopy.runtimeMobile}</strong>
        <button type="button" className="runtime-composer-panel-close" onClick={() => setMetaOpen(false)}>
          {shellCopy.sessionClose}
        </button>
      </div>
      <p className="runtime-composer-panel-hint">{shellCopy.runtimeSkillsHint}</p>
      <section className="conversation-inspector-section terminal-skill-section" data-testid="terminal-skill-selector">
        <strong>{copy.skills}</strong>
        <div className="conversation-check-list">
          {skillOptions.length > 0 ? skillOptions.map((skill) => (
            <label className="conversation-check-item" key={skill.id}>
              <input
                type="checkbox"
                value={skill.id}
                checked={skill.active}
                aria-label={skill.name}
                data-runtime-toggle-item="terminal-skills"
                onChange={(event) => toggleSkill(skill.id, event.target.checked)}
              />
              <span>
                <strong>{skill.name}</strong>
                <RouteMarkdownContent className="terminal-skill-description" value={skill.description} />
              </span>
            </label>
          )) : (
            <p className="route-empty-panel">{copy.noSkills}</p>
          )}
        </div>
      </section>
    </div>
  ) : null;
  const terminalTimelineItems = useMemo(() => buildTerminalTimelineItems({
    sessionID: activeSession?.id,
    turns,
    expandedTurns,
    expandedEvents,
    eventDetails,
    eventErrors,
    copy,
    language,
    onToggleTurn: toggleTurn,
    onToggleEvent: (turnID, eventID, hasDetail) => void toggleEvent(turnID, eventID, hasDetail),
    onPreviewAttachment: setPreviewAttachment,
  }), [
    copy,
    expandedEvents,
    expandedTurns,
    language,
    eventDetails,
    eventErrors,
    toggleEvent,
    toggleTurn,
    turns,
    activeSession?.id,
  ]);

  return {
    shell: {
      rootClassName: "runtime-workspace-view",
      rootProps: { "data-runtime-view": "terminal" },
      sessionPaneClassName: workbench.isMobileViewport && workbench.mobileSessionPaneOpen
        ? "is-open"
        : undefined,
      sessionPaneProps: { "data-runtime-session-pane": "terminal" },
      sessionPaneBackdrop: {
        ariaLabel: copy.hideSessions,
        onClick: closeMobileSessionPane,
        buttonProps: { "data-runtime-session-pane-close": "terminal" },
      },
      sessionPanePrimaryActionClassName: "is-primary",
      sessionPanePrimaryActionProps: { "data-runtime-create-session": "terminal" },
      sessionPaneSecondaryActionClassName: "runtime-workspace-session-pane-close",
      sessionPaneSecondaryActionProps: { "data-runtime-session-pane-close": "terminal" },
      sessionPaneTitle: copy.sessions,
      sessionPaneCountLabel: copy.sessionCount(visibleSessionCount),
      sessionPanePrimaryActionLabel: copy.newShort,
      onSessionPanePrimaryAction: startNewSession,
      sessionPaneSecondaryActionLabel: workbench.isMobileViewport ? copy.hideSessions : undefined,
      onSessionPaneSecondaryAction: workbench.isMobileViewport ? closeMobileSessionPane : undefined,
      workspaceProps: {
        "data-runtime-workspace": "terminal",
        "data-runtime-session-id": activeSession?.id || "",
        "data-runtime-status": activeStatus,
        "data-runtime-live": isWorkspaceLive,
      },
      workspaceBodyRef,
      mobileHeaderPlacement: workbench.isMobileViewport ? "body" : undefined,
      mobileHeaderProps: { "data-runtime-mobile-variant": "terminal" },
      mobileNavButtonClassName: "is-quiet conversation-mobile-nav-toggle",
      mobileNavButtonLabel: shellCopy.chatMenu,
      mobileNavButtonProps: { "aria-expanded": workbench.mobileNavOpen },
      onMobileNav: workbench.toggleMobileNav,
      mobileTitleButtonClassName: "conversation-mobile-title-toggle",
      mobileTitleButtonLabel: activeSessionTitle,
      mobileTitleStatusLabel: activeSession ? renderStatus(activeSession.status || "", copy) : copy.ready,
      mobileTitleTone: activeStatus,
      mobileTitleButtonProps: {
        "aria-haspopup": "dialog",
        "data-runtime-mobile-title": "terminal",
        disabled: !activeSession,
      },
      onMobileTitle: activeSession ? () => setSessionDetailsOpen((current) => !current) : undefined,
      mobilePrimaryButtonClassName: "is-primary conversation-mobile-new-session",
      mobilePrimaryButtonLabel: copy.newShort,
      mobilePrimaryButtonProps: {
        "data-runtime-create-session": "terminal",
        "data-runtime-mobile-primary": "terminal",
      },
      onMobilePrimary: startNewSession,
    },
    sessionList: {
      groups: showNewSessionPlaceholder
        ? [{
            key: "terminal-new",
            label: language === "zh" ? "今天" : "Today",
            items: [{
              id: TERMINAL_NEW_SESSION_PLACEHOLDER_ID,
              active: true,
              title: copy.newShort,
              meta: copy.noOutputMeta,
              shortHash: "",
              statusTone: "ready",
              statusLabel: copy.ready,
              activeLabel: copy.current,
              idleLabel: copy.sessionLabel,
              onSelect: startNewSession,
              shellClassName: "runtime-session-card is-active",
              shellProps: {
                "data-runtime-session-card": TERMINAL_NEW_SESSION_PLACEHOLDER_ID,
                "data-runtime-session-status": "ready",
                "data-runtime-session-tone": "ready",
              },
              buttonClassName: "runtime-session-select active",
              buttonProps: { "data-runtime-session-select": TERMINAL_NEW_SESSION_PLACEHOLDER_ID },
            }],
          }]
        : groupedSessions.map((group) => ({
            ...group,
            items: group.items.map((session) => {
              const active = session.id === activeSessionID;
              const tone = sessionSignalTone(session.status || "");
              return {
                id: session.id,
                active,
                title: normalizeText(session.title || session.id),
                meta: sessionLastOutputLabel(session, copy),
                shortHash: hashSessionIDShort(normalizeText(session.id)),
                statusTone: tone,
                statusLabel: renderStatus(session.status || "", copy),
                activeLabel: copy.current,
                idleLabel: copy.sessionLabel,
                onSelect: () => void selectSession(session.id),
                pinned: Boolean(session.pinned),
                pinning: pinningSessionID === session.id,
                onPinnedChange: (pinned) => void setSessionPinned(session.id, pinned),
                pinLabel: copy.pin,
                unpinLabel: copy.unpin,
                pinAriaLabel: copy.pinSession,
                unpinAriaLabel: copy.unpinSession,
                pinProps: { "data-runtime-pin-session": session.id },
                onViewDetails: () => void viewSessionDetails(session.id),
                viewDetailsLabel: copy.details,
                viewDetailsAriaLabel: copy.details,
                onDelete: () => void deleteSession(session.id),
                deleteLabel: copy.delete,
                deleteAriaLabel: copy.delete,
                deleteConfirmLabel: copy.deleteConfirm,
                actionsLabel: copy.sessionActions,
                actionsAriaLabel: copy.sessionActions,
                deleting: deletingSessionID === session.id,
                deleteProps: { "data-runtime-delete-session": session.id },
                shellClassName: active ? "runtime-session-card is-active" : "runtime-session-card",
                shellProps: {
                  "data-runtime-session-card": session.id,
                  "data-runtime-session-status": normalizeStatus(session.status || ""),
                  "data-runtime-session-tone": tone,
                },
                buttonClassName: active ? "runtime-session-select active" : "runtime-session-select",
                buttonProps: { "data-runtime-session-select": session.id },
              };
            }),
          })),
      listProps: { "data-runtime-session-list": "terminal" },
      emptyState: (
        <>
          {loadError ? <p className="route-empty-panel">{loadError}</p> : null}
          {!loadError && !loading && !showNewSessionPlaceholder && groupedSessions.length === 0 ? (
            <p className="route-empty-panel">{copy.empty}</p>
          ) : null}
        </>
      ),
    },
    header: {
      title: activeSessionTitle,
      statusLabel: activeSession ? renderStatus(activeSession.status || "", copy) : copy.ready,
      statusTone: activeStatus,
      detailsLabel: copy.details,
      detailsOpen: activeSession ? sessionDetailsOpen : false,
      onToggleDetails: activeSession ? () => setSessionDetailsOpen((current) => !current) : () => undefined,
      detailsDisabled: !activeSession,
      mobileCollapsed: workbench.isMobileViewport,
      detailsSummary: terminalDetailsSummary,
      detailsBody: null,
      detailsClassName: "runtime-workspace-meta-panel workspace-details-content",
      headerProps: { "data-runtime-header-kind": "conversation" },
      detailsPanelProps: { "data-runtime-details-panel": "terminal" },
    },
    screen: {
      panelClassName: "terminal-console-panel",
      panelProps: { "data-runtime-panel": "terminal-console" },
      screenProps: {
        "data-runtime-screen": "terminal",
        "data-runtime-status": activeStatus,
        onScroll: handleScroll,
      },
      screenRef: chatScreenRef,
    },
    timeline: {
      className: "terminal-log-tree",
      items: terminalTimelineItems,
      emptyState: !activeSession ? (
        <div className="terminal-log-empty">{loading ? copy.loading : showNewSessionPlaceholder ? copy.noOutput : copy.noSession}</div>
      ) : (
        <div className="terminal-log-empty">{loading ? copy.loading : copy.noOutput}</div>
      ),
      overlay: workbench.isMobileViewport && inputFocused ? null : (
        <ScrollJumpStrip
          scope="terminal"
          namespace="terminal"
          language={language}
          containerRef={chatScreenRef}
          itemSelector="[data-terminal-turn]"
          itemAttribute="data-terminal-turn"
          watchKey={`${activeSessionID}:${turns.length}:${Object.keys(expandedTurns).length}:${Object.keys(expandedEvents).length}:${Object.keys(eventDetails).length}:${metaOpen ? "meta" : "plain"}`}
          suppressNextTarget={submitting}
          onControlPointerDown={handleJumpControlPointerDown}
        />
      ),
    },
    composer: {
      runtimeKind: "terminal",
      shellRef: composerShellRef,
      onSubmit: (event) => {
        event.preventDefault();
        void submitInput();
      },
      fileInputRef: composerFileInputRef,
      fileInputAccept: copy.addAttachmentAccept,
      onFileChange: (event) => {
        void handleComposerAttachmentSelection(event.target.files);
      },
      attachments: draftAttachments,
      attachmentStripProps: { "data-runtime-attachments": "terminal" },
      attachmentPreviewLabel: (attachment) => `${copy.preview} ${attachment.name}`,
      attachmentRemoveLabel: (attachment) => `${copy.delete} ${attachment.name}`,
      previewAttachment,
      onPreviewAttachmentChange: setPreviewAttachment,
      onRemoveAttachment: (attachment) => updateDraftAttachments(activeDraftKey, (current) =>
        current.filter((item) => item.id !== attachment.id)),
      inputLabel: inputPlaceholder,
      inputId: "terminalRuntimeInput",
      inputRef: composerInputRef,
      inputValue: inputValue,
      inputProps: {
        placeholder: inputPlaceholder,
        disabled: !canInput || submitting,
        onPaste: handleComposerInputPaste,
      },
      inputAssistContent: terminalCodexSlashCommandAssist,
      onInputChange: setInputValue,
      onInputFocus: () => setInputFocused(true),
      onInputBlur: () => setInputFocused(false),
      utilityButtons: [
        {
          key: "session",
          label: shellCopy.runtimeMobile,
          icon: <RuntimeSessionControlIcon />,
          className: metaOpen ? "is-active" : undefined,
          onClick: () => {
            if (mobileSessionGestureLockRef.current) {
              return;
            }
            toggleSessionPanel();
          },
          buttonProps: {
            onPointerDownCapture: handleSessionUtilityPointerDownCapture,
            onTouchStartCapture: handleSessionUtilityTouchStartCapture,
          },
        },
      ],
      panelContent: terminalConfigPanel,
      onPanelDismiss: () => setMetaOpen(false),
      panelProps: {
        "data-runtime-config-surface": "terminal",
      },
      metaContent: composerNote || undefined,
      metaProps: { "data-runtime-status": activeStatus },
      addAttachmentLabel: copy.addAttachment,
      addAttachmentButtonProps: { disabled: !canInput || submitting },
      onAddAttachment: handleComposerAttachmentPicker,
      submitButtonProps: {
        id: "terminalSendButton",
        disabled: submitting || !canInput,
        onPointerDownCapture: handleSubmitPointerDownCapture,
        onTouchStartCapture: handleSubmitTouchStartCapture,
      },
      submitLabel: submitting ? copy.sending : copy.send,
      previewCloseLabel: copy.closePreview,
    },
  };
}

export function ReactManagedTerminalRouteBody() {
  const controller = useTerminalRuntimeController();
  return <RuntimeWorkspacePage controller={controller} />;
}

function terminalTurnRuntimeEvents(
  sessionID: string,
  turn: TerminalTurn,
): RuntimeTraceEvent[] {
  return normalizeRuntimeTraceEvents(turn.runtime_trace_events, {
    sessionID,
    turnID: turn.id,
  });
}

function runtimeTraceEventDetailBlocks(detail: RuntimeTraceEventDetail | undefined) {
  if (!detail) {
    return [];
  }
  if (detail.event) {
    return runtimeTraceEventToProcessDetailBlocks(detail.event);
  }
  if (Array.isArray(detail.blocks) && detail.blocks.length > 0) {
    return runtimeTraceEventToProcessDetailBlocks({
      id: "detail",
      turn_id: detail.turn_id || "",
      seq: 1,
      source: "adapter",
      provider: { engine: "codex", adapter: "codex_cli_json" },
      role: "assistant",
      kind: "unknown_provider_event",
      lifecycle: "completed",
      status: "completed",
      blocks: detail.blocks,
      visibility: "collapsed",
    });
  }
  return [];
}

function buildTerminalTimelineItems({
  sessionID,
  turns,
  expandedTurns,
  expandedEvents,
  eventDetails,
  eventErrors,
  copy,
  language,
  onToggleTurn,
  onToggleEvent,
  onPreviewAttachment,
}: {
  sessionID?: string;
  turns: TerminalTurn[];
  expandedTurns: Record<string, boolean>;
  expandedEvents: Record<string, boolean>;
  eventDetails: Record<string, RuntimeTraceEventDetail>;
  eventErrors: Record<string, string>;
  copy: TerminalCopy;
  language: "en" | "zh";
  onToggleTurn: (turnID: string) => void;
  onToggleEvent: (turnID: string, eventID: string, hasDetail: boolean) => void;
  onPreviewAttachment: (attachment: ComposerAttachment | null) => void;
}): RuntimeTimelineItem[] {
  return turns.map((turn) => {
    const runtimeEvents = terminalTurnRuntimeEvents(sessionID, turn);
    const turnAttachments = Array.isArray(turn.attachments) ? turn.attachments : [];
    const imageAttachments = turnAttachments.filter((attachment) => attachment.content_type.startsWith("image/"));
    const processOpen = expandedTurns[turn.id] ?? false;
    const hasProcess = runtimeEvents.length > 0 || normalizeStatus(turn.status || "") === "busy";
    const blocks = [];

    if (imageAttachments.length > 0) {
      blocks.push({
        type: "attachments" as const,
        galleryId: turn.id,
        className: "terminal-turn-attachments",
        items: imageAttachments.map((attachment) => {
          const attachmentID = `${turn.id}:${attachment.id || attachment.name}`;
          return {
            key: `${attachmentID}:${attachment.asset_url || attachment.data_url || ""}`,
            name: attachment.name,
            src: resolveComposerAttachmentPreviewURL({
              id: attachmentID,
              kind: "image",
              name: attachment.name,
              contentType: attachment.content_type,
              dataURL: attachment.data_url,
              assetURL: attachment.asset_url,
              previewURL: attachment.preview_url,
              size: 0,
            }),
            previewLabel: `${copy.preview} ${attachment.name}`,
            onPreview: () => onPreviewAttachment({
              id: attachmentID,
              kind: "image",
              name: attachment.name,
              contentType: attachment.content_type,
              dataURL: attachment.data_url,
              assetURL: attachment.asset_url,
              previewURL: attachment.preview_url,
              size: 0,
            }),
          };
        }),
      });
    }

    if (normalizeText(turn.prompt) !== "-") {
      blocks.push({
        type: "prompt" as const,
        className: "terminal-log-row kind-command terminal-turn-prompt runtime-message runtime-message-user",
        bubbleClassName: "msg-bubble runtime-message-bubble runtime-message-user-shell user-message-shell",
        textClassName: "terminal-log-main",
        timeClassName: "terminal-log-time",
        text: turn.prompt,
      });
    }

    if (hasProcess) {
      const processEvents: RuntimeTimelineProcessEvent[] = runtimeEvents.map((runtimeEvent) => {
        const eventID = runtimeTraceEventDetailID(runtimeEvent);
        const key = eventKey(turn.id, eventID);
        const detail = eventDetails[key];
        const error = eventErrors[key];
        const expanded = Boolean(expandedEvents[key]);
        const fallbackBlocks = runtimeTraceEventToProcessDetailBlocks(runtimeEvent);
        const detailBlocks = runtimeTraceEventDetailBlocks(detail);
        const hasDetail = Boolean(runtimeEvent.raw?.has_detail ?? fallbackBlocks.length > 0);
        const shouldFetchDetail = hasDetail && fallbackBlocks.length === 0;
        const fallbackContent = String(runtimeEvent.summary || runtimeEvent.title || "").trim();
        const eventType = normalizeText(runtimeEvent.raw?.type || runtimeEvent.kind || "").toLowerCase();
        const stepCategory = runtimeTraceEventDisclosureCategory(runtimeEvent);
        return {
          id: eventID,
          itemClassName: "terminal-step-item",
          itemProps: {
            "data-terminal-step-item": eventID,
            "data-runtime-event-kind": runtimeEvent.kind,
            "data-runtime-event-source": runtimeEvent.source,
            "data-runtime-event-category": stepCategory,
          },
          toggleClassName: "terminal-step-toggle",
          toggleProps: {
            "data-terminal-step-toggle": eventID,
            onClick: () => onToggleEvent(turn.id, eventID, shouldFetchDetail),
          },
          title: normalizeText(runtimeEvent.summary || runtimeEvent.title || runtimeEvent.kind),
          titleClassName: "terminal-step-title",
          meta: (
            <RuntimeProcessStepMeta
              event={runtimeEvent}
              language={language}
            />
          ),
          expanded,
          onToggle: () => onToggleEvent(turn.id, eventID, shouldFetchDetail),
          bodyClassName: "terminal-step-body",
          detail: (
            <div className="terminal-step-detail">
              {error ? <div className="terminal-step-detail-state is-error">{error}</div> : null}
              {!error ? (
                <RuntimeProcessDetailBlocks
                  blocks={detailBlocks.length > 0 ? detailBlocks : fallbackBlocks}
                  fallbackContent={fallbackContent}
                  fallbackType={eventType}
                  blockKeyPrefix={eventID}
                  emptyState={!hasDetail ? <div className="terminal-step-detail-state">{copy.noProcess}</div> : null}
                />
              ) : null}
            </div>
          ),
        };
      });

      blocks.push({
        type: "process" as const,
        shellClassName: `runtime-thinking-shell terminal-process-shell${processOpen ? "" : " is-collapsed"}`,
        shellProps: { "data-terminal-process-shell": turn.id },
        toggleClassName: "runtime-thinking-toggle terminal-process-toggle",
        toggleProps: { "data-terminal-process-toggle": turn.id },
        title: (
          <>
            <span className="terminal-step-toggle-icon" aria-hidden="true">
              {processOpen ? "v" : ">"}
            </span>
            <span className="terminal-process-copy">
              <span className="terminal-process-title">{copy.process}</span>
              <span className="terminal-process-summary">{copy.processEvents(runtimeEvents.length)}</span>
            </span>
          </>
        ),
        expanded: processOpen,
        onToggle: () => onToggleTurn(turn.id),
        bodyClassName: "terminal-process-body",
        emptyState: (
          <div className="terminal-process-empty">
            {normalizeStatus(turn.status || "") === "busy" ? copy.loading : copy.noProcess}
          </div>
        ),
        events: processEvents,
      });
    }

    if (normalizeText(turn.final_output) !== "-") {
      blocks.push({
        type: "markdown-shell" as const,
        markdown: turn.final_output || "",
        copyValue: turn.final_output,
        copyLabel: copy.copy,
        wrapperClassName: "terminal-final-output terminal-turn-output runtime-message runtime-message-assistant",
        wrapperProps: { "data-terminal-final-output": turn.id },
        bubbleClassName: "runtime-message-bubble runtime-message-assistant-shell assistant-message-shell",
        className: "terminal-final-text",
        toolbarClassName: "terminal-final-toolbar",
        copyButtonClassName: "terminal-final-copy",
        bodyClassName: "terminal-final-rendered",
      });
    }

    return {
      id: turn.id,
      className: "terminal-turn-card",
      articleProps: { "data-terminal-turn": turn.id },
      blocks,
    };
  });
}
