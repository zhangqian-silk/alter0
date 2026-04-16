import { memo } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import {
  getLegacySessionHistoryToggleLabel,
  getLegacyShellCopy,
  type LegacyShellLanguage,
} from "../legacyShellCopy";
import {
  requestLegacyShellSessionFocus,
  requestLegacyShellSessionRemoval,
  LEGACY_SHELL_SYNC_SESSION_PANE_EVENT,
  type LegacyShellSessionPaneDetail,
} from "../legacyShellBridge";
import { useLegacyShellSnapshot } from "../legacyShellSnapshot";

type SessionPaneProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  sessionHistoryCollapsed: boolean;
  onCreateSession: () => void;
  onClosePane: () => void;
  onToggleSessionHistoryCollapsed: () => void;
};

type SessionPaneSnapshotItem = NonNullable<LegacyShellSessionPaneDetail["items"]>[number];

function CopyIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="12" height="12" rx="2"></rect>
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
    </svg>
  );
}

function DeleteIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 6h18"></path>
      <path d="M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2"></path>
      <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"></path>
      <path d="M10 11v6"></path>
      <path d="M14 11v6"></path>
    </svg>
  );
}

const SessionListMount = memo(function SessionListMount({
  ariaLabel,
  items,
}: {
  ariaLabel: string;
  items: SessionPaneSnapshotItem[];
}) {
  return (
    <div
      id={LEGACY_SHELL_IDS.sessionList}
      className="session-list"
      role="listbox"
      aria-label={ariaLabel}
    >
      {items.map((item) => (
        <div key={item.id} className="session-card-row">
          <div className={item.active ? "session-card active" : "session-card"}>
            <button
              type="button"
              className="session-card-main"
              data-session-id={item.id}
              role="option"
              aria-selected={item.active ? "true" : "false"}
              onClick={() => requestLegacyShellSessionFocus(item.id)}
            >
              <p className="session-card-title" title={item.title}>{item.title}</p>
              <p className="session-card-meta">{item.meta}</p>
            </button>
            <div className="session-card-footer">
              {item.shortHash ? (
                <span className="session-card-id" title={item.id}>#{item.shortHash}</span>
              ) : (
                <span className="session-card-id is-empty" aria-hidden="true"></span>
              )}
              <div className="session-card-actions">
                {item.copyValue ? (
                  <button
                    type="button"
                    className="session-card-copy"
                    data-copy-value={item.copyValue}
                    title={item.copyLabel}
                    aria-label={item.copyLabel}
                  >
                    <CopyIcon />
                  </button>
                ) : null}
                <button
                  type="button"
                  className="session-card-delete"
                  title={item.deleteLabel}
                  aria-label={item.deleteLabel}
                  onClick={(event) => {
                    event.stopPropagation();
                    requestLegacyShellSessionRemoval(item.id);
                  }}
                >
                  <DeleteIcon />
                </button>
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
});

type SessionHistoryMountProps = {
  hidden: boolean;
  emptyLabel: string;
  hasSessions: boolean;
  loadError: string;
  items: SessionPaneSnapshotItem[];
  listAriaLabel: string;
};

const SessionHistoryMount = memo(function SessionHistoryMount({
  hidden,
  emptyLabel,
  hasSessions,
  loadError,
  items,
  listAriaLabel,
}: SessionHistoryMountProps) {
  return (
    <div className="session-history-panel" id="sessionHistoryPanel" hidden={hidden}>
      <SessionListMount ariaLabel={listAriaLabel} items={items} />
      <p className="session-empty" id="sessionEmpty" hidden={hasSessions}>{emptyLabel}</p>
      <p className="session-error" id="sessionLoadError" role="status" aria-live="polite" hidden={!loadError}>
        {loadError}
      </p>
    </div>
  );
});

type SessionPaneSnapshot = {
  route: string;
  hasSessions: boolean;
  loadError: string;
  items: SessionPaneSnapshotItem[];
};

function useLegacySessionPaneSnapshot(currentRoute: string): SessionPaneSnapshot {
  return useLegacyShellSnapshot<LegacyShellSessionPaneDetail, SessionPaneSnapshot>({
    currentRoute,
    eventName: LEGACY_SHELL_SYNC_SESSION_PANE_EVENT,
    fallback: () => ({
      route: currentRoute,
      hasSessions: false,
      loadError: "",
      items: [],
    }),
    normalizeDetail: normalizeSessionPaneSnapshot,
  });
}

function normalizeSessionPaneSnapshot(
  detail: LegacyShellSessionPaneDetail,
): SessionPaneSnapshot | null {
  if (!detail || typeof detail.route !== "string") {
    return null;
  }

  return {
    route: detail.route,
    hasSessions: Boolean(detail.hasSessions),
    loadError: detail.loadError,
    items: Array.isArray(detail.items) ? detail.items : [],
  };
}

export const SessionPane = memo(function SessionPane({
  currentRoute,
  language,
  sessionHistoryCollapsed,
  onCreateSession,
  onClosePane,
  onToggleSessionHistoryCollapsed,
}: SessionPaneProps) {
  const copy = getLegacyShellCopy(language);
  const sessionPaneSnapshot = useLegacySessionPaneSnapshot(currentRoute);
  const newChatLabel = currentRoute === "agent-runtime" ? copy.sessionNewAgent : copy.sessionNewChat;
  const sessionEmptyLabel = currentRoute === "agent-runtime" ? copy.sessionEmptyAgent : copy.sessionEmpty;
  const sessionListAriaLabel = currentRoute === "agent-runtime"
    ? copy.sessionAgentListAriaLabel
    : copy.sessionListAriaLabel;
  const sessionHistoryToggleLabel = getLegacySessionHistoryToggleLabel(language, sessionHistoryCollapsed);
  const sessionPaneClassName = sessionHistoryCollapsed ? "session-pane history-collapsed" : "session-pane";

  return (
    <aside className={sessionPaneClassName} aria-label={copy.sessionPaneLabel}>
      <header className="session-header">
        <h1 data-i18n="session.header">{copy.sessionHeader}</h1>
        <button
          className="pane-action"
          id="togglePaneButton"
          type="button"
          aria-label={copy.sessionClose}
          data-i18n="session.close"
          onClick={onClosePane}
        >
          {copy.sessionClose}
        </button>
      </header>

      <button
        className="new-chat"
        id={LEGACY_SHELL_IDS.newChatButton}
        type="button"
        data-i18n="session.new"
        onClick={onCreateSession}
      >
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
          hasSessions={sessionPaneSnapshot.hasSessions}
          loadError={sessionPaneSnapshot.loadError}
          items={sessionPaneSnapshot.items}
          listAriaLabel={sessionListAriaLabel}
        />
      </div>
    </aside>
  );
});
