import { normalizeText } from "./RouteBodyPrimitives";

export const CODEX_SLASH_COMMANDS = [
  {
    command: "/apps",
    label: {
      en: "Browse apps",
      zh: "浏览应用",
    },
  },
  {
    command: "/plugins",
    label: {
      en: "Manage plugins",
      zh: "管理插件",
    },
  },
  {
    command: "/hooks",
    label: {
      en: "List hooks",
      zh: "查看 hooks",
    },
  },
  {
    command: "/compact",
    label: {
      en: "Compact context",
      zh: "压缩上下文",
    },
  },
  {
    command: "/diff",
    label: {
      en: "Show git diff",
      zh: "查看 diff",
    },
  },
  {
    command: "/memories",
    label: {
      en: "Manage memories",
      zh: "管理记忆",
    },
  },
  {
    command: "/skills",
    label: {
      en: "Browse skills",
      zh: "浏览技能",
    },
  },
  {
    command: "/init",
    label: {
      en: "Create AGENTS.md",
      zh: "创建 AGENTS.md",
    },
  },
  {
    command: "/mcp",
    label: {
      en: "List MCP tools",
      zh: "查看 MCP 工具",
    },
  },
  {
    command: "/mention",
    label: {
      en: "Attach file",
      zh: "附加文件",
    },
  },
  {
    command: "/model",
    label: {
      en: "Select model",
      zh: "选择模型",
    },
  },
  {
    command: "/fast",
    label: {
      en: "Toggle Fast tier",
      zh: "切换 Fast",
    },
  },
  {
    command: "/plan",
    label: {
      en: "Plan mode",
      zh: "计划模式",
    },
  },
  {
    command: "/goal",
    label: {
      en: "Set goal",
      zh: "设置目标",
    },
  },
  {
    command: "/personality",
    label: {
      en: "Set style",
      zh: "设置风格",
    },
  },
  {
    command: "/ps",
    label: {
      en: "List running sessions",
      zh: "查看运行会话",
    },
  },
  {
    command: "/stop",
    label: {
      en: "Stop running sessions",
      zh: "停止运行会话",
    },
  },
  {
    command: "/review",
    label: {
      en: "Review changes",
      zh: "审查改动",
    },
  },
  {
    command: "/status",
    label: {
      en: "Show status",
      zh: "查看状态",
    },
  },
] as const;

export function codexSlashCommandQuery(value: string) {
  const match = value.match(/^\/[^\s]*/);
  return match ? match[0].toLowerCase() : "";
}

export function buildDraftWithCodexSlashCommand(draft: string, command: string) {
  const remainder = draft.replace(/^\/[^\s]*/, "").trimStart();
  return remainder ? `${command} ${remainder}` : `${command} `;
}

export function isCodexShellSession(session: { shell?: string } | null | undefined) {
  return normalizeText(session?.shell || "").toLowerCase().includes("codex");
}
