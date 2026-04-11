export const FRONTEND_DISPLAY_TIME_ZONE = "Asia/Shanghai";

type TimeInput = string | number | Date | null | undefined;
type DisplayParts = Partial<Record<Intl.DateTimeFormatPartTypes, string>>;

export function resolveDisplayTimeParts(value: TimeInput, fields: Intl.DateTimeFormatOptions = {}): DisplayParts | null {
  const parsed = toValidDate(value);
  if (!parsed) {
    return null;
  }

  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: FRONTEND_DISPLAY_TIME_ZONE,
    hour12: false,
    hourCycle: "h23",
    ...fields
  });
  const parts: DisplayParts = {};
  for (const part of formatter.formatToParts(parsed)) {
    if (part.type !== "literal") {
      parts[part.type] = part.value;
    }
  }
  return parts;
}

export function formatTimeLabel(value: TimeInput = Date.now()): string {
  const parts = resolveDisplayTimeParts(value, {
    hour: "2-digit",
    minute: "2-digit"
  });
  if (!parts?.hour || !parts.minute) {
    return "-";
  }
  return `${parts.hour}:${parts.minute}`;
}

export function formatDateTime(value: TimeInput): string {
  const text = typeof value === "string" ? value.trim() : "";
  if (typeof value === "string" && !text) {
    return "-";
  }

  const parts = resolveDisplayTimeParts(value, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  });
  if (!parts) {
    return text || "-";
  }
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second}`;
}

function toValidDate(value: TimeInput): Date | null {
  if (value == null || value === "") {
    return null;
  }
  const parsed = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }
  return parsed;
}
