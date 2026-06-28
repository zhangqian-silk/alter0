import type { ComposerAttachment } from "../../conversation-runtime/composerImageAttachments";
import {
  normalizeRuntimeTraceEvents,
  type RuntimeTraceEvent,
} from "./runtimeTraceEvents";

export type RuntimeSessionAttachment = {
  id?: string;
  name: string;
  content_type: string;
  data_url?: string;
  asset_url?: string;
  preview_url?: string;
};

export type RuntimeSessionTurn = {
  id: string;
  prompt: string;
  attachments?: RuntimeSessionAttachment[];
  status: string;
  started_at?: string | number;
  finished_at?: string | number;
  duration_ms?: number;
  final_output?: string;
  runtime_trace_events?: RuntimeTraceEvent[];
};

export type RuntimeSessionTurnPaging = {
  limit?: number;
  total?: number;
  byte_limit?: number;
  approx_bytes?: number;
  has_more_before?: boolean;
  has_more_after?: boolean;
  oldest_turn_id?: string;
  newest_turn_id?: string;
  next_before_turn_id?: string;
};

export type RuntimeSessionTimelineMessage = {
  id: string;
  role: "user" | "assistant";
  text: string;
  attachments: ComposerAttachment[];
  promptText?: string;
  promptAttachments?: ComposerAttachment[];
  assistantTextDerivedFromPrompt?: boolean;
  route: string;
  source: string;
  error: boolean;
  status: string;
  at: number;
  processEvents: RuntimeTraceEvent[];
  processCollapsed?: boolean;
};

export type RuntimeSessionViewSession = {
  id: string;
  title?: string;
  status?: string;
  pinned?: boolean;
  created_at?: string | number;
  updated_at?: string | number;
  last_output_at?: string | number;
  model_provider_id?: string;
  model_id?: string;
  tool_ids?: string[];
  skill_ids?: string[];
  mcp_ids?: string[];
  turns?: RuntimeSessionTurn[];
  turns_paging?: RuntimeSessionTurnPaging;
};

export function normalizeRuntimeSessionText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

export function parseRuntimeSessionTimestamp(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Date.parse(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

export function compareRuntimeSessionTurns(left: RuntimeSessionTurn, right: RuntimeSessionTurn): number {
  const leftAt = Math.max(parseRuntimeSessionTimestamp(left.started_at), parseRuntimeSessionTimestamp(left.finished_at));
  const rightAt = Math.max(parseRuntimeSessionTimestamp(right.started_at), parseRuntimeSessionTimestamp(right.finished_at));
  if (leftAt > 0 && rightAt > 0 && leftAt !== rightAt) {
    return leftAt - rightAt;
  }
  return normalizeRuntimeSessionText(left.id).localeCompare(normalizeRuntimeSessionText(right.id));
}

export function cloneRuntimeSessionTurn(turn: RuntimeSessionTurn): RuntimeSessionTurn {
  return {
    ...turn,
    attachments: Array.isArray(turn.attachments)
      ? turn.attachments.map((attachment) => ({ ...attachment }))
      : undefined,
    runtime_trace_events: Array.isArray(turn.runtime_trace_events)
      ? turn.runtime_trace_events.map((event) => ({
          ...event,
          provider: { ...event.provider },
          blocks: event.blocks.map((block) => ({ ...block })),
          action: event.action
            ? {
                ...event.action,
                permission: event.action.permission ? { ...event.action.permission } : undefined,
              }
            : undefined,
          error: event.error ? { ...event.error } : undefined,
          raw: event.raw ? { ...event.raw } : undefined,
        }))
      : undefined,
  };
}

export function cloneRuntimeSessionViewSession<TSession extends RuntimeSessionViewSession>(session: TSession): TSession {
  return {
    ...session,
    turns: Array.isArray(session.turns) ? session.turns.map(cloneRuntimeSessionTurn) : undefined,
    turns_paging: session.turns_paging ? { ...session.turns_paging } : undefined,
  };
}

export function mergeRuntimeSessionTurns(
  current: RuntimeSessionTurn[] | undefined,
  incoming: RuntimeSessionTurn[] | undefined,
): RuntimeSessionTurn[] | undefined {
  if (!Array.isArray(incoming)) {
    return current;
  }
  if (!Array.isArray(current) || current.length === 0) {
    return incoming.map(cloneRuntimeSessionTurn).sort(compareRuntimeSessionTurns);
  }
  const merged = new Map<string, RuntimeSessionTurn>();
  current.forEach((turn) => {
    const id = normalizeRuntimeSessionText(turn.id);
    if (id) {
      merged.set(id, cloneRuntimeSessionTurn(turn));
    }
  });
  incoming.forEach((turn) => {
    const id = normalizeRuntimeSessionText(turn.id);
    if (id) {
      merged.set(id, cloneRuntimeSessionTurn(turn));
    }
  });
  return Array.from(merged.values()).sort(compareRuntimeSessionTurns);
}

export function oldestRuntimeSessionTurnID(turns: RuntimeSessionTurn[] | undefined): string {
  if (!Array.isArray(turns) || turns.length === 0) {
    return "";
  }
  return normalizeRuntimeSessionText([...turns].sort(compareRuntimeSessionTurns)[0]?.id);
}

export function newestRuntimeSessionTurnID(turns: RuntimeSessionTurn[] | undefined): string {
  if (!Array.isArray(turns) || turns.length === 0) {
    return "";
  }
  const sorted = [...turns].sort(compareRuntimeSessionTurns);
  return normalizeRuntimeSessionText(sorted[sorted.length - 1]?.id);
}

export function hasRuntimeSessionTurn(turns: RuntimeSessionTurn[] | undefined, turnID: string): boolean {
  const normalized = normalizeRuntimeSessionText(turnID);
  return Boolean(normalized && Array.isArray(turns) && turns.some((turn) => normalizeRuntimeSessionText(turn.id) === normalized));
}

export function mergeRuntimeSessionTurnPaging(
  current: RuntimeSessionTurnPaging | undefined,
  incoming: RuntimeSessionTurnPaging | undefined,
  turns: RuntimeSessionTurn[] | undefined,
): RuntimeSessionTurnPaging | undefined {
  if (!current && !incoming) {
    return undefined;
  }
  const next: RuntimeSessionTurnPaging = {
    ...(current || {}),
    ...(incoming || {}),
  };
  const oldestTurnID = oldestRuntimeSessionTurnID(turns);
  const newestTurnID = newestRuntimeSessionTurnID(turns);
  if (oldestTurnID) {
    next.oldest_turn_id = oldestTurnID;
  }
  if (newestTurnID) {
    next.newest_turn_id = newestTurnID;
  }
  if (incoming?.has_more_before === false) {
    next.has_more_before = false;
    delete next.next_before_turn_id;
    return next;
  }
  if (current?.has_more_before === false && incoming?.has_more_before === true) {
    const incomingBeforeTurnID = normalizeRuntimeSessionText(incoming.next_before_turn_id || incoming.oldest_turn_id);
    if (!incomingBeforeTurnID || hasRuntimeSessionTurn(turns, incomingBeforeTurnID)) {
      next.has_more_before = false;
      delete next.next_before_turn_id;
      return next;
    }
  }
  if (next.has_more_before === true) {
    const beforeTurnID = normalizeRuntimeSessionText(next.next_before_turn_id || next.oldest_turn_id || oldestTurnID);
    if (beforeTurnID) {
      next.next_before_turn_id = beforeTurnID;
    }
  } else {
    delete next.next_before_turn_id;
  }
  return next;
}

export function mergeRuntimeSessionViewSession<TSession extends RuntimeSessionViewSession>(
  current: TSession | undefined,
  incoming: TSession,
): TSession {
  if (!current) {
    const turns = Array.isArray(incoming.turns) ? mergeRuntimeSessionTurns(undefined, incoming.turns) : incoming.turns;
    return {
      ...incoming,
      turns,
      turns_paging: mergeRuntimeSessionTurnPaging(undefined, incoming.turns_paging, turns),
    };
  }
  const merged = { ...current } as Record<string, unknown>;
  (Object.keys(incoming) as Array<keyof TSession>).forEach((key) => {
    const value = incoming[key];
    if (typeof value !== "undefined") {
      merged[key as string] = value;
    }
  });
  if (Array.isArray(incoming.turns)) {
    merged.turns = mergeRuntimeSessionTurns(current.turns, incoming.turns);
  }
  merged.turns_paging = mergeRuntimeSessionTurnPaging(
    current.turns_paging,
    incoming.turns_paging,
    merged.turns as RuntimeSessionTurn[] | undefined,
  );
  return merged as TSession;
}

export function runtimeSessionTurnEvents(
  sessionID: string | undefined,
  turn: RuntimeSessionTurn | undefined,
): RuntimeTraceEvent[] {
  if (!turn) {
    return [];
  }
  return normalizeRuntimeTraceEvents(turn.runtime_trace_events, {
    sessionID: sessionID || "",
    turnID: turn.id,
  });
}

export function runtimeSessionTurnsToTimelineMessages({
  sessionID,
  turns,
  expandedTurns,
  route,
  source,
}: {
  sessionID?: string;
  turns: RuntimeSessionTurn[];
  expandedTurns?: Record<string, boolean>;
  route: string;
  source: string;
}): RuntimeSessionTimelineMessage[] {
  return turns.map((turn) => {
    const runtimeEvents = runtimeSessionTurnEvents(sessionID, turn);
    const attachments = Array.isArray(turn.attachments) ? turn.attachments : [];
    const status = normalizeRuntimeSessionText(turn.status || "");
    const promptText = normalizeRuntimeSessionText(turn.prompt) === "-" ? "" : turn.prompt;
    const finalOutput = normalizeRuntimeSessionText(turn.final_output) === "-" ? "" : turn.final_output || "";
    const hasAssistantPayload =
      Boolean(normalizeRuntimeSessionText(finalOutput))
      || runtimeEvents.length > 0;
    return {
      id: `${turn.id}:assistant`,
      role: "assistant",
      text: hasAssistantPayload ? finalOutput : promptText,
      attachments: [],
      promptText,
      assistantTextDerivedFromPrompt: !hasAssistantPayload && Boolean(normalizeRuntimeSessionText(promptText)),
      promptAttachments: attachments.map((attachment) => ({
        id: `${turn.id}:${attachment.id || attachment.name}`,
        kind: attachment.content_type.startsWith("image/") ? "image" : "file",
        name: attachment.name,
        contentType: attachment.content_type,
        size: 0,
        dataURL: attachment.data_url,
        assetURL: attachment.asset_url,
        previewURL: attachment.preview_url,
      })),
      route,
      source,
      error: status === "failed" || status === "canceled" || status === "cancelled",
      status,
      at: parseRuntimeSessionTimestamp(turn.finished_at || turn.started_at),
      processEvents: runtimeEvents,
      processCollapsed: !(expandedTurns?.[turn.id] ?? false),
    };
  });
}

export function runtimeTimelineMessageTurnID(messageID: string): string {
  return normalizeRuntimeSessionText(messageID).replace(/:(user|assistant|prompt|response)$/, "");
}
