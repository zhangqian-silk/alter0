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
  it("normalizes server runtime events and keeps raw references as the detail id", () => {
    const events = normalizeRuntimeTraceEvents([{
      id: "event-1",
      turn_id: "",
      seq: "not-a-number",
      source: "adapter",
      provider: { engine: "codex", adapter: "codex_cli_json" },
      role: "assistant",
      kind: "plan",
      lifecycle: "completed",
      status: "completed",
      blocks: [],
      visibility: "collapsed",
      raw: { ref: "event-detail-1", type: "plan", has_detail: true },
    }], { sessionID: "session-1", turnID: "turn-1" });

    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({
      session_id: "session-1",
      turn_id: "turn-1",
      seq: 1,
    });
    expect(runtimeTraceEventDetailID(events[0])).toBe("event-detail-1");
  });

  it("uses event ids as detail ids when raw references are absent", () => {
    expect(runtimeTraceEventDetailID(event({ id: "event-direct" }))).toBe("event-direct");
    expect(runtimeTraceEventDetailID(event({ id: "", seq: 3 }))).toBe("turn-1:event:3");
  });

  it("defaults disclosure to important text only", () => {
    const commentary = event({ id: "commentary", kind: "assistant_commentary" });
    const reasoning = event({ id: "reasoning", kind: "reasoning" });
    const command = event({ id: "command", kind: "shell_command" });

    expect(runtimeTraceEventVisibleByFilter(commentary, DEFAULT_RUNTIME_EVENT_FILTER)).toBe(true);
    expect(runtimeTraceEventVisibleByFilter(reasoning, DEFAULT_RUNTIME_EVENT_FILTER)).toBe(false);
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
