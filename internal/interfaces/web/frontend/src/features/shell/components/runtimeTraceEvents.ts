export type RuntimeEventSource = "provider" | "adapter" | "alter0";

export type RuntimeRole =
  | "user"
  | "assistant"
  | "runtime"
  | "tool"
  | "system"
  | "subagent";

export type RuntimeEventKind =
  | "user_message"
  | "assistant_final"
  | "assistant_commentary"
  | "reasoning"
  | "plan"
  | "tool_call"
  | "tool_result"
  | "shell_command"
  | "file_read"
  | "file_write"
  | "file_edit"
  | "web_search"
  | "web_fetch"
  | "mcp_call"
  | "skill_context"
  | "skill_use"
  | "subagent_start"
  | "subagent_progress"
  | "subagent_result"
  | "hook_event"
  | "approval_request"
  | "rate_limit"
  | "system_event"
  | "error"
  | "unknown_provider_event";

export type RuntimeLifecycle =
  | "created"
  | "started"
  | "delta"
  | "updated"
  | "completed"
  | "failed"
  | "interrupted"
  | "canceled";

export type RuntimeStatus =
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "interrupted"
  | "canceled"
  | "requires_approval"
  | "unknown";

export type RuntimeVisibility = "user" | "collapsed" | "developer" | "internal";

export type RuntimeProviderRef = {
  engine: "codex" | "claude_code" | "openai" | "anthropic" | "alter0";
  adapter: "codex_cli_json" | "claude_cli_stream_json" | "claude_agent_sdk" | "alter0";
  model?: string;
  event_type?: string;
  item_type?: string;
  channel?: string;
  message_id?: string;
  item_id?: string;
};

export type RuntimeBlock =
  | { type: "text" | "markdown"; text: string }
  | { type: "thinking"; text: string; signature?: string }
  | { type: "tool_input"; json: unknown }
  | { type: "tool_output"; text?: string; json?: unknown; is_error?: boolean }
  | {
      type: "terminal";
      command?: string;
      output?: string;
      language?: "shell";
      exit_code?: number | null;
    }
  | {
      type: "code" | "diff";
      content: string;
      language?: string;
      file?: string;
      start_line?: number;
    }
  | {
      type: "attachment" | "image";
      name?: string;
      url?: string;
      content_type?: string;
    }
  | { type: "error"; message: string; code?: string };

export type RuntimeAction = {
  name?: string;
  family?:
    | "builtin_tool"
    | "shell"
    | "file"
    | "web"
    | "mcp"
    | "skill"
    | "subagent"
    | "hook"
    | "runtime";
  tool_use_id?: string;
  parent_tool_use_id?: string;
  mcp_server?: string;
  mcp_tool?: string;
  skill_id?: string;
  skill_name?: string;
  input?: unknown;
  output?: unknown;
  permission?: {
    required: boolean;
    decision?: "approved" | "denied" | "auto";
    rule?: string;
  };
};

export type RuntimeTraceEvent = {
  id: string;
  session_id?: string;
  turn_id: string;
  parent_id?: string;
  seq: number;
  source: RuntimeEventSource;
  provider: RuntimeProviderRef;
  role: RuntimeRole;
  kind: RuntimeEventKind;
  lifecycle: RuntimeLifecycle;
  status: RuntimeStatus;
  title?: string;
  summary?: string;
  blocks: RuntimeBlock[];
  action?: RuntimeAction;
  visibility: RuntimeVisibility;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  error?: { message: string; code?: string };
  raw?: { ref?: string; type?: string };
};

export type RuntimeEventFilterID =
  | "important_text"
  | "plan"
  | "reasoning"
  | "tools"
  | "commands"
  | "system";

export const DEFAULT_RUNTIME_EVENT_FILTER: RuntimeEventFilterID[] = [
  "important_text",
  "plan",
  "reasoning",
  "tools",
  "system",
];

export const RUNTIME_EVENT_FILTER_OPTIONS: Array<{
  id: RuntimeEventFilterID;
  label: Record<"en" | "zh", string>;
  description: Record<"en" | "zh", string>;
}> = [
  {
    id: "important_text",
    label: { en: "Important text", zh: "重要文本" },
    description: {
      en: "Provider commentary and concise progress notes.",
      zh: "运行时 commentary 与简明进度说明。",
    },
  },
  {
    id: "plan",
    label: { en: "Plan", zh: "计划" },
    description: { en: "Explicit plan events from the provider.", zh: "底层明确提供的 plan 事件。" },
  },
  {
    id: "reasoning",
    label: { en: "Reasoning", zh: "推理" },
    description: { en: "Explicit reasoning or thinking blocks.", zh: "底层明确提供的 reasoning / thinking 块。" },
  },
  {
    id: "tools",
    label: { en: "Tools", zh: "工具" },
    description: { en: "Tool, MCP, skill, hook and approval events.", zh: "Tool、MCP、Skill、Hook 与权限事件。" },
  },
  {
    id: "commands",
    label: { en: "Commands", zh: "命令" },
    description: { en: "Shell command execution records.", zh: "Shell 命令执行记录。" },
  },
  {
    id: "system",
    label: { en: "System", zh: "系统" },
    description: { en: "Runtime logs, rate limits and unknown provider events.", zh: "运行日志、限流与未知 provider 事件。" },
  },
];

export type LegacyProcessStep = {
  id?: string;
  kind?: string;
  title?: string;
  detail?: string;
  status?: string;
};

export type TerminalStepBlockLike = {
  type?: string;
  title?: string;
  content?: string;
  language?: string;
  file?: string;
  start_line?: number;
  status?: string;
  exit_code?: number | null;
};

export type TerminalStepLike = {
  id?: string;
  type?: string;
  title?: string;
  status?: string;
  preview?: string;
  blocks?: TerminalStepBlockLike[];
};

type RuntimeTraceEventContext = {
  sessionID?: string;
  turnID: string;
  seq: number;
  provider: RuntimeProviderRef;
};

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeLifecycle(status: string): RuntimeLifecycle {
  switch (status.toLowerCase()) {
    case "running":
    case "queued":
      return "started";
    case "failed":
    case "error":
      return "failed";
    case "interrupted":
      return "interrupted";
    case "canceled":
    case "cancelled":
      return "canceled";
    default:
      return "completed";
  }
}

function normalizeRuntimeStatus(status: string): RuntimeStatus {
  switch (status.toLowerCase()) {
    case "queued":
      return "queued";
    case "running":
    case "streaming":
    case "in_progress":
    case "inprogress":
      return "running";
    case "failed":
    case "error":
      return "failed";
    case "interrupted":
      return "interrupted";
    case "canceled":
    case "cancelled":
      return "canceled";
    case "requires_approval":
      return "requires_approval";
    case "completed":
    case "success":
    case "done":
    case "":
      return "completed";
    default:
      return "unknown";
  }
}

function normalizeLegacyKind(kind: string, provider: RuntimeProviderRef): RuntimeEventKind {
  const normalized = normalizeText(kind).toLowerCase();
  if (normalized === "analysis" || normalized === "commentary") {
    return "assistant_commentary";
  }
  if (normalized === "plan") {
    return "plan";
  }
  if (normalized === "reasoning" || normalized === "thinking") {
    return "reasoning";
  }
  if (normalized === "action") {
    return "tool_call";
  }
  if (normalized === "observation") {
    return "tool_result";
  }
  if (normalized === "command" || normalized === "command_execution") {
    return "shell_command";
  }
  if (provider.channel === "commentary") {
    return "assistant_commentary";
  }
  if (normalized === "") {
    return "assistant_commentary";
  }
  return "unknown_provider_event";
}

function eventVisibility(kind: RuntimeEventKind): RuntimeVisibility {
  switch (kind) {
    case "assistant_final":
    case "user_message":
      return "user";
    case "assistant_commentary":
    case "plan":
    case "reasoning":
    case "tool_call":
    case "tool_result":
    case "shell_command":
    case "mcp_call":
    case "skill_use":
    case "skill_context":
      return "collapsed";
    default:
      return "developer";
  }
}

export function processStepToRuntimeTraceEvent(
  step: LegacyProcessStep,
  context: RuntimeTraceEventContext,
): RuntimeTraceEvent {
  const kind = normalizeLegacyKind(step.kind || "", context.provider);
  const status = normalizeRuntimeStatus(step.status || "");
  const title = normalizeText(step.title) || defaultEventTitle(kind);
  const detail = normalizeText(step.detail);
  return {
    id: normalizeText(step.id) || `${context.turnID}:step:${context.seq}`,
    session_id: context.sessionID,
    turn_id: context.turnID,
    seq: context.seq,
    source: "adapter",
    provider: context.provider,
    role: kind === "tool_result" ? "tool" : "assistant",
    kind,
    lifecycle: normalizeLifecycle(step.status || ""),
    status,
    title,
    summary: detail || title,
    blocks: detail ? [{ type: "markdown", text: detail }] : [],
    action: kind === "tool_call" ? {
      family: "runtime",
      name: title,
    } : undefined,
    visibility: eventVisibility(kind),
  };
}

export function terminalStepToRuntimeTraceEvent(
  step: TerminalStepLike,
  context: RuntimeTraceEventContext,
): RuntimeTraceEvent {
  const stepType = normalizeText(step.type).toLowerCase();
  const kind = terminalStepKind(stepType);
  const status = normalizeRuntimeStatus(step.status || "");
  const title = normalizeText(step.title) || normalizeText(step.preview) || defaultEventTitle(kind);
  const blocks = terminalBlocks(step.blocks || [], step.preview || "", kind);
  return {
    id: normalizeText(step.id) || `${context.turnID}:step:${context.seq}`,
    session_id: context.sessionID,
    turn_id: context.turnID,
    seq: context.seq,
    source: "adapter",
    provider: context.provider,
    role: kind === "tool_result" ? "tool" : "assistant",
    kind,
    lifecycle: normalizeLifecycle(step.status || ""),
    status,
    title,
    summary: normalizeText(step.preview) || title,
    blocks,
    action: kind === "shell_command" ? {
      family: "shell",
      name: "shell",
    } : undefined,
    visibility: eventVisibility(kind),
  };
}

function terminalStepKind(stepType: string): RuntimeEventKind {
  switch (stepType) {
    case "reasoning":
      return "reasoning";
    case "plan":
      return "plan";
    case "command":
    case "command_execution":
      return "shell_command";
    case "log":
    case "system":
      return "system_event";
    case "diff":
      return "file_edit";
    case "message":
      return "assistant_commentary";
    default:
      return "unknown_provider_event";
  }
}

function terminalBlocks(
  blocks: TerminalStepBlockLike[],
  preview: string,
  kind: RuntimeEventKind,
): RuntimeBlock[] {
  const mapped = blocks
    .map((block): RuntimeBlock | null => {
      const blockType = normalizeText(block.type).toLowerCase();
      const content = typeof block.content === "string" ? block.content : "";
      if (blockType === "terminal") {
        const [command, ...outputParts] = content.split(/\n\n/);
        return {
          type: "terminal",
          command: normalizeText(command) || undefined,
          output: outputParts.join("\n\n").trim() || undefined,
          language: "shell",
          exit_code: typeof block.exit_code === "number" ? block.exit_code : block.exit_code ?? null,
        };
      }
      if (blockType === "diff") {
        return {
          type: "diff",
          content,
          language: normalizeText(block.language) || undefined,
          file: normalizeText(block.file) || undefined,
          start_line: block.start_line,
        };
      }
      if (blockType === "code") {
        return {
          type: "code",
          content,
          language: normalizeText(block.language) || undefined,
          file: normalizeText(block.file) || undefined,
          start_line: block.start_line,
        };
      }
      if (content) {
        return { type: "markdown", text: content };
      }
      return null;
    })
    .filter((block): block is RuntimeBlock => block !== null);
  if (mapped.length > 0) {
    return mapped;
  }
  const fallback = normalizeText(preview);
  if (!fallback) {
    return [];
  }
  return kind === "shell_command"
    ? [{ type: "terminal", command: fallback, language: "shell" }]
    : [{ type: "markdown", text: fallback }];
}

function defaultEventTitle(kind: RuntimeEventKind): string {
  switch (kind) {
    case "assistant_commentary":
      return "Progress";
    case "reasoning":
      return "Reasoning";
    case "plan":
      return "Plan";
    case "shell_command":
      return "Shell";
    case "tool_call":
      return "Tool";
    case "tool_result":
      return "Result";
    case "system_event":
      return "System";
    default:
      return "Event";
  }
}

export function runtimeTraceEventVisibleByFilter(
  event: RuntimeTraceEvent,
  filter: RuntimeEventFilterID[],
): boolean {
  const selected = new Set(filter.length > 0 ? filter : DEFAULT_RUNTIME_EVENT_FILTER);
  switch (event.kind) {
    case "assistant_commentary":
      return selected.has("important_text");
    case "plan":
      return selected.has("plan");
    case "reasoning":
      return selected.has("reasoning");
    case "tool_call":
    case "tool_result":
    case "mcp_call":
    case "skill_use":
    case "skill_context":
    case "hook_event":
    case "approval_request":
      return selected.has("tools");
    case "shell_command":
      return selected.has("commands");
    case "system_event":
    case "rate_limit":
    case "unknown_provider_event":
    case "error":
      return selected.has("system");
    default:
      return true;
  }
}
