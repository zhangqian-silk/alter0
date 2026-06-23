import { fireEvent, render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { buildChatTimelineItems, ChatMessageRegion, type ChatMessageSnapshot } from "./ChatMessageRegion";

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

function buildProcessSteps(): ChatMessageSnapshot["processSteps"] {
  return [
    {
      id: "step-1",
      title: "Inspect workspace",
      detail: "Repository root detected.",
    },
    {
      id: "step-2",
      title: "Review runtime styles",
      detail: "Located the mobile Thinking disclosure rules in `shell.css`.",
    },
    {
      id: "step-3",
      title: "Update inline expansion",
      detail: "Kept process details inside the current assistant message flow.",
    },
    {
      id: "step-4",
      title: "Verify regression coverage",
      detail: "Confirmed the style contract rejects fixed overlay Thinking panels.",
    },
  ];
}

describe("ChatMessageRegion", () => {
  it("reuses unchanged timeline items when only the active streaming message changes", () => {
    const stableMessage = buildAssistantMessage({
      id: "stable-message",
      text: "Stable markdown with `code`.",
    });
    const firstItems = buildChatTimelineItems({
      cacheScope: "streaming-cache-test",
      language: "en",
      messages: [
        stableMessage,
        buildAssistantMessage({
          id: "streaming-message",
          text: "A",
          status: "streaming",
        }),
      ],
    });

    const nextItems = buildChatTimelineItems({
      cacheScope: "streaming-cache-test",
      language: "en",
      messages: [
        stableMessage,
        buildAssistantMessage({
          id: "streaming-message",
          text: "AB",
          status: "streaming",
        }),
      ],
    });

    expect(nextItems[0]).toBe(firstItems[0]);
    expect(nextItems[1]).not.toBe(firstItems[1]);
  });

  it("hides completed chat assistant metadata and message timestamps", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[buildAssistantMessage()]}
      />,
    );

    const article = document.querySelector("[data-message-id='message-1']") as HTMLElement;
    expect(article).toHaveClass("terminal-turn-card");
    expect(article).toHaveClass("conversation-message");
    expect(article).toHaveClass("runtime-message");
    expect(article).toHaveClass("runtime-message-assistant");
    expect(article).toHaveClass("is-assistant");
    expect(article.querySelector(".message-markdown-shell")).toBeInTheDocument();
    expect(article.querySelector(".message-markdown-toolbar")).toBeInTheDocument();
    expect(article.querySelector(".message-markdown-copy")).toBeInTheDocument();
    expect(article.querySelector(".message-markdown-body")).toBeInTheDocument();
    expect(article.querySelector(".terminal-final-output")).toBeInTheDocument();
    expect(article.querySelector(".msg-bubble")).toBeInTheDocument();
    expect(article.querySelector(".assistant-message-shell")).toBeInTheDocument();
    expect(article.querySelector(".runtime-message-bubble")).toBeInTheDocument();
    expect(article.querySelector(".runtime-message-assistant-shell")).toBeInTheDocument();
    expect(article.querySelector(".msg-meta")).not.toBeInTheDocument();
    expect(article.textContent).not.toContain("CHAT");
    expect(article.textContent).not.toContain("MODEL");
    expect(article.textContent).not.toContain("Done");
    expect(article.textContent).not.toContain("10:20");
  });

  it("does not render legacy assistant status chrome while a chat reply is still streaming", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[buildAssistantMessage({ status: "streaming" })]}
      />,
    );

    expect(screen.queryByText("In Progress")).not.toBeInTheDocument();
    expect(document.querySelector(".status-pill")).not.toBeInTheDocument();
    expect(document.querySelector(".msg-meta")).not.toBeInTheDocument();
    expect(document.body.textContent).not.toContain("10:20");
    expect(screen.queryByText("CHAT")).not.toBeInTheDocument();
    expect(screen.queryByText("MODEL")).not.toBeInTheDocument();
  });

  it("hides user prompt timestamps in the shared conversation timeline", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            id: "message-2",
            role: "user",
            text: "Keep this compact",
          }),
        ]}
      />,
    );

    const article = document.querySelector("[data-message-id='message-2']") as HTMLElement;
    expect(article.querySelector(".terminal-turn-prompt")).toBeInTheDocument();
    expect(article).toHaveClass("runtime-message");
    expect(article).toHaveClass("runtime-message-user");
    expect(article.querySelector(".runtime-message-bubble")).toBeInTheDocument();
    expect(article.querySelector(".runtime-message-user-shell")).toBeInTheDocument();
    expect(article.querySelector(".terminal-log-time")).not.toBeInTheDocument();
    expect(article.textContent).not.toContain("10:20");
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

  it("renders inline code without leaking HTML entities", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            text: "链路：`请求接入 -> 召回 -> 粗排 -> 精排 -> 返回广告`",
          }),
        ]}
      />,
    );

    const code = document.querySelector(".chat-md-inline-code") as HTMLElement;
    expect(code).toBeInTheDocument();
    expect(code.textContent).toBe("请求接入 -> 召回 -> 粗排 -> 精排 -> 返回广告");
    expect(code.innerHTML).toContain("-&gt;");
    expect(code.innerHTML).not.toContain("&amp;gt;");
  });

  it("decodes html entities in assistant markdown output", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            text: "Open Chat &gt; Details &gt; Model to switch runtime.",
          }),
        ]}
      />,
    );

    const article = document.querySelector("[data-message-id='message-1']") as HTMLElement;
    expect(article).toBeInTheDocument();
    expect(article.textContent).toContain("Open Chat > Details > Model to switch runtime.");
    expect(article.textContent).not.toContain("&gt;");
  });

  it("renders chat assistant output through the shared runtime markdown shell", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            route: "chat",
            text: "Runtime result with `selectable` markdown.",
          }),
        ]}
      />,
    );

    const article = document.querySelector("[data-message-id='message-1']") as HTMLElement;
    const output = article.querySelector("[data-conversation-final-output='message-1']") as HTMLElement;
    expect(output).toBeInTheDocument();
    expect(output.querySelector(".message-markdown-shell")).toBeInTheDocument();
    expect(output.querySelector(".message-markdown-body")).toBeInTheDocument();
    expect(output.querySelector(".message-markdown-toolbar")).toBeInTheDocument();
    expect(output.querySelector(".message-markdown-rendered")).toBeInTheDocument();
    expect(output.querySelector(".chat-md-inline-code")).toHaveTextContent("selectable");
  });

  it("renders user image attachments from workspace original asset URLs", () => {
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

    const article = document.querySelector("[data-message-id='message-2']") as HTMLElement;
    const image = screen.getByRole("img", { name: "diagram.png" });
    expect(article).toHaveClass("terminal-turn-card");
    expect(article).toHaveClass("conversation-message");
    expect(article).toHaveClass("is-user");
    expect(article.querySelector(".terminal-turn-prompt")).toBeInTheDocument();
    expect(article.querySelector(".msg-bubble")).toBeInTheDocument();
    expect(article.querySelector(".user-message-shell")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-attachment-gallery='message-2']")).toBeInTheDocument();
    expect(image).toHaveAttribute("src", "/api/sessions/session-1/attachments/image-1/original");
    expect(screen.getByText("diagram.png")).toBeInTheDocument();
  });

  it("renders process answers through the shared runtime markdown body contract", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            text: [
              "[process] action: Inspect workspace",
              "[process] observation: Repository root detected.",
              "",
              "Final answer with `details`.",
            ].join("\n"),
            processSteps: buildProcessSteps(),
          }),
        ]}
      />,
    );

    const article = document.querySelector("[data-message-id='message-1']") as HTMLElement;
    expect(article.querySelector(".terminal-process-shell")).toHaveClass("runtime-thinking-shell");
    expect(article.querySelector(".terminal-process-toggle")).toHaveClass("runtime-thinking-toggle");
    expect(article.querySelector(".terminal-process-toggle")).toHaveTextContent("Thinking");
    expect(article.querySelector(".terminal-process-toggle")).toHaveTextContent("4 steps");
    expect(article.querySelector(".terminal-process-shell")).not.toHaveClass("conversation-process-shell");
    expect(article.querySelector(".terminal-process-toggle")).not.toHaveClass("conversation-process-toggle");
    expect(article.querySelector(".terminal-process-body")).toBeInTheDocument();
    expect(article.querySelector(".conversation-process-body")).not.toBeInTheDocument();
    expect(article.querySelectorAll(".terminal-step-item")).toHaveLength(4);
    expect(article.querySelectorAll(".conversation-process-step")).toHaveLength(0);
    expect(article.querySelector(".terminal-step-toggle")).toBeInTheDocument();
    expect(article.querySelector(".conversation-process-step-head")).not.toBeInTheDocument();
    expect(article.querySelector(".terminal-step-title")).toBeInTheDocument();
    expect(article.querySelector(".conversation-process-step-title")).not.toBeInTheDocument();
    expect(article).toHaveTextContent("Review runtime styles");
    expect(article).toHaveTextContent("Verify regression coverage");
    const answer = document.querySelector(".conversation-process-answer") as HTMLElement;
    expect(answer).toBeInTheDocument();
    expect(answer).toHaveClass("message-markdown-body");
    expect(answer.querySelector(".message-markdown-rendered")).toBeInTheDocument();
    expect(article.querySelector(".msg-bubble")).toBeInTheDocument();
  });

  it("renders chat process steps with the same collapsible rows as terminal", async () => {
    const onToggleProcessStep = vi.fn();
    const message = buildAssistantMessage({
      text: "Final answer.",
      processSteps: buildProcessSteps(),
      processCollapsed: false,
    });
    const { rerender } = render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[message]}
        expandedProcessSteps={{}}
        onToggleProcessStep={onToggleProcessStep}
      />,
    );

    const step = document.querySelector("[data-terminal-step-item='step-1']") as HTMLElement;
    expect(step).toHaveAttribute("data-conversation-process-step", "step-1");
    const toggle = step.querySelector("[data-terminal-step-toggle='step-1']") as HTMLButtonElement;
    expect(toggle).toBeInTheDocument();
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(step.querySelector(".terminal-step-body")).toHaveAttribute("hidden");
    expect(step.querySelector(".terminal-step-detail")).toBeInTheDocument();

    fireEvent.click(toggle);
    expect(onToggleProcessStep).toHaveBeenCalledWith("message-1", "step-1");

    rerender(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[message]}
        expandedProcessSteps={{ "message-1:step-1": true }}
        onToggleProcessStep={onToggleProcessStep}
      />,
    );
    const expandedToggle = document.querySelector("[data-terminal-step-toggle='step-1']") as HTMLButtonElement;
    expect(expandedToggle).toHaveAttribute("aria-expanded", "true");
    expect(document.querySelector("[data-terminal-step-item='step-1'] .terminal-step-body")).not.toHaveAttribute("hidden");
  });

  it("uses a compact localized thought disclosure for completed skill process details", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="zh"
        messages={[
          buildAssistantMessage({
            text: "最终答案。",
            processSteps: [
              { id: "step-1", title: "检索资料", detail: "确认当前会话输入。" },
              { id: "step-2", title: "检查样式", detail: "定位移动端展开规则。" },
              { id: "step-3", title: "补充测试", detail: "覆盖多步骤思考过程。" },
              { id: "step-4", title: "完成验证", detail: "确认折叠态文案稳定。" },
            ],
            processCollapsed: true,
          }),
        ]}
      />,
    );

    const process = document.querySelector("[data-conversation-process-shell='message-1']") as HTMLElement;
    const toggle = process.querySelector("[data-conversation-process-toggle='message-1']") as HTMLButtonElement;
    expect(process).toHaveClass("runtime-thinking-shell");
    expect(toggle).toHaveClass("runtime-thinking-toggle");
    expect(process).toHaveClass("is-collapsed");
    expect(toggle).toHaveTextContent("已思考");
    expect(toggle).toHaveTextContent("4 步");
    expect(toggle).not.toHaveTextContent("过程");
    expect(toggle.querySelector(".terminal-step-toggle-icon")).toHaveTextContent(">");
  });

  it("filters chat process disclosure by runtime event type", () => {
    const message = buildAssistantMessage({
      text: "最终答案。",
      processSteps: [
        { id: "commentary", kind: "analysis", title: "执行过程", detail: "正在处理重要进度。" },
        { id: "reasoning", kind: "reasoning", title: "Reasoning", detail: "内部推理摘要。" },
      ],
    });

    const defaultItems = buildChatTimelineItems({
      cacheScope: "runtime-event-filter-default",
      language: "zh",
      messages: [message],
    });
    expect(JSON.stringify(defaultItems)).toContain("正在处理重要进度。");
    expect(JSON.stringify(defaultItems)).toContain("内部推理摘要。");

    const expandedItems = buildChatTimelineItems({
      cacheScope: "runtime-event-filter-expanded",
      language: "zh",
      messages: [message],
      runtimeEventFilter: ["important_text", "reasoning"],
    });
    expect(JSON.stringify(expandedItems)).toContain("正在处理重要进度。");
    expect(JSON.stringify(expandedItems)).toContain("内部推理摘要。");
  });

});
