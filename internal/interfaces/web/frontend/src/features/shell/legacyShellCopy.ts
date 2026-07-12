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
  chatNewShort: string;
  workspaceEyebrow: string;
  routeEyebrow: string;
  workspaceModeLabel: string;
  workspaceModeConversation: string;
  workspaceModePage: string;
  workspaceFocusLabel: string;
  settingsSectionsLabel: string;
  sessionHeader: string;
  sessionPaneLabel: string;
  sessionPanelEyebrow: string;
  sessionClose: string;
  sessionHide: string;
  sessionRecent: string;
  sessionHistoryCollapse: string;
  sessionHistoryExpand: string;
  sessionListAriaLabel: string;
  sessionSkillListAriaLabel: string;
  sessionEmpty: string;
  sessionEmptySkill: string;
  promptDeckEyebrow: string;
  promptDeckTitle: string;
  promptDeckDescription: string;
  composerEyebrow: string;
  composerTitle: string;
  composerDescription: string;
  runtimeTarget: string;
  runtimeSkill: string;
  runtimeSkillPick: string;
  runtimeProvider: string;
  runtimeModel: string;
  runtimeModelShort: string;
  runtimeRawModel: string;
  runtimeServiceDefault: string;
  runtimeEmpty: string;
  runtimeHint: string;
  runtimeSkills: string;
  runtimeSkillsShort: string;
  runtimeTargetHint: string;
  runtimeSkillHint: string;
  runtimeModelHint: string;
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
      "Skill Studio": "Skill Studio",
      Control: "Control",
      Settings: "Settings",
    },
    routes: {
      chat: "Chat",
      settings: "Settings",
      general: "General",
      runtime: "Runtime",
      schedules: "Schedules",
      skill: "Profiles",
      skills: "Skills",
      mcp: "MCP",
      "cron-jobs": "Cron Jobs",
      channels: "Channels",
      models: "Models",
      "codex-accounts": "Codex Runtime",
    },
    routeTitles: {
      chat: "Chat",
      implementation: "Implementation",
      writing: "Writing",
      skill: "Skill 配置",
      channels: "Channels",
      sessions: "Sessions",
      tasks: "Tasks",
      settings: "Settings",
      general: "General",
      runtime: "Runtime",
      schedules: "Schedules",
      "cron-jobs": "Cron Jobs",
      skills: "Skills",
      mcp: "MCP",
      models: "Models",
      "codex-accounts": "Codex Runtime",
    },
    routeSubtitles: {
      chat: "Alter0 workspace for general-purpose conversations and orchestration",
      implementation: "Implementation workspace for repository analysis, implementation, and verification",
      writing: "Writing workspace for documentation, copy, and structured drafting",
      skill: "Create and configure reusable execution profiles for Chat sessions. Service-managed ID and version are generated automatically.",
      channels: "Manage connection channels",
      sessions: "View archived sessions with source filters",
      tasks: "Observe runtime tasks with source, status, and timeline filters",
      "cron-jobs": "Configure schedules and trace fired sessions",
      skills: "Skills configuration",
      mcp: "Model Context Protocol configuration",
      models: "Model capabilities",
      "codex-accounts": "Inspect Codex Direct health, auth state, model, reasoning depth, and runtime diagnostics",
      settings: "Configure runtime, skills, and schedules from one compact surface",
      general: "Preferences that apply across the workbench",
      runtime: "Claude Code, Codex Direct, native Memories, providers, configuration, and execution health",
      schedules: "Maintenance and skill-created schedules that can run silently or create Chat sessions",
    },
    primaryNavLabel: "Primary workspace navigation",
    chatMenu: "Menu",
    chatSessions: "Sessions",
    chatNewShort: "New",
    workspaceEyebrow: "Workspace cockpit",
    routeEyebrow: "Control surface",
    workspaceModeLabel: "Mode",
    workspaceModeConversation: "Conversation stream",
    workspaceModePage: "Operational page",
    workspaceFocusLabel: "Focus",
    settingsSectionsLabel: "Settings sections",
    sessionHeader: "Work with Alter0",
    sessionPaneLabel: "Session control center",
    sessionPanelEyebrow: "Session control",
    sessionClose: "Close",
    sessionHide: "Hide",
    sessionRecent: "Recent Sessions",
    sessionHistoryCollapse: "Collapse",
    sessionHistoryExpand: "Expand",
    sessionListAriaLabel: "Conversation sessions",
    sessionSkillListAriaLabel: "Skill sessions",
    sessionEmpty: "No sessions yet. Click New to start.",
    sessionEmptySkill: "No sessions yet. Click New to start.",
    promptDeckEyebrow: "Quick starts",
    promptDeckTitle: "Launch a strong first turn",
    promptDeckDescription: "Pick a prompt, refine the goal, or hand the runtime a clear operating angle.",
    composerEyebrow: "Compose",
    composerTitle: "Drive the next move",
    composerDescription: "Use the main composer for decisive instructions, follow-up context, or runtime tuning.",
    runtimeTarget: "Conversation Target",
    runtimeSkill: "Skill",
    runtimeSkillPick: "Choose Skill",
    runtimeProvider: "Provider",
    runtimeModel: "Model",
    runtimeModelShort: "Model",
    runtimeRawModel: "Raw Model",
    runtimeServiceDefault: "Service Default",
    runtimeEmpty: "No enabled model provider is available yet. Configure one in Models to enable session-level model switching.",
    runtimeHint: "Applies to upcoming messages in the current chat session.",
    runtimeSkills: "Skills",
    runtimeSkillsShort: "Skills",
    runtimeTargetHint: "Choose the execution target before the first message.",
    runtimeSkillHint: "Choose the Skill for this session before the first message.",
    runtimeModelHint: "Switches apply to upcoming messages in this session.",
    runtimeSkillsHint: "Select extra Skills for upcoming messages.",
    runtimeMobile: "Session",
    runtimeMobileHint: "Choose a model and skills for upcoming messages.",
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
      "Skill Studio": "Skill Studio",
      Control: "控制台",
      Settings: "设置",
    },
    routes: {
      chat: "对话",
      settings: "设置",
      general: "通用",
      runtime: "运行时",
      schedules: "定时任务",
      skill: "配置",
      skills: "技能",
      mcp: "MCP 协议",
      "cron-jobs": "定时任务",
      channels: "通道",
      models: "模型",
      "codex-accounts": "Codex 运行时",
    },
    routeTitles: {
      chat: "对话",
      implementation: "Implementation",
      writing: "Writing",
      skill: "Skill Profiles",
      channels: "通道",
      sessions: "会话列表",
      tasks: "任务观测",
      settings: "设置",
      general: "通用",
      runtime: "运行时",
      schedules: "定时任务",
      "cron-jobs": "定时任务",
      skills: "技能",
      mcp: "MCP 协议",
      models: "模型",
      "codex-accounts": "Codex 运行时",
    },
    routeSubtitles: {
      chat: "默认 Alter0 对话工作区，适合通用任务与 Skill 编排",
      implementation: "Implementation 工作区，面向仓库分析、实现与验证",
      writing: "Writing 工作区，面向文档、文案与结构化写作",
      skill: "维护可在 Chat 会话中复用的执行 Profile，ID 与版本由服务自动生成和管理。",
      channels: "管理连接通道",
      sessions: "查看归档会话并按来源筛选",
      tasks: "基于来源、状态和时间范围观测运行任务",
      "cron-jobs": "配置调度并追踪触发会话",
      skills: "技能配置",
      mcp: "Model Context Protocol 配置",
      models: "模型能力",
      "codex-accounts": "查看 Codex Direct 健康状态、认证状态、model、思考深度与运行时诊断",
      settings: "在同一设置页维护运行时、技能与定时任务",
      general: "维护工作台通用偏好",
      runtime: "Claude Code、Codex Direct、原生 Memories、Provider、配置与执行健康状态",
      schedules: "维护任务与 Skill 创建的定时任务，可静默运行或创建 Chat 会话",
    },
    primaryNavLabel: "主工作区导航",
    chatMenu: "菜单",
    chatSessions: "会话列表",
    chatNewShort: "新建",
    workspaceEyebrow: "工作区驾驶舱",
    routeEyebrow: "控制台界面",
    workspaceModeLabel: "模式",
    workspaceModeConversation: "会话流",
    workspaceModePage: "运营页面",
    workspaceFocusLabel: "焦点",
    settingsSectionsLabel: "设置分区",
    sessionHeader: "与 Alter0 协作",
    sessionPaneLabel: "会话控制中心",
    sessionPanelEyebrow: "会话控制",
    sessionClose: "关闭",
    sessionHide: "收起",
    sessionRecent: "最近会话",
    sessionHistoryCollapse: "折叠",
    sessionHistoryExpand: "展开",
    sessionListAriaLabel: "对话会话列表",
    sessionSkillListAriaLabel: "Skill 会话列表",
    sessionEmpty: "暂无会话，点击“新对话”开始。",
    sessionEmptySkill: "当前还没有 Skill 会话。请前往 Skill 页面开始。",
    promptDeckEyebrow: "快速起手",
    promptDeckTitle: "直接发起一轮高质量对话",
    promptDeckDescription: "可以选择快捷提示、补充目标，也可以先调整运行时再发送。",
    composerEyebrow: "输入区",
    composerTitle: "推动下一步执行",
    composerDescription: "主输入框用于发送明确指令、补充上下文，或结合运行时参数继续推进。",
    runtimeTarget: "会话目标",
    runtimeSkill: "Skill",
    runtimeSkillPick: "选择 Skill",
    runtimeProvider: "提供方",
    runtimeModel: "模型",
    runtimeModelShort: "模型",
    runtimeRawModel: "Raw Model",
    runtimeServiceDefault: "服务默认",
    runtimeEmpty: "当前还没有可用的启用模型 Provider。请先在 Models 页面完成配置。",
    runtimeHint: "会作用于当前会话后续发送的消息。",
    runtimeSkills: "技能",
    runtimeSkillsShort: "技能",
    runtimeTargetHint: "请在发送第一条消息前确定当前会话目标。",
    runtimeSkillHint: "请在发送第一条消息前为当前会话选择 Skill。",
    runtimeModelHint: "切换后会作用于当前会话后续发送的消息。",
    runtimeSkillsHint: "为后续消息选择额外启用的技能。",
    runtimeMobile: "会话设置",
    runtimeMobileHint: "为后续消息选择模型与技能。",
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
