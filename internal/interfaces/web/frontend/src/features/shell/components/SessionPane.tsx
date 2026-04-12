import { memo } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import {
  getLegacySessionHistoryToggleLabel,
  getLegacyShellCopy,
  type LegacyShellLanguage,
} from "../legacyShellCopy";

type SessionPaneProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  sessionHistoryCollapsed: boolean;
  onToggleSessionHistoryCollapsed: () => void;
};

const SessionListMount = memo(function SessionListMount({
  ariaLabel,
}: {
  ariaLabel: string;
}) {
  return (
    <div
      id={LEGACY_SHELL_IDS.sessionList}
      className="session-list"
      role="listbox"
      aria-label={ariaLabel}
    ></div>
  );
});

type SessionHistoryMountProps = {
  hidden: boolean;
  emptyLabel: string;
  listAriaLabel: string;
};

const SessionHistoryMount = memo(function SessionHistoryMount({
  hidden,
  emptyLabel,
  listAriaLabel,
}: SessionHistoryMountProps) {
  return (
    <div className="session-history-panel" id="sessionHistoryPanel" hidden={hidden}>
      <SessionListMount ariaLabel={listAriaLabel} />
      <p className="session-empty" id="sessionEmpty">{emptyLabel}</p>
      <p className="session-error" id="sessionLoadError" role="status" aria-live="polite"></p>
    </div>
  );
});

export const SessionPane = memo(function SessionPane({
  currentRoute,
  language,
  sessionHistoryCollapsed,
  onToggleSessionHistoryCollapsed,
}: SessionPaneProps) {
  const copy = getLegacyShellCopy(language);
  const newChatLabel = currentRoute === "agent-runtime" ? copy.sessionNewAgent : copy.sessionNewChat;
  const sessionEmptyLabel = currentRoute === "agent-runtime" ? copy.sessionEmptyAgent : copy.sessionEmpty;
  const sessionListAriaLabel = currentRoute === "agent-runtime"
    ? copy.sessionAgentListAriaLabel
    : copy.sessionListAriaLabel;
  const sessionHistoryToggleLabel = getLegacySessionHistoryToggleLabel(language, sessionHistoryCollapsed);
  const sessionPaneClassName = sessionHistoryCollapsed ? "session-pane history-collapsed" : "session-pane";

  return (
    <aside className={sessionPaneClassName}>
      <header className="session-header">
        <h1 data-i18n="session.header">{copy.sessionHeader}</h1>
        <button className="pane-action" id="togglePaneButton" type="button" aria-label={copy.sessionClose} data-i18n="session.close">
          {copy.sessionClose}
        </button>
      </header>

      <button className="new-chat" id={LEGACY_SHELL_IDS.newChatButton} type="button" data-i18n="session.new">
        {newChatLabel}
      </button>

      <div className="session-list-wrap">
        <div className="session-history-head">
          <p className="session-title" data-i18n="session.recent">{copy.sessionRecent}</p>
          <button
            className="session-history-toggle"
            id="sessionHistoryToggle"
            type="button"
            aria-controls="sessionHistoryPanel"
            aria-expanded={sessionHistoryCollapsed ? "false" : "true"}
            aria-label={sessionHistoryToggleLabel}
            data-collapsed-state={sessionHistoryCollapsed ? "collapsed" : "expanded"}
            onClick={onToggleSessionHistoryCollapsed}
          >
            {sessionHistoryToggleLabel}
          </button>
        </div>
        <SessionHistoryMount
          hidden={sessionHistoryCollapsed}
          emptyLabel={sessionEmptyLabel}
          listAriaLabel={sessionListAriaLabel}
        />
      </div>
    </aside>
  );
});
