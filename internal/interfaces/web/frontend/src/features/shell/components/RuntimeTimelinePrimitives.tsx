function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

export type RuntimeAttachmentGalleryItem = {
  key: string;
  name: string;
  src: string;
  previewLabel?: string;
  onPreview?: () => void;
};

export function RuntimeAttachmentGallery({
  galleryId,
  items,
  className,
}: {
  galleryId?: string;
  items: RuntimeAttachmentGalleryItem[];
  className?: string;
}) {
  if (!items.length) {
    return null;
  }

  return (
    <div
      className={joinClassNames("runtime-attachment-gallery", "message-image-grid", className)}
      data-runtime-attachment-gallery={galleryId}
    >
      {items.map((item) => (
        <figure key={item.key} className="runtime-attachment-card message-image-card">
          {item.onPreview ? (
            <button
              type="button"
              className="runtime-attachment-preview"
              aria-label={item.previewLabel || item.name}
              onClick={item.onPreview}
            >
              <img src={item.src} alt={item.name} loading="lazy" decoding="async" />
            </button>
          ) : (
            <img src={item.src} alt={item.name} loading="lazy" decoding="async" />
          )}
          <figcaption>{item.name}</figcaption>
        </figure>
      ))}
    </div>
  );
}

export function RuntimeMarkdownHTML({ html, className }: { html: string; className?: string }) {
  return (
    <div
      className={joinClassNames("runtime-markdown-rendered", className)}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

export function RuntimeMarkdownShell({
  html,
  copyValue,
  copyLabel,
  className,
  toolbarClassName,
  copyButtonClassName,
  bodyClassName,
}: {
  html: string;
  copyValue?: string;
  copyLabel?: string;
  className?: string;
  toolbarClassName?: string;
  copyButtonClassName?: string;
  bodyClassName?: string;
}) {
  if (!html.trim()) {
    return null;
  }

  return (
    <div className={joinClassNames("runtime-markdown-shell", className)}>
      <div className={joinClassNames("runtime-markdown-toolbar", toolbarClassName)}>
        {copyValue?.trim() ? (
          <button
            className={joinClassNames(
              "runtime-markdown-copy",
              "route-field-copy",
              copyButtonClassName,
            )}
            type="button"
            data-copy-value={copyValue}
            title={copyLabel}
            aria-label={copyLabel}
          >
            <CopyIcon />
          </button>
        ) : null}
      </div>
      <div className={joinClassNames("runtime-markdown-body", bodyClassName)}>
        <RuntimeMarkdownHTML html={html} />
      </div>
    </div>
  );
}

function CopyIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="12" height="12" rx="2"></rect>
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
    </svg>
  );
}
