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
