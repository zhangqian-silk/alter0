import { useEffect, useRef, useState, type MutableRefObject } from "react";
import { APIClientError, createAPIClient } from "../../../shared/api/client";
import { formatDateTime } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { normalizeText, RouteTagSection } from "./RouteBodyPrimitives";

const TASK_ROUTE_FILTERS_STORAGE_KEY = "alter0.web.tasks.route-filters.v1";
const TASK_ROUTE_PAGE_SIZE = 20;
const TASK_LOG_PAGE_SIZE = 200;

type TaskRouteFilters = {
  sessionID: string;
  status: string;
  triggerType: string;
  channelType: string;
  channelID: string;
  messageID: string;
  sourceMessageID: string;
  startAt: string;
  endAt: string;
};

type TaskRouteListItem = {
  task_id?: string;
  session_id?: string;
  source_message_id?: string;
  status?: string;
  progress?: number;
  trigger_type?: string;
  channel_type?: string;
  channel_id?: string;
  updated_at?: string;
  last_heartbeat_at?: string;
  timeout_at?: string;
  job_id?: string;
};

type TaskRoutePagination = {
  page?: number;
  total?: number;
  has_next?: boolean;
};

type TaskRouteListResponse = {
  items?: TaskRouteListItem[];
  pagination?: TaskRoutePagination;
};

type TaskControlActionState = {
  enabled?: boolean;
  reason?: string;
};

type TaskControlView = {
  task?: {
    id?: string;
    session_id?: string;
    source_message_id?: string;
    message_id?: string;
    task_type?: string;
    status?: string;
    phase?: string;
    progress?: number;
    queue_position?: number;
    queue_wait_ms?: number;
    retry_count?: number;
    accepted_at?: string;
    created_at?: string;
    updated_at?: string;
    started_at?: string;
    finished_at?: string;
    last_heartbeat_at?: string;
    timeout_at?: string;
    request_content?: string;
    error_code?: string;
    error_message?: string;
    result?: {
      output?: string;
    };
  };
  source?: {
    trigger_type?: string;
    channel_type?: string;
    channel_id?: string;
    correlation_id?: string;
    job_id?: string;
    job_name?: string;
    fired_at?: string;
  };
  actions?: {
    retry?: TaskControlActionState;
    cancel?: TaskControlActionState;
  };
  link?: {
    task_id?: string;
    session_id?: string;
    terminal_session_id?: string;
    request_message_id?: string;
    result_message_id?: string;
    task_detail_path?: string;
    session_tasks_path?: string;
    session_messages_path?: string;
  };
};

type TaskLogItem = {
  seq?: number;
  stage?: string;
  message?: string;
};

type TaskLogPage = {
  items?: TaskLogItem[];
  has_more?: boolean;
  next_cursor?: number;
};

type TaskTerminalInputResponse = {
  task_id?: string;
  anchor_task_id?: string;
  error?: string;
};

type TaskRouteCopy = {
  loading: string;
  loadFailed: (message: string) => string;
  empty: string;
  emptyHint: string;
  selectTaskHint: string;
  applying: string;
  copyTaskID: string;
  openDetail: string;
  close: string;
  progress: string;
  pageLabel: string;
  nextPage: string;
  taskDetail: string;
  detailRuntime: string;
  detailIdentifiers: string;
  quickActions: string;
  quickTimeline: string;
  quickInterview: string;
  logsTitle: string;
  logsEmpty: string;
  logsStreaming: string;
  logsDone: string;
  logsDisconnected: string;
  reconnect: string;
  replay: string;
  terminalTitle: string;
  terminalInputPlaceholder: string;
  terminalSend: string;
  terminalSending: string;
  terminalHint: string;
  terminalFollowupNote: string;
  resultTitle: string;
  retry: string;
  cancel: string;
  retryTip: string;
  replayTip: string;
  statusQueued: string;
  statusRunning: string;
  statusSuccess: string;
  statusFailed: string;
  statusCanceled: string;
  triggerUser: string;
  triggerCron: string;
  triggerSystem: string;
  channelCLI: string;
  channelWeb: string;
  channelScheduler: string;
  filterSession: string;
  filterStatus: string;
  filterTriggerType: string;
  filterChannelType: string;
  filterChannelID: string;
  filterMessageID: string;
  filterSourceMessageID: string;
  filterStartAt: string;
  filterEndAt: string;
  filterAdvancedShow: string;
  filterAdvancedHide: string;
  apply: string;
  reset: string;
  fieldID: string;
  fieldSession: string;
  fieldTaskType: string;
  fieldPhase: string;
  fieldRetryCount: string;
  fieldQueuePosition: string;
  fieldQueueWaitMS: string;
  fieldAcceptedAt: string;
  fieldStartedAt: string;
  fieldCreated: string;
  fieldUpdated: string;
  fieldLastHeartbeatAt: string;
  fieldTimeoutAt: string;
  fieldFinished: string;
  fieldTriggerType: string;
  fieldChannelType: string;
  fieldChannelID: string;
  fieldCorrelationID: string;
  fieldSourceMessage: string;
  fieldMessageID: string;
  fieldRequestMessageID: string;
  fieldResultMessageID: string;
  fieldJobID: string;
  fieldJobName: string;
  fieldFiredAt: string;
  fieldTags: string;
  fieldError: string;
  fieldDetailAPI: string;
  fieldMessages: string;
  unknownError: string;
};

type TaskRouteRequestState =
  | { status: "loading"; items: TaskRouteListItem[]; pagination: TaskRoutePagination; error: string }
  | { status: "ready"; items: TaskRouteListItem[]; pagination: TaskRoutePagination; error: string }
  | { status: "error"; items: TaskRouteListItem[]; pagination: TaskRoutePagination; error: string };

type TaskDetailState =
  | { status: "idle"; view: null; error: string }
  | { status: "loading"; view: null; error: string }
  | { status: "ready"; view: TaskControlView; error: string }
  | { status: "error"; view: null; error: string };

type TaskLogState = {
  items: TaskLogItem[];
  status: string;
  cursor: number;
};

const DEFAULT_FILTERS: TaskRouteFilters = {
  sessionID: "",
  status: "",
  triggerType: "",
  channelType: "",
  channelID: "",
  messageID: "",
  sourceMessageID: "",
  startAt: "",
  endAt: "",
};

const TASK_ROUTE_COPY: Record<LegacyShellLanguage, TaskRouteCopy> = {
  en: {
    loading: "Loading...",
    loadFailed: (message) => `Load failed: ${message}`,
    empty: "No tasks found.",
    emptyHint: "Try adjusting filters or check back in a moment.",
    selectTaskHint: "Select a task from the list to inspect runtime detail.",
    applying: "Applying filters...",
    copyTaskID: "Copy Task ID",
    openDetail: "Open Drawer",
    close: "Close",
    progress: "Progress",
    pageLabel: "Page",
    nextPage: "Next Page",
    taskDetail: "Task Detail",
    detailRuntime: "Runtime Details",
    detailIdentifiers: "Identifiers",
    quickActions: "Quick Actions",
    quickTimeline: "Generate timeline version",
    quickInterview: "Generate interview brief",
    logsTitle: "Execution Logs",
    logsEmpty: "No execution logs.",
    logsStreaming: "Streaming latest logs...",
    logsDone: "Log stream completed.",
    logsDisconnected: "Log stream disconnected. You can reconnect.",
    reconnect: "Reconnect",
    replay: "Replay",
    terminalTitle: "Terminal",
    terminalInputPlaceholder: "Type command or prompt...",
    terminalSend: "Send",
    terminalSending: "Sending...",
    terminalHint: "Supports follow-up interaction in the current terminal session.",
    terminalFollowupNote: "Each Send stays in the same Codex session thread.",
    resultTitle: "Result Output",
    retry: "Retry",
    cancel: "Cancel",
    retryTip: "Retry: run the task again from scratch.",
    replayTip: "Replay: reload and replay current task logs only.",
    statusQueued: "Queued",
    statusRunning: "Running",
    statusSuccess: "Success",
    statusFailed: "Failed",
    statusCanceled: "Canceled",
    triggerUser: "User",
    triggerCron: "Cron",
    triggerSystem: "System",
    channelCLI: "CLI",
    channelWeb: "Web",
    channelScheduler: "Scheduler",
    filterSession: "Session ID",
    filterStatus: "Status",
    filterTriggerType: "Trigger Type",
    filterChannelType: "Channel Type",
    filterChannelID: "Channel ID",
    filterMessageID: "Message ID",
    filterSourceMessageID: "Source Message ID",
    filterStartAt: "Start",
    filterEndAt: "End",
    filterAdvancedShow: "Show Advanced Filters",
    filterAdvancedHide: "Hide Advanced Filters",
    apply: "Apply",
    reset: "Reset",
    fieldID: "ID",
    fieldSession: "Session",
    fieldTaskType: "Task Type",
    fieldPhase: "Phase",
    fieldRetryCount: "Retry Count",
    fieldQueuePosition: "Queue Position",
    fieldQueueWaitMS: "Queue Wait",
    fieldAcceptedAt: "Accepted At",
    fieldStartedAt: "Started At",
    fieldCreated: "Created",
    fieldUpdated: "Updated",
    fieldLastHeartbeatAt: "Last Heartbeat",
    fieldTimeoutAt: "Timeout",
    fieldFinished: "Finished",
    fieldTriggerType: "Trigger Type",
    fieldChannelType: "Channel Type",
    fieldChannelID: "Channel ID",
    fieldCorrelationID: "Correlation ID",
    fieldSourceMessage: "Source Message",
    fieldMessageID: "Message ID",
    fieldRequestMessageID: "Request Message ID",
    fieldResultMessageID: "Result Message ID",
    fieldJobID: "Job ID",
    fieldJobName: "Job Name",
    fieldFiredAt: "Fired At",
    fieldTags: "Tags",
    fieldError: "Error",
    fieldDetailAPI: "Detail API",
    fieldMessages: "Messages",
    unknownError: "unknown_error",
  },
  zh: {
    loading: "加载中...",
    loadFailed: (message) => `加载失败：${message}`,
    empty: "暂无任务记录。",
    emptyHint: "试试调整筛选条件，或稍后再试。",
    selectTaskHint: "从左侧列表选择任务后查看运行详情。",
    applying: "筛选中...",
    copyTaskID: "复制任务 ID",
    openDetail: "查看详情",
    close: "关闭",
    progress: "进度",
    pageLabel: "页码",
    nextPage: "下一页",
    taskDetail: "任务详情",
    detailRuntime: "运行信息",
    detailIdentifiers: "标识信息",
    quickActions: "快捷操作",
    quickTimeline: "生成时间轴版",
    quickInterview: "生成面试速记版",
    logsTitle: "执行日志",
    logsEmpty: "暂无执行日志。",
    logsStreaming: "日志实时拉取中...",
    logsDone: "日志流已结束。",
    logsDisconnected: "日志流已断开，可手动重连。",
    reconnect: "重连",
    replay: "回放",
    terminalTitle: "终端",
    terminalInputPlaceholder: "输入命令或追问继续交互...",
    terminalSend: "发送",
    terminalSending: "发送中...",
    terminalHint: "支持在当前终端会话中继续交互。",
    terminalFollowupNote: "每次发送都会继续复用同一个 Codex 会话线程。",
    resultTitle: "终态输出",
    retry: "重试",
    cancel: "取消",
    retryTip: "重试：从头重新执行该任务。",
    replayTip: "回放：仅重新拉取并播放当前任务日志。",
    statusQueued: "排队中",
    statusRunning: "执行中",
    statusSuccess: "成功",
    statusFailed: "失败",
    statusCanceled: "已取消",
    triggerUser: "用户触发",
    triggerCron: "定时触发",
    triggerSystem: "系统触发",
    channelCLI: "CLI",
    channelWeb: "Web",
    channelScheduler: "Scheduler",
    filterSession: "会话 ID",
    filterStatus: "状态",
    filterTriggerType: "触发类型",
    filterChannelType: "通道类型",
    filterChannelID: "通道 ID",
    filterMessageID: "消息 ID",
    filterSourceMessageID: "源消息 ID",
    filterStartAt: "开始时间",
    filterEndAt: "结束时间",
    filterAdvancedShow: "展开高级筛选",
    filterAdvancedHide: "收起高级筛选",
    apply: "筛选",
    reset: "重置",
    fieldID: "ID",
    fieldSession: "会话",
    fieldTaskType: "任务类型",
    fieldPhase: "阶段",
    fieldRetryCount: "重试次数",
    fieldQueuePosition: "队列位置",
    fieldQueueWaitMS: "排队耗时",
    fieldAcceptedAt: "受理时间",
    fieldStartedAt: "开始时间",
    fieldCreated: "创建时间",
    fieldUpdated: "更新时间",
    fieldLastHeartbeatAt: "最近心跳",
    fieldTimeoutAt: "超时时间",
    fieldFinished: "完成时间",
    fieldTriggerType: "触发类型",
    fieldChannelType: "通道类型",
    fieldChannelID: "通道 ID",
    fieldCorrelationID: "关联 ID",
    fieldSourceMessage: "源消息",
    fieldMessageID: "消息 ID",
    fieldRequestMessageID: "请求消息 ID",
    fieldResultMessageID: "结果消息 ID",
    fieldJobID: "作业 ID",
    fieldJobName: "作业名称",
    fieldFiredAt: "触发时间",
    fieldTags: "标签",
    fieldError: "错误",
    fieldDetailAPI: "详情接口",
    fieldMessages: "总数",
    unknownError: "未知错误",
  },
};

export function ReactManagedTasksRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = TASK_ROUTE_COPY[language];
  const apiClient = useRef(createAPIClient()).current;
  const [draftFilters, setDraftFilters] = useState<TaskRouteFilters>(() => loadTaskRouteFilterDrafts());
  const [activeFilters, setActiveFilters] = useState<TaskRouteFilters>(() => loadTaskRouteFilters());
  const [page, setPage] = useState(1);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [listReloadToken, setListReloadToken] = useState(0);
  const [detailReloadToken, setDetailReloadToken] = useState(0);
  const [listState, setListState] = useState<TaskRouteRequestState>({
    status: "loading",
    items: [],
    pagination: { page: 1, total: 0, has_next: false },
    error: "",
  });
  const [activeTaskID, setActiveTaskID] = useState("");
  const [displayTaskID, setDisplayTaskID] = useState("");
  const [terminalAnchorTaskID, setTerminalAnchorTaskID] = useState("");
  const [detailState, setDetailState] = useState<TaskDetailState>({
    status: "idle",
    view: null,
    error: "",
  });
  const [logState, setLogState] = useState<TaskLogState>({
    items: [],
    status: copy.logsEmpty,
    cursor: 0,
  });
  const [terminalInput, setTerminalInput] = useState("");
  const [terminalSubmitting, setTerminalSubmitting] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    persistTaskRouteFilters(activeFilters);
    let disposed = false;
    setListState((current) => ({
      ...current,
      status: "loading",
      error: "",
    }));

    void apiClient
      .get<TaskRouteListResponse>(buildTaskListQuery(activeFilters, page))
      .then((payload) => {
        if (disposed) {
          return;
        }
        setListState({
          status: "ready",
          items: Array.isArray(payload?.items) ? payload.items : [],
          pagination: payload?.pagination ?? { page, total: 0, has_next: false },
          error: "",
        });
      })
      .catch((error: unknown) => {
        if (disposed) {
          return;
        }
        setListState({
          status: "error",
          items: [],
          pagination: { page, total: 0, has_next: false },
          error: toErrorMessage(error, copy.unknownError),
        });
      });

    return () => {
      disposed = true;
    };
  }, [activeFilters, apiClient, copy.unknownError, page, listReloadToken]);

  useEffect(() => {
    if (!activeTaskID) {
      closeLogStream(eventSourceRef);
      setDetailState({ status: "idle", view: null, error: "" });
      setLogState({ items: [], status: copy.logsEmpty, cursor: 0 });
      setTerminalInput("");
      return;
    }

    let disposed = false;
    closeLogStream(eventSourceRef);
    setDetailState({ status: "loading", view: null, error: "" });
    setLogState({ items: [], status: copy.logsEmpty, cursor: 0 });
    setTerminalInput("");

    void apiClient
      .get<TaskControlView>(`/api/control/tasks/${encodeURIComponent(activeTaskID)}`)
      .then(async (view) => {
        if (disposed) {
          return;
        }
        setDetailState({ status: "ready", view, error: "" });
        const nextAnchorTaskID = normalizeFilterValue(terminalAnchorTaskID) || activeTaskID;
        if (nextAnchorTaskID !== "-") {
          setTerminalAnchorTaskID(nextAnchorTaskID);
        }
        const logs = await loadAllTaskLogs(apiClient, activeTaskID);
        if (disposed) {
          return;
        }
        setLogState({
          items: logs.items,
          status: logs.items.length ? copy.logsStreaming : copy.logsEmpty,
          cursor: logs.cursor,
        });
        startTaskLogStream({
          taskID: activeTaskID,
          cursor: logs.cursor,
          eventSourceRef,
          onLog: (item, nextCursor) => {
            if (disposed) {
              return;
            }
            setLogState((current) => ({
              items: appendUniqueTaskLogs(current.items, [item]),
              status: copy.logsStreaming,
              cursor: nextCursor,
            }));
          },
          onDone: (nextCursor) => {
            if (disposed) {
              return;
            }
            setLogState((current) => ({
              ...current,
              status: copy.logsDone,
              cursor: nextCursor,
            }));
          },
          onDisconnected: () => {
            if (disposed) {
              return;
            }
            setLogState((current) => ({
              ...current,
              status: current.items.length ? copy.logsDisconnected : copy.logsEmpty,
            }));
          },
        });
      })
      .catch((error: unknown) => {
        if (disposed) {
          return;
        }
        setDetailState({
          status: "error",
          view: null,
          error: toErrorMessage(error, copy.unknownError),
        });
      });

    return () => {
      disposed = true;
      closeLogStream(eventSourceRef);
    };
  }, [activeTaskID, apiClient, copy, detailReloadToken, terminalAnchorTaskID]);

  async function handleCopyTaskID(taskID: string) {
    if (!taskID || taskID === "-") {
      return;
    }
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      await navigator.clipboard.writeText(taskID);
    }
  }

  async function handleTaskAction(action: "retry" | "cancel") {
    if (!activeTaskID) {
      return;
    }
    try {
      const payload = await requestTaskMutation<TaskControlView | { view?: TaskControlView; error?: string }>(
        `/api/control/tasks/${encodeURIComponent(activeTaskID)}/${action}`,
        { method: "POST" },
      );
      if ("view" in payload && payload.view) {
        setDetailState({ status: "ready", view: payload.view, error: "" });
      }
      setListReloadToken((current) => current + 1);
      setDetailReloadToken((current) => current + 1);
    } catch (error: unknown) {
      window.alert(toErrorMessage(error, copy.unknownError));
    }
  }

  async function handleTerminalSubmit(rawInput: string) {
    if (!activeTaskID) {
      return;
    }
    const input = normalizeFilterValue(rawInput);
    if (!input) {
      return;
    }

    setTerminalSubmitting(true);
    try {
      const anchorTaskID = normalizeFilterValue(terminalAnchorTaskID) || activeTaskID;
      const payload = await requestTaskMutation<TaskTerminalInputResponse>(
        `/api/control/tasks/${encodeURIComponent(activeTaskID)}/terminal/input`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            input,
            reuse_task: true,
            anchor_task_id: anchorTaskID,
          }),
        },
      );
      const nextTaskID = normalizeFilterValue(payload.task_id);
      const nextAnchorTaskID = normalizeFilterValue(payload.anchor_task_id) || anchorTaskID;
      if (nextTaskID) {
        setActiveTaskID(nextTaskID);
        setDisplayTaskID(nextAnchorTaskID);
        setTerminalAnchorTaskID(nextAnchorTaskID);
      }
    } catch (error: unknown) {
      window.alert(toErrorMessage(error, copy.unknownError));
    } finally {
      setTerminalSubmitting(false);
      setTerminalInput("");
    }
  }

  async function reconnectLogs() {
    if (!activeTaskID) {
      return;
    }
    closeLogStream(eventSourceRef);
    startTaskLogStream({
      taskID: activeTaskID,
      cursor: logState.cursor,
      eventSourceRef,
      onLog: (item, nextCursor) => {
        setLogState((current) => ({
          items: appendUniqueTaskLogs(current.items, [item]),
          status: copy.logsStreaming,
          cursor: nextCursor,
        }));
      },
      onDone: (nextCursor) => {
        setLogState((current) => ({
          ...current,
          status: copy.logsDone,
          cursor: nextCursor,
        }));
      },
      onDisconnected: () => {
        setLogState((current) => ({
          ...current,
          status: current.items.length ? copy.logsDisconnected : copy.logsEmpty,
        }));
      },
    });
  }

  async function replayLogs() {
    if (!activeTaskID) {
      return;
    }
    closeLogStream(eventSourceRef);
    const logs = await loadAllTaskLogs(apiClient, activeTaskID);
    setLogState({
      items: logs.items,
      status: logs.items.length ? copy.logsStreaming : copy.logsEmpty,
      cursor: logs.cursor,
    });
    await reconnectLogs();
  }

  const hasDrawer = Boolean(activeTaskID);
  const detailView = detailState.status === "ready" ? detailState.view : null;
  const detailTask = detailView?.task;
  const detailSource = detailView?.source;
  const detailActions = detailView?.actions;
  const detailLink = detailView?.link;
  const selectedDisplayTaskID = normalizeFilterValue(displayTaskID) || activeTaskID;
  const detailTags = [
    formatTriggerType(detailSource?.trigger_type, copy),
    formatChannelType(detailSource?.channel_type, copy),
    normalizeText(detailLink?.terminal_session_id),
  ].filter((item) => item !== "-");

  return (
    <section className="control-task-view" data-control-task-view>
      <form
        className="task-filter-form page-filter-form control-task-filter-form"
        onSubmit={(event) => {
          event.preventDefault();
          setPage(1);
          setActiveFilters(buildTaskRouteRequestFilters(draftFilters));
        }}
      >
        <div className="task-filter-primary-row control-task-filter-primary">
          <label>
            <span>{copy.filterSession}</span>
            <input
              type="text"
              name="session_id"
              placeholder="session-123"
              value={draftFilters.sessionID}
              onChange={(event) => {
                setDraftFilters((current) => ({
                  ...current,
                  sessionID: event.target.value,
                }));
              }}
            />
          </label>
          <label>
            <span>{copy.filterStatus}</span>
            <select
              name="status"
              value={draftFilters.status}
              onChange={(event) => {
                setDraftFilters((current) => ({
                  ...current,
                  status: normalizeFilterValue(event.target.value),
                }));
              }}
            >
              <option value="">-</option>
              <option value="queued">{copy.statusQueued}</option>
              <option value="running">{copy.statusRunning}</option>
              <option value="success">{copy.statusSuccess}</option>
              <option value="failed">{copy.statusFailed}</option>
              <option value="canceled">{copy.statusCanceled}</option>
            </select>
          </label>
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
        </div>
        <div className="task-filter-primary-actions control-task-filter-toolbar">
          <button
            className="task-filter-advanced-toggle"
            type="button"
            aria-expanded={advancedOpen ? "true" : "false"}
            onClick={() => {
              setAdvancedOpen((current) => !current);
            }}
          >
            {advancedOpen ? copy.filterAdvancedHide : copy.filterAdvancedShow}
          </button>
          <div className="task-filter-actions">
            <button className="task-filter-apply" type="submit">
              {copy.apply}
            </button>
            <button
              className="task-filter-reset"
              type="button"
              onClick={() => {
                setDraftFilters(DEFAULT_FILTERS);
                setActiveFilters(DEFAULT_FILTERS);
                setAdvancedOpen(false);
                setPage(1);
                setActiveTaskID("");
                setDisplayTaskID("");
                setTerminalAnchorTaskID("");
              }}
            >
              {copy.reset}
            </button>
          </div>
        </div>
        <div
          className="task-filter-advanced control-task-filter-advanced-panel"
          hidden={!advancedOpen}
        >
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
            <span>{copy.filterSourceMessageID}</span>
            <input
              type="text"
              name="source_message_id"
              placeholder="msg-source-123"
              value={draftFilters.sourceMessageID}
              onChange={(event) => {
                setDraftFilters((current) => ({
                  ...current,
                  sourceMessageID: event.target.value,
                }));
              }}
            />
          </label>
          <label>
            <span>{copy.filterStartAt}</span>
            <input
              type="datetime-local"
              name="start_at"
              value={draftFilters.startAt}
              onChange={(event) => {
                setDraftFilters((current) => ({
                  ...current,
                  startAt: event.target.value,
                }));
              }}
            />
          </label>
          <label>
            <span>{copy.filterEndAt}</span>
            <input
              type="datetime-local"
              name="end_at"
              value={draftFilters.endAt}
              onChange={(event) => {
                setDraftFilters((current) => ({
                  ...current,
                  endAt: event.target.value,
                }));
              }}
            />
          </label>
        </div>
      </form>

      <section className="control-task-layout">
        <div className="control-task-main">
          {listState.status === "loading" ? (
            <div className="task-summary-pagination">
              <p>
                <span>{copy.applying}</span>
              </p>
            </div>
          ) : null}

          <div className="task-summary-list task-summary-list-compact" data-control-task-list>
            {listState.status === "error" ? (
              <p className="route-empty">{copy.loadFailed(listState.error)}</p>
            ) : listState.items.length === 0 ? (
              <div className="task-empty-state">
                <p className="task-empty-title">{copy.empty}</p>
                <p className="task-empty-hint">{copy.emptyHint}</p>
              </div>
            ) : (
              listState.items.map((item) => {
                const taskID = normalizeText(item.task_id);
                const isActive = activeTaskID === item.task_id;
                return (
                  <button
                    key={taskID}
                    className={`task-summary-row${isActive ? " active" : ""}`}
                    aria-label={taskID}
                    aria-current={isActive ? "true" : undefined}
                    type="button"
                    onClick={() => {
                      setActiveTaskID(normalizeFilterValue(item.task_id));
                      setDisplayTaskID(normalizeFilterValue(item.task_id));
                      setTerminalAnchorTaskID(normalizeFilterValue(item.task_id));
                    }}
                  >
                    <div className="task-summary-row-main">
                      <strong className="task-summary-row-id" title={taskID}>
                        {taskID}
                      </strong>
                      <span className="task-summary-row-subline">
                        {formatTriggerType(item.trigger_type, copy)} · {formatChannelType(item.channel_type, copy)}
                      </span>
                    </div>
                    <div className="task-summary-row-side">
                      <span className={`task-summary-status ${taskStatusClassName(item.status)}`}>
                        {formatTaskStatus(item.status, copy)}
                      </span>
                    </div>
                    <div className="task-summary-row-meta">
                      <span>{normalizeText(item.session_id)}</span>
                      <span>{formatDateTime(item.updated_at)}</span>
                    </div>
                  </button>
                );
              })
            )}
          </div>

          <div className="task-summary-pagination-wrap" data-control-task-pagination>
            <div className="task-summary-pagination">
              <p>
                <span>{copy.fieldMessages}</span>
                <strong>{normalizeText(listState.pagination.total ?? 0)}</strong>
              </p>
              <p>
                <span>{copy.pageLabel}</span>
                <strong>{normalizeText(listState.pagination.page ?? page)}</strong>
              </p>
              <button
                className="task-summary-next"
                type="button"
                disabled={!listState.pagination.has_next}
                onClick={() => {
                  setPage((current) => current + 1);
                  setActiveTaskID("");
                }}
              >
                {copy.nextPage}
              </button>
            </div>
          </div>
        </div>

        <section className="control-task-drawer">
          <aside className="control-task-drawer-panel">
            <header className="control-task-drawer-head">
              <div>
                <h4>{copy.taskDetail}</h4>
                {hasDrawer ? <p>{selectedDisplayTaskID}</p> : null}
              </div>
              {hasDrawer ? (
                <button
                  className="task-summary-next"
                  type="button"
                  onClick={() => {
                    setActiveTaskID("");
                    setDisplayTaskID("");
                    setTerminalAnchorTaskID("");
                  }}
                >
                  {copy.close}
                </button>
              ) : null}
            </header>
            <div className="control-task-drawer-body" data-control-task-drawer-body>
              {!hasDrawer ? <div className="task-detail-placeholder">{copy.selectTaskHint}</div> : null}
              {hasDrawer && detailState.status === "loading" ? <p>{copy.loading}</p> : null}
              {hasDrawer && detailState.status === "error" ? <p>{copy.loadFailed(detailState.error)}</p> : null}
              {detailView ? (
                <section className="task-detail-card">
                  <header className="task-detail-head">
                    <div className="task-detail-id-wrap">
                      <h5 title={selectedDisplayTaskID}>{selectedDisplayTaskID}</h5>
                      <button
                        className="task-summary-copy"
                        type="button"
                        aria-label={copy.copyTaskID}
                        title={copy.copyTaskID}
                        onClick={() => {
                          void handleCopyTaskID(selectedDisplayTaskID);
                        }}
                      >
                        {copy.copyTaskID}
                      </button>
                    </div>
                    <span className={`task-summary-status ${taskStatusClassName(detailTask?.status)}`}>
                      {formatTaskStatus(detailTask?.status, copy)}
                    </span>
                  </header>
                  <div className="task-detail-meta">
                    <div className="task-detail-meta-core">
                      <p>
                        <span>{copy.fieldSession}</span>
                        <strong>{normalizeText(detailTask?.session_id)}</strong>
                      </p>
                      <p>
                        <span>{copy.fieldTaskType}</span>
                        <strong>{normalizeText(detailTask?.task_type)}</strong>
                      </p>
                      <p>
                        <span>{copy.fieldPhase}</span>
                        <strong>{normalizeText(detailTask?.phase || detailTask?.status)}</strong>
                      </p>
                      <p>
                        <span>{copy.fieldRetryCount}</span>
                        <strong>{normalizeText(detailTask?.retry_count)}</strong>
                      </p>
                    </div>
                    <div className="task-detail-progress">
                      <p>
                        <span>{copy.progress}</span>
                        <strong>{Math.max(0, Math.min(100, Number(detailTask?.progress || 0)))}%</strong>
                      </p>
                      <div className="task-detail-progress-track">
                        <span
                          className="task-detail-progress-fill"
                          style={{
                            width: `${Math.max(0, Math.min(100, Number(detailTask?.progress || 0)))}%`,
                          }}
                        ></span>
                      </div>
                    </div>
                    <details className="task-detail-meta-fold">
                      <summary>{copy.detailRuntime}</summary>
                      <div className="task-detail-meta-fold-body">
                        <p>
                          <span>{copy.fieldQueuePosition}</span>
                          <strong>{normalizeText(detailTask?.queue_position)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldQueueWaitMS}</span>
                          <strong>{formatDurationMS(detailTask?.queue_wait_ms)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldAcceptedAt}</span>
                          <strong>{formatDateTime(detailTask?.accepted_at)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldStartedAt}</span>
                          <strong>{formatDateTime(detailTask?.started_at)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldCreated}</span>
                          <strong>{formatDateTime(detailTask?.created_at)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldUpdated}</span>
                          <strong>{formatDateTime(detailTask?.updated_at)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldLastHeartbeatAt}</span>
                          <strong>{formatDateTime(detailTask?.last_heartbeat_at)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldTimeoutAt}</span>
                          <strong>{formatDateTime(detailTask?.timeout_at)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldFinished}</span>
                          <strong>{formatDateTime(detailTask?.finished_at)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldTriggerType}</span>
                          <strong>{formatTriggerType(detailSource?.trigger_type, copy)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldChannelType}</span>
                          <strong>{formatChannelType(detailSource?.channel_type, copy)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldJobID}</span>
                          <strong>{normalizeText(detailSource?.job_id)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldJobName}</span>
                          <strong>{normalizeText(detailSource?.job_name)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldFiredAt}</span>
                          <strong>{formatDateTime(detailSource?.fired_at)}</strong>
                        </p>
                      </div>
                    </details>
                    <details className="task-detail-meta-fold">
                      <summary>{copy.detailIdentifiers}</summary>
                      <div className="task-detail-meta-fold-body">
                        <p>
                          <span>{copy.fieldChannelID}</span>
                          <strong>{normalizeText(detailSource?.channel_id)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldCorrelationID}</span>
                          <strong>{normalizeText(detailSource?.correlation_id)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldSourceMessage}</span>
                          <strong>{normalizeText(detailTask?.source_message_id)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldMessageID}</span>
                          <strong>{normalizeText(detailTask?.message_id)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldRequestMessageID}</span>
                          <strong>{normalizeText(detailLink?.request_message_id)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldResultMessageID}</span>
                          <strong>{normalizeText(detailLink?.result_message_id)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldError}</span>
                          <strong>{normalizeText(detailTask?.error_message || detailTask?.error_code)}</strong>
                        </p>
                        <p>
                          <span>{copy.fieldDetailAPI}</span>
                          <strong>{normalizeText(detailLink?.task_detail_path)}</strong>
                        </p>
                      </div>
                    </details>
                  </div>
                  <RouteTagSection label={copy.fieldTags} tags={detailTags} />
                  <div className="task-detail-actions">
                    <button
                      type="button"
                      disabled={!detailActions?.retry?.enabled}
                      title={detailActions?.retry?.enabled ? copy.retryTip : normalizeText(detailActions?.retry?.reason)}
                      onClick={() => {
                        void handleTaskAction("retry");
                      }}
                    >
                      {copy.retry}
                    </button>
                    <button
                      type="button"
                      disabled={!detailActions?.cancel?.enabled}
                      onClick={() => {
                        void handleTaskAction("cancel");
                      }}
                    >
                      {copy.cancel}
                    </button>
                    <button type="button" onClick={() => void reconnectLogs()}>
                      {copy.reconnect}
                    </button>
                    <button type="button" title={copy.replayTip} onClick={() => void replayLogs()}>
                      {copy.replay}
                    </button>
                  </div>

                  <section className="task-detail-section">
                    <h5>{copy.terminalTitle}</h5>
                    <p className="control-task-log-state">{logState.status}</p>
                    <div className="control-task-log-stream">
                      {logState.items.length ? (
                        <div className="control-task-terminal-screen">
                          {logState.items.map((item) => {
                            const stage = normalizeFilterValue(item.stage).toLowerCase();
                            const tagClass =
                              stage === "success"
                                ? "is-success"
                                : stage === "running"
                                  ? "is-running"
                                  : stage === "accept" || stage === "accepted"
                                    ? "is-accept"
                                    : "is-default";
                            return (
                              <div key={`${item.seq || 0}:${item.message || ""}`} className="control-task-log-line">
                                {stage ? (
                                  <span className={`control-task-log-tag ${tagClass}`}>[{stage}]</span>
                                ) : null}
                                <span className="control-task-log-message">{normalizeText(item.message)}</span>
                              </div>
                            );
                          })}
                        </div>
                      ) : (
                        <div className="control-task-terminal-screen is-empty">{copy.logsEmpty}</div>
                      )}
                    </div>
                    <div className="control-task-quick-actions">
                      <span>{copy.quickActions}</span>
                      <button type="button" onClick={() => void handleTerminalSubmit(copy.quickTimeline)}>
                        {copy.quickTimeline}
                      </button>
                      <button type="button" onClick={() => void handleTerminalSubmit(copy.quickInterview)}>
                        {copy.quickInterview}
                      </button>
                    </div>
                    <form
                      className="control-task-terminal-input"
                      onSubmit={(event) => {
                        event.preventDefault();
                        void handleTerminalSubmit(terminalInput);
                      }}
                    >
                      <input
                        type="text"
                        maxLength={6000}
                        placeholder={copy.terminalInputPlaceholder}
                        value={terminalInput}
                        disabled={terminalSubmitting}
                        onChange={(event) => {
                          setTerminalInput(event.target.value);
                        }}
                      />
                      <button type="submit" disabled={terminalSubmitting}>
                        {terminalSubmitting ? copy.terminalSending : copy.terminalSend}
                      </button>
                    </form>
                    <p className="control-task-terminal-hint">{copy.terminalHint}</p>
                    <p className="control-task-terminal-note">{copy.terminalFollowupNote}</p>
                  </section>

                  <section className="task-detail-section">
                    <h5>{copy.resultTitle}</h5>
                    <div className="control-task-result-output">
                      <TaskResultOutput content={detailTask?.result?.output} />
                    </div>
                  </section>
                </section>
              ) : null}
            </div>
          </aside>
        </section>
      </section>
    </section>
  );
}

function TaskResultOutput({ content }: { content: string | undefined }) {
  const text = normalizeFilterValue(content);
  if (!text) {
    return <p>-</p>;
  }

  const items = text
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  const bullets = items.filter((line) => line.startsWith("- "));
  if (bullets.length === items.length) {
    return (
      <ul>
        {bullets.map((line) => (
          <li key={line}>{line.replace(/^- /, "")}</li>
        ))}
      </ul>
    );
  }
  return (
    <>
      {items.map((line) => (
        <p key={line}>{line.replace(/^- /, "")}</p>
      ))}
    </>
  );
}

function normalizeFilterValue(value: unknown) {
  return String(value || "").trim();
}

function normalizeTaskRouteFilters(filters: TaskRouteFilters): TaskRouteFilters {
  return {
    sessionID: normalizeFilterValue(filters.sessionID),
    status: normalizeFilterValue(filters.status),
    triggerType: normalizeFilterValue(filters.triggerType),
    channelType: normalizeFilterValue(filters.channelType),
    channelID: normalizeFilterValue(filters.channelID),
    messageID: normalizeFilterValue(filters.messageID),
    sourceMessageID: normalizeFilterValue(filters.sourceMessageID),
    startAt: normalizeFilterValue(filters.startAt),
    endAt: normalizeFilterValue(filters.endAt),
  };
}

function buildTaskRouteRequestFilters(filters: TaskRouteFilters): TaskRouteFilters {
  const normalized = normalizeTaskRouteFilters(filters);
  return {
    ...normalized,
    startAt: toRFC3339FilterValue(normalized.startAt),
    endAt: toRFC3339FilterValue(normalized.endAt),
  };
}

function buildTaskListQuery(filters: TaskRouteFilters, page: number) {
  const params = new URLSearchParams();
  params.set("page", String(Math.max(page, 1)));
  params.set("page_size", String(TASK_ROUTE_PAGE_SIZE));
  if (filters.sessionID) {
    params.set("session_id", filters.sessionID);
  }
  if (filters.status) {
    params.set("status", filters.status);
  }
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
  if (filters.sourceMessageID) {
    params.set("source_message_id", filters.sourceMessageID);
  }
  if (filters.startAt && filters.endAt) {
    params.set("time_range", `${filters.startAt},${filters.endAt}`);
  } else {
    if (filters.startAt) {
      params.set("start_at", filters.startAt);
    }
    if (filters.endAt) {
      params.set("end_at", filters.endAt);
    }
  }
  return `/api/control/tasks?${params.toString()}`;
}

function loadTaskRouteFilters(): TaskRouteFilters {
  if (typeof window === "undefined") {
    return DEFAULT_FILTERS;
  }
  const raw = window.sessionStorage.getItem(TASK_ROUTE_FILTERS_STORAGE_KEY);
  if (!raw) {
    return DEFAULT_FILTERS;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<TaskRouteFilters>;
    return normalizeTaskRouteFilters({
      ...DEFAULT_FILTERS,
      ...parsed,
    });
  } catch {
    return DEFAULT_FILTERS;
  }
}

function loadTaskRouteFilterDrafts(): TaskRouteFilters {
  const filters = loadTaskRouteFilters();
  return {
    ...filters,
    startAt: toDateTimeInputValue(filters.startAt),
    endAt: toDateTimeInputValue(filters.endAt),
  };
}

function persistTaskRouteFilters(filters: TaskRouteFilters) {
  if (typeof window === "undefined") {
    return;
  }
  window.sessionStorage.setItem(TASK_ROUTE_FILTERS_STORAGE_KEY, JSON.stringify(filters));
}

function formatTaskStatus(value: unknown, copy: TaskRouteCopy) {
  const status = normalizeFilterValue(value).toLowerCase();
  if (status === "queued") {
    return copy.statusQueued;
  }
  if (status === "running") {
    return copy.statusRunning;
  }
  if (status === "success") {
    return copy.statusSuccess;
  }
  if (status === "failed") {
    return copy.statusFailed;
  }
  if (status === "canceled") {
    return copy.statusCanceled;
  }
  return normalizeText(value);
}

function taskStatusClassName(value: unknown) {
  const status = normalizeFilterValue(value).toLowerCase();
  if (status === "success") {
    return "status-success";
  }
  if (status === "queued" || status === "running") {
    return "status-pending";
  }
  if (status === "failed" || status === "canceled") {
    return "status-failed";
  }
  return "status-neutral";
}

function formatTriggerType(value: unknown, copy: TaskRouteCopy) {
  const triggerType = normalizeFilterValue(value).toLowerCase();
  if (triggerType === "user") {
    return copy.triggerUser;
  }
  if (triggerType === "cron") {
    return copy.triggerCron;
  }
  if (triggerType === "system") {
    return copy.triggerSystem;
  }
  return normalizeText(value);
}

function formatChannelType(value: unknown, copy: TaskRouteCopy) {
  const channelType = normalizeFilterValue(value).toLowerCase();
  if (channelType === "cli") {
    return copy.channelCLI;
  }
  if (channelType === "web") {
    return copy.channelWeb;
  }
  if (channelType === "scheduler") {
    return copy.channelScheduler;
  }
  return normalizeText(value);
}

function formatDurationMS(value: unknown) {
  const duration = Number(value || 0);
  if (!Number.isFinite(duration) || duration <= 0) {
    return "-";
  }
  if (duration < 1000) {
    return `${Math.round(duration)}ms`;
  }
  if (duration < 60000) {
    return `${(duration / 1000).toFixed(1)}s`;
  }
  return `${Math.floor(duration / 60000)}m ${Math.round((duration % 60000) / 1000)}s`;
}

function toRFC3339FilterValue(value: string) {
  const text = normalizeFilterValue(value);
  if (!text) {
    return "";
  }
  const parsed = new Date(text);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  return parsed.toISOString();
}

function toDateTimeInputValue(value: string) {
  const text = normalizeFilterValue(value);
  if (!text) {
    return "";
  }
  const parsed = new Date(text);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  const year = parsed.getFullYear();
  const month = String(parsed.getMonth() + 1).padStart(2, "0");
  const day = String(parsed.getDate()).padStart(2, "0");
  const hour = String(parsed.getHours()).padStart(2, "0");
  const minute = String(parsed.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hour}:${minute}`;
}

async function loadAllTaskLogs(apiClient: ReturnType<typeof createAPIClient>, taskID: string) {
  let cursor = 0;
  let hasMore = true;
  let guard = 0;
  let items: TaskLogItem[] = [];

  while (hasMore && guard < 40) {
    guard += 1;
    const page = await apiClient.get<TaskLogPage>(
      `/api/control/tasks/${encodeURIComponent(taskID)}/logs?cursor=${cursor}&limit=${TASK_LOG_PAGE_SIZE}`,
    );
    items = appendUniqueTaskLogs(items, Array.isArray(page?.items) ? page.items : []);
    cursor = Number(page?.next_cursor || cursor);
    hasMore = Boolean(page?.has_more);
  }

  return {
    items,
    cursor,
  };
}

function appendUniqueTaskLogs(current: TaskLogItem[], incoming: TaskLogItem[]) {
  const next = [...current];
  const seen = new Set(next.map((item) => `${item.seq || 0}:${item.message || ""}`));
  for (const item of incoming) {
    const key = `${item.seq || 0}:${item.message || ""}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    next.push(item);
  }
  return next.sort((left, right) => Number(left.seq || 0) - Number(right.seq || 0));
}

function startTaskLogStream({
  taskID,
  cursor,
  eventSourceRef,
  onLog,
  onDone,
  onDisconnected,
}: {
  taskID: string;
  cursor: number;
  eventSourceRef: MutableRefObject<EventSource | null>;
  onLog: (item: TaskLogItem, nextCursor: number) => void;
  onDone: (nextCursor: number) => void;
  onDisconnected: () => void;
}) {
  closeLogStream(eventSourceRef);
  const stream = new EventSource(
    `/api/control/tasks/${encodeURIComponent(taskID)}/logs/stream?cursor=${Math.max(cursor, 0)}`,
  );
  eventSourceRef.current = stream;

  stream.addEventListener("log", (event) => {
    const detail = parseEventPayload<{ log?: TaskLogItem; next_cursor?: number }>(event);
    if (!detail?.log) {
      return;
    }
    onLog(detail.log, Number(detail.next_cursor || cursor));
  });

  stream.addEventListener("done", (event) => {
    const detail = parseEventPayload<{ next_cursor?: number }>(event);
    onDone(Number(detail?.next_cursor || cursor));
    closeLogStream(eventSourceRef);
  });

  stream.addEventListener("error", () => {
    onDisconnected();
    closeLogStream(eventSourceRef);
  });
}

function closeLogStream(eventSourceRef: MutableRefObject<EventSource | null>) {
  if (!eventSourceRef.current) {
    return;
  }
  eventSourceRef.current.close();
  eventSourceRef.current = null;
}

function parseEventPayload<T>(event: Event): T | null {
  if (!(event instanceof MessageEvent) || typeof event.data !== "string" || !event.data.trim()) {
    return null;
  }
  try {
    return JSON.parse(event.data) as T;
  } catch {
    return null;
  }
}

async function requestTaskMutation<T>(path: string, init: RequestInit) {
  const response = await fetch(path, init);
  const payload = (await readResponsePayload(response)) as T | { error?: string; message?: string };
  if (!response.ok) {
    const errorMessage =
      (payload as { error?: string; message?: string }).error ||
      (payload as { error?: string; message?: string }).message ||
      `HTTP ${response.status}`;
    throw new APIClientError(errorMessage, response.status, "", payload);
  }
  return payload as T;
}

async function readResponsePayload(response: Response) {
  const text = await response.text();
  if (!text) {
    return {};
  }
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return { message: text };
  }
}

function toErrorMessage(error: unknown, fallback: string) {
  if (error instanceof APIClientError) {
    return error.message || fallback;
  }
  if (error instanceof Error) {
    return error.message || fallback;
  }
  return fallback;
}
