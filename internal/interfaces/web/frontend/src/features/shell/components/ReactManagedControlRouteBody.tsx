import { useEffect, useState, type ReactNode } from "react";
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

type CronJobRouteRecord = {
  id?: string;
  name?: string;
  enabled?: boolean;
  builtin?: boolean;
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

type ProjectSkillCatalogRecord = {
  id?: string;
  name?: string;
  description?: string;
  configured_enabled?: boolean;
  codex_visible?: boolean;
  sync_status?: string;
  duplicate?: boolean;
  duplicate_group?: string;
};

type CodexSkillDependency = {
  type?: string;
  value?: string;
  command?: string;
  description?: string;
};

type CodexSkillCatalogRecord = {
  name?: string;
  description?: string;
  enabled?: boolean;
  scope?: string;
  location?: string;
  display_name?: string;
  short_description?: string;
  dependencies?: CodexSkillDependency[];
  duplicate?: boolean;
  duplicate_group?: string;
};

type SkillCatalogError = {
  code?: string;
  message?: string;
  location?: string;
};

type RouteRecord = ControlRouteRecord | CronJobRouteRecord;

type ControlRouteResponse = {
  items?: RouteRecord[];
  project_skills?: ProjectSkillCatalogRecord[];
  codex_skills?: CodexSkillCatalogRecord[];
  errors?: SkillCatalogError[];
};

type ControlRouteCopy = {
  loading: string;
  statusEnabled: string;
  statusDisabled: string;
  copyValue: string;
  fieldID: string;
  fieldType: string;
  fieldDescription: string;
  fieldName: string;
  fieldScope: string;
  fieldVersion: string;
  fieldScheduleMode: string;
  fieldTimezone: string;
  fieldCronExpression: string;
  fieldInterval: string;
  fieldPrompt: string;
  fieldRetryLimit: string;
  fieldOrigin: string;
  fieldLocation: string;
  fieldDependencies: string;
  fieldSyncStatus: string;
  fieldCodexVisible: string;
  builtinJob: string;
  customJob: string;
  enableJob: string;
  disableJob: string;
  actionFailed: (message: string) => string;
  emptySkills: string;
  emptyCronJobs: string;
  alter0Section: string;
  alter0SectionHint: string;
  codexSection: string;
  codexSectionHint: string;
  duplicateName: string;
  visibleYes: string;
  visibleNo: string;
  catalogIssues: string;
  retryCatalog: string;
  syncReady: string;
  syncMissing: string;
  syncStale: string;
  syncUnknown: string;
  syncCodexDisabled: string;
  loadFailed: (message: string) => string;
};

const CONTROL_ROUTE_COPY: Record<LegacyShellLanguage, ControlRouteCopy> = {
  en: {
    loading: "Loading...",
    statusEnabled: "Enabled",
    statusDisabled: "Disabled",
    copyValue: "Copy value",
    fieldID: "ID",
    fieldType: "Type",
    fieldDescription: "Description",
    fieldName: "Name",
    fieldScope: "Scope",
    fieldVersion: "Version",
    fieldScheduleMode: "Schedule Mode",
    fieldTimezone: "Timezone",
    fieldCronExpression: "Cron Expression",
    fieldInterval: "Interval",
    fieldPrompt: "Prompt",
    fieldRetryLimit: "Retry Limit",
    fieldOrigin: "Origin",
    fieldLocation: "Location",
    fieldDependencies: "Dependencies",
    fieldSyncStatus: "Sync",
    fieldCodexVisible: "Codex Visible",
    builtinJob: "Built-in",
    customJob: "Custom",
    enableJob: "Enable job",
    disableJob: "Disable job",
    actionFailed: (message) => `Action failed: ${message}`,
    emptySkills: "No Skills available.",
    emptyCronJobs: "No schedules available.",
    alter0Section: "Alter0 Built-in",
    alter0SectionHint: "Business Skills maintained and synchronized by Alter0.",
    codexSection: "Codex Skills",
    codexSectionHint: "Read-only catalog reported by the active Codex runtime.",
    duplicateName: "Duplicate name",
    visibleYes: "Visible",
    visibleNo: "Not visible",
    catalogIssues: "Catalog issues",
    retryCatalog: "Retry",
    syncReady: "Ready",
    syncMissing: "Not installed",
    syncStale: "Removal pending",
    syncUnknown: "Unavailable",
    syncCodexDisabled: "Disabled in Codex",
    loadFailed: (message) => `Load failed: ${message}`,
  },
  zh: {
    loading: "加载中...",
    statusEnabled: "启用",
    statusDisabled: "停用",
    copyValue: "复制内容",
    fieldID: "ID",
    fieldType: "类型",
    fieldDescription: "描述",
    fieldName: "名称",
    fieldScope: "范围",
    fieldVersion: "版本",
    fieldScheduleMode: "调度模式",
    fieldTimezone: "时区",
    fieldCronExpression: "Cron 表达式",
    fieldInterval: "间隔",
    fieldPrompt: "任务输入",
    fieldRetryLimit: "重试次数",
    fieldOrigin: "来源",
    fieldLocation: "位置",
    fieldDependencies: "依赖",
    fieldSyncStatus: "同步状态",
    fieldCodexVisible: "Codex 可见",
    builtinJob: "内置",
    customJob: "自定义",
    enableJob: "启用任务",
    disableJob: "停用任务",
    actionFailed: (message) => `操作失败：${message}`,
    emptySkills: "暂无可用技能。",
    emptyCronJobs: "暂无定时任务。",
    alter0Section: "Alter0 内置",
    alter0SectionHint: "由 Alter0 维护并同步的业务 Skill。",
    codexSection: "Codex Skills",
    codexSectionHint: "当前 Codex 运行时实际发现的只读目录。",
    duplicateName: "名称重复",
    visibleYes: "可见",
    visibleNo: "不可见",
    catalogIssues: "目录问题",
    retryCatalog: "重试",
    syncReady: "已同步",
    syncMissing: "未安装",
    syncStale: "待清理",
    syncUnknown: "不可用",
    syncCodexDisabled: "已在 Codex 停用",
    loadFailed: (message) => `加载失败：${message}`,
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
  markdown?: boolean;
};

type RequestState = {
  status: "loading" | "ready" | "error";
  items: RouteRecord[];
  projectSkills: ProjectSkillCatalogRecord[];
  codexSkills: CodexSkillCatalogRecord[];
  catalogErrors: SkillCatalogError[];
  error: string;
};

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
  fields: (item: RouteRecord, copy: ControlRouteCopy) => FieldSpec[];
};

const ROUTE_CONFIG: Record<ReactManagedControlRoute, RouteConfig> = {
  skills: {
    path: "/api/control/skill-catalog",
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
      {
        label: copy.fieldDescription,
        value: (item as ControlRouteRecord).description,
        multiline: true,
        preview: true,
        clampLines: 3,
        markdown: true,
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
      {
        label: copy.fieldOrigin,
        value: (item as CronJobRouteRecord).builtin ? copy.builtinJob : copy.customJob,
      },
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
        markdown: true,
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
  const [state, setState] = useState<RequestState>({
    status: "loading",
    items: [],
    projectSkills: [],
    codexSkills: [],
    catalogErrors: [],
    error: "",
  });
  const [reloadToken, setReloadToken] = useState(0);
  const [actionBusyID, setActionBusyID] = useState("");
  const [actionError, setActionError] = useState("");

  useEffect(() => {
    let disposed = false;

    setState({
      status: "loading",
      items: [],
      projectSkills: [],
      codexSkills: [],
      catalogErrors: [],
      error: "",
    });
    setActionBusyID("");
    setActionError("");

    void createAPIClient()
      .get<ControlRouteResponse>(routeConfig.path)
      .then((payload) => {
        if (disposed) {
          return;
        }
        setState({
          status: "ready",
          items: Array.isArray(payload?.items) ? payload.items : [],
          projectSkills: Array.isArray(payload?.project_skills) ? payload.project_skills : [],
          codexSkills: Array.isArray(payload?.codex_skills) ? payload.codex_skills : [],
          catalogErrors: Array.isArray(payload?.errors) ? payload.errors : [],
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
          projectSkills: [],
          codexSkills: [],
          catalogErrors: [],
          error: error instanceof Error ? error.message : "unknown_error",
        });
      });

    return () => {
      disposed = true;
    };
  }, [reloadToken, routeConfig]);

  if (state.status === "loading") {
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (state.status === "error") {
    return <p className="route-error">{copy.loadFailed(state.error)}</p>;
  }

  if (route === "skills") {
    return (
      <SkillCatalogView
        copy={copy}
        projectSkills={state.projectSkills}
        codexSkills={state.codexSkills}
        errors={state.catalogErrors}
        onRetry={() => setReloadToken((current) => current + 1)}
      />
    );
  }

  if (!state.items.length) {
    return <p className="route-empty">{routeConfig.empty(copy)}</p>;
  }

  async function toggleCronJobEnabled(item: RouteRecord) {
    if (route !== "cron-jobs") {
      return;
    }
    const cronJob = item as CronJobRouteRecord;
    const id = normalizeText(cronJob.id);
    if (id === "-") {
      return;
    }
    const enabled = !Boolean(cronJob.enabled);
    setActionError("");
    setActionBusyID(id);
    try {
      const updated = await createAPIClient().put<CronJobRouteRecord>(
        `/api/control/cron/jobs/${encodeURIComponent(id)}`,
        { enabled },
      );
      setState((current) => ({
        ...current,
        items: current.items.map((candidate) =>
          routeConfig.key(candidate) === id ? (updated as RouteRecord) : candidate,
        ),
      }));
    } catch (error) {
      setActionError(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    } finally {
      setActionBusyID("");
    }
  }

  function renderCronJobActions(item: RouteRecord) {
    if (route !== "cron-jobs") {
      return null;
    }
    const cronJob = item as CronJobRouteRecord;
    if (!cronJob.builtin) {
      return null;
    }
    const id = normalizeText(cronJob.id);
    const enabled = Boolean(cronJob.enabled);
    const label = enabled ? copy.disableJob : copy.enableJob;
    return (
      <button
        className="route-card-action"
        type="button"
        disabled={actionBusyID === id}
        aria-label={label}
        onClick={() => void toggleCronJobEnabled(item)}
      >
        {label}
      </button>
    );
  }

  return (
    <>
      {actionError ? <p className="route-error">{actionError}</p> : null}
      <section className="control-route-grid" data-control-route-grid={route}>
        {state.items.map((item) => (
          <RouteCard
            key={`${route}-${routeConfig.key(item)}`}
            title={routeConfig.title(item)}
            type={routeConfig.type(item)}
            enabled={routeConfig.enabled(item)}
            statusEnabledLabel={copy.statusEnabled}
            statusDisabledLabel={copy.statusDisabled}
            actions={renderCronJobActions(item)}
          >
            {routeConfig.fields(item, copy).map((field) => (
              <RouteFieldRow key={`${field.label}-${normalizeText(field.value)}`} copyLabel={copy.copyValue} {...field} />
            ))}
          </RouteCard>
        ))}
      </section>
    </>
  );
}

function SkillCatalogView({
  copy,
  projectSkills,
  codexSkills,
  errors,
  onRetry,
}: {
  copy: ControlRouteCopy;
  projectSkills: ProjectSkillCatalogRecord[];
  codexSkills: CodexSkillCatalogRecord[];
  errors: SkillCatalogError[];
  onRetry: () => void;
}) {
  return (
    <div className="skill-catalog" data-skill-catalog>
      <SkillCatalogSection title={copy.alter0Section} hint={copy.alter0SectionHint}>
        {projectSkills.length ? (
          <section className="control-route-grid" data-control-route-grid="alter0-skills">
            {projectSkills.map((item) => (
              <RouteCard
                key={`project-skill-${normalizeText(item.id)}`}
                title={item.name || item.id}
                type="alter0 skill"
                enabled={Boolean(item.configured_enabled)}
                statusEnabledLabel={copy.statusEnabled}
                statusDisabledLabel={copy.statusDisabled}
                actions={item.duplicate ? <span className="skill-catalog-conflict">{copy.duplicateName}</span> : null}
              >
                <RouteFieldRow label={copy.fieldID} value={item.id} copyLabel={copy.copyValue} mono />
                <RouteFieldRow label={copy.fieldSyncStatus} value={syncStatusLabel(item.sync_status, copy)} copyLabel={copy.copyValue} />
                <RouteFieldRow
                  label={copy.fieldCodexVisible}
                  value={item.codex_visible ? copy.visibleYes : copy.visibleNo}
                  copyLabel={copy.copyValue}
                />
                <RouteFieldRow
                  label={copy.fieldDescription}
                  value={item.description}
                  copyLabel={copy.copyValue}
                  multiline
                  markdown
                />
              </RouteCard>
            ))}
          </section>
        ) : (
          <p className="route-empty">{copy.emptySkills}</p>
        )}
      </SkillCatalogSection>

      <SkillCatalogSection title={copy.codexSection} hint={copy.codexSectionHint}>
        {codexSkills.length ? (
          <section className="control-route-grid" data-control-route-grid="codex-skills">
            {codexSkills.map((item, index) => (
              <RouteCard
                key={`codex-skill-${normalizeText(item.name)}-${normalizeText(item.location)}-${index}`}
                title={item.display_name || item.name}
                type="codex skill"
                enabled={Boolean(item.enabled)}
                statusEnabledLabel={copy.statusEnabled}
                statusDisabledLabel={copy.statusDisabled}
                actions={item.duplicate ? <span className="skill-catalog-conflict">{copy.duplicateName}</span> : null}
              >
                <RouteFieldRow label={copy.fieldName} value={item.name} copyLabel={copy.copyValue} mono />
                <RouteFieldRow label={copy.fieldLocation} value={skillLocationLabel(item.location)} copyLabel={copy.copyValue} />
                <RouteFieldRow label={copy.fieldScope} value={item.scope} copyLabel={copy.copyValue} />
                <RouteFieldRow
                  label={copy.fieldDependencies}
                  value={skillDependenciesLabel(item.dependencies)}
                  copyLabel={copy.copyValue}
                  mono
                />
                <RouteFieldRow
                  label={copy.fieldDescription}
                  value={item.short_description || item.description}
                  copyLabel={copy.copyValue}
                  multiline
                  markdown
                />
              </RouteCard>
            ))}
          </section>
        ) : (
          <p className="route-empty">{copy.emptySkills}</p>
        )}
        {errors.length ? (
          <aside className="skill-catalog-errors" aria-label={copy.catalogIssues}>
            <div>
              <strong>{copy.catalogIssues}</strong>
              {errors.map((error, index) => (
                <p key={`${normalizeText(error.code)}-${index}`}>
                  <span>{skillLocationLabel(error.location)}</span>
                  {normalizeText(error.message)}
                </p>
              ))}
            </div>
            <button type="button" className="route-card-action" onClick={onRetry}>{copy.retryCatalog}</button>
          </aside>
        ) : null}
      </SkillCatalogSection>
    </div>
  );
}

function SkillCatalogSection({ title, hint, children }: { title: string; hint: string; children: ReactNode }) {
  return (
    <section className="skill-catalog-section">
      <header className="skill-catalog-section-head">
        <h3>{title}</h3>
        <p>{hint}</p>
      </header>
      {children}
    </section>
  );
}

function skillLocationLabel(location: unknown) {
  switch (String(location || "").trim()) {
    case "alter0": return "Alter0 Built-in";
    case "user_agents": return "~/.agents/skills";
    case "codex_home": return "$CODEX_HOME/skills";
    case "repo": return "<repo>/.agents/skills";
    case "admin": return "/etc/codex/skills";
    case "system": return "Codex System";
    default: return "Other Codex location";
  }
}

function skillDependenciesLabel(dependencies: CodexSkillDependency[] | undefined) {
  if (!Array.isArray(dependencies) || !dependencies.length) {
    return "-";
  }
  return dependencies
    .map((dependency) => normalizeText(dependency.value || dependency.command))
    .filter((value) => value !== "-")
    .join(", ") || "-";
}

function syncStatusLabel(status: unknown, copy: ControlRouteCopy) {
  switch (String(status || "").trim()) {
    case "ready": return copy.syncReady;
    case "disabled": return copy.statusDisabled;
    case "missing": return copy.syncMissing;
    case "stale": return copy.syncStale;
    case "codex_disabled": return copy.syncCodexDisabled;
    default: return copy.syncUnknown;
  }
}
