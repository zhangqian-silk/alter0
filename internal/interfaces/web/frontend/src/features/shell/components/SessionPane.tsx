import { memo, useEffect, useState } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import {
  getLegacyRouteHeadingCopy,
  getLegacySessionTrackedCountLabel,
  getLegacySessionHistoryToggleLabel,
  getLegacyShellCopy,
  type LegacyShellLanguage,
} from "../legacyShellCopy";
import { getNavGroupForRoute } from "../legacyShellConfig";
import {
  requestLegacyShellSessionFocus,
  requestLegacyShellSessionRemoval,
  LEGACY_SHELL_SYNC_SESSION_PANE_EVENT,
  type LegacyShellSessionPaneDetail,
} from "../legacyShellBridge";

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
          <button
            type="button"
            className={item.active ? "session-card active" : "session-card"}
            data-session-id={item.id}
            role="option"
            aria-selected={item.active ? "true" : "false"}
            onClick={() => requestLegacyShellSessionFocus(item.id)}
          >
            <p className="session-card-title">{item.title}</p>
            <p className="session-card-meta">{item.meta}</p>
            {item.shortHash ? (
              <div className="session-card-footer">
                <span className="session-card-id" title={item.id}>#{item.shortHash}</span>
              </div>
            ) : null}
          </button>
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
              aria-label={item.deleteLabel}
              onClick={(event) => {
                event.stopPropagation();
                requestLegacyShellSessionRemoval(item.id);
              }}
            >
              {item.deleteLabel}
            </button>
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
  const [snapshot, setSnapshot] = useState<SessionPaneSnapshot | null>(null);

  useEffect(() => {
    const handleSnapshot = (event: Event) => {
      const detail = (event as CustomEvent<LegacyShellSessionPaneDetail>).detail;
      if (!detail || typeof detail.route !== "string") {
        return;
      }

      setSnapshot({
        route: detail.route,
        hasSessions: Boolean(detail.hasSessions),
        loadError: detail.loadError,
        items: Array.isArray(detail.items) ? detail.items : [],
      });
    };

    document.addEventListener(LEGACY_SHELL_SYNC_SESSION_PANE_EVENT, handleSnapshot as EventListener);
    return () => {
      document.removeEventListener(LEGACY_SHELL_SYNC_SESSION_PANE_EVENT, handleSnapshot as EventListener);
    };
  }, []);

  if (snapshot?.route === currentRoute) {
    return snapshot;
  }

  return {
    route: currentRoute,
    hasSessions: false,
    loadError: "",
    items: [],
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
  const routeHeadingCopy = getLegacyRouteHeadingCopy(language, currentRoute);
  const routeGroup = getNavGroupForRoute(currentRoute);
  const routeGroupLabel = routeGroup ? copy.headings[routeGroup.heading] ?? routeGroup.heading : copy.workspaceModePage;
  const newChatLabel = currentRoute === "agent-runtime" ? copy.sessionNewAgent : copy.sessionNewChat;
  const sessionEmptyLabel = currentRoute === "agent-runtime" ? copy.sessionEmptyAgent : copy.sessionEmpty;
  const sessionListAriaLabel = currentRoute === "agent-runtime"
    ? copy.sessionAgentListAriaLabel
    : copy.sessionListAriaLabel;
  const sessionHistoryToggleLabel = getLegacySessionHistoryToggleLabel(language, sessionHistoryCollapsed);
  const sessionPaneClassName = sessionHistoryCollapsed ? "session-pane history-collapsed" : "session-pane";

  return (
    <aside className={sessionPaneClassName} aria-label={copy.sessionPaneLabel}>
      <section className="session-context-card" data-shell-section="session-context">
        <p className="session-context-eyebrow">{copy.sessionPanelEyebrow}</p>
        <div className="session-context-copy">
          <strong>{routeHeadingCopy.title}</strong>
          <p>{routeHeadingCopy.subtitle}</p>
        </div>
        <div className="session-context-metrics">
          <span className="context-pill">{routeGroupLabel}</span>
          <span className="context-pill">{copy.sessionPanelBridgeValue}</span>
          <span className="context-pill is-strong">
            {getLegacySessionTrackedCountLabel(language, sessionPaneSnapshot.items.length)}
          </span>
        </div>
      </section>

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
