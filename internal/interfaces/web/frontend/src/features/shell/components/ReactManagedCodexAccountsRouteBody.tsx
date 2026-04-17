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
  title: string;
  subtitle: string;
  accountName: string;
  authFile: string;
  importAccount: string;
  startLogin: string;
  current: string;
  activePath: string;
  quotaHourly: string;
  quotaWeekly: string;
  plan: string;
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
    title: "Codex Accounts",
    subtitle: "Manage saved Codex auth snapshots, inspect runtime quota status, and switch the active runtime account.",
    accountName: "Account Name",
    authFile: "Auth File",
    importAccount: "Import auth.json",
    startLogin: "Start Codex Login",
    current: "Current",
    activePath: "Active auth path",
    quotaHourly: "Hourly Remaining",
    quotaWeekly: "Weekly Remaining",
    plan: "Plan",
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
    title: "Codex 账号",
    subtitle: "维护已保存的 Codex 认证快照，查看运行时额度状态，并切换当前生效账号。",
    accountName: "账号名称",
    authFile: "Auth 文件",
    importAccount: "导入 auth.json",
    startLogin: "启动 Codex 登录",
    current: "当前",
    activePath: "当前生效 auth 路径",
    quotaHourly: "小时剩余额度",
    quotaWeekly: "周剩余额度",
    plan: "套餐",
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
      const fileInput = document.getElementById("codex-account-auth-file") as HTMLInputElement | null;
      if (fileInput) {
        fileInput.value = "";
      }
      await reloadAccounts(copy.imported, "success");
    } catch (error: unknown) {
      setStatusKind("error");
      setStatusMessage(copy.actionFailed(error instanceof Error ? error.message : "unknown_error"));
    }
  }

  async function onSwitchAccount(accountName: string) {
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
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (requestState.status === "error") {
    return <p className="route-error">{copy.loadFailed(requestState.error)}</p>;
  }

  return (
    <section className="agent-studio-view">
      <div className="agent-route-header">
        <div>
          <h3>{copy.title}</h3>
          <p className="agent-route-pane-copy">{copy.subtitle}</p>
        </div>
      </div>

      {statusMessage ? (
        <p className={statusKind === "error" ? "route-error" : "route-status"}>{statusMessage}</p>
      ) : null}

      <div className="agent-route-layout">
        <div className="agent-route-list">
          {active?.auth_path ? (
            <article className="agent-route-card">
              <div className="agent-route-card-head">
                <div className="agent-route-card-copy">
                  <strong>{copy.activePath}</strong>
                  <span title={active.auth_path}>{active.auth_path}</span>
                </div>
                {active.managed?.name ? <span className="agent-route-state is-enabled">{copy.current}</span> : null}
              </div>
            </article>
          ) : null}

          {accounts.length === 0 ? (
            <p className="route-empty">{copy.empty}</p>
          ) : (
            accounts.map((item) => {
              const account = item.record ?? {};
              const snapshot = account.snapshot ?? {};
              const title = normalizeText(snapshot.account_name) || normalizeText(account.name) || "-";
              const plan = normalizeText(item.quota?.plan) || normalizeText(snapshot.plan);
              const hourly = item.quota?.hourly?.remaining_percent;
              const weekly = item.quota?.weekly?.remaining_percent;
              return (
                <article key={normalizeText(account.name) || title} className="agent-route-card">
                  <div className="agent-route-card-head">
                    <div className="agent-route-card-copy">
                      <strong>{title}</strong>
                      <span title={normalizeText(account.name)}>{normalizeText(account.name)}</span>
                    </div>
                    {item.current ? (
                      <span className="agent-route-state is-enabled">{copy.current}</span>
                    ) : (
                      <button
                        type="button"
                        className="agent-route-action"
                        onClick={() => void onSwitchAccount(normalizeText(account.name))}
                      >
                        {copy.switchTo(title)}
                      </button>
                    )}
                  </div>
                  <div className="agent-route-card-tags">
                    {normalizeText(snapshot.email) ? <span>{normalizeText(snapshot.email)}</span> : null}
                    {plan ? <span>{copy.plan}: {plan}</span> : null}
                    {typeof hourly === "number" ? <span>{copy.quotaHourly}: {hourly}%</span> : null}
                    {typeof weekly === "number" ? <span>{copy.quotaWeekly}: {weekly}%</span> : null}
                  </div>
                  {normalizeText(item.error) ? <p className="route-error">{item.error}</p> : null}
                </article>
              );
            })
          )}
        </div>

        <form
          className="agent-builder-form"
          onSubmit={(event) => {
            event.preventDefault();
            void onImportAuthFile();
          }}
        >
          <label>
            <span>{copy.accountName}</span>
            <input value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label>
            <span>{copy.authFile}</span>
            <input
              id="codex-account-auth-file"
              aria-label={copy.authFile}
              type="file"
              accept=".json,application/json"
              onChange={(event) => setAuthFile(event.target.files?.[0] ?? null)}
            />
          </label>
          <label className="agent-builder-option">
            <input
              type="checkbox"
              checked={overwrite}
              onChange={(event) => setOverwrite(event.target.checked)}
            />
            <span>Overwrite existing account</span>
          </label>
          <div className="agent-builder-actions">
            <button type="submit">{copy.importAccount}</button>
            <button type="button" onClick={() => void onStartLogin()}>
              {copy.startLogin}
            </button>
          </div>

          {loginSession ? (
            <div className="agent-builder-managed-item">
              <strong>{copy.loginStatus}</strong>
              <span>{normalizeText(loginSession.status) || "-"}</span>
              {normalizeText(loginSession.error) ? <p className="route-error">{loginSession.error}</p> : null}
              {normalizeText(loginSession.logs) ? (
                <>
                  <strong>{copy.loginLogs}</strong>
                  <pre>{loginSession.logs}</pre>
                </>
              ) : null}
            </div>
          ) : null}
        </form>
      </div>
    </section>
  );
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
