function renderTerminalStatus(status) {
  const normalized = normalizeTerminalSessionStatus(status);
  if (!normalized) {
    return "-";
  }
  return formatTaskStatus(normalized);
}

function normalizeTerminalSessionStatus(status) {
  const normalized = String(status || "").trim().toLowerCase();
  if (!normalized || normalized === "running") {
    return "ready";
  }
  if (normalized === "starting") {
    return "busy";
  }
  if (["ready", "busy", "exited", "failed", "interrupted"].includes(normalized)) {
    return normalized;
  }
  return normalized;
}

function isTerminalSessionLiveStatus(status) {
  const normalized = normalizeTerminalSessionStatus(status);
  return ["ready", "busy"].includes(normalized);
}

function canTerminalSessionAcceptInput(status) {
  return normalizeTerminalSessionStatus(status) !== "busy";
}

function isTerminalSessionBusyStatus(status) {
  return normalizeTerminalSessionStatus(status) === "busy";
}

function isTerminalTurnLiveStatus(status) {
  const normalized = String(status || "").trim().toLowerCase();
  return ["running", "starting"].includes(normalized);
}

function hasRecoverableTerminalThread(session) {
  if (!session) {
    return false;
  }
  const sessionID = normalizeText(session.id);
  const threadID = normalizeText(session.terminal_session_id);
  return Boolean(threadID && sessionID && threadID !== sessionID);
}

function getTerminalSessionLastOutputAt(session) {
  const snapshotLastOutputAt = Number(session?.last_output_at || 0);
  let lastOutputAt = Number.isFinite(snapshotLastOutputAt) && snapshotLastOutputAt > 0 ? snapshotLastOutputAt : 0;
  const entries = Array.isArray(session?.entries) ? session.entries : [];
  for (let index = entries.length - 1; index >= 0; index -= 1) {
    const entry = entries[index];
    if (String(entry?.role || "").trim().toLowerCase() !== "output") {
      continue;
    }
    const at = Number(entry?.at || 0);
    if (Number.isFinite(at) && at > 0) {
      lastOutputAt = Math.max(lastOutputAt, at);
      break;
    }
  }
  return lastOutputAt;
}

function getTerminalSessionSortAt(session) {
  const lastOutputAt = getTerminalSessionLastOutputAt(session);
  if (lastOutputAt > 0) {
    return lastOutputAt;
  }
  const createdAt = Number(session?.created_at || 0);
  if (Number.isFinite(createdAt) && createdAt > 0) {
    return createdAt;
  }
  const updatedAt = Number(session?.updated_at || 0);
  return Number.isFinite(updatedAt) && updatedAt > 0 ? updatedAt : 0;
}

function isPendingTerminalSession(session) {
  return Boolean(session?.pending_create);
}

function compareTerminalSessions(left, right) {
  if (isPendingTerminalSession(left) !== isPendingTerminalSession(right)) {
    return isPendingTerminalSession(left) ? -1 : 1;
  }
  const sortDiff = getTerminalSessionSortAt(right) - getTerminalSessionSortAt(left);
  if (sortDiff !== 0) {
    return sortDiff;
  }
  const createdDiff = Number(right?.created_at || 0) - Number(left?.created_at || 0);
  if (createdDiff !== 0) {
    return createdDiff;
  }
  return String(right?.id || "").localeCompare(String(left?.id || ""));
}

function normalizeTerminalLine(text, max = 240) {
  const safe = String(text || "").trim();
  if (!safe) {
    return "";
  }
  if (safe.length <= max) {
    return safe;
  }
  return `${safe.slice(0, max - 1)}...`;
}

function classifyTerminalLogKind(entry) {
  const role = String(entry?.role || "").trim().toLowerCase();
  const stream = String(entry?.stream || entry?.kind || "").trim().toLowerCase();
  if (role === "input" || stream === "input") {
    return "command";
  }
  if (role === "system" || stream === "system") {
    return "tag";
  }
  if (stream === "stderr") {
    return "action";
  }
  return "output";
}

function renderTerminalSessionCards(sessions, activeSessionID, options = {}) {
  if (!sessions.length) {
    return `<p class="route-empty-panel terminal-session-empty">${escapeHTML(t("route.terminal.empty"))}</p>`;
  }
  return sessions.map((session) => {
    const title = normalizeText(session.title);
    const sessionID = normalizeText(session.id);
    const active = session.id === activeSessionID;
    const statusClassName = taskStatusClassName(session.status);
    const listTimestamp = getTerminalSessionSortAt(session);
    const listTimeLabel = listTimestamp > 0 ? formatDateTime(new Date(listTimestamp).toISOString()) : "-";
    const lastOutputMeta = getTerminalSessionLastOutputAt(session) > 0
      ? t("route.terminal.last_output", { time: listTimeLabel })
      : t("route.terminal.no_output");
    const deleting = Boolean(options?.deleting) && sessionID === normalizeText(options?.deletingSessionID);
    const deleteLabel = deleting ? t("route.terminal.deleting") : t("route.terminal.delete");
    return `<div class="route-card terminal-session-card ${active ? "active" : ""}" data-terminal-session-card="${escapeHTML(sessionID)}" data-terminal-session-status="${escapeHTML(normalizeTerminalSessionStatus(session.status) || "unknown")}">
      <button class="route-card-button terminal-session-select ${active ? "active" : ""}" type="button" data-terminal-session-select="${escapeHTML(sessionID)}" ${active ? 'aria-current="true"' : ""}>
        <span class="terminal-session-head">
          <span class="route-card-title-copy">
            <span class="terminal-session-title">${escapeHTML(title)}</span>
            <span class="terminal-session-meta">${escapeHTML(lastOutputMeta)}</span>
          </span>
          <span class="task-summary-status ${statusClassName}">${escapeHTML(renderTerminalStatus(session.status))}</span>
        </span>
      </button>
      <button class="terminal-session-list-delete" type="button" data-terminal-delete-session="${escapeHTML(sessionID)}" aria-label="${escapeHTML(deleteLabel)}" ${Boolean(options?.deleting) ? "disabled" : ""}>${escapeHTML(deleteLabel)}</button>
    </div>`;
  }).join("");
}

function shouldCollapseTerminalOutput(text) {
  const content = String(text || "");
  if (!content.trim()) {
    return false;
  }
  const lines = content.replace(/\r\n/g, "\n").replace(/\r/g, "\n").split("\n");
  return content.length > 640 || lines.length > 12;
}

function resolveTerminalOutputCollapsed(session, turn) {
  if (!session || !turn) {
    return false;
  }
  const turnID = normalizeText(turn.id);
  if (!turnID || turnID === "-") {
    return false;
  }
  if (session.output_collapsed && Object.prototype.hasOwnProperty.call(session.output_collapsed, turnID)) {
    return Boolean(session.output_collapsed[turnID]);
  }
  return isMobileViewport() && shouldCollapseTerminalOutput(turn.final_output);
}

function renderTerminalWorkspaceMetaPanel(session) {
  return `<section class="terminal-meta-panel" data-terminal-meta-panel>
    ${routeFieldRow("route.terminal.session", session?.terminal_session_id || "-", { copyable: true, mono: true, clampLines: 2 })}
    ${routeFieldRow("route.terminal.shell", session?.shell || "-", { copyable: true, mono: true, clampLines: 2 })}
    ${routeFieldRow("route.terminal.path", session?.working_dir || "-", { copyable: true, multiline: true, mono: true, clampLines: 3 })}
    ${routeFieldRow("route.terminal.status", renderTerminalStatus(session?.status || "-"))}
  </section>`;
}

function renderTerminalLogRows(entries) {
  const safeEntries = Array.isArray(entries) ? entries : [];
  if (!safeEntries.length) {
    return `<div class="terminal-log-empty">${escapeHTML(t("route.terminal.logs.empty"))}</div>`;
  }
  return safeEntries.map((entry) => {
    const kind = classifyTerminalLogKind(entry);
    const rowClass = `terminal-log-row kind-${escapeHTML(kind)}`;
    const text = String(entry?.text || "");
    const prefix = kind === "command" ? ">" : "";
    if (!text) {
      return "";
    }
    return `<div class="${rowClass}">
      <div class="terminal-log-main">
        ${prefix ? `<span class="terminal-log-prefix">${escapeHTML(prefix)}</span>` : ""}
        <span class="terminal-log-text">${escapeHTML(text)}</span>
      </div>
      <span class="terminal-log-time">${escapeHTML(timeLabel(entry?.at))}</span>
    </div>`;
  }).join("");
}

function formatTerminalDuration(durationMS, startedAt = 0, finishedAt = 0) {
  let total = Number(durationMS || 0);
  if ((!Number.isFinite(total) || total <= 0) && Number.isFinite(Number(startedAt)) && Number(startedAt) > 0) {
    const end = Number.isFinite(Number(finishedAt)) && Number(finishedAt) > 0 ? Number(finishedAt) : Date.now();
    total = Math.max(0, end - Number(startedAt));
  }
  if (!Number.isFinite(total) || total <= 0) {
    return "<1s";
  }
  const seconds = Math.max(1, Math.round(total / 1000));
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainSeconds = seconds % 60;
  if (minutes < 60) {
    return remainSeconds > 0 ? `${minutes}m ${remainSeconds}s` : `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const remainMinutes = minutes % 60;
  return remainMinutes > 0 ? `${hours}h ${remainMinutes}m` : `${hours}h`;
}

function escapeHTMLWithHighlight(text, query = "") {
  const source = String(text || "");
  const needle = String(query || "").trim();
  if (!needle) {
    return escapeHTML(source);
  }
  const lowerSource = source.toLowerCase();
  const lowerNeedle = needle.toLowerCase();
  let cursor = 0;
  let html = "";
  while (cursor < source.length) {
    const index = lowerSource.indexOf(lowerNeedle, cursor);
    if (index < 0) {
      html += escapeHTML(source.slice(cursor));
      break;
    }
    html += escapeHTML(source.slice(cursor, index));
    html += `<mark>${escapeHTML(source.slice(index, index + needle.length))}</mark>`;
    cursor = index + needle.length;
  }
  return html;
}

function renderTerminalNumberedBlock(content, searchQuery = "") {
  const text = String(content || "");
  if (!text) {
    return `<div class="terminal-rich-empty">-</div>`;
  }
  const lines = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n").split("\n");
  return `<div class="terminal-numbered-block">${lines.map((line, index) => `<div class="terminal-numbered-line"><span class="terminal-numbered-index">${String(index + 1)}</span><span class="terminal-numbered-text">${escapeHTMLWithHighlight(line, searchQuery)}</span></div>`).join("")}</div>`;
}

function renderTerminalDiffBlock(content, searchQuery = "") {
  const text = String(content || "");
  const lines = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n").split("\n");
  return `<div class="terminal-diff-block">${lines.map((line, index) => {
    const marker = line.startsWith("+") ? "add" : line.startsWith("-") ? "remove" : "context";
    return `<div class="terminal-diff-line kind-${marker}"><span class="terminal-numbered-index">${String(index + 1)}</span><span class="terminal-numbered-text">${escapeHTMLWithHighlight(line, searchQuery)}</span></div>`;
  }).join("")}</div>`;
}

function renderTerminalRichBlock(stepID, block, searchQuery = "") {
  const type = String(block?.type || "text").trim().toLowerCase();
  const title = String(block?.title || "").trim();
  const status = String(block?.status || "").trim().toLowerCase();
  const exitCode = parseTerminalExitCode(block?.exit_code);
  const fileLabel = String(block?.file || "").trim();
  const lineLabel = Number.isFinite(Number(block?.start_line)) && Number(block.start_line) > 0
    ? `:${String(Number(block.start_line))}`
    : "";
  const blockHeader = title || status || exitCode !== null
    ? `<div class="terminal-rich-head">
        <div class="terminal-rich-copy">
          ${title ? `<strong>${escapeHTML(title)}</strong>` : ""}
          ${fileLabel ? `<span>${escapeHTML(`${fileLabel}${lineLabel}`)}</span>` : ""}
        </div>
        <div class="terminal-rich-meta">
          ${status ? `<span class="terminal-step-status status-${escapeHTML(status)}">${escapeHTML(renderTerminalStatus(status))}</span>` : ""}
          ${exitCode !== null ? `<span class="terminal-rich-exit">exit ${escapeHTML(String(exitCode))}</span>` : ""}
        </div>
      </div>`
    : "";
  let body = "";
  if (type === "terminal") {
    body = `<pre class="terminal-rich-pre">${escapeHTML(String(block?.content || ""))}</pre>`;
  } else if (type === "diff") {
    body = renderTerminalDiffBlock(block?.content || "", searchQuery);
  } else {
    body = renderTerminalNumberedBlock(block?.content || "", searchQuery);
  }
  return `<section class="route-surface-dark terminal-rich-block type-${escapeHTML(type)}">${blockHeader}${body}</section>`;
}

function renderTerminalStepDetail(session, turn, step) {
  const stepID = normalizeText(step?.id);
  const detail = session?.step_details?.[stepID] || null;
  const loading = Boolean(session?.step_loading?.[stepID]);
  const error = String(session?.step_errors?.[stepID] || "").trim();
  const searchQuery = String(session?.step_search?.[stepID] || "");
  if (loading) {
    return `<div class="terminal-step-detail-state">${escapeHTML(t("route.terminal.step.loading"))}</div>`;
  }
  if (error) {
    return `<div class="terminal-step-detail-state is-error">${escapeHTML(t("route.terminal.step.error", { error }))}</div>`;
  }
  if (!detail) {
    return "";
  }
  const blocks = Array.isArray(detail.blocks) ? detail.blocks : [];
  return `<div class="terminal-step-detail">
    ${detail.searchable ? `<div class="terminal-step-search"><input type="search" data-terminal-step-search="${escapeHTML(stepID)}" value="${escapeHTML(searchQuery)}" placeholder="${escapeHTML(t("route.terminal.step.search"))}"></div>` : ""}
    ${blocks.length ? blocks.map((block) => renderTerminalRichBlock(stepID, block, searchQuery)).join("") : `<div class="terminal-step-detail-state">${escapeHTML(t("route.terminal.process.empty"))}</div>`}
  </div>`;
}

function renderTerminalStepItems(session, turn) {
  const steps = Array.isArray(turn?.steps) ? turn.steps : [];
  if (!steps.length) {
    const waiting = String(turn?.status || "").trim().toLowerCase() === "running";
    return `<div class="terminal-process-empty">${escapeHTML(waiting ? t("route.terminal.process.loading") : t("route.terminal.process.empty"))}</div>`;
  }
  return steps.map((step) => {
    const stepID = normalizeText(step.id);
    const expanded = Boolean(session?.expanded_steps?.[stepID]);
    const duration = formatTerminalDuration(step.duration_ms, step.started_at, step.finished_at);
    const preview = String(step.preview || "").trim();
    const summary = preview || String(step.title || "-").trim() || "-";
    return `<article class="terminal-step-item" data-terminal-step-item="${escapeHTML(stepID)}">
      <button class="terminal-step-toggle" type="button" data-terminal-step-toggle="${escapeHTML(stepID)}" data-terminal-turn-id="${escapeHTML(normalizeText(turn.id))}" aria-expanded="${expanded ? "true" : "false"}">
        <span class="terminal-step-toggle-icon">${expanded ? "v" : ">"}</span>
        <span class="terminal-step-summary">
          <span class="terminal-step-title">${escapeHTML(summary)}</span>
        </span>
        <span class="terminal-step-meta">
          <span class="terminal-step-duration">${escapeHTML(duration)}</span>
          <span class="terminal-step-status status-${escapeHTML(String(step.status || "completed"))}">${escapeHTML(renderTerminalStatus(step.status))}</span>
        </span>
      </button>
      <div class="terminal-step-body" ${expanded ? "" : "hidden"}>
        ${renderTerminalStepDetail(session, turn, step)}
      </div>
    </article>`;
  }).join("");
}

function renderTerminalTurnProcess(session, turn) {
  const turnID = normalizeText(turn?.id);
  const collapsed = resolveTerminalProcessCollapsed(session, turn);
  const duration = formatTerminalDuration(turn?.duration_ms, turn?.started_at, turn?.finished_at);
  const stepCount = Array.isArray(turn?.steps) ? turn.steps.length : 0;
  return `<section class="terminal-process-shell ${collapsed ? "is-collapsed" : ""}" data-terminal-process-shell="${escapeHTML(turnID)}">
    <button class="terminal-process-toggle" type="button" data-terminal-process-toggle="${escapeHTML(turnID)}" aria-expanded="${collapsed ? "false" : "true"}">
      <span class="terminal-step-toggle-icon">${collapsed ? ">" : "v"}</span>
      <span class="terminal-process-copy">
        <span class="terminal-process-title">${escapeHTML(t("route.terminal.process.label"))}</span>
        <span class="terminal-process-summary">${escapeHTML(t("route.terminal.process.steps", { count: String(stepCount) }))}</span>
      </span>
      <span class="terminal-process-meta">${escapeHTML(duration)}</span>
    </button>
    <div class="terminal-process-body" ${collapsed ? "hidden" : ""}>
      ${renderTerminalStepItems(session, turn)}
    </div>
  </section>`;
}

function renderTerminalFinalOutput(session, turn) {
  const content = typeof turn?.final_output === "string" ? turn.final_output : "";
  if (!content.trim()) {
    return "";
  }
  const turnID = normalizeText(turn?.id);
  return `<div class="msg assistant terminal-final-output terminal-turn-output" data-terminal-final-output="${escapeHTML(turnID)}">
    <div class="msg-bubble">
      <div class="terminal-final-text">
        <div class="assistant-message-toolbar terminal-final-toolbar">
          ${renderAssistantCopyButton(content, "terminal-final-copy")}
        </div>
        <div class="terminal-final-rendered">${renderMarkdownToHTML(content)}</div>
      </div>
    </div>
  </div>`;
}

function renderTerminalTurns(session) {
  const turns = Array.isArray(session?.turns) ? session.turns : [];
  if (!turns.length) {
    return renderTerminalLogRows(session?.entries || []);
  }
  return turns.map((turn) => {
    const promptText = String(turn?.prompt || "").trim();
    const hasProcess = Array.isArray(turn?.steps) && turn.steps.length > 0;
    const bodyHTML = [
      hasProcess || String(turn?.status || "").trim().toLowerCase() === "running" ? renderTerminalTurnProcess(session, turn) : "",
      renderTerminalFinalOutput(session, turn)
    ].filter(Boolean).join("");
    return `<article class="terminal-turn-card" data-terminal-turn="${escapeHTML(normalizeText(turn.id))}">
      ${promptText ? `<div class="terminal-log-row kind-command terminal-turn-prompt"><div class="terminal-log-main"><span class="terminal-log-text">${escapeHTML(promptText)}</span></div><span class="terminal-log-time">${escapeHTML(timeLabel(turn?.started_at))}</span></div>` : ""}
      ${bodyHTML ? `<div class="terminal-turn-surface">${bodyHTML}</div>` : ""}
    </article>`;
  }).join("");
}

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
