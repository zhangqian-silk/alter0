import { describe, expect, it } from "vitest";
import { runtimeSessionTurnsToTimelineMessages } from "./runtimeSessionViewModel";

describe("runtimeSessionTurnsToTimelineMessages", () => {
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
});
