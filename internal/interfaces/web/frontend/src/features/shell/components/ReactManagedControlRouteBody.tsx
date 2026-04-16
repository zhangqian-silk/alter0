import { useEffect, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  isReactManagedControlRoute,
  type ReactManagedControlRoute,
} from "../reactManagedRouteContract";
import {
  normalizeText,
  RouteCard,
  RouteFieldRow,
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
  };
  value?: string;
  effective_value?: string;
  value_source?: string;
  pending_restart?: boolean;
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

type FieldSpec = {
  label: string;
  value: string | null | undefined;
  copyable?: boolean;
  mono?: boolean;
  multiline?: boolean;
  preview?: boolean;
  clampLines?: number;
};

type RequestState =
  | { status: "loading"; items: ControlRouteRecord[]; error: string }
  | { status: "ready"; items: ControlRouteRecord[]; error: string }
  | { status: "error"; items: ControlRouteRecord[]; error: string };

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
  const copy = CONTROL_ROUTE_COPY[language];
  const routeConfig = ROUTE_CONFIG[route];
  const [selectedRowKey, setSelectedRowKey] = useState("");
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

  useEffect(() => {
    if (route !== "environments") {
      return;
    }
    const items = state.items;
    if (!items.length) {
      if (selectedRowKey) {
        setSelectedRowKey("");
      }
      return;
    }
    const hasSelected = items.some((item) => routeConfig.key(item) === selectedRowKey);
    if (!hasSelected) {
      setSelectedRowKey(routeConfig.key(items[0]));
    }
  }, [route, routeConfig, selectedRowKey, state.items]);

  if (state.status === "loading") {
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (state.status === "error") {
    return <p className="route-error">{copy.loadFailed(state.error)}</p>;
  }

  if (!state.items.length) {
    return <p className="route-empty">{routeConfig.empty(copy)}</p>;
  }

  if (route === "environments") {
    const selectedItem =
      state.items.find((item) => routeConfig.key(item) === selectedRowKey) ??
      state.items[0] ??
      null;
    return (
      <EnvironmentTableView
        copy={copy}
        items={state.items as EnvironmentRouteRecord[]}
        selectedItem={selectedItem as EnvironmentRouteRecord | null}
        selectedRowKey={selectedRowKey}
        onSelect={setSelectedRowKey}
      />
    );
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

function EnvironmentTableView({
  copy,
  items,
  selectedItem,
  selectedRowKey,
  onSelect,
}: {
  copy: ControlRouteCopy;
  items: EnvironmentRouteRecord[];
  selectedItem: EnvironmentRouteRecord | null;
  selectedRowKey: string;
  onSelect: (key: string) => void;
}) {
  return (
    <section className="route-master-detail route-master-detail-environments">
      <div className="route-data-table-wrap">
        <table className="route-data-table" aria-label="Environment Config">
          <thead>
            <tr>
              <th>Status</th>
              <th>Key</th>
              <th>Value</th>
              <th>Type</th>
              <th>Module</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => {
              const key = normalizeText(item.definition?.key);
              const selected = key === selectedRowKey;
              const pendingRestart = Boolean(item.pending_restart);
              return (
                <tr
                  key={key}
                  className={selected ? "is-active" : undefined}
                  aria-selected={selected ? "true" : "false"}
                >
                  <td>
                    <span className={pendingRestart ? "status-badge disabled" : "status-badge"}>
                      <span className="status-dot"></span>
                      <span>{pendingRestart ? copy.statusPendingRestart : copy.statusApplied}</span>
                    </span>
                  </td>
                  <td>
                    <button
                      className="route-table-select"
                      type="button"
                      onClick={() => onSelect(key)}
                    >
                      {normalizeText(item.definition?.name || item.definition?.key)}
                    </button>
                    <span className="route-table-subtext">{key}</span>
                  </td>
                  <td>
                    <code>{normalizeText(item.effective_value || item.value)}</code>
                  </td>
                  <td>{normalizeText(item.definition?.type)}</td>
                  <td>{normalizeText(item.definition?.module)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <aside className="route-detail-panel environment-detail-panel">
        {selectedItem ? (
          <>
            <div className="route-detail-panel-head">
              <h4>{normalizeText(selectedItem.definition?.name || selectedItem.definition?.key)}</h4>
              <span className={selectedItem.pending_restart ? "status-badge disabled" : "status-badge"}>
                <span className="status-dot"></span>
                <span>{selectedItem.pending_restart ? copy.statusPendingRestart : copy.statusApplied}</span>
              </span>
            </div>
            <div className="route-detail-panel-grid">
              <RouteFieldRow label={copy.fieldKey} value={selectedItem.definition?.key} copyLabel={copy.copyValue} copyable mono />
              <RouteFieldRow label={copy.fieldModule} value={selectedItem.definition?.module} copyLabel={copy.copyValue} />
              <RouteFieldRow label={copy.fieldType} value={selectedItem.definition?.type} copyLabel={copy.copyValue} />
              <RouteFieldRow label={copy.fieldApplyMode} value={selectedItem.definition?.apply_mode} copyLabel={copy.copyValue} />
              <RouteFieldRow label={copy.fieldValue} value={selectedItem.value} copyLabel={copy.copyValue} mono multiline />
              <RouteFieldRow label={copy.fieldEffectiveValue} value={selectedItem.effective_value} copyLabel={copy.copyValue} mono multiline />
              <RouteFieldRow label={copy.fieldValueSource} value={selectedItem.value_source} copyLabel={copy.copyValue} />
            </div>
            <section className="route-detail-note">
              <h5>{copy.fieldDescription}</h5>
              <p>{normalizeText(selectedItem.definition?.description)}</p>
            </section>
          </>
        ) : null}
      </aside>
    </section>
  );
}
