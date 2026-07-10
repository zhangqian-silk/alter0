import { describe, expect, it } from "vitest";
import { runtimeSessionTurnsToTimelineMessages } from "./runtimeSessionViewModel";

describe("runtimeSessionTurnsToTimelineMessages", () => {
  it("uses client_request_id to reconcile an optimistic user message", () => {
    const messages = runtimeSessionTurnsToTimelineMessages({
      turns: [{
        id: "turn-7",
        client_request_id: "request-123",
        prompt: "hello",
        status: "running",
      }],
      route: "chat",
      source: "runtime",
    });

    expect(messages[0]?.id).toBe("turn-7:user");
    expect(messages[0]?.clientRequestID).toBe("request-123");
  });

  it("splits a runtime turn into separate user and assistant timeline messages", () => {
    const messages = runtimeSessionTurnsToTimelineMessages({
      sessionID: "session-1",
      route: "chatRuntime",
      source: "runtime",
      turns: [{
        id: "turn-1",
        prompt: "hello",
        status: "completed",
        started_at: "2026-06-29T08:00:00Z",
        finished_at: "2026-06-29T08:00:01Z",
        final_output: "Hello. I'm ready.",
        runtime_trace_events: [],
      }],
    });

    expect(messages).toHaveLength(2);
    expect(messages[0]).toMatchObject({
      id: "turn-1:user",
      role: "user",
      text: "hello",
      route: "chatRuntime",
      source: "runtime",
    });
    expect(messages[1]).toMatchObject({
      id: "turn-1:assistant",
      role: "assistant",
      text: "Hello. I'm ready.",
      route: "chatRuntime",
      source: "runtime",
    });
    expect(messages[1].promptText).toBe("");
  });

  it("keeps runtime turns whose ids arrive as numbers", () => {
    const messages = runtimeSessionTurnsToTimelineMessages({
      sessionID: "c_51jttwiv4yggqagk",
      route: "chatRuntime",
      source: "runtime",
      turns: [{
        id: 1,
        prompt: "成都旅游攻略",
        status: "running",
        started_at: 1783437617847,
        runtime_trace_events: [{
          id: 1,
          kind: "important_text",
          status: "completed",
          text: "我先确认你的出行环境要求。",
          created_at: 1783437627484,
        }],
      }],
    });

    expect(messages.map((message) => message.id)).toEqual(["1:user", "1:assistant"]);
    expect(messages[0]).toMatchObject({
      role: "user",
      text: "成都旅游攻略",
    });
    expect(messages[1].processEvents[0]).toMatchObject({
      id: "1",
      turn_id: "1",
      text: "我先确认你的出行环境要求。",
    });
  });
});
