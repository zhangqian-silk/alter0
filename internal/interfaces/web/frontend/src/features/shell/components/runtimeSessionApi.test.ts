import { describe, expect, it } from "vitest";
import {
  RUNTIME_SESSION_HISTORY_PAGE_TURN_LIMIT,
  runtimeSessionAttachmentsEndpoint,
  runtimeSessionCollectionEndpoint,
  runtimeSessionDetailEndpoint,
  runtimeSessionEventDetailEndpoint,
  runtimeSessionInputEndpoint,
  runtimeSessionPinEndpoint,
  runtimeSessionRecoverEndpoint,
  type RuntimeSessionRoute,
} from "./runtimeSessionApi";

describe("runtime session api", () => {
  const cases: Array<{ route: RuntimeSessionRoute; root: string }> = [
    { route: "chat", root: "/api/chat/sessions" },
    { route: "terminal", root: "/api/terminal/sessions" },
  ];

  it.each(cases)("builds $route endpoints from the same helpers", ({ route, root }) => {
    expect(runtimeSessionCollectionEndpoint(route)).toBe(root);
    expect(runtimeSessionRecoverEndpoint(route)).toBe(`${root}/recover`);
    expect(runtimeSessionDetailEndpoint(route, "session 1")).toBe(`${root}/session%201`);
    expect(runtimeSessionInputEndpoint(route, "session 1")).toBe(`${root}/session%201/input`);
    expect(runtimeSessionPinEndpoint(route, "session 1")).toBe(`${root}/session%201/pin`);
    expect(runtimeSessionAttachmentsEndpoint(route, "session 1")).toBe(`${root}/session%201/attachments`);
    expect(runtimeSessionEventDetailEndpoint(route, "session 1", "turn/1", "event 1")).toBe(
      `${root}/session%201/turns/turn%2F1/events/event%201`,
    );
  });

  it.each(cases)("builds $route history page requests with the shared turn limit", ({ route, root }) => {
    expect(runtimeSessionDetailEndpoint(route, "session-1", {
      turnBefore: "turn-3",
      turnLimit: RUNTIME_SESSION_HISTORY_PAGE_TURN_LIMIT,
    })).toBe(`${root}/session-1?turn_before=turn-3&turn_limit=20`);
  });
});
