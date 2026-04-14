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
