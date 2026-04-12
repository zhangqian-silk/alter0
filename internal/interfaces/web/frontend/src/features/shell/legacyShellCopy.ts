import { useEffect, useState } from "react";

export type LegacyShellLanguage = "en" | "zh";

type LegacyShellCopy = {
  headings: Record<string, string>;
  routes: Record<string, string>;
  routeTitles: Record<string, string>;
  routeSubtitles: Record<string, string>;
  chatMenu: string;
  chatSessions: string;
  terminalSessions: string;
  terminalNewShort: string;
  sessionHeader: string;
  sessionClose: string;
  sessionNewChat: string;
  sessionNewAgent: string;
  sessionRecent: string;
  sessionHistoryCollapse: string;
  sessionHistoryExpand: string;
  sessionListAriaLabel: string;
  sessionAgentListAriaLabel: string;
  sessionEmpty: string;
  sessionEmptyAgent: string;
  localeButton: string;
  localeShort: string;
  localeAriaLabel: string;
  navCollapseLabel: string;
  navExpandLabel: string;
};

const LEGACY_SHELL_COPY: Record<LegacyShellLanguage, LegacyShellCopy> = {
  en: {
    headings: {
      Workspace: "Workspace",
      "Agent Studio": "Agent Studio",
      Control: "Control",
      Settings: "Settings",
    },
    routes: {
      chat: "Chat",
      "agent-runtime": "Agent",
      terminal: "Terminal",
      agent: "Profiles",
      products: "Products",
      memory: "Memory",
      skills: "Skills",
      mcp: "MCP",
      sessions: "Sessions",
      tasks: "Tasks",
      "cron-jobs": "Cron Jobs",
      channels: "Channels",
      models: "Models",
      environments: "Environments",
    },
    routeTitles: {
      chat: "Chat",
      coding: "Coding",
      writing: "Writing",
      "agent-runtime": "Agent",
      agent: "Agent 配置",
      products: "Products",
      channels: "Channels",
      sessions: "Sessions",
      tasks: "Tasks",
      terminal: "Terminal",
      "cron-jobs": "Cron Jobs",
      memory: "Memory",
      skills: "Skills",
      mcp: "MCP",
      models: "Models",
      environments: "Environments",
    },
    routeSubtitles: {
      chat: "Alter0 workspace for general-purpose conversations and orchestration",
      coding: "Coding Agent workspace for repository analysis, implementation, and verification",
      writing: "Writing Agent workspace for documentation, copy, and structured drafting",
      "agent-runtime": "Run conversations through a selected Agent with independent session history",
      agent: "Create and configure the Agent Profiles available to Agent conversations. Service-managed ID and version are generated automatically.",
      products: "Manage product workspaces, master agents, and reusable product context",
      channels: "Manage connection channels",
      sessions: "View archived sessions with source filters",
      tasks: "Observe runtime tasks with source, status, and timeline filters",
      terminal: "Persistent Codex CLI sessions with runtime-aligned status",
      "cron-jobs": "Configure schedules and trace fired sessions",
      memory: "Summary-first memory view with task history drill-down",
      skills: "Skills configuration",
      mcp: "Model Context Protocol configuration",
      models: "Model capabilities",
      environments: "Environment and deployment settings",
    },
    chatMenu: "Menu",
    chatSessions: "Sessions",
    terminalSessions: "Sessions",
    terminalNewShort: "New",
    sessionHeader: "Work with alter0",
    sessionClose: "Close",
    sessionNewChat: "New Chat",
    sessionNewAgent: "New Agent Run",
    sessionRecent: "Recent Sessions",
    sessionHistoryCollapse: "Collapse",
    sessionHistoryExpand: "Expand",
    sessionListAriaLabel: "Conversation sessions",
    sessionAgentListAriaLabel: "Agent conversation sessions",
    sessionEmpty: "No sessions yet. Click New Chat to start.",
    sessionEmptyAgent: "No Agent sessions yet. Open Agent to start.",
    localeButton: "English",
    localeShort: "EN",
    localeAriaLabel: "Language",
    navCollapseLabel: "Collapse navigation",
    navExpandLabel: "Expand navigation",
  },
  zh: {
    headings: {
      Workspace: "工作区",
      "Agent Studio": "Agent Studio",
      Control: "控制台",
      Settings: "设置",
    },
    routes: {
      chat: "对话",
      "agent-runtime": "Agent",
      terminal: "终端代理",
      agent: "配置",
      products: "产品",
      memory: "记忆",
      skills: "技能",
      mcp: "MCP 协议",
      sessions: "会话列表",
      tasks: "任务观测",
      "cron-jobs": "定时任务",
      channels: "通道",
      models: "模型",
      environments: "环境",
    },
    routeTitles: {
      chat: "对话",
      coding: "Coding",
      writing: "Writing",
      "agent-runtime": "Agent",
      agent: "Agent Profiles",
      products: "Products",
      channels: "通道",
      sessions: "会话列表",
      tasks: "任务观测",
      terminal: "终端",
      "cron-jobs": "定时任务",
      memory: "记忆",
      skills: "技能",
      mcp: "MCP 协议",
      models: "模型",
      environments: "环境",
    },
    routeSubtitles: {
      chat: "默认 Alter0 对话工作区，适合通用任务与子 Agent 编排",
      coding: "Coding Agent 工作区，面向仓库分析、实现与验证",
      writing: "Writing Agent 工作区，面向文档、文案与结构化写作",
      "agent-runtime": "通过选定 Agent 执行会话，并维护独立的会话历史",
      agent: "维护可在 Agent 会话中使用的 Agent Profile，ID 与版本由服务自动生成和管理。",
      products: "管理 Product Workspace、主 Agent 与可复用产品上下文",
      channels: "管理连接通道",
      sessions: "查看归档会话并按来源筛选",
      tasks: "基于来源、状态和时间范围观测运行任务",
      terminal: "独立终端会话，状态与实际 shell 进程保持一致",
      "cron-jobs": "配置调度并追踪触发会话",
      memory: "任务摘要优先展示，支持按需下钻日志与产物",
      skills: "技能配置",
      mcp: "Model Context Protocol 配置",
      models: "模型能力",
      environments: "环境与部署设置",
    },
    chatMenu: "菜单",
    chatSessions: "会话",
    terminalSessions: "会话列表",
    terminalNewShort: "新建",
    sessionHeader: "与 alter0 协作",
    sessionClose: "关闭",
    sessionNewChat: "新对话",
    sessionNewAgent: "新 Agent 会话",
    sessionRecent: "最近会话",
    sessionHistoryCollapse: "折叠",
    sessionHistoryExpand: "展开",
    sessionListAriaLabel: "对话会话列表",
    sessionAgentListAriaLabel: "Agent 会话列表",
    sessionEmpty: "暂无会话，点击“新对话”开始。",
    sessionEmptyAgent: "当前还没有 Agent 会话。请前往 Agent 页面开始。",
    localeButton: "中文",
    localeShort: "中",
    localeAriaLabel: "语言",
    navCollapseLabel: "收起导航",
    navExpandLabel: "展开导航",
  },
};

export function normalizeLegacyShellLanguage(value?: string | null): LegacyShellLanguage {
  return value?.toLowerCase().startsWith("zh") ? "zh" : "en";
}

export function useLegacyShellLanguage(): LegacyShellLanguage {
  const [language, setLanguage] = useState<LegacyShellLanguage>(() =>
    normalizeLegacyShellLanguage(document.documentElement.lang),
  );

  useEffect(() => {
    const root = document.documentElement;
    const observer = new MutationObserver(() => {
      setLanguage(normalizeLegacyShellLanguage(root.lang));
    });

    observer.observe(root, {
      attributes: true,
      attributeFilter: ["lang"],
    });

    return () => observer.disconnect();
  }, []);

  return language;
}

export function getLegacyShellCopy(language: LegacyShellLanguage): LegacyShellCopy {
  return LEGACY_SHELL_COPY[language];
}

export function getLegacyRouteHeadingCopy(language: LegacyShellLanguage, route: string): {
  title: string;
  subtitle: string;
} {
  const copy = getLegacyShellCopy(language);
  return {
    title: copy.routeTitles[route] ?? "Page",
    subtitle: copy.routeSubtitles[route] ?? "Page content",
  };
}

export function getLegacySessionHistoryToggleLabel(language: LegacyShellLanguage, collapsed: boolean): string {
  const copy = getLegacyShellCopy(language);
  return collapsed ? copy.sessionHistoryExpand : copy.sessionHistoryCollapse;
}
