import { useContext, useEffect, useState } from "react";
import { WorkbenchContext } from "../../../app/WorkbenchContext";
import { createAPIClient } from "../../../shared/api/client";
import type { LegacyShellLanguage } from "../legacyShellCopy";

const AGENT_ROUTE_STORAGE_KEY = "alter0.web.agent-route.v1";

type AgentRecord = {
  id?: string;
  name?: string;
  enabled?: boolean;
  scope?: string;
  version?: string;
  system_prompt?: string;
  max_iterations?: number;
  tools?: string[];
  skills?: string[];
  mcps?: string[];
  memory_files?: string[];
};

type CapabilityRecord = {
  id?: string;
  name?: string;
  scope?: string;
  version?: string;
  description?: string;
};

type AgentResponse = {
  items?: AgentRecord[];
};

type CapabilityResponse = {
  items?: CapabilityRecord[];
};

type MemoryFileOption = {
  id: string;
  name: string;
  description: string;
};

type AgentDraft = {
  id: string;
  name: string;
  enabled: boolean;
  scope: string;
  version: string;
  systemPrompt: string;
  maxIterations: string;
  tools: string;
  skills: string[];
  mcps: string[];
  memoryFiles: string[];
};

type RequestState =
  | { status: "loading"; error: string }
  | { status: "ready"; error: string }
  | { status: "error"; error: string };

type AgentRouteCopy = {
  loading: string;
  empty: string;
  title: string;
  subtitle: string;
  edit: string;
  newAgent: string;
  managed: string;
  name: string;
  maxIterations: string;
  enabled: string;
  systemPrompt: string;
  tools: string;
  skills: string;
  mcps: string;
  memoryFiles: string;
  create: string;
  reset: string;
  save: string;
  delete: string;
  openAgent: string;
  pending: string;
  id: string;
  version: string;
  scope: string;
  saved: string;
  deleted: string;
  loadFailed: (message: string) => string;
  saveFailed: (message: string) => string;
  deleteFailed: (message: string) => string;
};

const AGENT_ROUTE_COPY: Record<LegacyShellLanguage, AgentRouteCopy> = {
  en: {
    loading: "Loading...",
    empty: "No managed agents yet.",
    title: "Agent Profiles",
    subtitle: "Create and maintain the user-managed agents available to runtime sessions.",
    edit: "Edit Agent",
    newAgent: "Unsaved Agent",
    managed: "Managed identity and version are assigned by the service.",
    name: "Name",
    maxIterations: "Max Iterations",
    enabled: "Enabled",
    systemPrompt: "System Prompt",
    tools: "Tools",
    skills: "Skills",
    mcps: "MCP",
    memoryFiles: "Memory Files",
    create: "Create Agent",
    reset: "Reset",
    save: "Save Agent",
    delete: "Delete Agent",
    openAgent: "Open Agent",
    pending: "Pending",
    id: "ID",
    version: "Version",
    scope: "Scope",
    saved: "Agent saved.",
    deleted: "Agent deleted.",
    loadFailed: (message) => `Load failed: ${message}`,
    saveFailed: (message) => `Save Agent failed: ${message}`,
    deleteFailed: (message) => `Delete Agent failed: ${message}`,
  },
  zh: {
    loading: "加载中...",
    empty: "暂无托管 Agent。",
    title: "Agent Profiles",
    subtitle: "创建并维护可供运行时会话使用的用户托管 Agent。",
    edit: "编辑 Agent",
    newAgent: "未保存 Agent",
    managed: "服务端负责分配和维护 ID 与版本。",
    name: "名称",
    maxIterations: "最大迭代数",
    enabled: "启用",
    systemPrompt: "系统提示词",
    tools: "工具",
    skills: "技能",
    mcps: "MCP",
    memoryFiles: "记忆文件",
    create: "新建 Agent",
    reset: "重置",
    save: "保存 Agent",
    delete: "删除 Agent",
    openAgent: "打开 Agent",
    pending: "待生成",
    id: "ID",
    version: "版本",
    scope: "范围",
    saved: "Agent 已保存。",
    deleted: "Agent 已删除。",
    loadFailed: (message) => `加载失败：${message}`,
    saveFailed: (message) => `保存 Agent 失败：${message}`,
    deleteFailed: (message) => `删除 Agent 失败：${message}`,
  },
};

const AGENT_MEMORY_FILE_OPTIONS: Record<LegacyShellLanguage, MemoryFileOption[]> = {
  en: [
    {
      id: "user_md",
      name: "USER.md",
      description: "User profile, collaboration preferences, and stable output conventions.",
    },
    {
      id: "soul_md",
      name: "SOUL.md",
      description: "Mandatory long-term rules and hard constraints with highest priority.",
    },
    {
      id: "agents_md",
      name: "AGENTS.md",
      description: "Private operating rules for the current agent.",
    },
    {
      id: "memory_long_term",
      name: "MEMORY.md / memory.md",
      description: "Long-term durable memory.",
    },
    {
      id: "memory_daily_today",
      name: "Daily Memory (Today)",
      description: "Today's daily memory log.",
    },
    {
      id: "memory_daily_yesterday",
      name: "Daily Memory (Yesterday)",
      description: "Yesterday's daily memory log.",
    },
  ],
  zh: [
    {
      id: "user_md",
      name: "USER.md",
      description: "用户画像、协作偏好与稳定输出约定。",
    },
    {
      id: "soul_md",
      name: "SOUL.md",
      description: "最高优先级的长期规则与硬约束。",
    },
    {
      id: "agents_md",
      name: "AGENTS.md",
      description: "当前 Agent 的私有协作规则。",
    },
    {
      id: "memory_long_term",
      name: "MEMORY.md / memory.md",
      description: "长期持久记忆。",
    },
    {
      id: "memory_daily_today",
      name: "当日日记忆",
      description: "当天 Daily Memory。",
    },
    {
      id: "memory_daily_yesterday",
      name: "前一日日记忆",
      description: "前一天 Daily Memory。",
    },
  ],
};

export function ReactManagedAgentRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const workbench = useContext(WorkbenchContext);
  const copy = AGENT_ROUTE_COPY[language];
  const apiClient = createAPIClient();
  const memoryFileOptions = AGENT_MEMORY_FILE_OPTIONS[language];
  const [requestState, setRequestState] = useState<RequestState>({ status: "loading", error: "" });
  const [agents, setAgents] = useState<AgentRecord[]>([]);
  const [skills, setSkills] = useState<CapabilityRecord[]>([]);
  const [mcps, setMCPs] = useState<CapabilityRecord[]>([]);
  const [selectedAgentID, setSelectedAgentID] = useState(() => loadSelectedAgentID());
  const [draft, setDraft] = useState<AgentDraft>(() => normalizeAgentDraft());
  const [statusMessage, setStatusMessage] = useState("");
  const [statusKind, setStatusKind] = useState<"success" | "error" | "">("");

  useEffect(() => {
    void reloadAgentRoute();
  }, []);

  useEffect(() => {
    persistSelectedAgentID(selectedAgentID);
  }, [selectedAgentID]);

  const selectedAgent = agents.find((item) => normalizeText(item.id) === selectedAgentID) ?? null;

  async function reloadAgentRoute(preferredAgentID: string = selectedAgentID, nextStatus = "", nextStatusKind: "success" | "error" | "" = "") {
    setRequestState({ status: "loading", error: "" });
    try {
      const [agentPayload, skillPayload, mcpPayload] = await Promise.all([
        apiClient.get<AgentResponse>("/api/control/agents"),
        apiClient.get<CapabilityResponse>("/api/control/skills"),
        apiClient.get<CapabilityResponse>("/api/control/mcps"),
      ]);

      const nextAgents = Array.isArray(agentPayload?.items) ? agentPayload.items : [];
      const nextSkills = Array.isArray(skillPayload?.items) ? skillPayload.items : [];
      const nextMCPs = Array.isArray(mcpPayload?.items) ? mcpPayload.items : [];
      const nextSelectedAgent =
        nextAgents.find((item) => normalizeText(item.id) === normalizeText(preferredAgentID)) ??
        nextAgents[0] ??
        null;

      setAgents(nextAgents);
      setSkills(nextSkills);
      setMCPs(nextMCPs);
      setSelectedAgentID(normalizeText(nextSelectedAgent?.id));
      setDraft(normalizeAgentDraft(nextSelectedAgent ?? undefined));
      setStatusMessage(nextStatus);
      setStatusKind(nextStatusKind);
      setRequestState({ status: "ready", error: "" });
    } catch (error: unknown) {
      setRequestState({
        status: "error",
        error: error instanceof Error ? error.message : "unknown_error",
      });
    }
  }

  async function handleSave() {
    const payload = {
      name: draft.name.trim(),
      enabled: draft.enabled,
      system_prompt: draft.systemPrompt.trim(),
      max_iterations: Number.parseInt(draft.maxIterations || "0", 10) || 0,
      tools: parseCSVList(draft.tools),
      skills: draft.skills,
      mcps: draft.mcps,
      memory_files: draft.memoryFiles,
    };

    try {
      const saved = selectedAgentID
        ? await apiClient.put<AgentRecord>(`/api/control/agents/${encodeURIComponent(selectedAgentID)}`, payload)
        : await apiClient.post<AgentRecord>("/api/control/agents", payload);

      await reloadAgentRoute(normalizeText(saved?.id), copy.saved, "success");
    } catch (error: unknown) {
      setStatusMessage(copy.saveFailed(error instanceof Error ? error.message : "unknown_error"));
      setStatusKind("error");
    }
  }

  async function handleDelete() {
    if (!selectedAgentID) {
      return;
    }
    try {
      await apiClient.delete(`/api/control/agents/${encodeURIComponent(selectedAgentID)}`);
      await reloadAgentRoute("", copy.deleted, "success");
    } catch (error: unknown) {
      setStatusMessage(copy.deleteFailed(error instanceof Error ? error.message : "unknown_error"));
      setStatusKind("error");
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
      <aside className="route-surface agent-studio-list-pane">
        <div className="agent-route-pane-head">
          <div className="agent-route-pane-copy">
            <h4>{copy.title}</h4>
          </div>
          <button
            className="route-primary-button"
            type="button"
            onClick={() => {
              setSelectedAgentID("");
              setDraft(normalizeAgentDraft());
              setStatusMessage("");
              setStatusKind("");
            }}
          >
            {copy.create}
          </button>
        </div>
        <div className="agent-route-list">
          {agents.length ? (
            agents.map((agent) => {
              const agentID = normalizeText(agent.id);
              const active = agentID === selectedAgentID;
              return (
                <button
                  key={agentID}
                  className={`agent-route-card${active ? " is-active" : ""}`}
                  type="button"
                  aria-pressed={active ? "true" : "false"}
                  onClick={() => {
                    setSelectedAgentID(agentID);
                    setDraft(normalizeAgentDraft(agent));
                    setStatusMessage("");
                    setStatusKind("");
                  }}
                >
                  <div className="agent-route-card-head">
                    <div className="agent-route-card-copy">
                      <strong title={normalizeText(agent.name || agent.id)}>{normalizeText(agent.name || agent.id)}</strong>
                      <span title={agentID}>{agentID}</span>
                    </div>
                    <span className={`agent-route-state ${agent.enabled !== false ? "is-enabled" : "is-disabled"}`}>
                      {agent.enabled !== false ? "Enabled" : "Disabled"}
                    </span>
                  </div>
                  <p className="agent-route-card-prompt">{normalizeText(agent.system_prompt)}</p>
                  <div className="agent-route-card-tags">
                    {buildAgentTagList(agent).map((tag) => (
                      <span key={`${agentID}-${tag}`}>{tag}</span>
                    ))}
                  </div>
                </button>
              );
            })
          ) : (
            <p className="route-empty">{copy.empty}</p>
          )}
        </div>
      </aside>

      <section className="route-surface agent-studio-form-pane">
        <div className="agent-route-pane-head">
          <div className="agent-route-pane-copy">
            <h4>{selectedAgentID ? copy.edit : copy.newAgent}</h4>
          </div>
          <div className="agent-builder-actions">
            <button
              type="button"
              disabled={!selectedAgentID}
              onClick={() => {
                if (workbench) {
                  workbench.navigate("agent-runtime");
                  return;
                }
                window.location.hash = "#agent-runtime";
              }}
            >
              {copy.openAgent}
            </button>
            <button
              type="button"
              onClick={() => {
                setDraft(normalizeAgentDraft(selectedAgent ?? undefined));
                setStatusMessage("");
                setStatusKind("");
              }}
            >
              {copy.reset}
            </button>
          </div>
        </div>

        {statusMessage ? (
          <p className={`agent-builder-status ${statusKind === "error" ? "is-error" : "is-success"}`}>
            {statusMessage}
          </p>
        ) : null}

        <div className="agent-builder-managed">
          <div className="agent-builder-managed-item">
            <span>{copy.id}</span>
            <strong>{selectedAgent?.id || copy.pending}</strong>
          </div>
          <div className="agent-builder-managed-item">
            <span>{copy.version}</span>
            <strong>{selectedAgent?.version || copy.pending}</strong>
          </div>
          <div className="agent-builder-managed-item">
            <span>{copy.scope}</span>
            <strong>{selectedAgent?.scope || draft.scope || "global"}</strong>
          </div>
        </div>

        <form
          className="agent-builder-form"
          onSubmit={(event) => {
            event.preventDefault();
            void handleSave();
          }}
        >
          <div className="agent-builder-top-grid">
            <label className="agent-builder-field agent-builder-field-name">
              <span>{copy.name}</span>
              <input
                aria-label={copy.name}
                type="text"
                value={draft.name}
                onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
                placeholder="Researcher"
              />
            </label>
            <label className="agent-builder-field agent-builder-field-iterations">
              <span>{copy.maxIterations}</span>
              <input
                aria-label={copy.maxIterations}
                type="number"
                min="0"
                value={draft.maxIterations}
                onChange={(event) => setDraft((current) => ({ ...current, maxIterations: event.target.value }))}
              />
            </label>
            <label className="agent-builder-toggle">
              <span>{copy.enabled}</span>
              <span className="agent-builder-switch">
                <input
                  aria-label={copy.enabled}
                  type="checkbox"
                  checked={draft.enabled}
                  onChange={(event) => setDraft((current) => ({ ...current, enabled: event.target.checked }))}
                />
                <span className="agent-builder-switch-track" aria-hidden="true">
                  <span className="agent-builder-switch-thumb"></span>
                </span>
              </span>
            </label>
          </div>
          <label className="agent-builder-wide">
            <span>{copy.systemPrompt}</span>
            <textarea
              aria-label={copy.systemPrompt}
              rows={10}
              value={draft.systemPrompt}
              onChange={(event) => setDraft((current) => ({ ...current, systemPrompt: event.target.value }))}
            />
          </label>
          <label className="agent-builder-wide">
            <span>{copy.tools}</span>
            <input
              type="text"
              value={draft.tools}
              onChange={(event) => setDraft((current) => ({ ...current, tools: event.target.value }))}
              placeholder="codex_exec, search_memory, read_memory, write_memory"
            />
          </label>

          <OptionSection
            title={copy.skills}
            options={skills.map((item) => ({
              id: normalizeText(item.id),
              name: normalizeText(item.name || item.id),
              description: [normalizeText(item.scope), normalizeText(item.version)]
                .filter((value) => value !== "-")
                .join(" · "),
            }))}
            selected={draft.skills}
            onToggle={(id) => {
              setDraft((current) => ({
                ...current,
                skills: toggleListValue(current.skills, id),
              }));
            }}
          />

          <OptionSection
            title={copy.mcps}
            options={mcps.map((item) => ({
              id: normalizeText(item.id),
              name: normalizeText(item.name || item.id),
              description: [normalizeText(item.scope), normalizeText(item.version)]
                .filter((value) => value !== "-")
                .join(" · "),
            }))}
            selected={draft.mcps}
            onToggle={(id) => {
              setDraft((current) => ({
                ...current,
                mcps: toggleListValue(current.mcps, id),
              }));
            }}
          />

          <OptionSection
            title={copy.memoryFiles}
            options={memoryFileOptions}
            selected={draft.memoryFiles}
            onToggle={(id) => {
              setDraft((current) => ({
                ...current,
                memoryFiles: toggleListValue(current.memoryFiles, id),
              }));
            }}
          />

          <div className="task-filter-actions">
            <button className="task-filter-apply" type="submit">
              {copy.save}
            </button>
            <button
              className="task-filter-reset"
              type="button"
              disabled={!selectedAgentID}
              onClick={() => {
                void handleDelete();
              }}
            >
              {copy.delete}
            </button>
          </div>
        </form>
      </section>
    </section>
  );
}

function OptionSection({
  title,
  options,
  selected,
  onToggle,
}: {
  title: string;
  options: Array<{ id: string; name: string; description: string }>;
  selected: string[];
  onToggle: (id: string) => void;
}) {
  return (
    <div className="agent-builder-wide agent-builder-section">
      <h5>{title}</h5>
      <div className="agent-builder-options">
        {options.map((option) => {
          const checked = selected.includes(option.id);
          const label = option.description ? `${option.name} ${option.description}` : option.name;
          return (
            <label key={option.id} className="agent-builder-option">
              <input
                type="checkbox"
                aria-label={label}
                checked={checked}
                onChange={() => onToggle(option.id)}
              />
              <span className="agent-builder-option-copy">
                <strong>{option.name}</strong>
                {option.description ? <small>{option.description}</small> : null}
              </span>
            </label>
          );
        })}
      </div>
    </div>
  );
}

function normalizeAgentDraft(agent?: AgentRecord): AgentDraft {
  return {
    id: normalizeText(agent?.id),
    name: String(agent?.name || "").trim(),
    enabled: agent?.enabled !== false,
    scope: String(agent?.scope || "global").trim() || "global",
    version: String(agent?.version || "").trim(),
    systemPrompt: String(agent?.system_prompt || "").trim(),
    maxIterations: Number.isFinite(Number(agent?.max_iterations)) ? String(Math.max(0, Number(agent?.max_iterations))) : "0",
    tools: Array.isArray(agent?.tools) && agent.tools.length
      ? agent.tools.map((item) => String(item || "").trim()).filter(Boolean).join(", ")
      : "codex_exec, search_memory, read_memory, write_memory",
    skills: Array.isArray(agent?.skills) ? agent.skills.map((item) => String(item || "").trim()).filter(Boolean) : [],
    mcps: Array.isArray(agent?.mcps) ? agent.mcps.map((item) => String(item || "").trim()).filter(Boolean) : [],
    memoryFiles: Array.isArray(agent?.memory_files)
      ? agent.memory_files.map((item) => String(item || "").trim()).filter(Boolean)
      : [],
  };
}

function parseCSVList(value: string) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function buildAgentTagList(agent: AgentRecord) {
  return [
    normalizeText(agent.scope),
    normalizeText(agent.version),
    ...(Array.isArray(agent.tools) ? agent.tools.slice(0, 2).map((item) => normalizeText(item)) : []),
  ].filter((value, index, items) => value !== "-" && items.indexOf(value) === index);
}

function toggleListValue(list: string[], value: string) {
  return list.includes(value) ? list.filter((item) => item !== value) : [...list, value];
}

function loadSelectedAgentID() {
  try {
    const raw = window.localStorage.getItem(AGENT_ROUTE_STORAGE_KEY);
    if (!raw) {
      return "";
    }
    const payload = JSON.parse(raw);
    return normalizeText(payload?.selected_agent_id);
  } catch {
    return "";
  }
}

function persistSelectedAgentID(selectedAgentID: string) {
  try {
    window.localStorage.setItem(
      AGENT_ROUTE_STORAGE_KEY,
      JSON.stringify({ selected_agent_id: normalizeText(selectedAgentID) }),
    );
  } catch {
  }
}

function normalizeText(value: unknown) {
  return String(value || "").trim();
}
