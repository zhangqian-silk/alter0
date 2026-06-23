import { describe, expect, it } from "vitest";
import {
  DEFAULT_RUNTIME_EVENT_FILTER,
  runtimeTraceEventVisibleByFilter,
  terminalStepToRuntimeTraceEvent,
  type TerminalStepLike,
} from "./runtimeTraceEvents";

describe("runtime trace events", () => {
  it("maps Terminal command steps to shell_command with terminal blocks", () => {
    const step: TerminalStepLike = {
      id: "step-command",
      type: "command",
      title: "Run command",
      status: "completed",
      preview: "git status",
      blocks: [{
        type: "terminal",
        title: "Shell",
        content: "git status\n\nnothing to commit",
        language: "shell",
        status: "completed",
        exit_code: 0,
      }],
    };

    const event = terminalStepToRuntimeTraceEvent(step, {
      turnID: "turn-1",
      seq: 1,
      provider: { engine: "codex", adapter: "codex_cli_json", item_type: "command_execution" },
    });

    expect(event.kind).toBe("shell_command");
    expect(event.source).toBe("adapter");
    expect(event.blocks).toEqual([{
      type: "terminal",
      command: "git status",
      output: "nothing to commit",
      language: "shell",
      exit_code: 0,
    }]);
  });

  it("defaults disclosure to every process type except commands", () => {
    const commentary = terminalStepToRuntimeTraceEvent({
      id: "commentary",
      type: "message",
      title: "执行过程",
      preview: "正在处理。",
      status: "completed",
    }, { turnID: "turn-1", seq: 1, provider: { engine: "codex", adapter: "terminal_turn" } });
    const reasoning = terminalStepToRuntimeTraceEvent({
      id: "reasoning",
      type: "reasoning",
      title: "Reasoning",
      status: "completed",
      preview: "internal reasoning summary",
      blocks: [{ type: "text", content: "internal reasoning summary" }],
    }, { turnID: "turn-1", seq: 2, provider: { engine: "codex", adapter: "codex_cli_json" } });
    const command = terminalStepToRuntimeTraceEvent({
      id: "command",
      type: "command",
      title: "Command",
      status: "completed",
      preview: "git status",
      blocks: [{ type: "terminal", title: "Shell", content: "git status", language: "shell" }],
    }, { turnID: "turn-1", seq: 3, provider: { engine: "codex", adapter: "codex_cli_json" } });

    expect(runtimeTraceEventVisibleByFilter(commentary, DEFAULT_RUNTIME_EVENT_FILTER)).toBe(true);
    expect(runtimeTraceEventVisibleByFilter(reasoning, DEFAULT_RUNTIME_EVENT_FILTER)).toBe(true);
    expect(runtimeTraceEventVisibleByFilter(command, DEFAULT_RUNTIME_EVENT_FILTER)).toBe(false);
  });
});
