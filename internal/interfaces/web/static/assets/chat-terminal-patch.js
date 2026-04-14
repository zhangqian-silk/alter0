function syncNodeText(node, value) {
  if (!node) {
    return;
  }
  const nextValue = String(value || "");
  if (node.textContent !== nextValue) {
    node.textContent = nextValue;
  }
}

function syncNodeHTML(node, html) {
  if (!node) {
    return;
  }
  const nextHTML = String(html || "");
  if (node.innerHTML !== nextHTML) {
    node.innerHTML = nextHTML;
  }
}

function syncNodeAttribute(node, name, value) {
  if (!node || !name) {
    return;
  }
  if (value === null || value === undefined) {
    node.removeAttribute(name);
    return;
  }
  const nextValue = String(value);
  if (node.getAttribute(name) !== nextValue) {
    node.setAttribute(name, nextValue);
  }
}

function syncRenderedBlock(parent, selector, html, anchorNode = null, position = "beforeend") {
  if (!parent || !selector) {
    return null;
  }
  const nextHTML = String(html || "").trim();
  const existing = parent.querySelector(selector);
  if (!nextHTML) {
    if (existing) {
      existing.remove();
    }
    return null;
  }
  if (existing) {
    if (existing.outerHTML !== nextHTML) {
      existing.outerHTML = nextHTML;
    }
    return parent.querySelector(selector);
  }
  if (anchorNode && typeof anchorNode.insertAdjacentHTML === "function") {
    anchorNode.insertAdjacentHTML(position, nextHTML);
    return parent.querySelector(selector);
  }
  parent.insertAdjacentHTML("beforeend", nextHTML);
  return parent.querySelector(selector);
}

function patchTerminalSessionPane(container, sessions, activeSessionID, mobileSessionListOpen, options = {}) {
  const paneNode = container.querySelector("[data-terminal-session-pane]");
  if (!paneNode) {
    return null;
  }
  paneNode.classList.toggle("is-open", Boolean(mobileSessionListOpen));
  syncNodeAttribute(paneNode.querySelector(".terminal-session-pane-backdrop"), "aria-label", t("route.terminal.hide_sessions"));
  syncNodeText(paneNode.querySelector(".terminal-session-pane-copy strong"), t("route.terminal.sessions"));
  syncNodeText(paneNode.querySelector(".terminal-session-pane-copy span"), t("route.terminal.session_count", { count: String(sessions.length) }));
  paneNode.querySelectorAll("[data-terminal-create]").forEach((node) => {
    syncNodeText(node, t("route.terminal.new_short"));
  });
  paneNode.querySelectorAll("[data-terminal-session-pane-close]").forEach((node) => {
    syncNodeText(node, t("route.terminal.hide_sessions"));
  });
  const sessionListNode = paneNode.querySelector("[data-terminal-session-list]");
  syncNodeHTML(sessionListNode, renderTerminalSessionCards(sessions, activeSessionID, options));
  return sessionListNode;
}

function patchTerminalWorkspaceNode(container, session, sending, closing = false, deleting = false, options = {}) {
  const workspaceNode = container.querySelector("[data-terminal-workspace]");
  if (!workspaceNode || !session) {
    return null;
  }
  const currentSessionID = normalizeText(workspaceNode.getAttribute("data-terminal-session-id"));
  const targetSessionID = normalizeText(session.id);
  if (!targetSessionID || currentSessionID !== targetSessionID) {
    return null;
  }

  const logRef = normalizeText(session.terminal_session_id);
  const isLive = isTerminalSessionLiveStatus(session.status);
  const canInput = canTerminalSessionAcceptInput(session.status);
  const normalizedStatus = normalizeTerminalSessionStatus(session.status);
  const placeholder = canInput
    ? t("route.terminal.input")
    : (isTerminalSessionBusyStatus(session.status) ? t("route.terminal.busy") : t("route.terminal.closed"));
  const showRuntimeNote = !sending && !Boolean(session?.pending_create);
  const note = showRuntimeNote
    ? (normalizedStatus === "interrupted"
      ? t("route.terminal.interrupted")
      : ((normalizedStatus === "exited" || normalizedStatus === "failed") ? t("route.terminal.closed") : ""))
    : "";
  const detail = showRuntimeNote
    ? (session.error_message || (parseTerminalExitCode(session.exit_code) !== null ? `exit code ${String(parseTerminalExitCode(session.exit_code))}` : ""))
    : "";
  const closeDisabled = sending || closing || deleting || !isTerminalSessionLiveStatus(session.status);
  const deleteDisabled = sending || closing || deleting;
  const showJumpBottom = Boolean(session?.chat_has_unread_output) || Number(session?.chat_bottom_offset || 0) > TERMINAL_JUMP_BOTTOM_SHOW_THRESHOLD;
  const metaExpanded = Boolean(session?.meta_expanded);
  const sessionCount = Number.isFinite(Number(options?.sessionCount)) ? Math.max(Number(options.sessionCount), 0) : 0;
  const sessionToggleLabel = options?.sessionSheetOpen ? t("route.terminal.hide_sessions") : t("route.terminal.sessions");
  const headerSubcopy = getTerminalSessionLastOutputAt(session) > 0
    ? t("route.terminal.last_output", { time: formatDateTime(new Date(getTerminalSessionLastOutputAt(session)).toISOString()) })
    : t("route.terminal.no_output");

  syncNodeAttribute(workspaceNode, "data-terminal-session-id", session.id);
  syncNodeAttribute(workspaceNode, "data-terminal-workspace-status", normalizeTerminalSessionStatus(session.status) || "unknown");
  syncNodeAttribute(workspaceNode, "data-terminal-workspace-live", isLive ? "true" : "false");

  const sessionPaneToggle = workspaceNode.querySelector("[data-terminal-session-pane-toggle]");
  syncNodeText(sessionPaneToggle, sessionToggleLabel);
  syncNodeAttribute(sessionPaneToggle, "aria-expanded", options?.sessionSheetOpen ? "true" : "false");
  workspaceNode.querySelectorAll(".terminal-mobile-actions [data-terminal-create]").forEach((node) => {
    syncNodeText(node, t("route.terminal.new_short"));
  });

  syncNodeText(workspaceNode.querySelector(".terminal-workspace-eyebrow"), t("route.terminal.logs.heading", { session: shorten(logRef === "-" ? "n/a" : logRef, 24) }));
  syncNodeText(workspaceNode.querySelector(".terminal-workspace-copy h4"), normalizeText(session.title));
  syncNodeText(workspaceNode.querySelector(".terminal-workspace-subcopy"), headerSubcopy);
  const runtimeStateNode = workspaceNode.querySelector("[data-terminal-runtime-state]");
  syncNodeAttribute(runtimeStateNode, "data-terminal-runtime-state", normalizedStatus || "unknown");
  syncNodeText(runtimeStateNode ? runtimeStateNode.querySelector(".terminal-runtime-state-text") : null, renderTerminalStatus(session.status));

  const metaToggleNode = workspaceNode.querySelector("[data-terminal-meta-toggle]");
  syncNodeText(metaToggleNode, metaExpanded ? t("route.terminal.details_hide") : t("route.terminal.details_show"));
  syncNodeAttribute(metaToggleNode, "aria-expanded", metaExpanded ? "true" : "false");

  const closeButtonNode = workspaceNode.querySelector("[data-terminal-close]");
  if (closeButtonNode) {
    closeButtonNode.disabled = closeDisabled;
  }
  syncNodeText(closeButtonNode, closing ? t("route.terminal.closing") : t("route.terminal.close"));
  const deleteButtonNode = workspaceNode.querySelector("[data-terminal-delete]");
  if (deleteButtonNode) {
    deleteButtonNode.disabled = deleteDisabled;
  }
  syncNodeText(deleteButtonNode, deleting ? t("route.terminal.deleting") : t("route.terminal.delete"));

  const workspaceHead = workspaceNode.querySelector(".terminal-workspace-head");
  if (workspaceHead) {
    syncRenderedBlock(
      workspaceHead,
      "[data-terminal-meta-panel]",
      metaExpanded ? renderTerminalWorkspaceMetaPanel(session) : ""
    );
  }

  const consolePanelNode = workspaceNode.querySelector("[data-terminal-console-panel]");
  const chatNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-chat-screen]") : null;
  if (chatNode) {
    syncNodeAttribute(chatNode, "data-terminal-chat-status", normalizeTerminalSessionStatus(session.status) || "unknown");
    syncNodeHTML(chatNode.querySelector(".terminal-log-tree"), renderTerminalTurns(session));
  }
  const jumpTopNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-jump-top]") : null;
  if (jumpTopNode) {
    jumpTopNode.classList.toggle("is-visible", Number(session?.chat_scroll_top || 0) > TERMINAL_JUMP_TOP_SHOW_THRESHOLD);
    syncNodeAttribute(jumpTopNode, "aria-label", t("route.terminal.jump_top"));
    syncNodeAttribute(jumpTopNode, "title", t("route.terminal.jump_top"));
  }
  const jumpPrevNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-jump-prev]") : null;
  if (jumpPrevNode) {
    const previousTurnID = normalizeText(session?.chat_previous_turn_id);
    jumpPrevNode.classList.toggle("is-visible", Boolean(previousTurnID));
    syncNodeAttribute(jumpPrevNode, "data-terminal-jump-target", previousTurnID);
    syncNodeAttribute(jumpPrevNode, "aria-label", t("route.terminal.jump_prev"));
    syncNodeAttribute(jumpPrevNode, "title", t("route.terminal.jump_prev"));
  }
  const jumpNextNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-jump-next]") : null;
  if (jumpNextNode) {
    const nextTurnID = normalizeText(session?.chat_next_turn_id);
    jumpNextNode.classList.toggle("is-visible", Boolean(nextTurnID));
    syncNodeAttribute(jumpNextNode, "data-terminal-jump-target", nextTurnID);
    syncNodeAttribute(jumpNextNode, "aria-label", t("route.terminal.jump_next"));
    syncNodeAttribute(jumpNextNode, "title", t("route.terminal.jump_next"));
  }
  const jumpBottomNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-jump-bottom]") : null;
  if (jumpBottomNode) {
    jumpBottomNode.classList.toggle("is-visible", showJumpBottom);
    jumpBottomNode.classList.toggle("has-unread", Boolean(session?.chat_has_unread_output));
    syncNodeAttribute(jumpBottomNode, "aria-label", t("route.terminal.jump_bottom"));
    syncNodeAttribute(jumpBottomNode, "title", t("route.terminal.jump_bottom"));
  }

  const composerShellNode = workspaceNode.querySelector(".terminal-composer-shell");
  const formNode = composerShellNode ? composerShellNode.querySelector("[data-terminal-input-form]") : null;
  const inputNode = formNode ? formNode.querySelector("[data-terminal-input]") : null;
  if (composerShellNode) {
    syncRenderedBlock(
      formNode,
      "[data-terminal-runtime-note]",
      note || detail
        ? `<div class="terminal-composer-note" data-terminal-runtime-note data-terminal-runtime-status="${escapeHTML(normalizeTerminalSessionStatus(session.status) || "unknown")}">${escapeHTML([note, detail].filter(Boolean).join(" | "))}</div>`
        : "",
      inputNode,
      "beforebegin"
    );
    syncRenderedBlock(
      composerShellNode,
      ".terminal-composer-meta",
      sessionCount > 0 ? `<div class="terminal-composer-meta">${escapeHTML(t("route.terminal.session_count", { count: String(sessionCount) }))}</div>` : "",
      formNode,
      "afterend"
    );
  }
  if (inputNode) {
    inputNode.placeholder = placeholder;
    inputNode.disabled = sending || !canInput;
  }
  const submitNode = formNode ? formNode.querySelector("[data-terminal-submit]") : null;
  if (submitNode) {
    submitNode.disabled = sending || !canInput;
    syncNodeAttribute(submitNode, "aria-label", sending ? t("route.terminal.sending") : t("route.terminal.send"));
    syncNodeText(submitNode.querySelector(".sr-only"), sending ? t("route.terminal.sending") : t("route.terminal.send"));
  }

  return {
    workspaceNode,
    chatNode,
    inputNode
  };
}
