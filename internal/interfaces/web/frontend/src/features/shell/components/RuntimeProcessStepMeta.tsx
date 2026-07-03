import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  runtimeTraceEventDisclosureCategory,
  runtimeTraceEventDisclosureLabel,
  type RuntimeTraceEvent,
} from "./runtimeTraceEvents";

const STATUS_COPY: Record<LegacyShellLanguage, {
  ready: string;
  busy: string;
  failed: string;
  interrupted: string;
}> = {
  en: {
    ready: "Ready",
    busy: "Busy",
    failed: "Failed",
    interrupted: "Interrupted",
  },
  zh: {
    ready: "就绪",
    busy: "执行中",
    failed: "失败",
    interrupted: "已中断",
  },
};

function normalizeStatusText(status: string | undefined): string {
  return String(status || "").trim().toLowerCase();
}

export function runtimeProcessDurationLabel(durationMS: number | undefined) {
  const value = Number(durationMS || 0);
  if (!Number.isFinite(value) || value <= 0) {
    return "<1s";
  }
  const seconds = Math.max(1, Math.round(value / 1000));
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remain = seconds % 60;
  return remain > 0 ? `${minutes}m ${remain}s` : `${minutes}m`;
}

export function runtimeProcessStepStatusClassName(status: string | undefined) {
  const normalized = normalizeStatusText(status);
  if (["busy", "running", "starting", "queued", "requires_approval"].includes(normalized)) {
    return "status-running";
  }
  if (["failed", "error"].includes(normalized)) {
    return "status-failed";
  }
  if (["interrupted", "cancelled", "canceled", "exited"].includes(normalized)) {
    return "status-interrupted";
  }
  if (["completed", "success", "ready", "done"].includes(normalized) || !normalized || normalized === "-") {
    return "status-success";
  }
  return "status-neutral";
}

export function runtimeProcessStepStatusLabel(status: string | undefined, language: LegacyShellLanguage) {
  const normalized = normalizeStatusText(status);
  const copy = STATUS_COPY[language];
  if (["busy", "running", "starting", "queued", "requires_approval"].includes(normalized)) {
    return copy.busy;
  }
  if (["failed", "error"].includes(normalized)) {
    return copy.failed;
  }
  if (["interrupted", "cancelled", "canceled", "exited"].includes(normalized)) {
    return copy.interrupted;
  }
  return copy.ready;
}

export function RuntimeProcessStepMeta({
  event,
  language,
  durationMS,
  status,
}: {
  event: RuntimeTraceEvent;
  language: LegacyShellLanguage;
  durationMS?: number;
  status?: string;
}) {
  const category = runtimeTraceEventDisclosureCategory(event);
  const resolvedDurationMS = typeof durationMS === "number" ? durationMS : event.duration_ms;
  const resolvedStatus = status || event.status;
  return (
    <span className="chatRuntime-step-meta">
      <span className={`chatRuntime-step-kind kind-${category}`} data-runtime-event-category={category}>
        {runtimeTraceEventDisclosureLabel(event, language)}
      </span>
      <span className="chatRuntime-step-duration">
        {runtimeProcessDurationLabel(resolvedDurationMS)}
      </span>
      <span className={`chatRuntime-step-status ${runtimeProcessStepStatusClassName(resolvedStatus)}`}>
        {runtimeProcessStepStatusLabel(resolvedStatus, language)}
      </span>
    </span>
  );
}
