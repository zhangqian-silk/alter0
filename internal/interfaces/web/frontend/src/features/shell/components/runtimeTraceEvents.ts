export type RuntimeEventSource = "provider" | "adapter" | "alter0";

export type RuntimeRole =
  | "user"
  | "assistant"
  | "runtime"
  | "tool"
  | "system"
  | "subagent";

export type RuntimeEventKind =
  | "important_text"
  | "user_message"
  | "assistant_final"
  | "assistant_commentary"
  | "reasoning"
  | "plan"
  | "tools"
  | "tool_call"
  | "tool_result"
  | "commands"
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
  | "system"
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
  | { type: "text" | "markdown"; text: string; title?: string }
  | { type: "thinking"; text: string; title?: string; signature?: string }
  | { type: "tool_input"; json: unknown }
  | { type: "tool_output"; text?: string; json?: unknown; is_error?: boolean }
  | {
      type: "chatRuntime";
      title?: string;
      command?: string;
      output?: string;
      language?: "shell";
      exit_code?: number | null;
    }
  | {
      type: "code" | "diff";
      title?: string;
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
  created_at?: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  error?: { message: string; code?: string };
  raw?: { ref?: string; type?: string; has_detail?: boolean };
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
  "reasoning",
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

type RuntimeTraceEventKindLike = {
  kind?: string;
};

export type RuntimeProcessDetailBlockLike = {
  type?: string;
  title?: string;
  content?: string;
  command?: string;
  output?: string;
  language?: string;
  file?: string;
  start_line?: number;
  status?: string;
  exit_code?: number | null;
};

type RuntimeTraceEventNormalizeContext = {
  sessionID?: string;
  turnID?: string;
};

function normalizeText(value: unknown): string {
  if (typeof value === "string") {
    return value.trim();
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }
  return "";
}

function normalizeRuntimeEventStatus(value: unknown): RuntimeStatus {
  const normalized = normalizeText(value).toLowerCase();
  switch (normalized) {
    case "queued":
    case "running":
    case "completed":
    case "failed":
    case "interrupted":
    case "canceled":
    case "requires_approval":
      return normalized;
    case "success":
    case "ready":
    case "done":
      return "completed";
    case "cancelled":
      return "canceled";
    default:
      return "unknown";
  }
}

function normalizeRuntimeEventLifecycle(status: RuntimeStatus, value: unknown): RuntimeLifecycle {
  const normalized = normalizeText(value).toLowerCase();
  switch (normalized) {
    case "created":
    case "started":
    case "delta":
    case "updated":
    case "completed":
    case "failed":
    case "interrupted":
    case "canceled":
      return normalized;
    case "cancelled":
      return "canceled";
    default:
      switch (status) {
        case "running":
        case "queued":
        case "requires_approval":
          return "started";
        case "failed":
          return "failed";
        case "interrupted":
          return "interrupted";
        case "canceled":
          return "canceled";
        default:
          return "completed";
      }
  }
}

function normalizeRuntimeEventSource(value: unknown): RuntimeEventSource {
  const normalized = normalizeText(value).toLowerCase();
  return normalized === "provider" || normalized === "adapter" || normalized === "alter0"
    ? normalized
    : "alter0";
}

function normalizeRuntimeEventRole(value: unknown): RuntimeRole {
  const normalized = normalizeText(value).toLowerCase();
  switch (normalized) {
    case "user":
    case "assistant":
    case "runtime":
    case "tool":
    case "system":
    case "subagent":
      return normalized;
    default:
      return "assistant";
  }
}

function normalizeRuntimeEventProvider(value: unknown): RuntimeProviderRef {
  if (value && typeof value === "object") {
    const record = value as Partial<RuntimeProviderRef>;
    return {
      engine: record.engine || "alter0",
      adapter: record.adapter || "alter0",
      model: record.model,
      event_type: record.event_type,
      item_type: record.item_type,
      channel: record.channel,
      message_id: record.message_id,
      item_id: record.item_id,
    };
  }
  return { engine: "alter0", adapter: "alter0" };
}

function normalizeRuntimeEventBlocks(value: unknown): RuntimeBlock[] {
  return Array.isArray(value) ? value as RuntimeBlock[] : [];
}

export function normalizeRuntimeTraceEvents(
  values: unknown,
  context: RuntimeTraceEventNormalizeContext = {},
): RuntimeTraceEvent[] {
  if (!Array.isArray(values)) {
    return [];
  }
  return values
    .map((item, index) => {
      if (!item || typeof item !== "object") {
        return null;
      }
      const record = item as RuntimeTraceEvent & {
        text?: unknown;
        detail_available?: unknown;
        created_at?: unknown;
      };
      const id = normalizeText(record.id);
      const kind = normalizeText(record.kind) as RuntimeEventKind;
      if (!id || !kind) {
        return null;
      }
      const seq = Number(record.seq);
      const status = normalizeRuntimeEventStatus(record.status);
      const title = normalizeText(record.title) || normalizeText(record.summary) || normalizeText(record.text);
      const raw = record.raw
        ? { ...record.raw }
        : typeof record.detail_available === "boolean"
          ? { has_detail: record.detail_available }
          : undefined;
      return {
        ...record,
        id,
        session_id: normalizeText(record.session_id) || context.sessionID,
        turn_id: normalizeText(record.turn_id) || normalizeText(context.turnID) || "",
        seq: Number.isFinite(seq) ? seq : Number.isFinite(Number(id)) ? Number(id) : index + 1,
        source: normalizeRuntimeEventSource(record.source),
        provider: normalizeRuntimeEventProvider(record.provider),
        role: normalizeRuntimeEventRole(record.role),
        kind,
        lifecycle: normalizeRuntimeEventLifecycle(status, record.lifecycle),
        status,
        title: title || undefined,
        summary: normalizeText(record.summary) || title || undefined,
        blocks: normalizeRuntimeEventBlocks(record.blocks),
        visibility: record.visibility || "collapsed",
        created_at: normalizeText(record.created_at) || undefined,
        started_at: record.started_at || normalizeText(record.created_at) || undefined,
        raw,
      };
    })
    .filter((item): item is RuntimeTraceEvent => item !== null);
}

export function runtimeTraceEventDetailID(event: RuntimeTraceEvent): string {
  const id = typeof event.id === "string" ? event.id.trim() : "";
  if (id) {
    return id;
  }
  const rawRef = typeof event.raw?.ref === "string" ? event.raw.ref.trim() : "";
  return rawRef || `${event.turn_id}:event:${event.seq}`;
}

export function runtimeTraceEventVisibleByFilter(
  event: RuntimeTraceEventKindLike,
  filter: RuntimeEventFilterID[],
): boolean {
  const selected = new Set(filter.length > 0 ? filter : DEFAULT_RUNTIME_EVENT_FILTER);
  return selected.has(runtimeTraceEventDisclosureCategory(event));
}

export function runtimeTraceEventDisclosureCategory(event: RuntimeTraceEventKindLike): RuntimeEventFilterID {
  switch (normalizeText(event.kind).toLowerCase()) {
    case "assistant_commentary":
    case "analysis":
    case "commentary":
    case "important_text":
      return "important_text";
    case "plan":
      return "plan";
    case "reasoning":
    case "thinking":
      return "reasoning";
    case "tool_call":
    case "tool_result":
    case "tools":
    case "mcp_call":
    case "skill_use":
    case "skill_context":
    case "hook_event":
    case "approval_request":
    case "file_read":
    case "file_write":
    case "file_edit":
    case "web_search":
    case "web_fetch":
    case "subagent_start":
    case "subagent_progress":
    case "subagent_result":
      return "tools";
    case "shell_command":
    case "commands":
      return "commands";
    case "system_event":
    case "system":
    case "rate_limit":
    case "unknown_provider_event":
    case "error":
      return "system";
    default:
      return "system";
  }
}

export function runtimeTraceEventDisclosureLabel(
  event: RuntimeTraceEventKindLike,
  language: "en" | "zh",
): string {
  const category = runtimeTraceEventDisclosureCategory(event);
  return RUNTIME_EVENT_FILTER_OPTIONS.find((option) => option.id === category)?.label[language] || category;
}
