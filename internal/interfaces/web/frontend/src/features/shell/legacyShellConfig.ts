export type NavItem = {
  label: string;
  route: string;
  abbr: string;
  icon: IconName;
  active?: boolean;
};

export type NavGroup = {
  heading: string;
  items: NavItem[];
  bottom?: boolean;
};

export type PromptItem = {
  i18n: string;
  prompt: string;
  label: string;
};

export type IconName =
  | "chat"
  | "agent"
  | "terminal"
  | "products"
  | "memory"
  | "skills"
  | "mcp"
  | "sessions"
  | "tasks"
  | "cron"
  | "channels"
  | "models"
  | "environments"
  | "codex";

export const NAV_GROUPS: NavGroup[] = [
  {
    heading: "Workspace",
    items: [
      { label: "Chat", route: "chat", abbr: "C", icon: "chat", active: true },
      { label: "Agent", route: "agent-runtime", abbr: "AR", icon: "agent" },
      { label: "Terminal", route: "terminal", abbr: "TE", icon: "terminal" }
    ]
  },
  {
    heading: "Agent Studio",
    items: [
      { label: "Profiles", route: "agent", abbr: "AG", icon: "agent" },
      { label: "Products", route: "products", abbr: "PR", icon: "products" },
      { label: "Memory", route: "memory", abbr: "ME", icon: "memory" },
      { label: "Skills", route: "skills", abbr: "SK", icon: "skills" },
      { label: "MCP", route: "mcp", abbr: "MC", icon: "mcp" }
    ]
  },
  {
    heading: "Control",
    items: [
      { label: "Sessions", route: "sessions", abbr: "SE", icon: "sessions" },
      { label: "Tasks", route: "tasks", abbr: "TS", icon: "tasks" },
      { label: "Cron Jobs", route: "cron-jobs", abbr: "CR", icon: "cron" }
    ]
  },
  {
    heading: "Settings",
    bottom: true,
    items: [
      { label: "Channels", route: "channels", abbr: "CH", icon: "channels" },
      { label: "Models", route: "models", abbr: "MO", icon: "models" },
      { label: "Environments", route: "environments", abbr: "EN", icon: "environments" },
      { label: "Codex Accounts", route: "codex-accounts", abbr: "CA", icon: "codex" }
    ]
  }
];

export const PROMPTS: PromptItem[] = [
  {
    i18n: "prompt.journey",
    prompt: "Let's start a new journey!",
    label: "Let's start a new journey!"
  },
  {
    i18n: "prompt.skills",
    prompt: "Can you tell me what skills you have?",
    label: "Can you tell me what skills you have?"
  }
];

export function toI18nKey(value: string) {
  return value.toLowerCase().replace(/[\s-]+/g, "_");
}

export function getNavGroupForRoute(route: string): NavGroup | undefined {
  return NAV_GROUPS.find((group) => group.items.some((item) => item.route === route));
}
