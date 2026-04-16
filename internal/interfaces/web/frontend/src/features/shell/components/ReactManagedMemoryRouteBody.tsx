import { useEffect, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTime } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { RouteCard, RouteFieldRow } from "./RouteBodyPrimitives";

type MemoryDocument = {
  exists?: boolean;
  path?: string;
  updated_at?: string;
  content?: string;
  error?: string;
};

type DailyMemoryItem = MemoryDocument & {
  date?: string;
};

type MemoryPayload = {
  long_term?: MemoryDocument;
  daily?: {
    directory?: string;
    items?: DailyMemoryItem[];
    error?: string;
  };
  mandatory?: MemoryDocument;
  specification?: MemoryDocument;
};

type TaskSummaryItem = {
  task_id?: string;
  task_type?: string;
  goal?: string;
  result?: string;
  status?: string;
  updated_at?: string;
  finished_at?: string;
  last_heartbeat_at?: string;
  timeout_at?: string;
  tags?: string[];
};

type TaskListPayload = {
  items?: TaskSummaryItem[];
  pagination?: {
    page?: number;
    total?: number;
    has_next?: boolean;
  };
};

type TaskDetailPayload = {
  meta?: {
    task_id?: string;
    task_type?: string;
    session_id?: string;
    source_message_id?: string;
    status?: string;
    progress?: number;
    retry_count?: number;
    updated_at?: string;
    finished_at?: string;
    last_heartbeat_at?: string;
    timeout_at?: string;
  };
  summary_refs?: Array<{
    tier?: string;
    date?: string;
    path?: string;
  }>;
};

type TaskLogsPayload = {
  items?: Array<{
    seq?: number;
    stage?: string;
    level?: string;
    created_at?: string;
    timestamp?: string;
    message?: string;
  }>;
  next_cursor?: number;
  has_more?: boolean;
  error_code?: string;
};

type TaskArtifactsPayload = {
  items?: Array<{
    artifact_id?: string;
    artifact_type?: string;
    name?: string;
    summary?: string;
    content_type?: string;
    created_at?: string;
    download_url?: string;
    preview_url?: string;
  }>;
  error?: string;
};

type MemoryTab = "tasks" | "long_term" | "daily" | "mandatory" | "specification";

type Copy = {
  loading: string;
  longTerm: string;
  daily: string;
  tasks: string;
  mandatory: string;
  specification: string;
  readOnly: string;
  noLongTerm: string;
  noDaily: string;
  noMandatory: string;
  noSpecification: string;
  noTaskHistory: string;
  sourceDirectory: string;
  summary: string;
  taskDetail: string;
  selectTask: string;
  status: string;
  taskType: string;
  start: string;
  end: string;
  apply: string;
  reset: string;
  viewDetail: string;
  page: string;
  nextPage: string;
  loadLogs: string;
  loadMoreLogs: string;
  noLogs: string;
  logsHint: string;
  loadArtifacts: string;
  noArtifacts: string;
  rebuild: string;
  back: string;
  path: string;
  updated: string;
  date: string;
  session: string;
  sourceMessage: string;
  progress: string;
  retryCount: string;
  finished: string;
  lastHeartbeat: string;
  refs: string;
  loadFailed: (message: string) => string;
};

const COPY: Record<LegacyShellLanguage, Copy> = {
  en: {
    loading: "Loading...",
    longTerm: "Long-Term",
    daily: "Daily",
    tasks: "Task History",
    mandatory: "SOUL.md",
    specification: "Specification",
    readOnly: "Read-only",
    noLongTerm: "No long-term memory file available.",
    noDaily: "No daily memory files available.",
    noMandatory: "No SOUL.md file available.",
    noSpecification: "No memory specification document available.",
    noTaskHistory: "No task history.",
    sourceDirectory: "Source Directory",
    summary: "Summary",
    taskDetail: "Task Detail",
    selectTask: "Select a task summary to view detail.",
    status: "Status",
    taskType: "Task Type",
    start: "Start",
    end: "End",
    apply: "Apply",
    reset: "Reset",
    viewDetail: "View Detail",
    page: "Page",
    nextPage: "Next Page",
    loadLogs: "Load Logs",
    loadMoreLogs: "Load More Logs",
    noLogs: "No logs available.",
    logsHint: "Logs unavailable. You can rebuild summary.",
    loadArtifacts: "Load Artifacts",
    noArtifacts: "No artifacts available.",
    rebuild: "Rebuild Summary",
    back: "Back to Summary",
    path: "Path",
    updated: "Updated",
    date: "Date",
    session: "Session",
    sourceMessage: "Source Message",
    progress: "Progress",
    retryCount: "Retry Count",
    finished: "Finished",
    lastHeartbeat: "Last Heartbeat",
    refs: "Summary Refs",
    loadFailed: (message) => `Load failed: ${message}`,
  },
  zh: {
    loading: "加载中...",
    longTerm: "长期记忆",
    daily: "天级记忆",
    tasks: "任务历史",
    mandatory: "SOUL.md",
    specification: "说明文档",
    readOnly: "只读",
    noLongTerm: "暂无长期记忆文件。",
    noDaily: "暂无天级记忆文件。",
    noMandatory: "暂无 SOUL.md 文件。",
    noSpecification: "暂无记忆模块说明文档。",
    noTaskHistory: "暂无任务历史。",
    sourceDirectory: "来源目录",
    summary: "摘要",
    taskDetail: "任务详情",
    selectTask: "选择一条任务摘要查看详情。",
    status: "状态",
    taskType: "任务类型",
    start: "开始时间",
    end: "结束时间",
    apply: "筛选",
    reset: "重置",
    viewDetail: "查看详情",
    page: "页码",
    nextPage: "下一页",
    loadLogs: "加载日志",
    loadMoreLogs: "加载更多日志",
    noLogs: "暂无日志。",
    logsHint: "日志不可用，可执行摘要重建。",
    loadArtifacts: "加载产物",
    noArtifacts: "暂无产物。",
    rebuild: "重建摘要",
    back: "返回摘要",
    path: "路径",
    updated: "更新时间",
    date: "日期",
    session: "会话",
    sourceMessage: "源消息",
    progress: "进度",
    retryCount: "重试次数",
    finished: "完成时间",
    lastHeartbeat: "最近心跳",
    refs: "摘要引用",
    loadFailed: (message) => `加载失败：${message}`,
  },
};

const DEFAULT_TASK_FILTERS = {
  status: "",
  taskType: "",
  startAt: "",
  endAt: "",
};

export function ReactManagedMemoryRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = COPY[language];
  const apiClient = createAPIClient();
  const [activeTab, setActiveTab] = useState<MemoryTab>("tasks");
  const [memoryPayload, setMemoryPayload] = useState<MemoryPayload | null>(null);
  const [taskList, setTaskList] = useState<TaskListPayload>({ items: [], pagination: { page: 1, total: 0, has_next: false } });
  const [taskFilters, setTaskFilters] = useState(DEFAULT_TASK_FILTERS);
  const [taskDraftFilters, setTaskDraftFilters] = useState(DEFAULT_TASK_FILTERS);
  const [taskDetail, setTaskDetail] = useState<TaskDetailPayload | null>(null);
  const [taskLogs, setTaskLogs] = useState<TaskLogsPayload | null>(null);
  const [taskArtifacts, setTaskArtifacts] = useState<TaskArtifactsPayload | null>(null);
  const [requestState, setRequestState] = useState<{ status: "loading" | "ready" | "error"; error: string }>({
    status: "loading",
    error: "",
  });

  useEffect(() => {
    let disposed = false;

    async function loadInitial() {
      setRequestState({ status: "loading", error: "" });
      try {
        const [memory, tasks] = await Promise.all([
          apiClient.get<MemoryPayload>("/api/agent/memory"),
          apiClient.get<TaskListPayload>(buildTaskHistoryQuery(DEFAULT_TASK_FILTERS, 1)),
        ]);
        if (disposed) {
          return;
        }
        setMemoryPayload(memory);
        setTaskList(tasks);
        setRequestState({ status: "ready", error: "" });
      } catch (error: unknown) {
        if (disposed) {
          return;
        }
        setRequestState({
          status: "error",
          error: error instanceof Error ? error.message : "unknown_error",
        });
      }
    }

    void loadInitial();
    return () => {
      disposed = true;
    };
  }, []);

  if (requestState.status === "loading") {
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (requestState.status === "error") {
    return <p className="route-error">{copy.loadFailed(requestState.error)}</p>;
  }

  const taskItems = Array.isArray(taskList.items) ? taskList.items : [];
  const pagination = taskList.pagination ?? { page: 1, total: 0, has_next: false };

  return (
    <section className="memory-view">
      <div className="memory-tabs" role="tablist" aria-label="Memory">
        {[
          { id: "tasks" as const, label: copy.tasks },
          { id: "long_term" as const, label: copy.longTerm },
          { id: "daily" as const, label: copy.daily },
          { id: "mandatory" as const, label: copy.mandatory },
          { id: "specification" as const, label: copy.specification },
        ].map((tab) => (
          <button
            key={tab.id}
            className={`memory-tab${activeTab === tab.id ? " active" : ""}`}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.id ? "true" : "false"}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === "tasks" ? (
        <section className="memory-panel memory-panel-tasks active">
          <section className="task-history-view" data-task-history-view>
            <form
              className="task-filter-form page-filter-form page-filter-grid-4"
              onSubmit={(event) => {
                event.preventDefault();
                void applyTaskFilters(taskDraftFilters, 1);
              }}
            >
              <label>
                <span>{copy.status}</span>
                <select
                  value={taskDraftFilters.status}
                  onChange={(event) => setTaskDraftFilters((current) => ({ ...current, status: event.target.value }))}
                >
                  <option value="">-</option>
                  <option value="queued">queued</option>
                  <option value="running">running</option>
                  <option value="success">success</option>
                  <option value="failed">failed</option>
                  <option value="canceled">canceled</option>
                </select>
              </label>
              <label>
                <span>{copy.taskType}</span>
                <input
                  type="text"
                  value={taskDraftFilters.taskType}
                  onChange={(event) => setTaskDraftFilters((current) => ({ ...current, taskType: event.target.value }))}
                  placeholder="release"
                />
              </label>
              <label>
                <span>{copy.start}</span>
                <input
                  type="datetime-local"
                  value={taskDraftFilters.startAt}
                  onChange={(event) => setTaskDraftFilters((current) => ({ ...current, startAt: event.target.value }))}
                />
              </label>
              <label>
                <span>{copy.end}</span>
                <input
                  type="datetime-local"
                  value={taskDraftFilters.endAt}
                  onChange={(event) => setTaskDraftFilters((current) => ({ ...current, endAt: event.target.value }))}
                />
              </label>
              <div className="task-filter-actions">
                <button type="submit">{copy.apply}</button>
                <button
                  type="button"
                  onClick={() => {
                    setTaskDraftFilters(DEFAULT_TASK_FILTERS);
                    void applyTaskFilters(DEFAULT_TASK_FILTERS, 1);
                  }}
                >
                  {copy.reset}
                </button>
              </div>
            </form>

            <section className="route-master-detail route-master-detail-memory">
              <div className="route-data-table-wrap">
                {taskItems.length ? (
                  <table className="route-data-table" aria-label="Task History">
                    <thead>
                      <tr>
                        <th>ID</th>
                        <th>{copy.taskType}</th>
                        <th>{copy.status}</th>
                        <th>{copy.start}</th>
                        <th>{copy.end}</th>
                        <th>Tags</th>
                      </tr>
                    </thead>
                    <tbody>
                      {taskItems.map((item) => {
                        const taskID = normalizeText(item.task_id);
                        const active = normalizeText(taskDetail?.meta?.task_id) === taskID;
                        return (
                          <tr key={taskID} className={active ? "is-active" : undefined}>
                            <td>
                              <button
                                className="route-table-select"
                                type="button"
                                onClick={() => {
                                  void loadTaskDetail(taskID);
                                }}
                              >
                                {taskID}
                              </button>
                            </td>
                            <td>{normalizeText(item.task_type)}</td>
                            <td>
                              <span className={`task-summary-status ${taskStatusClassName(item.status)}`}>
                                {normalizeText(item.status)}
                              </span>
                            </td>
                            <td>{formatDateTime(item.updated_at)}</td>
                            <td>{formatDateTime(item.finished_at)}</td>
                            <td>{Array.isArray(item.tags) && item.tags.length ? item.tags.join(", ") : "-"}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                ) : (
                  <p className="route-empty">{copy.noTaskHistory}</p>
                )}

                <div className="task-summary-pagination-wrap">
                  <div className="task-summary-pagination">
                    <p><span>{copy.page}</span><strong>{String(pagination.page || 1)}</strong></p>
                    <p><span>Total</span><strong>{String(pagination.total || 0)}</strong></p>
                    <button
                      className="task-summary-next"
                      type="button"
                      disabled={!pagination.has_next}
                      onClick={() => {
                        void applyTaskFilters(taskFilters, Number(pagination.page || 1) + 1);
                      }}
                    >
                      {copy.nextPage}
                    </button>
                  </div>
                </div>
              </div>

              <aside className="route-detail-panel memory-detail-panel">
                <div className="task-detail-body">
                  {taskDetail ? (
                    <>
                      <div className="route-detail-panel-head">
                        <div>
                          <h4>{copy.taskDetail}</h4>
                          <p>{normalizeText(taskDetail.meta?.task_id)}</p>
                        </div>
                      </div>
                      <TaskDetailView
                        copy={copy}
                        detail={taskDetail}
                        logs={taskLogs}
                        artifacts={taskArtifacts}
                        onLoadLogs={() => {
                          void loadTaskLogs();
                        }}
                        onLoadArtifacts={() => {
                          void loadTaskArtifacts();
                        }}
                        onRebuild={() => {
                          void rebuildTaskSummary();
                        }}
                        onBack={() => {
                          setTaskDetail(null);
                          setTaskLogs(null);
                          setTaskArtifacts(null);
                        }}
                      />
                    </>
                  ) : (
                    <div className="task-detail-placeholder">{copy.selectTask}</div>
                  )}
                </div>
              </aside>
            </section>
          </section>
        </section>
      ) : null}

      {activeTab === "long_term" ? (
        <section className="memory-panel active">
          <MemoryDocumentCard
            title={copy.longTerm}
            copy={copy}
            payload={memoryPayload?.long_term}
            empty={copy.noLongTerm}
          />
        </section>
      ) : null}

      {activeTab === "daily" ? (
        <section className="memory-panel active">
          <RouteCard
            title={copy.daily}
            type="memory"
            enabled={true}
            statusEnabledLabel={copy.readOnly}
            statusDisabledLabel={copy.readOnly}
          >
            <RouteFieldRow
              label={copy.sourceDirectory}
              value={memoryPayload?.daily?.directory}
              copyLabel="Copy value"
              copyable={true}
              mono={true}
              multiline={true}
            />
          </RouteCard>
          <div className="memory-daily-list">
            {renderDailyCards(memoryPayload?.daily?.items, copy)}
          </div>
        </section>
      ) : null}

      {activeTab === "mandatory" ? (
        <section className="memory-panel active">
          <MemoryDocumentCard
            title={copy.mandatory}
            copy={copy}
            payload={memoryPayload?.mandatory}
            empty={copy.noMandatory}
          />
        </section>
      ) : null}

      {activeTab === "specification" ? (
        <section className="memory-panel memory-panel-spec active">
          <MemorySpecificationCard copy={copy} payload={memoryPayload?.specification} />
        </section>
      ) : null}
    </section>
  );

  async function applyTaskFilters(filters: typeof DEFAULT_TASK_FILTERS, page: number) {
    const payload = await apiClient.get<TaskListPayload>(buildTaskHistoryQuery(filters, page));
    setTaskFilters(filters);
    setTaskDraftFilters(filters);
    setTaskList(payload);
  }

  async function loadTaskDetail(taskID: string) {
    const payload = await apiClient.get<TaskDetailPayload>(`/api/memory/tasks/${encodeURIComponent(taskID)}`);
    setTaskDetail(payload);
    setTaskLogs(null);
    setTaskArtifacts(null);
  }

  async function loadTaskLogs(cursor = 0) {
    const taskID = normalizeText(taskDetail?.meta?.task_id);
    if (!taskID) {
      return;
    }
    const payload = await apiClient.get<TaskLogsPayload>(`/api/memory/tasks/${encodeURIComponent(taskID)}/logs?cursor=${cursor}&limit=20`);
    setTaskLogs(payload);
  }

  async function loadTaskArtifacts() {
    const taskID = normalizeText(taskDetail?.meta?.task_id);
    if (!taskID) {
      return;
    }
    const payload = await apiClient.get<TaskArtifactsPayload>(`/api/memory/tasks/${encodeURIComponent(taskID)}/artifacts`);
    setTaskArtifacts(payload);
  }

  async function rebuildTaskSummary() {
    const taskID = normalizeText(taskDetail?.meta?.task_id);
    if (!taskID) {
      return;
    }
    await apiClient.post(`/api/memory/tasks/${encodeURIComponent(taskID)}/rebuild-summary`);
    await loadTaskDetail(taskID);
  }
}

function MemoryDocumentCard({
  title,
  copy,
  payload,
  empty,
}: {
  title: string;
  copy: Copy;
  payload?: MemoryDocument;
  empty: string;
}) {
  return (
    <RouteCard
      title={title}
      type="memory"
      enabled={true}
      statusEnabledLabel={copy.readOnly}
      statusDisabledLabel={copy.readOnly}
      body={
        payload?.error ? (
          <p className="route-error">{copy.loadFailed(payload.error)}</p>
        ) : payload?.exists && normalizeText(payload.content) ? (
          <pre className="memory-content">{normalizeText(payload.content)}</pre>
        ) : (
          <p className="route-empty">{empty}</p>
        )
      }
    >
      <RouteFieldRow label={copy.path} value={payload?.path} copyLabel="Copy value" copyable={true} mono={true} multiline={true} />
      <RouteFieldRow label={copy.updated} value={formatDateTime(payload?.updated_at)} copyLabel="Copy value" />
      <RouteFieldRow label={copy.readOnly} value={copy.readOnly} copyLabel="Copy value" />
    </RouteCard>
  );
}

function MemorySpecificationCard({
  copy,
  payload,
}: {
  copy: Copy;
  payload?: MemoryDocument;
}) {
  const content = normalizeText(payload?.content);
  const sections = splitMarkdownSections(content);

  return (
    <RouteCard
      title={copy.specification}
      type="memory"
      enabled={true}
      statusEnabledLabel={copy.readOnly}
      statusDisabledLabel={copy.readOnly}
      body={
        payload?.error ? (
          <p className="route-error">{copy.loadFailed(payload.error)}</p>
        ) : payload?.exists && content ? (
          sections.length ? (
            <div className="memory-spec-sections">
              {sections.map((section) => (
                <section key={section.title} className="memory-spec-section">
                  <h5 className="memory-spec-title">{section.title}</h5>
                  <pre className="memory-content">{section.content.trim()}</pre>
                </section>
              ))}
            </div>
          ) : (
            <pre className="memory-content">{content}</pre>
          )
        ) : (
          <p className="route-empty">{copy.noSpecification}</p>
        )
      }
    >
      <RouteFieldRow label={copy.path} value={payload?.path} copyLabel="Copy value" copyable={true} mono={true} multiline={true} />
      <RouteFieldRow label={copy.updated} value={formatDateTime(payload?.updated_at)} copyLabel="Copy value" />
      <RouteFieldRow label={copy.readOnly} value={copy.readOnly} copyLabel="Copy value" />
    </RouteCard>
  );
}

function TaskDetailView({
  copy,
  detail,
  logs,
  artifacts,
  onLoadLogs,
  onLoadArtifacts,
  onRebuild,
  onBack,
}: {
  copy: Copy;
  detail: TaskDetailPayload;
  logs: TaskLogsPayload | null;
  artifacts: TaskArtifactsPayload | null;
  onLoadLogs: () => void;
  onLoadArtifacts: () => void;
  onRebuild: () => void;
  onBack: () => void;
}) {
  const meta = detail.meta;
  const refs = Array.isArray(detail.summary_refs) ? detail.summary_refs : [];

  return (
    <section className="task-detail-card">
      <header className="task-detail-head">
        <div className="task-detail-id-wrap">
          <h5 className="task-summary-id">{normalizeText(meta?.task_id)}</h5>
        </div>
        <span className={`task-summary-status ${taskStatusClassName(meta?.status)}`}>{normalizeText(meta?.status)}</span>
      </header>
      <div className="task-detail-meta route-meta">
        <RouteFieldRow label={copy.taskType} value={meta?.task_type} copyLabel="Copy value" />
        <RouteFieldRow label={copy.session} value={meta?.session_id} copyLabel="Copy value" copyable={true} mono={true} />
        <RouteFieldRow label={copy.sourceMessage} value={meta?.source_message_id} copyLabel="Copy value" copyable={true} mono={true} />
        <RouteFieldRow label={copy.progress} value={meta?.progress} copyLabel="Copy value" />
        <RouteFieldRow label={copy.retryCount} value={meta?.retry_count} copyLabel="Copy value" />
        <RouteFieldRow label={copy.updated} value={formatDateTime(meta?.updated_at)} copyLabel="Copy value" />
        <RouteFieldRow label={copy.finished} value={formatDateTime(meta?.finished_at)} copyLabel="Copy value" />
        <RouteFieldRow label={copy.lastHeartbeat} value={formatDateTime(meta?.last_heartbeat_at || meta?.timeout_at)} copyLabel="Copy value" />
      </div>
      <div className="task-detail-section">
        <h5>{copy.refs}</h5>
        {refs.length ? (
          <ul className="task-detail-refs">
            {refs.map((item) => (
              <li key={`${normalizeText(item.tier)}-${normalizeText(item.path)}`}>
                <strong>{normalizeText(item.tier)}</strong>
                <span>{normalizeText(item.date)}</span>
                <code>{normalizeText(item.path)}</code>
              </li>
            ))}
          </ul>
        ) : (
          <p className="route-empty">-</p>
        )}
      </div>
      <div className="task-detail-actions">
        <button type="button" onClick={onLoadLogs}>{copy.loadLogs}</button>
        <button type="button" onClick={onLoadArtifacts}>{copy.loadArtifacts}</button>
        <button type="button" onClick={onRebuild}>{copy.rebuild}</button>
        <button type="button" onClick={onBack}>{copy.back}</button>
      </div>
      <div className="task-detail-section">
        <h5>Logs</h5>
        <div className="task-detail-logs">
          {renderLogs(logs, copy)}
        </div>
      </div>
      <div className="task-detail-section">
        <h5>Artifacts</h5>
        <div className="task-detail-artifacts">
          {renderArtifacts(artifacts, copy)}
        </div>
      </div>
    </section>
  );
}

function renderDailyCards(items: DailyMemoryItem[] | undefined, copy: Copy) {
  const dailyItems = Array.isArray(items) ? items : [];
  if (!dailyItems.length) {
    return <p className="route-empty">{copy.noDaily}</p>;
  }
  return dailyItems.map((item) => (
    <RouteCard
      key={normalizeText(item.date || item.path)}
      title={normalizeText(item.date)}
      type="memory"
      enabled={true}
      statusEnabledLabel={copy.readOnly}
      statusDisabledLabel={copy.readOnly}
      body={
        item.error ? (
          <p className="route-error">{copy.loadFailed(item.error)}</p>
        ) : normalizeText(item.content) ? (
          <>
            <p className="memory-summary"><span>{copy.summary}</span><strong>{summarizeMemoryContent(item.content)}</strong></p>
            <pre className="memory-content">{normalizeText(item.content)}</pre>
          </>
        ) : (
          <p className="route-empty">{copy.noDaily}</p>
        )
      }
    >
      <RouteFieldRow label={copy.date} value={item.date} copyLabel="Copy value" />
      <RouteFieldRow label={copy.path} value={item.path} copyLabel="Copy value" copyable={true} mono={true} multiline={true} />
      <RouteFieldRow label={copy.updated} value={formatDateTime(item.updated_at)} copyLabel="Copy value" />
      <RouteFieldRow label={copy.readOnly} value={copy.readOnly} copyLabel="Copy value" />
    </RouteCard>
  ));
}

function renderLogs(logs: TaskLogsPayload | null, copy: Copy) {
  if (!logs) {
    return <p className="route-empty">{copy.noLogs}</p>;
  }
  if (logs.error_code) {
    return <p className="route-error">{copy.logsHint}</p>;
  }
  const items = Array.isArray(logs.items) ? logs.items : [];
  if (!items.length) {
    return <p className="route-empty">{copy.noLogs}</p>;
  }
  return (
    <ul className="task-detail-log-list">
      {items.map((item) => (
        <li key={`${normalizeText(item.seq)}-${normalizeText(item.stage)}`} className="task-detail-list-item">
          <div className="task-detail-list-head">
            <strong>{`#${normalizeText(item.seq)}`}</strong>
            <span>{normalizeText(item.stage)}</span>
            <span>{normalizeText(item.level)}</span>
            <span>{formatDateTime(item.created_at || item.timestamp)}</span>
          </div>
          <pre className="task-detail-list-content">{normalizeText(item.message)}</pre>
        </li>
      ))}
    </ul>
  );
}

function renderArtifacts(artifacts: TaskArtifactsPayload | null, copy: Copy) {
  if (!artifacts) {
    return <p className="route-empty">{copy.noArtifacts}</p>;
  }
  if (artifacts.error) {
    return <p className="route-error">{copy.loadFailed(artifacts.error)}</p>;
  }
  const items = Array.isArray(artifacts.items) ? artifacts.items : [];
  if (!items.length) {
    return <p className="route-empty">{copy.noArtifacts}</p>;
  }
  return (
    <ul className="task-detail-artifact-list">
      {items.map((item) => (
        <li key={normalizeText(item.artifact_id || item.name)} className="task-detail-list-item">
          <div className="task-detail-list-head">
            <strong>{normalizeText(item.artifact_type || item.name)}</strong>
            <span>{formatDateTime(item.created_at)}</span>
          </div>
          <p className="task-detail-list-content">{normalizeText(item.summary || item.content_type)}</p>
        </li>
      ))}
    </ul>
  );
}

function splitMarkdownSections(content: string) {
  const text = normalizeText(content).replace(/\r\n/g, "\n");
  if (!text) {
    return [];
  }
  const rows = text.split("\n");
  const sections: Array<{ title: string; content: string }> = [];
  let current: { title: string; content: string } | null = null;

  for (const row of rows) {
    const heading = row.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      if (current) {
        sections.push(current);
      }
      current = { title: heading[2].trim(), content: "" };
      continue;
    }
    if (!current) {
      current = { title: "Overview", content: "" };
    }
    current.content += `${row}\n`;
  }

  if (current) {
    sections.push(current);
  }

  return sections;
}

function summarizeMemoryContent(content?: string) {
  const rows = normalizeText(content)
    .split("\n")
    .map((row) => row.trim())
    .filter(Boolean);
  return rows[0] || "-";
}

function taskStatusClassName(status: unknown) {
  const normalized = normalizeText(status).toLowerCase();
  if (!normalized) {
    return "";
  }
  return `is-${normalized}`;
}

function buildTaskHistoryQuery(filters: typeof DEFAULT_TASK_FILTERS, page: number) {
  const params = new URLSearchParams();
  params.set("page", String(page));
  params.set("page_size", "10");
  if (normalizeText(filters.status)) {
    params.set("status", normalizeText(filters.status));
  }
  if (normalizeText(filters.taskType)) {
    params.set("task_type", normalizeText(filters.taskType));
  }
  if (normalizeText(filters.startAt)) {
    params.set("start_at", toISOStringOrEmpty(filters.startAt));
  }
  if (normalizeText(filters.endAt)) {
    params.set("end_at", toISOStringOrEmpty(filters.endAt));
  }
  return `/api/memory/tasks?${params.toString()}`;
}

function toISOStringOrEmpty(value: string) {
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? "" : parsed.toISOString();
}

function normalizeText(value: unknown) {
  return String(value || "").trim();
}
