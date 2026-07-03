import type { ReactElement, ReactNode } from "react";
import { MessageMarkdownHTML } from "./MessageMarkdownShell";
import { renderMessageMarkdownToHTML } from "./MessageMarkdown";
import type { RuntimeBlock, RuntimeProcessDetailBlockLike, RuntimeTraceEvent } from "./runtimeTraceEvents";

export type RuntimeProcessDetailBlock = RuntimeProcessDetailBlockLike;

type RuntimeProcessDetailBlocksProps = {
  blocks?: RuntimeProcessDetailBlock[];
  fallbackContent?: string;
  fallbackType?: string;
  emptyState?: ReactNode;
  blockKeyPrefix?: string;
};

const RUNTIME_PROCESS_NARRATIVE_BLOCK_TYPES = new Set([
  "text",
  "markdown",
  "message",
  "reasoning",
  "thinking",
  "plan",
  "log",
  "error",
  "tool_output",
]);

export function RuntimeProcessDetailBlocks({
  blocks = [],
  fallbackContent = "",
  fallbackType = "text",
  emptyState = null,
  blockKeyPrefix = "runtime-detail",
}: RuntimeProcessDetailBlocksProps) {
  const renderedBlocks = blocks
    .map((block, index) => renderRuntimeProcessDetailBlock(block, `${blockKeyPrefix}:${index}`))
    .filter((block): block is ReactElement => block !== null);
  if (renderedBlocks.length > 0) {
    return <>{renderedBlocks}</>;
  }
  const fallback = String(fallbackContent || "").trim();
  if (fallback) {
    return renderRuntimeProcessDetailBlock(
      { type: fallbackType, content: fallback },
      `${blockKeyPrefix}:fallback`,
    );
  }
  return <>{emptyState}</>;
}

export function runtimeTraceEventToProcessDetailBlocks(event: RuntimeTraceEvent): RuntimeProcessDetailBlock[] {
  return event.blocks
    .map(runtimeBlockToProcessDetailBlock)
    .filter((block): block is RuntimeProcessDetailBlock => block !== null);
}

export function isRuntimeProcessNarrativeBlockType(blockType: string) {
  return RUNTIME_PROCESS_NARRATIVE_BLOCK_TYPES.has(normalizeBlockType(blockType));
}

function runtimeBlockToProcessDetailBlock(block: RuntimeBlock): RuntimeProcessDetailBlock | null {
  switch (block.type) {
    case "chatRuntime":
      return {
        type: "chatRuntime",
        title: block.title || "Shell",
        content: [block.command, block.output].filter(Boolean).join("\n\n"),
        language: block.language,
        exit_code: typeof block.exit_code === "number" ? block.exit_code : block.exit_code ?? null,
      };
    case "code":
    case "diff":
      return {
        type: block.type,
        title: block.title,
        content: block.content,
        language: block.language,
        file: block.file,
        start_line: block.start_line,
      };
    case "text":
    case "markdown":
      return { type: block.type, title: block.title, content: block.text };
    case "thinking":
      return { type: "reasoning", title: block.title, content: block.text };
    case "tool_output":
      if (typeof block.text === "string" && block.text.trim()) {
        return {
          type: block.is_error ? "log" : "tool_output",
          title: block.is_error ? "Error" : "Tool output",
          content: block.text,
        };
      }
      if (block.json !== undefined) {
        return {
          type: "code",
          title: "Tool output",
          content: JSON.stringify(block.json, null, 2),
          language: "json",
        };
      }
      return null;
    case "tool_input":
      return {
        type: "code",
        title: "Tool input",
        content: JSON.stringify(block.json, null, 2),
        language: "json",
      };
    case "error":
      return {
        type: "log",
        title: block.code ? `Error ${block.code}` : "Error",
        content: block.message,
      };
    case "attachment":
    case "image":
      return null;
    default:
      return null;
  }
}

function renderRuntimeProcessDetailBlock(block: RuntimeProcessDetailBlock, key: string) {
  const blockType = normalizeBlockType(block.type) || "text";
  const blockTitle = normalizeText(block.title);
  const blockFile = normalizeText(block.file);
  const content = typeof block.content === "string" ? block.content : "";
  if (!content.trim() && !blockTitle && !blockFile) {
    return null;
  }
  return (
    <section
      key={key}
      className={`route-surface-dark chatRuntime-rich-block type-${blockType || "text"}`}
    >
      {blockTitle || blockFile ? (
        <div className="chatRuntime-rich-head">
          <div className="chatRuntime-rich-copy">
            {blockTitle ? <strong>{blockTitle}</strong> : null}
            {blockFile ? (
              <span>
                {blockFile}
                {block.start_line ? `:${block.start_line}` : ""}
              </span>
            ) : null}
          </div>
        </div>
      ) : null}
      {isRuntimeProcessNarrativeBlockType(blockType) ? (
        <MessageMarkdownHTML
          html={renderMessageMarkdownToHTML(content)}
          className="chatRuntime-step-content chatRuntime-step-richtext"
        />
      ) : (
        <pre className={`chatRuntime-rich-pre chatRuntime-step-content${blockType === "diff" ? " chatRuntime-diff-block" : ""}`}>
          <code>{content}</code>
        </pre>
      )}
    </section>
  );
}

function normalizeBlockType(value: unknown): string {
  return normalizeText(value).toLowerCase();
}

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
