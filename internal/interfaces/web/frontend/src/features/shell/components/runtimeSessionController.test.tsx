import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { useRuntimeSessionController, type RuntimeSessionPayload } from "./runtimeSessionController";

type TestSession = {
  id: string;
  title: string;
  status: string;
};

const apiClientMock = {
  get: vi.fn(async () => ({ items: [] })),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
};

vi.mock("../../../shared/api/client", () => ({
  createAPIClient: () => apiClientMock,
}));

function normalizeSession(payload: RuntimeSessionPayload): TestSession | null {
  if (typeof payload.id !== "string" || payload.id.trim() === "") {
    return null;
  }
  return {
    id: payload.id,
    title: typeof payload.title === "string" ? payload.title : payload.id,
    status: typeof payload.status === "string" ? payload.status : "ready",
  };
}

function mergeSession(previous: TestSession | undefined, incoming: TestSession): TestSession {
  return {
    ...previous,
    ...incoming,
  };
}

function sortSessions(sessions: TestSession[]): TestSession[] {
  return [...sessions].sort((left, right) => left.id.localeCompare(right.id));
}

function RuntimeSessionControllerHarness() {
  const [refreshResultIDs, setRefreshResultIDs] = useState("");
  const controller = useRuntimeSessionController<TestSession>({
    route: "chat",
    initialSessions: [{ id: "chat-1", title: "Cached chat", status: "ready" }],
    initialActiveSessionID: "chat-1",
    normalizeSession,
    mergeSession,
    sortSessions,
    preserveMissingSessionsOnRefresh: true,
  });

  return (
    <div>
      <button
        type="button"
        onClick={() => {
          void controller.refreshList().then((sessions) => {
            setRefreshResultIDs(sessions.map((session) => session.id).join("|"));
          });
        }}
      >
        refresh list
      </button>
      <output data-testid="session-ids">{controller.sessions.map((session) => session.id).join("|")}</output>
      <output data-testid="refresh-result-ids">{refreshResultIDs}</output>
      <output data-testid="active-session-id">{controller.activeSessionID}</output>
      <output data-testid="active-session-title">{controller.activeSession?.title || ""}</output>
    </div>
  );
}

describe("useRuntimeSessionController", () => {
  it("keeps preserved sessions active when a refresh briefly returns an empty list", async () => {
    apiClientMock.get.mockResolvedValueOnce({ items: [] });

    render(<RuntimeSessionControllerHarness />);
    expect(screen.getByTestId("active-session-title")).toHaveTextContent("Cached chat");

    fireEvent.click(screen.getByRole("button", { name: "refresh list" }));

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    expect(screen.getByTestId("session-ids")).toHaveTextContent("chat-1");
    expect(screen.getByTestId("refresh-result-ids")).toHaveTextContent("chat-1");
    expect(screen.getByTestId("active-session-id")).toHaveTextContent("chat-1");
    expect(screen.getByTestId("active-session-title")).toHaveTextContent("Cached chat");
  });
});
