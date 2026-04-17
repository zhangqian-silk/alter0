import { useEffect, useMemo, useRef, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTime, formatTimeLabel } from "../../../shared/time/format";
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
  },
};

type JumpState = {
  previousTurnID: string;
  nextTurnID: string;
  showTop: boolean;
  showBottom: boolean;
};

const POLL_INTERVAL_MS = 2000;
const JUMP_TOP_THRESHOLD = 180;
const JUMP_BOTTOM_THRESHOLD = 220;

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

export function ReactManagedTerminalRouteBody() {
  const apiClient = useMemo(() => createAPIClient(), []);
  const [language, setLanguage] = useState<"en" | "zh">(() => resolveLanguage());
  const copy = TERMINAL_COPY[language];
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

  const activeSession = sessions.find((session) => session.id === activeSessionID) || null;
  const turns = Array.isArray(activeSession?.turns) ? activeSession.turns : [];

  const refreshList = async () => {
    const payload = await apiClient.get<TerminalSessionsResponse>("/api/terminal/sessions");
    const nextSessions = sortSessions(Array.isArray(payload.items) ? payload.items : []);
    setSessions(nextSessions);
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
        ? current.map((session) => (session.id === sessionID ? nextSession : session))
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
    window.localStorage.setItem(`terminal:${activeSessionID}`, inputValue);
  }, [activeSessionID, inputValue]);

  useEffect(() => {
    if (!activeSessionID) {
      return;
    }
    void refreshActiveSession(activeSessionID);
  }, [activeSessionID]);

  useEffect(() => {
    if (!activeSessionID || !activeSession || !isLiveStatus(activeSession.status || "")) {
      return;
    }
    const timer = window.setTimeout(() => {
      void refreshList();
      void refreshActiveSession(activeSessionID);
    }, POLL_INTERVAL_MS);
    return () => window.clearTimeout(timer);
  }, [activeSession, activeSessionID, sessions]);

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

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      syncJumpState(chatScreenRef.current, setJumpState);
    });
    return () => window.cancelAnimationFrame(frame);
  }, [activeSessionID, turns, expandedTurns, expandedSteps, stepDetails, metaOpen]);

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
    syncJumpState(chatScreenRef.current, setJumpState);
  };

  const scrollToTurn = (turnID: string) => {
    const chatNode = chatScreenRef.current;
    const turnNode = chatNode?.querySelector<HTMLElement>(`[data-terminal-turn="${turnID}"]`) || null;
    if (!chatNode || !turnNode) {
      return;
    }
    chatNode.scrollTo({ top: Math.max(turnNode.offsetTop - 12, 0), behavior: "smooth" });
    window.requestAnimationFrame(() => syncJumpState(chatNode, setJumpState));
  };

  const activeStatus = normalizeStatus(activeSession?.status || "");
  const activeNote = runtimeNote(activeSession?.status || "", copy);
  const canClose = Boolean(activeSession && isLiveStatus(activeSession.status || ""));
  const isWorkspaceLive = canClose ? "true" : "false";

  return (
    <section className="terminal-view" data-terminal-view>
      <aside
        className={`terminal-session-pane${sessionSheetOpen ? " is-open" : ""}`}
        data-terminal-session-pane
      >
        <button
          className="terminal-session-pane-backdrop"
          type="button"
          data-terminal-session-pane-close
          aria-label={copy.hideSessions}
          onClick={() => setSessionSheetOpen(false)}
        ></button>
        <div className="route-surface terminal-session-pane-shell">
          <div className="terminal-session-pane-head">
            <div className="terminal-session-pane-copy">
              <strong>{copy.sessions}</strong>
              <span>{copy.sessionCount(sessions.length)}</span>
            </div>
            <div className="terminal-session-pane-actions">
              <button
                className="terminal-session-pane-action is-primary"
                type="button"
                data-terminal-create
                onClick={() => void createSession()}
              >
                {copy.newShort}
              </button>
              <button
                className="terminal-session-pane-action terminal-session-pane-close"
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
              <p className="route-empty-panel">{copy.empty}</p>
            ) : null}
            {sessions.map((session) => {
              const active = session.id === activeSessionID;
              const statusLabel = renderStatus(session.status || "", copy);
              return (
                <div
                  key={session.id}
                  className={active ? "terminal-session-card active" : "terminal-session-card"}
                >
                  <button
                    className="terminal-session-card-main"
                    type="button"
                    data-terminal-session-select={session.id}
                    onClick={() => void selectSession(session.id)}
                  >
                    <span className="terminal-session-card-title">
                      {normalizeText(session.title || session.id)}
                    </span>
                    <span className="terminal-session-card-meta">{statusLabel}</span>
                    <span className="terminal-session-card-meta">
                      {formatTimeLabel(session.updated_at || session.created_at || Date.now())}
                    </span>
                  </button>
                  <button
                    className="terminal-session-card-delete"
                    type="button"
                    data-terminal-delete-session={session.id}
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
        className="terminal-workspace"
        data-terminal-workspace
        data-terminal-session-id={activeSession?.id || ""}
        data-terminal-workspace-status={activeStatus}
        data-terminal-workspace-live={isWorkspaceLive}
      >
        <div className="terminal-workspace-row terminal-workspace-title-row">
          <div className="terminal-workspace-copy">
            <p className="route-hero-eyebrow">{copy.sessionRuntime}</p>
            <h3>{activeSession ? normalizeText(activeSession.title || activeSession.id) : copy.noSession}</h3>
            <p>{activeSession ? normalizeText(activeSession.id) : copy.empty}</p>
          </div>
          <div className="terminal-workspace-actions">
            <button
              className="pane-action"
              type="button"
              data-terminal-session-pane-toggle
              onClick={() => setSessionSheetOpen((current) => !current)}
            >
              {copy.sessions}
            </button>
            <button
              className="pane-action"
              type="button"
              data-terminal-meta-toggle
              aria-expanded={metaOpen}
              onClick={() => setMetaOpen((current) => !current)}
            >
              {copy.details}
            </button>
            <button
              className="pane-action"
              type="button"
              data-terminal-close
              disabled={!canClose || closing}
              onClick={() => void closeSession()}
            >
              {copy.close}
            </button>
            <button
              className="pane-action"
              type="button"
              data-terminal-delete
              disabled={!activeSession || deletingSessionID === activeSession.id}
              onClick={() => activeSession && void deleteSession(activeSession.id)}
            >
              {copy.delete}
            </button>
          </div>
        </div>

        {activeSession ? (
          <>
            <div className="terminal-workspace-row">
              <div className="terminal-runtime-state" data-terminal-runtime-state={activeStatus}>
                <span className="terminal-runtime-state-dot"></span>
                <span>{renderStatus(activeSession.status || "", copy)}</span>
              </div>
              <span className="context-pill">
                {copy.updatedAt}: {formatTimeLabel(activeSession.updated_at || activeSession.created_at || Date.now())}
              </span>
            </div>

            {activeNote ? (
              <div className="terminal-runtime-note" data-terminal-runtime-note>
                {activeNote}
              </div>
            ) : null}

            {metaOpen ? (
              <div className="route-surface">
                <div className="terminal-workspace-row">
                  <strong>{copy.metadata}</strong>
                </div>
                <RouteFieldRow label={copy.session} value={activeSession.id} copyLabel={copy.session} mono />
                <RouteFieldRow label={copy.shell} value={activeSession.shell} copyLabel={copy.shell} mono />
                <RouteFieldRow label={copy.path} value={activeSession.working_dir} copyLabel={copy.path} mono />
                <RouteFieldRow label={copy.status} value={renderStatus(activeSession.status || "", copy)} copyLabel={copy.status} />
                <RouteFieldRow label={copy.updatedAt} value={formatDateTime(activeSession.updated_at || activeSession.created_at)} copyLabel={copy.updatedAt} />
              </div>
            ) : null}

            <div className="terminal-chat-shell">
              <div
                className="terminal-chat-screen"
                data-terminal-chat-screen
                ref={chatScreenRef}
                onScroll={handleScroll}
              >
                {turns.length === 0 ? (
                  <div className="route-empty-panel">{loading ? copy.loading : copy.noOutput}</div>
                ) : (
                  turns.map((turn) => {
                    const steps = Array.isArray(turn.steps) ? turn.steps : [];
                    const processOpen = expandedTurns[turn.id] ?? false;
                    return (
                      <article key={turn.id} className="terminal-turn-card" data-terminal-turn={turn.id}>
                        <div className="terminal-turn-head">
                          <div className="terminal-turn-prompt-wrap">
                            <strong className="terminal-turn-prompt">{turn.prompt}</strong>
                            <span className="terminal-turn-meta">
                              {renderStatus(turn.status, copy)} · {formatTimeLabel(turn.started_at || turn.finished_at || Date.now())}
                            </span>
                          </div>
                          <span className="context-pill">{durationLabel(turn.duration_ms)}</span>
                        </div>

                        {steps.length ? (
                          <>
                            <button
                              className="terminal-process-toggle"
                              type="button"
                              data-terminal-process-toggle={turn.id}
                              aria-expanded={processOpen}
                              onClick={() => toggleTurn(turn.id)}
                            >
                              <span>{copy.process}</span>
                              <span>{copy.processSteps(steps.length)}</span>
                            </button>
                            <div className="terminal-process-body" hidden={!processOpen}>
                              {steps.map((step) => {
                                const key = stepKey(turn.id, step.id);
                                const detail = stepDetails[key];
                                const error = stepErrors[key];
                                const expanded = Boolean(expandedSteps[key]);
                                return (
                                  <div key={step.id} className="terminal-step-card" data-terminal-step-item={step.id}>
                                    <button
                                      className="terminal-step-toggle"
                                      type="button"
                                      data-terminal-step-toggle={step.id}
                                      aria-expanded={expanded}
                                      onClick={() => void toggleStep(turn.id, step.id, Boolean(step.has_detail))}
                                    >
                                      <span>{step.title || step.type}</span>
                                      <span>{durationLabel(step.duration_ms)}</span>
                                    </button>
                                    <div className="terminal-step-body" hidden={!expanded}>
                                      {step.preview ? (
                                        <p className="terminal-step-preview">
                                          <strong>{copy.preview}</strong>
                                          <span>{step.preview}</span>
                                        </p>
                                      ) : null}
                                      {error ? <p className="session-error">{error}</p> : null}
                                      {detail?.blocks?.map((block, index) => (
                                        <div key={`${step.id}-${index}`} className="terminal-step-block">
                                          {block.title ? <strong>{block.title}</strong> : null}
                                          {block.file ? (
                                            <p className="terminal-step-file">
                                              {block.file}
                                              {block.start_line ? `:${block.start_line}` : ""}
                                            </p>
                                          ) : null}
                                          {block.content ? (
                                            <pre className="terminal-step-content">
                                              <code>{block.content}</code>
                                            </pre>
                                          ) : null}
                                        </div>
                                      ))}
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          </>
                        ) : (
                          <p className="terminal-process-empty">{copy.noProcess}</p>
                        )}

                        {turn.final_output ? (
                          <div className="terminal-final-output" data-terminal-final-output={turn.id}>
                            <pre>{turn.final_output}</pre>
                          </div>
                        ) : null}
                      </article>
                    );
                  })
                )}
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
            </div>
          </>
        ) : (
          <div className="route-empty-panel">{loading ? copy.loading : copy.noSession}</div>
        )}

        <footer className="terminal-composer-shell">
          <form
            className="composer"
            data-terminal-input-form
            data-composer-form="terminal-runtime"
            onSubmit={(event) => {
              event.preventDefault();
              void submitInput();
            }}
          >
            <label className="sr-only" htmlFor="terminalRuntimeInput">
              {copy.inputPlaceholder}
            </label>
            <textarea
              id="terminalRuntimeInput"
              value={inputValue}
              className="terminal-composer-input"
              placeholder={copy.inputPlaceholder}
              data-terminal-input
              data-composer-input="terminal-runtime"
              onChange={(event) => setInputValue(event.target.value)}
            ></textarea>
            <div className="composer-actions">
              <div className="composer-submit-bar">
                <button
                  type="submit"
                  id="terminalSendButton"
                  data-terminal-submit
                  data-composer-submit="terminal-runtime"
                  disabled={submitting}
                >
                  {submitting ? copy.sending : copy.send}
                </button>
              </div>
            </div>
          </form>
        </footer>
      </section>
    </section>
  );
}
