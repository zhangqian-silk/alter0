import { memo, useMemo } from "react";
import { renderMessageMarkdownToHTML } from "./MessageMarkdown";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

export const MessageMarkdownHTML = memo(function MessageMarkdownHTML({ html, className }: { html: string; className?: string }) {
  const innerHTML = useMemo(() => ({ __html: html }), [html]);
  return (
    <div
      className={joinClassNames("message-markdown-rendered", className)}
      dangerouslySetInnerHTML={innerHTML}
    />
  );
});

export const MessageMarkdownShell = memo(function MessageMarkdownShell({
  markdown,
  className,
  bodyClassName,
}: {
  markdown: string;
  className?: string;
  bodyClassName?: string;
}) {
  const html = useMemo(() => renderMessageMarkdownToHTML(markdown), [markdown]);
  if (!html.trim()) {
    return null;
  }

  return (
    <div className={joinClassNames("message-markdown-shell", className)}>
      <div className={joinClassNames("message-markdown-body", bodyClassName)}>
        <MessageMarkdownHTML html={html} />
      </div>
    </div>
  );
});
