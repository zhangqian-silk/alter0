import { useEffect, useLayoutEffect, useMemo, useRef, useState, type PointerEvent, type TouchEvent } from "react";
import { useWorkbenchContext } from "../../app/WorkbenchContext";
import { ChatMessageRegion } from "../shell/components/ChatMessageRegion";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../shell/legacyShellCopy";
import { useConversationRuntime } from "./ConversationRuntimeProvider";

type ConversationWorkspaceProps = {
  language: LegacyShellLanguage;
};

export function ConversationWorkspace({ language }: ConversationWorkspaceProps) {
  const workbench = useWorkbenchContext();
  const runtime = useConversationRuntime();
  const copy = getLegacyShellCopy(language);
  const [sessionPaneOpen, setSessionPaneOpen] = useState(false);
  const [inputFocused, setInputFocused] = useState(false);
  const composerInputRef = useRef<HTMLTextAreaElement | null>(null);
  const activeMessages = runtime.activeSession?.messages || [];
  const isEmptyState = activeMessages.length === 0;
  const isCompactRuntimeHeader = runtime.route === "chat" || runtime.route === "agent-runtime";
  const isMobileEmptyState = workbench.isMobileViewport && isEmptyState;
  const isMobileEmptyHeader = workbench.isMobileViewport && isEmptyState;
  const compactModelButtonLabel = isCompactRuntimeHeader && workbench.isMobileViewport ? copy.runtimeModelShort : copy.runtimeModel;
  const compactSecondaryButtonLabel = runtime.route === "agent-runtime"
    ? (isCompactRuntimeHeader && workbench.isMobileViewport ? copy.runtimeAgent : copy.runtimeAgentPick)
    : (isCompactRuntimeHeader && workbench.isMobileViewport ? copy.runtimeToolsShort : copy.runtimeToolsMcp);
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
  const composerMetaLabel = `${sessionCountLabel} · ${runtime.draft.length}/${10000}`;

  useEffect(() => {
    if (!workbench.isMobileViewport || workbench.mobileNavOpen) {
      setSessionPaneOpen(false);
    }
  }, [workbench.isMobileViewport, workbench.mobileNavOpen]);

  useEffect(() => {
    setSessionPaneOpen(false);
  }, [runtime.route]);

  const handleCreateSession = () => {
    runtime.createSession();
    setSessionPaneOpen(false);
  };

  const handleFocusSession = (sessionID: string) => {
    runtime.focusSession(sessionID);
    setSessionPaneOpen(false);
  };

  const handleRemoveSession = (sessionID: string) => {
    setSessionPaneOpen(false);
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

  const toggleSessionPane = () => {
    if (!sessionPaneOpen) {
      workbench.closeMobileNav();
    }
    setSessionPaneOpen((current) => !current);
  };

  const capabilityGroups = useMemo(() => ({
    activeCapabilities: runtime.capabilities.filter((item) => item.active),
    availableCapabilities: runtime.capabilities.filter((item) => !item.active),
    activeSkills: runtime.skills.filter((item) => item.active),
    availableSkills: runtime.skills.filter((item) => !item.active),
  }), [runtime.capabilities, runtime.skills]);

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

  return (
    <section
      className="conversation-runtime-view terminal-runtime-view"
      data-conversation-view={runtime.route}
    >
      <aside
        className={`terminal-session-pane conversation-session-pane${sessionPaneOpen ? " is-open" : ""}`}
        data-conversation-session-pane
        data-mobile-open={sessionPaneOpen ? "true" : "false"}
        data-testid="conversation-session-pane"
      >
        <button
          className="terminal-session-pane-backdrop conversation-session-pane-backdrop"
          type="button"
          aria-label={copy.sessionClose}
          onClick={() => setSessionPaneOpen(false)}
        ></button>
        <div className="route-surface terminal-session-pane-shell conversation-session-pane-shell">
          <div className="terminal-session-pane-head conversation-session-pane-head">
            <div className="terminal-session-pane-copy conversation-session-pane-copy">
              <strong>{sessionPaneTitle}</strong>
              <span>{runtime.sessionItems.length}</span>
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
                  onClick={() => setSessionPaneOpen(false)}
                >
                  {copy.sessionClose}
                </button>
              ) : null}
            </div>
          </div>
          <div className="conversation-session-list" data-conversation-session-list>
            {runtime.sessionItems.map((item) => (
              <div
                key={item.id}
                className={item.active ? "conversation-session-card is-active" : "conversation-session-card"}
              >
                <button
                  className="conversation-session-select"
                  type="button"
                  aria-current={item.active ? "true" : undefined}
                  onClick={() => handleFocusSession(item.id)}
                >
                  <span className="conversation-session-title">{item.title}</span>
                  <span className="conversation-session-meta">{item.meta}</span>
                  <span className="conversation-session-hash">#{item.shortHash}</span>
                </button>
                <button
                  className="conversation-session-delete"
                  type="button"
                  aria-label={language === "zh" ? "删除会话" : "Delete session"}
                  onClick={() => void handleRemoveSession(item.id)}
                >
                  {language === "zh" ? "删除" : "Delete"}
                </button>
              </div>
            ))}
          </div>
        </div>
      </aside>

      <section
        className="terminal-workspace conversation-workspace"
        data-conversation-workspace
        data-conversation-route={runtime.route}
      >
        <div className="terminal-workspace-body conversation-workspace-body">
          {workbench.isMobileViewport ? (
            <header className="terminal-mobile-header" data-conversation-mobile-header>
              <button
                className="conversation-mobile-action terminal-inline-button is-quiet conversation-mobile-nav-toggle"
                type="button"
                aria-expanded={workbench.mobileNavOpen}
                onClick={() => {
                  setSessionPaneOpen(false);
                  workbench.toggleMobileNav();
                }}
              >
                {copy.chatMenu}
              </button>
              <div className="terminal-mobile-header-actions">
                <button
                  className="conversation-mobile-action terminal-inline-button is-quiet conversation-mobile-session-toggle"
                  type="button"
                  aria-expanded={sessionPaneOpen}
                  onClick={toggleSessionPane}
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

          <header className={`terminal-workspace-head conversation-workspace-head${isCompactRuntimeHeader ? " is-compact" : ""}${isMobileEmptyHeader ? " is-mobile-empty" : ""}`}>
              <div className={`terminal-workspace-row terminal-workspace-title-row conversation-workspace-row${isCompactRuntimeHeader ? " is-compact" : ""}${isMobileEmptyHeader ? " is-mobile-empty" : ""}`}>
                <div className={`terminal-workspace-copy conversation-workspace-copy${isCompactRuntimeHeader ? " is-compact" : ""}${isMobileEmptyHeader ? " is-mobile-empty" : ""}`}>
                  {!isCompactRuntimeHeader ? (
                    <span className="terminal-workspace-eyebrow conversation-workspace-eyebrow">
                      {runtime.route === "agent-runtime" ? copy.runtimeAgent : "Chat"}
                    </span>
                  ) : null}
                  <h4>{runtime.activeSession?.title || emptyStateTitle}</h4>
                  {!isCompactRuntimeHeader ? (
                    <span className="terminal-workspace-subcopy conversation-workspace-subcopy">
                      {runtime.route === "agent-runtime"
                        ? `${copy.runtimeAgent}: ${runtime.target.name || "-"}`
                        : `${runtime.selectedModelLabel || "Default"} · ${runtime.toolCount} / ${runtime.skillCount}`}
                    </span>
                  ) : null}
                </div>
                <div className={`terminal-workspace-actions conversation-workspace-actions${isMobileEmptyHeader ? " is-mobile-empty" : ""}`}>
                  <button
                    className="terminal-inline-button"
                    type="button"
                    onClick={() => runtime.toggleInspector("model")}
                  >
                    {compactModelButtonLabel}
                  </button>
                  <button
                    className="terminal-inline-button"
                    type="button"
                    onClick={() => runtime.toggleInspector(runtime.route === "agent-runtime" ? "target" : "capabilities")}
                  >
                    {compactSecondaryButtonLabel}
                  </button>
                </div>
              </div>

              {runtime.inspectorOpen ? (
                <section className="conversation-inspector" data-conversation-inspector>
                  <div className="conversation-inspector-tabs">
                    {runtime.route === "agent-runtime" ? (
                      <button
                        className={runtime.inspectorTab === "target" ? "is-active" : ""}
                        type="button"
                        onClick={() => runtime.toggleInspector("target")}
                      >
                        {copy.runtimeAgentPick}
                      </button>
                    ) : null}
                    <button
                      className={runtime.inspectorTab === "model" ? "is-active" : ""}
                      type="button"
                      onClick={() => runtime.toggleInspector("model")}
                    >
                      {copy.runtimeModel}
                    </button>
                    <button
                      className={runtime.inspectorTab === "capabilities" ? "is-active" : ""}
                      type="button"
                      onClick={() => runtime.toggleInspector("capabilities")}
                    >
                      {copy.runtimeToolsMcp}
                    </button>
                    <button
                      className={runtime.inspectorTab === "skills" ? "is-active" : ""}
                      type="button"
                      onClick={() => runtime.toggleInspector("skills")}
                    >
                      {copy.runtimeSkills}
                    </button>
                    <button type="button" onClick={() => runtime.closeInspector()}>
                      {language === "zh" ? "关闭" : "Close"}
                    </button>
                  </div>

                  {runtime.inspectorTab === "target" && runtime.route === "agent-runtime" ? (
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

                  {runtime.inspectorTab === "model" ? (
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

                  {runtime.inspectorTab === "capabilities" ? (
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

                  {runtime.inspectorTab === "skills" ? (
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
                </section>
              ) : null}
          </header>

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

          <footer className="terminal-composer-shell conversation-composer-shell">
            <form
              className="terminal-chat-form conversation-chat-form"
              onSubmit={(event) => {
                event.preventDefault();
                void runtime.sendPrompt();
              }}
            >
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
                  type="submit"
                  className="terminal-chat-submit conversation-chat-submit"
                  aria-label={composerSend}
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
        </div>
      </section>
    </section>
  );
}
