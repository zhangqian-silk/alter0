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
