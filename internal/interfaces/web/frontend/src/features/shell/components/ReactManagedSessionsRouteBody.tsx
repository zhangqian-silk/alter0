import { useEffect, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTime } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  normalizeText,
  RouteCard,
  RouteFieldRow,
  RouteTagSection,
} from "./RouteBodyPrimitives";

const SESSION_ROUTE_FILTERS_STORAGE_KEY = "alter0.web.sessions.route-filters.v1";

type SessionRouteFilters = {
  triggerType: string;
  channelType: string;
  channelID: string;
  messageID: string;
  jobID: string;
};

type SessionRouteRecord = {
  session_id?: string;
  channel_type?: string;
  channel_id?: string;
  last_message_id?: string;
  updated_at?: string;
  last_message_at?: string;
  created_at?: string;
  started_at?: string;
  message_count?: number;
  trigger_type?: string;
  job_id?: string;
  job_name?: string;
  fired_at?: string;
};

type SessionRouteResponse = {
  items?: SessionRouteRecord[];
};

type SessionRouteCopy = {
  loading: string;
  loadFailed: (message: string) => string;
  statusEnabled: string;
  statusDisabled: string;
  copyValue: string;
  title: string;
  empty: string;
  viewDetail: string;
  filterTriggerType: string;
  filterChannelType: string;
  filterChannelID: string;
  filterMessageID: string;
  filterJobID: string;
  apply: string;
  reset: string;
  triggerUser: string;
  triggerCron: string;
  triggerSystem: string;
  channelCLI: string;
  channelWeb: string;
  channelScheduler: string;
  fieldID: string;
  fieldChannelType: string;
  fieldLastMessageID: string;
  fieldUpdated: string;
  fieldChannelID: string;
  fieldCreated: string;
  fieldMessages: string;
  fieldTriggerType: string;
  fieldJobID: string;
  fieldJobName: string;
  fieldFiredAt: string;
  fieldTags: string;
};

const SESSION_ROUTE_COPY: Record<LegacyShellLanguage, SessionRouteCopy> = {
  en: {
    loading: "Loading...",
    loadFailed: (message) => `Load failed: ${message}`,
    statusEnabled: "Enabled",
    statusDisabled: "Disabled",
    copyValue: "Copy value",
    title: "Sessions",
    empty: "No sessions found.",
    viewDetail: "View Detail",
    filterTriggerType: "Trigger Type",
    filterChannelType: "Channel Type",
    filterChannelID: "Channel ID",
    filterMessageID: "Message ID",
    filterJobID: "Job ID",
    apply: "Apply",
    reset: "Reset",
    triggerUser: "User",
    triggerCron: "Cron",
    triggerSystem: "System",
    channelCLI: "CLI",
    channelWeb: "Web",
    channelScheduler: "Scheduler",
    fieldID: "ID",
    fieldChannelType: "Channel Type",
    fieldLastMessageID: "Last Message ID",
    fieldUpdated: "Updated",
    fieldChannelID: "Channel ID",
    fieldCreated: "Created",
    fieldMessages: "Messages",
    fieldTriggerType: "Trigger Type",
    fieldJobID: "Job ID",
    fieldJobName: "Job Name",
    fieldFiredAt: "Fired At",
    fieldTags: "Tags",
  },
  zh: {
    loading: "加载中...",
    loadFailed: (message) => `加载失败：${message}`,
    statusEnabled: "启用",
    statusDisabled: "停用",
    copyValue: "复制内容",
    title: "会话列表",
    empty: "暂无会话记录。",
    viewDetail: "查看详情",
    filterTriggerType: "触发类型",
    filterChannelType: "通道类型",
    filterChannelID: "通道 ID",
    filterMessageID: "消息 ID",
    filterJobID: "任务 ID",
    apply: "应用",
    reset: "重置",
    triggerUser: "用户触发",
    triggerCron: "定时触发",
    triggerSystem: "系统触发",
    channelCLI: "CLI",
    channelWeb: "Web",
    channelScheduler: "Scheduler",
    fieldID: "ID",
    fieldChannelType: "通道类型",
    fieldLastMessageID: "最近消息 ID",
    fieldUpdated: "更新时间",
    fieldChannelID: "通道 ID",
    fieldCreated: "创建时间",
    fieldMessages: "消息数",
    fieldTriggerType: "触发类型",
    fieldJobID: "作业 ID",
    fieldJobName: "作业名称",
    fieldFiredAt: "触发时间",
    fieldTags: "标签",
  },
};

type RequestState =
  | { status: "loading"; items: SessionRouteRecord[]; error: string }
  | { status: "ready"; items: SessionRouteRecord[]; error: string }
  | { status: "error"; items: SessionRouteRecord[]; error: string };

const DEFAULT_FILTERS: SessionRouteFilters = {
  triggerType: "",
  channelType: "",
  channelID: "",
  messageID: "",
  jobID: "",
};

export function ReactManagedSessionsRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = SESSION_ROUTE_COPY[language];
  const [draftFilters, setDraftFilters] = useState<SessionRouteFilters>(() => loadSessionRouteFilters());
  const [activeFilters, setActiveFilters] = useState<SessionRouteFilters>(() => loadSessionRouteFilters());
  const [state, setState] = useState<RequestState>({
    status: "loading",
    items: [],
    error: "",
  });

  useEffect(() => {
    let disposed = false;
    persistSessionRouteFilters(activeFilters);
    setState({
      status: "loading",
      items: [],
      error: "",
    });

    void createAPIClient()
      .get<SessionRouteResponse>(buildSessionListQuery(activeFilters))
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
  }, [activeFilters]);

  return (
    <section className="session-history-view">
      <form
        className="task-filter-form page-filter-form page-filter-grid-4 session-filter-form"
        onSubmit={(event) => {
          event.preventDefault();
          setActiveFilters(normalizeSessionRouteFilters(draftFilters));
        }}
      >
        <label>
          <span>{copy.filterTriggerType}</span>
          <select
            name="trigger_type"
            value={draftFilters.triggerType}
            onChange={(event) => {
              setDraftFilters((current) => ({
                ...current,
                triggerType: normalizeFilterValue(event.target.value),
              }));
            }}
          >
            <option value="">-</option>
            <option value="user">{copy.triggerUser}</option>
            <option value="cron">{copy.triggerCron}</option>
            <option value="system">{copy.triggerSystem}</option>
          </select>
        </label>
        <label>
          <span>{copy.filterChannelType}</span>
          <select
            name="channel_type"
            value={draftFilters.channelType}
            onChange={(event) => {
              setDraftFilters((current) => ({
                ...current,
                channelType: normalizeFilterValue(event.target.value),
              }));
            }}
          >
            <option value="">-</option>
            <option value="cli">{copy.channelCLI}</option>
            <option value="web">{copy.channelWeb}</option>
            <option value="scheduler">{copy.channelScheduler}</option>
          </select>
        </label>
        <label>
          <span>{copy.filterChannelID}</span>
          <input
            type="text"
            name="channel_id"
            placeholder="web-default"
            value={draftFilters.channelID}
            onChange={(event) => {
              setDraftFilters((current) => ({
                ...current,
                channelID: event.target.value,
              }));
            }}
          />
        </label>
        <label>
          <span>{copy.filterMessageID}</span>
          <input
            type="text"
            name="message_id"
            placeholder="msg-123"
            value={draftFilters.messageID}
            onChange={(event) => {
              setDraftFilters((current) => ({
                ...current,
                messageID: event.target.value,
              }));
            }}
          />
        </label>
        <label>
          <span>{copy.filterJobID}</span>
          <input
            type="text"
            name="job_id"
            placeholder="job-daily-report"
            value={draftFilters.jobID}
            onChange={(event) => {
              setDraftFilters((current) => ({
                ...current,
                jobID: event.target.value,
              }));
            }}
          />
        </label>
        <div className="task-filter-actions session-filter-actions">
          <button type="submit">{copy.apply}</button>
          <button
            type="button"
            onClick={() => {
              setDraftFilters(DEFAULT_FILTERS);
              setActiveFilters(DEFAULT_FILTERS);
            }}
          >
            {copy.reset}
          </button>
        </div>
      </form>

      <div className="task-summary-list">
        {state.status === "loading" ? <p className="route-loading">{copy.loading}</p> : null}
        {state.status === "error" ? <p className="route-error">{copy.loadFailed(state.error)}</p> : null}
        {state.status === "ready" && !state.items.length ? (
          <p className="route-empty">{copy.empty}</p>
        ) : null}
        {state.status === "ready"
          ? state.items.map((item) => (
              <SessionRouteCard key={normalizeText(item.session_id)} item={item} copy={copy} />
            ))
          : null}
      </div>
    </section>
  );
}

function SessionRouteCard({
  item,
  copy,
}: {
  item: SessionRouteRecord;
  copy: SessionRouteCopy;
}) {
  const sessionID = typeof item.session_id === "string" ? item.session_id : "";
  const channelType = typeof item.channel_type === "string" ? item.channel_type : "";
  const channelID = typeof item.channel_id === "string" ? item.channel_id : "";
  const lastMessageID = typeof item.last_message_id === "string" ? item.last_message_id : "";
  const updatedAt =
    typeof item.updated_at === "string" && item.updated_at.trim()
      ? item.updated_at
      : item.last_message_at;
  const createdAt =
    typeof item.created_at === "string" && item.created_at.trim()
      ? item.created_at
      : item.started_at;
  const messageCount = Number(item.message_count || 0);
  const triggerType = typeof item.trigger_type === "string" ? item.trigger_type : "";
  const jobID = typeof item.job_id === "string" ? item.job_id : "";
  const jobName = typeof item.job_name === "string" ? item.job_name : "";
  const firedAt = typeof item.fired_at === "string" ? item.fired_at : "";
  const tags = [formatTriggerType(triggerType, copy), formatChannelType(channelType, copy)];
  if (jobName) {
    tags.push(jobName);
  }

  return (
    <RouteCard
      title={sessionID || copy.title}
      type="session"
      enabled
      statusEnabledLabel={copy.statusEnabled}
      statusDisabledLabel={copy.statusDisabled}
      className="session-route-card"
      body={
        <details className="session-route-detail">
          <summary>{copy.viewDetail}</summary>
          <div className="route-meta">
            <RouteFieldRow label={copy.fieldChannelID} value={channelID} copyLabel={copy.copyValue} copyable mono />
            <RouteFieldRow label={copy.fieldCreated} value={formatDateTime(createdAt)} copyLabel={copy.copyValue} />
            <RouteFieldRow label={copy.fieldMessages} value={messageCount} copyLabel={copy.copyValue} />
            <RouteFieldRow label={copy.fieldTriggerType} value={formatTriggerType(triggerType, copy)} copyLabel={copy.copyValue} />
            <RouteFieldRow label={copy.fieldJobID} value={jobID} copyLabel={copy.copyValue} copyable mono />
            <RouteFieldRow label={copy.fieldJobName} value={jobName} copyLabel={copy.copyValue} />
            <RouteFieldRow label={copy.fieldFiredAt} value={formatDateTime(firedAt)} copyLabel={copy.copyValue} />
          </div>
        </details>
      }
      footer={<RouteTagSection label={copy.fieldTags} tags={tags} />}
    >
      <RouteFieldRow label={copy.fieldID} value={sessionID} copyLabel={copy.copyValue} copyable mono />
      <RouteFieldRow label={copy.fieldChannelType} value={formatChannelType(channelType, copy)} copyLabel={copy.copyValue} />
      <RouteFieldRow label={copy.fieldLastMessageID} value={lastMessageID} copyLabel={copy.copyValue} copyable mono />
      <RouteFieldRow label={copy.fieldUpdated} value={formatDateTime(updatedAt)} copyLabel={copy.copyValue} />
    </RouteCard>
  );
}

function loadSessionRouteFilters(): SessionRouteFilters {
  const raw = window.sessionStorage.getItem(SESSION_ROUTE_FILTERS_STORAGE_KEY);
  if (!raw) {
    return DEFAULT_FILTERS;
  }
  try {
    return normalizeSessionRouteFilters(JSON.parse(raw));
  } catch {
    return DEFAULT_FILTERS;
  }
}

function persistSessionRouteFilters(filters: SessionRouteFilters) {
  window.sessionStorage.setItem(SESSION_ROUTE_FILTERS_STORAGE_KEY, JSON.stringify(filters));
}

function normalizeSessionRouteFilters(filters: Partial<SessionRouteFilters>): SessionRouteFilters {
  return {
    triggerType: normalizeFilterValue(filters.triggerType),
    channelType: normalizeFilterValue(filters.channelType),
    channelID: String(filters.channelID || "").trim(),
    messageID: String(filters.messageID || "").trim(),
    jobID: String(filters.jobID || "").trim(),
  };
}

function normalizeFilterValue(value: unknown) {
  return String(value || "").trim().toLowerCase();
}

function buildSessionListQuery(filters: SessionRouteFilters) {
  const params = new URLSearchParams();
  params.set("page", "1");
  params.set("page_size", "50");
  if (filters.triggerType) {
    params.set("trigger_type", filters.triggerType);
  }
  if (filters.channelType) {
    params.set("channel_type", filters.channelType);
  }
  if (filters.channelID) {
    params.set("channel_id", filters.channelID);
  }
  if (filters.messageID) {
    params.set("message_id", filters.messageID);
  }
  if (filters.jobID) {
    params.set("job_id", filters.jobID);
  }
  return `/api/sessions?${params.toString()}`;
}

function formatTriggerType(value: string, copy: SessionRouteCopy) {
  const normalized = normalizeText(value).toLowerCase();
  if (normalized === "user") {
    return copy.triggerUser;
  }
  if (normalized === "cron") {
    return copy.triggerCron;
  }
  if (normalized === "system") {
    return copy.triggerSystem;
  }
  return normalizeText(value);
}

function formatChannelType(value: string, copy: SessionRouteCopy) {
  const normalized = normalizeText(value).toLowerCase();
  if (normalized === "cli") {
    return copy.channelCLI;
  }
  if (normalized === "web") {
    return copy.channelWeb;
  }
  if (normalized === "scheduler") {
    return copy.channelScheduler;
  }
  return normalizeText(value);
}
