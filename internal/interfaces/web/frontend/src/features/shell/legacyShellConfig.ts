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
  | "terminal"
  | "memory"
  | "skills"
  | "mcp"
  | "sessions"
  | "tasks"
  | "cron"
  | "channels"
  | "models"
  | "environments"
  | "codex"
  | "settings";

export const SETTINGS_WORKBENCH_ROUTE = "settings";
export const MANAGEMENT_WORKBENCH_ROUTE = SETTINGS_WORKBENCH_ROUTE;
export const TOP_LEVEL_WORKBENCH_ROUTES = [
  "chat",
  "terminal",
  SETTINGS_WORKBENCH_ROUTE,
] as const;

export type TopLevelWorkbenchRoute = (typeof TOP_LEVEL_WORKBENCH_ROUTES)[number];
export const MANAGEMENT_DEFAULT_SECTION_ROUTE = "runtime";

export const NAV_GROUPS: NavGroup[] = [
  {
    heading: "Workspace",
    items: [
      { label: "Chat", route: "chat", abbr: "C", icon: "chat", active: true },
      { label: "Terminal", route: "terminal", abbr: "TE", icon: "terminal" },
      { label: "Settings", route: SETTINGS_WORKBENCH_ROUTE, abbr: "SE", icon: "settings" }
    ]
  }
];

export const MANAGEMENT_ROUTE_GROUPS: NavGroup[] = [
  {
    heading: "Settings",
    items: [
      { label: "Runtime", route: "runtime", abbr: "RU", icon: "models" },
      { label: "Environments", route: "environments", abbr: "EN", icon: "environments" },
      { label: "Skills", route: "skills", abbr: "SK", icon: "skills" },
      { label: "Memory", route: "memory", abbr: "ME", icon: "memory" },
      { label: "Maintenance", route: "maintenance", abbr: "MA", icon: "cron" },
      { label: "Workspaces", route: "workspaces", abbr: "WO", icon: "sessions" },
      { label: "Schedules", route: "schedules", abbr: "SC", icon: "cron" }
    ]
  }
];

export const ALL_WORKBENCH_ROUTE_GROUPS: NavGroup[] = [
  ...NAV_GROUPS,
  ...MANAGEMENT_ROUTE_GROUPS,
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
  return ALL_WORKBENCH_ROUTE_GROUPS.find((group) => group.items.some((item) => item.route === route));
}
