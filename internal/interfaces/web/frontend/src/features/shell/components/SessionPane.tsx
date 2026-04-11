import { LEGACY_SHELL_IDS } from "../legacyDomContract";

export function SessionPane() {
  return (
    <aside className="session-pane">
      <header className="session-header">
        <h1 data-i18n="session.header">Work with alter0</h1>
        <button className="pane-action" id="togglePaneButton" type="button" aria-label="Toggle panel" data-i18n="session.close">
          Close
        </button>
      </header>

      <button className="new-chat" id={LEGACY_SHELL_IDS.newChatButton} type="button" data-i18n="session.new">
        New Chat
      </button>

      <div className="session-list-wrap">
        <div className="session-history-head">
          <p className="session-title" data-i18n="session.recent">Recent Sessions</p>
          <button className="session-history-toggle" id="sessionHistoryToggle" type="button" aria-controls="sessionHistoryPanel" aria-expanded="true">
            Collapse
          </button>
        </div>
        <div className="session-history-panel" id="sessionHistoryPanel">
          <div id={LEGACY_SHELL_IDS.sessionList} className="session-list" role="listbox" aria-label="Conversation sessions"></div>
          <p className="session-empty" id="sessionEmpty" data-i18n="session.empty">No sessions yet. Click New Chat to start.</p>
          <p className="session-error" id="sessionLoadError" role="status" aria-live="polite"></p>
        </div>
      </div>
    </aside>
  );
}
