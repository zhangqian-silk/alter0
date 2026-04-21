import { memo } from "react";
import { formatTimeLabel } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";

export type ChatMessageSnapshot = {
  id: string;
  role: "user" | "assistant";
  text: string;
  route: string;
  source: string;
  error: boolean;
  status: string;
  at: number;
  processSteps: ChatMessageProcessStepSnapshot[];
  agentProcessCollapsed?: boolean;
};

export type ChatMessageProcessStepSnapshot = {
  id: string;
  kind: string;
  title: string;
  detail: string;
  status: string;
};

type MessageCopy = {
  statusInProgress: string;
  statusQueued: string;
  statusRunning: string;
  statusCanceled: string;
  statusSuccess: string;
  statusFailed: string;
  statusDone: string;
  processLabel: string;
  processSteps: (count: number) => string;
  processEmpty: string;
  processObservation: string;
  copyValue: string;
};

const MESSAGE_COPY: Record<LegacyShellLanguage, MessageCopy> = {
  en: {
    statusInProgress: "In Progress",
    statusQueued: "Queued",
    statusRunning: "Running",
    statusCanceled: "Canceled",
    statusSuccess: "Success",
    statusFailed: "Failed",
    statusDone: "Done",
    processLabel: "Process",
    processSteps: (count) => `${count} steps`,
    processEmpty: "No execution details.",
    processObservation: "Observation",
    copyValue: "Copy value",
  },
  zh: {
    statusInProgress: "进行中",
    statusQueued: "排队中",
    statusRunning: "运行中",
    statusCanceled: "已取消",
    statusSuccess: "成功",
    statusFailed: "失败",
    statusDone: "完成",
    processLabel: "过程",
    processSteps: (count) => `${count} 步`,
    processEmpty: "暂无执行细节。",
    processObservation: "观察",
    copyValue: "复制内容",
  },
};

export const ChatMessageRegion = memo(function ChatMessageRegion({
  sessionId,
  messages,
  language,
  onToggleProcess,
}: {
  sessionId: string;
  messages: ChatMessageSnapshot[];
  language: LegacyShellLanguage;
  onToggleProcess?: (messageID: string) => void;
}) {
  return (
    <div className="message-list" data-message-session-id={sessionId}>
      {messages.map((message) => (
        <ChatMessageArticle
          key={message.id}
          message={message}
          language={language}
          onToggleProcess={onToggleProcess}
        />
      ))}
    </div>
  );
});

const ChatMessageArticle = memo(function ChatMessageArticle({
  message,
  language,
  onToggleProcess,
}: {
  message: ChatMessageSnapshot;
  language: LegacyShellLanguage;
  onToggleProcess?: (messageID: string) => void;
}) {
  const classNames = ["msg", message.role];
  if (message.error) {
    classNames.push("error");
  }
  if (message.status === "streaming") {
    classNames.push("streaming");
  }

  return (
    <article className={classNames.join(" ")} data-message-id={message.id}>
      <div className="msg-bubble">
        {message.role === "assistant" ? (
          <AssistantMessageBody message={message} language={language} onToggleProcess={onToggleProcess} />
        ) : (
          <MarkdownHTML html={renderMarkdownToHTML(message.text)} />
        )}
      </div>
      <div className="msg-meta">
        {message.role === "assistant" ? (
          <>
            {shouldShowAssistantStatus(message) ? (
              <span className={`status-pill ${message.status || "done"}`}>
                {assistantStatusLabel(message.status, language)}
              </span>
            ) : null}
          </>
        ) : null}
        <span>{formatTimeLabel(message.at)}</span>
      </div>
    </article>
  );
});

function AssistantMessageBody({
  message,
  language,
  onToggleProcess,
}: {
  message: ChatMessageSnapshot;
  language: LegacyShellLanguage;
  onToggleProcess?: (messageID: string) => void;
}) {
  const copy = MESSAGE_COPY[language];
  const parsed = resolveAgentExecutionContent(message, language);
  if (!parsed.steps.length) {
    if (message.status === "streaming") {
      return <MarkdownHTML html={renderMarkdownToHTML(message.text)} />;
    }
    return (
      <AssistantFinalBody
        contentHTML={renderMarkdownToHTML(message.text)}
        copyValue={message.text.trim()}
        copyLabel={copy.copyValue}
      />
    );
  }

  const collapsed =
    typeof message.agentProcessCollapsed === "boolean"
      ? message.agentProcessCollapsed
      : Boolean(parsed.answer.trim()) && message.status !== "streaming";

  return (
    <>
      <section
        className={`agent-process-shell ${collapsed ? "is-collapsed" : ""}`}
        data-agent-process-shell={message.id}
      >
        <button
          className="agent-process-toggle"
          type="button"
          data-agent-process-toggle={message.id}
          aria-expanded={collapsed ? "false" : "true"}
          onClick={() => onToggleProcess?.(message.id)}
        >
          <span className="agent-process-toggle-icon">{collapsed ? ">" : "v"}</span>
          <span className="agent-process-copy">
            <span className="agent-process-title">{copy.processLabel}</span>
            <span className="agent-process-summary">{copy.processSteps(parsed.steps.length)}</span>
          </span>
        </button>
        <div className="agent-process-body" hidden={collapsed}>
          {parsed.steps.length ? (
            parsed.steps.map((step, index) => (
              <article key={step.id || `${step.title}-${index}`} className="agent-process-step">
                <div className="agent-process-step-head">
                  <span className="agent-process-step-index">{index + 1}</span>
                  <span className="agent-process-step-title">
                    {step.title || `${copy.processLabel} ${index + 1}`}
                  </span>
                </div>
                {step.detail ? (
                  <div className="agent-process-step-body">
                    <MarkdownHTML html={renderMarkdownToHTML(step.detail)} />
                  </div>
                ) : null}
              </article>
            ))
          ) : (
            <div className="agent-process-empty">{copy.processEmpty}</div>
          )}
        </div>
      </section>
      {parsed.answer.trim() ? (
        <AssistantFinalBody
          contentHTML={`<div class="agent-process-answer">${renderMarkdownToHTML(parsed.answer)}</div>`}
          copyValue={parsed.answer}
          copyLabel={copy.copyValue}
          className="agent-process-answer-shell"
        />
      ) : null}
    </>
  );
}

function AssistantFinalBody({
  contentHTML,
  copyValue,
  copyLabel,
  className = "",
}: {
  contentHTML: string;
  copyValue: string;
  copyLabel: string;
  className?: string;
}) {
  if (!contentHTML.trim()) {
    return null;
  }

  return (
    <div className={`assistant-message-shell${className ? ` ${className}` : ""}`}>
      <div className="assistant-message-toolbar">
        {copyValue.trim() ? (
          <button
            className="route-field-copy assistant-message-copy"
            type="button"
            data-copy-value={copyValue}
            title={copyLabel}
            aria-label={copyLabel}
          >
            <CopyIcon />
          </button>
        ) : null}
      </div>
      <div className="assistant-message-body">
        <MarkdownHTML html={contentHTML} />
      </div>
    </div>
  );
}

function MarkdownHTML({ html }: { html: string }) {
  return <div dangerouslySetInnerHTML={{ __html: html }} />;
}

function assistantStatusLabel(status: string, language: LegacyShellLanguage) {
  const copy = MESSAGE_COPY[language];
  switch (status) {
    case "streaming":
      return copy.statusInProgress;
    case "queued":
      return copy.statusQueued;
    case "running":
      return copy.statusRunning;
    case "canceled":
      return copy.statusCanceled;
    case "success":
      return copy.statusSuccess;
    case "failed":
    case "error":
      return copy.statusFailed;
    default:
      return copy.statusDone;
  }
}

function shouldShowAssistantStatus(message: ChatMessageSnapshot) {
  if (message.error) {
    return true;
  }
  const normalized = message.status.trim().toLowerCase();
  return normalized !== "" && normalized !== "done" && normalized !== "success";
}

function resolveAgentExecutionContent(
  message: ChatMessageSnapshot,
  language: LegacyShellLanguage,
) {
  if (message.processSteps.length) {
    return {
      steps: message.processSteps,
      answer: message.text.trim(),
    };
  }
  return parseAgentExecutionText(message.text, language);
}

function parseAgentExecutionText(value: string, language: LegacyShellLanguage) {
  const copy = MESSAGE_COPY[language];
  const normalized = value.replace(/\r\n?/g, "\n");
  if (!normalized.trim()) {
    return { steps: [] as ChatMessageProcessStepSnapshot[], answer: "" };
  }

  const lines = normalized.split("\n");
  const steps: ChatMessageProcessStepSnapshot[] = [];
  const answerLines: string[] = [];
  let currentStep: ChatMessageProcessStepSnapshot | null = null;
  let index = 0;

  const pushCurrentStep = () => {
    if (!currentStep) {
      return;
    }
    if (!currentStep.title.trim() && !currentStep.detail.trim()) {
      currentStep = null;
      return;
    }
    steps.push(currentStep);
    currentStep = null;
  };

  while (index < lines.length) {
    const line = lines[index];
    const trimmed = line.trim();
    if (trimmed.startsWith("[agent] action:")) {
      pushCurrentStep();
      currentStep = {
        id: "",
        kind: "action",
        title: trimmed.slice("[agent] action:".length).trim(),
        detail: "",
        status: "",
      };
      index += 1;
      continue;
    }
    if (trimmed === "[agent] observation:") {
      const detailLines: string[] = [];
      index += 1;
      while (index < lines.length) {
        const nextLine = lines[index];
        const nextTrimmed = nextLine.trim();
        if (nextTrimmed.startsWith("[agent] action:") || nextTrimmed === "[agent] observation:") {
          break;
        }
        detailLines.push(nextLine);
        index += 1;
      }
      const detail = detailLines.join("\n").trim();
      if (currentStep) {
        currentStep.detail = detail;
      } else {
        currentStep = {
          id: "",
          kind: "observation",
          title: copy.processObservation,
          detail,
          status: "",
        };
        pushCurrentStep();
      }
      continue;
    }
    answerLines.push(line);
    index += 1;
  }

  pushCurrentStep();
  return {
    steps,
    answer: answerLines.join("\n").trim(),
  };
}

function renderMarkdownToHTML(value: string) {
  const normalized = String(value ?? "").replace(/\r\n?/g, "\n");
  if (!normalized.trim()) {
    return "";
  }
  const tokens: Array<{ type: "markdown" | "code"; content: string; language?: string }> = [];
  const fencePattern = /```([a-zA-Z0-9_-]+)?\n([\s\S]*?)```/g;
  let cursor = 0;
  let match = fencePattern.exec(normalized);
  while (match) {
    if (match.index > cursor) {
      tokens.push({ type: "markdown", content: normalized.slice(cursor, match.index) });
    }
    tokens.push({
      type: "code",
      language: String(match[1] || "").trim().toLowerCase(),
      content: String(match[2] || "").replace(/\n$/, ""),
    });
    cursor = match.index + match[0].length;
    match = fencePattern.exec(normalized);
  }
  if (cursor < normalized.length) {
    tokens.push({ type: "markdown", content: normalized.slice(cursor) });
  }
  return tokens
    .map((token) => {
      if (token.type === "code") {
        const languageClass = token.language ? ` class="language-${escapeHTML(token.language)}"` : "";
        return `<pre class="chat-md-pre"><code${languageClass}>${escapeHTML(token.content)}</code></pre>`;
      }
      return renderMarkdownBlocks(token.content);
    })
    .join("");
}

function renderMarkdownBlocks(content: string) {
  const lines = String(content || "").split("\n");
  const html: string[] = [];
  let paragraphLines: string[] = [];
  let quoteLines: string[] = [];
  let listType = "";
  let listItems: string[] = [];

  const flushParagraph = () => {
    if (!paragraphLines.length) {
      return;
    }
    html.push(`<p>${paragraphLines.map((line) => renderMarkdownInline(line)).join("<br>")}</p>`);
    paragraphLines = [];
  };

  const flushQuote = () => {
    if (!quoteLines.length) {
      return;
    }
    html.push(`<blockquote>${renderMarkdownBlocks(quoteLines.join("\n"))}</blockquote>`);
    quoteLines = [];
  };

  const flushList = () => {
    if (!listType || !listItems.length) {
      listType = "";
      listItems = [];
      return;
    }
    html.push(
      `<${listType}>${listItems
        .map((item) => `<li>${renderMarkdownInline(item)}</li>`)
        .join("")}</${listType}>`,
    );
    listType = "";
    listItems = [];
  };

  const flushAll = () => {
    flushParagraph();
    flushQuote();
    flushList();
  };

  for (const rawLine of lines) {
    const trimmed = rawLine.trim();
    if (!trimmed) {
      flushAll();
      continue;
    }

    if (/^>\s?/.test(trimmed)) {
      flushParagraph();
      flushList();
      quoteLines.push(trimmed.replace(/^>\s?/, ""));
      continue;
    }
    flushQuote();

    const unorderedMatch = /^[-*+]\s+(.+)$/.exec(trimmed);
    if (unorderedMatch) {
      flushParagraph();
      if (listType && listType !== "ul") {
        flushList();
      }
      listType = "ul";
      listItems.push(unorderedMatch[1]);
      continue;
    }

    const orderedMatch = /^(\d+)\.\s+(.+)$/.exec(trimmed);
    if (orderedMatch) {
      flushParagraph();
      if (listType && listType !== "ol") {
        flushList();
      }
      listType = "ol";
      listItems.push(orderedMatch[2]);
      continue;
    }

    flushList();

    const headingMatch = /^(#{1,6})\s+(.+)$/.exec(trimmed);
    if (headingMatch) {
      flushParagraph();
      const level = headingMatch[1].length;
      html.push(`<h${level}>${renderMarkdownInline(headingMatch[2])}</h${level}>`);
      continue;
    }

    if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph();
      html.push("<hr>");
      continue;
    }

    paragraphLines.push(trimmed);
  }

  flushAll();
  return html.join("");
}

function renderMarkdownInline(content: string) {
  let rendered = escapeHTML(String(content ?? ""));
  const placeholders: string[] = [];
  const reserve = (html: string) => {
    const token = `\u0000${placeholders.length}\u0000`;
    placeholders.push(html);
    return token;
  };

  rendered = rendered.replace(/`([^`]+)`/g, (_, code: string) =>
    reserve(`<code class="chat-md-inline-code">${escapeHTML(code)}</code>`),
  );
  rendered = rendered.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label: string, url: string) => {
    const href = sanitizeMarkdownURL(url);
    if (!href) {
      return renderMarkdownInline(label);
    }
    return reserve(
      `<a href="${href}" target="_blank" rel="noreferrer noopener">${renderMarkdownInline(label)}</a>`,
    );
  });
  rendered = rendered.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  rendered = rendered.replace(/__([^_]+)__/g, "<strong>$1</strong>");
  rendered = rendered.replace(/(^|[\s(>])\*([^*\n]+)\*(?=$|[\s).,!?:;<])/g, "$1<em>$2</em>");
  rendered = rendered.replace(/(^|[\s(>])_([^_\n]+)_(?=$|[\s).,!?:;<])/g, "$1<em>$2</em>");

  return rendered.replace(/\u0000(\d+)\u0000/g, (_, index: string) => placeholders[Number(index)] || "");
}

function sanitizeMarkdownURL(rawURL: string) {
  const value = String(rawURL || "").trim();
  if (!value) {
    return "";
  }
  const normalized = value.replace(/^<|>$/g, "");
  if (/^(https?:|mailto:)/i.test(normalized) || normalized.startsWith("/") || normalized.startsWith("#")) {
    return escapeHTML(normalized);
  }
  return "";
}

function escapeHTML(value: string) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => {
    if (char === "&") return "&amp;";
    if (char === "<") return "&lt;";
    if (char === ">") return "&gt;";
    if (char === '"') return "&quot;";
    return "&#39;";
  });
}

function CopyIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="12" height="12" rx="2"></rect>
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
    </svg>
  );
}
