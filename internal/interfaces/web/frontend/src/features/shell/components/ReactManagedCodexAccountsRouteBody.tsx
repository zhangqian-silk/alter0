import { useEffect, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { APIClientError, createAPIClient } from "../../../shared/api/client";
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

type CodexLoginDeviceInfo = {
  verification_uri?: string;
  verification_uri_complete?: string;
  user_code?: string;
  expires_in?: number;
  interval?: number;
  message?: string;
};

type CodexLoginSession = {
  id?: string;
  account_name?: string;
  auth_method?: "browser" | "device_auth" | string;
  status?: "pending" | "running" | "succeeded" | "failed" | string;
  logs?: string;
  error?: string;
  device?: CodexLoginDeviceInfo | null;
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
  provider_type?: string;
  api_type?: string;
  base_url?: string;
  api_key?: string;
  default_model?: string;
  models?: LLMProviderModel[];
  is_enabled?: boolean;
  is_default?: boolean;
};

type LLMProviderModel = {
  id?: string;
  name?: string;
  is_enabled?: boolean;
};

type LLMProviderResponse = {
  items?: LLMProviderRecord[];
};

type RuntimeRestartStatus = {
  status?: string;
  error?: string;
  sync_remote_master?: boolean;
  confirm_discard_tracked_changes?: boolean;
  started_at?: string;
  updated_at?: string;
};

const runtimeRestartDiscardConfirmationRequired = "runtime_restart_discard_confirmation_required";

type ProviderRegistrationForm = {
  name: string;
  baseURL: string;
  apiKey: string;
  models: string;
  providerType: string;
  apiType: string;
  isEnabled: boolean;
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
  providerRegisterTitle: string;
  providerRegisterSubtitle: string;
  providerListTitle: string;
  providerListSubtitle: string;
  providerName: string;
  providerBaseURL: string;
  providerAPIKey: string;
  providerModels: string;
  providerModelsHelp: string;
  providerStoredAPIKeyHelp: string;
  providerRegisterAction: string;
  providerUpdateAction: string;
  providerNewAction: string;
  providerEditButton: string;
  providerEditAction: (name: string) => string;
  providerRegistering: string;
  providerUpdating: string;
  providerRegisterSucceeded: string;
  providerUpdateSucceeded: string;
  providerDefaultModel: (model: string) => string;
  providerModelCount: (count: number) => string;
  providerEnabled: string;
  providerDisabled: string;
  providerDefault: string;
  providerNoModels: string;
  activeProfile: string;
  identityName: string;
  identityPlan: string;
  identityAuthMode: string;
  quotaHourly: string;
  quotaWeekly: string;
  quotaRemaining: string;
  quotaResets: string;
  codexDefault: string;
  deviceLoginTitle: string;
  deviceLoginSubtitle: string;
  startDeviceLogin: string;
  startingDeviceLogin: string;
  deviceLoginPending: string;
  deviceLoginSucceeded: string;
  deviceLoginFailed: string;
  deviceVerificationLink: string;
  deviceUserCode: string;
  deviceExpiresIn: (seconds: number) => string;
  devicePollInterval: (seconds: number) => string;
  loginLogs: string;
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
  restartLastStatus: string;
  restartStatusPreparing: string;
  restartStatusSwitching: string;
  restartStatusCompleted: string;
  restartStatusFailed: string;
  restartStatusUnknown: string;
  restartUpdatedAt: (value: string) => string;
  restartSyncEnabled: string;
  restartFailureReason: (message: string) => string;
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
    providerRegisterTitle: "Claude Code Provider",
    providerRegisterSubtitle: "Register an OpenAI-compatible endpoint for Claude Code execution.",
    providerListTitle: "Configured Providers",
    providerListSubtitle: "Review registered endpoints and load one into the form to update its URL, key, or models.",
    providerName: "Provider Name",
    providerBaseURL: "Base URL",
    providerAPIKey: "API Key",
    providerModels: "Provider Models",
    providerModelsHelp: "Use one model per line or separate models with commas. The first model becomes the default.",
    providerStoredAPIKeyHelp: "Leave blank to keep the stored API key.",
    providerRegisterAction: "Register provider",
    providerUpdateAction: "Update provider",
    providerNewAction: "New provider",
    providerEditButton: "Edit",
    providerEditAction: (name) => `Edit ${name}`,
    providerRegistering: "Registering...",
    providerUpdating: "Updating...",
    providerRegisterSucceeded: "Provider registered for Claude Code.",
    providerUpdateSucceeded: "Provider updated for Claude Code.",
    providerDefaultModel: (model) => `Default: ${model || "-"}`,
    providerModelCount: (count) => `${count} model${count === 1 ? "" : "s"}`,
    providerEnabled: "Enabled",
    providerDisabled: "Disabled",
    providerDefault: "Default",
    providerNoModels: "No models configured",
    activeProfile: "Profile",
    identityName: "Account",
    identityPlan: "Plan",
    identityAuthMode: "Auth Mode",
    quotaHourly: "Hourly",
    quotaWeekly: "Weekly",
    quotaRemaining: "Remaining",
    quotaResets: "Resets",
    codexDefault: "Codex default",
    deviceLoginTitle: "Device Code Login",
    deviceLoginSubtitle: "Start a headless ChatGPT sign-in for this runtime and enter the one-time code in your browser.",
    startDeviceLogin: "Start device login",
    startingDeviceLogin: "Starting...",
    deviceLoginPending: "Waiting for browser confirmation.",
    deviceLoginSucceeded: "Codex login succeeded. Runtime identity has been refreshed.",
    deviceLoginFailed: "Codex login failed.",
    deviceVerificationLink: "Verification link",
    deviceUserCode: "User code",
    deviceExpiresIn: (seconds) => `Expires in ${seconds}s`,
    devicePollInterval: (seconds) => `Poll every ${seconds}s`,
    loginLogs: "Login output",
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
    restartLastStatus: "Last restart",
    restartStatusPreparing: "Preparing candidate",
    restartStatusSwitching: "Switching runtime",
    restartStatusCompleted: "Completed",
    restartStatusFailed: "Failed and rolled back",
    restartStatusUnknown: "Status unavailable",
    restartUpdatedAt: (value) => `Updated ${value}`,
    restartSyncEnabled: "Remote master sync requested",
    restartFailureReason: (message) => `Failure reason: ${message}`,
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
    providerRegisterTitle: "Claude Code Provider",
    providerRegisterSubtitle: "注册 OpenAI-compatible endpoint，供 Claude Code 执行链使用。",
    providerListTitle: "已配置 Provider",
    providerListSubtitle: "查看已注册 endpoint，并载入表单修改 URL、密钥或 models。",
    providerName: "Provider 名称",
    providerBaseURL: "Base URL",
    providerAPIKey: "API Key",
    providerModels: "Provider Models",
    providerModelsHelp: "每行填写一个 model，也可用逗号分隔；第一个 model 会作为默认模型。",
    providerStoredAPIKeyHelp: "留空表示保留已保存的 API key。",
    providerRegisterAction: "注册 Provider",
    providerUpdateAction: "更新 Provider",
    providerNewAction: "新建 Provider",
    providerEditButton: "编辑",
    providerEditAction: (name) => `编辑 ${name}`,
    providerRegistering: "注册中...",
    providerUpdating: "更新中...",
    providerRegisterSucceeded: "Claude Code Provider 已注册。",
    providerUpdateSucceeded: "Claude Code Provider 已更新。",
    providerDefaultModel: (model) => `默认：${model || "-"}`,
    providerModelCount: (count) => `${count} 个 model`,
    providerEnabled: "已启用",
    providerDisabled: "已停用",
    providerDefault: "默认",
    providerNoModels: "暂无 models",
    activeProfile: "Profile",
    identityName: "账号",
    identityPlan: "计划",
    identityAuthMode: "认证模式",
    quotaHourly: "小时额度",
    quotaWeekly: "周额度",
    quotaRemaining: "剩余",
    quotaResets: "重置",
    codexDefault: "Codex 默认值",
    deviceLoginTitle: "Device Code 登录",
    deviceLoginSubtitle: "为当前运行时启动无头 ChatGPT 登录，并在浏览器中输入一次性验证码。",
    startDeviceLogin: "启动 device 登录",
    startingDeviceLogin: "启动中...",
    deviceLoginPending: "等待浏览器确认。",
    deviceLoginSucceeded: "Codex 登录成功，运行时身份已刷新。",
    deviceLoginFailed: "Codex 登录失败。",
    deviceVerificationLink: "验证链接",
    deviceUserCode: "用户码",
    deviceExpiresIn: (seconds) => `${seconds}s 后过期`,
    devicePollInterval: (seconds) => `每 ${seconds}s 轮询`,
    loginLogs: "登录输出",
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
    restartLastStatus: "最近一次重启",
    restartStatusPreparing: "正在准备候选版本",
    restartStatusSwitching: "正在切换运行时",
    restartStatusCompleted: "已完成",
    restartStatusFailed: "失败并已回滚",
    restartStatusUnknown: "状态不可用",
    restartUpdatedAt: (value) => `更新于 ${value}`,
    restartSyncEnabled: "已请求同步远端 master",
    restartFailureReason: (message) => `失败原因：${message}`,
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
  const [restartStatus, setRestartStatus] = useState<RuntimeRestartStatus | null>(null);
  const [loginSession, setLoginSession] = useState<CodexLoginSession | null>(null);
  const [loginBusy, setLoginBusy] = useState(false);
  const [editingProviderID, setEditingProviderID] = useState("");
  const [providerForm, setProviderForm] = useState<ProviderRegistrationForm>(() => createProviderRegistrationForm("Claude Code"));
  const [providerBusy, setProviderBusy] = useState(false);
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
    if (!isActiveRestartStatus(restartStatus)) {
      return;
    }
    const timer = window.setInterval(() => {
      void reloadRestartStatus();
    }, 2500);
    return () => window.clearInterval(timer);
  }, [restartStatus?.status]);

  useEffect(() => {
    const sessionID = normalizeText(loginSession?.id);
    if (!sessionID || !isLoginSessionActive(loginSession)) {
      return;
    }
    const intervalSeconds = normalizePositiveNumber(loginSession?.device?.interval) || 5;
    const timer = window.setTimeout(() => {
      void pollLoginSession(sessionID);
    }, Math.max(2, intervalSeconds) * 1000);
    return () => window.clearTimeout(timer);
  }, [loginSession?.id, loginSession?.status, loginSession?.device?.interval]);

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

  async function reloadRuntime(
    nextMessage = "",
    nextKind: "success" | "error" | "" = "",
    options: { resetProviderForm?: boolean } = {},
  ) {
    setRequestState({ status: "loading", error: "" });
    try {
      const [runtimeStatus, providerPayload, nextRestartStatus] = await Promise.all([
        apiClient.get<RuntimeStatus>("/api/control/codex/runtime"),
        apiClient.get<LLMProviderResponse>("/api/control/llm/providers"),
        apiClient.get<RuntimeRestartStatus>("/api/control/runtime/restart").catch(() => null),
      ]);
      const nextSelection = deriveRuntimeSelection(runtimeStatus);
      const providerItems = Array.isArray(providerPayload?.items) ? providerPayload.items : [];
      setRuntime(runtimeStatus);
      setRestartStatus(normalizeRestartStatus(nextRestartStatus));
      setProviders(providerItems);
      setProviderForm((current) => {
        const nextName = nextClaudeCodeProviderName(providerItems);
        if (options.resetProviderForm) {
          return createProviderRegistrationForm(nextName);
        }
        return isProviderRegistrationFormPristine(current) ? { ...current, name: nextName } : current;
      });
      if (options.resetProviderForm) {
        setEditingProviderID("");
      }
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

  async function reloadRestartStatus() {
    try {
      const nextStatus = await apiClient.get<RuntimeRestartStatus>("/api/control/runtime/restart");
      setRestartStatus(normalizeRestartStatus(nextStatus));
    } catch {
      setRestartStatus(null);
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

  async function registerClaudeCodeProvider() {
    const name = normalizeText(providerForm.name) || "Claude Code";
    const baseURL = normalizeText(providerForm.baseURL);
    const apiKey = normalizeText(providerForm.apiKey);
    const modelIDs = parseProviderModelIDs(providerForm.models);
    if (!baseURL || (!apiKey && !editingProviderID) || modelIDs.length === 0) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed("base_url, api_key, and at least one model are required"));
      return;
    }

    setProviderBusy(true);
    setStatusMessage("");
    setStatusKind("");
    try {
      const payload = {
        id: editingProviderID || undefined,
        name,
        provider_type: normalizeText(providerForm.providerType) || "openai-compatible",
        api_type: normalizeText(providerForm.apiType) || "openai-completions",
        base_url: baseURL,
        api_key: apiKey,
        default_model: modelIDs[0],
        models: modelIDs.map((modelID) => ({
          id: modelID,
          name: modelID,
          supports_tools: true,
          supports_vision: true,
          supports_streaming: true,
          is_enabled: true,
        })),
        is_enabled: providerForm.isEnabled,
      };
      if (editingProviderID) {
        await apiClient.put(`/api/control/llm/providers/${encodeURIComponent(editingProviderID)}`, payload);
        await reloadRuntime(copy.providerUpdateSucceeded, "success", { resetProviderForm: true });
      } else {
        await apiClient.post("/api/control/llm/providers", payload);
        await reloadRuntime(copy.providerRegisterSucceeded, "success", { resetProviderForm: true });
      }
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    } finally {
      setProviderBusy(false);
    }
  }

  function startProviderEdit(provider: LLMProviderRecord) {
    const providerID = normalizeText(provider.id);
    if (!providerID) {
      return;
    }
    setEditingProviderID(providerID);
    setProviderForm(providerRecordToForm(provider));
    setStatusMessage("");
    setStatusKind("");
  }

  function startProviderCreate() {
    setEditingProviderID("");
    setProviderForm(createProviderRegistrationForm(nextClaudeCodeProviderName(providers)));
    setStatusMessage("");
    setStatusKind("");
  }

  async function requestRuntimeRestart(syncRemoteMaster: boolean, confirmDiscardTrackedChanges = false) {
    setRestartBusy(true);
    try {
      const acceptedStatus = await apiClient.post<RuntimeRestartStatus>("/api/control/runtime/restart", {
        sync_remote_master: syncRemoteMaster,
        confirm_discard_tracked_changes: confirmDiscardTrackedChanges,
      });
      setRestartStatus(normalizeRestartStatus(acceptedStatus));
      setRestartDialog({ open: false, syncRemoteMaster: false, confirmDiscard: false });
      setStatusKind("success");
      setStatusMessage(copy.restartAccepted);
    } catch (error: unknown) {
      if (
        syncRemoteMaster &&
        !confirmDiscardTrackedChanges &&
        error instanceof APIClientError &&
        error.code === runtimeRestartDiscardConfirmationRequired
      ) {
        setRestartDialog({ open: true, syncRemoteMaster: true, confirmDiscard: true });
        setStatusKind("");
        setStatusMessage("");
        return;
      }
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
      void reloadRestartStatus();
    } finally {
      setRestartBusy(false);
    }
  }

  async function startDeviceLogin() {
    setLoginBusy(true);
    setStatusMessage("");
    setStatusKind("");
    try {
      const session = await apiClient.post<CodexLoginSession>("/api/control/codex/accounts/login-sessions", {
        name: "runtime-device",
        overwrite: true,
        auth_method: "device_auth",
      });
      setLoginSession(session);
      const sessionID = normalizeText(session?.id);
      if (sessionID) {
        await pollLoginSession(sessionID);
      }
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    } finally {
      setLoginBusy(false);
    }
  }

  async function pollLoginSession(sessionID: string, refreshOnSuccess = true) {
    try {
      const session = await apiClient.get<CodexLoginSession>(
        `/api/control/codex/accounts/login-sessions/${encodeURIComponent(sessionID)}`,
      );
      setLoginSession(session);
      if (session.status === "succeeded" && refreshOnSuccess) {
        await reloadRuntime(copy.deviceLoginSucceeded, "success");
      }
      if (session.status === "failed") {
        setStatusKind("error");
        setStatusMessage(`${copy.deviceLoginFailed}${session.error ? ` ${session.error}` : ""}`);
      }
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
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
  const restartStatusVisible = restartStatus && normalizeText(restartStatus.status) !== "" && normalizeText(restartStatus.status) !== "idle";
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
          {restartStatusVisible && restartStatus ? <RuntimeRestartStatusNote status={restartStatus} copy={copy} /> : null}
        </div>
        <div className="codex-runtime-service-actions">
          <button
            className="route-card-action codex-runtime-service-primary-action"
            type="button"
            onClick={() => setRestartDialog({ open: true, syncRemoteMaster: true, confirmDiscard: false })}
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

        <section className="codex-runtime-device-login" aria-label={copy.deviceLoginTitle}>
          <div className="codex-runtime-device-login-head">
            <div className="codex-runtime-title-block">
              <h4>{copy.deviceLoginTitle}</h4>
              <p>{copy.deviceLoginSubtitle}</p>
            </div>
            <button
              className="route-card-action codex-runtime-device-login-action"
              type="button"
              disabled={loginBusy || isLoginSessionActive(loginSession)}
              onClick={() => void startDeviceLogin()}
            >
              {loginBusy ? copy.startingDeviceLogin : copy.startDeviceLogin}
            </button>
          </div>
          {loginSession ? <RuntimeDeviceLoginSession copy={copy} session={loginSession} /> : null}
        </section>

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

        <RuntimeProviderConsole
          copy={copy}
          providers={providers}
          providerCount={providerCount}
          providerForm={providerForm}
          providerBusy={providerBusy}
          editingProviderID={editingProviderID}
          editing={Boolean(editingProviderID)}
          onEdit={startProviderEdit}
          onChangeProviderForm={setProviderForm}
          onNew={startProviderCreate}
          onSubmit={() => void registerClaudeCodeProvider()}
        />
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

function RuntimeRestartStatusNote({ status, copy }: { status: RuntimeRestartStatus; copy: RuntimeCopy }) {
  const normalizedStatus = normalizeText(status.status);
  const updatedAt = normalizeText(status.updated_at || status.started_at);
  const statusText = formatRestartStatusLabel(normalizedStatus, copy);
  const isError = normalizedStatus === "failed";
  return (
    <div className={isError ? "codex-runtime-restart-status is-error" : "codex-runtime-restart-status"}>
      <strong>{copy.restartLastStatus}</strong>
      <span>{statusText}</span>
      {updatedAt ? <small>{copy.restartUpdatedAt(formatDateTimeMinute(updatedAt))}</small> : null}
      {status.sync_remote_master ? <small>{copy.restartSyncEnabled}</small> : null}
      {status.error ? <small>{copy.restartFailureReason(status.error)}</small> : null}
    </div>
  );
}

function formatRestartStatusLabel(status: string, copy: RuntimeCopy) {
  switch (status) {
    case "preparing":
      return copy.restartStatusPreparing;
    case "switching":
    case "restarting":
      return copy.restartStatusSwitching;
    case "completed":
      return copy.restartStatusCompleted;
    case "failed":
      return copy.restartStatusFailed;
    default:
      return copy.restartStatusUnknown;
  }
}

function normalizeRestartStatus(status: RuntimeRestartStatus | null | undefined): RuntimeRestartStatus | null {
  if (!status || typeof status !== "object") {
    return null;
  }
  const normalizedStatus = normalizeText(status.status);
  if (!normalizedStatus || normalizedStatus === "idle") {
    return null;
  }
  return { ...status, status: normalizedStatus };
}

function isActiveRestartStatus(status: RuntimeRestartStatus | null): boolean {
  const normalizedStatus = normalizeText(status?.status);
  return normalizedStatus === "preparing" || normalizedStatus === "switching" || normalizedStatus === "restarting";
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

function RuntimeProviderConsole({
  copy,
  providers,
  providerCount,
  providerForm,
  providerBusy,
  editingProviderID,
  editing,
  onEdit,
  onChangeProviderForm,
  onNew,
  onSubmit,
}: {
  copy: RuntimeCopy;
  providers: LLMProviderRecord[];
  providerCount: number;
  providerForm: ProviderRegistrationForm;
  providerBusy: boolean;
  editingProviderID: string;
  editing: boolean;
  onEdit: (provider: LLMProviderRecord) => void;
  onChangeProviderForm: (value: ProviderRegistrationForm) => void;
  onNew: () => void;
  onSubmit: () => void;
}) {
  return (
    <section className="codex-runtime-provider-console" aria-label={copy.providerRegisterTitle}>
      <header className="codex-runtime-provider-console-head">
        <div className={providerCount > 0 ? "codex-runtime-provider-note is-ready" : "codex-runtime-provider-note is-empty"}>
          <strong>{providerCount > 0 ? copy.providerRegistered(providerCount) : copy.providersMissing}</strong>
          <span>{providerCount > 0 ? copy.providersReadyHint : copy.providersMissingHint}</span>
        </div>
      </header>
      <div className={providerCount > 0 ? "codex-runtime-provider-console-grid" : "codex-runtime-provider-console-grid is-empty"}>
        {providerCount > 0 ? (
          <RuntimeProviderList
            copy={copy}
            providers={providers}
            editingProviderID={editingProviderID}
            onEdit={onEdit}
          />
        ) : null}
        <div className="codex-runtime-provider-editor">
          <RuntimeProviderRegistrationForm
            copy={copy}
            value={providerForm}
            busy={providerBusy}
            editing={editing}
            onChange={onChangeProviderForm}
            onNew={onNew}
            onSubmit={onSubmit}
          />
        </div>
      </div>
    </section>
  );
}

function RuntimeProviderList({
  copy,
  providers,
  editingProviderID,
  onEdit,
}: {
  copy: RuntimeCopy;
  providers: LLMProviderRecord[];
  editingProviderID: string;
  onEdit: (provider: LLMProviderRecord) => void;
}) {
  return (
    <section className="codex-runtime-provider-registry" aria-label={copy.providerListTitle}>
      <div className="codex-runtime-provider-list-head">
        <div className="codex-runtime-title-block">
          <h4>{copy.providerListTitle}</h4>
          <p>{copy.providerListSubtitle}</p>
        </div>
      </div>
      <div className="codex-runtime-provider-items">
        {providers.map((provider) => {
          const providerID = normalizeText(provider.id);
          const providerName = normalizeText(provider.name) || providerID || "Provider";
          const modelIDs = providerModelIDs(provider);
          const defaultModel = normalizeText(provider.default_model) || modelIDs[0] || "";
          const modelSummary = modelIDs.length > 0 ? modelIDs.join(", ") : copy.providerNoModels;
          return (
            <article
              className={providerID && providerID === editingProviderID ? "codex-runtime-provider-item is-editing" : "codex-runtime-provider-item"}
              key={providerID || providerName}
            >
              <div className="codex-runtime-provider-item-main">
                <div className="codex-runtime-provider-item-title">
                  <strong>{providerName}</strong>
                  <span>{provider.is_enabled === false ? copy.providerDisabled : copy.providerEnabled}</span>
                  {provider.is_default ? <span>{copy.providerDefault}</span> : null}
                </div>
                <p>{normalizeText(provider.base_url) || "-"}</p>
                <div className="codex-runtime-provider-model-row">
                  <span>{copy.providerDefaultModel(defaultModel)}</span>
                  <span>{copy.providerModelCount(modelIDs.length)}</span>
                </div>
                <small>{modelSummary}</small>
              </div>
              <button
                className="route-card-action codex-runtime-provider-edit"
                type="button"
                aria-label={copy.providerEditAction(providerName)}
                onClick={() => onEdit(provider)}
              >
                {copy.providerEditButton}
              </button>
            </article>
          );
        })}
      </div>
    </section>
  );
}

function RuntimeProviderRegistrationForm({
  copy,
  value,
  busy,
  editing,
  onChange,
  onNew,
  onSubmit,
}: {
  copy: RuntimeCopy;
  value: ProviderRegistrationForm;
  busy: boolean;
  editing: boolean;
  onChange: (value: ProviderRegistrationForm) => void;
  onNew: () => void;
  onSubmit: () => void;
}) {
  return (
    <form
      className="codex-runtime-provider-form"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      <div className="codex-runtime-provider-form-head">
        <div className="codex-runtime-title-block">
          <h4>{copy.providerRegisterTitle}</h4>
          <p>{copy.providerRegisterSubtitle}</p>
        </div>
        <div className="codex-runtime-provider-actions">
          {editing ? (
            <button className="route-card-action codex-runtime-provider-new" type="button" disabled={busy} onClick={onNew}>
              {copy.providerNewAction}
            </button>
          ) : null}
          <button className="route-card-action codex-runtime-provider-submit" type="submit" disabled={busy}>
            {busy ? (editing ? copy.providerUpdating : copy.providerRegistering) : editing ? copy.providerUpdateAction : copy.providerRegisterAction}
          </button>
        </div>
      </div>
      <div className="codex-runtime-provider-fields">
        <RuntimeTextField
          label={copy.providerName}
          value={value.name}
          onChange={(nextValue) => onChange({ ...value, name: nextValue })}
        />
        <RuntimeTextField
          label={copy.providerBaseURL}
          value={value.baseURL}
          onChange={(nextValue) => onChange({ ...value, baseURL: nextValue })}
          placeholder="https://api.example.com/v1"
        />
        <RuntimeTextField
          label={copy.providerAPIKey}
          value={value.apiKey}
          onChange={(nextValue) => onChange({ ...value, apiKey: nextValue })}
          type="password"
          autoComplete="off"
          help={editing ? copy.providerStoredAPIKeyHelp : ""}
        />
        <RuntimeTextField
          label={copy.providerModels}
          value={value.models}
          onChange={(nextValue) => onChange({ ...value, models: nextValue })}
          placeholder={"claude-sonnet-4\nclaude-opus-4\nclaude-haiku-4"}
          help={copy.providerModelsHelp}
          fieldClassName="codex-runtime-provider-models-field"
          multiline
          rows={4}
        />
      </div>
    </form>
  );
}

function RuntimeTextField({
  label,
  value,
  onChange,
  type = "text",
  placeholder = "",
  autoComplete,
  multiline = false,
  help = "",
  fieldClassName = "",
  rows = 2,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: "text" | "password";
  placeholder?: string;
  autoComplete?: string;
  multiline?: boolean;
  help?: string;
  fieldClassName?: string;
  rows?: number;
}) {
  return (
    <label className={fieldClassName ? `codex-runtime-text-field ${fieldClassName}` : "codex-runtime-text-field"}>
      <span>{label}</span>
      {multiline ? (
        <textarea
          aria-label={label}
          value={value}
          placeholder={placeholder}
          rows={rows}
          onChange={(event) => onChange(event.target.value)}
        />
      ) : (
        <input
          aria-label={label}
          type={type}
          value={value}
          placeholder={placeholder}
          autoComplete={autoComplete}
          onChange={(event) => onChange(event.target.value)}
        />
      )}
      {help ? <small>{help}</small> : null}
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

function RuntimeDeviceLoginSession({
  copy,
  session,
}: {
  copy: RuntimeCopy;
  session: CodexLoginSession;
}) {
  const device = session.device ?? null;
  const verificationLink = normalizeText(device?.verification_uri_complete) || normalizeText(device?.verification_uri);
  const userCode = normalizeText(device?.user_code);
  const expiresIn = normalizePositiveNumber(device?.expires_in);
  const interval = normalizePositiveNumber(device?.interval);
  const status = normalizeText(session.status);
  const statusLabel =
    status === "succeeded"
      ? copy.deviceLoginSucceeded
      : status === "failed"
        ? `${copy.deviceLoginFailed}${session.error ? ` ${session.error}` : ""}`
        : copy.deviceLoginPending;

  return (
    <div className={`codex-runtime-device-session is-${status || "pending"}`}>
      <p className="codex-runtime-device-session-status">{statusLabel}</p>
      <div className="codex-runtime-device-grid">
        {verificationLink ? (
          <RuntimeDeviceDetail label={copy.deviceVerificationLink}>
            <a href={verificationLink} target="_blank" rel="noreferrer">
              {verificationLink}
            </a>
          </RuntimeDeviceDetail>
        ) : null}
        {userCode ? (
          <RuntimeDeviceDetail label={copy.deviceUserCode}>
            <strong className="codex-runtime-device-code">{userCode}</strong>
          </RuntimeDeviceDetail>
        ) : null}
      </div>
      {expiresIn || interval ? (
        <div className="codex-runtime-device-timing">
          {expiresIn ? <span>{copy.deviceExpiresIn(expiresIn)}</span> : null}
          {interval ? <span>{copy.devicePollInterval(interval)}</span> : null}
        </div>
      ) : null}
      {session.logs ? (
        <details className="codex-runtime-device-logs" open>
          <summary>{copy.loginLogs}</summary>
          <pre>{session.logs}</pre>
        </details>
      ) : null}
    </div>
  );
}

function RuntimeDeviceDetail({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="codex-runtime-device-detail">
      <span>{label}</span>
      <div>{children}</div>
    </div>
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

function isLoginSessionActive(session: CodexLoginSession | null) {
  const status = normalizeText(session?.status);
  return status === "pending" || status === "running";
}

function normalizePositiveNumber(value: unknown) {
  const parsed = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 0;
  }
  return Math.round(parsed);
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

function createProviderRegistrationForm(name: string): ProviderRegistrationForm {
  return {
    name,
    baseURL: "",
    apiKey: "",
    models: "",
    providerType: "openai-compatible",
    apiType: "openai-completions",
    isEnabled: true,
  };
}

function providerRecordToForm(provider: LLMProviderRecord): ProviderRegistrationForm {
  return {
    name: normalizeText(provider.name),
    baseURL: normalizeText(provider.base_url),
    apiKey: "",
    models: providerModelIDs(provider).join("\n"),
    providerType: normalizeText(provider.provider_type) || "openai-compatible",
    apiType: normalizeText(provider.api_type) || "openai-completions",
    isEnabled: provider.is_enabled !== false,
  };
}

function providerModelIDs(provider: LLMProviderRecord) {
  const seen = new Set<string>();
  const models: string[] = [];
  for (const model of provider.models ?? []) {
    const modelID = normalizeText(model.id) || normalizeText(model.name);
    if (!modelID || seen.has(modelID)) {
      continue;
    }
    seen.add(modelID);
    models.push(modelID);
  }
  return models;
}

function nextClaudeCodeProviderName(providers: LLMProviderRecord[]) {
  const names = new Set(
    providers
      .map((provider) => normalizeText(provider.name).toLowerCase())
      .filter(Boolean),
  );
  if (!names.has("claude code")) {
    return "Claude Code";
  }
  for (let index = 2; ; index += 1) {
    const candidate = `Claude Code ${index}`;
    if (!names.has(candidate.toLowerCase())) {
      return candidate;
    }
  }
}

function isProviderRegistrationFormPristine(form: ProviderRegistrationForm) {
  const name = normalizeText(form.name);
  return (
    !normalizeText(form.baseURL) &&
    !normalizeText(form.apiKey) &&
    !normalizeText(form.models) &&
    normalizeText(form.providerType) === "openai-compatible" &&
    normalizeText(form.apiType) === "openai-completions" &&
    form.isEnabled &&
    (!name || /^Claude Code(?: \d+)?$/i.test(name))
  );
}

function parseProviderModelIDs(value: unknown) {
  const seen = new Set<string>();
  const models: string[] = [];
  for (const item of String(value || "").split(/[\n,，]+/)) {
    const modelID = normalizeText(item);
    if (!modelID || seen.has(modelID)) {
      continue;
    }
    seen.add(modelID);
    models.push(modelID);
  }
  return models;
}
