function renderTerminalViewShell(sessions, activeSessionID, session, sending, closing = false, deleting = false, options = {}) {
  const sessionSheetOpen = Boolean(options?.sessionSheetOpen);
  return `<section class="terminal-view" data-terminal-view>
    <aside class="terminal-session-pane ${sessionSheetOpen ? "is-open" : ""}" data-terminal-session-pane>
      <button class="terminal-session-pane-backdrop" type="button" data-terminal-session-pane-close aria-label="${escapeHTML(t("route.terminal.hide_sessions"))}"></button>
      <div class="route-surface terminal-session-pane-shell">
        <div class="terminal-session-pane-head">
          <div class="terminal-session-pane-copy">
            <strong>${escapeHTML(t("route.terminal.sessions"))}</strong>
            <span>${escapeHTML(t("route.terminal.session_count", { count: String(sessions.length) }))}</span>
          </div>
          <div class="terminal-session-pane-actions">
            <button class="terminal-session-pane-action is-primary" type="button" data-terminal-create>${escapeHTML(t("route.terminal.new_short"))}</button>
            <button class="terminal-session-pane-action terminal-session-pane-close" type="button" data-terminal-session-pane-close>${escapeHTML(t("route.terminal.hide_sessions"))}</button>
          </div>
        </div>
        <div class="terminal-session-list" data-terminal-session-list>${renderTerminalSessionCards(sessions, activeSessionID, {
    deleting,
    deletingSessionID: options?.deletingSessionID
  })}</div>
      </div>
    </aside>
    <section class="terminal-workspace">
      ${renderTerminalWorkspace(session, sending, closing, deleting, {
        sessionCount: sessions.length,
        sessionSheetOpen
      })}
    </section>
  </section>`;
}
