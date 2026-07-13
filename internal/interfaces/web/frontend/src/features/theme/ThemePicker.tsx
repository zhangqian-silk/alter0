import { useTheme } from "./useTheme";
import type { Theme } from "./themes";

/**
 * Theme picker card grid for the Settings → General section.
 *
 * Each card shows the theme name, a short description, and a
 * five-color preview strip (bg / surface / text / accent / border).
 * The active theme is highlighted with an accent border.
 */
export function ThemePicker({ language = "en" }: { language?: "en" | "zh" }) {
  const { theme, availableThemes, setTheme, isManual, resetToSystem } = useTheme();

  const label = language === "zh" ? "界面主题" : "Appearance";
  const subtitle = language === "zh"
    ? "选择适合你工作环境的配色方案"
    : "Choose a color scheme that suits your workspace";
  const systemLabel = language === "zh" ? "跟随系统" : "System";
  const activeLabel = language === "zh" ? "当前" : "Active";

  return (
    <section
      className="settings-theme-section"
      data-settings-section="theme"
      aria-label={label}
    >
      <div className="settings-theme-panel">
        <div className="settings-general-heading">
          <h4>{label}</h4>
          <p>{subtitle}</p>
        </div>

        <div className="theme-picker-grid" role="radiogroup" aria-label={label}>
          {availableThemes.map((t) => (
            <ThemeCard
              key={t.id}
              theme={t}
              isActive={theme.id === t.id}
              language={language}
              activeLabel={activeLabel}
              onSelect={() => setTheme(t.id)}
            />
          ))}
        </div>

        {!isManual ? (
          <div className="theme-picker-system-note">
            <span>{systemLabel}</span>
            <button
              type="button"
              className="theme-picker-reset"
              onClick={resetToSystem}
            >
              {language === "zh" ? "重置" : "Reset"}
            </button>
          </div>
        ) : null}
      </div>
    </section>
  );
}

/* ── Individual theme card ───────────────────────────────── */

function ThemeCard({
  theme,
  isActive,
  language,
  activeLabel,
  onSelect,
}: {
  theme: Theme;
  isActive: boolean;
  language: "en" | "zh";
  activeLabel: string;
  onSelect: () => void;
}) {
  const name = language === "zh" ? theme.nameZh : theme.name;
  const desc = language === "zh" ? theme.descriptionZh : theme.description;

  return (
    <button
      type="button"
      className={`theme-picker-card${isActive ? " is-active" : ""}`}
      role="radio"
      aria-checked={isActive}
      onClick={onSelect}
      title={desc}
    >
      {/* Color preview strip */}
      <div className="theme-picker-preview" aria-hidden="true">
        <span
          className="theme-picker-swatch theme-picker-swatch--bg"
          style={{ backgroundColor: theme.preview.bg }}
        />
        <span
          className="theme-picker-swatch theme-picker-swatch--surface"
          style={{ backgroundColor: theme.preview.surface }}
        />
        <span
          className="theme-picker-swatch theme-picker-swatch--text"
          style={{ backgroundColor: theme.preview.text }}
        />
        <span
          className="theme-picker-swatch theme-picker-swatch--accent"
          style={{ backgroundColor: theme.preview.accent }}
        />
        <span
          className="theme-picker-swatch theme-picker-swatch--border"
          style={{ backgroundColor: theme.preview.border }}
        />
      </div>

      <div className="theme-picker-meta">
        <span className="theme-picker-name">{name}</span>
        {isActive ? (
          <span className="theme-picker-badge">{activeLabel}</span>
        ) : null}
      </div>
    </button>
  );
}
