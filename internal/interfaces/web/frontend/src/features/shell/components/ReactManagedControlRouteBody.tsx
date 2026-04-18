import { useEffect, useRef, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTime } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  isReactManagedControlRoute,
  type ReactManagedControlRoute,
} from "../reactManagedRouteContract";
import {
  normalizeText,
  RouteCard,
  RouteFieldRow,
  RouteTagSection,
} from "./RouteBodyPrimitives";

type ControlRouteRecord = {
  id?: string;
  type?: string;
  name?: string;
  description?: string;
  scope?: string;
  version?: string;
  enabled?: boolean;
};

type LLMProviderRecord = {
  id?: string;
  name?: string;
  provider_type?: string;
  api_type?: string;
  base_url?: string;
  default_model?: string;
  models?: Array<{
    id?: string;
    name?: string;
    is_enabled?: boolean;
  }>;
  is_enabled?: boolean;
};

type EnvironmentRouteRecord = {
  definition?: {
    key?: string;
    name?: string;
    module?: string;
    description?: string;
    type?: string;
    apply_mode?: string;
    default_value?: string;
    sensitive?: boolean;
    hot_reload?: boolean;
    validation?: {
      required?: boolean;
      min?: string;
      max?: string;
      allowed?: string[];
    };
  };
  value?: string;
  effective_value?: string;
  value_source?: string;
  pending_restart?: boolean;
  masked?: boolean;
};

type EnvironmentAuditRecord = {
  operator?: string;
  occurred_at?: string;
  requires_restart?: boolean;
  changes?: Array<{
    key?: string;
    old_value?: string;
    new_value?: string;
    apply_mode?: string;
  }>;
};

type RuntimeInfoRecord = {
  started_at?: string;
  commit_hash?: string;
};

type CronJobRouteRecord = {
  id?: string;
  name?: string;
  enabled?: boolean;
  schedule_mode?: string;
  timezone?: string;
  cron_expression?: string;
  interval?: string;
  channel_id?: string;
  task_config?: {
    input?: string;
    retry_limit?: number;
  };
};

type RouteRecord = ControlRouteRecord | LLMProviderRecord | EnvironmentRouteRecord | CronJobRouteRecord;

type ControlRouteResponse = {
  items?: RouteRecord[];
};

type ControlRouteCopy = {
  loading: string;
  statusEnabled: string;
  statusDisabled: string;
  statusApplied: string;
  statusPendingRestart: string;
  copyValue: string;
  fieldID: string;
  fieldType: string;
  fieldDescription: string;
  fieldName: string;
  fieldScope: string;
  fieldVersion: string;
  fieldProviderType: string;
  fieldAPIType: string;
  fieldBaseURL: string;
  fieldDefaultModel: string;
  fieldModels: string;
  fieldKey: string;
  fieldValue: string;
  fieldEffectiveValue: string;
  fieldValueSource: string;
  fieldApplyMode: string;
  fieldModule: string;
  fieldScheduleMode: string;
  fieldTimezone: string;
  fieldCronExpression: string;
  fieldInterval: string;
  fieldPrompt: string;
  fieldRetryLimit: string;
  emptyChannels: string;
  emptySkills: string;
  emptyMCP: string;
  emptyModels: string;
  emptyEnvironments: string;
  emptyCronJobs: string;
  loadFailed: (message: string) => string;
};

const CONTROL_ROUTE_COPY: Record<LegacyShellLanguage, ControlRouteCopy> = {
  en: {
    loading: "Loading...",
    statusEnabled: "Enabled",
    statusDisabled: "Disabled",
    statusApplied: "Applied",
    statusPendingRestart: "Pending restart",
    copyValue: "Copy value",
    fieldID: "ID",
    fieldType: "Type",
    fieldDescription: "Description",
    fieldName: "Name",
    fieldScope: "Scope",
    fieldVersion: "Version",
    fieldProviderType: "Provider Type",
    fieldAPIType: "API Type",
    fieldBaseURL: "Base URL",
    fieldDefaultModel: "Default Model",
    fieldModels: "Models",
    fieldKey: "Key",
    fieldValue: "Value",
    fieldEffectiveValue: "Effective Value",
    fieldValueSource: "Source",
    fieldApplyMode: "Apply Mode",
    fieldModule: "Module",
    fieldScheduleMode: "Schedule Mode",
    fieldTimezone: "Timezone",
    fieldCronExpression: "Cron Expression",
    fieldInterval: "Interval",
    fieldPrompt: "Prompt",
    fieldRetryLimit: "Retry Limit",
    emptyChannels: "No Channels available.",
    emptySkills: "No Skills available.",
    emptyMCP: "No MCP available.",
    emptyModels: "No model providers available.",
    emptyEnvironments: "No environment config available.",
    emptyCronJobs: "No cron jobs available.",
    loadFailed: (message) => `Load failed: ${message}`,
  },
  zh: {
    loading: "加载中...",
    statusEnabled: "启用",
    statusDisabled: "停用",
    statusApplied: "已生效",
    statusPendingRestart: "待重启",
    copyValue: "复制内容",
    fieldID: "ID",
    fieldType: "类型",
    fieldDescription: "描述",
    fieldName: "名称",
    fieldScope: "范围",
    fieldVersion: "版本",
    fieldProviderType: "提供方类型",
    fieldAPIType: "接口类型",
    fieldBaseURL: "基础地址",
    fieldDefaultModel: "默认模型",
    fieldModels: "模型",
    fieldKey: "键",
    fieldValue: "当前值",
    fieldEffectiveValue: "生效值",
    fieldValueSource: "来源",
    fieldApplyMode: "生效方式",
    fieldModule: "模块",
    fieldScheduleMode: "调度模式",
    fieldTimezone: "时区",
    fieldCronExpression: "Cron 表达式",
    fieldInterval: "间隔",
    fieldPrompt: "任务输入",
    fieldRetryLimit: "重试次数",
    emptyChannels: "暂无可用通道。",
    emptySkills: "暂无可用技能。",
    emptyMCP: "暂无 MCP 配置。",
    emptyModels: "暂无模型提供方配置。",
    emptyEnvironments: "暂无环境配置。",
    emptyCronJobs: "暂无定时任务配置。",
    loadFailed: (message) => `加载失败：${message}`,
  },
};

type EnvironmentRouteCopy = {
  loading: string;
  loadFailed: (message: string) => string;
  restartService: string;
  confirmRestart: string;
  cancelRestart: string;
  restarting: string;
  restartingSync: string;
  restartFailed: (message: string) => string;
  restartSuccess: string;
  restartWaitTimeout: string;
  restartConfirm: string;
  restartConfirmDescription: string;
  restartSyncMaster: string;
  restartSyncMasterHint: string;
  lastRestart: string;
  commitHash: string;
  reload: string;
  revealSensitive: string;
  hideSensitive: string;
  saveChanges: string;
  saved: string;
  saveFailed: (message: string) => string;
  noChanges: string;
  auditsTitle: string;
  auditsEmpty: string;
  auditOperator: string;
  auditAt: string;
  auditRequiresRestart: string;
  auditChange: (key: string, oldValue: string, newValue: string, applyMode: string) => string;
  restartNotice: (keys: string) => string;
  currentValue: string;
  defaultValue: string;
  effectiveValue: string;
  valueType: string;
  applyMode: string;
  source: string;
  validation: string;
  details: string;
  pendingRestart: string;
  hotReload: string;
  hidden: string;
  fieldDescription: string;
  fieldModule: string;
  fieldKey: string;
  fieldType: string;
  fieldApplyMode: string;
  fieldSource: string;
  fieldDefault: string;
  fieldCurrent: string;
  fieldEffective: string;
  tags: string;
  statusApplied: string;
  statusPendingRestart: string;
  statusEnabled: string;
  statusDisabled: string;
  sourceDefault: string;
  sourceRuntime: string;
  sourcePersisted: string;
  applyImmediate: string;
  applyRestart: string;
  typeInteger: string;
  typeDuration: string;
  typeString: string;
  typeEnum: string;
  typeUnknown: string;
  validationNone: string;
  copyValue: string;
  emptyEnvironments: string;
};

const ENVIRONMENT_ROUTE_COPY: Record<LegacyShellLanguage, EnvironmentRouteCopy> = {
  en: {
    loading: "Loading environments...",
    loadFailed: (message) => `Load failed: ${message}`,
    restartService: "Restart Service",
    confirmRestart: "Confirm Restart",
    cancelRestart: "Cancel",
    restarting: "Restarting service...",
    restartingSync: "Syncing remote master and restarting service...",
    restartFailed: (message) => `Restart failed: ${message}`,
    restartSuccess: "Service restart completed. The page is now connected to the latest runtime.",
    restartWaitTimeout: "Restart is taking longer than expected. Refresh and retry in a moment.",
    restartConfirm: "Restart the service now?",
    restartConfirmDescription: "The page will reload automatically after the new runtime passes health checks.",
    restartSyncMaster: "Sync remote master changes before restart",
    restartSyncMasterHint: "Recommended. Requires local branch master and a clean tracked working tree.",
    lastRestart: "Last Restart",
    commitHash: "Commit Hash",
    reload: "Reload",
    revealSensitive: "Reveal Sensitive",
    hideSensitive: "Hide Sensitive",
    saveChanges: "Save Changes",
    saved: "Environment configuration saved.",
    saveFailed: (message) => `Save failed: ${message}`,
    noChanges: "No configuration changes.",
    auditsTitle: "Change Audits",
    auditsEmpty: "No environment audits.",
    auditOperator: "Operator",
    auditAt: "Changed At",
    auditRequiresRestart: "Requires Restart",
    auditChange: (key, oldValue, newValue, applyMode) => `${key}: ${oldValue} -> ${newValue} (${applyMode})`,
    restartNotice: (keys) => `Some changes require restart: ${keys}`,
    currentValue: "Configured",
    defaultValue: "Default",
    effectiveValue: "Effective",
    valueType: "Value Type",
    applyMode: "Apply Mode",
    source: "Source",
    validation: "Validation",
    details: "More Details",
    pendingRestart: "Pending Restart",
    hotReload: "Hot Reload",
    hidden: "Hidden value",
    fieldDescription: "Description",
    fieldModule: "Module",
    fieldKey: "Key",
    fieldType: "Type",
    fieldApplyMode: "Apply Mode",
    fieldSource: "Source",
    fieldDefault: "Default",
    fieldCurrent: "Configured",
    fieldEffective: "Effective",
    tags: "Tags",
    statusApplied: "Applied",
    statusPendingRestart: "Pending restart",
    statusEnabled: "Enabled",
    statusDisabled: "Disabled",
    sourceDefault: "Default",
    sourceRuntime: "Runtime",
    sourcePersisted: "Persisted",
    applyImmediate: "Immediate",
    applyRestart: "Restart",
    typeInteger: "Integer",
    typeDuration: "Duration, e.g. 5s / 2m / 1h",
    typeString: "Text",
    typeEnum: "Enumerated option",
    typeUnknown: "Unknown",
    validationNone: "No constraints",
    copyValue: "Copy value",
    emptyEnvironments: "No environment config available.",
  },
  zh: {
    loading: "正在加载环境配置...",
    loadFailed: (message) => `加载失败：${message}`,
    restartService: "重启服务",
    confirmRestart: "确认重启",
    cancelRestart: "取消",
    restarting: "服务正在重启...",
    restartingSync: "正在同步远端 master 并重启服务...",
    restartFailed: (message) => `重启失败：${message}`,
    restartSuccess: "服务重启已完成，当前页面已连接到最新运行实例。",
    restartWaitTimeout: "服务重启时间超出预期，请稍后刷新后重试。",
    restartConfirm: "现在重启服务吗？",
    restartConfirmDescription: "新实例探活通过后，当前页面会自动刷新并重新连接。",
    restartSyncMaster: "重启前同步远端 master 最新改动",
    restartSyncMasterHint: "默认开启。要求当前本地分支为 master，且已跟踪工作区保持干净。",
    lastRestart: "最近重启时间",
    commitHash: "Commit Hash",
    reload: "重新加载",
    revealSensitive: "显示敏感项",
    hideSensitive: "隐藏敏感项",
    saveChanges: "保存变更",
    saved: "环境配置已保存。",
    saveFailed: (message) => `保存失败：${message}`,
    noChanges: "没有配置变更。",
    auditsTitle: "变更审计",
    auditsEmpty: "暂无环境配置审计。",
    auditOperator: "操作人",
    auditAt: "变更时间",
    auditRequiresRestart: "需要重启",
    auditChange: (key, oldValue, newValue, applyMode) => `${key}: ${oldValue} -> ${newValue}（${applyMode}）`,
    restartNotice: (keys) => `以下配置需重启后生效：${keys}`,
    currentValue: "配置值",
    defaultValue: "默认值",
    effectiveValue: "生效值",
    valueType: "值类型",
    applyMode: "生效方式",
    source: "来源",
    validation: "校验规则",
    details: "更多信息",
    pendingRestart: "待重启生效",
    hotReload: "热更新",
    hidden: "隐藏值",
    fieldDescription: "描述",
    fieldModule: "模块",
    fieldKey: "键",
    fieldType: "类型",
    fieldApplyMode: "生效方式",
    fieldSource: "来源",
    fieldDefault: "默认值",
    fieldCurrent: "配置值",
    fieldEffective: "生效值",
    tags: "标签",
    statusApplied: "已生效",
    statusPendingRestart: "待重启",
    statusEnabled: "启用",
    statusDisabled: "停用",
    sourceDefault: "默认值",
    sourceRuntime: "运行时",
    sourcePersisted: "持久化",
    applyImmediate: "即时生效",
    applyRestart: "重启生效",
    typeInteger: "整数",
    typeDuration: "时长，例如 5s / 2m / 1h",
    typeString: "文本",
    typeEnum: "枚举选项",
    typeUnknown: "未知",
    validationNone: "无约束",
    copyValue: "复制内容",
    emptyEnvironments: "暂无环境配置。",
  },
};

type FieldSpec = {
  label: string;
  value: unknown;
  copyable?: boolean;
  mono?: boolean;
  multiline?: boolean;
  preview?: boolean;
  clampLines?: number;
};

type RequestState =
  | { status: "loading"; items: RouteRecord[]; error: string }
  | { status: "ready"; items: RouteRecord[]; error: string }
  | { status: "error"; items: RouteRecord[]; error: string };

type ReactManagedControlRouteBodyProps = {
  route: ReactManagedControlRoute;
  language: LegacyShellLanguage;
};

export { isReactManagedControlRoute } from "../reactManagedRouteContract";

type RouteConfig = {
  path: string;
  empty: (copy: ControlRouteCopy) => string;
  key: (item: RouteRecord) => string;
  title: (item: RouteRecord) => string | null | undefined;
  type: (item: RouteRecord) => string | null | undefined;
  enabled: (item: RouteRecord) => boolean;
  statusLabels?: (copy: ControlRouteCopy) => {
    enabled: string;
    disabled: string;
  };
  fields: (item: RouteRecord, copy: ControlRouteCopy) => FieldSpec[];
};

const ROUTE_CONFIG: Record<ReactManagedControlRoute, RouteConfig> = {
  channels: {
    path: "/api/control/channels",
    empty: (copy) => copy.emptyChannels,
    key: (item) => normalizeText((item as ControlRouteRecord).id),
    title: (item) => (item as ControlRouteRecord).id,
    type: (item) => (item as ControlRouteRecord).type,
    enabled: (item) => Boolean((item as ControlRouteRecord).enabled),
    fields: (item, copy) => [
      { label: copy.fieldID, value: (item as ControlRouteRecord).id, copyable: true, mono: true },
      { label: copy.fieldType, value: (item as ControlRouteRecord).type },
      {
        label: copy.fieldDescription,
        value: (item as ControlRouteRecord).description,
        multiline: true,
        preview: true,
        clampLines: 3,
      },
    ],
  },
  skills: {
    path: "/api/control/skills",
    empty: (copy) => copy.emptySkills,
    key: (item) => normalizeText((item as ControlRouteRecord).id),
    title: (item) => (item as ControlRouteRecord).id,
    type: (item) => (item as ControlRouteRecord).type,
    enabled: (item) => Boolean((item as ControlRouteRecord).enabled),
    fields: (item, copy) => [
      { label: copy.fieldID, value: (item as ControlRouteRecord).id, copyable: true, mono: true },
      { label: copy.fieldType, value: (item as ControlRouteRecord).type },
      { label: copy.fieldName, value: (item as ControlRouteRecord).name },
      { label: copy.fieldScope, value: (item as ControlRouteRecord).scope },
      { label: copy.fieldVersion, value: (item as ControlRouteRecord).version },
    ],
  },
  mcp: {
    path: "/api/control/mcps",
    empty: (copy) => copy.emptyMCP,
    key: (item) => normalizeText((item as ControlRouteRecord).id),
    title: (item) => (item as ControlRouteRecord).id,
    type: (item) => (item as ControlRouteRecord).type,
    enabled: (item) => Boolean((item as ControlRouteRecord).enabled),
    fields: (item, copy) => [
      { label: copy.fieldID, value: (item as ControlRouteRecord).id, copyable: true, mono: true },
      { label: copy.fieldType, value: (item as ControlRouteRecord).type },
      { label: copy.fieldName, value: (item as ControlRouteRecord).name },
      { label: copy.fieldScope, value: (item as ControlRouteRecord).scope },
      { label: copy.fieldVersion, value: (item as ControlRouteRecord).version },
    ],
  },
  models: {
    path: "/api/control/llm/providers",
    empty: (copy) => copy.emptyModels,
    key: (item) => normalizeText((item as LLMProviderRecord).id || (item as LLMProviderRecord).name),
    title: (item) => (item as LLMProviderRecord).name || (item as LLMProviderRecord).id,
    type: (item) => (item as LLMProviderRecord).provider_type || (item as LLMProviderRecord).api_type,
    enabled: (item) => Boolean((item as LLMProviderRecord).is_enabled),
    fields: (item, copy) => [
      { label: copy.fieldID, value: (item as LLMProviderRecord).id, copyable: true, mono: true },
      { label: copy.fieldProviderType, value: (item as LLMProviderRecord).provider_type },
      { label: copy.fieldAPIType, value: (item as LLMProviderRecord).api_type },
      {
        label: copy.fieldBaseURL,
        value: (item as LLMProviderRecord).base_url,
        mono: true,
        preview: true,
        clampLines: 2,
      },
      { label: copy.fieldDefaultModel, value: (item as LLMProviderRecord).default_model },
      { label: copy.fieldModels, value: summarizeProviderModels(item as LLMProviderRecord), multiline: true, preview: true, clampLines: 3 },
    ],
  },
  environments: {
    path: "/api/control/environments",
    empty: (copy) => copy.emptyEnvironments,
    key: (item) => normalizeText((item as EnvironmentRouteRecord).definition?.key),
    title: (item) => (item as EnvironmentRouteRecord).definition?.name || (item as EnvironmentRouteRecord).definition?.key,
    type: (item) => (item as EnvironmentRouteRecord).definition?.module || (item as EnvironmentRouteRecord).definition?.type,
    enabled: (item) => !Boolean((item as EnvironmentRouteRecord).pending_restart),
    statusLabels: (copy) => ({
      enabled: copy.statusApplied,
      disabled: copy.statusPendingRestart,
    }),
    fields: (item, copy) => [
      { label: copy.fieldKey, value: (item as EnvironmentRouteRecord).definition?.key, copyable: true, mono: true },
      { label: copy.fieldModule, value: (item as EnvironmentRouteRecord).definition?.module },
      { label: copy.fieldType, value: (item as EnvironmentRouteRecord).definition?.type },
      { label: copy.fieldApplyMode, value: (item as EnvironmentRouteRecord).definition?.apply_mode },
      { label: copy.fieldValue, value: (item as EnvironmentRouteRecord).value, mono: true },
      { label: copy.fieldEffectiveValue, value: (item as EnvironmentRouteRecord).effective_value, mono: true },
      { label: copy.fieldValueSource, value: (item as EnvironmentRouteRecord).value_source },
      {
        label: copy.fieldDescription,
        value: (item as EnvironmentRouteRecord).definition?.description,
        multiline: true,
        preview: true,
        clampLines: 3,
      },
    ],
  },
  "cron-jobs": {
    path: "/api/control/cron/jobs",
    empty: (copy) => copy.emptyCronJobs,
    key: (item) => normalizeText((item as CronJobRouteRecord).id),
    title: (item) => (item as CronJobRouteRecord).name || (item as CronJobRouteRecord).id,
    type: (item) => (item as CronJobRouteRecord).schedule_mode,
    enabled: (item) => Boolean((item as CronJobRouteRecord).enabled),
    fields: (item, copy) => [
      { label: copy.fieldID, value: (item as CronJobRouteRecord).id, copyable: true, mono: true },
      { label: copy.fieldScheduleMode, value: (item as CronJobRouteRecord).schedule_mode },
      { label: copy.fieldTimezone, value: (item as CronJobRouteRecord).timezone },
      { label: copy.fieldCronExpression, value: (item as CronJobRouteRecord).cron_expression, mono: true },
      { label: copy.fieldInterval, value: (item as CronJobRouteRecord).interval, mono: true },
      {
        label: copy.fieldPrompt,
        value: (item as CronJobRouteRecord).task_config?.input,
        multiline: true,
        preview: true,
        clampLines: 2,
      },
      { label: copy.fieldRetryLimit, value: (item as CronJobRouteRecord).task_config?.retry_limit },
    ],
  },
};

export function ReactManagedControlRouteBody({
  route,
  language,
}: ReactManagedControlRouteBodyProps) {
  if (route === "environments") {
    return <ReactManagedEnvironmentRouteBody language={language} />;
  }

  const copy = CONTROL_ROUTE_COPY[language];
  const routeConfig = ROUTE_CONFIG[route];
  const [state, setState] = useState<RequestState>({
    status: "loading",
    items: [],
    error: "",
  });

  useEffect(() => {
    let disposed = false;

    setState({
      status: "loading",
      items: [],
      error: "",
    });

    void createAPIClient()
      .get<ControlRouteResponse>(routeConfig.path)
      .then((payload) => {
        if (disposed) {
          return;
        }
        setState({
          status: "ready",
          items: Array.isArray(payload?.items) ? payload.items : [],
          error: "",
        });
      })
      .catch((error: unknown) => {
        if (disposed) {
          return;
        }
        setState({
          status: "error",
          items: [],
          error: error instanceof Error ? error.message : "unknown_error",
        });
      });

    return () => {
      disposed = true;
    };
  }, [routeConfig]);

  if (state.status === "loading") {
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (state.status === "error") {
    return <p className="route-error">{copy.loadFailed(state.error)}</p>;
  }

  if (!state.items.length) {
    return <p className="route-empty">{routeConfig.empty(copy)}</p>;
  }

  const statusLabels = routeConfig.statusLabels?.(copy) ?? {
    enabled: copy.statusEnabled,
    disabled: copy.statusDisabled,
  };

  return (
    <>
      {state.items.map((item) => (
        <RouteCard
          key={`${route}-${routeConfig.key(item)}`}
          title={routeConfig.title(item)}
          type={routeConfig.type(item)}
          enabled={routeConfig.enabled(item)}
          statusEnabledLabel={statusLabels.enabled}
          statusDisabledLabel={statusLabels.disabled}
        >
          {routeConfig.fields(item, copy).map((field) => (
            <RouteFieldRow key={`${field.label}-${normalizeText(field.value)}`} copyLabel={copy.copyValue} {...field} />
          ))}
        </RouteCard>
      ))}
    </>
  );
}

function summarizeProviderModels(item: LLMProviderRecord) {
  const models = Array.isArray(item.models) ? item.models : [];
  const enabledModels = models
    .filter((model) => model?.is_enabled)
    .map((model) => String(model?.name || model?.id || "").trim())
    .filter(Boolean);

  if (enabledModels.length) {
    return enabledModels.join(", ");
  }

  const fallbackModels = models
    .map((model) => String(model?.name || model?.id || "").trim())
    .filter(Boolean);

  return fallbackModels.join(", ");
}

function ReactManagedEnvironmentRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = ENVIRONMENT_ROUTE_COPY[language];
  const apiClient = createAPIClient();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [items, setItems] = useState<EnvironmentRouteRecord[]>([]);
  const [audits, setAudits] = useState<EnvironmentAuditRecord[]>([]);
  const [runtimeInfo, setRuntimeInfo] = useState<RuntimeInfoRecord>({});
  const [revealSensitive, setRevealSensitive] = useState(false);
  const [draftValues, setDraftValues] = useState<Record<string, string>>({});
  const [dirtyKeys, setDirtyKeys] = useState<Record<string, boolean>>({});
  const [statusMessage, setStatusMessage] = useState("");
  const [statusKind, setStatusKind] = useState<"success" | "error" | "">("");
  const [restartConfirmOpen, setRestartConfirmOpen] = useState(false);
  const [syncRemoteMaster, setSyncRemoteMaster] = useState(true);
  const restartPollTimerRef = useRef<number | null>(null);

  useEffect(() => {
    const notice = consumeRuntimeRestartNotice();
    if (notice) {
      setStatusMessage(copy.restartSuccess);
      setStatusKind("success");
    }
  }, [copy.restartSuccess]);

  useEffect(() => {
    void reloadEnvironmentState();
    return () => {
      clearRestartPollTimer(restartPollTimerRef);
    };
  }, [revealSensitive]);

  async function reloadEnvironmentState(nextStatusMessage = "", nextStatusKind: "success" | "error" | "" = "") {
    setLoading(true);
    setError("");
    try {
      const query = revealSensitive ? "?reveal_sensitive=true" : "";
      const [environmentPayload, auditPayload, runtimePayload] = await Promise.all([
        apiClient.get<{ items?: EnvironmentRouteRecord[] }>(`/api/control/environments${query}`),
        apiClient.get<{ items?: EnvironmentAuditRecord[] }>(`/api/control/environments/audits${query}`),
        apiClient.get<RuntimeInfoRecord>("/api/control/runtime"),
      ]);
      const nextItems = Array.isArray(environmentPayload?.items) ? environmentPayload.items : [];
      setItems(nextItems);
      setAudits(Array.isArray(auditPayload?.items) ? auditPayload.items : []);
      setRuntimeInfo(runtimePayload || {});
      setDraftValues(buildEnvironmentDrafts(nextItems, revealSensitive));
      setDirtyKeys({});
      setStatusMessage(nextStatusMessage);
      setStatusKind(nextStatusKind);
      setRestartConfirmOpen(false);
    } catch (requestError: unknown) {
      setItems([]);
      setAudits([]);
      setRuntimeInfo({});
      setError(requestError instanceof Error ? requestError.message : "unknown_error");
      setStatusMessage(nextStatusMessage);
      setStatusKind(nextStatusKind);
    } finally {
      setLoading(false);
    }
  }

  async function handleSave() {
    const changedValues = buildChangedEnvironmentValues(items, draftValues, dirtyKeys, revealSensitive);
    const changedKeys = Object.keys(changedValues);
    if (!changedKeys.length) {
      setStatusMessage(copy.noChanges);
      setStatusKind("");
      return;
    }

    try {
      const result = await apiClient.put<{
        needs_restart?: boolean;
        restart_keys?: string[];
      }>("/api/control/environments", {
        operator: "web-console",
        values: changedValues,
      });
      const restartKeys = Array.isArray(result?.restart_keys) ? result.restart_keys.filter(Boolean) : [];
      const nextMessage = restartKeys.length ? copy.restartNotice(restartKeys.join(", ")) : copy.saved;
      await reloadEnvironmentState(nextMessage, "success");
    } catch (saveError: unknown) {
      setStatusMessage(copy.saveFailed(saveError instanceof Error ? saveError.message : "unknown_error"));
      setStatusKind("error");
    }
  }

  async function handleConfirmRestart() {
    const previousRuntime = {
      started_at: normalizeTextValue(runtimeInfo.started_at),
      commit_hash: normalizeTextValue(runtimeInfo.commit_hash),
    };

    try {
      await apiClient.post("/api/control/runtime/restart", {
        sync_remote_master: syncRemoteMaster,
      });
      setStatusMessage(syncRemoteMaster ? copy.restartingSync : copy.restarting);
      setStatusKind("success");
      setRestartConfirmOpen(false);
      scheduleRuntimeRestartPoll({
        apiClient,
        previousRuntime,
        onTimeout: () => {
          setStatusMessage(copy.restartWaitTimeout);
          setStatusKind("error");
        },
        onReady: (nextRuntime) => {
          setRuntimeInfo(nextRuntime);
          persistRuntimeRestartNotice();
          window.location.reload();
        },
        timerRef: restartPollTimerRef,
      });
    } catch (restartError: unknown) {
      setStatusMessage(copy.restartFailed(restartError instanceof Error ? restartError.message : "unknown_error"));
      setStatusKind("error");
    }
  }

  if (loading) {
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (error) {
    return <p className="route-error">{copy.loadFailed(error)}</p>;
  }

  if (!items.length) {
    return <p className="route-empty">{copy.emptyEnvironments}</p>;
  }

  const groupedItems = groupEnvironmentItems(items);

  return (
    <section className="environment-view">
      <div className="route-surface environment-toolbar">
        <div className="environment-toolbar-main">
          <div className="environment-runtime-meta">
            <div className="environment-runtime-meta-item">
              <span>{copy.lastRestart}</span>
              <strong>{formatDateTime(runtimeInfo.started_at)}</strong>
            </div>
            <div className="environment-runtime-meta-item">
              <span>{copy.commitHash}</span>
              <strong>
                <code title={normalizeTextValue(runtimeInfo.commit_hash)}>
                  {shortenCommitHash(runtimeInfo.commit_hash)}
                </code>
              </strong>
            </div>
          </div>
          {statusMessage ? (
            <p className={`environment-status ${statusKind === "error" ? "is-error" : "is-success"}`}>
              {statusMessage}
            </p>
          ) : null}
          {restartConfirmOpen ? (
            <div className="environment-restart-confirm">
              <p className="environment-restart-confirm-copy">{copy.restartConfirm}</p>
              <p className="environment-restart-confirm-hint">{copy.restartConfirmDescription}</p>
              <label className="environment-restart-confirm-option">
                <input
                  aria-label={copy.restartSyncMaster}
                  checked={syncRemoteMaster}
                  type="checkbox"
                  onChange={(event) => setSyncRemoteMaster(event.target.checked)}
                />
                <span>
                  {copy.restartSyncMaster}
                  <small>{copy.restartSyncMasterHint}</small>
                </span>
              </label>
            </div>
          ) : null}
        </div>

        <div className="route-card-actions environment-toolbar-actions">
          <button type="button" onClick={() => setRevealSensitive((current) => !current)}>
            {revealSensitive ? copy.hideSensitive : copy.revealSensitive}
          </button>
          <button type="button" onClick={() => void reloadEnvironmentState()}>
            {copy.reload}
          </button>
          <button type="button" onClick={() => setRestartConfirmOpen((current) => !current)}>
            {copy.restartService}
          </button>
          <button type="button" onClick={() => void handleSave()}>
            {copy.saveChanges}
          </button>
          {restartConfirmOpen ? (
            <>
              <button type="button" onClick={() => void handleConfirmRestart()}>
                {copy.confirmRestart}
              </button>
              <button type="button" onClick={() => setRestartConfirmOpen(false)}>
                {copy.cancelRestart}
              </button>
            </>
          ) : null}
        </div>
      </div>

      <div className="environment-modules">
        {Object.entries(groupedItems).map(([moduleName, moduleItems]) => (
          <section key={moduleName} className="environment-module">
            <h4>{moduleName}</h4>
            <div className="environment-module-grid">
              {moduleItems.map((item) => {
                const key = normalizeTextValue(item.definition?.key);
                const hiddenSensitive = Boolean(item.definition?.sensitive) && !revealSensitive;
                const inputValue = draftValues[key] ?? "";
                return (
                  <RouteCard
                    key={key}
                    title={item.definition?.name || key}
                    type={item.definition?.type}
                    enabled={!item.pending_restart}
                    statusEnabledLabel={copy.statusApplied}
                    statusDisabledLabel={copy.statusPendingRestart}
                    className="environment-item"
                    body={
                      <div className="environment-card-body">
                        <div className="environment-summary">
                          <div className="environment-item-key">
                            <code title={key}>{key}</code>
                            {item.pending_restart ? <span className="environment-pending">{copy.pendingRestart}</span> : null}
                          </div>
                          <div className="environment-description">
                            <span>{copy.fieldDescription}</span>
                            <p className="environment-description-text">{normalizeText(item.definition?.description)}</p>
                          </div>
                          <label className="environment-input-row">
                            <span>{copy.currentValue}</span>
                            {renderEnvironmentInput({
                              copy,
                              hiddenSensitive,
                              item,
                              value: inputValue,
                              onChange: (value) => {
                                setDraftValues((current) => ({
                                  ...current,
                                  [key]: value,
                                }));
                                setDirtyKeys((current) => ({
                                  ...current,
                                  [key]: true,
                                }));
                              },
                            })}
                          </label>
                          <details className="environment-details">
                            <summary>{copy.details}</summary>
                            <div className="environment-details-body">
                              <RouteFieldRow label={copy.fieldKey} value={key} copyLabel={copy.copyValue} copyable mono />
                              <RouteFieldRow label={copy.fieldModule} value={item.definition?.module} copyLabel={copy.copyValue} />
                              <RouteFieldRow label={copy.fieldType} value={formatEnvironmentType(item.definition?.type, copy)} copyLabel={copy.copyValue} />
                              <RouteFieldRow label={copy.fieldApplyMode} value={formatEnvironmentApplyMode(item.definition?.apply_mode, copy)} copyLabel={copy.copyValue} />
                              <RouteFieldRow label={copy.fieldSource} value={formatEnvironmentSource(item.value_source, copy)} copyLabel={copy.copyValue} />
                              <RouteFieldRow label={copy.fieldDefault} value={item.definition?.default_value} copyLabel={copy.copyValue} mono multiline />
                              <RouteFieldRow label={copy.fieldCurrent} value={item.value} copyLabel={copy.copyValue} mono multiline />
                              <RouteFieldRow label={copy.fieldEffective} value={item.effective_value} copyLabel={copy.copyValue} mono multiline />
                              <RouteFieldRow label={copy.validation} value={formatEnvironmentValidation(item, copy)} copyLabel={copy.copyValue} multiline />
                            </div>
                          </details>
                        </div>
                      </div>
                    }
                    footer={
                      <RouteTagSection
                        label={copy.tags}
                        tags={buildEnvironmentTags(item, copy)}
                      />
                    }
                    footerClassName="route-card-footer-spread"
                  />
                );
              })}
            </div>
          </section>
        ))}
      </div>

      <section className="environment-audits">
        <h4>{copy.auditsTitle}</h4>
        <div className="environment-audit-list">
          {audits.length ? (
            audits.map((audit, index) => (
              <RouteCard
                key={`${normalizeTextValue(audit.occurred_at)}-${index}`}
                title={audit.operator || copy.auditOperator}
                type={audit.requires_restart ? copy.statusPendingRestart : copy.statusApplied}
                enabled={!audit.requires_restart}
                statusEnabledLabel={copy.statusApplied}
                statusDisabledLabel={copy.auditRequiresRestart}
                className="environment-audit-item"
              >
                <RouteFieldRow label={copy.auditOperator} value={audit.operator} copyLabel={copy.copyValue} />
                <RouteFieldRow label={copy.auditAt} value={formatDateTime(audit.occurred_at)} copyLabel={copy.copyValue} />
                <div className="environment-audit-changes">
                  <span>{copy.fieldDescription}</span>
                  <ul>
                    {(audit.changes || []).map((change, changeIndex) => (
                      <li key={`${normalizeTextValue(change.key)}-${changeIndex}`}>
                        {copy.auditChange(
                          normalizeText(change.key),
                          normalizeText(change.old_value),
                          normalizeText(change.new_value),
                          formatEnvironmentApplyMode(change.apply_mode, copy),
                        )}
                      </li>
                    ))}
                  </ul>
                </div>
              </RouteCard>
            ))
          ) : (
            <p className="route-empty">{copy.auditsEmpty}</p>
          )}
        </div>
      </section>
    </section>
  );
}

function renderEnvironmentInput({
  copy,
  hiddenSensitive,
  item,
  value,
  onChange,
}: {
  copy: EnvironmentRouteCopy;
  hiddenSensitive: boolean;
  item: EnvironmentRouteRecord;
  value: string;
  onChange: (value: string) => void;
}) {
  const options = Array.isArray(item.definition?.validation?.allowed)
    ? item.definition?.validation?.allowed?.filter(Boolean) || []
    : [];

  if (String(item.definition?.type || "").toLowerCase() === "enum" && options.length) {
    return (
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        {options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    );
  }

  return (
    <input
      placeholder={hiddenSensitive ? copy.hidden : normalizeTextValue(item.definition?.default_value)}
      type={hiddenSensitive ? "password" : "text"}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}

function buildEnvironmentDrafts(items: EnvironmentRouteRecord[], revealSensitive: boolean) {
  return items.reduce<Record<string, string>>((accumulator, item) => {
    const key = normalizeTextValue(item.definition?.key);
    if (!key) {
      return accumulator;
    }
    accumulator[key] =
      Boolean(item.definition?.sensitive) && Boolean(item.masked) && !revealSensitive
        ? ""
        : normalizeTextValue(item.value);
    return accumulator;
  }, {});
}

function buildChangedEnvironmentValues(
  items: EnvironmentRouteRecord[],
  draftValues: Record<string, string>,
  dirtyKeys: Record<string, boolean>,
  revealSensitive: boolean,
) {
  return items.reduce<Record<string, string>>((accumulator, item) => {
    const key = normalizeTextValue(item.definition?.key);
    if (!key) {
      return accumulator;
    }
    const nextValue = draftValues[key] ?? "";
    const sensitiveMasked = Boolean(item.definition?.sensitive) && Boolean(item.masked) && !revealSensitive;
    if (sensitiveMasked) {
      if (dirtyKeys[key]) {
        accumulator[key] = nextValue;
      }
      return accumulator;
    }
    if (normalizeTextValue(item.value) !== nextValue.trim()) {
      accumulator[key] = nextValue;
    }
    return accumulator;
  }, {});
}

function groupEnvironmentItems(items: EnvironmentRouteRecord[]) {
  return items.reduce<Record<string, EnvironmentRouteRecord[]>>((accumulator, item) => {
    const moduleName = normalizeText(item.definition?.module);
    if (!accumulator[moduleName]) {
      accumulator[moduleName] = [];
    }
    accumulator[moduleName].push(item);
    return accumulator;
  }, {});
}

function buildEnvironmentTags(item: EnvironmentRouteRecord, copy: EnvironmentRouteCopy) {
  return [
    formatEnvironmentApplyMode(item.definition?.apply_mode, copy),
    formatEnvironmentSource(item.value_source, copy),
    item.definition?.hot_reload ? copy.hotReload : copy.statusDisabled,
  ];
}

function formatEnvironmentApplyMode(value: unknown, copy: EnvironmentRouteCopy) {
  const normalized = normalizeTextValue(value).toLowerCase();
  if (normalized === "immediate") {
    return copy.applyImmediate;
  }
  if (normalized === "restart") {
    return copy.applyRestart;
  }
  return normalizeText(value);
}

function formatEnvironmentSource(value: unknown, copy: EnvironmentRouteCopy) {
  const normalized = normalizeTextValue(value).toLowerCase();
  if (normalized === "default") {
    return copy.sourceDefault;
  }
  if (normalized === "runtime") {
    return copy.sourceRuntime;
  }
  if (normalized === "persisted" || normalized === "control") {
    return copy.sourcePersisted;
  }
  return normalizeText(value);
}

function formatEnvironmentType(value: unknown, copy: EnvironmentRouteCopy) {
  const normalized = normalizeTextValue(value).toLowerCase();
  if (normalized === "integer") {
    return copy.typeInteger;
  }
  if (normalized === "duration") {
    return copy.typeDuration;
  }
  if (normalized === "string") {
    return copy.typeString;
  }
  if (normalized === "enum") {
    return copy.typeEnum;
  }
  if (!normalized) {
    return copy.typeUnknown;
  }
  return normalizeText(value);
}

function formatEnvironmentValidation(item: EnvironmentRouteRecord, copy: EnvironmentRouteCopy) {
  const validation = item.definition?.validation;
  if (!validation) {
    return copy.validationNone;
  }
  const parts: string[] = [];
  if (validation.required) {
    parts.push("required");
  }
  if (normalizeTextValue(validation.min)) {
    parts.push(`min=${normalizeTextValue(validation.min)}`);
  }
  if (normalizeTextValue(validation.max)) {
    parts.push(`max=${normalizeTextValue(validation.max)}`);
  }
  const allowed = Array.isArray(validation.allowed)
    ? validation.allowed.map((option) => normalizeTextValue(option)).filter(Boolean)
    : [];
  if (allowed.length) {
    parts.push(`allowed=${allowed.join(", ")}`);
  }
  return parts.length ? parts.join(" | ") : copy.validationNone;
}

function shortenCommitHash(value: unknown) {
  const normalized = normalizeTextValue(value);
  if (!normalized) {
    return "-";
  }
  if (normalized.length <= 16) {
    return normalized;
  }
  return `${normalized.slice(0, 12)}...`;
}

function normalizeTextValue(value: unknown) {
  const normalized = String(value || "").trim();
  return normalized;
}

function persistRuntimeRestartNotice() {
  try {
    window.sessionStorage.setItem(
      "alter0.runtime-restart-notice",
      JSON.stringify({ status: "success", created_at: Date.now() }),
    );
  } catch {
    // ignore storage failures
  }
}

function consumeRuntimeRestartNotice() {
  try {
    const raw = window.sessionStorage.getItem("alter0.runtime-restart-notice");
    window.sessionStorage.removeItem("alter0.runtime-restart-notice");
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as { status?: string } | null;
    return parsed?.status === "success" ? parsed : null;
  } catch {
    return null;
  }
}

function clearRestartPollTimer(timerRef: { current: number | null }) {
  if (timerRef.current) {
    window.clearTimeout(timerRef.current);
    timerRef.current = null;
  }
}

function scheduleRuntimeRestartPoll({
  apiClient,
  previousRuntime,
  onTimeout,
  onReady,
  timerRef,
}: {
  apiClient: ReturnType<typeof createAPIClient>;
  previousRuntime: RuntimeInfoRecord;
  onTimeout: () => void;
  onReady: (nextRuntime: RuntimeInfoRecord) => void;
  timerRef: { current: number | null };
}) {
  clearRestartPollTimer(timerRef);
  const deadline = Date.now() + 120000;

  const poll = async () => {
    try {
      const nextRuntime = await apiClient.get<RuntimeInfoRecord>("/api/control/runtime");
      const nextStartedAt = normalizeTextValue(nextRuntime.started_at);
      const nextCommitHash = normalizeTextValue(nextRuntime.commit_hash);
      if (
        nextStartedAt !== normalizeTextValue(previousRuntime.started_at) ||
        nextCommitHash !== normalizeTextValue(previousRuntime.commit_hash)
      ) {
        onReady(nextRuntime);
        return;
      }
    } catch {
      // runtime may be temporarily unavailable while restarting
    }

    if (Date.now() >= deadline) {
      clearRestartPollTimer(timerRef);
      onTimeout();
      return;
    }

    timerRef.current = window.setTimeout(() => {
      void poll();
    }, 1500);
  };

  timerRef.current = window.setTimeout(() => {
    void poll();
  }, 1500);
}
