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
    processEvents: [],
    ...overrides,
  };
}

function buildProcessSteps(): ChatMessageSnapshot["processEvents"] {
  return [
    buildProcessEvent("step-1", "Inspect workspace", "Repository root detected."),
    buildProcessEvent("step-2", "Review runtime styles", "Located the mobile Thinking disclosure rules in `shell.css`."),
    buildProcessEvent("step-3", "Update inline expansion", "Kept process details inside the current assistant message flow."),
    buildProcessEvent("step-4", "Verify regression coverage", "Confirmed the style contract rejects fixed overlay Thinking panels."),
  ];
}

function buildProcessEvent(
  id: string,
  title: string,
  text: string,
  kind: ChatMessageSnapshot["processEvents"][number]["kind"] = "assistant_commentary",
): ChatMessageSnapshot["processEvents"][number] {
  return {
    id,
    turn_id: "turn-1",
    seq: 1,
    source: "adapter",
    provider: { engine: "codex", adapter: "codex_cli_json" },
    role: "assistant",
    kind,
    lifecycle: "completed",
    status: "completed",
    title,
    summary: title,
    blocks: [{ type: "markdown", text }],
    visibility: "collapsed",
  };
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
                previewURL: "/api/chat/sessions/session-1/attachments/image-1/preview",
                assetURL: "/api/chat/sessions/session-1/attachments/image-1/original",
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
    expect(image).toHaveAttribute("src", "/api/chat/sessions/session-1/attachments/image-1/original");
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
            processEvents: buildProcessSteps(),
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

  it("renders chat process events with the same collapsible rows as terminal", async () => {
    const onToggleProcessEvent = vi.fn();
    const message = buildAssistantMessage({
      text: "Final answer.",
      processEvents: buildProcessSteps(),
      processCollapsed: false,
    });
    const { rerender } = render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[message]}
        expandedProcessEvents={{}}
        onToggleProcessEvent={onToggleProcessEvent}
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
    expect(onToggleProcessEvent).toHaveBeenCalledWith("message-1", "step-1");

    rerender(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[message]}
        expandedProcessEvents={{ "message-1:step-1": true }}
        onToggleProcessEvent={onToggleProcessEvent}
      />,
    );
    const expandedToggle = document.querySelector("[data-terminal-step-toggle='step-1']") as HTMLButtonElement;
    expect(expandedToggle).toHaveAttribute("aria-expanded", "true");
    expect(document.querySelector("[data-terminal-step-item='step-1'] .terminal-step-body")).not.toHaveAttribute("hidden");
  });

  it("renders shell process event details as terminal blocks on the first expanded frame", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            text: "",
            processEvents: [
              {
                id: "shell-step",
                session_id: "session-1",
                turn_id: "turn-1",
                seq: 1,
                source: "adapter",
                provider: { engine: "codex", adapter: "codex_cli_json" },
                role: "assistant",
                kind: "shell_command",
                lifecycle: "completed",
                status: "completed",
                title: "Shell",
                summary: "sed -n '1,120p' AGENTS.md",
                blocks: [
                  {
                    type: "terminal",
                    command: "sed -n '1,120p' AGENTS.md",
                    output: "# Rule\n\n## Collaboration",
                    language: "shell",
                    exit_code: 0,
                  },
                ],
                visibility: "collapsed",
              },
            ],
            processCollapsed: false,
          }),
        ]}
        expandedProcessEvents={{ "message-1:shell-step": true }}
        runtimeEventFilter={["commands"]}
      />,
    );

    const step = document.querySelector("[data-terminal-step-item='shell-step']") as HTMLElement;
    const content = step.querySelector(".terminal-step-content code") as HTMLElement;
    expect(content).toBeInTheDocument();
    expect(content.textContent).toBe("sed -n '1,120p' AGENTS.md\n\n# Rule\n\n## Collaboration");
    expect(step.querySelector(".terminal-step-detail > .message-markdown-rendered")).not.toBeInTheDocument();
  });

  it("renders markdown process event details through the final rich block on the first expanded frame", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            text: "",
            processEvents: [
              {
                id: "commentary-step",
                session_id: "session-1",
                turn_id: "turn-1",
                seq: 1,
                source: "provider",
                provider: { engine: "codex", adapter: "codex_cli_json" },
                role: "assistant",
                kind: "assistant_commentary",
                lifecycle: "completed",
                status: "completed",
                title: "Message",
                summary: "Ready.",
                blocks: [
                  {
                    type: "markdown",
                    text: "Review **state** before rendering.",
                  },
                ],
                visibility: "collapsed",
              },
            ],
            processCollapsed: false,
          }),
        ]}
        expandedProcessEvents={{ "message-1:commentary-step": true }}
        runtimeEventFilter={["important_text"]}
      />,
    );

    const step = document.querySelector("[data-terminal-step-item='commentary-step']") as HTMLElement;
    const block = step.querySelector(".terminal-rich-block.type-markdown") as HTMLElement;
    expect(block).toBeInTheDocument();
    expect(block.querySelector(".terminal-step-content.terminal-step-richtext")).toBeInTheDocument();
    expect(block.querySelector(".message-markdown-rendered")).toHaveTextContent("Review state before rendering.");
    expect(step.querySelector(".terminal-step-detail > .message-markdown-rendered")).not.toBeInTheDocument();
  });

  it("renders runtime code process details with the terminal rich header", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            text: "",
            processEvents: [
              {
                id: "code-step",
                session_id: "session-1",
                turn_id: "turn-1",
                seq: 1,
                source: "adapter",
                provider: { engine: "codex", adapter: "codex_cli_json" },
                role: "assistant",
                kind: "file_read",
                lifecycle: "completed",
                status: "completed",
                title: "Read source",
                summary: "src/features/runtime.ts",
                blocks: [
                  {
                    type: "code",
                    content: "export const enabled = true;\n",
                    language: "ts",
                    file: "src/features/runtime.ts",
                    start_line: 7,
                  },
                ],
                visibility: "collapsed",
              },
            ],
            processCollapsed: false,
          }),
        ]}
        expandedProcessEvents={{ "message-1:code-step": true }}
        runtimeEventFilter={["tools"]}
      />,
    );

    const step = document.querySelector("[data-terminal-step-item='code-step']") as HTMLElement;
    const block = step.querySelector(".terminal-rich-block.type-code") as HTMLElement;
    expect(block).toBeInTheDocument();
    expect(block.querySelector(".terminal-rich-head")).toHaveTextContent("src/features/runtime.ts:7");
    expect(block.querySelector(".terminal-rich-pre.terminal-step-content code")).toHaveTextContent("export const enabled = true;");
  });

  it("renders process event detail through the final rich block on the first expanded frame", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[
          buildAssistantMessage({
            text: "Final answer.",
            processEvents: buildProcessSteps(),
            processCollapsed: false,
          }),
        ]}
        expandedProcessEvents={{ "message-1:step-1": true }}
      />,
    );

    const step = document.querySelector("[data-terminal-step-item='step-1']") as HTMLElement;
    const block = step.querySelector(".terminal-rich-block.type-markdown") as HTMLElement;
    expect(block).toBeInTheDocument();
    expect(block.querySelector(".terminal-step-content.terminal-step-richtext")).toBeInTheDocument();
    expect(block.querySelector(".message-markdown-rendered")).toHaveTextContent("Repository root detected.");
    expect(step.querySelector(".terminal-step-detail > .message-markdown-rendered")).not.toBeInTheDocument();
  });

  it("uses a compact localized thought disclosure for completed skill process details", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="zh"
        messages={[
          buildAssistantMessage({
            text: "最终答案。",
            processEvents: [
              buildProcessEvent("step-1", "检索资料", "确认当前会话输入。"),
              buildProcessEvent("step-2", "检查样式", "定位移动端展开规则。"),
              buildProcessEvent("step-3", "补充测试", "覆盖多步骤思考过程。"),
              buildProcessEvent("step-4", "完成验证", "确认折叠态文案稳定。"),
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
      processEvents: [
        buildProcessEvent("commentary", "执行过程", "正在处理重要进度。", "assistant_commentary"),
        buildProcessEvent("reasoning", "Reasoning", "内部推理摘要。", "reasoning"),
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

  it("discloses runtime event categories on chat process events", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        runtimeEventFilter={["important_text", "plan"]}
        messages={[buildAssistantMessage({
          processCollapsed: false,
          processEvents: [
            {
              id: "plan-step",
              turn_id: "turn-1",
              seq: 1,
              source: "provider",
              provider: { engine: "codex", adapter: "codex_cli_json" },
              role: "assistant",
              kind: "plan",
              lifecycle: "completed",
              status: "completed",
              title: "Route build",
              blocks: [{ type: "markdown", text: "Plan detail." }],
              visibility: "collapsed",
            },
          ],
        })]}
      />,
    );

    const step = document.querySelector("[data-terminal-step-item='plan-step']") as HTMLElement;
    expect(step.querySelector(".terminal-step-kind")).toHaveTextContent("Plan");
  });

  it("uses the shared terminal step meta for runtime chat process rows", () => {
    render(
      <ChatMessageRegion
        sessionId="session-1"
        language="en"
        messages={[buildAssistantMessage({
          processCollapsed: false,
          processEvents: [
            {
              id: "command-step",
              turn_id: "turn-1",
              seq: 1,
              source: "adapter",
              provider: { engine: "codex", adapter: "codex_cli_json" },
              role: "assistant",
              kind: "shell_command",
              lifecycle: "completed",
              status: "completed",
              duration_ms: 18200,
              title: "sed -n '1,120p' internal/terminal/application/session.go",
              blocks: [{ type: "terminal", command: "sed -n '1,120p' internal/terminal/application/session.go" }],
              visibility: "collapsed",
            },
          ],
        })]}
        runtimeEventFilter={["commands"]}
      />,
    );

    const step = document.querySelector("[data-terminal-step-item='command-step']") as HTMLElement;
    const meta = step.querySelector(".terminal-step-meta") as HTMLElement;
    expect(meta).toBeInTheDocument();
    expect(meta.querySelector(".terminal-step-kind")).toHaveTextContent("Commands");
    expect(meta.querySelector(".terminal-step-duration")).toHaveTextContent("18s");
    expect(meta.querySelector(".terminal-step-status")).toHaveTextContent("Ready");
    expect(meta.querySelector(".terminal-step-status")).toHaveClass("status-success");
  });

});
