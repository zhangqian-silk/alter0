import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type PointerEvent, type ReactNode, type TouchEvent } from "react";
import { useWorkbenchContext } from "../../app/WorkbenchContext";
import { formatDateTime } from "../../shared/time/format";
import { groupSessionListItems } from "../../shared/time/sessionListGroups";
import { buildChatTimelineItems } from "../shell/components/ChatMessageRegion";
import { normalizeText, RouteFieldRow } from "../shell/components/RouteBodyPrimitives";
import { RuntimeComposer } from "../shell/components/RuntimeComposer";
import { RuntimeWorkspacePage, type RuntimeWorkspacePageController } from "../shell/components/RuntimeWorkspacePage";
import { ScrollJumpStrip } from "../shell/components/ScrollJumpStrip";
import { useRuntimeComposerViewportSync } from "../shell/components/useRuntimeComposerViewportSync";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../shell/legacyShellCopy";
import {
  isComposerImageAttachment,
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
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

function renderAgentDeliverablesSection(
  language: LegacyShellLanguage,
  deliverables: Array<{
    id?: string;
    label?: string;
    description?: string;
    format?: string;
    required?: boolean;
    session_attribute_key?: string;
  }>,
  attributes: Record<string, string>,
) {
  if (!deliverables.length) {
    return null;
  }
  return (
    <section className="conversation-inspector-section">
      <strong>{language === "zh" ? "交付契约" : "Delivery Contract"}</strong>
      <div className="conversation-check-list">
        {deliverables.map((deliverable) => {
          const attributeKey = String(deliverable.session_attribute_key || "").trim();
          const resolvedValue = attributeKey ? String(attributes[attributeKey] || "").trim() : "";
          const stateLabel = resolvedValue
            ? (language === "zh" ? "已关联" : "Linked")
            : deliverable.required
              ? (language === "zh" ? "必交付" : "Required")
              : (language === "zh" ? "可选" : "Optional");
          const formatLabel = String(deliverable.format || "").trim();
          const meta = [stateLabel, formatLabel, resolvedValue].filter(Boolean).join(" · ");
          const detail = [meta, String(deliverable.description || "").trim()].filter(Boolean).join(" · ");
          return (
            <div key={String(deliverable.id || deliverable.label)} className="conversation-check-item">
              <span>
                <strong>{String(deliverable.label || deliverable.id || "").trim()}</strong>
                <small>{detail || (language === "zh" ? "无额外说明" : "No extra guidance")}</small>
              </span>
            </div>
          );
        })}
      </div>
    </section>
  );
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

type ConversationSessionSignalTone = "ready" | "busy" | "failed";

function normalizeConversationSessionMessageStatus(value: string) {
  return normalizeText(value).toLowerCase();
}

function resolveConversationSessionSignalTone(session: {
  status?: string;
  messages?: Array<{
    role?: string;
    status?: string;
    error?: boolean;
    taskPending?: boolean;
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
    if (message.taskPending || ["streaming", "queued", "running", "in_progress", "inprogress"].includes(status)) {
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

function buildAgentPickerMonogram(name: string) {
  const parts = normalizeText(name).split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "AG";
  }
  return parts.slice(0, 2).map((part) => part.slice(0, 1).toUpperCase()).join("");
}

function AgentPickerGlyph({
  agentID,
  fallbackName,
}: {
  agentID: string;
  fallbackName: string;
}) {
  switch (normalizeText(agentID).toLowerCase()) {
    case "main":
      return (
        <svg viewBox="0 0 24 24" fill="none" focusable="false" aria-hidden="true">
          <circle cx="12" cy="12" r="3.25" stroke="currentColor" strokeWidth="1.8" />
          <path d="M12 3.75v2.4M12 17.85v2.4M20.25 12h-2.4M6.15 12h-2.4M17.84 6.16l-1.7 1.69M7.86 16.14l-1.7 1.7M17.84 17.84l-1.7-1.7M7.86 7.86l-1.7-1.7" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
        </svg>
      );
    case "coding":
      return (
        <svg viewBox="0 0 24 24" fill="none" focusable="false" aria-hidden="true">
          <path d="m9.1 7.25-4.2 4.75 4.2 4.75M14.9 7.25l4.2 4.75-4.2 4.75M13.2 5.75l-2.4 12.5" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case "writing":
      return (
        <svg viewBox="0 0 24 24" fill="none" focusable="false" aria-hidden="true">
          <path d="m6.2 17.8 1.25-4.15L15.9 5.2a1.7 1.7 0 0 1 2.4 0l.5.5a1.7 1.7 0 0 1 0 2.4l-8.45 8.45Z" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" />
          <path d="M13.9 7.2 17 10.3M6.05 18h11.9" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      );
    case "travel":
      return (
        <svg viewBox="0 0 24 24" fill="none" focusable="false" aria-hidden="true">
          <path d="M20.2 8.7 13.95 11 11.7 4.8a1.05 1.05 0 0 0-1.98.08v7.1L4.25 14a1 1 0 0 0 .18 1.93l5.3.7v2.62l-1.8 1.05a.85.85 0 0 0 .43 1.58h2.9c.78 0 1.42-.63 1.42-1.42v-3.83l6.84-5.88a1.1 1.1 0 0 0-.32-1.85Z" stroke="currentColor" strokeWidth="1.65" strokeLinejoin="round" />
        </svg>
      );
    default:
      return <span className="conversation-target-card-monogram">{buildAgentPickerMonogram(fallbackName)}</span>;
  }
}

function summarizeAgentPickerSubtitle(text: string, language: LegacyShellLanguage) {
  const normalized = normalizeText(text);
  if (!normalized) {
    return language === "zh" ? "可直接开始" : "Ready to start";
  }
  const firstSentence = normalized
    .split(/[。.!?]/)
    .map((part) => normalizeText(part))
    .find(Boolean);
  const concise = firstSentence || normalized;
  if (concise.length <= 44) {
    return concise;
  }
  return `${concise.slice(0, 41).trimEnd()}…`;
}

function resolveAgentPickerSubtitle(
  agent: { id: string; subtitle: string },
  language: LegacyShellLanguage,
) {
  const agentID = normalizeText(agent.id).toLowerCase();
  if (language === "zh") {
    switch (agentID) {
      case "main":
        return "统筹当前任务并分派合适的专家 Agent";
      case "coding":
        return "编写代码、排查问题并交付改动";
      case "writing":
        return "起草文档、文案与结构化内容";
      case "travel":
        return "规划行程、路线与城市指南";
      default:
        return summarizeAgentPickerSubtitle(agent.subtitle, language);
    }
  }
  switch (agentID) {
    case "main":
      return "Coordinate the right specialist for the task";
    case "coding":
      return "Code, debug, and ship with confidence";
    case "writing":
      return "Draft docs, blogs, and product copy";
    case "travel":
      return "Plan trips, routes, and guides";
    default:
      return summarizeAgentPickerSubtitle(agent.subtitle, language);
  }
}

function AgentPickerChevronIcon() {
  return (
    <svg viewBox="0 0 16 16" fill="none" focusable="false" aria-hidden="true">
      <path d="M6 3.5 10.5 8 6 12.5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function AgentPickerCheckIcon() {
  return (
    <svg viewBox="0 0 16 16" fill="none" focusable="false" aria-hidden="true">
      <path d="m4.25 8.1 2.3 2.35 5.2-5.1" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
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
  const pendingHistoryScrollRestoreRef = useRef<{
    sessionID: string;
    scrollHeight: number;
    scrollTop: number;
  } | null>(null);
  const [timelineWindow, setTimelineWindow] = useState({
    sessionID: "",
    visibleCount: INITIAL_VISIBLE_CHAT_MESSAGES,
  });
  const activeMessages = runtime.activeSession?.messages || [];
  const activeSessionID = runtime.activeSession?.id || "";
  const visibleMessageCount = timelineWindow.sessionID === activeSessionID
    ? timelineWindow.visibleCount
    : INITIAL_VISIBLE_CHAT_MESSAGES;
  const hiddenMessageCount = Math.max(0, activeMessages.length - visibleMessageCount);
  const visibleMessages = useMemo(
    () => hiddenMessageCount > 0 ? activeMessages.slice(-visibleMessageCount) : activeMessages,
    [activeMessages, hiddenMessageCount, visibleMessageCount],
  );
  const isEmptyState = activeMessages.length === 0;
  const isMobileEmptyHeader = workbench.isMobileViewport && isEmptyState;
  const emptyStateTitle = runtime.route === "agent-runtime"
    ? (language === "zh" ? "选择 Agent 并开始执行" : "Pick an agent and start a run")
    : (language === "zh" ? "开始新的工作流" : "Start a new workspace flow");
  const emptyStateDescription = runtime.route === "agent-runtime"
    ? (language === "zh"
      ? "会话、过程步骤和最终输出会按 Terminal 工作区方式持续沉淀。"
      : "Sessions, process steps, and final output stay in one terminal-style workspace.")
    : (language === "zh"
      ? "对话、过程和交付结果都在同一条时间线里推进。"
      : "Conversation, process, and delivery stay in a single timeline.");
  const sessionPaneTitle = copy.terminalSessions;
  const newSessionLabel = copy.terminalNewShort;
  const sessionCountLabel = language === "zh"
    ? `${runtime.sessionItems.length} 个会话`
    : `${runtime.sessionItems.length} sessions`;
  const activeSessionBadgeLabel = language === "zh" ? "当前" : "Current";
  const idleSessionBadgeLabel = language === "zh" ? "会话" : "Session";
  const deleteSessionLabel = language === "zh" ? "删除" : "Delete";
  const deleteSessionAriaLabel = language === "zh" ? "删除会话" : "Delete session";
  const groupedSessionItems = useMemo(
    () => groupSessionListItems(runtime.sessionItems, {
      language,
      getTimestamp: (item) => item.createdAt,
    }),
    [language, runtime.sessionItems],
  );
  const sessionEmptyLabel = runtime.route === "agent-runtime" ? copy.sessionEmptyAgent : copy.sessionEmpty;
  const compactDetailsLabel = language === "zh" ? "详情" : "Details";
  const sessionProfileFields = runtime.activeSessionProfile?.fields || runtime.activeAgent?.session_profile_fields || [];
  const activeAgentDeliverables = runtime.activeAgent?.deliverables || [];
  const sessionProfileAttributes = runtime.activeSessionProfile?.attributes || {};
  const activeSessionItem = runtime.sessionItems.find((item) => item.active) || null;
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
  const sessionListGroups = useMemo(
    () => groupedSessionItems.map((group) => ({
      ...group,
      items: group.items.map((item) => ({
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
        onDelete: () => void handleRemoveSession(item.id),
        deleteLabel: deleteSessionLabel,
        deleteAriaLabel: deleteSessionAriaLabel,
        shellClassName: item.active ? "runtime-session-card is-active" : "runtime-session-card",
        shellProps: {
          "data-runtime-session-state": item.active ? "active" : "idle",
          "data-runtime-session-card": item.id,
          "data-runtime-session-tone": sessionStatusByID[item.id]?.tone || "ready",
        },
        buttonClassName: item.active ? "runtime-session-select active" : "runtime-session-select",
      })),
    })),
    [
      activeSessionBadgeLabel,
      deleteSessionAriaLabel,
      deleteSessionLabel,
      groupedSessionItems,
      idleSessionBadgeLabel,
      language,
      sessionStatusByID,
    ],
  );
  const routeLabel = runtime.route === "agent-runtime"
    ? (language === "zh" ? "Agent" : "Agent")
    : (language === "zh" ? "对话" : "Chat");
  const conversationDetailsSummary = runtime.activeSession ? [
    { label: language === "zh" ? "会话" : "Session", value: runtime.activeSession.id, copyLabel: language === "zh" ? "会话" : "Session", mono: true },
    { label: language === "zh" ? "路由" : "Route", value: routeLabel, copyLabel: language === "zh" ? "路由" : "Route" },
    { label: language === "zh" ? "状态" : "Status", value: activeSessionStatus.label, copyLabel: language === "zh" ? "状态" : "Status" },
    { label: language === "zh" ? "短标识" : "Short hash", value: activeSessionItem?.shortHash || "-", copyLabel: language === "zh" ? "短标识" : "Short hash", mono: true },
    ...(runtime.route === "agent-runtime" ? [{ label: copy.runtimeAgent, value: runtime.target.name || "-", copyLabel: copy.runtimeAgent }] : []),
    { label: language === "zh" ? "消息数" : "Messages", value: String(activeMessages.length), copyLabel: language === "zh" ? "消息数" : "Messages" },
    { label: language === "zh" ? "创建时间" : "Created", value: activeSessionItem ? formatDateTime(activeSessionItem.createdAt) : "-", copyLabel: language === "zh" ? "创建时间" : "Created" },
  ] : [];

  useEffect(() => {
    workbench.closeMobileSessionPane();
  }, [runtime.route]);

  const handleCreateSession = useCallback(() => {
    runtime.createSession();
    workbench.closeMobileSessionPane();
  }, [runtime, workbench]);

  const handleFocusSession = useCallback((sessionID: string) => {
    runtime.focusSession(sessionID);
    workbench.closeMobileSessionPane();
  }, [runtime, workbench]);

  const handleRemoveSession = useCallback((sessionID: string) => {
    workbench.closeMobileSessionPane();
    return runtime.removeSession(sessionID);
  }, [runtime, workbench]);

  const sessionDetailsBody = runtime.route === "agent-runtime" && (sessionProfileFields.length > 0 || activeAgentDeliverables.length > 0) ? (
    <div className="conversation-inspector-sections">
      {renderAgentDeliverablesSection(language, activeAgentDeliverables, sessionProfileAttributes)}
      {sessionProfileFields.length > 0 ? (
        <section className="conversation-inspector-section">
          <strong>{language === "zh" ? "实例属性" : "Instance Attributes"}</strong>
          <div className="workspace-details-summary">
            {sessionProfileFields.map((field) => {
              const value = sessionProfileAttributes[field.key] || "-";
              const mono = field.readonly === true || field.key.includes("path") || field.key.includes("branch");
              return (
                <RouteFieldRow
                  key={field.key}
                  label={field.label}
                  value={value}
                  copyLabel={language === "zh" ? "复制值" : "Copy value"}
                  copyable={field.readonly !== false}
                  mono={mono}
                  multiline={value.length > 48}
                  markdown={!mono}
                />
              );
            })}
          </div>
        </section>
      ) : null}
    </div>
  ) : null;

  const timelineItems = useMemo(
    () => buildChatTimelineItems({
      messages: visibleMessages,
      language,
      onToggleProcess: runtime.toggleAgentProcess,
    }),
    [language, runtime.toggleAgentProcess, visibleMessages],
  );
  const loadEarlierMessages = useCallback(() => {
    if (!activeSessionID || hiddenMessageCount <= 0) {
      return;
    }
    const node = timelineScreenRef.current;
    pendingHistoryScrollRestoreRef.current = {
      sessionID: activeSessionID,
      scrollHeight: node?.scrollHeight || 0,
      scrollTop: node?.scrollTop || 0,
    };
    setTimelineWindow((current) => {
      const currentVisibleCount = current.sessionID === activeSessionID
        ? current.visibleCount
        : INITIAL_VISIBLE_CHAT_MESSAGES;
      return {
        sessionID: activeSessionID,
        visibleCount: Math.min(
          activeMessages.length,
          currentVisibleCount + CHAT_MESSAGE_LOAD_BATCH_SIZE,
        ),
      };
    });
  }, [activeMessages.length, activeSessionID, hiddenMessageCount, timelineScreenRef]);
  useEffect(() => {
    setTimelineWindow((current) => {
      if (current.sessionID === activeSessionID) {
        return current;
      }
      return {
        sessionID: activeSessionID,
        visibleCount: INITIAL_VISIBLE_CHAT_MESSAGES,
      };
    });
    pendingHistoryScrollRestoreRef.current = null;
  }, [activeSessionID]);
  useEffect(() => {
    if (hiddenMessageCount <= 0) {
      return undefined;
    }
    const node = timelineScreenRef.current;
    if (!node) {
      return undefined;
    }
    const handleScroll = () => {
      if (node.scrollTop <= CHAT_HISTORY_AUTO_LOAD_TOP_OFFSET) {
        loadEarlierMessages();
      }
    };
    node.addEventListener("scroll", handleScroll, { passive: true });
    return () => {
      node.removeEventListener("scroll", handleScroll);
    };
  }, [hiddenMessageCount, loadEarlierMessages, timelineScreenRef]);
  useLayoutEffect(() => {
    const pending = pendingHistoryScrollRestoreRef.current;
    if (!pending || pending.sessionID !== activeSessionID) {
      return;
    }
    const node = timelineScreenRef.current;
    if (!node) {
      return;
    }
    pendingHistoryScrollRestoreRef.current = null;
    node.scrollTop = Math.max(0, node.scrollHeight - pending.scrollHeight + pending.scrollTop);
  }, [activeSessionID, timelineItems.length, timelineScreenRef]);
  useLayoutEffect(() => {
    const previousSessionID = activeTimelineSessionRef.current;
    const previousMessageCount = previousActiveMessageCountRef.current;
    const sessionChanged = previousSessionID !== activeSessionID;
    const appendedMessages = activeMessages.slice(previousMessageCount);
    const userMessageAppended = appendedMessages.some((message) => message.role === "user");
    const messageAppended = !sessionChanged && activeSessionID && activeMessages.length > previousMessageCount && userMessageAppended;
    activeTimelineSessionRef.current = activeSessionID;
    previousActiveMessageCountRef.current = activeMessages.length;
    if (!activeSessionID) {
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
  }, [activeMessages, activeMessages.length, activeSessionID, timelineItems.length, timelineScreenRef]);
  const timelineEmptyState = useMemo(
    () => (
      <div className="conversation-empty-state">
        <h5>{emptyStateTitle}</h5>
        <p>{emptyStateDescription}</p>
        {runtime.route === "agent-runtime" && runtime.targetOptions.length > 0 ? (
          <div className="conversation-empty-targets">
            <div
              className={workbench.isMobileViewport
                ? "conversation-empty-target-list"
                : "conversation-inspector-grid conversation-empty-target-grid"}
              role="radiogroup"
              aria-label={language === "zh" ? "选择 Agent" : "Choose agent"}
              data-agent-picker-layout={workbench.isMobileViewport ? "list" : "grid"}
            >
              {runtime.targetOptions.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  role="radio"
                  aria-checked={item.active}
                  className={[
                    "conversation-target-card",
                    workbench.isMobileViewport ? "is-list-row" : "",
                    item.active ? "is-active" : "",
                  ].filter(Boolean).join(" ")}
                  disabled={runtime.lockedTarget}
                  onClick={() => runtime.selectTarget(item.id)}
                >
                  <span className="conversation-target-card-leading" aria-hidden="true">
                    <span className="conversation-target-card-icon" data-agent-picker-icon={item.id}>
                      <AgentPickerGlyph agentID={item.id} fallbackName={item.name} />
                    </span>
                  </span>
                  <span className="conversation-target-card-copy">
                    <strong>{item.name}</strong>
                    <span>{resolveAgentPickerSubtitle(item, language)}</span>
                  </span>
                  <span className="conversation-target-card-trailing" aria-hidden="true">
                    <span className={item.active ? "conversation-target-card-indicator is-active" : "conversation-target-card-indicator"}>
                      {item.active ? <AgentPickerCheckIcon /> : <AgentPickerChevronIcon />}
                    </span>
                  </span>
                </button>
              ))}
            </div>
          </div>
        ) : null}
      </div>
    ),
    [emptyStateDescription, emptyStateTitle, language, runtime.lockedTarget, runtime.route, runtime.selectTarget, runtime.targetOptions, workbench.isMobileViewport],
  );
  const timelineOverlay = useMemo(
    () => (workbench.isMobileViewport && inputFocused ? null : (
      <ScrollJumpStrip
        scope={runtime.route === "agent-runtime" ? "agent" : "chat"}
        language={language}
        containerRef={timelineScreenRef}
        itemSelector="[data-message-id]"
        itemAttribute="data-message-id"
        watchKey={`${runtime.route}:${activeMessages.length}:${isEmptyState ? "empty" : "active"}`}
      />
    )),
    [activeMessages.length, inputFocused, isEmptyState, language, runtime.route, workbench.isMobileViewport],
  );
  const timelineTopContent = useMemo(() => {
    if (hiddenMessageCount <= 0) {
      return null;
    }
    const label = language === "zh" ? "加载更早消息" : "Load earlier messages";
    const countLabel = language === "zh"
      ? `还有 ${hiddenMessageCount} 条`
      : `${hiddenMessageCount} earlier`;
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
  }, [hiddenMessageCount, language, loadEarlierMessages]);
  const shell = useMemo(() => ({
    shell: {
      rootClassName: "runtime-workspace-view",
      rootProps: {
        "data-runtime-view": "conversation",
        "data-runtime-route": runtime.route,
      },
      sessionPaneClassName: workbench.isMobileViewport && workbench.mobileSessionPaneOpen
        ? "is-open"
        : undefined,
      sessionPaneProps: {
        "data-runtime-session-pane": "conversation",
        "data-mobile-open": workbench.mobileSessionPaneOpen ? "true" : "false",
        "data-testid": "conversation-session-pane",
      },
      sessionPaneBackdrop: {
        ariaLabel: copy.sessionHide,
        onClick: workbench.closeMobileSessionPane,
      },
      sessionPanePrimaryActionClassName: "is-primary",
      sessionPaneTitle,
      sessionPaneCountLabel: sessionCountLabel,
      sessionPanePrimaryActionLabel: newSessionLabel,
      onSessionPanePrimaryAction: handleCreateSession,
      sessionPaneSecondaryActionLabel: workbench.isMobileViewport ? copy.sessionHide : undefined,
      onSessionPaneSecondaryAction: workbench.isMobileViewport ? workbench.closeMobileSessionPane : undefined,
      workspaceProps: {
        "data-runtime-workspace": "conversation",
        "data-runtime-route": runtime.route,
      },
      workspaceBodyRef,
      mobileHeaderPlacement: workbench.isMobileViewport ? "body" : undefined,
      mobileHeaderProps: { "data-runtime-mobile-variant": "conversation" },
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
        "data-runtime-mobile-title": "conversation",
      },
      onMobileTitle: () => setSessionDetailsOpen((current) => !current),
      mobileSessionButtonClassName: "is-quiet conversation-mobile-session-toggle",
      mobileSessionButtonLabel: copy.terminalSessions,
      mobileSessionButtonProps: { "aria-expanded": workbench.mobileSessionPaneOpen },
      onMobileSession: workbench.toggleMobileSessionPane,
      mobilePrimaryButtonClassName: "is-primary conversation-mobile-new-session",
      mobilePrimaryButtonLabel: newSessionLabel,
      mobilePrimaryButtonProps: { "data-runtime-mobile-primary": "conversation" },
      onMobilePrimary: handleCreateSession,
    },
  }), [
    copy.chatMenu,
    activeSessionStatus.label,
    activeSessionStatus.tone,
    copy.sessionHide,
    copy.terminalSessions,
    emptyStateTitle,
    handleCreateSession,
    newSessionLabel,
    runtime.route,
    runtime.activeSession?.title,
    sessionCountLabel,
    sessionPaneTitle,
    workbench.closeMobileSessionPane,
    workbench.isMobileViewport,
    workbench.mobileNavOpen,
    workbench.mobileSessionPaneOpen,
    workbench.toggleMobileNav,
    workbench.toggleMobileSessionPane,
  ]);
  const sessionList = useMemo(() => ({
    sessionList: {
      groups: sessionListGroups,
      listProps: { "data-runtime-session-list": "conversation" },
      emptyState: groupedSessionItems.length === 0 ? (
        <p className="route-empty-panel">{sessionEmptyLabel}</p>
      ) : null,
    },
  }), [groupedSessionItems.length, sessionEmptyLabel, sessionListGroups]);
  const header = useMemo(() => ({
    header: {
      title: runtime.activeSession?.title || emptyStateTitle,
      statusLabel: activeSessionStatus.label,
      statusTone: activeSessionStatus.tone,
      detailsLabel: compactDetailsLabel,
      detailsOpen: sessionDetailsOpen,
      onToggleDetails: () => setSessionDetailsOpen((current) => !current),
      detailsDisabled: false,
      mobileEmpty: isMobileEmptyHeader,
      mobileCollapsed: workbench.isMobileViewport,
      detailsClassName: "conversation-inspector conversation-session-details workspace-details-content",
      detailsSummary: conversationDetailsSummary,
      detailsBody: runtime.activeSession ? sessionDetailsBody : null,
      headerProps: { "data-runtime-header-kind": "conversation" },
      detailsPanelProps: {
        "data-runtime-details-panel": "conversation",
        "data-conversation-session-details": "",
      },
    },
  }), [
    activeSessionStatus.label,
    activeSessionStatus.tone,
    compactDetailsLabel,
    emptyStateTitle,
    isMobileEmptyHeader,
    runtime.activeSession,
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
      screenProps: { "data-runtime-screen": "conversation" },
      screenRef: timelineScreenRef,
    },
  }), [isEmptyState]);
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

function ConversationComposerSection({
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
  const runtime = useConversationRuntimeWorkspace();
  const composerRuntime = useConversationRuntimeComposer();
  const copy = getLegacyShellCopy(language);
  const [composerAttachmentError, setComposerAttachmentError] = useState("");
  const [previewAttachment, setPreviewAttachment] = useState<ComposerAttachment | null>(null);
  const composerInputRef = useRef<HTMLTextAreaElement | null>(null);
  const composerFileInputRef = useRef<HTMLInputElement | null>(null);
  const composerShellRef = useRef<HTMLElement | null>(null);
  const mobileSubmitGestureLockRef = useRef(false);
  const mobileSessionGestureLockRef = useRef(false);
  const composerPlaceholder = language === "zh" ? "输入消息，继续推进当前工作区..." : "Type a message to continue this workspace...";
  const composerSend = language === "zh" ? "发送" : "Send";
  const composerMetaLabel = composerAttachmentError || undefined;
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
  const inspectorTabOpen = runtime.inspectorOpen && runtime.inspectorTabOpen;
  const targetInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "target";
  const modelInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "model";
  const capabilitiesInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "capabilities";
  const skillsInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "skills";
  const sessionProfileInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "session-profile";
  const deliverablesInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "deliverables";
  const sessionProfileFields = runtime.activeSessionProfile?.fields || runtime.activeAgent?.session_profile_fields || [];
  const activeAgentDeliverables = runtime.activeAgent?.deliverables || [];
  const sessionProfileAttributes = runtime.activeSessionProfile?.attributes || {};
  const capabilityGroups = useMemo(() => ({
    activeCapabilities: runtime.capabilities.filter((item) => item.active),
    availableCapabilities: runtime.capabilities.filter((item) => !item.active),
    activeSkills: runtime.skills.filter((item) => item.active),
    availableSkills: runtime.skills.filter((item) => !item.active && item.visibility !== "agent-private"),
  }), [runtime.capabilities, runtime.skills]);

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

  const submitDraft = () => {
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

  const releaseMobileSessionGestureLock = () => {
    window.setTimeout(() => {
      mobileSessionGestureLockRef.current = false;
    }, 0);
  };

  const toggleSessionInspector = () => {
    runtime.toggleInspector(runtime.inspectorTab);
  };

  const openMobileSessionInspectorOnPress = () => {
    mobileSessionGestureLockRef.current = true;
    releaseMobileSessionGestureLock();
    blurComposerInput();
    toggleSessionInspector();
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

  const handleSessionUtilityPointerDownCapture = (event: PointerEvent<HTMLButtonElement>) => {
    if (!workbench.isMobileViewport || event.pointerType === "mouse" || mobileSessionGestureLockRef.current) {
      return;
    }
    event.preventDefault();
    openMobileSessionInspectorOnPress();
  };

  const handleSessionUtilityTouchStartCapture = (event: TouchEvent<HTMLButtonElement>) => {
    if (!workbench.isMobileViewport || mobileSessionGestureLockRef.current) {
      return;
    }
    event.preventDefault();
    openMobileSessionInspectorOnPress();
  };

  const handleComposerAttachmentPicker = useCallback(() => {
    composerFileInputRef.current?.click();
  }, []);

  const handleComposerAttachmentSelection = useCallback(async (files: FileList | null) => {
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
  }, [composerImageLimitError, composerRuntime]);

  useRuntimeComposerViewportSync({
    isMobileViewport: workbench.isMobileViewport,
    inputFocused,
    workspaceBodyRef,
    composerShellRef,
  });

  const configPanelHint = targetInspectorOpen
    ? copy.runtimeAgentHint
    : modelInspectorOpen
      ? copy.runtimeModelHint
      : capabilitiesInspectorOpen
        ? copy.runtimeToolsHint
        : deliverablesInspectorOpen
          ? (language === "zh" ? "专项 Agent 会在这里声明本轮必须交付的最终产物。"
            : "Specialist agents declare the required final deliverables for this run here.")
        : skillsInspectorOpen
          ? copy.runtimeSkillsHint
          : undefined;
  const configPanelTabs = [
    ...(runtime.route === "agent-runtime" ? [{
      key: "target" as const,
      label: copy.runtimeAgent,
    }, {
      key: "deliverables" as const,
      label: language === "zh" ? "交付物" : "Deliverables",
    }] : []),
    { key: "model" as const, label: copy.runtimeModel },
    { key: "capabilities" as const, label: copy.runtimeToolsShort },
    { key: "skills" as const, label: copy.runtimeSkillsShort },
  ];
  const conversationComposerPanel = runtime.inspectorOpen && runtime.inspectorTabOpen ? (
    <div
      className="conversation-inspector runtime-composer-config-panel"
      data-runtime-config-panel="conversation"
      data-runtime-config-tab={runtime.inspectorTab}
    >
      <div className="runtime-composer-panel-head">
        <strong>{copy.runtimeMobile}</strong>
        <button type="button" className="runtime-composer-panel-close" onClick={() => runtime.closeInspector()}>
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
            aria-selected={runtime.inspectorTab === tab.key}
            className={runtime.inspectorTab === tab.key ? "is-active" : undefined}
            onClick={() => {
              if (runtime.inspectorTab !== tab.key) {
                runtime.toggleInspector(tab.key);
              }
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {targetInspectorOpen && runtime.route === "agent-runtime" ? (
        <div className="conversation-inspector-sections">
          <div className="conversation-inspector-grid">
            {runtime.targetOptions.map((item) => (
              <button
                key={item.id}
                className={item.active ? "conversation-target-card is-active" : "conversation-target-card"}
                type="button"
                disabled={runtime.lockedTarget}
                onClick={() => runtime.selectTarget(item.id)}
              >
                <strong>{item.name}</strong>
                <span>{item.subtitle}</span>
              </button>
            ))}
          </div>
          {renderAgentDeliverablesSection(language, activeAgentDeliverables, sessionProfileAttributes)}
        </div>
      ) : null}

      {deliverablesInspectorOpen && runtime.route === "agent-runtime" ? (
        <div className="conversation-inspector-sections">
          {renderAgentDeliverablesSection(language, activeAgentDeliverables, sessionProfileAttributes)}
        </div>
      ) : null}

      {modelInspectorOpen ? (
        <div className="conversation-inspector-sections">
          {runtime.providers.map((provider) => (
            <section key={provider.id} className="conversation-inspector-section">
              <strong>{provider.name}</strong>
              <div className="conversation-chip-list">
                {provider.models.map((model) => (
                  <button
                    key={model.id}
                    className={model.active ? "conversation-chip is-active" : "conversation-chip"}
                    type="button"
                    onClick={() => runtime.selectModel(provider.id, model.id)}
                  >
                    {model.name}
                  </button>
                ))}
              </div>
            </section>
          ))}
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
                    onChange={(event) => runtime.toggleCapability(item.id, item.kind === "tool" ? "tool" : "mcp", event.target.checked)}
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
                    onChange={(event) => runtime.toggleCapability(item.id, item.kind === "tool" ? "tool" : "mcp", event.target.checked)}
                  />
                  <span><strong>{item.name}</strong><small>{item.description}</small></span>
                </label>
              ))}
            </div>
          </section>
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
                        runtime.toggleSkill(item.id, event.target.checked);
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
                  <input type="checkbox" checked={item.active} onChange={(event) => runtime.toggleSkill(item.id, event.target.checked)} />
                  <span><strong>{item.name}</strong><small>{item.description}</small></span>
                </label>
              ))}
            </div>
          </section>
        </div>
      ) : null}

      {sessionProfileInspectorOpen && runtime.route === "agent-runtime" ? (
        <div className="conversation-inspector-sections">
          {renderAgentDeliverablesSection(language, activeAgentDeliverables, sessionProfileAttributes)}
          <section className="conversation-inspector-section">
            <strong>{language === "zh" ? "实例属性" : "Instance Attributes"}</strong>
            <div className="workspace-details-summary">
              {sessionProfileFields.map((field) => {
                const value = sessionProfileAttributes[field.key] || "-";
                const mono = field.readonly === true || field.key.includes("path") || field.key.includes("branch");
                return (
                  <RouteFieldRow
                    key={field.key}
                    label={field.label}
                    value={value}
                    copyLabel={language === "zh" ? "复制值" : "Copy value"}
                    copyable={field.readonly !== false}
                    mono={mono}
                    multiline={value.length > 48}
                    markdown={!mono}
                  />
                );
              })}
            </div>
          </section>
        </div>
      ) : null}
    </div>
  ) : null;

  return (
    <RuntimeComposer
      runtimeKind={runtime.route === "agent-runtime" ? "agent" : "chat"}
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
      attachmentStripProps={{ "data-runtime-attachments": "conversation" }}
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
        placeholder: composerPlaceholder,
      }}
      onInputChange={composerRuntime.setDraft}
      onInputFocus={() => onInputFocusedChange(true)}
      onInputBlur={() => onInputFocusedChange(false)}
      onInputPointerDownCapture={handleComposerPointerDownCapture}
      onInputTouchStartCapture={handleComposerTouchStartCapture}
      utilityButtons={[
        {
          key: "session",
          label: copy.runtimeMobile,
          icon: <RuntimeSessionControlIcon />,
          className: runtime.inspectorOpen ? "is-active" : undefined,
          onClick: () => {
            if (mobileSessionGestureLockRef.current) {
              return;
            }
            toggleSessionInspector();
          },
          buttonProps: {
            onPointerDownCapture: handleSessionUtilityPointerDownCapture,
            onTouchStartCapture: handleSessionUtilityTouchStartCapture,
          },
        },
      ]}
      panelContent={conversationComposerPanel}
      onPanelDismiss={() => runtime.closeInspector()}
      panelProps={{
        "data-runtime-config-surface": "conversation",
      }}
      metaContent={composerMetaLabel}
      addAttachmentLabel={composerAddAttachmentLabel}
      onAddAttachment={handleComposerAttachmentPicker}
      submitButtonProps={{
        onPointerDownCapture: handleSubmitPointerDownCapture,
        onTouchStartCapture: handleSubmitTouchStartCapture,
      }}
      submitLabel={composerSend}
      previewCloseLabel={composerClosePreviewLabel}
    />
  );
}

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
