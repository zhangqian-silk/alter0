import { useEffect, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTime } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { normalizeText, RouteCard, RouteFieldRow, RouteTagSection } from "./RouteBodyPrimitives";

type MaintenanceJobID = "system-memory-maintenance" | "system-session-cleanup";

type MaintenanceRunRecord = {
  job_id?: string;
  status?: string;
  started_at?: string;
  finished_at?: string;
  next_run_at?: string;
  deleted_count?: number;
  skipped_pinned_count?: number;
  skipped_protected_count?: number;
  scanned_count?: number;
  terminal_deleted_count?: number;
  terminal_skipped_pinned_count?: number;
  terminal_skipped_protected_count?: number;
  terminal_scanned_count?: number;
  changed_files?: string[];
  error?: string;
};

type MaintenanceStatusResponse = {
  items?: MaintenanceRunRecord[];
};

type MaintenanceRouteCopy = {
  loading: string;
  empty: string;
  loadFailed: (message: string) => string;
  actionFailed: (message: string) => string;
  memoryTitle: string;
  sessionTitle: string;
  runMemory: string;
  cleanupSessions: string;
  retry: string;
  running: string;
  statusIdle: string;
  statusSuccess: string;
  statusFailed: string;
  statusRunning: string;
  fieldStatus: string;
  fieldLastRun: string;
  fieldNextRun: string;
  fieldDeleted: string;
  fieldSkippedPinned: string;
  fieldSkippedProtected: string;
  fieldScanned: string;
  fieldTerminalCleanup: string;
  fieldChangedFiles: string;
  fieldError: string;
  copyValue: string;
  cleanupSummary: (deleted: number, pinned: number, protectedCount: number, scanned: number) => string;
  terminalCleanupSummary: (deleted: number, pinned: number, protectedCount: number, scanned: number) => string;
};

const MAINTENANCE_COPY: Record<LegacyShellLanguage, MaintenanceRouteCopy> = {
  en: {
    loading: "Loading maintenance...",
    empty: "No maintenance jobs available.",
    loadFailed: (message) => `Load failed: ${message}`,
    actionFailed: (message) => `Maintenance failed: ${message}`,
    memoryTitle: "Memory Maintenance",
    sessionTitle: "Session Cleanup",
    runMemory: "Run maintenance",
    cleanupSessions: "Clean up now",
    retry: "Retry",
    running: "Running...",
    statusIdle: "Idle",
    statusSuccess: "Success",
    statusFailed: "Failed",
    statusRunning: "Running",
    fieldStatus: "Status",
    fieldLastRun: "Last Run",
    fieldNextRun: "Next Run",
    fieldDeleted: "Deleted",
    fieldSkippedPinned: "Pinned Skipped",
    fieldSkippedProtected: "Task Protected",
    fieldScanned: "Scanned",
    fieldTerminalCleanup: "Terminal Cleanup",
    fieldChangedFiles: "Changed Files",
    fieldError: "Error",
    copyValue: "Copy value",
    cleanupSummary: (deleted, pinned, protectedCount, scanned) => `Deleted ${deleted} · pinned ${pinned} · protected ${protectedCount} · scanned ${scanned}`,
    terminalCleanupSummary: (deleted, pinned, protectedCount, scanned) => `Terminal deleted ${deleted} · pinned ${pinned} · active ${protectedCount} · scanned ${scanned}`,
  },
  zh: {
    loading: "加载维护状态...",
    empty: "暂无维护任务。",
    loadFailed: (message) => `加载失败：${message}`,
    actionFailed: (message) => `维护执行失败：${message}`,
    memoryTitle: "记忆维护",
    sessionTitle: "会话清理",
    runMemory: "执行维护",
    cleanupSessions: "立即清理",
    retry: "重试",
    running: "执行中...",
    statusIdle: "空闲",
    statusSuccess: "成功",
    statusFailed: "失败",
    statusRunning: "执行中",
    fieldStatus: "状态",
    fieldLastRun: "上次执行",
    fieldNextRun: "下次执行",
    fieldDeleted: "已删除",
    fieldSkippedPinned: "置顶跳过",
    fieldSkippedProtected: "任务保护",
    fieldScanned: "扫描数",
    fieldTerminalCleanup: "Terminal 清理",
    fieldChangedFiles: "变更文件",
    fieldError: "错误",
    copyValue: "复制内容",
    cleanupSummary: (deleted, pinned, protectedCount, scanned) => `删除 ${deleted} · 置顶 ${pinned} · 保护 ${protectedCount} · 扫描 ${scanned}`,
    terminalCleanupSummary: (deleted, pinned, protectedCount, scanned) => `Terminal 删除 ${deleted} · 置顶 ${pinned} · 活跃 ${protectedCount} · 扫描 ${scanned}`,
  },
};

type RequestState =
  | { status: "loading"; items: MaintenanceRunRecord[]; error: string }
  | { status: "ready"; items: MaintenanceRunRecord[]; error: string }
  | { status: "error"; items: MaintenanceRunRecord[]; error: string };

export function ReactManagedMaintenanceRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = MAINTENANCE_COPY[language];
  const [state, setState] = useState<RequestState>({ status: "loading", items: [], error: "" });
  const [runningJobID, setRunningJobID] = useState("");
  const [actionError, setActionError] = useState("");

  useEffect(() => {
    let disposed = false;
    setState({ status: "loading", items: [], error: "" });
    void createAPIClient()
      .get<MaintenanceStatusResponse>("/api/maintenance")
      .then((payload) => {
        if (disposed) {
          return;
        }
        setState({ status: "ready", items: normalizeMaintenanceItems(payload?.items), error: "" });
      })
      .catch((error: unknown) => {
        if (disposed) {
          return;
        }
        setState({ status: "error", items: [], error: error instanceof Error ? error.message : "unknown_error" });
      });
    return () => {
      disposed = true;
    };
  }, []);

  function runJob(jobID: MaintenanceJobID) {
    setActionError("");
    setRunningJobID(jobID);
    const path =
      jobID === "system-memory-maintenance"
        ? "/api/maintenance/memory/run"
        : "/api/maintenance/sessions/cleanup";
    void createAPIClient()
      .post<MaintenanceRunRecord>(path)
      .then((record) => {
        setState((current) => ({
          ...current,
          status: current.status === "error" ? "ready" : current.status,
          items: replaceMaintenanceItem(current.items, record),
          error: "",
        }));
      })
      .catch((error: unknown) => {
        setActionError(error instanceof Error ? error.message : "unknown_error");
      })
      .finally(() => setRunningJobID(""));
  }

  if (state.status === "loading") {
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (state.status === "error") {
    return <p className="route-error">{copy.loadFailed(state.error)}</p>;
  }

  if (!state.items.length) {
    return <p className="route-empty">{copy.empty}</p>;
  }

  return (
    <section className="maintenance-route-grid" data-maintenance-route-grid="true">
      {actionError ? <p className="route-error">{copy.actionFailed(actionError)}</p> : null}
      {state.items.map((item) => (
        <MaintenanceRouteCard
          key={normalizeMaintenanceJobID(item.job_id)}
          item={item}
          copy={copy}
          running={runningJobID === normalizeMaintenanceJobID(item.job_id)}
          onRun={runJob}
        />
      ))}
    </section>
  );
}

function MaintenanceRouteCard({
  item,
  copy,
  running,
  onRun,
}: {
  item: MaintenanceRunRecord;
  copy: MaintenanceRouteCopy;
  running: boolean;
  onRun: (jobID: MaintenanceJobID) => void;
}) {
  const jobID = normalizeMaintenanceJobID(item.job_id);
  const status = normalizeText(item.status || "idle").toLowerCase();
  const failed = status === "failed";
  const changedFiles = Array.isArray(item.changed_files) ? item.changed_files.filter(Boolean) : [];
  const deletedCount = Number(item.deleted_count || 0);
  const skippedPinnedCount = Number(item.skipped_pinned_count || 0);
  const skippedProtectedCount = Number(item.skipped_protected_count || 0);
  const scannedCount = Number(item.scanned_count || 0);
  const terminalDeletedCount = Number(item.terminal_deleted_count || 0);
  const terminalSkippedPinnedCount = Number(item.terminal_skipped_pinned_count || 0);
  const terminalSkippedProtectedCount = Number(item.terminal_skipped_protected_count || 0);
  const terminalScannedCount = Number(item.terminal_scanned_count || 0);
  const hasTerminalCleanup =
    terminalDeletedCount > 0 ||
    terminalSkippedPinnedCount > 0 ||
    terminalSkippedProtectedCount > 0 ||
    terminalScannedCount > 0;
  const actionLabel = running
    ? copy.running
    : failed
      ? copy.retry
      : jobID === "system-memory-maintenance"
        ? copy.runMemory
        : copy.cleanupSessions;

  return (
    <RouteCard
      title={jobID === "system-memory-maintenance" ? copy.memoryTitle : copy.sessionTitle}
      type="maintenance"
      enabled={!failed}
      statusEnabledLabel={formatMaintenanceStatus(status, copy)}
      statusDisabledLabel={formatMaintenanceStatus(status, copy)}
      className="maintenance-route-card"
      actions={
        <button
          className="route-card-action"
          type="button"
          disabled={running}
          onClick={() => onRun(jobID)}
        >
          {actionLabel}
        </button>
      }
      footer={
        changedFiles.length ? (
          <RouteTagSection label={copy.fieldChangedFiles} tags={changedFiles} />
        ) : null
      }
    >
      <RouteFieldRow label={copy.fieldStatus} value={formatMaintenanceStatus(status, copy)} copyLabel={copy.copyValue} />
      <RouteFieldRow label={copy.fieldLastRun} value={formatDateTime(item.finished_at || item.started_at)} copyLabel={copy.copyValue} />
      <RouteFieldRow label={copy.fieldNextRun} value={formatDateTime(item.next_run_at)} copyLabel={copy.copyValue} />
      {jobID === "system-session-cleanup" ? (
        <>
          <RouteFieldRow label={copy.fieldDeleted} value={deletedCount} copyLabel={copy.copyValue} />
          <RouteFieldRow label={copy.fieldSkippedPinned} value={skippedPinnedCount} copyLabel={copy.copyValue} />
          <RouteFieldRow label={copy.fieldSkippedProtected} value={skippedProtectedCount} copyLabel={copy.copyValue} />
          <RouteFieldRow label={copy.fieldScanned} value={scannedCount} copyLabel={copy.copyValue} />
          <RouteFieldRow label={copy.fieldChangedFiles} value={copy.cleanupSummary(deletedCount, skippedPinnedCount, skippedProtectedCount, scannedCount)} copyLabel={copy.copyValue} />
          {hasTerminalCleanup ? (
            <RouteFieldRow
              label={copy.fieldTerminalCleanup}
              value={copy.terminalCleanupSummary(terminalDeletedCount, terminalSkippedPinnedCount, terminalSkippedProtectedCount, terminalScannedCount)}
              copyLabel={copy.copyValue}
            />
          ) : null}
        </>
      ) : null}
      {item.error ? <RouteFieldRow label={copy.fieldError} value={item.error} copyLabel={copy.copyValue} multiline /> : null}
    </RouteCard>
  );
}

function normalizeMaintenanceItems(items: MaintenanceRunRecord[] | undefined) {
  return Array.isArray(items) ? items : [];
}

function replaceMaintenanceItem(items: MaintenanceRunRecord[], next: MaintenanceRunRecord) {
  const nextJobID = normalizeMaintenanceJobID(next.job_id);
  const replaced = items.map((item) =>
    normalizeMaintenanceJobID(item.job_id) === nextJobID ? next : item,
  );
  if (replaced.some((item) => normalizeMaintenanceJobID(item.job_id) === nextJobID)) {
    return replaced;
  }
  return [...replaced, next];
}

function normalizeMaintenanceJobID(value: unknown): MaintenanceJobID {
  return normalizeText(value) === "system-memory-maintenance"
    ? "system-memory-maintenance"
    : "system-session-cleanup";
}

function formatMaintenanceStatus(value: string, copy: MaintenanceRouteCopy) {
  switch (value) {
    case "success":
      return copy.statusSuccess;
    case "failed":
      return copy.statusFailed;
    case "running":
      return copy.statusRunning;
    default:
      return copy.statusIdle;
  }
}
