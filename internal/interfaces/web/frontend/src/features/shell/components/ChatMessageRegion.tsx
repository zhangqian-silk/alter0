import { memo } from "react";
import { formatTimeLabel } from "../../../shared/time/format";
import { resolveComposerAttachmentPreviewURL } from "../../conversation-runtime/composerImageAttachments";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  RuntimeTimeline,
  type RuntimeTimelineItem,
} from "./RuntimeTimeline";
import { RuntimeMarkdownHTML } from "./RuntimeTimelinePrimitives";
import { renderRuntimeMarkdownToHTML } from "./RuntimeMarkdown";

export type ChatMessageSnapshot = {
  id: string;
  role: "user" | "assistant";
  text: string;
  attachments: Array<{
    id: string;
    name: string;
    contentType: string;
    size: number;
    dataURL?: string;
    previewDataURL?: string;
    assetURL?: string;
    previewURL?: string;
  }>;
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
    <RuntimeTimeline
      className="message-list"
      timelineProps={{ "data-message-session-id": sessionId }}
      items={buildChatTimelineItems({ messages, language, onToggleProcess })}
    />
  );
});

export function buildChatTimelineItems({
  messages,
  language,
  onToggleProcess,
}: {
  messages: ChatMessageSnapshot[];
  language: LegacyShellLanguage;
  onToggleProcess?: (messageID: string) => void;
}) {
  const copy = MESSAGE_COPY[language];
  return messages.map((message) => buildChatTimelineItem(message, language, copy, onToggleProcess));
}

function buildChatTimelineItem(
  message: ChatMessageSnapshot,
  language: LegacyShellLanguage,
  copy: MessageCopy,
  onToggleProcess?: (messageID: string) => void,
): RuntimeTimelineItem {
  const classNames = ["msg", message.role];
  if (message.error) {
    classNames.push("error");
  }
  if (message.status === "streaming") {
    classNames.push("streaming");
  }

  const footer = (
    <div className="msg-meta">
      {message.role === "assistant" && shouldShowAssistantStatus(message) ? (
        <span className={`status-pill ${message.status || "done"}`}>
          {assistantStatusLabel(message.status, language)}
        </span>
      ) : null}
      <span>{formatTimeLabel(message.at)}</span>
    </div>
  );

  if (message.role === "user") {
    return {
      id: message.id,
      className: classNames.join(" "),
      articleProps: { "data-message-id": message.id },
      bubbleClassName: "msg-bubble",
      blocks: [
        {
          type: "attachments",
          galleryId: message.id,
          items: message.attachments.map((attachment) => ({
            key: attachment.id,
            name: attachment.name,
            src: resolveComposerAttachmentPreviewURL(attachment),
          })),
        },
        ...(message.text.trim() ? [{
          type: "rich-text" as const,
          html: renderRuntimeMarkdownToHTML(message.text),
        }] : []),
      ],
      footer,
    };
  }

  const parsed = resolveAgentExecutionContent(message, language);
  if (!parsed.steps.length) {
    return {
      id: message.id,
      className: classNames.join(" "),
      articleProps: { "data-message-id": message.id },
    bubbleClassName: "msg-bubble",
    blocks: message.status === "streaming"
        ? [{ type: "rich-text", html: renderRuntimeMarkdownToHTML(message.text) }]
        : [{
          type: "markdown-shell",
          html: renderRuntimeMarkdownToHTML(message.text),
          copyValue: message.text.trim(),
          copyLabel: copy.copyValue,
        }],
      footer,
    };
  }

  const collapsed =
    typeof message.agentProcessCollapsed === "boolean"
      ? message.agentProcessCollapsed
      : Boolean(parsed.answer.trim()) && message.status !== "streaming";

  return {
    id: message.id,
    className: classNames.join(" "),
    articleProps: { "data-message-id": message.id },
    bubbleClassName: "msg-bubble",
    blocks: [
      {
        type: "process",
        shellClassName: `agent-process-shell ${collapsed ? "is-collapsed" : ""}`,
        shellProps: { "data-agent-process-shell": message.id },
        toggleClassName: "agent-process-toggle",
        toggleProps: { "data-agent-process-toggle": message.id },
        title: (
          <>
            <span className="agent-process-toggle-icon">{collapsed ? ">" : "v"}</span>
            <span className="agent-process-copy">
              <span className="agent-process-title">{copy.processLabel}</span>
              <span className="agent-process-summary">{copy.processSteps(parsed.steps.length)}</span>
            </span>
          </>
        ),
        expanded: !collapsed,
        onToggle: () => onToggleProcess?.(message.id),
        bodyClassName: "agent-process-body",
        emptyState: <div className="agent-process-empty">{copy.processEmpty}</div>,
        steps: parsed.steps.map((step, index) => ({
          id: step.id || `${step.title}-${index}`,
          itemClassName: "agent-process-step",
          toggleable: false,
          title: step.title || `${copy.processLabel} ${index + 1}`,
          meta: <span className="agent-process-step-index">{index + 1}</span>,
          expanded: true,
          onToggle: () => undefined,
          toggleClassName: "agent-process-step-head",
          bodyClassName: "agent-process-step-body",
          detail: step.detail ? <RuntimeMarkdownHTML html={renderRuntimeMarkdownToHTML(step.detail)} /> : null,
        })),
      },
      ...(parsed.answer.trim() ? [{
        type: "markdown-shell" as const,
        html: renderRuntimeMarkdownToHTML(parsed.answer),
        copyValue: parsed.answer,
        copyLabel: copy.copyValue,
        className: "agent-process-answer-shell",
        bodyClassName: "agent-process-answer",
      }] : []),
    ],
    footer,
  };
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
