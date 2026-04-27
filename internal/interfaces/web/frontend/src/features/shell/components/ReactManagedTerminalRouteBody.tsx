import { useEffect, useLayoutEffect, useMemo, useRef, useState, type PointerEvent, type TouchEvent } from "react";
import { useWorkbenchContext } from "../../../app/WorkbenchContext";
import { createAPIClient } from "../../../shared/api/client";
import { hashSessionIDShort } from "../../../shared/session/sessionHash";
import { groupSessionListItems } from "../../../shared/time/sessionListGroups";
import { formatDateTime, formatTimeLabel } from "../../../shared/time/format";
import {
  canPreviewComposerAttachment,
  readComposerFiles,
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  resolveComposerAttachmentPreviewURL,
  type ComposerAttachment,
} from "../../conversation-runtime/composerImageAttachments";
import { getLegacyShellCopy } from "../legacyShellCopy";
import { RuntimeWorkspacePage, type RuntimeWorkspacePageController } from "./RuntimeWorkspacePage";
import type { RuntimeTimelineItem, RuntimeTimelineProcessStep } from "./RuntimeTimeline";
import { normalizeText } from "./RouteBodyPrimitives";
import { renderRuntimeMarkdownToHTML } from "./RuntimeMarkdown";
import { useRuntimeComposerViewportSync } from "./useRuntimeComposerViewportSync";

type TerminalStatus = "ready" | "busy" | "exited" | "failed" | "interrupted";

type TerminalStepSummary = {
  id: string;
  type: string;
  title: string;
  status: string;
  duration_ms?: number;
  preview?: string;
  has_detail?: boolean;
};

type TerminalTurn = {
  id: string;
  prompt: string;
  attachments?: TerminalAttachment[];
  status: string;
  started_at?: string | number;
  finished_at?: string | number;
  duration_ms?: number;
  final_output?: string;
  steps?: TerminalStepSummary[];
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
  created_at?: string | number;
  last_output_at?: string | number;
  updated_at?: string | number;
  error_message?: string;
  turns?: TerminalTurn[];
};

type TerminalSessionsResponse = {
  items?: TerminalSession[];
};

type TerminalSessionResponse = {
  session?: TerminalSession;
};

type TerminalStepBlock = {
  type?: string;
  title?: string;
  content?: string;
  language?: string;
  file?: string;
  start_line?: number;
  status?: string;
  exit_code?: number | null;
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

type TerminalStepDetail = {
  turn_id?: string;
  step?: TerminalStepSummary;
  blocks?: TerminalStepBlock[];
  searchable?: boolean;
};

type TerminalStepDetailResponse = {
  step?: TerminalStepDetail;
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
  processSteps: (count: number) => string;
  noProcess: string;
  noOutput: string;
  noOutputMeta: string;
  lastOutput: (label: string) => string;
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
    hideSessions: "Hide sessions",
    empty: "No terminal sessions yet.",
    ready: "Ready",
    busy: "Busy",
    exited: "Exited",
    failed: "Failed",
    interrupted: "Interrupted",
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
    process: "Process",
    processSteps: (count) => `${count} steps`,
    noProcess: "No execution details.",
    noOutput: "No output yet.",
    noOutputMeta: "No output yet.",
    lastOutput: (label) => `Last output ${label}`,
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
    hideSessions: "收起会话",
    empty: "暂时还没有终端会话。",
    ready: "就绪",
    busy: "执行中",
    exited: "已退出",
    failed: "失败",
    interrupted: "已中断",
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
    process: "过程",
    processSteps: (count) => `${count} 步`,
    noProcess: "暂无执行细节。",
    noOutput: "暂时还没有输出。",
    noOutputMeta: "暂无输出。",
    lastOutput: (label) => `最近输出 ${label}`,
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

type JumpState = {
  previousTurnID: string;
  nextTurnID: string;
  showTop: boolean;
  showBottom: boolean;
};

type TerminalJumpMeasurement = {
  id: string;
  top: number;
  bottom: number;
};

const POLL_INTERVAL_MS = 2000;
const INTERACTION_POLL_INTERVAL_MS = 6000;
const HIDDEN_POLL_INTERVAL_MS = 12000;
const IDLE_LIST_POLL_INTERVAL_MS = 12000;
const HIDDEN_IDLE_LIST_POLL_INTERVAL_MS = 30000;
const SCROLL_IDLE_MS = 1200;
const SCROLL_BOTTOM_ANCHOR_THRESHOLD = 24;
const JUMP_TOP_THRESHOLD = 180;
const JUMP_BOTTOM_THRESHOLD = 220;
const TERMINAL_ATTACHMENT_DRAFT_STORAGE_KEY = "alter0.web.terminal.attachments.v1";
const TERMINAL_PENDING_DRAFT_KEY = "__pending__";

type TerminalPollPlan = {
  enabled: boolean;
  interval: number;
  refreshActiveSession: boolean;
};

const DEFAULT_TERMINAL_SKILL_IDS = ["frontend-design", "deploy-test-service"] as const;

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
  return visibility !== "agent-private" && visibility !== "private";
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
  const available = new Set(skills.map((skill) => skill.id));
  return DEFAULT_TERMINAL_SKILL_IDS.filter((id) => available.has(id));
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
    if (options.scrollingActive || options.inputFocused) {
      return {
        enabled: false,
        interval: 0,
        refreshActiveSession: false,
      };
    }
    return {
      enabled: true,
      interval: options.pageHidden ? HIDDEN_IDLE_LIST_POLL_INTERVAL_MS : IDLE_LIST_POLL_INTERVAL_MS,
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
  return merged as TerminalSession;
}

function durationLabel(durationMS: number | undefined) {
  const value = Number(durationMS || 0);
  if (!Number.isFinite(value) || value <= 0) {
    return "<1s";
  }
  const seconds = Math.max(1, Math.round(value / 1000));
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remain = seconds % 60;
  return remain > 0 ? `${minutes}m ${remain}s` : `${minutes}m`;
}

function stepKey(turnID: string, stepID: string) {
  return `${turnID}:${stepID}`;
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

function stepStatusClassName(status: string) {
  const normalized = normalizeText(status).toLowerCase();
  if (["busy", "running", "starting"].includes(normalized)) {
    return "status-running";
  }
  if (["failed", "error"].includes(normalized)) {
    return "status-failed";
  }
  if (["interrupted", "cancelled", "canceled", "exited"].includes(normalized)) {
    return "status-interrupted";
  }
  if (["completed", "success", "ready", "done"].includes(normalized) || !normalized || normalized === "-") {
    return "status-success";
  }
  return "status-neutral";
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
  return copy.lastOutput(formatDateTime(labelAt));
}

function syncJumpState(
  chatScreenNode: HTMLDivElement | null,
  measurementCacheRef: { current: TerminalJumpMeasurement[] | null },
  measurementDirtyRef: { current: boolean },
  suppressNextTurn: boolean,
  setJumpState: (value: JumpState) => void,
) {
  if (!chatScreenNode) {
    measurementCacheRef.current = null;
    measurementDirtyRef.current = true;
    setJumpState({
      previousTurnID: "",
      nextTurnID: "",
      showTop: false,
      showBottom: false,
    });
    return;
  }
  const scrollTop = Math.max(chatScreenNode.scrollTop, 0);
  const viewportBottom = scrollTop + chatScreenNode.clientHeight;
  const remaining = Math.max(chatScreenNode.scrollHeight - scrollTop - chatScreenNode.clientHeight, 0);
  const measureEntries = () =>
    [...chatScreenNode.querySelectorAll<HTMLElement>("[data-terminal-turn]")]
      .map((node) => {
        const id = node.getAttribute("data-terminal-turn") || "";
        const top = node.offsetTop;
        const height = Math.max(node.offsetHeight, 0);
        return {
          id,
          top,
          bottom: top + height,
        };
      })
      .filter((entry) => entry.id);
  let entries = !measurementDirtyRef.current && measurementCacheRef.current
    ? measurementCacheRef.current
    : measureEntries();
  const cachedLastBottom = entries[entries.length - 1]?.bottom ?? 0;
  if (!measurementDirtyRef.current && entries.length > 0 && scrollTop > cachedLastBottom) {
    entries = measureEntries();
    measurementCacheRef.current = entries;
  }
  if (measurementDirtyRef.current) {
    measurementCacheRef.current = entries;
    measurementDirtyRef.current = false;
  }
  if (!entries.length) {
    setJumpState({
      previousTurnID: "",
      nextTurnID: "",
      showTop: scrollTop > JUMP_TOP_THRESHOLD,
      showBottom: remaining > JUMP_BOTTOM_THRESHOLD,
    });
    return;
  }
  let visibleEntries = entries.filter((entry) => entry.bottom > scrollTop && entry.top < viewportBottom);
  if (!visibleEntries.length && !measurementDirtyRef.current) {
    const remeasuredEntries = measureEntries();
    if (remeasuredEntries.length > 0) {
      entries = remeasuredEntries;
      measurementCacheRef.current = remeasuredEntries;
      visibleEntries = remeasuredEntries.filter((entry) => entry.bottom > scrollTop && entry.top < viewportBottom);
    }
  }
  if (!visibleEntries.length) {
    setJumpState({
      previousTurnID: "",
      nextTurnID: "",
      showTop: scrollTop > JUMP_TOP_THRESHOLD,
      showBottom: remaining > JUMP_BOTTOM_THRESHOLD,
    });
    return;
  }
  const previousTurnID = visibleEntries[0]?.id || "";
  let nextTurnID = "";
  const lastVisibleID = visibleEntries[visibleEntries.length - 1]?.id || "";
  const lastVisibleIndex = lastVisibleID
    ? entries.findIndex((entry) => entry.id === lastVisibleID)
    : -1;
  if (suppressNextTurn) {
    nextTurnID = "";
  } else if (remaining <= SCROLL_BOTTOM_ANCHOR_THRESHOLD) {
    nextTurnID = "";
  } else if (lastVisibleIndex >= 0 && lastVisibleIndex === entries.length - 1) {
    nextTurnID = "";
  } else if (visibleEntries.length > 1) {
    nextTurnID = lastVisibleID;
  } else {
    const visibleID = visibleEntries[0]?.id || "";
    const visibleIndex = entries.findIndex((entry) => entry.id === visibleID);
    nextTurnID = visibleIndex >= 0 ? entries[visibleIndex + 1]?.id || "" : "";
  }
  setJumpState({
    previousTurnID,
    nextTurnID,
    showTop: scrollTop > JUMP_TOP_THRESHOLD,
    showBottom: remaining > JUMP_BOTTOM_THRESHOLD,
  });
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
  const [sessions, setSessions] = useState<TerminalSession[]>([]);
  const [activeSessionID, setActiveSessionID] = useState("");
  const [metaOpen, setMetaOpen] = useState(false);
  const [sessionDetailsOpen, setSessionDetailsOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [deletingSessionID, setDeletingSessionID] = useState("");
  const [loading, setLoading] = useState(true);
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
  const [expandedSteps, setExpandedSteps] = useState<Record<string, boolean>>({});
  const [stepDetails, setStepDetails] = useState<Record<string, TerminalStepDetail>>({});
  const [stepErrors, setStepErrors] = useState<Record<string, string>>({});
  const [jumpState, setJumpState] = useState<JumpState>({
    previousTurnID: "",
    nextTurnID: "",
    showTop: false,
    showBottom: false,
  });
  const chatScreenRef = useRef<HTMLDivElement | null>(null);
  const composerInputRef = useRef<HTMLTextAreaElement | null>(null);
  const composerFileInputRef = useRef<HTMLInputElement | null>(null);
  const composerShellRef = useRef<HTMLElement | null>(null);
  const workspaceBodyRef = useRef<HTMLDivElement | null>(null);
  const scrollIdleTimerRef = useRef<number | null>(null);
  const jumpSyncFrameRef = useRef<number | null>(null);
  const jumpMeasurementCacheRef = useRef<TerminalJumpMeasurement[] | null>(null);
  const jumpMeasurementDirtyRef = useRef(true);
  const submittingRef = useRef(false);
  const scrollRestoreSnapshotRef = useRef<{
    top: number;
    anchoredToBottom: boolean;
  } | null>(null);
  const draftPersistTimerRef = useRef<number | null>(null);
  const mobileSubmitGestureLockRef = useRef(false);
  const groupedSessions = useMemo(
    () => groupSessionListItems(sessions, {
      language,
      getTimestamp: (session) =>
        parseTimestamp(session.updated_at) || parseTimestamp(session.last_output_at) || parseTimestamp(session.created_at),
    }),
    [language, sessions],
  );

  const activeSession = sessions.find((session) => session.id === activeSessionID) || null;
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

  const handleComposerPointerDownCapture = (event: PointerEvent<HTMLTextAreaElement>) => {
    if (!workbench.isMobileViewport || event.pointerType === "mouse" || inputFocused) {
      return;
    }
    event.preventDefault();
    focusComposerInputWithoutScroll();
  };

  const handleComposerTouchStartCapture = (event: TouchEvent<HTMLTextAreaElement>) => {
    if (!workbench.isMobileViewport || inputFocused) {
      return;
    }
    event.preventDefault();
    focusComposerInputWithoutScroll();
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

  const scheduleJumpStateSync = () => {
    if (jumpSyncFrameRef.current !== null) {
      return;
    }
    jumpSyncFrameRef.current = window.requestAnimationFrame(() => {
      jumpSyncFrameRef.current = null;
      syncJumpState(
        chatScreenRef.current,
        jumpMeasurementCacheRef,
        jumpMeasurementDirtyRef,
        submittingRef.current,
        setJumpState,
      );
    });
  };

  const refreshList = async () => {
    const payload = await apiClient.get<TerminalSessionsResponse>("/api/terminal/sessions");
    const nextSessions = Array.isArray(payload.items) ? payload.items : [];
    setSessions((current) => {
      const currentMap = new Map(current.map((session) => [session.id, session]));
      return sortSessions(
        nextSessions.map((session) => mergeSessionSnapshot(currentMap.get(session.id), session)),
      );
    });
    setActiveSessionID((current) => {
      if (nextSessions.some((session) => session.id === current)) {
        return current;
      }
      return nextSessions[0]?.id || "";
    });
    return nextSessions;
  };

  const refreshActiveSession = async (sessionID: string) => {
    if (!sessionID) {
      return null;
    }
    const payload = await apiClient.get<TerminalSessionResponse>(
      `/api/terminal/sessions/${encodeURIComponent(sessionID)}`,
    );
    const nextSession = payload.session || null;
    if (!nextSession) {
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
    workbench.closeMobileSessionPane();
    setMetaOpen(false);
    setExpandedTurns({});
    setExpandedSteps({});
    setStepDetails({});
    setStepErrors({});
    return nextSession;
  };

  useEffect(() => {
    const observer = new MutationObserver(() => setLanguage(resolveLanguage()));
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["lang"],
    });
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const handleVisibilityChange = () => setPageHidden(document.hidden);
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
  }, []);

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
    if (!activeSessionID) {
      return;
    }
    void refreshActiveSession(activeSessionID);
  }, [activeSessionID]);

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
        void refreshActiveSession(activeSessionID);
      }
    }, pollPlan.interval);
    return () => window.clearTimeout(timer);
  }, [activeSession, activeSessionID, pollPlan.enabled, pollPlan.interval, pollPlan.refreshActiveSession, sessions]);

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
    jumpMeasurementDirtyRef.current = true;
    restoreScrollSnapshot();
    scheduleJumpStateSync();
  }, [activeSessionID, turns, expandedTurns, expandedSteps, stepDetails, metaOpen]);

  useEffect(() => {
    const handleResize = () => {
      jumpMeasurementDirtyRef.current = true;
      scheduleJumpStateSync();
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  useEffect(() => {
    return () => {
      if (scrollIdleTimerRef.current !== null) {
        window.clearTimeout(scrollIdleTimerRef.current);
      }
      if (jumpSyncFrameRef.current !== null) {
        window.cancelAnimationFrame(jumpSyncFrameRef.current);
      }
      if (draftPersistTimerRef.current !== null) {
        window.clearTimeout(draftPersistTimerRef.current);
      }
    };
  }, []);

  const selectSession = async (sessionID: string) => {
    setActiveSessionID(sessionID);
    workbench.closeMobileSessionPane();
    setMetaOpen(false);
    setExpandedTurns({});
    setExpandedSteps({});
    setStepDetails({});
    setStepErrors({});
    await refreshActiveSession(sessionID);
  };

  const deleteSession = async (sessionID: string) => {
    if (!window.confirm(copy.deleteConfirm)) {
      return;
    }
    setDeletingSessionID(sessionID);
    try {
      await apiClient.delete(`/api/terminal/sessions/${encodeURIComponent(sessionID)}`);
      setSessions((current) => {
        const next = current.filter((session) => session.id !== sessionID);
        if (activeSessionID === sessionID) {
          setActiveSessionID(next[0]?.id || "");
        }
        return next;
      });
      window.localStorage.removeItem(`terminal:${sessionID}`);
      clearDraftAttachments(sessionID);
      workbench.closeMobileSessionPane();
    } finally {
      setDeletingSessionID("");
    }
  };

  const handleComposerAttachmentPicker = () => {
    composerFileInputRef.current?.click();
  };

  const handleComposerAttachmentSelection = async (files: FileList | null) => {
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

  const submitInput = async () => {
    const content = inputValue.trim();
    const draftKey = activeDraftKey;
    if (submitting) {
      return;
    }
    submittingRef.current = true;
    setSubmitting(true);
    setJumpState((current) => current.nextTurnID
      ? { ...current, nextTurnID: "" }
      : current);
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
              item.id === session!.id ? (payload.session as TerminalSession) : item,
            ),
          ),
        );
      }
      await refreshActiveSession(session.id);
      window.requestAnimationFrame(() => {
        const node = chatScreenRef.current;
        if (node) {
          node.scrollTop = node.scrollHeight;
          jumpMeasurementDirtyRef.current = true;
          syncJumpState(
            node,
            jumpMeasurementCacheRef,
            jumpMeasurementDirtyRef,
            submittingRef.current,
            setJumpState,
          );
        }
      });
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  };

  const toggleTurn = (turnID: string) => {
    setExpandedTurns((current) => ({ ...current, [turnID]: !current[turnID] }));
  };

  const toggleStep = async (turnID: string, stepID: string, hasDetail: boolean) => {
    const key = stepKey(turnID, stepID);
    const nextExpanded = !expandedSteps[key];
    setExpandedSteps((current) => ({ ...current, [key]: nextExpanded }));
    if (!nextExpanded || !hasDetail || stepDetails[key] || !activeSession) {
      return;
    }
    try {
      const payload = await apiClient.get<TerminalStepDetailResponse>(
        `/api/terminal/sessions/${encodeURIComponent(activeSession.id)}/turns/${encodeURIComponent(turnID)}/steps/${encodeURIComponent(stepID)}`,
      );
      if (payload.step) {
        setStepDetails((current) => ({ ...current, [key]: payload.step as TerminalStepDetail }));
      }
    } catch (error) {
      setStepErrors((current) => ({
        ...current,
        [key]: error instanceof Error ? error.message : "unknown error",
      }));
    }
  };

  const handleScroll = () => {
    setScrollingActive(true);
    if (scrollIdleTimerRef.current !== null) {
      window.clearTimeout(scrollIdleTimerRef.current);
    }
    scrollIdleTimerRef.current = window.setTimeout(() => {
      setScrollingActive(false);
      scrollIdleTimerRef.current = null;
    }, SCROLL_IDLE_MS);
    scheduleJumpStateSync();
  };

  const handleJumpControlPointerDown = (event: PointerEvent<HTMLButtonElement>) => {
    if (event.pointerType === "mouse") {
      event.preventDefault();
    }
  };

  const scrollToTurn = (turnID: string) => {
    const chatNode = chatScreenRef.current;
    const turnNode = chatNode?.querySelector<HTMLElement>(`[data-terminal-turn="${turnID}"]`) || null;
    if (!chatNode || !turnNode) {
      return;
    }
    chatNode.scrollTo({ top: Math.max(turnNode.offsetTop - 12, 0), behavior: "smooth" });
    scheduleJumpStateSync();
  };

  const activeNote = runtimeNote(activeSession?.status || "", copy);
  const runtimeDetail = String(activeSession?.error_message || "").trim();
  const composerNote = [activeNote, runtimeDetail, composerAttachmentError].filter(Boolean).join(" | ");
  const canInput = !activeSession || activeStatus !== "busy";
  const isWorkspaceLive = activeSession && isLiveStatus(activeSession.status || "") ? "true" : "false";
  const inputPlaceholder = canInput ? copy.inputPlaceholder : copy.busy;
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
                <small>{skill.description}</small>
              </span>
            </label>
          )) : (
            <p className="route-empty-panel">{copy.noSkills}</p>
          )}
        </div>
      </section>
    </div>
  ) : null;
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
        onClick: workbench.closeMobileSessionPane,
        buttonProps: { "data-runtime-session-pane-close": "terminal" },
      },
      sessionPanePrimaryActionClassName: "is-primary",
      sessionPanePrimaryActionProps: { "data-runtime-create-session": "terminal" },
      sessionPaneSecondaryActionClassName: "runtime-workspace-session-pane-close",
      sessionPaneSecondaryActionProps: { "data-runtime-session-pane-close": "terminal" },
      sessionPaneTitle: copy.sessions,
      sessionPaneCountLabel: copy.sessionCount(sessions.length),
      sessionPanePrimaryActionLabel: copy.newShort,
      onSessionPanePrimaryAction: () => void createSession(),
      sessionPaneSecondaryActionLabel: copy.hideSessions,
      onSessionPaneSecondaryAction: workbench.closeMobileSessionPane,
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
      mobileSessionButtonClassName: "is-quiet conversation-mobile-session-toggle",
      mobileSessionButtonLabel: copy.sessions,
      mobileSessionButtonProps: { "aria-expanded": workbench.mobileSessionPaneOpen },
      onMobileSession: workbench.toggleMobileSessionPane,
      mobilePrimaryButtonClassName: "is-primary conversation-mobile-new-session",
      mobilePrimaryButtonLabel: copy.newShort,
      mobilePrimaryButtonProps: { "data-runtime-create-session": "terminal" },
      onMobilePrimary: () => void createSession(),
    },
    sessionList: {
      groups: groupedSessions.map((group) => ({
        ...group,
        items: group.items.map((session) => {
          const active = session.id === activeSessionID;
          return {
            id: session.id,
            active,
            title: normalizeText(session.title || session.id),
            meta: sessionLastOutputLabel(session, copy),
            shortHash: hashSessionIDShort(normalizeText(session.id)),
            activeLabel: copy.current,
            idleLabel: copy.sessionLabel,
            onSelect: () => void selectSession(session.id),
            onDelete: () => void deleteSession(session.id),
            deleteLabel: copy.delete,
            deleteAriaLabel: copy.delete,
            deleting: deletingSessionID === session.id,
            deleteProps: { "data-runtime-delete-session": session.id },
            shellClassName: active ? "runtime-session-card is-active" : "runtime-session-card",
            shellProps: {
              "data-runtime-session-card": session.id,
              "data-runtime-session-status": normalizeStatus(session.status || ""),
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
          {!loadError && !loading && groupedSessions.length === 0 ? (
            <p className="route-empty-panel">{copy.empty}</p>
          ) : null}
        </>
      ),
    },
    header: {
      title: activeSession ? normalizeText(activeSession.title || activeSession.id) : copy.noSession,
      statusLabel: activeSession ? renderStatus(activeSession.status || "", copy) : copy.ready,
      statusTone: activeStatus,
      detailsLabel: copy.details,
      detailsOpen: sessionDetailsOpen,
      onToggleDetails: () => setSessionDetailsOpen((current) => !current),
      detailsDisabled: !activeSession,
      detailsSummary: terminalDetailsSummary,
      detailsBody: null,
      detailsClassName: "runtime-workspace-meta-panel workspace-details-content",
      headerProps: { "data-runtime-header-kind": "terminal" },
      statusButtonProps: { "data-runtime-status-indicator": activeStatus },
      detailsButtonProps: { "data-runtime-details-toggle": "terminal" },
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
      items: buildTerminalTimelineItems({
        turns,
        expandedTurns,
        expandedSteps,
        stepDetails,
        stepErrors,
        copy,
        onToggleTurn: toggleTurn,
        onToggleStep: (turnID, stepID, hasDetail) => void toggleStep(turnID, stepID, hasDetail),
        onPreviewAttachment: setPreviewAttachment,
      }),
      emptyState: !activeSession ? (
        <div className="terminal-log-empty">{loading ? copy.loading : copy.noSession}</div>
      ) : (
        <div className="terminal-log-empty">{loading ? copy.loading : copy.noOutput}</div>
      ),
      overlay: (
        <div className="terminal-jump-cluster" aria-label="Turn navigation">
          <button
            className={jumpState.showTop ? "terminal-jump-control terminal-jump-top is-visible" : "terminal-jump-control terminal-jump-top"}
            type="button"
            data-terminal-jump-top
            aria-label={copy.top}
            title={copy.top}
            onPointerDown={handleJumpControlPointerDown}
            onClick={() => {
              const node = chatScreenRef.current;
              if (node) {
                node.scrollTo({ top: 0, behavior: "smooth" });
              }
            }}
          >
            <span className="terminal-jump-control-icon" aria-hidden="true">↑↑</span>
          </button>
          <button
            className={jumpState.previousTurnID ? "terminal-jump-control terminal-jump-prev is-visible" : "terminal-jump-control terminal-jump-prev"}
            type="button"
            data-terminal-jump-prev
            data-terminal-jump-target={jumpState.previousTurnID}
            aria-label={copy.prev}
            title={copy.prev}
            onPointerDown={handleJumpControlPointerDown}
            onClick={() => scrollToTurn(jumpState.previousTurnID)}
          >
            <span className="terminal-jump-control-icon" aria-hidden="true">↑</span>
          </button>
          <button
            className={jumpState.nextTurnID ? "terminal-jump-control terminal-jump-next is-visible" : "terminal-jump-control terminal-jump-next"}
            type="button"
            data-terminal-jump-next
            data-terminal-jump-target={jumpState.nextTurnID}
            aria-label={copy.next}
            title={copy.next}
            onPointerDown={handleJumpControlPointerDown}
            onClick={() => scrollToTurn(jumpState.nextTurnID)}
          >
            <span className="terminal-jump-control-icon" aria-hidden="true">↓</span>
          </button>
          <button
            className={jumpState.showBottom ? "terminal-jump-control terminal-jump-bottom is-visible" : "terminal-jump-control terminal-jump-bottom"}
            type="button"
            data-terminal-jump-bottom
            aria-label={copy.bottom}
            title={copy.bottom}
            onPointerDown={handleJumpControlPointerDown}
            onClick={() => {
              const node = chatScreenRef.current;
              if (node) {
                node.scrollTo({ top: node.scrollHeight, behavior: "smooth" });
              }
            }}
          >
            <span className="terminal-jump-control-icon" aria-hidden="true">↓↓</span>
          </button>
        </div>
      ),
    },
    composer: {
      runtimeKind: "terminal",
      shellRef: composerShellRef,
      note: composerNote,
      noteProps: { "data-runtime-note": "terminal", "data-runtime-status": activeStatus },
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
      },
      onInputChange: setInputValue,
      onInputFocus: () => setInputFocused(true),
      onInputBlur: () => setInputFocused(false),
      onInputPointerDownCapture: handleComposerPointerDownCapture,
      onInputTouchStartCapture: handleComposerTouchStartCapture,
      utilityButtons: [
        {
          key: "session",
          label: shellCopy.runtimeMobile,
          icon: <RuntimeSessionControlIcon />,
          className: metaOpen ? "is-active" : undefined,
          onClick: () => setMetaOpen((current) => !current),
        },
      ],
      panelContent: terminalConfigPanel,
      panelProps: {
        "data-runtime-config-surface": "terminal",
      },
      metaContent: undefined,
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

function buildTerminalTimelineItems({
  turns,
  expandedTurns,
  expandedSteps,
  stepDetails,
  stepErrors,
  copy,
  onToggleTurn,
  onToggleStep,
  onPreviewAttachment,
}: {
  turns: TerminalTurn[];
  expandedTurns: Record<string, boolean>;
  expandedSteps: Record<string, boolean>;
  stepDetails: Record<string, TerminalStepDetail>;
  stepErrors: Record<string, string>;
  copy: TerminalCopy;
  onToggleTurn: (turnID: string) => void;
  onToggleStep: (turnID: string, stepID: string, hasDetail: boolean) => void;
  onPreviewAttachment: (attachment: ComposerAttachment | null) => void;
}): RuntimeTimelineItem[] {
  return turns.map((turn) => {
    const steps = Array.isArray(turn.steps) ? turn.steps : [];
    const turnAttachments = Array.isArray(turn.attachments) ? turn.attachments : [];
    const imageAttachments = turnAttachments.filter((attachment) => attachment.content_type.startsWith("image/"));
    const processOpen = expandedTurns[turn.id] ?? false;
    const hasProcess = steps.length > 0 || normalizeStatus(turn.status || "") === "busy";
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
        className: "terminal-log-row kind-command terminal-turn-prompt",
        textClassName: "terminal-log-main",
        timeClassName: "terminal-log-time",
        text: turn.prompt,
        timeLabel: formatTimeLabel(turn.started_at || turn.finished_at || Date.now()),
      });
    }

    if (hasProcess) {
      const processSteps: RuntimeTimelineProcessStep[] = steps.map((step) => {
        const key = stepKey(turn.id, step.id);
        const detail = stepDetails[key];
        const error = stepErrors[key];
        const expanded = Boolean(expandedSteps[key]);
        const fallbackContent = String(step.preview || "").trim();
        return {
          id: step.id,
          itemClassName: "terminal-step-item",
          itemProps: { "data-terminal-step-item": step.id },
          toggleClassName: "terminal-step-toggle",
          toggleProps: {
            "data-terminal-step-toggle": step.id,
            onClick: () => onToggleStep(turn.id, step.id, Boolean(step.has_detail)),
          },
          title: normalizeText(step.preview || step.title || step.type),
          meta: (
            <span className="terminal-step-meta">
              <span className="terminal-step-duration">
                {durationLabel(step.duration_ms)}
              </span>
              <span className={`terminal-step-status ${stepStatusClassName(step.status || "")}`}>
                {renderStatus(step.status || "", copy)}
              </span>
            </span>
          ),
          expanded,
          onToggle: () => onToggleStep(turn.id, step.id, Boolean(step.has_detail)),
          bodyClassName: "terminal-step-body",
          detail: (
            <div className="terminal-step-detail">
              {error ? <div className="terminal-step-detail-state is-error">{error}</div> : null}
              {!error && detail?.blocks?.map((block, index) => {
                const blockType = String(block.type || "text").trim().toLowerCase();
                const blockTitle = String(block.title || "").trim();
                const blockFile = String(block.file || "").trim();
                const blockStatus = String(block.status || "").trim();
                const content = String(block.content || "");
                return (
                  <section
                    key={`${step.id}-${index}`}
                    className={`route-surface-dark terminal-rich-block type-${blockType || "text"}`}
                  >
                    {blockTitle || blockFile || blockStatus ? (
                      <div className="terminal-rich-head">
                        <div className="terminal-rich-copy">
                          {blockTitle ? <strong>{blockTitle}</strong> : null}
                          {blockFile ? (
                            <span>
                              {blockFile}
                              {block.start_line ? `:${block.start_line}` : ""}
                            </span>
                          ) : null}
                        </div>
                        {blockStatus ? (
                          <div className="terminal-rich-meta">
                            <span className={`terminal-step-status ${stepStatusClassName(blockStatus)}`}>
                              {renderStatus(blockStatus, copy)}
                            </span>
                          </div>
                        ) : null}
                      </div>
                    ) : null}
                    <pre className={`terminal-rich-pre terminal-step-content${blockType === "diff" ? " terminal-diff-block" : ""}`}>
                      <code>{content}</code>
                    </pre>
                  </section>
                );
              })}
              {!error && !detail?.blocks?.length && fallbackContent ? (
                <section className="route-surface-dark terminal-rich-block type-terminal">
                  <pre className="terminal-rich-pre terminal-step-content">
                    <code>{fallbackContent}</code>
                  </pre>
                </section>
              ) : null}
              {!error && !detail?.blocks?.length && !fallbackContent && !step.has_detail ? (
                <div className="terminal-step-detail-state">{copy.noProcess}</div>
              ) : null}
            </div>
          ),
        };
      });

      blocks.push({
        type: "process" as const,
        shellClassName: `terminal-process-shell${processOpen ? "" : " is-collapsed"}`,
        shellProps: { "data-terminal-process-shell": turn.id },
        toggleClassName: "terminal-process-toggle",
        toggleProps: { "data-terminal-process-toggle": turn.id },
        title: (
          <>
            <span className="terminal-step-toggle-icon" aria-hidden="true">
              {processOpen ? "v" : ">"}
            </span>
            <span className="terminal-process-copy">
              <span className="terminal-process-title">{copy.process}</span>
              <span className="terminal-process-summary">{copy.processSteps(steps.length)}</span>
            </span>
          </>
        ),
        meta: <span className="terminal-process-meta">{durationLabel(turn.duration_ms)}</span>,
        expanded: processOpen,
        onToggle: () => onToggleTurn(turn.id),
        bodyClassName: "terminal-process-body",
        emptyState: (
          <div className="terminal-process-empty">
            {normalizeStatus(turn.status || "") === "busy" ? copy.loading : copy.noProcess}
          </div>
        ),
        steps: processSteps,
      });
    }

    if (normalizeText(turn.final_output) !== "-") {
      blocks.push({
        type: "markdown-shell" as const,
        html: renderRuntimeMarkdownToHTML(turn.final_output || ""),
        copyValue: turn.final_output,
        copyLabel: copy.copy,
        wrapperClassName: "msg assistant terminal-final-output terminal-turn-output",
        wrapperProps: { "data-terminal-final-output": turn.id },
        bubbleClassName: "msg-bubble",
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
