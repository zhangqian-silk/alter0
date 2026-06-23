import { memo } from "react";
import { resolveComposerAttachmentViewerURL } from "../../conversation-runtime/composerImageAttachments";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  RuntimeTimeline,
  type RuntimeTimelineItem,
} from "./RuntimeTimeline";
import { MessageMarkdownHTML } from "./MessageMarkdownShell";
import { renderMessageMarkdownToHTML } from "./MessageMarkdown";
import {
  DEFAULT_RUNTIME_EVENT_FILTER,
  runtimeTraceEventVisibleByFilter,
  type RuntimeBlock,
  type RuntimeEventFilterID,
  type RuntimeTraceEvent,
} from "./runtimeTraceEvents";

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
  processSteps: RuntimeTraceEvent[];
  processCollapsed?: boolean;
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

const TIMELINE_ITEM_CACHE_LIMIT = 384;
const timelineItemCache = new Map<string, { signature: string; item: RuntimeTimelineItem }>();
const messageSignatureCache = new WeakMap<ChatMessageSnapshot, string>();
const callbackCacheIDs = new WeakMap<Function, number>();
let nextCallbackCacheID = 1;

const MESSAGE_COPY: Record<LegacyShellLanguage, MessageCopy> = {
  en: {
    statusInProgress: "In Progress",
    statusQueued: "Queued",
    statusRunning: "Running",
    statusCanceled: "Canceled",
    statusSuccess: "Success",
    statusFailed: "Failed",
    statusDone: "Done",
    processLabel: "Thinking",
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
    processLabel: "已思考",
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
  expandedProcessSteps,
  onToggleProcessStep,
  runtimeEventFilter,
}: {
  sessionId: string;
  messages: ChatMessageSnapshot[];
  language: LegacyShellLanguage;
  onToggleProcess?: (messageID: string) => void;
  expandedProcessSteps?: Record<string, boolean>;
  onToggleProcessStep?: (messageID: string, stepID: string) => void;
  runtimeEventFilter?: RuntimeEventFilterID[];
}) {
  return (
    <RuntimeTimeline
      className="message-list"
      timelineProps={{ "data-message-session-id": sessionId }}
      items={buildChatTimelineItems({
        cacheScope: sessionId,
        messages,
        language,
        onToggleProcess,
        expandedProcessSteps,
        onToggleProcessStep,
        runtimeEventFilter,
      })}
    />
  );
});

export function buildChatTimelineItems({
  cacheScope = "default",
  messages,
  language,
  onToggleProcess,
  expandedProcessSteps,
  onToggleProcessStep,
  runtimeEventFilter,
}: BuildChatTimelineItemsOptions) {
  const callbackCacheID = resolveCallbackCacheID(onToggleProcess);
  const stepCallbackCacheID = resolveCallbackCacheID(onToggleProcessStep);
  const copy = MESSAGE_COPY[language];
  const filter = runtimeEventFilter || DEFAULT_RUNTIME_EVENT_FILTER;
  const expandedStepMap = expandedProcessSteps || {};
  return messages.map((message) => {
    const cacheKey = `${cacheScope}\u0000${language}\u0000${callbackCacheID}\u0000${stepCallbackCacheID}\u0000${message.id}`;
    const signature = [
      resolveChatTimelineItemSignature(message),
      filter.join(","),
      resolveExpandedProcessStepSignature(message.id, expandedStepMap),
    ].join("\u0000");
    const cached = timelineItemCache.get(cacheKey);
    if (cached?.signature === signature) {
      return cached.item;
    }
    const item = buildChatTimelineItem(
      message,
      language,
      copy,
      onToggleProcess,
      expandedStepMap,
      onToggleProcessStep,
      filter,
    );
    timelineItemCache.set(cacheKey, { signature, item });
    trimTimelineItemCache();
    return item;
  });
}

type BuildChatTimelineItemsOptions = {
  cacheScope?: string;
  messages: ChatMessageSnapshot[];
  language: LegacyShellLanguage;
  onToggleProcess?: (messageID: string) => void;
  expandedProcessSteps?: Record<string, boolean>;
  onToggleProcessStep?: (messageID: string, stepID: string) => void;
  runtimeEventFilter?: RuntimeEventFilterID[];
};

function resolveCallbackCacheID(callback?: Function) {
  if (!callback) {
    return "none";
  }
  const cached = callbackCacheIDs.get(callback);
  if (cached) {
    return String(cached);
  }
  const next = nextCallbackCacheID;
  nextCallbackCacheID += 1;
  callbackCacheIDs.set(callback, next);
  return String(next);
}

function trimTimelineItemCache() {
  while (timelineItemCache.size > TIMELINE_ITEM_CACHE_LIMIT) {
    const oldest = timelineItemCache.keys().next().value;
    if (!oldest) {
      return;
    }
    timelineItemCache.delete(oldest);
  }
}
function chatProcessStepKey(messageID: string, stepID: string) {
  return `${messageID}:${stepID}`;
}

function resolveExpandedProcessStepSignature(messageID: string, expandedProcessSteps: Record<string, boolean>) {
  const prefix = `${messageID}:`;
  return Object.keys(expandedProcessSteps)
    .filter((key) => key.startsWith(prefix) && expandedProcessSteps[key])
    .sort()
    .join(",");
}

function resolveChatTimelineItemSignature(message: ChatMessageSnapshot) {
  const cached = messageSignatureCache.get(message);
  if (cached) {
    return cached;
  }
  const signature = buildChatTimelineItemSignature(message);
  messageSignatureCache.set(message, signature);
  return signature;
}

function buildChatTimelineItemSignature(message: ChatMessageSnapshot) {
  return JSON.stringify({
    role: message.role,
    text: message.text,
    attachments: message.attachments.map((attachment) => ({
      id: attachment.id,
      name: attachment.name,
      contentType: attachment.contentType,
      size: attachment.size,
      dataURL: attachment.dataURL,
      previewDataURL: attachment.previewDataURL,
      assetURL: attachment.assetURL,
      previewURL: attachment.previewURL,
    })),
    route: message.route,
    source: message.source,
    error: message.error,
    status: message.status,
    processSteps: message.processSteps,
    processCollapsed: message.processCollapsed,
  });
}

function buildChatTimelineItem(
  message: ChatMessageSnapshot,
  language: LegacyShellLanguage,
  copy: MessageCopy,
  onToggleProcess?: (messageID: string) => void,
  expandedProcessSteps: Record<string, boolean> = {},
  onToggleProcessStep?: (messageID: string, stepID: string) => void,
  runtimeEventFilter: RuntimeEventFilterID[] = DEFAULT_RUNTIME_EVENT_FILTER,
): RuntimeTimelineItem {
  const footer = message.role === "assistant" && shouldShowAssistantStatus(message) ? (
    <div className="msg-meta">
        <span className={`status-pill ${message.status || "done"}`}>
          {assistantStatusLabel(message.status, language)}
        </span>
    </div>
  ) : undefined;

  if (message.role === "user") {
    const blocks: RuntimeTimelineItem["blocks"] = [
      {
        type: "attachments",
        galleryId: message.id,
        className: "terminal-turn-attachments",
        items: message.attachments.map((attachment) => ({
          key: attachment.id,
          name: attachment.name,
          src: resolveComposerAttachmentViewerURL(attachment),
        })),
      },
    ];
    if (message.text.trim()) {
      blocks.push({
        type: "prompt",
        className: "terminal-log-row kind-command terminal-turn-prompt conversation-turn-prompt",
        textClassName: "terminal-log-main",
        timeClassName: "terminal-log-time",
        text: message.text,
      });
    }
    return {
      id: message.id,
      className: "msg user terminal-turn-card conversation-turn-card runtime-message runtime-message-user conversation-message conversation-turn-user is-user",
      articleProps: { "data-message-id": message.id },
      bubbleClassName: "msg-bubble runtime-message-bubble runtime-message-user-shell user-message-shell",
      blocks,
    };
  }

  const parsed = resolveExecutionContent(message, language, runtimeEventFilter);
  if (!parsed.steps.length) {
    const markdown = parsed.hadProcess ? parsed.answer.trim() : (parsed.answer.trim() || message.text);
    return {
      id: message.id,
      className: "msg assistant terminal-turn-card conversation-turn-card runtime-message runtime-message-assistant conversation-message conversation-turn-assistant is-assistant",
      articleProps: { "data-message-id": message.id },
      bubbleClassName: "msg-bubble runtime-message-bubble runtime-message-assistant-shell assistant-message-shell",
      blocks: markdown.trim() ? [
        {
          type: "markdown-shell" as const,
          markdown,
          copyValue: message.status === "streaming" ? undefined : markdown.trim(),
          copyLabel: copy.copyValue,
          wrapperClassName: [
            "terminal-final-output",
            "conversation-final-output",
            message.status === "streaming" ? "is-streaming" : "",
            message.error ? "is-error" : "",
          ].filter(Boolean).join(" "),
          wrapperProps: { "data-conversation-final-output": message.id },
          className: "terminal-final-text conversation-final-text",
          bodyClassName: "terminal-final-rendered conversation-final-rendered",
        },
      ] : [],
      footer,
    };
  }

  const collapsed =
    typeof message.processCollapsed === "boolean"
      ? message.processCollapsed
      : Boolean(parsed.answer.trim()) && message.status !== "streaming";

  return {
    id: message.id,
    className: "msg assistant terminal-turn-card conversation-turn-card runtime-message runtime-message-assistant conversation-message conversation-turn-assistant is-assistant",
    articleProps: { "data-message-id": message.id },
    bubbleClassName: "msg-bubble runtime-message-bubble runtime-message-assistant-shell assistant-message-shell",
    blocks: [
      {
        type: "process",
        shellClassName: `runtime-thinking-shell terminal-process-shell ${collapsed ? "is-collapsed" : ""}`,
        shellProps: { "data-conversation-process-shell": message.id },
        toggleClassName: "runtime-thinking-toggle terminal-process-toggle",
        toggleProps: { "data-conversation-process-toggle": message.id },
        title: (
          <>
            <span className="terminal-step-toggle-icon" aria-hidden="true">{collapsed ? ">" : "v"}</span>
            <span className="terminal-process-copy">
              <span className="terminal-process-title">{copy.processLabel}</span>
              <span className="terminal-process-summary">{copy.processSteps(parsed.steps.length)}</span>
            </span>
          </>
        ),
        expanded: !collapsed,
        onToggle: () => onToggleProcess?.(message.id),
        bodyClassName: "terminal-process-body",
        emptyState: <div className="terminal-process-empty">{copy.processEmpty}</div>,
        steps: parsed.steps.map((step, index) => {
          const stepID = step.id || `${step.title}-${index}`;
          const expanded = Boolean(expandedProcessSteps[chatProcessStepKey(message.id, stepID)]);
          return {
            id: stepID,
            itemClassName: "terminal-step-item",
            itemProps: {
              "data-terminal-step-item": stepID,
              "data-conversation-process-step": stepID,
              ...(isRuntimeTraceEvent(step) ? {
                "data-runtime-event-kind": step.kind,
                "data-runtime-event-source": step.source,
              } : {}),
            },
            title: step.title || `${copy.processLabel} ${index + 1}`,
            titleClassName: "terminal-step-title",
            expanded,
            onToggle: () => onToggleProcessStep?.(message.id, stepID),
            toggleClassName: "terminal-step-toggle",
            toggleProps: {
              "data-terminal-step-toggle": stepID,
              "data-conversation-process-step-toggle": stepID,
            },
            bodyClassName: "terminal-step-body",
            detail: (
              <div className="terminal-step-detail">
                {isRuntimeTraceEvent(step)
                  ? runtimeEventDetail(step)
                  : step.detail
                    ? <MessageMarkdownHTML html={renderMessageMarkdownToHTML(step.detail)} />
                    : null}
              </div>
            ),
          };
        }),
      },
      ...(parsed.answer.trim() ? [
        {
          type: "markdown-shell" as const,
          markdown: parsed.answer,
          copyValue: parsed.answer,
          copyLabel: copy.copyValue,
          wrapperClassName: "terminal-final-output conversation-final-output",
          wrapperProps: { "data-conversation-final-output": message.id },
          className: "terminal-final-text conversation-process-answer-shell conversation-final-text",
          bodyClassName: "terminal-final-rendered conversation-process-answer conversation-final-rendered",
        },
      ] : []),
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
  return false;
}

function resolveExecutionContent(
  message: ChatMessageSnapshot,
  language: LegacyShellLanguage,
  runtimeEventFilter: RuntimeEventFilterID[] = DEFAULT_RUNTIME_EVENT_FILTER,
) {
  void language;
  if (message.processSteps.length) {
    const steps = message.processSteps
      .filter((event) => runtimeTraceEventVisibleByFilter(event, runtimeEventFilter));
    return {
      steps,
      answer: message.text.trim(),
      hadProcess: true,
    };
  }
  return {
    steps: [] as RuntimeTraceEvent[],
    answer: message.text.trim(),
    hadProcess: false,
  };
}

function isRuntimeTraceEvent(value: unknown): value is RuntimeTraceEvent {
  return Boolean(value && typeof value === "object" && "blocks" in value && "kind" in value);
}

function runtimeEventDetail(event: RuntimeTraceEvent) {
  const markdown = event.blocks
    .map(runtimeBlockMarkdown)
    .filter(Boolean)
    .join("\n\n")
    .trim();
  return markdown ? <MessageMarkdownHTML html={renderMessageMarkdownToHTML(markdown)} /> : null;
}

function runtimeBlockMarkdown(block: RuntimeBlock): string {
  switch (block.type) {
    case "text":
    case "markdown":
    case "thinking":
      return block.text;
    case "tool_output":
      return typeof block.text === "string" ? block.text : "";
    case "terminal":
      return [block.command, block.output].filter(Boolean).join("\n\n");
    case "code":
    case "diff":
      return block.content;
    case "error":
      return block.message;
    default:
      return "";
  }
}
