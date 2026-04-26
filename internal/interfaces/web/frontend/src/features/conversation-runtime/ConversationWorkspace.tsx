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
  isComposerImageAttachment,
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  readComposerFiles,
  type ComposerAttachment,
} from "./composerImageAttachments";
import { useConversationRuntime } from "./ConversationRuntimeProvider";

type ConversationWorkspaceProps = {
  language: LegacyShellLanguage;
};

function RuntimeSessionControlIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <rect x="3" y="3" width="5" height="5" rx="1.4" stroke="currentColor" strokeWidth="1.6" />
      <rect x="12" y="3" width="5" height="5" rx="1.4" stroke="currentColor" strokeWidth="1.6" />
      <rect x="3" y="12" width="5" height="5" rx="1.4" stroke="currentColor" strokeWidth="1.6" />
      <rect x="12" y="12" width="5" height="5" rx="1.4" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  );
}

export function useConversationRuntimeController(language: LegacyShellLanguage): RuntimeWorkspacePageController {
  const workbench = useWorkbenchContext();
  const runtime = useConversationRuntime();
  const copy = getLegacyShellCopy(language);
  const [inputFocused, setInputFocused] = useState(false);
  const [sessionDetailsOpen, setSessionDetailsOpen] = useState(false);
  const [composerAttachmentError, setComposerAttachmentError] = useState("");
  const [previewAttachment, setPreviewAttachment] = useState<ComposerAttachment | null>(null);
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
  const compactStatusLabel = language === "zh" ? "就绪" : "Ready";
  const compactDetailsLabel = language === "zh" ? "详情" : "Details";
  const inspectorTabOpen = runtime.inspectorOpen && runtime.inspectorTabOpen;
  const targetInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "target";
  const modelInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "model";
  const capabilitiesInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "capabilities";
  const skillsInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "skills";
  const sessionProfileInspectorOpen = inspectorTabOpen && runtime.inspectorTab === "session-profile";
  const sessionProfileFields = runtime.activeSessionProfile?.fields || runtime.activeAgent?.session_profile_fields || [];
  const activeSessionItem = runtime.sessionItems.find((item) => item.active) || null;
  const routeLabel = runtime.route === "agent-runtime"
    ? (language === "zh" ? "Agent" : "Agent")
    : (language === "zh" ? "对话" : "Chat");
  const conversationDetailsSummary = runtime.activeSession ? [
    { label: language === "zh" ? "会话" : "Session", value: runtime.activeSession.id, copyLabel: language === "zh" ? "会话" : "Session", mono: true },
    { label: language === "zh" ? "路由" : "Route", value: routeLabel, copyLabel: language === "zh" ? "路由" : "Route" },
    { label: language === "zh" ? "状态" : "Status", value: compactStatusLabel, copyLabel: language === "zh" ? "状态" : "Status" },
    { label: language === "zh" ? "短标识" : "Short hash", value: activeSessionItem?.shortHash || "-", copyLabel: language === "zh" ? "短标识" : "Short hash", mono: true },
    ...(runtime.route === "agent-runtime" ? [{ label: copy.runtimeAgent, value: runtime.target.name || "-", copyLabel: copy.runtimeAgent }] : []),
    { label: language === "zh" ? "消息数" : "Messages", value: String(activeMessages.length), copyLabel: language === "zh" ? "消息数" : "Messages" },
    { label: language === "zh" ? "创建时间" : "Created", value: activeSessionItem ? formatDateTime(activeSessionItem.createdAt) : "-", copyLabel: language === "zh" ? "创建时间" : "Created" },
  ] : [];

  useEffect(() => {
    workbench.closeMobileSessionPane();
  }, [runtime.route]);

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
    if (runtime.draftAttachments.some(isComposerImageAttachment) && !runtime.selectedModelSupportsVision) {
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
    availableSkills: runtime.skills.filter((item) => !item.active && item.visibility !== "agent-private"),
  }), [runtime.capabilities, runtime.skills]);

  const handleComposerAttachmentPicker = () => {
    composerFileInputRef.current?.click();
  };

  const handleComposerAttachmentSelection = async (files: FileList | null) => {
    if (!files || files.length === 0) {
      return;
    }
    if ((runtime.draftAttachments.length + files.length) > MAX_COMPOSER_IMAGE_ATTACHMENTS) {
      setComposerAttachmentError(composerImageLimitError);
      return;
    }
    try {
      const attachments = await readComposerFiles(files);
      await runtime.addDraftAttachments(attachments);
      setComposerAttachmentError("");
    } catch (error) {
      setComposerAttachmentError(error instanceof Error ? error.message : "Failed to add attachment.");
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

  const sessionDetailsBody = runtime.route === "agent-runtime" && sessionProfileFields.length > 0 ? (
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
  ) : null;

  const configPanelHint = targetInspectorOpen
    ? copy.runtimeAgentHint
    : modelInspectorOpen
      ? copy.runtimeModelHint
      : capabilitiesInspectorOpen
        ? copy.runtimeToolsHint
        : skillsInspectorOpen
          ? copy.runtimeSkillsHint
          : undefined;
  const configPanelTabs = [
    ...(runtime.route === "agent-runtime" ? [{
      key: "target" as const,
      label: copy.runtimeAgent,
    }] : []),
    {
      key: "model" as const,
      label: copy.runtimeModel,
    },
    {
      key: "capabilities" as const,
      label: copy.runtimeToolsShort,
    },
    {
      key: "skills" as const,
      label: copy.runtimeSkillsShort,
    },
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
                    onChange={(event) =>
                      runtime.toggleCapability(item.id, item.kind === "tool" ? "tool" : "mcp", event.target.checked)}
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
                    onChange={(event) =>
                      runtime.toggleCapability(item.id, item.kind === "tool" ? "tool" : "mcp", event.target.checked)}
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
  ) : null;

  return {
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
          shellProps: { "data-runtime-session-state": item.active ? "active" : "idle" },
          buttonClassName: item.active ? "runtime-session-select active" : "runtime-session-select",
        })),
      })),
      listProps: { "data-runtime-session-list": "conversation" },
      emptyState: groupedSessionItems.length === 0 ? (
        <p className="route-empty-panel">{sessionEmptyLabel}</p>
      ) : null,
    },
    header: {
      title: runtime.activeSession?.title || emptyStateTitle,
      statusLabel: compactStatusLabel,
      statusTone: "ready",
      detailsLabel: compactDetailsLabel,
      detailsOpen: sessionDetailsOpen,
      onToggleDetails: () => setSessionDetailsOpen((current) => !current),
      detailsDisabled: false,
      mobileEmpty: isMobileEmptyHeader,
      detailsClassName: "conversation-inspector conversation-session-details workspace-details-content",
      detailsSummary: conversationDetailsSummary,
      detailsBody: runtime.activeSession ? sessionDetailsBody : null,
      headerProps: { "data-runtime-header-kind": "conversation" },
      detailsPanelProps: {
        "data-runtime-details-panel": "conversation",
        "data-conversation-session-details": "",
      },
    },
    screen: {
      panelClassName: `conversation-console-panel${isEmptyState ? " is-empty" : ""}`,
      screenClassName: isEmptyState
        ? "is-empty"
        : undefined,
      screenProps: { "data-runtime-screen": "conversation" },
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
      runtimeKind: runtime.route === "agent-runtime" ? "agent" : "chat",
      shellRef: composerShellRef,
      onSubmit: (event) => {
        event.preventDefault();
        submitDraft();
      },
      fileInputRef: composerFileInputRef,
      fileInputAccept: "image/*,.txt,.md,.json,.yaml,.yml,.csv,.log,.pdf",
      onFileChange: (event) => {
        void handleComposerAttachmentSelection(event.target.files);
      },
      attachments: runtime.draftAttachments,
      attachmentStripProps: { "data-runtime-attachments": "conversation" },
      attachmentPreviewLabel: (attachment) => `${composerPreviewPrefix} ${attachment.name}`,
      attachmentRemoveLabel: (attachment) => `${composerRemovePrefix} ${attachment.name}`,
      previewAttachment,
      onPreviewAttachmentChange: setPreviewAttachment,
      onRemoveAttachment: (attachment) => runtime.removeDraftAttachment(attachment.id),
      inputLabel: composerPlaceholder,
      inputId: "conversationRuntimeInput",
      inputRef: composerInputRef,
      inputValue: runtime.draft,
      inputProps: {
        maxLength: 10000,
        placeholder: composerPlaceholder,
      },
      onInputChange: runtime.setDraft,
      onInputFocus: () => setInputFocused(true),
      onInputBlur: () => setInputFocused(false),
      onInputPointerDownCapture: handleComposerPointerDownCapture,
      onInputTouchStartCapture: handleComposerTouchStartCapture,
      utilityButtons: [
        {
          key: "session",
          label: copy.runtimeMobile,
          icon: <RuntimeSessionControlIcon />,
          className: runtime.inspectorOpen ? "is-pill is-active" : "is-pill",
          onClick: () => runtime.toggleInspector(runtime.inspectorTab),
        },
      ],
      panelContent: conversationComposerPanel,
      panelProps: {
        "data-runtime-config-surface": "conversation",
      },
      metaContent: composerMetaLabel,
      addAttachmentLabel: composerAddAttachmentLabel,
      onAddAttachment: handleComposerAttachmentPicker,
      submitButtonProps: { onTouchStartCapture: handleSubmitTouchStartCapture },
      submitLabel: composerSend,
      previewCloseLabel: composerClosePreviewLabel,
    },
  };
}

export function ConversationWorkspace({ language }: ConversationWorkspaceProps) {
  const controller = useConversationRuntimeController(language);
  return <RuntimeWorkspacePage controller={controller} />;
}
