import { useEffect, useMemo, useRef, useState, type PointerEvent, type TouchEvent } from "react";
import { useWorkbenchContext } from "../../app/WorkbenchContext";
import { formatDateTime } from "../../shared/time/format";
import { groupSessionListItems } from "../../shared/time/sessionListGroups";
import { buildChatTimelineItems } from "../shell/components/ChatMessageRegion";
import { normalizeText, RouteFieldRow } from "../shell/components/RouteBodyPrimitives";
import { RuntimeWorkspacePage, type RuntimeWorkspacePageController } from "../shell/components/RuntimeWorkspacePage";
import { useRuntimeComposerViewportSync } from "../shell/components/useRuntimeComposerViewportSync";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../shell/legacyShellCopy";
import {
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  readComposerImageFiles,
  type ComposerImageAttachment,
} from "./composerImageAttachments";
import { useConversationRuntime } from "./ConversationRuntimeProvider";

type ConversationWorkspaceProps = {
  language: LegacyShellLanguage;
};

export function useConversationRuntimeController(language: LegacyShellLanguage): RuntimeWorkspacePageController {
  const workbench = useWorkbenchContext();
  const runtime = useConversationRuntime();
  const copy = getLegacyShellCopy(language);
  const [inputFocused, setInputFocused] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [composerAttachmentError, setComposerAttachmentError] = useState("");
  const [previewAttachment, setPreviewAttachment] = useState<ComposerImageAttachment | null>(null);
  const composerInputRef = useRef<HTMLTextAreaElement | null>(null);
  const composerFileInputRef = useRef<HTMLInputElement | null>(null);
  const composerShellRef = useRef<HTMLElement | null>(null);
  const workspaceBodyRef = useRef<HTMLDivElement | null>(null);
  const activeMessages = runtime.activeSession?.messages || [];
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
  const composerPlaceholder = language === "zh" ? "输入消息，继续推进当前工作区..." : "Type a message to continue this workspace...";
  const composerSend = language === "zh" ? "发送" : "Send";
  const sessionPaneTitle = runtime.route === "agent-runtime"
    ? (language === "zh" ? "Agent 会话" : "Agent sessions")
    : (language === "zh" ? "对话会话" : "Chat sessions");
  const mobileNewLabel = runtime.route === "agent-runtime" ? copy.sessionNewAgent : copy.sessionNewChat;
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
  const composerMetaLabel = composerAttachmentError
    || `${runtime.draft.length}/${10000}`;
  const composerAddImageLabel = language === "zh" ? "添加图片" : "Add image";
  const composerClosePreviewLabel = language === "zh" ? "关闭预览" : "Close preview";
  const composerPreviewPrefix = language === "zh" ? "预览" : "Preview";
  const composerRemovePrefix = language === "zh" ? "删除" : "Remove";
  const composerImageLimitError = language === "zh"
    ? `最多可暂存 ${MAX_COMPOSER_IMAGE_ATTACHMENTS} 张图片。`
    : `You can attach up to ${MAX_COMPOSER_IMAGE_ATTACHMENTS} images.`;
  const composerVisionUnsupported = language === "zh"
    ? "当前模型不支持图片输入，请切换到支持视觉的模型后再发送。"
    : "The selected model does not support image input. Switch to a vision-capable model before sending.";
  const detailsLabel = language === "zh" ? "详情" : "Details";
  const routeLabel = runtime.route === "agent-runtime"
    ? (language === "zh" ? "Agent Runtime" : "Agent Runtime")
    : "Chat";
  const activeSessionItem = runtime.sessionItems.find((item) => item.id === runtime.activeSession?.id) || null;
  const selectedProviderName = runtime.providers.find((provider) => provider.id === runtime.selectedProviderId)?.name
    || copy.runtimeServiceDefault;
  const activeCapabilityNames = capabilityListLabel(runtime.capabilities.filter((item) => item.active).map((item) => item.name));
  const activeSkillNames = capabilityListLabel(runtime.skills.filter((item) => item.active).map((item) => item.name));
  const runtimeStatus = resolveConversationWorkspaceStatus(activeMessages, language);
  const targetInspectorOpen = runtime.inspectorOpen && runtime.inspectorTab === "target";
  const modelInspectorOpen = runtime.inspectorOpen && runtime.inspectorTab === "model";
  const capabilitiesInspectorOpen = runtime.inspectorOpen && runtime.inspectorTab === "capabilities";
  const skillsInspectorOpen = runtime.inspectorOpen && runtime.inspectorTab === "skills";
  const sessionProfileInspectorOpen = runtime.inspectorOpen && runtime.inspectorTab === "session-profile";
  const sessionProfileFields = runtime.activeSessionProfile?.fields || runtime.activeAgent?.session_profile_fields || [];

  useEffect(() => {
    workbench.closeMobileSessionPane();
  }, [runtime.route]);

  useEffect(() => {
    setDetailsOpen(false);
  }, [runtime.route, runtime.activeSession?.id]);

  const handleCreateSession = () => {
    runtime.createSession();
    workbench.closeMobileSessionPane();
  };

  const handleFocusSession = (sessionID: string) => {
    runtime.focusSession(sessionID);
    workbench.closeMobileSessionPane();
  };

  const handleRemoveSession = (sessionID: string) => {
    workbench.closeMobileSessionPane();
    return runtime.removeSession(sessionID);
  };

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
    if (runtime.draftAttachments.length > 0 && !runtime.selectedModelSupportsVision) {
      setComposerAttachmentError(composerVisionUnsupported);
      return;
    }
    void runtime.sendPrompt(runtime.draft);
  };

  const handleSubmitTouchStartCapture = (event: TouchEvent<HTMLButtonElement>) => {
    if (!workbench.isMobileViewport) {
      return;
    }
    event.preventDefault();
    submitDraft();
  };

  const capabilityGroups = useMemo(() => ({
    activeCapabilities: runtime.capabilities.filter((item) => item.active),
    availableCapabilities: runtime.capabilities.filter((item) => !item.active),
    activeSkills: runtime.skills.filter((item) => item.active),
    availableSkills: runtime.skills.filter((item) => !item.active),
  }), [runtime.capabilities, runtime.skills]);

  const handleComposerImagePicker = () => {
    composerFileInputRef.current?.click();
  };

  const handleComposerImageSelection = async (files: FileList | null) => {
    if (!files || files.length === 0) {
      return;
    }
    if ((runtime.draftAttachments.length + files.length) > MAX_COMPOSER_IMAGE_ATTACHMENTS) {
      setComposerAttachmentError(composerImageLimitError);
      return;
    }
    try {
      const attachments = await readComposerImageFiles(files);
      await runtime.addDraftAttachments(attachments);
      setComposerAttachmentError("");
    } catch (error) {
      setComposerAttachmentError(error instanceof Error ? error.message : "Failed to add image.");
    } finally {
      if (composerFileInputRef.current) {
        composerFileInputRef.current.value = "";
      }
    }
  };

  useRuntimeComposerViewportSync({
    isMobileViewport: workbench.isMobileViewport,
    inputFocused,
    workspaceBodyRef,
    composerShellRef,
  });

  return {
    shell: {
      rootClassName: "conversation-runtime-view",
      rootProps: { "data-conversation-view": runtime.route },
      sessionPaneClassName: workbench.isMobileViewport && workbench.mobileSessionPaneOpen ? "is-open" : undefined,
      sessionPaneProps: {
        "data-conversation-session-pane": "",
        "data-mobile-open": workbench.mobileSessionPaneOpen ? "true" : "false",
        "data-testid": "conversation-session-pane",
      },
      sessionPaneBackdrop: {
        ariaLabel: copy.sessionClose,
        onClick: workbench.closeMobileSessionPane,
      },
      sessionPanePrimaryActionClassName: "is-primary",
      sessionPaneTitle,
      sessionPaneCountLabel: sessionCountLabel,
      sessionPanePrimaryActionLabel: workbench.isMobileViewport ? copy.terminalNewShort : mobileNewLabel,
      onSessionPanePrimaryAction: handleCreateSession,
      sessionPaneSecondaryActionLabel: workbench.isMobileViewport ? copy.sessionClose : undefined,
      onSessionPaneSecondaryAction: workbench.isMobileViewport ? workbench.closeMobileSessionPane : undefined,
      workspaceProps: {
        "data-conversation-workspace": "",
        "data-conversation-route": runtime.route,
      },
      workspaceBodyRef,
      mobileHeaderPlacement: workbench.isMobileViewport ? "body" : undefined,
      mobileHeaderProps: { "data-conversation-mobile-header": "" },
      mobileNavButtonClassName: "is-quiet conversation-mobile-nav-toggle",
      mobileNavButtonLabel: copy.chatMenu,
      mobileNavButtonProps: { "aria-expanded": workbench.mobileNavOpen },
      onMobileNav: workbench.toggleMobileNav,
      mobileSessionButtonClassName: "is-quiet conversation-mobile-session-toggle",
      mobileSessionButtonLabel: copy.chatSessions,
      mobileSessionButtonProps: { "aria-expanded": workbench.mobileSessionPaneOpen },
      onMobileSession: workbench.toggleMobileSessionPane,
      mobilePrimaryButtonClassName: "is-primary conversation-mobile-new-session",
      mobilePrimaryButtonLabel: copy.terminalNewShort,
      onMobilePrimary: handleCreateSession,
    },
    sessionList: {
      groups: groupedSessionItems.map((group) => ({
        ...group,
        items: group.items.map((item) => ({
          id: item.id,
          active: item.active,
          title: item.title,
          meta: item.meta,
          shortHash: item.shortHash,
          activeLabel: activeSessionBadgeLabel,
          idleLabel: idleSessionBadgeLabel,
          onSelect: () => handleFocusSession(item.id),
          onDelete: () => void handleRemoveSession(item.id),
          deleteLabel: deleteSessionLabel,
          deleteAriaLabel: deleteSessionAriaLabel,
          shellClassName: item.active ? "runtime-session-card is-active" : "runtime-session-card",
          shellProps: { "data-conversation-session-state": item.active ? "active" : "idle" },
          buttonClassName: item.active ? "runtime-session-select active" : "runtime-session-select",
        })),
      })),
      listProps: { "data-conversation-session-list": "true" },
      emptyState: groupedSessionItems.length === 0 ? (
        <p className="route-empty-panel conversation-session-empty">{sessionEmptyLabel}</p>
      ) : null,
    },
    header: {
      title: runtime.activeSession?.title || emptyStateTitle,
      statusLabel: runtimeStatus.label,
      statusTone: runtimeStatus.tone,
      detailsLabel,
      detailsOpen,
      onToggleDetails: () => setDetailsOpen((current) => !current),
      detailsDisabled: !runtime.activeSession,
      mobileEmpty: isMobileEmptyHeader,
      detailsPanelProps: {
        "data-conversation-inspector": "",
      },
      detailsClassName: "conversation-inspector conversation-session-details workspace-details-content",
      detailsSummary: runtime.activeSession ? [
        { label: language === "zh" ? "会话" : "Session", value: runtime.activeSession.id, copyLabel: language === "zh" ? "会话" : "Session", mono: true },
        { label: language === "zh" ? "路由" : "Route", value: routeLabel, copyLabel: language === "zh" ? "路由" : "Route" },
        { label: language === "zh" ? "状态" : "Status", value: runtimeStatus.label, copyLabel: language === "zh" ? "状态" : "Status" },
        { label: language === "zh" ? "短标识" : "Short hash", value: activeSessionItem?.shortHash || "-", copyLabel: language === "zh" ? "短标识" : "Short hash", mono: true },
        { label: copy.runtimeModel, value: runtime.selectedModelLabel || copy.runtimeServiceDefault, copyLabel: copy.runtimeModel },
        { label: copy.runtimeProvider, value: selectedProviderName, copyLabel: copy.runtimeProvider },
        ...(runtime.route === "agent-runtime" ? [{ label: copy.runtimeAgent, value: runtime.target.name || "-", copyLabel: copy.runtimeAgent }] : []),
        { label: language === "zh" ? "消息数" : "Messages", value: String(activeMessages.length), copyLabel: language === "zh" ? "消息数" : "Messages" },
        { label: language === "zh" ? "创建时间" : "Created", value: activeSessionItem ? formatDateTime(activeSessionItem.createdAt) : "-", copyLabel: language === "zh" ? "创建时间" : "Created" },
        { label: copy.runtimeToolsMcp, value: activeCapabilityNames, copyLabel: copy.runtimeToolsMcp, multiline: true },
        { label: copy.runtimeSkills, value: activeSkillNames, copyLabel: copy.runtimeSkills, multiline: true },
      ] : [],
      detailsBody: runtime.activeSession ? (
        <div className="conversation-inspector-body">
          <div className="conversation-inspector-tabs">
            {runtime.route === "agent-runtime" ? (
              <button className={targetInspectorOpen ? "is-active" : ""} type="button" onClick={() => runtime.toggleInspector("target")}>
                {copy.runtimeAgentPick}
              </button>
            ) : null}
            <button className={modelInspectorOpen ? "is-active" : ""} type="button" onClick={() => runtime.toggleInspector("model")}>
              {copy.runtimeModel}
            </button>
            <button className={capabilitiesInspectorOpen ? "is-active" : ""} type="button" onClick={() => runtime.toggleInspector("capabilities")}>
              {copy.runtimeToolsMcp}
            </button>
            <button className={skillsInspectorOpen ? "is-active" : ""} type="button" onClick={() => runtime.toggleInspector("skills")}>
              {copy.runtimeSkills}
            </button>
            {runtime.route === "agent-runtime" && sessionProfileFields.length > 0 ? (
              <button className={sessionProfileInspectorOpen ? "is-active" : ""} type="button" onClick={() => runtime.toggleInspector("session-profile")}>
                {language === "zh" ? "Session Profile" : "Session Profile"}
              </button>
            ) : null}
          </div>

          {targetInspectorOpen && runtime.route === "agent-runtime" ? (
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
                      <input type="checkbox" checked={item.active} onChange={(event) => runtime.toggleSkill(item.id, event.target.checked)} />
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
              <section className="conversation-inspector-section">
                <strong>{language === "zh" ? "实例属性" : "Instance Attributes"}</strong>
                <div className="workspace-details-summary">
                  {sessionProfileFields.map((field) => {
                    const value = runtime.activeSessionProfile?.attributes?.[field.key] || "-";
                    return (
                      <RouteFieldRow
                        key={field.key}
                        label={field.label}
                        value={value}
                        copyLabel={language === "zh" ? "复制值" : "Copy value"}
                        copyable={field.readonly !== false}
                        mono={field.readonly === true || field.key.includes("path") || field.key.includes("branch")}
                        multiline={value.length > 48}
                      />
                    );
                  })}
                </div>
              </section>
            </div>
          ) : null}
        </div>
      ) : null,
    },
    screen: {
      panelClassName: `conversation-console-panel${isEmptyState ? " is-empty" : ""}`,
      screenClassName: isEmptyState ? "is-empty" : undefined,
      screenProps: { "data-conversation-chat-screen": "" },
    },
    timeline: {
      items: buildChatTimelineItems({
        messages: activeMessages,
        language,
        onToggleProcess: runtime.toggleAgentProcess,
      }),
      emptyState: (
        <div className="conversation-empty-state">
          <h5>{emptyStateTitle}</h5>
          <p>{emptyStateDescription}</p>
        </div>
      ),
    },
    composer: {
      shellRef: composerShellRef,
      onSubmit: (event) => {
        event.preventDefault();
        submitDraft();
      },
      fileInputRef: composerFileInputRef,
      onFileChange: (event) => {
        void handleComposerImageSelection(event.target.files);
      },
      attachments: runtime.draftAttachments,
      attachmentStripProps: { "data-composer-attachments": "" },
      attachmentPreviewLabel: (attachment) => `${composerPreviewPrefix} ${attachment.name}`,
      attachmentRemoveLabel: (attachment) => `${composerRemovePrefix} ${attachment.name}`,
      previewAttachment,
      onPreviewAttachmentChange: setPreviewAttachment,
      onRemoveAttachment: (attachment) => runtime.removeDraftAttachment(attachment.id),
      inputLabel: composerPlaceholder,
      inputId: "conversationRuntimeInput",
      inputRef: composerInputRef,
      inputValue: runtime.draft,
      inputProps: { maxLength: 10000, placeholder: composerPlaceholder },
      onInputChange: runtime.setDraft,
      onInputFocus: () => setInputFocused(true),
      onInputBlur: () => setInputFocused(false),
      onInputPointerDownCapture: handleComposerPointerDownCapture,
      onInputTouchStartCapture: handleComposerTouchStartCapture,
      metaContent: composerMetaLabel,
      addAttachmentLabel: composerAddImageLabel,
      onAddAttachment: handleComposerImagePicker,
      submitButtonProps: { onTouchStartCapture: handleSubmitTouchStartCapture },
      submitLabel: composerSend,
      submitIcon: (
        <svg viewBox="0 0 20 20" fill="none" focusable="false">
          <path d="M10 14.75V5.25" stroke="currentColor" strokeWidth="2.35" strokeLinecap="round" />
          <path d="M5.75 9.5 10 5.25l4.25 4.25" stroke="currentColor" strokeWidth="2.35" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      ),
      previewCloseLabel: composerClosePreviewLabel,
    },
  };
}

export function ConversationWorkspace({ language }: ConversationWorkspaceProps) {
  const controller = useConversationRuntimeController(language);
  return <RuntimeWorkspacePage controller={controller} />;
}

function capabilityListLabel(items: string[]) {
  const values = items.map((item) => normalizeText(item)).filter((item) => item !== "-");
  return values.length > 0 ? values.join(", ") : "-";
}

function resolveConversationWorkspaceStatus(
  messages: Array<{
    role?: string;
    status?: string;
    error?: boolean;
    taskPending?: boolean;
    taskStatus?: string;
  }>,
  language: LegacyShellLanguage,
): {
  tone: "ready" | "busy" | "failed" | "interrupted";
  label: string;
} {
  const latestAssistant = [...messages].reverse().find((message) => message.role === "assistant");
  const normalizedStatus = normalizeText(latestAssistant?.taskPending ? latestAssistant?.taskStatus : latestAssistant?.status).toLowerCase();

  if (latestAssistant?.error || normalizedStatus === "error" || normalizedStatus === "failed") {
    return {
      tone: "failed",
      label: language === "zh" ? "失败" : "Failed",
    };
  }

  if (normalizedStatus === "canceled" || normalizedStatus === "cancelled" || normalizedStatus === "interrupted") {
    return {
      tone: "interrupted",
      label: language === "zh" ? "已取消" : "Canceled",
    };
  }

  if (
    latestAssistant?.taskPending
    || normalizedStatus === "streaming"
    || normalizedStatus === "queued"
    || normalizedStatus === "running"
    || normalizedStatus === "in_progress"
    || normalizedStatus === "busy"
    || normalizedStatus === "starting"
  ) {
    return {
      tone: "busy",
      label: language === "zh" ? "执行中" : "Running",
    };
  }

  return {
    tone: "ready",
    label: language === "zh" ? "就绪" : "Ready",
  };
}
