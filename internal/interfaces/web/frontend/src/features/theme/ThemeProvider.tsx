import {
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  BUILTIN_THEMES,
  DEFAULT_THEME_ID,
  getThemeById,
  getThemeByMode,
  type Theme,
  type ThemeMode,
} from "./themes";

const STORAGE_KEY = "alter0.web.theme.v1";
const MANUAL_FLAG_KEY = "alter0.web.theme.manual.v1";

/* ── Context ─────────────────────────────────────────────── */

export interface ThemeContextValue {
  /** Currently active theme (full metadata object). */
  theme: Theme;
  /** All themes available for selection. */
  availableThemes: Theme[];
  /** Switch to a different theme by id.  Marks the choice as "manual"
   *  so it survives `prefers-color-scheme` changes. */
  setTheme: (id: string) => void;
  /** Whether the user has explicitly chosen a theme (vs. auto). */
  isManual: boolean;
  /** Revert to system preference. */
  resetToSystem: () => void;
}

export const ThemeContext = createContext<ThemeContextValue | null>(null);

/* ── Helpers ─────────────────────────────────────────────── */

function readStoredTheme(): string | null {
  try {
    return localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

function writeStoredTheme(id: string): void {
  try {
    localStorage.setItem(STORAGE_KEY, id);
  } catch {
    /* storage unavailable — ignore */
  }
}

function readManualFlag(): boolean {
  try {
    return localStorage.getItem(MANUAL_FLAG_KEY) === "1";
  } catch {
    return false;
  }
}

function writeManualFlag(value: boolean): void {
  try {
    if (value) {
      localStorage.setItem(MANUAL_FLAG_KEY, "1");
    } else {
      localStorage.removeItem(MANUAL_FLAG_KEY);
    }
  } catch {
    /* ignore */
  }
}

function getSystemColorScheme(): ThemeMode {
  if (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function"
  ) {
    try {
      return window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light";
    } catch {
      return "light";
    }
  }
  return "light";
}

function applyThemeToDom(id: string): void {
  if (typeof document === "undefined") return;
  document.documentElement.dataset.theme = id;
}

/* ── Provider ────────────────────────────────────────────── */

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themeId, setThemeId] = useState<string>(() => {
    const stored = readStoredTheme();
    if (stored && BUILTIN_THEMES.some((t) => t.id === stored)) {
      return stored;
    }
    /* First visit — match system preference. */
    const systemMode = getSystemColorScheme();
    const matched = getThemeByMode(systemMode);
    return matched.id;
  });

  const [isManual, setIsManual] = useState<boolean>(() => readManualFlag());

  /* Apply to DOM whenever themeId changes. */
  useEffect(() => {
    applyThemeToDom(themeId);
  }, [themeId]);

  /* Listen for system color-scheme changes when NOT in manual mode. */
  useEffect(() => {
    if (isManual) return;
    if (typeof window.matchMedia !== "function") return;

    let mql: MediaQueryList;
    try {
      mql = window.matchMedia("(prefers-color-scheme: dark)");
    } catch {
      return;
    }
    const handler = (e: MediaQueryListEvent) => {
      const mode: ThemeMode = e.matches ? "dark" : "light";
      const matched = getThemeByMode(mode);
      if (matched.id !== themeId) {
        setThemeId(matched.id);
        writeStoredTheme(matched.id);
      }
    };

    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, [isManual, themeId]);

  const setTheme = useCallback((id: string) => {
    const theme = getThemeById(id);
    setThemeId(theme.id);
    writeStoredTheme(theme.id);
    setIsManual(true);
    writeManualFlag(true);
  }, []);

  const resetToSystem = useCallback(() => {
    const mode = getSystemColorScheme();
    const matched = getThemeByMode(mode);
    setThemeId(matched.id);
    writeStoredTheme(matched.id);
    setIsManual(false);
    writeManualFlag(false);
  }, []);

  const theme = useMemo(() => getThemeById(themeId), [themeId]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      theme,
      availableThemes: BUILTIN_THEMES,
      setTheme,
      isManual,
      resetToSystem,
    }),
    [theme, setTheme, isManual, resetToSystem],
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

/* ── Re-export convenience types ─────────────────────────── */

export type { Theme, ThemeMode } from "./themes";
export { BUILTIN_THEMES, DEFAULT_THEME_ID, getThemeById };
