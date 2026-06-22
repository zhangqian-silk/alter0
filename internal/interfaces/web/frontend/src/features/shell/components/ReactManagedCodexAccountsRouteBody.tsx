import { useEffect, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTimeMinute } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";

type RuntimeStatus = {
  command?: string;
  auth_path?: string;
  config_path?: string;
  has_auth?: boolean;
  has_config?: boolean;
  profile?: string;
  model?: string;
  reasoning_effort?: string;
  model_origin?: RuntimeConfigOrigin | null;
  reasoning_origin?: RuntimeConfigOrigin | null;
  models?: RuntimeModel[];
  current?: RuntimeCurrentStatus | null;
};

type RuntimeCurrentStatus = {
  live?: RuntimeAuthSnapshot | null;
  managed?: RuntimeManagedRecord | null;
  auth_path?: string;
  quota?: RuntimeQuotaStatus | null;
  error?: string;
};

type RuntimeManagedRecord = {
  name?: string;
  snapshot?: RuntimeAuthSnapshot | null;
};

type RuntimeAuthSnapshot = {
  auth_mode?: string;
  account_name?: string;
  email?: string;
  user_id?: string;
  account_id?: string;
  plan?: string;
  last_refresh_at?: string;
};

type RuntimeQuotaStatus = {
  hourly?: RuntimeQuotaWindow | null;
  weekly?: RuntimeQuotaWindow | null;
  plan?: string;
};

type RuntimeQuotaWindow = {
  remaining_percent?: number;
  reset_at?: string;
};

type RuntimeConfigOrigin = {
  key_path?: string;
  file_path?: string;
  version?: string;
};

type RuntimeReasoningMode = {
  reasoning_effort?: string;
  description?: string;
};

type RuntimeModel = {
  id?: string;
  model?: string;
  display_name?: string;
  description?: string;
  hidden?: boolean;
  is_default?: boolean;
  default_reasoning_effort?: string;
  supported_reasoning_effort?: RuntimeReasoningMode[];
  input_modalities?: string[];
};

type LLMProviderRecord = {
  id?: string;
  name?: string;
  is_enabled?: boolean;
};

type LLMProviderResponse = {
  items?: LLMProviderRecord[];
};

type RequestState =
  | { status: "loading"; error: string }
  | { status: "ready"; error: string }
  | { status: "error"; error: string };

type RuntimeCopy = {
  loading: string;
  overview: string;
  overviewSubtitle: string;
  model: string;
  reasoningDepth: string;
  modelUnavailable: string;
  reasoningUnavailable: string;
  providerRegistered: (count: number) => string;
  providersMissing: string;
  providersMissingHint: string;
  providersReadyHint: string;
  activeProfile: string;
  identityName: string;
  identityPlan: string;
  identityAuthMode: string;
  quotaHourly: string;
  quotaWeekly: string;
  quotaRemaining: string;
  quotaResets: string;
  codexDefault: string;
  serviceControls: string;
  serviceControlsSubtitle: string;
  restartService: string;
  restartConfirmTitle: string;
  restartConfirmBody: string;
  updateBeforeRestart: string;
  updateBeforeRestartHint: string;
  cancel: string;
  confirmRestart: string;
  discardConfirmTitle: string;
  discardConfirmBody: string;
  discardAndRestart: string;
  back: string;
  restarting: string;
  restartAccepted: string;
  loadFailed: (message: string) => string;
  actionFailed: (message: string) => string;
};

const RUNTIME_COPY: Record<LegacyShellLanguage, RuntimeCopy> = {
  en: {
    loading: "Loading...",
    overview: "Codex Runtime",
    overviewSubtitle: "Identity, quota, and active runtime settings.",
    model: "Model",
    reasoningDepth: "Reasoning Depth",
    modelUnavailable: "No runtime models available",
    reasoningUnavailable: "No reasoning modes available",
    providerRegistered: (count) => `${count} registered provider${count === 1 ? "" : "s"}`,
    providersMissing: "No LLM providers registered. Codex Direct remains available.",
    providersMissingHint: "Provider-based execution is disabled until a provider appears in the Models registry.",
    providersReadyHint: "Provider-based execution can be used by Chat sessions.",
    activeProfile: "Profile",
    identityName: "Account",
    identityPlan: "Plan",
    identityAuthMode: "Auth Mode",
    quotaHourly: "Hourly",
    quotaWeekly: "Weekly",
    quotaRemaining: "Remaining",
    quotaResets: "Resets",
    codexDefault: "Codex default",
    serviceControls: "Service controls",
    serviceControlsSubtitle: "Restart the running service when runtime settings or deployment state need to be reapplied.",
    restartService: "Restart service",
    restartConfirmTitle: "Restart service?",
    restartConfirmBody: "The service will restart and active browser streams may reconnect.",
    updateBeforeRestart: "Update from remote master before restarting",
    updateBeforeRestartHint: "Fetch and fast-forward when the working tree has no tracked changes, rebuild, then restart.",
    cancel: "Cancel",
    confirmRestart: "Restart",
    discardConfirmTitle: "Discard local tracked changes?",
    discardConfirmBody: "Updating from remote master will discard tracked local changes before rebuilding. Untracked files are kept.",
    discardAndRestart: "Discard and restart",
    back: "Back",
    restarting: "Restarting...",
    restartAccepted: "Restart accepted. The service will come back online shortly.",
    loadFailed: (message) => `Load failed: ${message}`,
    actionFailed: (message) => `Action failed: ${message}`,
  },
  zh: {
    loading: "加载中...",
    overview: "Codex 运行时",
    overviewSubtitle: "展示身份、额度与当前运行时设置。",
    model: "Model",
    reasoningDepth: "思考深度",
    modelUnavailable: "暂无运行时 model",
    reasoningUnavailable: "暂无思考深度选项",
    providerRegistered: (count) => `已注册 ${count} 个 Provider`,
    providersMissing: "暂无 LLM Provider 注册；Codex Direct 仍可使用。",
    providersMissingHint: "Provider 执行链需等 Models 注册表出现可用 Provider 后启用。",
    providersReadyHint: "Chat 会话可使用 Provider 执行链。",
    activeProfile: "Profile",
    identityName: "账号",
    identityPlan: "计划",
    identityAuthMode: "认证模式",
    quotaHourly: "小时额度",
    quotaWeekly: "周额度",
    quotaRemaining: "剩余",
    quotaResets: "重置",
    codexDefault: "Codex 默认值",
    serviceControls: "服务控制",
    serviceControlsSubtitle: "当运行时设置或部署状态需要重新加载时，重启当前服务。",
    restartService: "重启服务",
    restartConfirmTitle: "确认重启服务？",
    restartConfirmBody: "服务将重新启动，进行中的浏览器流式连接可能需要重新连接。",
    updateBeforeRestart: "重启前从远端 master 更新",
    updateBeforeRestartHint: "仅在没有已跟踪本地改动时拉取并快进、重新构建，然后重启。",
    cancel: "取消",
    confirmRestart: "确认重启",
    discardConfirmTitle: "丢弃本地已跟踪改动？",
    discardConfirmBody: "从远端 master 更新会先丢弃已跟踪的本地改动再重新构建；未跟踪文件会保留。",
    discardAndRestart: "丢弃并重启",
    back: "返回",
    restarting: "重启中...",
    restartAccepted: "已接受重启请求，服务稍后会重新上线。",
    loadFailed: (message) => `加载失败：${message}`,
    actionFailed: (message) => `操作失败：${message}`,
  },
};

export function ReactManagedCodexAccountsRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = RUNTIME_COPY[language];
  const apiClient = createAPIClient();
  const [requestState, setRequestState] = useState<RequestState>({ status: "loading", error: "" });
  const [runtime, setRuntime] = useState<RuntimeStatus | null>(null);
  const [providers, setProviders] = useState<LLMProviderRecord[]>([]);
  const [selectedModel, setSelectedModel] = useState("");
  const [selectedReasoning, setSelectedReasoning] = useState("");
  const [statusMessage, setStatusMessage] = useState("");
  const [statusKind, setStatusKind] = useState<"success" | "error" | "">("");
  const [restartDialog, setRestartDialog] = useState<{ open: boolean; syncRemoteMaster: boolean; confirmDiscard: boolean }>({
    open: false,
    syncRemoteMaster: false,
    confirmDiscard: false,
  });
  const [restartBusy, setRestartBusy] = useState(false);

  useEffect(() => {
    void reloadRuntime();
  }, []);

  useEffect(() => {
    if (!restartDialog.open) {
      return;
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape" && !restartBusy) {
        setRestartDialog({ open: false, syncRemoteMaster: false, confirmDiscard: false });
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [restartBusy, restartDialog.open]);

  async function reloadRuntime(nextMessage = "", nextKind: "success" | "error" | "" = "") {
    setRequestState({ status: "loading", error: "" });
    try {
      const [runtimeStatus, providerPayload] = await Promise.all([
        apiClient.get<RuntimeStatus>("/api/control/codex/runtime"),
        apiClient.get<LLMProviderResponse>("/api/control/llm/providers"),
      ]);
      const nextSelection = deriveRuntimeSelection(runtimeStatus);
      setRuntime(runtimeStatus);
      setProviders(Array.isArray(providerPayload?.items) ? providerPayload.items : []);
      setSelectedModel(nextSelection.model);
      setSelectedReasoning(nextSelection.reasoning);
      setStatusMessage(nextMessage);
      setStatusKind(nextKind);
      setRequestState({ status: "ready", error: "" });
    } catch (error: unknown) {
      setRequestState({
        status: "error",
        error: error instanceof Error ? error.message : "unknown_error",
      });
    }
  }

  function onModelSelectionChange(nextModel: string) {
    setSelectedModel(nextModel);
    const nextRuntimeModel = findRuntimeModel(visibleRuntimeModels(runtime), nextModel);
    const nextOptions = runtimeReasoningOptions(nextRuntimeModel, selectedReasoning);
    const nextDefaultReasoning =
      nextOptions.find((option) => normalizeText(option.reasoning_effort) === selectedReasoning)?.reasoning_effort ||
      normalizeText(nextRuntimeModel?.default_reasoning_effort) ||
      normalizeText(nextOptions[0]?.reasoning_effort);
    setSelectedReasoning(nextDefaultReasoning);
    void persistRuntimeSettings(nextModel, nextDefaultReasoning);
  }

  function onReasoningSelectionChange(nextReasoning: string) {
    setSelectedReasoning(nextReasoning);
    void persistRuntimeSettings(selectedModel, nextReasoning);
  }

  async function persistRuntimeSettings(model: string, reasoning: string) {
    if (!model || !reasoning) {
      return;
    }
    try {
      const runtimeStatus = await apiClient.put<RuntimeStatus>("/api/control/codex/runtime", {
        model: model.trim(),
        reasoning_effort: reasoning.trim(),
      });
      const nextSelection = deriveRuntimeSelection(runtimeStatus);
      setRuntime(runtimeStatus);
      setSelectedModel(nextSelection.model);
      setSelectedReasoning(nextSelection.reasoning);
      setStatusMessage("");
      setStatusKind("");
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    }
  }

  async function requestRuntimeRestart(syncRemoteMaster: boolean, confirmDiscardTrackedChanges = false) {
    setRestartBusy(true);
    try {
      await apiClient.post("/api/control/runtime/restart", {
        sync_remote_master: syncRemoteMaster,
        confirm_discard_tracked_changes: confirmDiscardTrackedChanges,
      });
      setRestartDialog({ open: false, syncRemoteMaster: false, confirmDiscard: false });
      setStatusKind("success");
      setStatusMessage(copy.restartAccepted);
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    } finally {
      setRestartBusy(false);
    }
  }

  if (requestState.status === "loading") {
    return <RuntimeLoadingView copy={copy} />;
  }

  if (requestState.status === "error") {
    return <p className="route-error">{copy.loadFailed(requestState.error)}</p>;
  }

  const runtimeModels = visibleRuntimeModels(runtime);
  const selectedReasoningOptions = runtimeReasoningOptions(findRuntimeModel(runtimeModels, selectedModel), selectedReasoning);
  const runtimeProfile = normalizeText(runtime?.profile) || copy.codexDefault;
  const runtimeIdentity = runtimeIdentityDetails(runtime);
  const providerCount = providers.length;
  const showingDiscardConfirm = restartDialog.syncRemoteMaster && restartDialog.confirmDiscard;
  const restartModal =
    restartDialog.open && typeof document !== "undefined"
      ? createPortal(
          <div
            className="runtime-restart-overlay"
            onMouseDown={(event) => {
              if (!restartBusy && event.currentTarget === event.target) {
                setRestartDialog({ open: false, syncRemoteMaster: false, confirmDiscard: false });
              }
            }}
          >
            <section
              className="runtime-restart-panel"
              role="dialog"
              aria-modal="true"
              aria-labelledby="runtime-restart-title"
            >
              <header className="modal-header">
                <h3 id="runtime-restart-title">{showingDiscardConfirm ? copy.discardConfirmTitle : copy.restartConfirmTitle}</h3>
              </header>
              <div className="modal-body">
                <p>{showingDiscardConfirm ? copy.discardConfirmBody : copy.restartConfirmBody}</p>
                {showingDiscardConfirm ? null : (
                  <label className="codex-runtime-restart-option">
                    <input
                      type="checkbox"
                      checked={restartDialog.syncRemoteMaster}
                      onChange={(event) =>
                        setRestartDialog((current) => ({
                          ...current,
                          syncRemoteMaster: event.target.checked,
                        }))
                      }
                    />
                    <span>
                      <strong>{copy.updateBeforeRestart}</strong>
                      <small>{copy.updateBeforeRestartHint}</small>
                    </span>
                  </label>
                )}
              </div>
              <footer className="modal-footer">
                <button
                  type="button"
                  data-variant="secondary"
                  disabled={restartBusy}
                  onClick={() =>
                    showingDiscardConfirm
                      ? setRestartDialog((current) => ({ ...current, confirmDiscard: false }))
                      : setRestartDialog({ open: false, syncRemoteMaster: false, confirmDiscard: false })
                  }
                >
                  {showingDiscardConfirm ? copy.back : copy.cancel}
                </button>
                <button
                  type="button"
                  disabled={restartBusy}
                  onClick={() => {
                    if (restartDialog.syncRemoteMaster && !restartDialog.confirmDiscard) {
                      setRestartDialog((current) => ({ ...current, confirmDiscard: true }));
                      return;
                    }
                    void requestRuntimeRestart(restartDialog.syncRemoteMaster, restartDialog.syncRemoteMaster && restartDialog.confirmDiscard);
                  }}
                >
                  {restartBusy ? copy.restarting : showingDiscardConfirm ? copy.discardAndRestart : copy.confirmRestart}
                </button>
              </footer>
            </section>
          </div>,
          document.body,
        )
      : null;

  return (
    <section className="codex-runtime-view">
      {statusMessage ? (
        <p className={`codex-runtime-status ${statusKind === "error" ? "is-error" : "is-success"}`}>
          {statusMessage}
        </p>
      ) : null}

      <section className="codex-runtime-service-controls route-surface">
        <div className="codex-runtime-title-block">
          <h4>{copy.serviceControls}</h4>
          <p>{copy.serviceControlsSubtitle}</p>
        </div>
        <div className="codex-runtime-service-actions">
          <button
            className="route-card-action codex-runtime-service-primary-action"
            type="button"
            onClick={() => setRestartDialog({ open: true, syncRemoteMaster: false, confirmDiscard: false })}
          >
            {copy.restartService}
          </button>
        </div>
      </section>

      <section className="codex-runtime-status-band route-surface">
        <div className="codex-runtime-status-band-head">
          <div className="codex-runtime-title-block">
            <h4>{copy.overview}</h4>
            <p>{copy.overviewSubtitle}</p>
          </div>
        </div>

        <div className="codex-runtime-identity-card">
          <div className="codex-runtime-account-pane">
            <div className="codex-runtime-account-primary">
              <span>{copy.identityName}</span>
              <strong>{runtimeIdentity.name}</strong>
              <p>{runtimeIdentity.email}</p>
            </div>
            <span className="codex-runtime-account-mark" aria-hidden="true" />
          </div>
          <div
            className="codex-runtime-chip-row"
            aria-label={`${copy.identityPlan} ${copy.identityAuthMode} ${copy.activeProfile}`}
          >
            <RuntimeMetaItem label={copy.identityPlan} value={runtimeIdentity.plan} />
            <RuntimeMetaItem label={copy.identityAuthMode} value={runtimeIdentity.authMode} />
            <RuntimeMetaItem label={copy.activeProfile} value={runtimeProfile} />
          </div>
        </div>

        <div className="codex-runtime-quick-form">
          <div className="codex-runtime-ledger-grid is-editable">
            <RuntimeSelectItem
              label={copy.model}
              value={selectedModel}
              disabled={runtimeModels.length === 0}
              onChange={onModelSelectionChange}
            >
              {runtimeModels.length === 0 ? (
                <option value="">{copy.modelUnavailable}</option>
              ) : (
                runtimeModels.map((item) => {
                  const value = runtimeModelKey(item);
                  return (
                    <option key={value} value={value}>
                      {formatRuntimeModelSummary(item)}
                    </option>
                  );
                })
              )}
            </RuntimeSelectItem>
            <RuntimeSelectItem
              label={copy.reasoningDepth}
              value={selectedReasoning}
              disabled={selectedReasoningOptions.length === 0}
              onChange={onReasoningSelectionChange}
            >
              {selectedReasoningOptions.length === 0 ? (
                <option value="">{copy.reasoningUnavailable}</option>
              ) : (
                selectedReasoningOptions.map((option) => {
                  const value = normalizeText(option.reasoning_effort);
                  return (
                    <option key={value} value={value}>
                      {formatReasoningOption(option)}
                    </option>
                  );
                })
              )}
            </RuntimeSelectItem>
          </div>
        </div>

        <div className="codex-runtime-quota-grid is-compact">
          <RuntimeQuotaItem label={copy.quotaHourly} copy={copy} window={runtime?.current?.quota?.hourly} />
          <RuntimeQuotaItem label={copy.quotaWeekly} copy={copy} window={runtime?.current?.quota?.weekly} />
        </div>

        <div className={providerCount > 0 ? "codex-runtime-provider-note is-ready" : "codex-runtime-provider-note is-empty"}>
          <strong>{providerCount > 0 ? copy.providerRegistered(providerCount) : copy.providersMissing}</strong>
          <span>{providerCount > 0 ? copy.providersReadyHint : copy.providersMissingHint}</span>
        </div>
      </section>
      {restartModal}
    </section>
  );
}

function RuntimeLoadingView({ copy }: { copy: RuntimeCopy }) {
  return (
    <section className="codex-runtime-view codex-runtime-view-loading" aria-busy="true">
      <section className="codex-runtime-status-band route-surface codex-runtime-skeleton-card">
        <div className="codex-runtime-status-band-head">
          <div>
            <h4>{copy.overview}</h4>
            <p>{copy.loading}</p>
          </div>
        </div>
        <div className="codex-runtime-skeleton-stack codex-runtime-skeleton-ledger" aria-hidden="true">
          <span className="runtime-skeleton-line codex-runtime-skeleton-field" />
          <span className="runtime-skeleton-line codex-runtime-skeleton-field" />
          <span className="runtime-skeleton-line codex-runtime-skeleton-block" />
        </div>
      </section>
    </section>
  );
}

function RuntimeSelectItem({
  label,
  value,
  disabled,
  onChange,
  children,
}: {
  label: string;
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
  children: ReactNode;
}) {
  return (
    <label className="codex-runtime-kv-select">
      <span>{label}</span>
      <select
        aria-label={label}
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
      >
        {children}
      </select>
    </label>
  );
}

function RuntimeQuotaItem({
  label,
  copy,
  window,
}: {
  label: string;
  copy: RuntimeCopy;
  window?: RuntimeQuotaWindow | null;
}) {
  const percent = normalizeQuotaPercent(window?.remaining_percent);
  const value = percent == null ? "-" : `${percent}%`;
  const reset = formatDateTimeMinute(window?.reset_at);
  return (
    <article className="codex-runtime-quota-item">
      <div className="codex-runtime-quota-head">
        <span>{label}</span>
        <strong>{value}</strong>
      </div>
      <div
        className="codex-runtime-quota-meter"
        role="progressbar"
        aria-label={`${label} ${copy.quotaRemaining}`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent ?? 0}
      >
        <span style={{ width: `${percent ?? 0}%` }} />
      </div>
      <div className="codex-runtime-quota-reset">
        <span>{copy.quotaResets}</span>
        <strong>{reset}</strong>
      </div>
    </article>
  );
}

function RuntimeMetaItem({ label, value }: { label: string; value: string }) {
  return (
    <span className="codex-runtime-meta-item">
      <span>{label}</span>
      <strong>{value}</strong>
    </span>
  );
}

function runtimeIdentityDetails(runtime: RuntimeStatus | null) {
  const live = runtime?.current?.live ?? null;
  const managed = runtime?.current?.managed ?? null;
  const snapshot = live ?? managed?.snapshot ?? null;
  return {
    name: normalizeText(snapshot?.account_name) || normalizeText(managed?.name) || "-",
    email: normalizeText(snapshot?.email) || "-",
    plan: normalizeText(runtime?.current?.quota?.plan) || normalizeText(snapshot?.plan) || "-",
    authMode: normalizeText(snapshot?.auth_mode) || "-",
  };
}

function normalizeQuotaPercent(value: unknown) {
  const parsed = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(parsed)) {
    return null;
  }
  return Math.max(0, Math.min(100, Math.round(parsed)));
}

function deriveRuntimeSelection(runtime: RuntimeStatus | null) {
  const models = visibleRuntimeModels(runtime);
  const fallbackModel = models.find((item) => item.is_default) ?? models[0] ?? null;
  const currentModel = normalizeText(runtime?.model);
  const model = findRuntimeModel(models, currentModel) ? currentModel : runtimeModelKey(fallbackModel);
  const selectedRuntimeModel = findRuntimeModel(models, model);
  const currentReasoning = normalizeText(runtime?.reasoning_effort);
  const options = runtimeReasoningOptions(selectedRuntimeModel, currentReasoning);
  const reasoning = options.some((option) => normalizeText(option.reasoning_effort) === currentReasoning)
    ? currentReasoning
    : (normalizeText(selectedRuntimeModel?.default_reasoning_effort) || normalizeText(options[0]?.reasoning_effort));
  return { model, reasoning };
}

function visibleRuntimeModels(runtime: RuntimeStatus | null) {
  const currentModel = normalizeText(runtime?.model);
  return Array.isArray(runtime?.models)
    ? runtime.models.filter((item) => {
      const value = runtimeModelKey(item);
      return Boolean(value) && (!item.hidden || value === currentModel);
    })
    : [];
}

function findRuntimeModel(models: RuntimeModel[], value: string) {
  return models.find((item) => runtimeModelKey(item) === value) ?? null;
}

function runtimeReasoningOptions(model: RuntimeModel | null, currentReasoning = "") {
  const deduped = new Map<string, RuntimeReasoningMode>();
  for (const option of model?.supported_reasoning_effort ?? []) {
    const effort = normalizeText(option.reasoning_effort);
    if (!effort) {
      continue;
    }
    deduped.set(effort, option);
  }
  if (deduped.size > 0) {
    return Array.from(deduped.values());
  }
  const fallback = normalizeText(model?.default_reasoning_effort) || normalizeText(currentReasoning);
  return fallback ? [{ reasoning_effort: fallback }] : [];
}

function runtimeModelKey(model: RuntimeModel | null | undefined) {
  return normalizeText(model?.model) || normalizeText(model?.id);
}

function runtimeModelDisplayName(model: RuntimeModel | null | undefined) {
  return normalizeText(model?.display_name) || runtimeModelKey(model);
}

function formatRuntimeModelSummary(model: RuntimeModel | null | undefined) {
  const label = runtimeModelDisplayName(model) || "-";
  const value = runtimeModelKey(model);
  return value && value !== label ? `${label} (${value})` : label;
}

function formatReasoningOption(option: RuntimeReasoningMode | null | undefined) {
  return formatReasoningEffort(option?.reasoning_effort);
}

function formatReasoningEffort(value: unknown) {
  const effort = normalizeText(value).toLowerCase();
  if (!effort) {
    return "";
  }
  if (effort === "xhigh") {
    return "Max";
  }
  return effort
    .split(/[_-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function normalizeText(value: unknown) {
  return String(value || "").trim();
}
