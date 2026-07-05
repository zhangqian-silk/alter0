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
  | "memory"
  | "skills"
  | "cron"
  | "models"
  | "codex"
  | "settings";

export const SETTINGS_WORKBENCH_ROUTE = "settings";
export const TOP_LEVEL_WORKBENCH_ROUTES = [
  "chat",
  SETTINGS_WORKBENCH_ROUTE,
] as const;

export type TopLevelWorkbenchRoute = (typeof TOP_LEVEL_WORKBENCH_ROUTES)[number];
export const SETTINGS_DEFAULT_SECTION_ROUTE = "runtime";

export const NAV_GROUPS: NavGroup[] = [
  {
    heading: "Workspace",
    items: [
      { label: "Chat", route: "chat", abbr: "C", icon: "chat", active: true }
    ]
  }
];

export const SETTINGS_ROUTE_GROUPS: NavGroup[] = [
  {
    heading: "Settings",
    items: [
      { label: "Runtime", route: "runtime", abbr: "RU", icon: "models" },
      { label: "Skills", route: "skills", abbr: "SK", icon: "skills" },
      { label: "Memory", route: "memory", abbr: "ME", icon: "memory" },
      { label: "Schedules", route: "schedules", abbr: "SC", icon: "cron" },
      { label: "General", route: "general", abbr: "GE", icon: "settings" }
    ]
  }
];

export const ALL_WORKBENCH_ROUTE_GROUPS: NavGroup[] = [
  ...NAV_GROUPS,
  ...SETTINGS_ROUTE_GROUPS,
];

export const PROMPTS: PromptItem[] = [];

export function toI18nKey(value: string) {
  return value.toLowerCase().replace(/[\s-]+/g, "_");
}

export function getNavGroupForRoute(route: string): NavGroup | undefined {
  return ALL_WORKBENCH_ROUTE_GROUPS.find((group) => group.items.some((item) => item.route === route));
}
