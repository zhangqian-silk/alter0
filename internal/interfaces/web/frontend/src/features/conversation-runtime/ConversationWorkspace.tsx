import { useEffect, useMemo, useState } from "react";
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
  const activeMessages = runtime.activeSession?.messages || [];
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

  return (
    <section className="conversation-runtime-view" data-conversation-view={runtime.route}>
      <aside
        className={`conversation-session-pane${sessionPaneOpen ? " is-open" : ""}`}
        data-conversation-session-pane
        data-mobile-open={sessionPaneOpen ? "true" : "false"}
        data-testid="conversation-session-pane"
      >
        <button
          className="conversation-session-pane-backdrop"
          type="button"
          aria-label={copy.sessionClose}
          onClick={() => setSessionPaneOpen(false)}
        ></button>
        <div className="conversation-session-pane-shell">
          <div className="conversation-session-pane-head">
            <div className="conversation-session-pane-copy">
              <strong>{sessionPaneTitle}</strong>
              <span>{runtime.sessionItems.length}</span>
            </div>
            <div className="conversation-session-pane-actions">
              <button
                className="conversation-session-pane-action is-primary"
                type="button"
                onClick={handleCreateSession}
              >
                {workbench.isMobileViewport ? copy.terminalNewShort : mobileNewLabel}
              </button>
              {workbench.isMobileViewport ? (
                <button
                  className="conversation-session-pane-action"
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
        className="conversation-workspace"
        data-conversation-workspace
        data-conversation-route={runtime.route}
      >
        <div className="conversation-workspace-body">
          <header className="conversation-workspace-head">
            {workbench.isMobileViewport ? (
              <div className="conversation-mobile-actions">
                <button
                  className="conversation-mobile-action"
                  type="button"
                  aria-expanded={workbench.mobileNavOpen}
                  onClick={() => {
                    setSessionPaneOpen(false);
                    workbench.toggleMobileNav();
                  }}
                >
                  {copy.chatMenu}
                </button>
                <button
                  className="conversation-mobile-action"
                  type="button"
                  aria-expanded={sessionPaneOpen}
                  onClick={toggleSessionPane}
                >
                  {copy.chatSessions}
                </button>
                <button
                  className="conversation-mobile-action is-primary"
                  type="button"
                  onClick={handleCreateSession}
                >
                  {mobileNewLabel}
                </button>
              </div>
            ) : null}
            <div className="conversation-workspace-row">
              <div className="conversation-workspace-copy">
                <span className="conversation-workspace-eyebrow">
                  {runtime.route === "agent-runtime" ? copy.runtimeAgent : "Chat"}
                </span>
                <h4>{runtime.activeSession?.title || emptyStateTitle}</h4>
                <span className="conversation-workspace-subcopy">
                  {runtime.route === "agent-runtime"
                    ? `${copy.runtimeAgent}: ${runtime.target.name || "-"}`
                    : `${runtime.selectedModelLabel || "Default"} · ${runtime.toolCount} / ${runtime.skillCount}`}
                </span>
              </div>
              <div className="conversation-workspace-actions">
                <button
                  className="terminal-inline-button"
                  type="button"
                  onClick={() => runtime.toggleInspector("model")}
                >
                  {copy.runtimeModel}
                </button>
                <button
                  className="terminal-inline-button"
                  type="button"
                  onClick={() => runtime.toggleInspector(runtime.route === "agent-runtime" ? "target" : "capabilities")}
                >
                  {runtime.route === "agent-runtime" ? copy.runtimeAgentPick : copy.runtimeToolsMcp}
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

          <section className="conversation-console-panel">
            <div className="conversation-chat-screen" data-conversation-chat-screen>
              {activeMessages.length ? (
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

          <footer className="conversation-composer-shell">
            <form
              className="conversation-chat-form"
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
                className="conversation-composer-input"
                value={runtime.draft}
                maxLength={10000}
                placeholder={composerPlaceholder}
                onChange={(event) => runtime.setDraft(event.target.value)}
              ></textarea>
              <button type="submit" className="conversation-chat-submit">
                {composerSend}
              </button>
            </form>
            <div className="conversation-composer-meta">
              {runtime.draft.length}/{10000}
            </div>
          </footer>
        </div>
      </section>
    </section>
  );
}
