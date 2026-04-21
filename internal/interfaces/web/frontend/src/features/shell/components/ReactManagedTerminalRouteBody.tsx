import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useWorkbenchContext } from "../../../app/WorkbenchContext";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTime, formatTimeLabel } from "../../../shared/time/format";
import { getLegacyShellCopy } from "../legacyShellCopy";
import { normalizeText, RouteFieldRow } from "./RouteBodyPrimitives";

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
  status: string;
  started_at?: string | number;
  finished_at?: string | number;
  duration_ms?: number;
  final_output?: string;
  steps?: TerminalStepSummary[];
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

type TerminalStepDetail = {
  turn_id?: string;
  step?: TerminalStepSummary;
  blocks?: TerminalStepBlock[];
  searchable?: boolean;
};

type TerminalStepDetailResponse = {
  step?: TerminalStepDetail;
};

type TerminalCopy = {
  sessions: string;
  sessionCount: (count: number) => string;
  newShort: string;
  hideSessions: string;
  empty: string;
  ready: string;
  busy: string;
  exited: string;
  failed: string;
  interrupted: string;
  close: string;
  delete: string;
  deleteConfirm: string;
  inputPlaceholder: string;
  send: string;
  sending: string;
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
  copy: string;
};

const TERMINAL_COPY: Record<"en" | "zh", TerminalCopy> = {
  en: {
    sessions: "Sessions",
    sessionCount: (count) => `${count} sessions`,
    newShort: "New",
    hideSessions: "Hide sessions",
    empty: "No terminal sessions yet.",
    ready: "Ready",
    busy: "Busy",
    exited: "Exited",
    failed: "Failed",
    interrupted: "Interrupted",
    close: "Close",
    delete: "Delete",
    deleteConfirm: "Delete this terminal session?",
    inputPlaceholder: "Type command or prompt...",
    send: "Send",
    sending: "Sending...",
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
    copy: "Copy output",
  },
  zh: {
    sessions: "会话列表",
    sessionCount: (count) => `${count} 个会话`,
    newShort: "新建",
    hideSessions: "收起会话",
    empty: "暂时还没有终端会话。",
    ready: "就绪",
    busy: "执行中",
    exited: "已退出",
    failed: "失败",
    interrupted: "已中断",
    close: "关闭",
    delete: "删除",
    deleteConfirm: "确认删除这个终端会话？",
    inputPlaceholder: "输入命令或继续追问...",
    send: "发送",
    sending: "发送中...",
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
    copy: "复制输出",
  },
};

type JumpState = {
  previousTurnID: string;
  nextTurnID: string;
  showTop: boolean;
  showBottom: boolean;
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

type TerminalPollPlan = {
  enabled: boolean;
  interval: number;
  refreshActiveSession: boolean;
};

function resolveLanguage(): "en" | "zh" {
  return document.documentElement.lang.toLowerCase().startsWith("zh") ? "zh" : "en";
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
  setJumpState: (value: JumpState) => void,
) {
  if (!chatScreenNode) {
    setJumpState({
      previousTurnID: "",
      nextTurnID: "",
      showTop: false,
      showBottom: false,
    });
    return;
  }
  const turnNodes = [...chatScreenNode.querySelectorAll<HTMLElement>("[data-terminal-turn]")];
  const entries = turnNodes
    .map((node) => ({
      id: node.getAttribute("data-terminal-turn") || "",
      top: node.offsetTop,
    }))
    .filter((entry) => entry.id);
  const scrollTop = Math.max(chatScreenNode.scrollTop, 0);
  const remaining = Math.max(chatScreenNode.scrollHeight - scrollTop - chatScreenNode.clientHeight, 0);
  if (!entries.length) {
    setJumpState({
      previousTurnID: "",
      nextTurnID: "",
      showTop: scrollTop > JUMP_TOP_THRESHOLD,
      showBottom: remaining > JUMP_BOTTOM_THRESHOLD,
    });
    return;
  }
  const viewportAnchor = scrollTop + 24;
  let currentIndex = 0;
  for (let index = 0; index < entries.length; index += 1) {
    const entry = entries[index];
    const next = entries[index + 1];
    if (viewportAnchor < entry.top) {
      currentIndex = Math.max(index - 1, 0);
      break;
    }
    currentIndex = index;
    if (!next || viewportAnchor < next.top) {
      break;
    }
  }
  setJumpState({
    previousTurnID: currentIndex > 0 ? entries[currentIndex - 1]?.id || "" : "",
    nextTurnID: currentIndex < entries.length - 1 ? entries[currentIndex + 1]?.id || "" : "",
    showTop: scrollTop > JUMP_TOP_THRESHOLD,
    showBottom: remaining > JUMP_BOTTOM_THRESHOLD,
  });
}

function MarkdownHTML({ html }: { html: string }) {
  return <div dangerouslySetInnerHTML={{ __html: html }} />;
}

function renderMarkdownToHTML(value: string) {
  const normalized = String(value ?? "").replace(/\r\n?/g, "\n");
  if (!normalized.trim()) {
    return "";
  }
  const tokens: Array<{ type: "markdown" | "code"; content: string; language?: string }> = [];
  const fencePattern = /```([a-zA-Z0-9_-]+)?\n([\s\S]*?)```/g;
  let cursor = 0;
  let match = fencePattern.exec(normalized);
  while (match) {
    if (match.index > cursor) {
      tokens.push({ type: "markdown", content: normalized.slice(cursor, match.index) });
    }
    tokens.push({
      type: "code",
      language: String(match[1] || "").trim().toLowerCase(),
      content: String(match[2] || "").replace(/\n$/, ""),
    });
    cursor = match.index + match[0].length;
    match = fencePattern.exec(normalized);
  }
  if (cursor < normalized.length) {
    tokens.push({ type: "markdown", content: normalized.slice(cursor) });
  }
  return tokens
    .map((token) => {
      if (token.type === "code") {
        const languageClass = token.language ? ` class="language-${escapeHTML(token.language)}"` : "";
        return `<pre class="chat-md-pre"><code${languageClass}>${escapeHTML(token.content)}</code></pre>`;
      }
      return renderMarkdownBlocks(token.content);
    })
    .join("");
}

function renderMarkdownBlocks(content: string) {
  const lines = String(content || "").split("\n");
  const html: string[] = [];
  let paragraphLines: string[] = [];
  let quoteLines: string[] = [];
  let listType = "";
  let listItems: string[] = [];

  const flushParagraph = () => {
    if (!paragraphLines.length) {
      return;
    }
    html.push(`<p>${paragraphLines.map((line) => renderMarkdownInline(line)).join("<br>")}</p>`);
    paragraphLines = [];
  };

  const flushQuote = () => {
    if (!quoteLines.length) {
      return;
    }
    html.push(`<blockquote>${renderMarkdownBlocks(quoteLines.join("\n"))}</blockquote>`);
    quoteLines = [];
  };

  const flushList = () => {
    if (!listType || !listItems.length) {
      listType = "";
      listItems = [];
      return;
    }
    html.push(
      `<${listType}>${listItems
        .map((item) => `<li>${renderMarkdownInline(item)}</li>`)
        .join("")}</${listType}>`,
    );
    listType = "";
    listItems = [];
  };

  const flushAll = () => {
    flushParagraph();
    flushQuote();
    flushList();
  };

  for (const rawLine of lines) {
    const trimmed = rawLine.trim();
    if (!trimmed) {
      flushAll();
      continue;
    }

    if (/^>\s?/.test(trimmed)) {
      flushParagraph();
      flushList();
      quoteLines.push(trimmed.replace(/^>\s?/, ""));
      continue;
    }
    flushQuote();

    const unorderedMatch = /^[-*+]\s+(.+)$/.exec(trimmed);
    if (unorderedMatch) {
      flushParagraph();
      if (listType && listType !== "ul") {
        flushList();
      }
      listType = "ul";
      listItems.push(unorderedMatch[1]);
      continue;
    }

    const orderedMatch = /^(\d+)\.\s+(.+)$/.exec(trimmed);
    if (orderedMatch) {
      flushParagraph();
      if (listType && listType !== "ol") {
        flushList();
      }
      listType = "ol";
      listItems.push(orderedMatch[2]);
      continue;
    }

    flushList();

    const headingMatch = /^(#{1,6})\s+(.+)$/.exec(trimmed);
    if (headingMatch) {
      flushParagraph();
      const level = headingMatch[1].length;
      html.push(`<h${level}>${renderMarkdownInline(headingMatch[2])}</h${level}>`);
      continue;
    }

    if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph();
      html.push("<hr>");
      continue;
    }

    paragraphLines.push(trimmed);
  }

  flushAll();
  return html.join("");
}

function renderMarkdownInline(content: string) {
  let rendered = escapeHTML(String(content ?? ""));
  const placeholders: string[] = [];
  const reserve = (html: string) => {
    const token = `\u0000${placeholders.length}\u0000`;
    placeholders.push(html);
    return token;
  };

  rendered = rendered.replace(/`([^`]+)`/g, (_, code: string) =>
    reserve(`<code class="chat-md-inline-code">${escapeHTML(code)}</code>`),
  );
  rendered = rendered.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label: string, url: string) => {
    const href = sanitizeMarkdownURL(url);
    if (!href) {
      return renderMarkdownInline(label);
    }
    return reserve(
      `<a href="${href}" target="_blank" rel="noreferrer noopener">${renderMarkdownInline(label)}</a>`,
    );
  });
  rendered = rendered.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  rendered = rendered.replace(/__([^_]+)__/g, "<strong>$1</strong>");
  rendered = rendered.replace(/(^|[\s(>])\*([^*\n]+)\*(?=$|[\s).,!?:;<])/g, "$1<em>$2</em>");
  rendered = rendered.replace(/(^|[\s(>])_([^_\n]+)_(?=$|[\s).,!?:;<])/g, "$1<em>$2</em>");

  return rendered.replace(/\u0000(\d+)\u0000/g, (_, index: string) => placeholders[Number(index)] || "");
}

function sanitizeMarkdownURL(rawURL: string) {
  const value = String(rawURL || "").trim();
  if (!value) {
    return "";
  }
  const normalized = value.replace(/^<|>$/g, "");
  if (/^(https?:|mailto:)/i.test(normalized) || normalized.startsWith("/") || normalized.startsWith("#")) {
    return escapeHTML(normalized);
  }
  return "";
}

function escapeHTML(value: string) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => {
    if (char === "&") return "&amp;";
    if (char === "<") return "&lt;";
    if (char === ">") return "&gt;";
    if (char === '"') return "&quot;";
    return "&#39;";
  });
}

function CopyIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="12" height="12" rx="2"></rect>
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
    </svg>
  );
}

export function ReactManagedTerminalRouteBody() {
  const workbench = useWorkbenchContext();
  const apiClient = useMemo(() => createAPIClient(), []);
  const [language, setLanguage] = useState<"en" | "zh">(() => resolveLanguage());
  const copy = TERMINAL_COPY[language];
  const shellCopy = getLegacyShellCopy(workbench.language);
  const [sessions, setSessions] = useState<TerminalSession[]>([]);
  const [activeSessionID, setActiveSessionID] = useState("");
  const [sessionSheetOpen, setSessionSheetOpen] = useState(false);
  const [metaOpen, setMetaOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [closing, setClosing] = useState(false);
  const [deletingSessionID, setDeletingSessionID] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [inputValue, setInputValue] = useState("");
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
  const scrollIdleTimerRef = useRef<number | null>(null);
  const jumpSyncFrameRef = useRef<number | null>(null);
  const scrollRestoreSnapshotRef = useRef<{
    top: number;
    anchoredToBottom: boolean;
  } | null>(null);
  const draftPersistTimerRef = useRef<number | null>(null);

  const activeSession = sessions.find((session) => session.id === activeSessionID) || null;
  const turns = Array.isArray(activeSession?.turns) ? activeSession.turns : [];
  const activeStatus = normalizeStatus(activeSession?.status || "");
  const pollPlan = resolveTerminalPollPlan({
    status: activeSession?.status || "",
    pageHidden,
    scrollingActive,
    inputFocused,
  });

  useEffect(() => {
    if (!workbench.isMobileViewport || workbench.mobileNavOpen) {
      setSessionSheetOpen(false);
    }
  }, [workbench.isMobileViewport, workbench.mobileNavOpen]);

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
      syncJumpState(chatScreenRef.current, setJumpState);
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
    setSessionSheetOpen(false);
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
    restoreScrollSnapshot();
    scheduleJumpStateSync();
  }, [activeSessionID, turns, expandedTurns, expandedSteps, stepDetails, metaOpen]);

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
    setSessionSheetOpen(false);
    setMetaOpen(false);
    setExpandedTurns({});
    setExpandedSteps({});
    setStepDetails({});
    setStepErrors({});
    await refreshActiveSession(sessionID);
  };

  const closeSession = async () => {
    if (!activeSession || closing) {
      return;
    }
    setClosing(true);
    try {
      const payload = await apiClient.post<TerminalSessionResponse>(
        `/api/terminal/sessions/${encodeURIComponent(activeSession.id)}/close`,
        {},
      );
      if (payload.session) {
        setSessions((current) =>
          sortSessions(
            current.map((session) =>
              session.id === activeSession.id ? (payload.session as TerminalSession) : session,
            ),
          ),
        );
      }
    } finally {
      setClosing(false);
    }
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
    } finally {
      setDeletingSessionID("");
    }
  };

  const submitInput = async () => {
    const content = inputValue.trim();
    if (!content) {
      return;
    }
    let session = activeSession;
    if (!session) {
      session = await createSession();
    }
    if (!session) {
      return;
    }
    setSubmitting(true);
    try {
      const payload = await apiClient.post<TerminalSessionResponse>(
        `/api/terminal/sessions/${encodeURIComponent(session.id)}/input`,
        { input: content },
      );
      window.localStorage.removeItem(`terminal:${session.id}`);
      setInputValue("");
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
          syncJumpState(node, setJumpState);
        }
      });
    } finally {
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
  const composerNote = [activeNote, runtimeDetail].filter(Boolean).join(" | ");
  const canClose = Boolean(activeSession && isLiveStatus(activeSession.status || ""));
  const canInput = !activeSession || activeStatus !== "busy";
  const isWorkspaceLive = canClose ? "true" : "false";
  const inputPlaceholder = canInput ? copy.inputPlaceholder : copy.busy;

  return (
    <section className="terminal-view conversation-runtime-view" data-terminal-view>
      {workbench.isMobileViewport ? (
        <header className="terminal-mobile-header" data-terminal-mobile-header>
          <button
            className="nav-toggle conversation-mobile-action terminal-inline-button is-quiet"
            type="button"
            aria-expanded={workbench.mobileNavOpen}
            onClick={() => {
              setSessionSheetOpen(false);
              workbench.toggleMobileNav();
            }}
          >
            {shellCopy.chatMenu}
          </button>
          <div className="terminal-mobile-header-actions">
            <button
              className="panel-toggle conversation-mobile-action terminal-inline-button is-quiet"
              type="button"
              aria-expanded={sessionSheetOpen}
              onClick={() => {
                if (!sessionSheetOpen) {
                  workbench.closeMobileNav();
                }
                setSessionSheetOpen((current) => !current);
              }}
            >
              {copy.sessions}
            </button>
            <button
              className="mobile-new-chat conversation-mobile-action terminal-inline-button is-primary"
              type="button"
              onClick={() => void createSession()}
            >
              {copy.newShort}
            </button>
          </div>
        </header>
      ) : null}
      <aside
        className={`terminal-session-pane conversation-session-pane${sessionSheetOpen ? " is-open" : ""}`}
        data-terminal-session-pane
      >
        <button
          className="terminal-session-pane-backdrop conversation-session-pane-backdrop"
          type="button"
          data-terminal-session-pane-close
          aria-label={copy.hideSessions}
          onClick={() => setSessionSheetOpen(false)}
        ></button>
        <div className="route-surface terminal-session-pane-shell conversation-session-pane-shell">
          <div className="terminal-session-pane-head conversation-session-pane-head">
            <div className="terminal-session-pane-copy conversation-session-pane-copy">
              <strong>{copy.sessions}</strong>
              <span>{copy.sessionCount(sessions.length)}</span>
            </div>
            <div className="terminal-session-pane-actions conversation-session-pane-actions">
              <button
                className="terminal-session-pane-action conversation-session-pane-action is-primary"
                type="button"
                data-terminal-create
                onClick={() => void createSession()}
              >
                {copy.newShort}
              </button>
              <button
                className="terminal-session-pane-action conversation-session-pane-action terminal-session-pane-close"
                type="button"
                data-terminal-session-pane-close
                onClick={() => setSessionSheetOpen(false)}
              >
                {copy.hideSessions}
              </button>
            </div>
          </div>
          <div className="terminal-session-list" data-terminal-session-list>
            {loadError ? <p className="route-empty-panel">{loadError}</p> : null}
            {!loadError && !loading && sessions.length === 0 ? (
              <p className="route-empty-panel terminal-session-empty">{copy.empty}</p>
            ) : null}
            {sessions.map((session) => {
              const active = session.id === activeSessionID;
              return (
                <div
                  key={session.id}
                  className={`route-card terminal-session-card${active ? " active" : ""}`}
                  data-terminal-session-card={session.id}
                  data-terminal-session-status={normalizeStatus(session.status || "")}
                >
                  <button
                    className={`route-card-button terminal-session-select${active ? " active" : ""}`}
                    type="button"
                    data-terminal-session-select={session.id}
                    aria-current={active ? "true" : undefined}
                    onClick={() => void selectSession(session.id)}
                  >
                    <span className="terminal-session-head">
                      <span className="route-card-title-copy">
                        <span className="terminal-session-title">
                          {normalizeText(session.title || session.id)}
                        </span>
                        <span className="terminal-session-meta">
                          {sessionLastOutputLabel(session, copy)}
                        </span>
                      </span>
                      <span className={`task-summary-status ${sessionStatusClassName(session.status || "")}`}>
                        {renderStatus(session.status || "", copy)}
                      </span>
                    </span>
                  </button>
                  <button
                    className="terminal-session-list-delete"
                    type="button"
                    data-terminal-delete-session={session.id}
                    aria-label={copy.delete}
                    disabled={deletingSessionID === session.id}
                    onClick={() => void deleteSession(session.id)}
                  >
                    {copy.delete}
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      </aside>

      <section
        className="terminal-workspace conversation-workspace"
        data-terminal-workspace
        data-terminal-session-id={activeSession?.id || ""}
        data-terminal-workspace-status={activeStatus}
        data-terminal-workspace-live={isWorkspaceLive}
      >
        <div className="terminal-workspace-body conversation-workspace-body">
          <header className="terminal-workspace-head conversation-workspace-head is-compact">
            <div className="terminal-workspace-row terminal-workspace-title-row conversation-workspace-row is-compact">
              <div className="terminal-workspace-copy conversation-workspace-copy is-compact">
                <span className="terminal-workspace-eyebrow conversation-workspace-eyebrow">{copy.sessionRuntime}</span>
                <h4>{activeSession ? normalizeText(activeSession.title || activeSession.id) : copy.noSession}</h4>
                <span className="terminal-workspace-subcopy conversation-workspace-subcopy">
                  {activeSession ? sessionLastOutputLabel(activeSession, copy) : copy.empty}
                </span>
              </div>
              {activeSession ? (
                <div className="terminal-runtime-state" data-terminal-runtime-state={activeStatus}>
                  <span className="terminal-runtime-state-dot"></span>
                  <span className="terminal-runtime-state-text">
                    {renderStatus(activeSession.status || "", copy)}
                  </span>
                </div>
              ) : null}
              <div className="terminal-workspace-actions conversation-workspace-actions">
                <button
                  className="terminal-inline-button is-quiet"
                  type="button"
                  data-terminal-session-pane-toggle
                  aria-expanded={sessionSheetOpen}
                  onClick={() => setSessionSheetOpen((current) => !current)}
                >
                  {copy.sessions}
                </button>
                <button
                  className="terminal-inline-button"
                  type="button"
                  data-terminal-meta-toggle
                  aria-expanded={metaOpen}
                  disabled={!activeSession}
                  onClick={() => setMetaOpen((current) => !current)}
                >
                  {copy.details}
                </button>
                <button
                  className="terminal-session-close"
                  type="button"
                  data-terminal-close
                  disabled={!canClose || closing}
                  onClick={() => void closeSession()}
                >
                  {copy.close}
                </button>
              </div>
            </div>

            {activeSession && metaOpen ? (
              <section className="terminal-meta-panel" data-terminal-meta-panel>
                <RouteFieldRow label={copy.session} value={activeSession.id} copyLabel={copy.session} mono />
                <RouteFieldRow label={copy.shell} value={activeSession.shell} copyLabel={copy.shell} mono />
                <RouteFieldRow label={copy.path} value={activeSession.working_dir} copyLabel={copy.path} mono multiline />
                <RouteFieldRow label={copy.status} value={renderStatus(activeSession.status || "", copy)} copyLabel={copy.status} />
                <RouteFieldRow label={copy.updatedAt} value={formatDateTime(activeSession.updated_at || activeSession.created_at)} copyLabel={copy.updatedAt} />
              </section>
            ) : null}
          </header>

          <section className="terminal-console-panel" data-terminal-console-panel>
            <div
              className="terminal-chat-screen"
              data-terminal-chat-screen
              data-terminal-chat-status={activeStatus}
              ref={chatScreenRef}
              onScroll={handleScroll}
            >
              <div className="terminal-log-tree">
                {!activeSession ? (
                  <div className="terminal-log-empty">{loading ? copy.loading : copy.noSession}</div>
                ) : turns.length === 0 ? (
                  <div className="terminal-log-empty">{loading ? copy.loading : copy.noOutput}</div>
                ) : (
                  turns.map((turn) => {
                      const steps = Array.isArray(turn.steps) ? turn.steps : [];
                      const processOpen = expandedTurns[turn.id] ?? false;
                      const hasProcess = steps.length > 0 || normalizeStatus(turn.status || "") === "busy";
                      return (
                        <article key={turn.id} className="terminal-turn-card" data-terminal-turn={turn.id}>
                          {normalizeText(turn.prompt) !== "-" ? (
                            <div className="terminal-log-row kind-command terminal-turn-prompt">
                              <div className="terminal-log-main">
                                <span className="terminal-log-text">{turn.prompt}</span>
                              </div>
                              <span className="terminal-log-time">
                                {formatTimeLabel(turn.started_at || turn.finished_at || Date.now())}
                              </span>
                            </div>
                          ) : null}

                          {hasProcess || normalizeText(turn.final_output) !== "-" ? (
                            <div className="terminal-turn-surface">
                              {hasProcess ? (
                                <section
                                  className={`terminal-process-shell${processOpen ? "" : " is-collapsed"}`}
                                  data-terminal-process-shell={turn.id}
                                >
                                  <button
                                    className="terminal-process-toggle"
                                    type="button"
                                    data-terminal-process-toggle={turn.id}
                                    aria-expanded={processOpen}
                                    onClick={() => toggleTurn(turn.id)}
                                  >
                                    <span className="terminal-step-toggle-icon" aria-hidden="true">
                                      {processOpen ? "v" : ">"}
                                    </span>
                                    <span className="terminal-process-copy">
                                      <span className="terminal-process-title">{copy.process}</span>
                                      <span className="terminal-process-summary">
                                        {copy.processSteps(steps.length)}
                                      </span>
                                    </span>
                                    <span className="terminal-process-meta">
                                      {durationLabel(turn.duration_ms)}
                                    </span>
                                  </button>
                                  <div className="terminal-process-body" hidden={!processOpen}>
                                    {steps.length ? (
                                      steps.map((step) => {
                                        const key = stepKey(turn.id, step.id);
                                        const detail = stepDetails[key];
                                        const error = stepErrors[key];
                                        const expanded = Boolean(expandedSteps[key]);
                                        const fallbackContent = String(step.preview || "").trim();
                                        return (
                                          <article
                                            key={step.id}
                                            className="terminal-step-item"
                                            data-terminal-step-item={step.id}
                                          >
                                            <button
                                              className="terminal-step-toggle"
                                              type="button"
                                              data-terminal-step-toggle={step.id}
                                              aria-expanded={expanded}
                                              onClick={() =>
                                                void toggleStep(turn.id, step.id, Boolean(step.has_detail))
                                              }
                                            >
                                              <span className="terminal-step-toggle-icon" aria-hidden="true">
                                                {expanded ? "v" : ">"}
                                              </span>
                                              <span className="terminal-step-summary">
                                                <span className="terminal-step-title">
                                                  {normalizeText(step.preview || step.title || step.type)}
                                                </span>
                                              </span>
                                              <span className="terminal-step-meta">
                                                <span className="terminal-step-duration">
                                                  {durationLabel(step.duration_ms)}
                                                </span>
                                                <span
                                                  className={`terminal-step-status ${stepStatusClassName(step.status || "")}`}
                                                >
                                                  {renderStatus(step.status || "", copy)}
                                                </span>
                                              </span>
                                            </button>
                                            <div className="terminal-step-body" hidden={!expanded}>
                                              <div className="terminal-step-detail">
                                                {error ? (
                                                  <div className="terminal-step-detail-state is-error">{error}</div>
                                                ) : null}
                                                {!error && detail?.blocks?.map((block, index) => {
                                                  const blockType = String(block.type || "text")
                                                    .trim()
                                                    .toLowerCase();
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
                                                              <span
                                                                className={`terminal-step-status ${stepStatusClassName(blockStatus)}`}
                                                              >
                                                                {renderStatus(blockStatus, copy)}
                                                              </span>
                                                            </div>
                                                          ) : null}
                                                        </div>
                                                      ) : null}
                                                      <pre
                                                        className={`terminal-rich-pre terminal-step-content${blockType === "diff" ? " terminal-diff-block" : ""}`}
                                                      >
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
                                                {!error &&
                                                !detail?.blocks?.length &&
                                                !fallbackContent &&
                                                !step.has_detail ? (
                                                  <div className="terminal-step-detail-state">
                                                    {copy.noProcess}
                                                  </div>
                                                ) : null}
                                              </div>
                                            </div>
                                          </article>
                                        );
                                      })
                                    ) : (
                                      <div className="terminal-process-empty">
                                        {normalizeStatus(turn.status || "") === "busy"
                                          ? copy.loading
                                          : copy.noProcess}
                                      </div>
                                    )}
                                  </div>
                                </section>
                              ) : null}

                              {normalizeText(turn.final_output) !== "-" ? (
                                <div
                                  className="msg assistant terminal-final-output terminal-turn-output"
                                  data-terminal-final-output={turn.id}
                                >
                                  <div className="msg-bubble">
                                    <div className="terminal-final-text">
                                      <div className="assistant-message-toolbar terminal-final-toolbar">
                                        <button
                                          className="route-field-copy assistant-message-copy terminal-final-copy"
                                          type="button"
                                          data-copy-value={turn.final_output}
                                          title={copy.copy}
                                          aria-label={copy.copy}
                                        >
                                          <CopyIcon />
                                        </button>
                                      </div>
                                      <div className="terminal-final-rendered">
                                        <MarkdownHTML html={renderMarkdownToHTML(turn.final_output || "")} />
                                      </div>
                                    </div>
                                  </div>
                                </div>
                              ) : null}
                            </div>
                          ) : null}
                        </article>
                      );
                  })
                )}
              </div>
            </div>

            <div className="terminal-jump-cluster" aria-label="Turn navigation">
              <button
                className={jumpState.showTop ? "terminal-jump-control terminal-jump-top is-visible" : "terminal-jump-control terminal-jump-top"}
                type="button"
                data-terminal-jump-top
                aria-label={copy.top}
                title={copy.top}
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
          </section>

          <footer className="terminal-composer-shell">
            {composerNote ? (
              <div
                className="terminal-composer-note"
                data-terminal-runtime-note
                data-terminal-runtime-status={activeStatus}
              >
                {composerNote}
              </div>
            ) : null}

            <form
              className="terminal-chat-form"
              data-terminal-input-form
              data-composer-form="terminal-runtime"
              onSubmit={(event) => {
                event.preventDefault();
                void submitInput();
              }}
            >
              <label className="sr-only" htmlFor="terminalRuntimeInput">
                {inputPlaceholder}
              </label>
              <textarea
                id="terminalRuntimeInput"
                value={inputValue}
                className="terminal-composer-input"
                placeholder={inputPlaceholder}
                data-terminal-input
                data-composer-input="terminal-runtime"
                disabled={!canInput || submitting}
                onChange={(event) => setInputValue(event.target.value)}
                onFocus={() => setInputFocused(true)}
                onBlur={() => setInputFocused(false)}
              ></textarea>
              <div className="terminal-composer-tools">
                <div
                  className={sessions.length > 0 ? "terminal-composer-meta" : "terminal-composer-meta is-empty"}
                  aria-hidden={sessions.length > 0 ? undefined : "true"}
                >
                  {sessions.length > 0 ? copy.sessionCount(sessions.length) : ""}
                </div>
                <button
                  type="submit"
                  id="terminalSendButton"
                  data-terminal-submit
                  data-composer-submit="terminal-runtime"
                  aria-label={submitting ? copy.sending : copy.send}
                  disabled={submitting || !canInput}
                >
                  <span className="terminal-chat-form-button-icon" aria-hidden="true">
                    <svg viewBox="0 0 20 20" fill="none" focusable="false">
                      <path d="M10 14.75V5.25" stroke="currentColor" strokeWidth="2.35" strokeLinecap="round" />
                      <path d="M5.75 9.5 10 5.25l4.25 4.25" stroke="currentColor" strokeWidth="2.35" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  </span>
                  <span className="sr-only">{submitting ? copy.sending : copy.send}</span>
                </button>
              </div>
            </form>
          </footer>
        </div>
      </section>
    </section>
  );
}
