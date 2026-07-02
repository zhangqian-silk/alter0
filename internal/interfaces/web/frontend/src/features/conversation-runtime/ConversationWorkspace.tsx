import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ClipboardEvent, type PointerEvent, type ReactNode, type TouchEvent } from "react";
import { useWorkbenchContext } from "../../app/WorkbenchContext";
import { formatDateTime } from "../../shared/time/format";
import { groupSessionListItems } from "../../shared/time/sessionListGroups";
import { buildRuntimeSessionTimelineItems, type ChatMessageSnapshot } from "../shell/components/ChatMessageRegion";
import {
  buildDraftWithCodexSlashCommand,
  CODEX_SLASH_COMMANDS,
  codexSlashCommandQuery,
} from "../shell/components/codexSlashCommands";
import { conversationMarkdownSyntaxFixture } from "../shell/components/MessageMarkdownSyntaxFixture";
import { normalizeText } from "../shell/components/RouteBodyPrimitives";
import { RUNTIME_EVENT_FILTER_OPTIONS } from "../shell/components/runtimeTraceEvents";
import { RuntimeComposer } from "../shell/components/RuntimeComposer";
import { resolveRuntimeMobileLayoutState } from "../shell/components/runtimeMobileLayout";
import { RuntimeWorkspacePage, type RuntimeWorkspacePageController } from "../shell/components/RuntimeWorkspacePage";
import { ScrollJumpStrip } from "../shell/components/ScrollJumpStrip";
import { useViewportScrollAnchor } from "../shell/components/useViewportScrollAnchor";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../shell/legacyShellCopy";
import {
  isComposerImageAttachment,
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  getPastedComposerImageFiles,
  readComposerFiles,
  type ComposerAttachment,
} from "./composerImageAttachments";
import {
  useConversationRuntimeComposer,
  useConversationRuntimeWorkspace,
} from "./ConversationRuntimeProvider";

type ConversationWorkspaceProps = {
  language: LegacyShellLanguage;
};

type ConversationWorkspaceSharedRefs = {
  timelineScreenRef: { current: HTMLDivElement | null };
  workspaceBodyRef: { current: HTMLDivElement | null };
};

const INITIAL_VISIBLE_CHAT_MESSAGES = 32;
const CHAT_MESSAGE_LOAD_BATCH_SIZE = 32;
const CHAT_HISTORY_AUTO_LOAD_TOP_OFFSET = 32;
const CODEX_RUNTIME_PROVIDER_ID = "alter0-codex";
const CODEX_RUNTIME_MODEL_ID = "codex";
const MARKDOWN_SYNTAX_DEMO_QUERY_KEY = "markdown_demo";

type TimelineViewportAnchor = {
  messageID: string;
  topOffset: number;
};

type TimelineMessageIdentity = {
  id: string;
};

type TimelineRenderWindowSnapshot = {
  sessionID: string;
  messageIDs: string[];
  visibleMessageIDs: string[];
};

const EMPTY_TIMELINE_RENDER_WINDOW: TimelineRenderWindowSnapshot = {
  sessionID: "",
  messageIDs: [],
  visibleMessageIDs: [],
};

function timelineMessageIDs(messages: TimelineMessageIdentity[]): string[] {
  return messages.map((message) => message.id);
}

function isTailPreservingHistoryPrepend(
  previous: TimelineRenderWindowSnapshot,
  sessionID: string,
  messageIDs: string[],
): boolean {
  return Boolean(
    sessionID
    && previous.sessionID === sessionID
    && previous.messageIDs.length > 0
    && messageIDs.length > previous.messageIDs.length
    && previous.messageIDs.every((id, index) =>
      messageIDs[messageIDs.length - previous.messageIDs.length + index] === id,
    ),
  );
}

function resolveVisibleTimelineMessages<T extends TimelineMessageIdentity>(options: {
  messages: T[];
  visibleCount: number;
  previousWindow: TimelineRenderWindowSnapshot;
  sessionID: string;
  historyExpansionRequested: boolean;
}): T[] {
  const { messages, visibleCount, previousWindow, sessionID, historyExpansionRequested } = options;
  const messageIDs = timelineMessageIDs(messages);
  const shouldPreservePreviousWindow =
    isTailPreservingHistoryPrepend(previousWindow, sessionID, messageIDs)
    && !historyExpansionRequested
    && previousWindow.visibleMessageIDs.length > 0;
  if (shouldPreservePreviousWindow) {
    const visibleIDs = new Set(previousWindow.visibleMessageIDs);
    const preserved = messages.filter((message) => visibleIDs.has(message.id));
    if (preserved.length > 0) {
      return preserved;
    }
  }
  return messages.length > visibleCount ? messages.slice(-visibleCount) : messages;
}

function snapshotTimelineRenderWindow(
  sessionID: string,
  messages: TimelineMessageIdentity[],
  visibleMessages: TimelineMessageIdentity[],
): TimelineRenderWindowSnapshot {
  return {
    sessionID,
    messageIDs: timelineMessageIDs(messages),
    visibleMessageIDs: timelineMessageIDs(visibleMessages),
  };
}

function findTimelineMessageElement(container: HTMLElement, messageID: string): HTMLElement | null {
  for (const item of container.querySelectorAll<HTMLElement>("[data-message-id]")) {
    if (item.getAttribute("data-message-id") === messageID) {
      return item;
    }
  }
  return null;
}

function readTimelineViewportAnchor(container: HTMLElement | null): TimelineViewportAnchor | null {
  if (!container) {
    return null;
  }
  const containerRect = container.getBoundingClientRect();
  if (containerRect.height <= 0 && containerRect.bottom <= containerRect.top) {
    return null;
  }
  for (const item of container.querySelectorAll<HTMLElement>("[data-message-id]")) {
    const messageID = item.getAttribute("data-message-id")?.trim() || "";
    if (!messageID) {
      continue;
    }
    const itemRect = item.getBoundingClientRect();
    const itemHeight = itemRect.height || itemRect.bottom - itemRect.top;
    if (itemHeight <= 0) {
      continue;
    }
    if (itemRect.bottom >= containerRect.top && itemRect.top <= containerRect.bottom) {
      return {
        messageID,
        topOffset: itemRect.top - containerRect.top,
      };
    }
  }
  return null;
}

function restoreTimelineViewportAnchor(container: HTMLElement, anchor: TimelineViewportAnchor | null | undefined): boolean {
  if (!anchor?.messageID) {
    return false;
  }
  const item = findTimelineMessageElement(container, anchor.messageID);
  if (!item) {
    return false;
  }
  const containerRect = container.getBoundingClientRect();
  const itemRect = item.getBoundingClientRect();
  const currentTopOffset = itemRect.top - containerRect.top;
  const delta = currentTopOffset - anchor.topOffset;
  if (Math.abs(delta) > 0.5) {
    container.scrollTop += delta;
  }
  return true;
}

function shouldShowConversationMarkdownSyntaxDemo(route: string) {
  if (route !== "chat" || typeof window === "undefined") {
    return false;
  }
  return new URLSearchParams(window.location.search).get(MARKDOWN_SYNTAX_DEMO_QUERY_KEY) === "1";
}

function buildConversationMarkdownSyntaxDemoMessage(language: LegacyShellLanguage): ChatMessageSnapshot {
  return {
    id: "markdown-syntax-demo-assistant",
    role: "assistant",
    text: conversationMarkdownSyntaxFixture.markdown,
    attachments: [],
    route: "chat",
    source: language === "zh" ? "Markdown 语法演示" : "Markdown syntax demo",
    error: false,
    status: "done",
    at: Date.parse("2026-06-08T00:00:00Z"),
    processEvents: [],
  };
}

type ConversationSessionSignalTone = "ready" | "busy" | "failed";

function normalizeConversationSessionMessageStatus(value: string) {
  return normalizeText(value).toLowerCase();
}

function isDirectCodexModelSelection(providerID: string, modelID: string) {
  return normalizeText(providerID) === CODEX_RUNTIME_PROVIDER_ID && normalizeText(modelID) === CODEX_RUNTIME_MODEL_ID;
}

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

function resolveConversationSessionSignalTone(session: {
  status?: string;
  messages?: Array<{
    role?: string;
    status?: string;
    error?: boolean;
  }>;
} | null | undefined): ConversationSessionSignalTone {
  const messages = Array.isArray(session?.messages) ? session.messages : [];
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message?.role !== "assistant") {
      continue;
    }
    const status = normalizeConversationSessionMessageStatus(message.status || "");
    if (message.error || ["error", "failed", "canceled", "cancelled"].includes(status)) {
      return "failed";
    }
    if (["streaming", "queued", "running", "in_progress", "inprogress"].includes(status)) {
      return "busy";
    }
    return "ready";
  }
  const sessionStatus = normalizeConversationSessionMessageStatus(session?.status || "");
  if (["error", "failed", "canceled", "cancelled"].includes(sessionStatus)) {
    return "failed";
  }
  if (["streaming", "queued", "running", "in_progress", "inprogress", "busy"].includes(sessionStatus)) {
    return "busy";
  }
  return "ready";
}

function conversationSessionStatusLabel(
  tone: ConversationSessionSignalTone,
  language: LegacyShellLanguage,
) {
  if (language === "zh") {
    switch (tone) {
      case "busy":
        return "进行中";
      case "failed":
        return "异常";
      default:
        return "就绪";
    }
  }
  switch (tone) {
    case "busy":
      return "Busy";
    case "failed":
      return "Failed";
    default:
      return "Ready";
  }
}

function useConversationWorkspaceController(
  language: LegacyShellLanguage,
  sharedRefs: ConversationWorkspaceSharedRefs,
  composerNode: ReactNode,
  inputFocused: boolean,
): RuntimeWorkspacePageController {
  const workbench = useWorkbenchContext();
  const runtime = useConversationRuntimeWorkspace();
  const copy = getLegacyShellCopy(language);
  const [sessionDetailsOpen, setSessionDetailsOpen] = useState(false);
  const { timelineScreenRef, workspaceBodyRef } = sharedRefs;
  const activeTimelineSessionRef = useRef("");
  const previousActiveMessageCountRef = useRef(0);
  const previousTimelineMessageIDsRef = useRef<string[]>([]);
  const previousTimelineRenderWindowRef = useRef<TimelineRenderWindowSnapshot>(EMPTY_TIMELINE_RENDER_WINDOW);
  const historyExpansionRequestedRef = useRef(false);
  const pendingHistoryScrollRestoreRef = useRef<{
    sessionID: string;
    scrollHeight: number;
    scrollTop: number;
    anchor: TimelineViewportAnchor | null;
  } | null>(null);
  const [timelineWindow, setTimelineWindow] = useState({
    sessionID: "",
    visibleCount: INITIAL_VISIBLE_CHAT_MESSAGES,
  });
  const [expandedProcessEvents, setExpandedProcessSteps] = useState<Record<string, boolean>>({});
  const activeMessages = runtime.activeSession?.messages || [];
  const activeSessionID = runtime.activeSession?.id || "";
  const showMarkdownSyntaxDemo = shouldShowConversationMarkdownSyntaxDemo(runtime.route);
  const timelineMessages = useMemo(
    () => showMarkdownSyntaxDemo ? [buildConversationMarkdownSyntaxDemoMessage(language)] : activeMessages,
    [activeMessages, language, showMarkdownSyntaxDemo],
  );
  const timelineSessionID = showMarkdownSyntaxDemo
    ? `${activeSessionID || "chat"}:markdown-demo`
    : activeSessionID;
  const toggleProcessRef = useRef(runtime.toggleProcess);
  useEffect(() => {
    toggleProcessRef.current = runtime.toggleProcess;
  }, [runtime.toggleProcess]);
  const captureTimelineViewportAnchor = useViewportScrollAnchor({
    active: inputFocused,
    containerRef: timelineScreenRef,
    enabled: workbench.isMobileViewport,
    focusSelector: "[data-composer-input='conversation']",
  });
  const toggleProcess = useCallback((messageID: string) => {
    const processStepKeyPrefix = `${messageID}:`;
    setExpandedProcessSteps((current) => {
      let changed = false;
      const next: Record<string, boolean> = {};
      Object.entries(current).forEach(([key, value]) => {
        if (key.startsWith(processStepKeyPrefix)) {
          changed = true;
          return;
        }
        next[key] = value;
      });
      return changed ? next : current;
    });
    toggleProcessRef.current(messageID);
  }, []);
  const toggleProcessStep = useCallback((messageID: string, stepID: string) => {
    const key = `${messageID}:${stepID}`;
    setExpandedProcessSteps((current) => ({
      ...current,
      [key]: !current[key],
    }));
    if (!expandedProcessEvents[key]) {
      void runtime.loadProcessEventDetail(messageID, stepID);
    }
  }, [expandedProcessEvents, runtime]);
  useEffect(() => {
    setExpandedProcessSteps({});
  }, [timelineSessionID]);
  const visibleMessageCount = timelineWindow.sessionID === timelineSessionID
    ? timelineWindow.visibleCount
    : INITIAL_VISIBLE_CHAT_MESSAGES;
  const previousRenderWindow = previousTimelineRenderWindowRef.current;
  const visibleMessages = resolveVisibleTimelineMessages({
    messages: timelineMessages,
    visibleCount: visibleMessageCount,
    previousWindow: previousRenderWindow,
    sessionID: timelineSessionID,
    historyExpansionRequested: historyExpansionRequestedRef.current,
  });
  const hiddenMessageCount = Math.max(0, timelineMessages.length - visibleMessages.length);
  const hasRemoteEarlierMessages = Boolean(
    runtime.activeSession?.serverBacked === true
    && runtime.activeSession.messagesLoaded === true
    && runtime.activeSession.turnsPaging?.has_more_before === true,
  );
  const isEmptyState = timelineMessages.length === 0;
  const isMobileEmptyHeader = workbench.isMobileViewport && isEmptyState;
  const emptyStateTitle = language === "zh" ? "开始新的工作流" : "Start a new workspace flow";
  const emptyStateDescription = language === "zh"
    ? "对话、过程和交付结果都在同一条时间线里推进。"
    : "Conversation, process, and delivery stay in a single timeline.";
  const sessionPaneTitle = copy.terminalSessions;
  const newSessionLabel = copy.terminalNewShort;
  const sessionCountLabel = language === "zh"
    ? `${runtime.sessionItems.length} 个会话`
    : `${runtime.sessionItems.length} sessions`;
  const runtimeViewAlias = runtime.route === "terminal" ? "terminal" : "conversation";
  const activeSessionBadgeLabel = language === "zh" ? "当前" : "Current";
  const idleSessionBadgeLabel = language === "zh" ? "会话" : "Session";
  const viewSessionDetailsLabel = language === "zh" ? "详情" : "Details";
  const viewSessionDetailsAriaLabel = language === "zh" ? "查看会话详情" : "View session details";
  const deleteSessionLabel = language === "zh" ? "删除" : "Delete";
  const deleteSessionAriaLabel = language === "zh" ? "删除会话" : "Delete session";
  const deleteSessionConfirmLabel = language === "zh" ? "确认删除这个会话？" : "Delete this session?";
  const sessionActionsLabel = language === "zh" ? "会话操作" : "Session actions";
  const pinSessionLabel = language === "zh" ? "置顶" : "Pin";
  const unpinSessionLabel = language === "zh" ? "取消置顶" : "Unpin";
  const pinSessionAriaLabel = language === "zh" ? "置顶会话" : "Pin session";
  const unpinSessionAriaLabel = language === "zh" ? "取消置顶会话" : "Unpin session";
  const groupedSessionItems = useMemo(
    () => groupSessionListItems(runtime.sessionItems, {
      language,
      getTimestamp: (item) => item.createdAt,
      getPinned: (item) => item.pinned,
    }),
    [language, runtime.sessionItems],
  );
  const sessionEmptyLabel = copy.sessionEmpty;
  const compactDetailsLabel = language === "zh" ? "详情" : "Details";
  const activeSessionItem = runtime.sessionItems.find((item) => item.active) || null;
  const activeSessionIsDraft = Boolean(activeSessionItem?.draft);
  const sessionStatusByID = useMemo(
    () => Object.fromEntries(
      runtime.sessions.map((session) => {
        const tone = resolveConversationSessionSignalTone(session);
        return [session.id, {
          tone,
          label: conversationSessionStatusLabel(tone, language),
        }];
      }),
    ) as Record<string, { tone: ConversationSessionSignalTone; label: string }>,
    [language, runtime.sessions],
  );
  const activeSessionStatus = activeSessionItem
    ? sessionStatusByID[activeSessionItem.id] || {
      tone: "ready" as const,
      label: conversationSessionStatusLabel("ready", language),
    }
    : {
      tone: "ready" as const,
      label: conversationSessionStatusLabel("ready", language),
    };
  const handleFocusSession = useCallback((sessionID: string) => {
    runtime.focusSession(sessionID);
    workbench.closeMobileSessionPane();
  }, [runtime, workbench]);
  const handleViewSessionDetails = useCallback((sessionID: string) => {
    runtime.focusSession(sessionID);
    setSessionDetailsOpen(true);
  }, [runtime]);
  const handleRemoveSession = useCallback((sessionID: string) => {
    workbench.closeMobileSessionPane();
    return runtime.removeSession(sessionID);
  }, [runtime, workbench]);
  const handlePinnedChange = useCallback((sessionID: string, pinned: boolean) => {
    void runtime.setSessionPinned(sessionID, pinned);
  }, [runtime]);
  const sessionListGroups = useMemo(
    () => groupedSessionItems.map((group) => ({
      ...group,
      items: group.items.map((item) => {
        const isDraft = Boolean(item.draft);
        return {
          statusTone: sessionStatusByID[item.id]?.tone || "ready",
          statusLabel: sessionStatusByID[item.id]?.label || conversationSessionStatusLabel("ready", language),
          id: item.id,
          active: item.active,
          title: item.title,
          contextLabel: item.contextLabel,
          meta: item.meta,
          shortHash: item.shortHash,
          activeLabel: activeSessionBadgeLabel,
          idleLabel: idleSessionBadgeLabel,
          onSelect: () => handleFocusSession(item.id),
          onViewDetails: isDraft ? undefined : () => handleViewSessionDetails(item.id),
          viewDetailsLabel: isDraft ? undefined : viewSessionDetailsLabel,
          viewDetailsAriaLabel: isDraft ? undefined : viewSessionDetailsAriaLabel,
          pinned: isDraft ? false : item.pinned,
          pinning: isDraft ? false : item.pinning,
          onPinnedChange: isDraft ? undefined : (pinned) => handlePinnedChange(item.id, pinned),
          pinLabel: isDraft ? undefined : pinSessionLabel,
          unpinLabel: isDraft ? undefined : unpinSessionLabel,
          pinAriaLabel: isDraft ? undefined : pinSessionAriaLabel,
          unpinAriaLabel: isDraft ? undefined : unpinSessionAriaLabel,
          onDelete: isDraft ? undefined : () => void handleRemoveSession(item.id),
          deleteLabel: isDraft ? undefined : deleteSessionLabel,
          deleteAriaLabel: isDraft ? undefined : deleteSessionAriaLabel,
          deleteConfirmLabel: isDraft ? undefined : deleteSessionConfirmLabel,
          actionsLabel: isDraft ? undefined : sessionActionsLabel,
          actionsAriaLabel: isDraft ? undefined : sessionActionsLabel,
          shellClassName: item.active ? "runtime-session-card is-active" : "runtime-session-card",
          shellProps: {
            "data-runtime-session-state": item.active ? "active" : "idle",
            "data-runtime-session-card": item.id,
            "data-runtime-session-tone": sessionStatusByID[item.id]?.tone || "ready",
          },
          buttonClassName: item.active ? "runtime-session-select active" : "runtime-session-select",
          buttonProps: { "data-runtime-session-select": item.id },
        };
      }),
    })),
    [
      activeSessionBadgeLabel,
      deleteSessionAriaLabel,
      deleteSessionConfirmLabel,
      deleteSessionLabel,
      groupedSessionItems,
      handleFocusSession,
      handlePinnedChange,
      handleRemoveSession,
      handleViewSessionDetails,
      idleSessionBadgeLabel,
      language,
      pinSessionAriaLabel,
      pinSessionLabel,
      sessionStatusByID,
      sessionActionsLabel,
      unpinSessionAriaLabel,
      unpinSessionLabel,
      viewSessionDetailsAriaLabel,
      viewSessionDetailsLabel,
    ],
  );
  const routeLabel = language === "zh" ? "对话" : "Chat";
  const conversationDetailsSummary = runtime.activeSession ? [
    { label: language === "zh" ? "会话" : "Session", value: runtime.activeSession.id, copyLabel: language === "zh" ? "会话" : "Session", mono: true },
    { label: language === "zh" ? "路由" : "Route", value: routeLabel, copyLabel: language === "zh" ? "路由" : "Route" },
    { label: language === "zh" ? "状态" : "Status", value: activeSessionStatus.label, copyLabel: language === "zh" ? "状态" : "Status" },
    { label: language === "zh" ? "短标识" : "Short hash", value: activeSessionItem?.shortHash || "-", copyLabel: language === "zh" ? "短标识" : "Short hash", mono: true },
    { label: language === "zh" ? "消息数" : "Messages", value: String(timelineMessages.length), copyLabel: language === "zh" ? "消息数" : "Messages" },
    { label: language === "zh" ? "创建时间" : "Created", value: activeSessionItem ? formatDateTime(activeSessionItem.createdAt) : "-", copyLabel: language === "zh" ? "创建时间" : "Created" },
  ] : [];
  useEffect(() => {
    workbench.closeMobileSessionPane();
  }, [runtime.route]);

  const handleCreateSession = useCallback(() => {
    runtime.createSession();
    workbench.closeMobileSessionPane();
  }, [runtime, workbench]);

  const sessionDetailsBody = null;

  const timelineItems = useMemo(
    () => buildRuntimeSessionTimelineItems({
      cacheScope: timelineSessionID,
      messages: visibleMessages,
      language,
      onToggleProcess: toggleProcess,
      expandedProcessEvents,
      onToggleProcessEvent: toggleProcessStep,
      runtimeEventFilter: runtime.runtimeEventFilter,
    }),
    [expandedProcessEvents, language, runtime.runtimeEventFilter, timelineSessionID, toggleProcess, toggleProcessStep, visibleMessages],
  );
  const loadEarlierMessages = useCallback(() => {
    if (!timelineSessionID || (hiddenMessageCount <= 0 && !hasRemoteEarlierMessages)) {
      return;
    }
    if (pendingHistoryScrollRestoreRef.current?.sessionID === timelineSessionID) {
      return;
    }
    historyExpansionRequestedRef.current = true;
    const node = timelineScreenRef.current;
    pendingHistoryScrollRestoreRef.current = {
      sessionID: timelineSessionID,
      scrollHeight: node?.scrollHeight || 0,
      scrollTop: node?.scrollTop || 0,
      anchor: readTimelineViewportAnchor(node),
    };
    if (hiddenMessageCount > 0) {
      setTimelineWindow((current) => {
        const currentVisibleCount = current.sessionID === timelineSessionID
          ? current.visibleCount
          : INITIAL_VISIBLE_CHAT_MESSAGES;
        return {
          sessionID: timelineSessionID,
          visibleCount: Math.min(
            timelineMessages.length,
            currentVisibleCount + CHAT_MESSAGE_LOAD_BATCH_SIZE,
          ),
        };
      });
      return;
    }
    void runtime.loadEarlierHistory().then((loaded) => {
      if (!loaded && pendingHistoryScrollRestoreRef.current?.sessionID === timelineSessionID) {
        pendingHistoryScrollRestoreRef.current = null;
      }
    }).catch(() => {
      if (pendingHistoryScrollRestoreRef.current?.sessionID === timelineSessionID) {
        pendingHistoryScrollRestoreRef.current = null;
      }
    });
  }, [hasRemoteEarlierMessages, hiddenMessageCount, runtime, timelineMessages.length, timelineScreenRef, timelineSessionID]);
  useEffect(() => {
    setTimelineWindow((current) => {
      if (current.sessionID === timelineSessionID) {
        return current;
      }
      return {
        sessionID: timelineSessionID,
        visibleCount: INITIAL_VISIBLE_CHAT_MESSAGES,
      };
    });
    pendingHistoryScrollRestoreRef.current = null;
    historyExpansionRequestedRef.current = false;
  }, [timelineSessionID]);
  useLayoutEffect(() => {
    previousTimelineRenderWindowRef.current = snapshotTimelineRenderWindow(
      timelineSessionID,
      timelineMessages,
      visibleMessages,
    );
    historyExpansionRequestedRef.current = false;
  }, [timelineMessages, timelineSessionID, visibleMessages]);
  useEffect(() => {
    const node = timelineScreenRef.current;
    if (!node) {
      return undefined;
    }
    const handleScroll = () => {
      captureTimelineViewportAnchor();
      if (node.scrollTop <= CHAT_HISTORY_AUTO_LOAD_TOP_OFFSET && (hiddenMessageCount > 0 || hasRemoteEarlierMessages)) {
        loadEarlierMessages();
      }
    };
    node.addEventListener("scroll", handleScroll, { passive: true });
    return () => {
      node.removeEventListener("scroll", handleScroll);
    };
  }, [captureTimelineViewportAnchor, hasRemoteEarlierMessages, hiddenMessageCount, loadEarlierMessages, timelineScreenRef]);
  useLayoutEffect(() => {
    const pending = pendingHistoryScrollRestoreRef.current;
    if (!pending || pending.sessionID !== timelineSessionID) {
      return;
    }
    const node = timelineScreenRef.current;
    if (!node) {
      return;
    }
    const restore = () => {
      if (!restoreTimelineViewportAnchor(node, pending.anchor)) {
        node.scrollTop = Math.max(0, node.scrollHeight - pending.scrollHeight + pending.scrollTop);
      }
    };
    restore();
    const frame = window.requestAnimationFrame(() => {
      restore();
      if (pendingHistoryScrollRestoreRef.current === pending) {
        pendingHistoryScrollRestoreRef.current = null;
      }
    });
    return () => {
      window.cancelAnimationFrame(frame);
      if (pendingHistoryScrollRestoreRef.current === pending) {
        pendingHistoryScrollRestoreRef.current = null;
      }
    };
  }, [timelineItems.length, timelineScreenRef, timelineSessionID]);
  useLayoutEffect(() => {
    const previousMessageCount = previousActiveMessageCountRef.current;
    const previousMessageIDs = previousTimelineMessageIDsRef.current;
    const currentMessageIDs = timelineMessages.map((message) => message.id);
    const messageAppendedToEnd = previousMessageIDs.length > 0
      && currentMessageIDs.length > previousMessageIDs.length
      && previousMessageIDs.every((id, index) => currentMessageIDs[index] === id);
    if (
      !timelineSessionID
      || activeTimelineSessionRef.current !== timelineSessionID
      || previousMessageCount <= 0
      || timelineMessages.length <= previousMessageCount
      || !messageAppendedToEnd
      || visibleMessageCount < previousMessageCount
      || visibleMessageCount >= timelineMessages.length
    ) {
      return;
    }
    setTimelineWindow((current) => {
      if (current.sessionID !== timelineSessionID || current.visibleCount >= timelineMessages.length) {
        return current;
      }
      return {
        sessionID: timelineSessionID,
        visibleCount: timelineMessages.length,
      };
    });
  }, [timelineMessages.length, timelineSessionID, visibleMessageCount]);
  useLayoutEffect(() => {
    const previousSessionID = activeTimelineSessionRef.current;
    const previousMessageIDs = previousTimelineMessageIDsRef.current;
    const currentMessageIDs = timelineMessages.map((message) => message.id);
    const sessionChanged = previousSessionID !== timelineSessionID;
    const messageAppendedToEnd = !sessionChanged
      && previousMessageIDs.length > 0
      && currentMessageIDs.length > previousMessageIDs.length
      && previousMessageIDs.every((id, index) => currentMessageIDs[index] === id);
    const appendedMessages = messageAppendedToEnd
      ? timelineMessages.slice(previousMessageIDs.length)
      : [];
    const userMessageAppended = appendedMessages.some((message) => message.role === "user");
    const messageAppended = Boolean(timelineSessionID && messageAppendedToEnd && userMessageAppended);
    activeTimelineSessionRef.current = timelineSessionID;
    previousActiveMessageCountRef.current = timelineMessages.length;
    previousTimelineMessageIDsRef.current = currentMessageIDs;
    if (!timelineSessionID) {
      return;
    }
    if (timelineItems.length === 0 || (!sessionChanged && !messageAppended)) {
      return;
    }

    const node = timelineScreenRef.current;
    if (!node) {
      return;
    }
    const pinToBottom = () => {
      node.scrollTop = node.scrollHeight;
      captureTimelineViewportAnchor();
    };
    pinToBottom();
    const pinnedTop = node.scrollTop;
    const frame = window.requestAnimationFrame(() => {
      if (node.scrollTop === pinnedTop) {
        pinToBottom();
      }
    });
    return () => {
      window.cancelAnimationFrame(frame);
    };
  }, [timelineItems.length, timelineMessages, timelineMessages.length, timelineScreenRef, timelineSessionID]);
  const timelineEmptyState = useMemo(
    () => (
      <div className="conversation-empty-state">
        <h5>{emptyStateTitle}</h5>
        <p>{emptyStateDescription}</p>
      </div>
    ),
    [emptyStateDescription, emptyStateTitle],
  );
  const timelineOverlay = useMemo(
    () => (workbench.isMobileViewport && inputFocused ? null : (
      <ScrollJumpStrip
        scope="chat"
        language={language}
        containerRef={timelineScreenRef}
        itemSelector=".runtime-message-user[data-message-id]"
        itemAttribute="data-message-id"
        watchKey={`${runtime.route}:${timelineMessages.length}:${isEmptyState ? "empty" : "active"}`}
      />
    )),
    [inputFocused, isEmptyState, language, runtime.route, timelineMessages.length, workbench.isMobileViewport],
  );
  const timelineTopContent = useMemo(() => {
    if (hiddenMessageCount <= 0 && !hasRemoteEarlierMessages) {
      return null;
    }
    const label = language === "zh" ? "加载更早消息" : "Load earlier messages";
    const countLabel = language === "zh"
      ? hiddenMessageCount > 0 ? `还有 ${hiddenMessageCount} 条` : "继续加载"
      : hiddenMessageCount > 0 ? `${hiddenMessageCount} earlier` : "Load more";
    return (
      <div className="conversation-history-loader" data-conversation-history-loader="true">
        <button
          type="button"
          className="conversation-history-loader-button"
          aria-label={label}
          data-conversation-load-earlier="true"
          onClick={loadEarlierMessages}
        >
          <span>{label}</span>
          <small aria-hidden="true">{countLabel}</small>
        </button>
      </div>
    );
  }, [hasRemoteEarlierMessages, hiddenMessageCount, language, loadEarlierMessages]);
  const mobileLayoutState = resolveRuntimeMobileLayoutState({
    isMobileViewport: workbench.isMobileViewport,
    inputFocused,
    primaryNavOpen: workbench.mobileNavOpen,
    sessionPaneOpen: workbench.mobileSessionPaneOpen,
    composerPanelOpen: runtime.inspectorOpen && runtime.inspectorTabOpen,
  });
  const shell = useMemo(() => ({
    shell: {
      rootClassName: "runtime-workspace-view",
      mobileLayoutState,
      rootProps: {
        "data-runtime-view": runtimeViewAlias,
        "data-runtime-route": runtime.route,
      },
      sessionPaneClassName: workbench.isMobileViewport && workbench.mobileSessionPaneOpen
        ? "is-open"
        : undefined,
      sessionPaneProps: {
        "data-runtime-session-pane": runtimeViewAlias,
        "data-mobile-open": workbench.mobileSessionPaneOpen ? "true" : "false",
        "data-testid": "conversation-session-pane",
      },
      sessionPaneBackdrop: {
        ariaLabel: copy.sessionHide,
        onClick: workbench.closeMobileSessionPane,
      },
      sessionPanePrimaryActionClassName: "is-primary",
      sessionPanePrimaryActionProps: { "data-runtime-create-session": runtime.route },
      sessionPaneTitle,
      sessionPaneCountLabel: sessionCountLabel,
      sessionPanePrimaryActionLabel: newSessionLabel,
      onSessionPanePrimaryAction: handleCreateSession,
      sessionPaneSecondaryActionLabel: workbench.isMobileViewport ? copy.sessionHide : undefined,
      onSessionPaneSecondaryAction: workbench.isMobileViewport ? workbench.closeMobileSessionPane : undefined,
      workspaceProps: {
        "data-runtime-workspace": runtimeViewAlias,
        "data-runtime-route": runtime.route,
      },
      workspaceBodyRef,
      mobileHeaderPlacement: workbench.isMobileViewport ? "body" : undefined,
      mobileHeaderProps: { "data-runtime-mobile-variant": runtimeViewAlias },
      mobileNavButtonClassName: "is-quiet conversation-mobile-nav-toggle",
      mobileNavButtonLabel: copy.chatMenu,
      mobileNavButtonProps: { "aria-expanded": workbench.mobileNavOpen },
      onMobileNav: workbench.toggleMobileNav,
      mobileTitleButtonClassName: "conversation-mobile-title-toggle",
      mobileTitleButtonLabel: runtime.activeSession?.title || emptyStateTitle,
      mobileTitleStatusLabel: activeSessionStatus.label,
      mobileTitleTone: activeSessionStatus.tone,
      mobileTitleButtonProps: {
        "aria-haspopup": "dialog",
        "data-runtime-mobile-title": runtimeViewAlias,
        disabled: activeSessionIsDraft,
      },
      onMobileTitle: activeSessionIsDraft ? undefined : () => setSessionDetailsOpen((current) => !current),
      mobilePrimaryButtonClassName: "is-primary conversation-mobile-new-session",
      mobilePrimaryButtonLabel: newSessionLabel,
      mobilePrimaryButtonProps: { "data-runtime-mobile-primary": runtimeViewAlias },
      onMobilePrimary: handleCreateSession,
    },
  }), [
    copy.chatMenu,
    activeSessionStatus.label,
    activeSessionStatus.tone,
    activeSessionIsDraft,
    copy.sessionHide,
    emptyStateTitle,
    handleCreateSession,
    newSessionLabel,
    runtime.route,
    runtimeViewAlias,
    runtime.inspectorOpen,
    runtime.inspectorTabOpen,
    runtime.activeSession?.title,
    sessionCountLabel,
    sessionPaneTitle,
    mobileLayoutState,
    workbench.closeMobileSessionPane,
    workbench.isMobileViewport,
    workbench.mobileNavOpen,
    workbench.mobileSessionPaneOpen,
    workbench.toggleMobileNav,
  ]);
  const sessionList = useMemo(() => ({
    sessionList: {
      groups: sessionListGroups,
      listProps: { "data-runtime-session-list": runtimeViewAlias },
      emptyState: groupedSessionItems.length === 0 ? (
        <p className="route-empty-panel">{sessionEmptyLabel}</p>
      ) : null,
    },
  }), [groupedSessionItems.length, runtimeViewAlias, sessionEmptyLabel, sessionListGroups]);
  const header = useMemo(() => ({
    header: {
      title: runtime.activeSession?.title || emptyStateTitle,
      statusLabel: activeSessionStatus.label,
      statusTone: activeSessionStatus.tone,
      detailsLabel: compactDetailsLabel,
      detailsOpen: activeSessionIsDraft ? false : sessionDetailsOpen,
      onToggleDetails: activeSessionIsDraft ? () => undefined : () => setSessionDetailsOpen((current) => !current),
      detailsDisabled: activeSessionIsDraft,
      mobileEmpty: isMobileEmptyHeader,
      mobileCollapsed: workbench.isMobileViewport,
      detailsClassName: "conversation-inspector conversation-session-details workspace-details-content",
      detailsSummary: conversationDetailsSummary,
      detailsBody: runtime.activeSession ? sessionDetailsBody : null,
      headerProps: { "data-runtime-header-kind": "conversation" },
      detailsPanelProps: {
        "data-runtime-details-panel": runtimeViewAlias,
        "data-conversation-session-details": "",
      },
    },
  }), [
    activeSessionStatus.label,
    activeSessionStatus.tone,
    activeSessionIsDraft,
    compactDetailsLabel,
    emptyStateTitle,
    isMobileEmptyHeader,
    runtime.activeSession,
    runtimeViewAlias,
    sessionDetailsBody,
    sessionDetailsOpen,
    workbench.isMobileViewport,
  ]);
  const screen = useMemo(() => ({
    screen: {
      panelClassName: `conversation-console-panel${isEmptyState ? " is-empty" : ""}`,
      screenClassName: isEmptyState
        ? "is-empty"
        : undefined,
      screenProps: { "data-runtime-screen": runtimeViewAlias },
      screenRef: timelineScreenRef,
    },
  }), [isEmptyState, runtimeViewAlias]);
  const timeline = useMemo(() => ({
    timeline: {
      items: timelineItems,
      topContent: timelineTopContent,
      emptyState: timelineEmptyState,
      overlay: timelineOverlay,
    },
  }), [timelineEmptyState, timelineItems, timelineOverlay, timelineTopContent]);

  return useMemo(() => ({
    ...shell,
    ...sessionList,
    ...header,
    ...screen,
    ...timeline,
    composerNode,
  }), [composerNode, header, screen, sessionList, shell, timeline]);
}

const ConversationComposerSection = memo(function ConversationComposerSection({
  language,
  workspaceBodyRef,
  inputFocused,
  onInputFocusedChange,
}: {
  language: LegacyShellLanguage;
  workspaceBodyRef: { current: HTMLDivElement | null };
  inputFocused: boolean;
  onInputFocusedChange: (focused: boolean) => void;
}) {
  const workbench = useWorkbenchContext();
  const composerRuntime = useConversationRuntimeComposer();
  const copy = getLegacyShellCopy(language);
  const [composerAttachmentError, setComposerAttachmentError] = useState("");
  const [previewAttachment, setPreviewAttachment] = useState<ComposerAttachment | null>(null);
  const composerInputRef = useRef<HTMLTextAreaElement | null>(null);
  const composerFileInputRef = useRef<HTMLInputElement | null>(null);
  const composerShellRef = useRef<HTMLElement | null>(null);
  const mobileSubmitGestureLockRef = useRef(false);
  const composerPlaceholder = language === "zh" ? "输入消息，继续推进当前工作区..." : "Type a message to continue this workspace...";
  const composerSend = language === "zh" ? "发送" : "Send";
  const composerMetaLabel = composerAttachmentError || undefined;
  const composerBusy = composerRuntime.busy;
  const runtimeComposerKind = composerRuntime.route === "terminal" ? "terminal" : "chat";
  const composerAddAttachmentLabel = language === "zh" ? "添加附件" : "Add attachment";
  const composerClosePreviewLabel = language === "zh" ? "关闭预览" : "Close preview";
  const composerPreviewPrefix = language === "zh" ? "预览" : "Preview";
  const composerRemovePrefix = language === "zh" ? "删除" : "Remove";
  const composerImageLimitError = language === "zh"
    ? `最多可暂存 ${MAX_COMPOSER_IMAGE_ATTACHMENTS} 个附件。`
    : `You can attach up to ${MAX_COMPOSER_IMAGE_ATTACHMENTS} attachments.`;
  const composerVisionUnsupported = language === "zh"
    ? "当前模型不支持图片输入，请切换到支持视觉的模型后再发送。"
    : "The selected model does not support image input. Switch to a vision-capable model before sending.";
  const codexSlashCommandsLabel = language === "zh" ? "Codex 斜线命令" : "Codex slash commands";
  const inspectorTabOpen = composerRuntime.inspectorOpen && composerRuntime.inspectorTabOpen;
  const modelInspectorOpen = inspectorTabOpen && composerRuntime.inspectorTab === "model";
  const capabilitiesInspectorOpen = inspectorTabOpen && composerRuntime.inspectorTab === "capabilities";
  const skillsInspectorOpen = inspectorTabOpen && composerRuntime.inspectorTab === "skills";
  const capabilityGroups = useMemo(() => ({
    activeCapabilities: composerRuntime.capabilities.filter((item) => item.active),
    availableCapabilities: composerRuntime.capabilities.filter((item) => !item.active),
    activeSkills: composerRuntime.skills.filter((item) => item.active),
    availableSkills: composerRuntime.skills.filter((item) => !item.active),
  }), [composerRuntime.capabilities, composerRuntime.skills]);
  const directCodexSelected = isDirectCodexModelSelection(
    composerRuntime.selectedProviderId,
    composerRuntime.selectedModelId,
  );
  const codexSlashQuery = directCodexSelected ? codexSlashCommandQuery(composerRuntime.draft) : "";
  const codexSlashCommandCandidates = codexSlashQuery
    ? CODEX_SLASH_COMMANDS.filter((item) => item.command.startsWith(codexSlashQuery))
    : [];

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

  const submitDraft = () => {
    if (composerBusy) {
      return;
    }
    if (composerRuntime.draftAttachments.some(isComposerImageAttachment) && !composerRuntime.selectedModelSupportsVision) {
      setComposerAttachmentError(composerVisionUnsupported);
      return;
    }
    void composerRuntime.sendPrompt(composerRuntime.draft);
  };

  const releaseMobileSubmitGestureLock = () => {
    window.setTimeout(() => {
      mobileSubmitGestureLockRef.current = false;
    }, 0);
  };

  const submitMobileDraftOnPress = () => {
    mobileSubmitGestureLockRef.current = true;
    releaseMobileSubmitGestureLock();
    blurComposerInput();
    submitDraft();
  };

  const handleSubmitPointerDownCapture = (event: PointerEvent<HTMLButtonElement>) => {
    if (!workbench.isMobileViewport || event.pointerType === "mouse" || mobileSubmitGestureLockRef.current) {
      return;
    }
    event.preventDefault();
    submitMobileDraftOnPress();
  };

  const handleSubmitTouchStartCapture = (event: TouchEvent<HTMLButtonElement>) => {
    if (!workbench.isMobileViewport || mobileSubmitGestureLockRef.current) {
      return;
    }
    event.preventDefault();
    submitMobileDraftOnPress();
  };

  const applyCodexSlashCommand = (command: string) => {
    composerRuntime.setDraft(buildDraftWithCodexSlashCommand(composerRuntime.draft, command));
    focusComposerInputWithoutScroll();
  };

  const handleComposerAttachmentPicker = useCallback(() => {
    if (composerBusy) {
      return;
    }
    composerFileInputRef.current?.click();
  }, [composerBusy]);

  const handleComposerAttachmentSelection = useCallback(async (files: FileList | File[] | null) => {
    if (composerBusy) {
      return;
    }
    if (!files || files.length === 0) {
      return;
    }
    if ((composerRuntime.draftAttachments.length + files.length) > MAX_COMPOSER_IMAGE_ATTACHMENTS) {
      setComposerAttachmentError(composerImageLimitError);
      return;
    }
    try {
      const attachments = await readComposerFiles(files);
      await composerRuntime.addDraftAttachments(attachments);
      setComposerAttachmentError("");
    } catch (error) {
      setComposerAttachmentError(error instanceof Error ? error.message : "Failed to add attachment.");
    } finally {
      if (composerFileInputRef.current) {
        composerFileInputRef.current.value = "";
      }
    }
  }, [composerBusy, composerImageLimitError, composerRuntime]);

  const handleComposerInputPaste = useCallback((event: ClipboardEvent<HTMLTextAreaElement>) => {
    const imageFiles = getPastedComposerImageFiles(event.clipboardData);
    if (imageFiles.length === 0) {
      return;
    }
    event.preventDefault();
    void handleComposerAttachmentSelection(imageFiles);
  }, [handleComposerAttachmentSelection]);

  const configPanelHint = modelInspectorOpen
      ? copy.runtimeModelHint
      : capabilitiesInspectorOpen
        ? copy.runtimeToolsHint
        : skillsInspectorOpen
          ? copy.runtimeSkillsHint
          : undefined;
  const configPanelTabs = [
    { key: "model" as const, label: copy.runtimeModel },
    { key: "capabilities" as const, label: copy.runtimeToolsShort },
    { key: "skills" as const, label: copy.runtimeSkillsShort },
  ];
  const runtimeEventDisclosureSection = (
    <section className="conversation-inspector-section">
      <strong>{language === "zh" ? "过程披露" : "Process disclosure"}</strong>
      <div className="conversation-check-list">
        {RUNTIME_EVENT_FILTER_OPTIONS.map((option) => (
          <label key={option.id} className="conversation-check-item">
            <input
              type="checkbox"
              checked={composerRuntime.runtimeEventFilter.includes(option.id)}
              onChange={(event) => composerRuntime.toggleRuntimeEventFilter(option.id, event.target.checked)}
            />
            <span>
              <strong>{option.label[language]}</strong>
              <small>{option.description[language]}</small>
            </span>
          </label>
        ))}
      </div>
    </section>
  );
  const conversationComposerPanel = composerRuntime.inspectorOpen && composerRuntime.inspectorTabOpen ? (
    <div
      className="conversation-inspector runtime-composer-config-panel"
      data-runtime-config-panel="conversation"
      data-runtime-config-tab={composerRuntime.inspectorTab}
    >
      <div className="runtime-composer-panel-head">
        <strong>{copy.runtimeMobile}</strong>
        <button type="button" className="runtime-composer-panel-close" onClick={() => composerRuntime.closeInspector()}>
          {language === "zh" ? "关闭" : "Close"}
        </button>
      </div>
      {configPanelHint ? <p className="runtime-composer-panel-hint">{configPanelHint}</p> : null}
      <div className="conversation-inspector-tabs" role="tablist" aria-label={copy.runtimeMobile}>
        {configPanelTabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={composerRuntime.inspectorTab === tab.key}
            className={composerRuntime.inspectorTab === tab.key ? "is-active" : undefined}
            onClick={() => {
              if (composerRuntime.inspectorTab !== tab.key) {
                composerRuntime.toggleInspector(tab.key);
              }
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {modelInspectorOpen ? (
        <div className="conversation-inspector-sections">
          {composerRuntime.providers.map((provider) => (
            <section key={provider.id} className="conversation-inspector-section">
              <strong>{provider.name}</strong>
              <div className="conversation-chip-list">
                {provider.models.map((model) => (
                  <button
                    key={model.id}
                    className={model.active ? "conversation-chip is-active" : "conversation-chip"}
                    type="button"
                    onClick={() => composerRuntime.selectModel(provider.id, model.id)}
                  >
                    {model.name}
                  </button>
                ))}
              </div>
            </section>
          ))}
          {runtimeEventDisclosureSection}
        </div>
      ) : null}

      {capabilitiesInspectorOpen ? (
        <div className="conversation-inspector-sections">
          <section className="conversation-inspector-section">
            <strong>{language === "zh" ? "已启用" : "Active"}</strong>
            <div className="conversation-check-list">
              {capabilityGroups.activeCapabilities.map((item) => (
                <label key={item.id} className="conversation-check-item">
                  <input
                    type="checkbox"
                    checked={item.active}
                    onChange={(event) => composerRuntime.toggleCapability(item.id, item.kind === "tool" ? "tool" : "mcp", event.target.checked)}
                  />
                  <span><strong>{item.name}</strong><small>{item.description}</small></span>
                </label>
              ))}
            </div>
          </section>
          <section className="conversation-inspector-section">
            <strong>{language === "zh" ? "可选" : "Available"}</strong>
            <div className="conversation-check-list">
              {capabilityGroups.availableCapabilities.map((item) => (
                <label key={item.id} className="conversation-check-item">
                  <input
                    type="checkbox"
                    checked={item.active}
                    onChange={(event) => composerRuntime.toggleCapability(item.id, item.kind === "tool" ? "tool" : "mcp", event.target.checked)}
                  />
                  <span><strong>{item.name}</strong><small>{item.description}</small></span>
                </label>
              ))}
            </div>
          </section>
          {runtimeEventDisclosureSection}
        </div>
      ) : null}

      {skillsInspectorOpen ? (
        <div className="conversation-inspector-sections">
          <section className="conversation-inspector-section">
            <strong>{language === "zh" ? "已启用" : "Active"}</strong>
            <div className="conversation-check-list">
              {capabilityGroups.activeSkills.map((item) => (
                <label key={item.id} className="conversation-check-item">
                  <input
                    type="checkbox"
                    checked={item.active}
                    disabled={item.locked}
                    onChange={(event) => {
                      if (!item.locked) {
                        composerRuntime.toggleSkill(item.id, event.target.checked);
                      }
                    }}
                  />
                  <span><strong>{item.name}</strong><small>{item.description}</small></span>
                </label>
              ))}
            </div>
          </section>
          <section className="conversation-inspector-section">
            <strong>{language === "zh" ? "可选" : "Available"}</strong>
            <div className="conversation-check-list">
              {capabilityGroups.availableSkills.map((item) => (
                <label key={item.id} className="conversation-check-item">
                  <input type="checkbox" checked={item.active} onChange={(event) => composerRuntime.toggleSkill(item.id, event.target.checked)} />
                  <span><strong>{item.name}</strong><small>{item.description}</small></span>
                </label>
              ))}
            </div>
          </section>
        </div>
      ) : null}

    </div>
  ) : null;
  const codexSlashCommandAssist = codexSlashCommandCandidates.length > 0 ? (
    <div
      className="runtime-composer-command-list"
      role="listbox"
      aria-label={codexSlashCommandsLabel}
      data-runtime-composer-command-list="codex"
    >
      {codexSlashCommandCandidates.map((item) => (
        <button
          key={item.command}
          type="button"
          role="option"
          className="runtime-composer-command-option"
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => applyCodexSlashCommand(item.command)}
        >
          <strong>{item.command}</strong>
          <span>{item.label[language]}</span>
        </button>
      ))}
    </div>
  ) : null;

  return (
    <RuntimeComposer
      runtimeKind={runtimeComposerKind}
      shellRef={composerShellRef}
      onSubmit={(event) => {
        event.preventDefault();
        submitDraft();
      }}
      fileInputRef={composerFileInputRef}
      fileInputAccept="image/*,.txt,.md,.json,.yaml,.yml,.csv,.log,.pdf"
      onFileChange={(event) => {
        void handleComposerAttachmentSelection(event.target.files);
      }}
      attachments={composerRuntime.draftAttachments}
      attachmentStripProps={{ "data-runtime-attachments": runtimeComposerKind === "terminal" ? "terminal" : "conversation" }}
      attachmentPreviewLabel={(attachment) => `${composerPreviewPrefix} ${attachment.name}`}
      attachmentRemoveLabel={(attachment) => `${composerRemovePrefix} ${attachment.name}`}
      previewAttachment={previewAttachment}
      onPreviewAttachmentChange={setPreviewAttachment}
      onRemoveAttachment={(attachment) => composerRuntime.removeDraftAttachment(attachment.id)}
      inputLabel={composerPlaceholder}
      inputId="conversationRuntimeInput"
      inputRef={composerInputRef}
      inputValue={composerRuntime.draft}
      inputProps={{
        maxLength: 10000,
        onPaste: handleComposerInputPaste,
        placeholder: composerPlaceholder,
        disabled: composerBusy,
      }}
      inputAssistContent={codexSlashCommandAssist}
      onInputChange={composerRuntime.setDraft}
      onInputFocus={() => onInputFocusedChange(true)}
      onInputBlur={() => onInputFocusedChange(false)}
      utilityButtons={[
        {
          key: "session",
          label: copy.runtimeMobile,
          icon: <RuntimeSessionControlIcon />,
          className: composerRuntime.inspectorOpen ? "is-active" : undefined,
          onClick: () => composerRuntime.toggleInspector(),
        },
      ]}
      panelContent={conversationComposerPanel}
      onPanelDismiss={() => composerRuntime.closeInspector()}
      panelProps={{
        "data-runtime-config-surface": runtimeComposerKind === "terminal" ? "terminal" : "conversation",
      }}
      metaContent={composerMetaLabel}
      addAttachmentLabel={composerAddAttachmentLabel}
      addAttachmentButtonProps={{ disabled: composerBusy }}
      onAddAttachment={handleComposerAttachmentPicker}
      submitButtonProps={{
        disabled: composerBusy,
        onPointerDownCapture: handleSubmitPointerDownCapture,
        onTouchStartCapture: handleSubmitTouchStartCapture,
      }}
      submitLabel={composerSend}
      previewCloseLabel={composerClosePreviewLabel}
    />
  );
});

export function ConversationWorkspace({ language }: ConversationWorkspaceProps) {
  const timelineScreenRef = useRef<HTMLDivElement | null>(null);
  const workspaceBodyRef = useRef<HTMLDivElement | null>(null);
  const [inputFocused, setInputFocused] = useState(false);
  const composerNode = (
    <ConversationComposerSection
      language={language}
      workspaceBodyRef={workspaceBodyRef}
      inputFocused={inputFocused}
      onInputFocusedChange={setInputFocused}
    />
  );
  const controller = useConversationWorkspaceController(
    language,
    { timelineScreenRef, workspaceBodyRef },
    composerNode,
    inputFocused,
  );
  return <RuntimeWorkspacePage controller={controller} />;
}
