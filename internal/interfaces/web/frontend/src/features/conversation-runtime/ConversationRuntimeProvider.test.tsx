import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ConversationRuntimeProvider, useConversationRuntime } from "./ConversationRuntimeProvider";

const SESSION_STORAGE_KEY = "alter0.web.sessions.v3";
const ACTIVE_SESSION_STORAGE_KEY = "alter0.web.session.active.v1";
const COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY = "alter0.web.composer.attachments.v1";

const apiClientMock = {
  get: vi.fn(async () => ({ items: [] })),
  post: vi.fn(),
  delete: vi.fn(),
};

vi.mock("../../shared/api/client", () => ({
  createAPIClient: () => apiClientMock,
}));

function RuntimeHarness() {
  const runtime = useConversationRuntime();
  const assistantMessage = runtime.activeSession?.messages.find((message) => message.role === "assistant");

  return (
    <div>
      <button
        type="button"
        onClick={() => void runtime.addDraftAttachments([
          {
            id: "local-image-1",
            name: "trace.png",
            contentType: "image/png",
            size: 12,
            dataURL: "data:image/png;base64,ZmFrZS1pbWFnZS1kYXRh",
            previewDataURL: "data:image/webp;base64,c21hbGwtcHJldmlldw==",
          },
        ])}
      >
        attach
      </button>
      <button type="button" onClick={() => void runtime.sendPrompt("Inspect this image")}>
        send
      </button>
      <output data-testid="assistant-text">{assistantMessage?.text || ""}</output>
    </div>
  );
}

describe("ConversationRuntimeProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.clear();
    window.sessionStorage.setItem(
      SESSION_STORAGE_KEY,
      JSON.stringify([
        {
          id: "session-1",
          title: "Image session",
          titleAuto: false,
          titleScore: 1,
          createdAt: Date.parse("2026-04-23T03:30:00Z"),
          targetType: "model",
          targetID: "raw-model",
          targetName: "Raw Model",
          modelProviderID: "",
          modelID: "",
          toolIDs: [],
          skillIDs: [],
          mcpIDs: [],
          messages: [],
        },
      ]),
    );
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "session-1", "agent-runtime": "" }),
    );
    window.sessionStorage.setItem(
      COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY,
      JSON.stringify({
        "session-1": [
          {
            id: "image-1",
            name: "trace.png",
            content_type: "image/png",
            size: 12,
            asset_url: "/api/sessions/session-1/attachments/image-1/original",
            preview_url: "/api/sessions/session-1/attachments/image-1/preview",
          },
        ],
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not rewrite stored sessions for streaming deltas after an image message is queued", async () => {
    const encoder = new TextEncoder();
    let streamController: ReadableStreamDefaultController<Uint8Array> | null = null;
    const fetchMock = vi.fn(async () => new Response(new ReadableStream({
      start(controller) {
        streamController = controller;
      },
    }), {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
      },
    }));
    vi.stubGlobal("fetch", fetchMock);

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    const setItemSpy = vi.spyOn(Storage.prototype, "setItem");

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const request = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body)) as {
      attachments?: Array<Record<string, string>>;
    };
    expect(request.attachments?.[0]).toMatchObject({
      id: "image-1",
      asset_url: "/api/sessions/session-1/attachments/image-1/original",
      preview_url: "/api/sessions/session-1/attachments/image-1/preview",
    });
    expect(request.attachments?.[0]?.data_url).toBeUndefined();

    await act(async () => {
      await new Promise((resolve) => window.setTimeout(resolve, 220));
    });

    const sessionWritesAfterQueue = setItemSpy.mock.calls.filter(([key]) => key === SESSION_STORAGE_KEY).length;
    expect(sessionWritesAfterQueue).toBe(1);
    const latestQueuedSessionPayload = setItemSpy.mock.calls
      .filter(([key]) => key === SESSION_STORAGE_KEY)
      .at(-1)?.[1];
    expect(typeof latestQueuedSessionPayload).toBe("string");
    const queuedSessions = JSON.parse(String(latestQueuedSessionPayload)) as Array<{
      messages?: Array<{ attachments?: Array<{ preview_url?: string; data_url?: string }> }>;
    }>;
    expect(queuedSessions[0]?.messages?.[0]?.attachments?.[0]?.preview_url).toBe("/api/sessions/session-1/attachments/image-1/preview");
    expect(queuedSessions[0]?.messages?.[0]?.attachments?.[0]?.data_url).toBeUndefined();

    await act(async () => {
      streamController?.enqueue(encoder.encode('event: delta\ndata: {"delta":"Analyzing"}\n\n'));
      await Promise.resolve();
      await new Promise((resolve) => window.setTimeout(resolve, 220));
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Analyzing"));

    const sessionWritesAfterDelta = setItemSpy.mock.calls.filter(([key]) => key === SESSION_STORAGE_KEY).length;
    expect(sessionWritesAfterDelta).toBe(1);

    await act(async () => {
      streamController?.enqueue(encoder.encode('event: done\ndata: {"result":{"output":"Analyzing complete"}}\n\n'));
      streamController?.close();
      await Promise.resolve();
      await new Promise((resolve) => window.setTimeout(resolve, 220));
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Analyzing complete"));

    const sessionWritesAfterDone = setItemSpy.mock.calls.filter(([key]) => key === SESSION_STORAGE_KEY).length;
    expect(sessionWritesAfterDone).toBe(2);
  });

  it("uploads draft images into the session workspace before they are persisted locally", async () => {
    apiClientMock.post.mockResolvedValueOnce({
      items: [
        {
          id: "image-1",
          name: "trace.png",
          content_type: "image/png",
          size: 12,
          asset_url: "/api/sessions/session-1/attachments/image-1/original",
          preview_url: "/api/sessions/session-1/attachments/image-1/preview",
        },
      ],
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "attach" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/sessions/session-1/attachments",
      {
        attachments: [
          {
            name: "trace.png",
            content_type: "image/png",
            data_url: "data:image/png;base64,ZmFrZS1pbWFnZS1kYXRh",
            preview_data_url: "data:image/webp;base64,c21hbGwtcHJldmlldw==",
          },
        ],
      },
    ));

    const persistedDrafts = JSON.parse(
      window.sessionStorage.getItem(COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY) || "{}",
    ) as Record<string, Array<Record<string, string>>>;
    expect(persistedDrafts["session-1"]?.[0]).toMatchObject({
      id: "image-1",
      assetURL: "/api/sessions/session-1/attachments/image-1/original",
      previewURL: "/api/sessions/session-1/attachments/image-1/preview",
    });
    expect(persistedDrafts["session-1"]?.[0]?.dataURL).toBeUndefined();
  });
});
