import { describe, expect, it } from "vitest";
import {
  DEFAULT_RUNTIME_EVENT_FILTER,
  normalizeRuntimeTraceEvents,
  runtimeTraceEventDetailID,
  runtimeTraceEventDisclosureCategory,
  runtimeTraceEventDisclosureLabel,
  runtimeTraceEventVisibleByFilter,
  type RuntimeTraceEvent,
} from "./runtimeTraceEvents";

function event(overrides: Partial<RuntimeTraceEvent>): RuntimeTraceEvent {
  return {
    id: "event-1",
    turn_id: "turn-1",
    seq: 1,
    source: "adapter",
    provider: { engine: "codex", adapter: "codex_cli_json" },
    role: "assistant",
    kind: "assistant_commentary",
    lifecycle: "completed",
    status: "completed",
    blocks: [],
    visibility: "collapsed",
    ...overrides,
  };
}

describe("runtime trace events", () => {
  it("normalizes lightweight server runtime events and uses event ids as detail ids", () => {
    const events = normalizeRuntimeTraceEvents([{
      id: 41,
      kind: "plan",
      lifecycle: "completed",
      status: "completed",
      text: "Review the change surface.",
      detail_available: true,
      created_at: "2026-04-23T03:31:00Z",
    }], { sessionID: "session-1", turnID: "turn-1" });

    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({
      id: "41",
      session_id: "session-1",
      turn_id: "turn-1",
      seq: 41,
      kind: "plan",
      title: "Review the change surface.",
      summary: "Review the change surface.",
      blocks: [],
      raw: { has_detail: true },
    });
    expect(runtimeTraceEventDetailID(events[0])).toBe("41");
  });

  it("uses event ids as detail ids when raw references are absent", () => {
    expect(runtimeTraceEventDetailID(event({ id: "event-direct", raw: { ref: "legacy-ref" } }))).toBe("event-direct");
    expect(runtimeTraceEventDetailID(event({ id: "", seq: 3, raw: { ref: "legacy-ref" } }))).toBe("legacy-ref");
    expect(runtimeTraceEventDetailID(event({ id: "", seq: 3 }))).toBe("turn-1:event:3");
  });

  it("defaults disclosure to important text and reasoning", () => {
    const commentary = event({ id: "commentary", kind: "assistant_commentary" });
    const reasoning = event({ id: "reasoning", kind: "reasoning" });
    const command = event({ id: "command", kind: "shell_command" });

    expect(runtimeTraceEventVisibleByFilter(commentary, DEFAULT_RUNTIME_EVENT_FILTER)).toBe(true);
    expect(runtimeTraceEventVisibleByFilter(reasoning, DEFAULT_RUNTIME_EVENT_FILTER)).toBe(true);
    expect(runtimeTraceEventVisibleByFilter(command, DEFAULT_RUNTIME_EVENT_FILTER)).toBe(false);
  });

  it("maps process events to their disclosure categories and labels", () => {
    const command = event({ id: "command", kind: "shell_command" });
    const diff = event({ id: "diff", kind: "file_edit" });

    expect(runtimeTraceEventDisclosureCategory(command)).toBe("commands");
    expect(runtimeTraceEventDisclosureLabel(command, "en")).toBe("Commands");
    expect(runtimeTraceEventDisclosureCategory(diff)).toBe("tools");
    expect(runtimeTraceEventDisclosureLabel(diff, "zh")).toBe("工具");
  });
});
