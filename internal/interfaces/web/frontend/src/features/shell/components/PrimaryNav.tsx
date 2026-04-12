import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import { NAV_GROUPS, toI18nKey } from "../legacyShellConfig";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../legacyShellCopy";
import { NavIcon } from "./NavIcon";

type LegacyWindow = Window & {
  toggleLanguage?: () => void;
};

type PrimaryNavProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  navCollapsed: boolean;
  onToggleNavCollapsed: () => void;
};

const legacyWindow = window as LegacyWindow;

export function PrimaryNav({
  currentRoute,
  language,
  navCollapsed,
  onToggleNavCollapsed,
}: PrimaryNavProps) {
  const copy = getLegacyShellCopy(language);
  const navToggleLabel = navCollapsed ? copy.navExpandLabel : copy.navCollapseLabel;

  return (
    <aside className="primary-nav">
      <div className="brand">
        <div className="brand-copy">
          <strong>alter0</strong>
        </div>
        <button
          className="nav-collapse"
          id={LEGACY_SHELL_IDS.navCollapseButton}
          type="button"
          aria-label={navToggleLabel}
          aria-expanded={navCollapsed ? "false" : "true"}
          onClick={onToggleNavCollapsed}
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="15 18 9 12 15 6"></polyline>
          </svg>
        </button>
      </div>

      <nav className="menu">
        {NAV_GROUPS.map((group) => (
          <section
            key={group.heading}
            className={group.bottom ? "menu-group menu-group-bottom" : "menu-group"}
          >
            <h2 data-i18n={`nav.${toI18nKey(group.heading)}`}>{copy.headings[group.heading] ?? group.heading}</h2>
            {group.items.map((item) => (
              <button
                key={item.route}
                className={item.route === currentRoute ? "menu-item active" : "menu-item"}
                type="button"
                data-route={item.route}
                data-abbr={item.abbr}
              >
                <span className="menu-icon" aria-hidden="true">
                  <NavIcon icon={item.icon} />
                </span>
                <span className="menu-label" data-i18n={`nav.${toI18nKey(item.route)}`}>
                  {copy.routes[item.route] ?? item.label}
                </span>
              </button>
            ))}
          </section>
        ))}
      </nav>

      <div className="nav-locale">
        <button
          className="locale nav-locale-button"
          type="button"
          aria-label={copy.localeAriaLabel}
          data-short-lang={copy.localeShort}
          onClick={() => legacyWindow.toggleLanguage?.()}
        >
          {copy.localeButton}
        </button>
      </div>
    </aside>
  );
}
