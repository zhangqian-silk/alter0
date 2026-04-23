import { render, screen, within } from "@testing-library/react";
import { ChatMessageRegion, type ChatMessageSnapshot } from "./ChatMessageRegion";

function buildAssistantMessage(overrides: Partial<ChatMessageSnapshot> = {}): ChatMessageSnapshot {
  return {
    id: "message-1",
    role: "assistant",
    text: "Completed the update.",
    attachments: [],
    route: "chat",
    source: "model",
    error: false,
    status: "done",
    at: Date.parse("2026-04-21T02:20:00Z"),
    processSteps: [],
    ...overrides,
  };
}

describe("ChatMessageRegion", () => {
  it("keeps completed chat assistant metadata minimal", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[buildAssistantMessage()]}
      />,
    );

    const article = document.querySelector("[data-message-id='message-1']") as HTMLElement;
    const meta = article.querySelector(".msg-meta") as HTMLElement;

    expect(meta).toBeInTheDocument();
    expect(within(meta).queryByText("CHAT")).not.toBeInTheDocument();
    expect(within(meta).queryByText("MODEL")).not.toBeInTheDocument();
    expect(within(meta).queryByText("Done")).not.toBeInTheDocument();
    expect(meta.textContent).toContain("10:20");
  });

  it("keeps transient assistant status visible while a chat reply is still streaming", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[buildAssistantMessage({ status: "streaming" })]}
      />,
    );

    expect(screen.getByText("In Progress")).toBeInTheDocument();
    expect(screen.queryByText("CHAT")).not.toBeInTheDocument();
    expect(screen.queryByText("MODEL")).not.toBeInTheDocument();
  });

  it("renders assistant markdown images as lazy-loaded message media", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            text: "Preview:\n\n![Generated diagram](https://cdn.example.com/generated-diagram.png)",
          }),
        ]}
      />,
    );

    const image = screen.getByRole("img", { name: "Generated diagram" });
    expect(image).toHaveAttribute("src", "https://cdn.example.com/generated-diagram.png");
    expect(image).toHaveAttribute("loading", "lazy");
    expect(image).toHaveAttribute("decoding", "async");
  });

  it("renders user image attachments from workspace preview URLs", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            id: "message-2",
            role: "user",
            text: "Inspect this diagram",
            attachments: [
              {
                id: "image-1",
                name: "diagram.png",
                contentType: "image/png",
                size: 1024,
                previewURL: "/api/sessions/session-1/attachments/image-1/preview",
                assetURL: "/api/sessions/session-1/attachments/image-1/original",
              },
            ],
          }),
        ]}
      />,
    );

    const image = screen.getByRole("img", { name: "diagram.png" });
    expect(image).toHaveAttribute("src", "/api/sessions/session-1/attachments/image-1/preview");
    expect(screen.getByText("diagram.png")).toBeInTheDocument();
  });
});
