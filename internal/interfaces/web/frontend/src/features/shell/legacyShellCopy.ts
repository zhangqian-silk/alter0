import { useEffect, useState } from "react";

export type LegacyShellLanguage = "en" | "zh";

type LegacyShellCopy = {
  headings: Record<string, string>;
  routes: Record<string, string>;
  routeTitles: Record<string, string>;
  routeSubtitles: Record<string, string>;
  primaryNavLabel: string;
  chatMenu: string;
  chatSessions: string;
  terminalSessions: string;
  terminalNewShort: string;
  workspaceEyebrow: string;
  routeEyebrow: string;
  workspaceModeLabel: string;
  workspaceModeConversation: string;
  workspaceModePage: string;
  workspaceModeTerminal: string;
  workspaceFocusLabel: string;
  workspaceBridgeLabel: string;
  workspaceBridgeValue: string;
  sessionHeader: string;
  sessionPaneLabel: string;
  sessionPanelEyebrow: string;
  sessionPanelBridgeValue: string;
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
  promptDeckEyebrow: string;
  promptDeckTitle: string;
  promptDeckDescription: string;
  composerEyebrow: string;
  composerTitle: string;
  composerDescription: string;
  composerNote: string;
  runtimeTarget: string;
  runtimeAgent: string;
  runtimeAgentPick: string;
  runtimeProvider: string;
  runtimeModel: string;
  runtimeModelShort: string;
  runtimeRawModel: string;
  runtimeServiceDefault: string;
  runtimeEmpty: string;
  runtimeHint: string;
  runtimeToolsMcp: string;
  runtimeToolsShort: string;
  runtimeSkills: string;
  runtimeSkillsShort: string;
  runtimeTargetHint: string;
  runtimeAgentHint: string;
  runtimeModelHint: string;
  runtimeToolsHint: string;
  runtimeSkillsHint: string;
  runtimeMobile: string;
  runtimeMobileHint: string;
  runtimeActive: string;
  runtimeAvailable: string;
  runtimeCategoryTools: string;
  runtimeCategoryMcps: string;
  runtimeLocked: string;
  runtimeNone: string;
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
      "codex-accounts": "Codex Accounts",
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
      "codex-accounts": "Codex Accounts",
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
      "codex-accounts": "Manage saved Codex auth snapshots, inspect quota status, and switch the active runtime account",
    },
    primaryNavLabel: "Primary workspace navigation",
    chatMenu: "Menu",
    chatSessions: "Sessions",
    terminalSessions: "Sessions",
    terminalNewShort: "New",
    workspaceEyebrow: "Workspace cockpit",
    routeEyebrow: "Control surface",
    workspaceModeLabel: "Mode",
    workspaceModeConversation: "Conversation stream",
    workspaceModePage: "Operational page",
    workspaceModeTerminal: "Terminal runtime",
    workspaceFocusLabel: "Focus",
    workspaceBridgeLabel: "Runtime",
    workspaceBridgeValue: "Legacy bridge active",
    sessionHeader: "Work with Alter0",
    sessionPaneLabel: "Session control center",
    sessionPanelEyebrow: "Session control",
    sessionPanelBridgeValue: "Live history",
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
    promptDeckEyebrow: "Quick starts",
    promptDeckTitle: "Launch a strong first turn",
    promptDeckDescription: "Pick a prompt, refine the goal, or hand the runtime a clear operating angle.",
    composerEyebrow: "Compose",
    composerTitle: "Drive the next move",
    composerDescription: "Use the main composer for decisive instructions, follow-up context, or runtime tuning.",
    composerNote: "Bridge-powered input, source-owned shell.",
    runtimeTarget: "Conversation Target",
    runtimeAgent: "Agent",
    runtimeAgentPick: "Choose Agent",
    runtimeProvider: "Provider",
    runtimeModel: "Model",
    runtimeModelShort: "Model",
    runtimeRawModel: "Raw Model",
    runtimeServiceDefault: "Service Default",
    runtimeEmpty: "No enabled model provider is available yet. Configure one in Models to enable session-level model switching.",
    runtimeHint: "Applies to upcoming messages in the current chat session.",
    runtimeToolsMcp: "Tools / MCP",
    runtimeToolsShort: "Tools",
    runtimeSkills: "Skills",
    runtimeSkillsShort: "Skills",
    runtimeTargetHint: "Choose the execution target before the first message.",
    runtimeAgentHint: "Choose the Agent for this session before the first message.",
    runtimeModelHint: "Switches apply to upcoming messages in this session.",
    runtimeToolsHint: "Select extra Tools and MCP integrations for upcoming messages.",
    runtimeSkillsHint: "Select extra Skills for upcoming messages.",
    runtimeMobile: "Session",
    runtimeMobileHint: "Choose model, tools, and skills for upcoming messages.",
    runtimeActive: "Active",
    runtimeAvailable: "Available",
    runtimeCategoryTools: "Tools",
    runtimeCategoryMcps: "MCP",
    runtimeLocked: "Conversation target is locked after the first message.",
    runtimeNone: "No items in this section.",
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
      "codex-accounts": "Codex 账号",
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
      "codex-accounts": "Codex 账号",
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
      "codex-accounts": "维护已保存的 Codex 认证快照，查看额度状态，并切换当前运行时账号",
    },
    primaryNavLabel: "主工作区导航",
    chatMenu: "菜单",
    chatSessions: "会话",
    terminalSessions: "会话列表",
    terminalNewShort: "新建",
    workspaceEyebrow: "工作区驾驶舱",
    routeEyebrow: "控制台界面",
    workspaceModeLabel: "模式",
    workspaceModeConversation: "会话流",
    workspaceModePage: "运营页面",
    workspaceModeTerminal: "终端运行态",
    workspaceFocusLabel: "焦点",
    workspaceBridgeLabel: "运行时",
    workspaceBridgeValue: "Legacy 桥接激活",
    sessionHeader: "与 Alter0 协作",
    sessionPaneLabel: "会话控制中心",
    sessionPanelEyebrow: "会话控制",
    sessionPanelBridgeValue: "实时历史",
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
    promptDeckEyebrow: "快速起手",
    promptDeckTitle: "直接发起一轮高质量对话",
    promptDeckDescription: "可以选择快捷提示、补充目标，也可以先调整运行时再发送。",
    composerEyebrow: "输入区",
    composerTitle: "推动下一步执行",
    composerDescription: "主输入框用于发送明确指令、补充上下文，或结合运行时参数继续推进。",
    composerNote: "桥接负责执行，壳层负责表达。",
    runtimeTarget: "会话目标",
    runtimeAgent: "Agent",
    runtimeAgentPick: "选择 Agent",
    runtimeProvider: "提供方",
    runtimeModel: "模型",
    runtimeModelShort: "模型",
    runtimeRawModel: "Raw Model",
    runtimeServiceDefault: "服务默认",
    runtimeEmpty: "当前还没有可用的启用模型 Provider。请先在 Models 页面完成配置。",
    runtimeHint: "会作用于当前会话后续发送的消息。",
    runtimeToolsMcp: "工具 / MCP",
    runtimeToolsShort: "工具",
    runtimeSkills: "技能",
    runtimeSkillsShort: "技能",
    runtimeTargetHint: "请在发送第一条消息前确定当前会话目标。",
    runtimeAgentHint: "请在发送第一条消息前为当前会话选择 Agent。",
    runtimeModelHint: "切换后会作用于当前会话后续发送的消息。",
    runtimeToolsHint: "为后续消息选择额外启用的工具与 MCP。",
    runtimeSkillsHint: "为后续消息选择额外启用的技能。",
    runtimeMobile: "会话设置",
    runtimeMobileHint: "为后续消息集中选择模型、工具与技能。",
    runtimeActive: "已启用",
    runtimeAvailable: "可启用",
    runtimeCategoryTools: "工具",
    runtimeCategoryMcps: "MCP 服务",
    runtimeLocked: "发送第一条消息后，会话目标不可切换。",
    runtimeNone: "该分区暂无项目。",
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

export function getLegacySessionTrackedCountLabel(language: LegacyShellLanguage, count: number): string {
  if (language === "zh") {
    return `已跟踪 ${count} 条`;
  }
  return `${count} tracked`;
}
