import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
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
      <button type="button" onClick={() => void controller.refreshActiveSession("chat-1")}>refresh detail</button>
      <button type="button" onClick={() => void controller.createSession()}>create session</button>
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

  it("does not let an older detail request overwrite a newer response", async () => {
    let resolveFirst: (value: unknown) => void = () => undefined;
    let resolveSecond: (value: unknown) => void = () => undefined;
    const first = new Promise((resolve) => { resolveFirst = resolve; });
    const second = new Promise((resolve) => { resolveSecond = resolve; });
    apiClientMock.get
      .mockReturnValueOnce(first)
      .mockReturnValueOnce(second);

    render(<RuntimeSessionControllerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "refresh detail" }));
    fireEvent.click(screen.getByRole("button", { name: "refresh detail" }));

    await act(async () => {
      resolveSecond({ session: { id: "chat-1", title: "Newest detail", status: "ready" } });
      await second;
    });
    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Newest detail"));

    await act(async () => {
      resolveFirst({ session: { id: "chat-1", title: "Older detail", status: "ready" } });
      await first;
    });
    expect(screen.getByTestId("active-session-title")).toHaveTextContent("Newest detail");
  });

  it("does not let a list requested before a mutation overwrite the mutation result", async () => {
    let resolveList: (value: unknown) => void = () => undefined;
    const pendingList = new Promise((resolve) => { resolveList = resolve; });
    apiClientMock.get.mockReturnValueOnce(pendingList);
    apiClientMock.post.mockResolvedValueOnce({
      session: { id: "chat-1", title: "Created detail", status: "ready" },
    });

    render(<RuntimeSessionControllerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "refresh list" }));
    fireEvent.click(screen.getByRole("button", { name: "create session" }));
    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Created detail"));

    await act(async () => {
      resolveList({ items: [{ id: "chat-1", title: "Older list", status: "ready" }] });
      await pendingList;
    });
    expect(screen.getByTestId("active-session-title")).toHaveTextContent("Created detail");
  });
});
