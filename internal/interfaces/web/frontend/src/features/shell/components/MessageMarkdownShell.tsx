import { memo, useMemo } from "react";
import { renderMessageMarkdownToHTML } from "./MessageMarkdown";
import { CopyValueButton } from "./RouteBodyPrimitives";

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
  copyValue,
  copyLabel,
  className,
  toolbarClassName,
  copyButtonClassName,
  bodyClassName,
}: {
  markdown: string;
  copyValue?: string;
  copyLabel?: string;
  className?: string;
  toolbarClassName?: string;
  copyButtonClassName?: string;
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
      <div className={joinClassNames("message-markdown-toolbar", toolbarClassName)}>
        {copyValue?.trim() ? (
          <CopyValueButton
            className={joinClassNames(
              "message-markdown-copy",
              "route-field-copy",
              copyButtonClassName,
            )}
            value={copyValue}
            label={copyLabel}
          />
        ) : null}
      </div>
    </div>
  );
});
