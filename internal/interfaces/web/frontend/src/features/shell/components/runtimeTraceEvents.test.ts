import { describe, expect, it } from "vitest";
import {
  DEFAULT_RUNTIME_EVENT_FILTER,
  processStepToRuntimeTraceEvent,
  runtimeTraceEventVisibleByFilter,
  terminalStepToRuntimeTraceEvent,
  type LegacyProcessStep,
  type TerminalStepLike,
} from "./runtimeTraceEvents";

describe("runtime trace events", () => {
  it("migrates legacy Chat commentary process steps into adapter-derived commentary events", () => {
    const step: LegacyProcessStep = {
      id: "msg-commentary",
      kind: "analysis",
      title: "执行过程",
      detail: "我会先确认工作区。",
      status: "completed",
    };

    const event = processStepToRuntimeTraceEvent(step, {
      turnID: "message-1",
      seq: 2,
      provider: {
        engine: "codex",
        adapter: "codex_cli_json",
        event_type: "item.completed",
        item_type: "agent_message",
        channel: "commentary",
      },
    });

    expect(event).toMatchObject({
      id: "msg-commentary",
      turn_id: "message-1",
      seq: 2,
      source: "adapter",
      role: "assistant",
      kind: "assistant_commentary",
      lifecycle: "completed",
      status: "completed",
      visibility: "collapsed",
      title: "执行过程",
    });
    expect(event.blocks).toEqual([{ type: "markdown", text: "我会先确认工作区。" }]);
  });

  it("maps legacy action steps to tool_call without guessing beyond the explicit adapter step", () => {
    const event = processStepToRuntimeTraceEvent({
      id: "step-1",
      kind: "action",
      title: "codex_exec",
      detail: "检查仓库状态",
      status: "completed",
    }, {
      turnID: "message-1",
      seq: 1,
      provider: { engine: "codex", adapter: "codex_cli_json" },
    });

    expect(event.kind).toBe("tool_call");
    expect(event.source).toBe("adapter");
    expect(event.action).toMatchObject({ family: "runtime", name: "codex_exec" });
  });

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
    const commentary = processStepToRuntimeTraceEvent({
      id: "commentary",
      kind: "commentary",
      title: "执行过程",
      detail: "正在处理。",
      status: "completed",
    }, { turnID: "message-1", seq: 1, provider: { engine: "codex", adapter: "codex_cli_json" } });
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
