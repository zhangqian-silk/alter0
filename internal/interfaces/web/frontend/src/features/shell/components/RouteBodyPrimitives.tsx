import type { CSSProperties, ReactNode } from "react";

type RouteCardProps = {
  title: unknown;
  type: unknown;
  enabled: boolean;
  statusEnabledLabel: string;
  statusDisabledLabel: string;
  className?: string;
  children?: ReactNode;
  body?: ReactNode;
  footer?: ReactNode;
  footerClassName?: string;
};

type RouteFieldRowProps = {
  label: string;
  value: unknown;
  copyLabel: string;
  copyable?: boolean;
  mono?: boolean;
  multiline?: boolean;
  preview?: boolean;
  clampLines?: number;
};

export function RouteCard({
  title,
  type,
  enabled,
  statusEnabledLabel,
  statusDisabledLabel,
  className = "",
  children,
  body,
  footer,
  footerClassName = "",
}: RouteCardProps) {
  const classNames = ["route-card"];
  if (className) {
    classNames.push(className);
  }
  const footerClasses = ["route-card-footer"];
  if (footerClassName) {
    footerClasses.push(footerClassName);
  }

  return (
    <article className={classNames.join(" ")}>
      <div className="route-card-head">
        <div className="route-card-title-wrap">
          <div className="route-card-icon" aria-hidden="true">
            <RouteTypeIcon type={type} />
          </div>
          <div className="route-card-title-copy">
            <h4 title={normalizeText(title)}>{normalizeText(title)}</h4>
          </div>
        </div>
        <div className={enabled ? "status-badge" : "status-badge disabled"}>
          <span className="status-dot"></span>
          <span>{enabled ? statusEnabledLabel : statusDisabledLabel}</span>
        </div>
      </div>
      {children ? <div className="route-meta">{children}</div> : null}
      {body ? <div className="route-card-body">{body}</div> : null}
      {footer ? <footer className={footerClasses.join(" ")}>{footer}</footer> : null}
    </article>
  );
}

export function RouteFieldRow({
  label,
  value,
  copyLabel,
  copyable = false,
  mono = false,
  multiline = false,
  preview = false,
  clampLines = 0,
}: RouteFieldRowProps) {
  const safeValue = normalizeText(value);
  const classNames = ["route-field-value"];

  if (multiline) {
    classNames.push("is-multiline");
  }
  if (mono) {
    classNames.push("is-mono");
  }
  if (preview || clampLines > 0) {
    classNames.push("is-preview");
  }

  const style =
    clampLines > 0
      ? ({ "--line-clamp": String(Math.max(2, clampLines)) } as CSSProperties)
      : undefined;

  return (
    <p className="route-field-row">
      <span>{label}</span>
      <span className="route-field-value-wrap">
        <strong className={classNames.join(" ")} title={safeValue} style={style}>
          {safeValue}
        </strong>
        {copyable && safeValue !== "-" ? (
          <button
            className="route-field-copy"
            type="button"
            data-copy-value={safeValue}
            title={copyLabel}
            aria-label={copyLabel}
          >
            <CopyIcon />
          </button>
        ) : null}
      </span>
    </p>
  );
}

export function RouteTagSection({
  label,
  tags,
}: {
  label: string;
  tags: string[];
}) {
  const safeTags = tags.map((tag) => normalizeText(tag)).filter((tag) => tag !== "-");

  return (
    <div className="route-card-tag-section">
      <span>{label}</span>
      {safeTags.length ? (
        <div className="route-tag-list">
          {safeTags.map((tag) => (
            <span key={tag} className="route-tag">
              {tag}
            </span>
          ))}
        </div>
      ) : (
        <span className="route-tag-placeholder">-</span>
      )}
    </div>
  );
}

export function normalizeText(value: unknown) {
  const text = String(value || "").trim();
  return text || "-";
}

function RouteTypeIcon({ type }: { type: unknown }) {
  const normalized = String(type || "").toLowerCase();

  if (normalized.includes("http") || normalized.includes("web")) {
    return (
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="9"></circle>
        <path d="M3 12h18"></path>
        <path d="M12 3a15 15 0 0 1 0 18"></path>
        <path d="M12 3a15 15 0 0 0 0 18"></path>
      </svg>
    );
  }

  if (normalized.includes("mcp") || normalized.includes("proto")) {
    return (
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 3 4 7.5v9L12 21l8-4.5v-9L12 3z"></path>
        <path d="m4 7.5 8 4.5 8-4.5"></path>
        <path d="M12 12v9"></path>
      </svg>
    );
  }

  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 17h16"></path>
      <path d="M7 17V7h10v10"></path>
      <path d="m9.5 10.5 1.8 1.8-1.8 1.8"></path>
      <path d="M13.2 14.1h2.3"></path>
    </svg>
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
