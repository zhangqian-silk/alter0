/**
 * Built-in theme definitions for Alter0.
 *
 * Each theme describes its metadata (id, name, mode, description)
 * plus a set of preview swatches used by the ThemePicker UI.
 *
 * The actual token values live in `src/styles/tokens.css` under
 * `[data-theme="<id>"]` selectors.  This file is the single
 * source of truth for the *catalogue* of available themes.
 */

export type ThemeMode = "light" | "dark";

export interface ThemePreview {
  /** Page background */
  bg: string;
  /** Card / surface color */
  surface: string;
  /** Primary text */
  text: string;
  /** Brand accent */
  accent: string;
  /** Border / divider */
  border: string;
}

export interface Theme {
  /** Stable identifier, matches the `data-theme` attribute value */
  id: string;
  /** Display name (English) */
  name: string;
  /** Display name (Chinese) */
  nameZh: string;
  /** Light or dark — used for `prefers-color-scheme` matching */
  mode: ThemeMode;
  /** Short description shown in the picker (English) */
  description: string;
  /** Short description shown in the picker (Chinese) */
  descriptionZh: string;
  /** Five-color preview strip for the picker card */
  preview: ThemePreview;
}

export const DEFAULT_THEME_ID = "light";

export const BUILTIN_THEMES: Theme[] = [
  {
    id: "light",
    name: "Light",
    nameZh: "浅色",
    mode: "light",
    description: "Clean light workbench with teal accent. Default daily-driver theme.",
    descriptionZh: "清爽浅色工作台，青绿主色，日常默认主题。",
    preview: {
      bg: "#f6f7f9",
      surface: "#ffffff",
      text: "#111827",
      accent: "#0f9f8f",
      border: "rgba(15,23,42,0.08)",
    },
  },
  {
    id: "dark",
    name: "Dark",
    nameZh: "深色",
    mode: "dark",
    description: "Low-luminance dark theme for focused night sessions.",
    descriptionZh: "低亮度深色主题，适合夜间专注使用。",
    preview: {
      bg: "#0f1115",
      surface: "#1a1e25",
      text: "#e5e7eb",
      accent: "#14b8a6",
      border: "rgba(255,255,255,0.08)",
    },
  },
  {
    id: "maldives",
    name: "Maldives",
    nameZh: "马尔代夫",
    mode: "light",
    description: "Turquoise lagoon, white sand, and lush tropical green.",
    descriptionZh: "碧蓝潟湖、纯白沙滩、翠绿植被，清新通透。",
    preview: {
      bg: "#eef7f8",
      surface: "#ffffff",
      text: "#0d3a48",
      accent: "#2ea85c",
      border: "rgba(0,120,130,0.07)",
    },
  },
  {
    id: "moonrise",
    name: "Moonrise",
    nameZh: "海上升明月",
    mode: "dark",
    description: "Dreamy indigo night with warm golden moonlight on calm waters.",
    descriptionZh: "梦幻靛蓝夜空，暖金月光洒在平静海面上，静谧辽阔。",
    preview: {
      bg: "#0c1a30",
      surface: "#152a4a",
      text: "#ece4d4",
      accent: "#e8c878",
      border: "rgba(200,192,168,0.06)",
    },
  },
  {
    id: "qianli",
    name: "Qianli",
    nameZh: "千里江山图",
    mode: "light",
    description: "Mineral blue-green landscape on warm silk, cinnabar seals.",
    descriptionZh: "石青石绿矿物山水，绢本暖色底，朱砂印章点缀。",
    preview: {
      bg: "#e8dcc0",
      surface: "#f8f2e0",
      text: "#2a2418",
      accent: "#b83a2a",
      border: "rgba(60,48,28,0.08)",
    },
  },
  {
    id: "dunhuang",
    name: "Dunhuang",
    nameZh: "敦煌暖",
    mode: "dark",
    description: "Warm earth tones on a deep cave-dark background.",
    descriptionZh: "深洞底色配温暖土色调，敦煌壁画灵感。",
    preview: {
      bg: "#1a1410",
      surface: "#28201a",
      text: "#f5e6d3",
      accent: "#c2410c",
      border: "rgba(212,196,168,0.08)",
    },
  },
  {
    id: "morandi",
    name: "Morandi",
    nameZh: "莫兰迪",
    mode: "light",
    description: "Muted gray palette. Calm, restrained, editorial.",
    descriptionZh: "柔和灰调，克制安静，编辑室质感。",
    preview: {
      bg: "#f5f5f4",
      surface: "#ffffff",
      text: "#27272a",
      accent: "#71717a",
      border: "rgba(39,39,42,0.08)",
    },
  },
];

/** Look up a theme by id; falls back to the default theme. */
export function getThemeById(id: string | null | undefined): Theme {
  if (!id) {
    return BUILTIN_THEMES[0];
  }
  return BUILTIN_THEMES.find((t) => t.id === id) ?? BUILTIN_THEMES[0];
}

/** Return the first theme whose `mode` matches the given value. */
export function getThemeByMode(mode: ThemeMode): Theme {
  return BUILTIN_THEMES.find((t) => t.mode === mode) ?? BUILTIN_THEMES[0];
}
