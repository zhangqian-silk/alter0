import { useEffect, useRef, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import type { LegacyShellLanguage } from "../legacyShellCopy";

type AccountRecord = {
  name?: string;
  snapshot?: {
    account_name?: string;
    email?: string;
    plan?: string;
  };
};

type LiveSnapshot = {
  account_name?: string;
  email?: string;
  plan?: string;
};

type AccountStatus = {
  record?: AccountRecord;
  current?: boolean;
  quota?: {
    hourly?: { remaining_percent?: number };
    weekly?: { remaining_percent?: number };
    plan?: string;
  };
  error?: string;
};

type CurrentStatus = {
  live?: LiveSnapshot | null;
  managed?: AccountRecord | null;
  auth_path?: string;
};

type LoginSession = {
  id?: string;
  account_name?: string;
  status?: string;
  logs?: string;
  error?: string;
};

type AccountResponse = {
  items?: AccountStatus[];
  active?: CurrentStatus | null;
};

type RequestState =
  | { status: "loading"; error: string }
  | { status: "ready"; error: string }
  | { status: "error"; error: string };

type CodexAccountsCopy = {
  loading: string;
  empty: string;
  emptyHint: string;
  unmanagedCurrentHint: string;
  overview: string;
  managedAccounts: string;
  operationsTitle: string;
  operationsSubtitle: string;
  loginSessionTitle: string;
  loginSessionEmpty: string;
  accountName: string;
  authFile: string;
  chooseFile: string;
  noFileSelected: string;
  overwrite: string;
  importAccount: string;
  startLogin: string;
  current: string;
  saved: string;
  activeAccount: string;
  managedCount: string;
  activePath: string;
  quotaHourly: string;
  quotaWeekly: string;
  plan: string;
  state: string;
  sessionID: string;
  sessionAccount: string;
  switchTo: (name: string) => string;
  imported: string;
  switched: (name: string) => string;
  loginStarted: string;
  loginCompleted: string;
  loginFailed: string;
  loginLogs: string;
  loginStatus: string;
  loadFailed: (message: string) => string;
  actionFailed: (message: string) => string;
};

const CODEX_ACCOUNTS_COPY: Record<LegacyShellLanguage, CodexAccountsCopy> = {
  en: {
    loading: "Loading...",
    empty: "No managed Codex accounts yet.",
    emptyHint: "Import auth.json to create the first managed snapshot.",
    unmanagedCurrentHint: "Current auth.json is active but not managed yet. Import it to create the first managed snapshot.",
    overview: "Runtime Overview",
    managedAccounts: "Managed Accounts",
    operationsTitle: "Import or Add Account",
    operationsSubtitle: "Import an existing auth.json or start an isolated Codex login session without replacing the active runtime account immediately.",
    loginSessionTitle: "Login Session",
    loginSessionEmpty: "No login session started.",
    accountName: "Account Name",
    authFile: "Auth File",
    chooseFile: "Choose auth.json",
    noFileSelected: "No file selected",
    overwrite: "Overwrite existing account",
    importAccount: "Import auth.json",
    startLogin: "Start Codex Login",
    current: "Current",
    saved: "Saved",
    activeAccount: "Active Account",
    managedCount: "Managed Count",
    activePath: "Active Auth Path",
    quotaHourly: "Hourly Remaining",
    quotaWeekly: "Weekly Remaining",
    plan: "Plan",
    state: "State",
    sessionID: "Session ID",
    sessionAccount: "Target Account",
    switchTo: (name) => `Switch to ${name}`,
    imported: "Account imported.",
    switched: (name) => `Switched to ${name}.`,
    loginStarted: "Login started.",
    loginCompleted: "Login completed.",
    loginFailed: "Login failed.",
    loginLogs: "Login Logs",
    loginStatus: "Login Status",
    loadFailed: (message) => `Load failed: ${message}`,
    actionFailed: (message) => `Action failed: ${message}`,
  },
  zh: {
    loading: "加载中...",
    empty: "暂无托管 Codex 账号。",
    emptyHint: "导入 auth.json 后即可创建第一条托管快照。",
    unmanagedCurrentHint: "当前 auth.json 已生效，但尚未纳入托管；导入后即可创建第一条托管快照。",
    overview: "运行时概览",
    managedAccounts: "托管账号",
    operationsTitle: "导入或新增账号",
    operationsSubtitle: "支持导入现有 auth.json，或启动隔离的 Codex 登录会话，新账号生成后不会立刻替换当前运行时认证。",
    loginSessionTitle: "登录会话",
    loginSessionEmpty: "当前还没有启动登录会话。",
    accountName: "账号名称",
    authFile: "Auth 文件",
    chooseFile: "选择 auth.json",
    noFileSelected: "未选择任何文件",
    overwrite: "覆盖同名账号",
    importAccount: "导入 auth.json",
    startLogin: "启动 Codex 登录",
    current: "当前",
    saved: "已保存",
    activeAccount: "当前账号",
    managedCount: "托管数量",
    activePath: "当前生效 Auth 路径",
    quotaHourly: "小时剩余额度",
    quotaWeekly: "周剩余额度",
    plan: "套餐",
    state: "状态",
    sessionID: "会话 ID",
    sessionAccount: "目标账号",
    switchTo: (name) => `切换到 ${name}`,
    imported: "账号已导入。",
    switched: (name) => `已切换到 ${name}。`,
    loginStarted: "登录已启动。",
    loginCompleted: "登录完成。",
    loginFailed: "登录失败。",
    loginLogs: "登录日志",
    loginStatus: "登录状态",
    loadFailed: (message) => `加载失败：${message}`,
    actionFailed: (message) => `操作失败：${message}`,
  },
};

export function ReactManagedCodexAccountsRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = CODEX_ACCOUNTS_COPY[language];
  const apiClient = createAPIClient();
  const pollTimerRef = useRef<number | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [requestState, setRequestState] = useState<RequestState>({ status: "loading", error: "" });
  const [accounts, setAccounts] = useState<AccountStatus[]>([]);
  const [active, setActive] = useState<CurrentStatus | null>(null);
  const [name, setName] = useState("");
  const [overwrite, setOverwrite] = useState(false);
  const [authFile, setAuthFile] = useState<File | null>(null);
  const [statusMessage, setStatusMessage] = useState("");
  const [statusKind, setStatusKind] = useState<"success" | "error" | "">("");
  const [loginSession, setLoginSession] = useState<LoginSession | null>(null);

  useEffect(() => {
    void reloadAccounts();
    return () => {
      if (pollTimerRef.current != null) {
        window.clearTimeout(pollTimerRef.current);
      }
    };
  }, []);

  async function reloadAccounts(nextMessage = "", nextKind: "success" | "error" | "" = "") {
    setRequestState({ status: "loading", error: "" });
    try {
      const payload = await apiClient.get<AccountResponse>("/api/control/codex/accounts");
      setAccounts(Array.isArray(payload?.items) ? payload.items : []);
      setActive(payload?.active ?? null);
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

  async function onImportAuthFile() {
    if (!authFile) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(copy.authFile));
      return;
    }
    try {
      const authFileContent = await readFileAsText(authFile);
      await apiClient.post("/api/control/codex/accounts", {
        name: name.trim(),
        overwrite,
        auth_file_content: authFileContent,
      });
      setAuthFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
      await reloadAccounts(copy.imported, "success");
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    }
  }

  async function onSwitchAccount(accountName: string) {
    if (!accountName) {
      return;
    }
    try {
      await apiClient.post(`/api/control/codex/accounts/${encodeURIComponent(accountName)}/switch`);
      await reloadAccounts(copy.switched(accountName), "success");
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    }
  }

  async function onStartLogin() {
    try {
      const session = await apiClient.post<LoginSession>("/api/control/codex/accounts/login-sessions", {
        name: name.trim(),
        overwrite,
      });
      setLoginSession(session);
      setStatusKind("success");
      setStatusMessage(copy.loginStarted);
      if (session.id) {
        await refreshLoginSession(session.id);
      }
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    }
  }

  async function refreshLoginSession(sessionID: string) {
    try {
      const session = await apiClient.get<LoginSession>(`/api/control/codex/accounts/login-sessions/${encodeURIComponent(sessionID)}`);
      setLoginSession(session);
      if (session.status === "running") {
        pollTimerRef.current = window.setTimeout(() => {
          void refreshLoginSession(sessionID);
        }, 1500);
        return;
      }
      if (session.status === "succeeded") {
        await reloadAccounts(copy.loginCompleted, "success");
      } else if (session.status === "failed") {
        setStatusKind("error");
        setStatusMessage(copy.loginFailed);
      }
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    }
  }

  if (requestState.status === "loading") {
    return <CodexAccountsLoadingView copy={copy} />;
  }

  if (requestState.status === "error") {
    return <p className="route-error">{copy.loadFailed(requestState.error)}</p>;
  }

  const activeAccount =
    normalizeText(active?.managed?.snapshot?.account_name) ||
    normalizeText(active?.live?.account_name) ||
    normalizeText(active?.live?.email) ||
    normalizeText(active?.managed?.name) ||
    "-";
  const activePath = normalizeText(active?.auth_path) || "-";
  const accountsHint = accounts.length === 0 && active?.live && !active?.managed
    ? copy.unmanagedCurrentHint
    : copy.emptyHint;

  return (
    <section className="codex-accounts-view">
      {statusMessage ? (
        <p className={`codex-accounts-status ${statusKind === "error" ? "is-error" : "is-success"}`}>
          {statusMessage}
        </p>
      ) : null}

      <div className="codex-accounts-overview route-surface">
        <div className="codex-accounts-section-head">
          <div>
            <h4>{copy.overview}</h4>
            <p>{copy.activePath}</p>
          </div>
        </div>
        <div className="codex-accounts-summary-grid">
          <article className="codex-accounts-summary-card">
            <span>{copy.activeAccount}</span>
            <strong title={activeAccount}>{activeAccount}</strong>
          </article>
          <article className="codex-accounts-summary-card">
            <span>{copy.managedCount}</span>
            <strong>{accounts.length}</strong>
          </article>
          <article className="codex-accounts-summary-card is-wide">
            <span>{copy.activePath}</span>
            <strong title={activePath}>{activePath}</strong>
          </article>
        </div>
      </div>

      <div className="codex-accounts-workspace">
        <section className="codex-accounts-main route-surface">
          <div className="codex-accounts-section-head">
            <div>
              <h4>{copy.managedAccounts}</h4>
              <p>{accountsHint}</p>
            </div>
          </div>

          {accounts.length === 0 ? (
            <div className="route-empty-panel codex-accounts-empty">
              <div>
                <strong>{copy.empty}</strong>
                <p>{accountsHint}</p>
              </div>
            </div>
          ) : (
            <div className="codex-accounts-list" role="list" aria-label={copy.managedAccounts}>
              {accounts.map((item) => {
                const account = item.record ?? {};
                const snapshot = account.snapshot ?? {};
                const managedName = normalizeText(account.name);
                const title = normalizeText(snapshot.account_name) || managedName || "-";
                const email = normalizeText(snapshot.email);
                const plan = normalizeText(item.quota?.plan) || normalizeText(snapshot.plan) || "-";
                const hourly = normalizePercent(item.quota?.hourly?.remaining_percent);
                const weekly = normalizePercent(item.quota?.weekly?.remaining_percent);
                const itemKey = `${managedName || title}:${plan}`;

                return (
                  <article
                    key={itemKey}
                    className="codex-account-card"
                    role="listitem"
                    data-testid="codex-account-card"
                  >
                    <div className="codex-account-card-head">
                      <div className="codex-accounts-account-cell">
                        <strong>{title}</strong>
                        <span>{managedName || "-"}</span>
                        {email ? <p>{email}</p> : null}
                      </div>
                      <span className={`codex-account-badge ${item.current ? "is-current" : "is-saved"}`}>
                        {item.current ? copy.current : copy.saved}
                      </span>
                    </div>

                    <div className="codex-account-card-metrics">
                      <div className="codex-account-metric">
                        <span>{copy.plan}</span>
                        <strong>{plan}</strong>
                      </div>
                      <div className="codex-account-metric">
                        <span>{copy.quotaHourly}</span>
                        <strong>{renderPercent(hourly)}</strong>
                      </div>
                      <div className="codex-account-metric">
                        <span>{copy.quotaWeekly}</span>
                        <strong>{renderPercent(weekly)}</strong>
                      </div>
                    </div>

                    {normalizeText(item.error) ? <p className="route-error">{normalizeText(item.error)}</p> : null}

                    <div className="codex-account-card-actions">
                      {item.current ? (
                        <span className="codex-account-card-current">{copy.current}</span>
                      ) : (
                        <button
                          type="button"
                          className="route-primary-button codex-account-switch-button"
                          onClick={() => void onSwitchAccount(managedName)}
                        >
                          {copy.switchTo(title)}
                        </button>
                      )}
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </section>

        <aside className="codex-accounts-side">
          <section className="codex-accounts-panel route-surface">
            <div className="codex-accounts-section-head">
              <div>
                <h4>{copy.operationsTitle}</h4>
                <p>{copy.operationsSubtitle}</p>
              </div>
            </div>

            <form
              className="codex-accounts-form"
              onSubmit={(event) => {
                event.preventDefault();
                void onImportAuthFile();
              }}
            >
              <label className="codex-accounts-field">
                <span>{copy.accountName}</span>
                <input value={name} onChange={(event) => setName(event.target.value)} />
              </label>

              <div className="codex-accounts-field">
                <span>{copy.authFile}</span>
                <div className="codex-accounts-file-picker">
                  <button
                    type="button"
                    className="route-primary-button codex-accounts-file-button"
                    onClick={() => fileInputRef.current?.click()}
                  >
                    {copy.chooseFile}
                  </button>
                  <span className="codex-accounts-file-name" title={authFile?.name || copy.noFileSelected}>
                    {authFile?.name || copy.noFileSelected}
                  </span>
                </div>
                <input
                  ref={fileInputRef}
                  id="codex-account-auth-file"
                  aria-label={copy.authFile}
                  className="codex-accounts-file-input"
                  type="file"
                  accept=".json,application/json"
                  onChange={(event) => setAuthFile(event.target.files?.[0] ?? null)}
                />
              </div>

              <label className="codex-accounts-checkbox">
                <input
                  type="checkbox"
                  checked={overwrite}
                  onChange={(event) => setOverwrite(event.target.checked)}
                />
                <span>{copy.overwrite}</span>
              </label>

              <div className="codex-accounts-actions">
                <button type="submit" className="route-primary-button">{copy.importAccount}</button>
                <button type="button" onClick={() => void onStartLogin()}>{copy.startLogin}</button>
              </div>
            </form>
          </section>

          <section className="codex-accounts-panel route-surface">
            <div className="codex-accounts-section-head">
              <div>
                <h4>{copy.loginSessionTitle}</h4>
                <p>{copy.loginStatus}</p>
              </div>
            </div>

            {loginSession ? (
              <div className="codex-accounts-session">
                <div className="codex-accounts-session-grid">
                  <div className="codex-accounts-session-item">
                    <span>{copy.loginStatus}</span>
                    <strong>{normalizeText(loginSession.status) || "-"}</strong>
                  </div>
                  <div className="codex-accounts-session-item">
                    <span>{copy.sessionAccount}</span>
                    <strong>{normalizeText(loginSession.account_name) || "-"}</strong>
                  </div>
                  <div className="codex-accounts-session-item">
                    <span>{copy.sessionID}</span>
                    <strong>{normalizeText(loginSession.id) || "-"}</strong>
                  </div>
                </div>
                {normalizeText(loginSession.error) ? <p className="route-error">{normalizeText(loginSession.error)}</p> : null}
                {normalizeText(loginSession.logs) ? (
                  <div className="codex-accounts-logs">
                    <strong>{copy.loginLogs}</strong>
                    <pre>{loginSession.logs}</pre>
                  </div>
                ) : null}
              </div>
            ) : (
              <p className="route-empty">{copy.loginSessionEmpty}</p>
            )}
          </section>
        </aside>
      </div>
    </section>
  );
}

function CodexAccountsLoadingView({ copy }: { copy: CodexAccountsCopy }) {
  return (
    <section className="codex-accounts-view codex-accounts-view-loading" aria-busy="true">
      <div className="codex-accounts-overview route-surface">
        <div className="codex-accounts-section-head">
          <div>
            <h4>{copy.overview}</h4>
            <p>{copy.loading}</p>
          </div>
        </div>
        <div className="codex-accounts-summary-grid">
          {Array.from({ length: 3 }).map((_, index) => (
            <article key={`summary-${index}`} className="codex-accounts-summary-card codex-accounts-skeleton-card">
              <span className="task-skeleton-line codex-accounts-skeleton-label" aria-hidden="true" />
              <strong className="task-skeleton-line codex-accounts-skeleton-value" aria-hidden="true" />
            </article>
          ))}
        </div>
      </div>

      <div className="codex-accounts-workspace">
        <section className="codex-accounts-main route-surface codex-accounts-skeleton-card">
          <div className="codex-accounts-section-head">
            <div>
              <h4>{copy.managedAccounts}</h4>
              <p>{copy.loading}</p>
            </div>
          </div>
          <div className="route-empty-panel codex-accounts-empty codex-accounts-loading-panel">
            <div className="codex-accounts-skeleton-stack" aria-hidden="true">
              <span className="task-skeleton-line codex-accounts-skeleton-title" />
              <span className="task-skeleton-line codex-accounts-skeleton-block" />
              <span className="task-skeleton-line codex-accounts-skeleton-meta" />
            </div>
          </div>
        </section>

        <aside className="codex-accounts-side">
          <section className="codex-accounts-panel route-surface codex-accounts-skeleton-card">
            <div className="codex-accounts-section-head">
              <div>
                <h4>{copy.operationsTitle}</h4>
                <p>{copy.loading}</p>
              </div>
            </div>
            <div className="codex-accounts-skeleton-stack" aria-hidden="true">
              <span className="task-skeleton-line codex-accounts-skeleton-title" />
              <span className="task-skeleton-line codex-accounts-skeleton-field" />
              <span className="task-skeleton-line codex-accounts-skeleton-field" />
              <span className="task-skeleton-line codex-accounts-skeleton-field" />
              <span className="task-skeleton-line codex-accounts-skeleton-button" />
            </div>
          </section>

          <section className="codex-accounts-panel route-surface codex-accounts-skeleton-card">
            <div className="codex-accounts-section-head">
              <div>
                <h4>{copy.loginSessionTitle}</h4>
                <p>{copy.loading}</p>
              </div>
            </div>
            <div className="codex-accounts-skeleton-stack" aria-hidden="true">
              <span className="task-skeleton-line codex-accounts-skeleton-field" />
              <span className="task-skeleton-line codex-accounts-skeleton-meta" />
            </div>
          </section>
        </aside>
      </div>
    </section>
  );
}

function renderPercent(value: number | null) {
  return value == null ? "-" : `${value}%`;
}

function normalizePercent(value: unknown) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) {
    return null;
  }
  return Math.max(0, Math.min(100, Math.round(numeric)));
}

function normalizeText(value: unknown) {
  return String(value || "").trim();
}

async function readFileAsText(file: File) {
  if (typeof file.text === "function") {
    return file.text();
  }
  if (typeof file.arrayBuffer === "function") {
    const buffer = await file.arrayBuffer();
    return new TextDecoder().decode(buffer);
  }
  return String(file);
}
