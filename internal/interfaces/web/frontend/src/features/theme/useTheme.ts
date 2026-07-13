import { useContext } from "react";
import { ThemeContext, type ThemeContextValue } from "./ThemeProvider";

/**
 * Access the current theme and theme-switching API.
 *
 * Must be used inside a `<ThemeProvider>`.
 *
 * @example
 * ```tsx
 * const { theme, setTheme, availableThemes } = useTheme();
 * ```
 */
export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within a <ThemeProvider>");
  }
  return ctx;
}
