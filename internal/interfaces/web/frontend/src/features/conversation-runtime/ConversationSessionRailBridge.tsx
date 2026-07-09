import { useCallback, useEffect, useMemo } from "react";
import { useWorkbenchContext, type WorkbenchSessionRail } from "../../app/WorkbenchContext";
import { groupSessionListItems } from "../../shared/time/sessionListGroups";
import {
  RuntimeWorkspaceNavigationSessionList,
  type RuntimeWorkspacePageController,
  type RuntimeWorkspaceSessionItem,
} from "../shell/components/RuntimeWorkspacePage";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../shell/legacyShellCopy";
import { useConversationRuntimeWorkspace } from "./ConversationRuntimeProvider";

type ConversationSessionSignalTone = "ready" | "busy" | "failed";

function normalizeStatus(value: string | undefined) {
  return String(value || "").trim().toLowerCase();
}

function resolveSessionSignalTone(session: {
  status?: string;
  messages?: Array<{
    role?: string;
    status?: string;
    error?: boolean;
  }>;
}): ConversationSessionSignalTone {
  const status = normalizeStatus(session.status);
  if (["error", "failed", "canceled", "cancelled", "interrupted"].includes(status)) {
    return "failed";
  }
  if (["streaming", "queued", "running", "in_progress", "inprogress", "busy", "local_running", "recovering"].includes(status)) {
    return "busy";
  }
  if (status) {
    return "ready";
  }
  const messages = Array.isArray(session.messages) ? session.messages : [];
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message?.role !== "assistant") {
      continue;
    }
    const status = normalizeStatus(message.status);
    if (message.error || ["error", "failed", "canceled", "cancelled"].includes(status)) {
      return "failed";
    }
    if (["streaming", "queued", "running", "in_progress", "inprogress"].includes(status)) {
      return "busy";
    }
    return "ready";
  }
  return "ready";
}

function sessionStatusLabel(tone: ConversationSessionSignalTone, language: LegacyShellLanguage) {
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

export function ConversationSessionRailBridge({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const workbench = useWorkbenchContext();
  const runtime = useConversationRuntimeWorkspace();
  const copy = getLegacyShellCopy(language);
  const sessionCountLabel = language === "zh"
    ? `${runtime.sessionItems.length} 个会话`
    : `${runtime.sessionItems.length} sessions`;
  const activeSessionBadgeLabel = language === "zh" ? "当前" : "Current";
  const idleSessionBadgeLabel = language === "zh" ? "会话" : "Session";
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
      getTimestamp: (item) => item.updatedAt || item.createdAt,
      getPinned: (item) => item.pinned,
    }),
    [language, runtime.sessionItems],
  );
  const sessionStatusByID = useMemo(
    () => Object.fromEntries(
      runtime.sessions.map((session) => {
        const tone = resolveSessionSignalTone(session);
        return [session.id, {
          tone,
          label: sessionStatusLabel(tone, language),
        }];
      }),
    ) as Record<string, { tone: ConversationSessionSignalTone; label: string }>,
    [language, runtime.sessions],
  );
  const handleCreateSession = useCallback(() => {
    runtime.createSession();
    workbench.navigate("chat");
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
  const handlePinnedChange = useCallback((sessionID: string, pinned: boolean) => {
    void runtime.setSessionPinned(sessionID, pinned);
  }, [runtime]);
  const sessionListGroups = useMemo<RuntimeWorkspacePageController["sessionList"]["groups"]>(
    () => groupedSessionItems.map((group) => ({
      ...group,
      items: group.items.map((item): RuntimeWorkspaceSessionItem => {
        const isDraft = Boolean(item.draft);
        return {
          statusTone: sessionStatusByID[item.id]?.tone || "ready",
          statusLabel: sessionStatusByID[item.id]?.label || sessionStatusLabel("ready", language),
          id: item.id,
          active: item.active,
          title: item.title,
          contextLabel: item.contextLabel,
          meta: item.meta,
          activeLabel: activeSessionBadgeLabel,
          idleLabel: idleSessionBadgeLabel,
          onSelect: () => handleFocusSession(item.id),
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
          } as RuntimeWorkspaceSessionItem["shellProps"],
          buttonClassName: item.active ? "runtime-session-select active" : "runtime-session-select",
          buttonProps: {
            "data-runtime-session-select": item.id,
          } as RuntimeWorkspaceSessionItem["buttonProps"],
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
      idleSessionBadgeLabel,
      language,
      pinSessionAriaLabel,
      pinSessionLabel,
      sessionActionsLabel,
      sessionStatusByID,
      unpinSessionAriaLabel,
      unpinSessionLabel,
    ],
  );
  const sessionList = useMemo<RuntimeWorkspacePageController["sessionList"]>(() => ({
    groups: sessionListGroups,
    listProps: {
      "data-runtime-session-list": "conversation",
    } as RuntimeWorkspacePageController["sessionList"]["listProps"],
    emptyState: groupedSessionItems.length === 0 ? (
      <p className="route-empty-panel">{copy.sessionEmpty}</p>
    ) : null,
  }), [copy.sessionEmpty, groupedSessionItems.length, sessionListGroups]);
  const sessionPaneBody = useMemo(() => (
    <RuntimeWorkspaceNavigationSessionList sessionList={sessionList} />
  ), [sessionList]);
  const railVersionKey = useMemo(
    () => [
      language,
      sessionCountLabel,
      ...runtime.sessionItems.map((item) => [
        item.id,
        item.title,
        item.meta,
        item.active ? "active" : "idle",
        item.pinned ? "pinned" : "unpinned",
        item.pinning ? "pinning" : "idle-pin",
        sessionStatusByID[item.id]?.tone || "ready",
      ].join(":")),
    ].join("|"),
    [language, runtime.sessionItems, sessionCountLabel, sessionStatusByID],
  );
  const runtimeSessionRail = useMemo<WorkbenchSessionRail>(() => ({
    route: "chat",
    countLabel: sessionCountLabel,
    versionKey: railVersionKey,
    onPrimaryAction: handleCreateSession,
    primaryActionClassName: "is-primary",
    primaryActionProps: {
      "data-runtime-create-session": runtime.route,
    } as WorkbenchSessionRail["primaryActionProps"],
    body: sessionPaneBody,
  }), [
    handleCreateSession,
    railVersionKey,
    runtime.route,
    sessionCountLabel,
    sessionPaneBody,
  ]);

  useEffect(() => {
    workbench.setRuntimeSessionRail?.(runtimeSessionRail);
    return () => {
      workbench.setRuntimeSessionRail?.(null);
    };
  }, [runtimeSessionRail, workbench]);

  return null;
}
