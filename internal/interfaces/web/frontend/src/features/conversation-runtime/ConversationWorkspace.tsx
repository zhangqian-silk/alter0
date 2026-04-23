import { useEffect, useLayoutEffect, useMemo, useRef, useState, type PointerEvent, type TouchEvent } from "react";
import { useWorkbenchContext } from "../../app/WorkbenchContext";
import { formatDateTime } from "../../shared/time/format";
import { groupSessionListItems } from "../../shared/time/sessionListGroups";
import { ChatMessageRegion } from "../shell/components/ChatMessageRegion";
import { normalizeText, RouteFieldRow } from "../shell/components/RouteBodyPrimitives";
import { RuntimeWorkspaceFrame } from "../shell/components/RuntimeWorkspaceFrame";
import { RuntimeWorkspaceHeader } from "../shell/components/RuntimeWorkspaceHeader";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../shell/legacyShellCopy";
import {
  MAX_COMPOSER_IMAGE_ATTACHMENTS,
  readComposerImageFiles,
  resolveComposerAttachmentPreviewURL,
  type ComposerImageAttachment,
} from "./composerImageAttachments";
import { useConversationRuntime } from "./ConversationRuntimeProvider";

type ConversationWorkspaceProps = {
  language: LegacyShellLanguage;
};

export function ConversationWorkspace({ language }: ConversationWorkspaceProps) {
  /* Source contract markers:
     className="conversation-chat-form"
     className="conversation-composer-input"
     className="conversation-chat-submit"
     className="conversation-session-list"
     data-testid="conversation-session-pane"
     data-conversation-view={runtime.route}
     workbench.toggleMobileNav();
  */
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

  useLayoutEffect(() => {
    if (!workbench.isMobileViewport || !inputFocused) {
      return;
    }
    const keepViewportAnchored = () => {
      if (window.scrollX !== 0 || window.scrollY !== 0) {
        window.scrollTo({ left: 0, top: 0, behavior: "auto" });
      }
    };
    const frameID = window.requestAnimationFrame(keepViewportAnchored);
    const visualViewport = window.visualViewport;
    window.addEventListener("scroll", keepViewportAnchored, { passive: true });
    visualViewport?.addEventListener("resize", keepViewportAnchored);
    visualViewport?.addEventListener("scroll", keepViewportAnchored);
    return () => {
      window.cancelAnimationFrame(frameID);
      window.removeEventListener("scroll", keepViewportAnchored);
      visualViewport?.removeEventListener("resize", keepViewportAnchored);
      visualViewport?.removeEventListener("scroll", keepViewportAnchored);
    };
  }, [inputFocused, workbench.isMobileViewport]);

  useLayoutEffect(() => {
    const workspaceBodyNode = workspaceBodyRef.current;
    const composerShellNode = composerShellRef.current;
    if (!workspaceBodyNode) {
      return;
    }
    if (!workbench.isMobileViewport || !composerShellNode) {
      workspaceBodyNode.style.removeProperty("--runtime-composer-inset");
      workspaceBodyNode.style.removeProperty("--runtime-composer-rest-inset");
      return;
    }

    const syncComposerInset = () => {
      const workspaceRect = workspaceBodyNode.getBoundingClientRect();
      const composerRect = composerShellNode.getBoundingClientRect();
      workspaceBodyNode.style.setProperty(
        "--runtime-composer-rest-inset",
        `${Math.max(0, Math.ceil(composerRect.height))}px`,
      );
      workspaceBodyNode.style.setProperty(
        "--runtime-composer-inset",
        `${Math.max(0, Math.ceil(workspaceRect.bottom - composerRect.top))}px`,
      );
    };

    let settleFrameID = 0;
    let settleLateFrameID = 0;
    let settleTimeoutID = 0;
    const clearScheduledSync = () => {
      if (settleFrameID) {
        window.cancelAnimationFrame(settleFrameID);
        settleFrameID = 0;
      }
      if (settleLateFrameID) {
        window.cancelAnimationFrame(settleLateFrameID);
        settleLateFrameID = 0;
      }
      if (settleTimeoutID) {
        window.clearTimeout(settleTimeoutID);
        settleTimeoutID = 0;
      }
    };
    const scheduleComposerInsetSync = () => {
      syncComposerInset();
      clearScheduledSync();
      settleFrameID = window.requestAnimationFrame(() => {
        settleFrameID = 0;
        syncComposerInset();
        settleLateFrameID = window.requestAnimationFrame(() => {
          settleLateFrameID = 0;
          syncComposerInset();
        });
      });
      settleTimeoutID = window.setTimeout(() => {
        settleTimeoutID = 0;
        syncComposerInset();
      }, 260);
    };

    scheduleComposerInsetSync();

    const resizeObserver = typeof ResizeObserver === "undefined"
      ? null
      : new ResizeObserver(() => scheduleComposerInsetSync());
    resizeObserver?.observe(composerShellNode);
    window.addEventListener("resize", scheduleComposerInsetSync);
    window.visualViewport?.addEventListener("resize", scheduleComposerInsetSync);
    window.visualViewport?.addEventListener("scroll", scheduleComposerInsetSync);
    composerShellNode.addEventListener("transitionend", scheduleComposerInsetSync);
    return () => {
      clearScheduledSync();
      resizeObserver?.disconnect();
      window.removeEventListener("resize", scheduleComposerInsetSync);
      window.visualViewport?.removeEventListener("resize", scheduleComposerInsetSync);
      window.visualViewport?.removeEventListener("scroll", scheduleComposerInsetSync);
      composerShellNode.removeEventListener("transitionend", scheduleComposerInsetSync);
      workspaceBodyNode.style.removeProperty("--runtime-composer-inset");
      workspaceBodyNode.style.removeProperty("--runtime-composer-rest-inset");
    };
  }, [workbench.isMobileViewport]);

  return (
    <>
      <RuntimeWorkspaceFrame
      rootClassName="conversation-runtime-view terminal-runtime-view"
      rootProps={{ "data-conversation-view": runtime.route }}
      sessionPaneClassName={`terminal-session-pane conversation-session-pane${workbench.isMobileViewport && workbench.mobileSessionPaneOpen ? " is-open" : ""}`}
      sessionPaneProps={{
        "data-conversation-session-pane": "",
        "data-mobile-open": workbench.mobileSessionPaneOpen ? "true" : "false",
        "data-testid": "conversation-session-pane",
      }}
      sessionPaneBackdrop={{
        className: "terminal-session-pane-backdrop conversation-session-pane-backdrop",
        ariaLabel: copy.sessionClose,
        onClick: workbench.closeMobileSessionPane,
      }}
      sessionPaneShellClassName="route-surface terminal-session-pane-shell conversation-session-pane-shell"
      sessionPaneContent={(
        <>
          <div className="terminal-session-pane-head conversation-session-pane-head">
            <div className="terminal-session-pane-copy conversation-session-pane-copy">
              <strong>{sessionPaneTitle}</strong>
              <span>{sessionCountLabel}</span>
            </div>
            <div className="terminal-session-pane-actions conversation-session-pane-actions">
              <button
                className="terminal-session-pane-action conversation-session-pane-action is-primary"
                type="button"
                onClick={handleCreateSession}
              >
                {workbench.isMobileViewport ? copy.terminalNewShort : mobileNewLabel}
              </button>
              {workbench.isMobileViewport ? (
                <button
                  className="terminal-session-pane-action conversation-session-pane-action"
                  type="button"
                  onClick={workbench.closeMobileSessionPane}
                >
                  {copy.sessionClose}
                </button>
              ) : null}
            </div>
          </div>
          <div className="conversation-session-list menu" data-conversation-session-list role="list">
            {groupedSessionItems.length === 0 ? <p className="route-empty-panel conversation-session-empty">{sessionEmptyLabel}</p> : null}
            {groupedSessionItems.map((group) => (
              <section
                key={group.key}
                className="conversation-session-group menu-group"
                aria-label={group.label}
              >
                <h2 className="conversation-session-group-label">{group.label}</h2>
                <div className="conversation-session-group-items">
                  {group.items.map((item) => (
                    <div
                      key={item.id}
                      role="listitem"
                      className={item.active ? "conversation-session-card is-active" : "conversation-session-card"}
                      data-conversation-session-state={item.active ? "active" : "idle"}
                    >
                      <button
                        className={item.active ? "conversation-session-select active" : "conversation-session-select"}
                        type="button"
                        aria-current={item.active ? "true" : undefined}
                        onClick={() => handleFocusSession(item.id)}
                      >
                        <span className="conversation-session-main">
                          <span className="conversation-session-topline">
                            <span
                              className={item.active ? "conversation-session-badge is-active" : "conversation-session-badge"}
                              data-conversation-session-badge={item.active ? "active" : "idle"}
                            >
                              {item.active ? activeSessionBadgeLabel : idleSessionBadgeLabel}
                            </span>
                          </span>
                          <span className="conversation-session-title-row">
                            <span className="conversation-session-title">{item.title}</span>
                          </span>
                          <span className="conversation-session-summary-row">
                            <span className="conversation-session-meta">{item.meta}</span>
                          </span>
                          <span className="conversation-session-bottomline">
                            <span className="conversation-session-hash">{item.shortHash}</span>
                          </span>
                        </span>
                      </button>
                      <button
                        className="conversation-session-delete"
                        type="button"
                        aria-label={deleteSessionAriaLabel}
                        onClick={() => void handleRemoveSession(item.id)}
                      >
                        {deleteSessionLabel}
                      </button>
                    </div>
                  ))}
                </div>
              </section>
            ))}
          </div>
        </>
      )}
      workspaceClassName="terminal-workspace conversation-workspace"
      workspaceProps={{
        "data-conversation-workspace": "",
        "data-conversation-route": runtime.route,
      }}
      workspaceBodyClassName="terminal-workspace-body conversation-workspace-body"
      workspaceBodyRef={workspaceBodyRef}
      mobileHeader={workbench.isMobileViewport ? (
        <header className="terminal-mobile-header" data-conversation-mobile-header>
          <button
            className="conversation-mobile-action terminal-inline-button is-quiet conversation-mobile-nav-toggle"
            type="button"
            aria-expanded={workbench.mobileNavOpen}
            onClick={workbench.toggleMobileNav}
          >
            {copy.chatMenu}
          </button>
          <div className="terminal-mobile-header-actions">
            <button
              className="conversation-mobile-action terminal-inline-button is-quiet conversation-mobile-session-toggle"
              type="button"
              aria-expanded={workbench.mobileSessionPaneOpen}
              onClick={workbench.toggleMobileSessionPane}
            >
              {copy.chatSessions}
            </button>
            <button
              className="conversation-mobile-action terminal-inline-button is-primary conversation-mobile-new-session"
              type="button"
              onClick={handleCreateSession}
            >
              {copy.terminalNewShort}
            </button>
          </div>
        </header>
      ) : null}
      workspaceHeader={(
        <RuntimeWorkspaceHeader
          title={runtime.activeSession?.title || emptyStateTitle}
          statusLabel={runtimeStatus.label}
          statusTone={runtimeStatus.tone}
          detailsLabel={detailsLabel}
          detailsOpen={detailsOpen}
          onToggleDetails={() => setDetailsOpen((current) => !current)}
          detailsDisabled={!runtime.activeSession}
          mobileEmpty={isMobileEmptyHeader}
          detailsPanelProps={{
            "data-conversation-inspector": "",
          }}
          detailsContent={runtime.activeSession ? (
            <section className="conversation-inspector conversation-session-details workspace-details-content">
              <div className="conversation-inspector-meta workspace-details-summary">
                <RouteFieldRow label={language === "zh" ? "会话" : "Session"} value={runtime.activeSession.id} copyLabel={language === "zh" ? "会话" : "Session"} mono />
                <RouteFieldRow label={language === "zh" ? "路由" : "Route"} value={routeLabel} copyLabel={language === "zh" ? "路由" : "Route"} />
                <RouteFieldRow label={language === "zh" ? "状态" : "Status"} value={runtimeStatus.label} copyLabel={language === "zh" ? "状态" : "Status"} />
                <RouteFieldRow label={language === "zh" ? "短标识" : "Short hash"} value={activeSessionItem?.shortHash || "-"} copyLabel={language === "zh" ? "短标识" : "Short hash"} mono />
                <RouteFieldRow label={copy.runtimeModel} value={runtime.selectedModelLabel || copy.runtimeServiceDefault} copyLabel={copy.runtimeModel} />
                <RouteFieldRow label={copy.runtimeProvider} value={selectedProviderName} copyLabel={copy.runtimeProvider} />
                {runtime.route === "agent-runtime" ? (
                  <RouteFieldRow label={copy.runtimeAgent} value={runtime.target.name || "-"} copyLabel={copy.runtimeAgent} />
                ) : null}
                <RouteFieldRow label={language === "zh" ? "消息数" : "Messages"} value={String(activeMessages.length)} copyLabel={language === "zh" ? "消息数" : "Messages"} />
                <RouteFieldRow
                  label={language === "zh" ? "创建时间" : "Created"}
                  value={activeSessionItem ? formatDateTime(activeSessionItem.createdAt) : "-"}
                  copyLabel={language === "zh" ? "创建时间" : "Created"}
                />
                <RouteFieldRow label={copy.runtimeToolsMcp} value={activeCapabilityNames} copyLabel={copy.runtimeToolsMcp} multiline />
                <RouteFieldRow label={copy.runtimeSkills} value={activeSkillNames} copyLabel={copy.runtimeSkills} multiline />
              </div>

              <div className="conversation-inspector-body workspace-details-body">
                <div className="conversation-inspector-tabs">
                  {runtime.route === "agent-runtime" ? (
                    <button
                      className={targetInspectorOpen ? "is-active" : ""}
                      type="button"
                      onClick={() => runtime.toggleInspector("target")}
                    >
                      {copy.runtimeAgentPick}
                    </button>
                  ) : null}
                  <button
                    className={modelInspectorOpen ? "is-active" : ""}
                    type="button"
                    onClick={() => runtime.toggleInspector("model")}
                  >
                    {copy.runtimeModel}
                  </button>
                  <button
                    className={capabilitiesInspectorOpen ? "is-active" : ""}
                    type="button"
                    onClick={() => runtime.toggleInspector("capabilities")}
                  >
                    {copy.runtimeToolsMcp}
                  </button>
                  <button
                    className={skillsInspectorOpen ? "is-active" : ""}
                    type="button"
                    onClick={() => runtime.toggleInspector("skills")}
                  >
                    {copy.runtimeSkills}
                  </button>
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
                            <span>
                              <strong>{item.name}</strong>
                              <small>{item.description}</small>
                            </span>
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
                            <span>
                              <strong>{item.name}</strong>
                              <small>{item.description}</small>
                            </span>
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
                              onChange={(event) => runtime.toggleSkill(item.id, event.target.checked)}
                            />
                            <span>
                              <strong>{item.name}</strong>
                              <small>{item.description}</small>
                            </span>
                          </label>
                        ))}
                      </div>
                    </section>
                    <section className="conversation-inspector-section">
                      <strong>{language === "zh" ? "可选" : "Available"}</strong>
                      <div className="conversation-check-list">
                        {capabilityGroups.availableSkills.map((item) => (
                          <label key={item.id} className="conversation-check-item">
                            <input
                              type="checkbox"
                              checked={item.active}
                              onChange={(event) => runtime.toggleSkill(item.id, event.target.checked)}
                            />
                            <span>
                              <strong>{item.name}</strong>
                              <small>{item.description}</small>
                            </span>
                          </label>
                        ))}
                      </div>
                    </section>
                  </div>
                ) : null}
              </div>
            </section>
          ) : null}
        />
      )}
      workspaceContent={(
        <section className={`conversation-console-panel${isEmptyState ? " is-empty" : ""}`}>
          <div
            className={`terminal-chat-screen conversation-chat-screen${isEmptyState ? " is-empty" : ""}`}
            data-conversation-chat-screen
          >
            {!isEmptyState ? (
              <ChatMessageRegion
                sessionId={runtime.activeSession?.id || ""}
                messages={activeMessages}
                language={language}
                onToggleProcess={runtime.toggleAgentProcess}
              />
            ) : (
              <div className="conversation-empty-state">
                <h5>{emptyStateTitle}</h5>
                <p>{emptyStateDescription}</p>
              </div>
            )}
          </div>
        </section>
      )}
      workspaceFooter={(
        <footer ref={composerShellRef} className="terminal-composer-shell conversation-composer-shell">
          <form
            className="terminal-chat-form conversation-chat-form"
            onSubmit={(event) => {
              event.preventDefault();
              submitDraft();
            }}
          >
            <input
              ref={composerFileInputRef}
              type="file"
              accept="image/*"
              hidden
              multiple
              onChange={(event) => {
                void handleComposerImageSelection(event.target.files);
              }}
            />
            {runtime.draftAttachments.length > 0 ? (
              <div className="conversation-composer-attachments" data-composer-attachments>
                {runtime.draftAttachments.map((attachment) => (
                  <article key={attachment.id} className="conversation-composer-attachment">
                    <button
                      type="button"
                      className="conversation-composer-attachment-preview"
                      aria-label={`${composerPreviewPrefix} ${attachment.name}`}
                      onClick={() => setPreviewAttachment(attachment)}
                    >
                      <img src={resolveComposerAttachmentPreviewURL(attachment)} alt={attachment.name} loading="lazy" decoding="async" />
                    </button>
                    <button
                      type="button"
                      className="conversation-composer-attachment-remove"
                      aria-label={`${composerRemovePrefix} ${attachment.name}`}
                      onClick={() => runtime.removeDraftAttachment(attachment.id)}
                    >
                      ×
                    </button>
                  </article>
                ))}
              </div>
            ) : null}
            <label className="sr-only" htmlFor="conversationRuntimeInput">
              {composerPlaceholder}
            </label>
            <textarea
              id="conversationRuntimeInput"
              ref={composerInputRef}
              className="terminal-composer-input conversation-composer-input"
              value={runtime.draft}
              maxLength={10000}
              placeholder={composerPlaceholder}
              onPointerDownCapture={handleComposerPointerDownCapture}
              onTouchStartCapture={handleComposerTouchStartCapture}
              onChange={(event) => runtime.setDraft(event.target.value)}
              onFocus={() => setInputFocused(true)}
              onBlur={() => setInputFocused(false)}
            ></textarea>
            <div className="terminal-composer-tools">
              <div className="terminal-composer-meta">{composerMetaLabel}</div>
              <button
                type="button"
                className="conversation-chat-upload"
                aria-label={composerAddImageLabel}
                onClick={handleComposerImagePicker}
              >
                <span aria-hidden="true">+</span>
                <span>{composerAddImageLabel}</span>
              </button>
              <button
                type="submit"
                className="terminal-chat-submit conversation-chat-submit"
                aria-label={composerSend}
                onTouchStartCapture={handleSubmitTouchStartCapture}
              >
                <span className="terminal-chat-form-button-icon" aria-hidden="true">
                  <svg viewBox="0 0 20 20" fill="none" focusable="false">
                    <path d="M10 14.75V5.25" stroke="currentColor" strokeWidth="2.35" strokeLinecap="round" />
                    <path d="M5.75 9.5 10 5.25l4.25 4.25" stroke="currentColor" strokeWidth="2.35" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </span>
                <span className="sr-only">{composerSend}</span>
              </button>
            </div>
          </form>
        </footer>
      )}
      />
      {previewAttachment ? (
        <div
          className="conversation-image-preview-backdrop"
          onClick={() => setPreviewAttachment(null)}
        >
          <div
            className="conversation-image-preview-dialog"
            role="dialog"
            aria-modal="true"
            aria-label={previewAttachment.name}
            onClick={(event) => event.stopPropagation()}
          >
            <button
              type="button"
              className="conversation-image-preview-close"
              aria-label={composerClosePreviewLabel}
              onClick={() => setPreviewAttachment(null)}
            >
              ×
            </button>
            <img src={resolveComposerAttachmentPreviewURL(previewAttachment)} alt={previewAttachment.name} decoding="async" />
          </div>
        </div>
      ) : null}
    </>
  );
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
