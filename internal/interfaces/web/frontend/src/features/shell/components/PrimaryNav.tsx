import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import { NAV_GROUPS, toI18nKey } from "../legacyShellConfig";
import { NavIcon } from "./NavIcon";

type LegacyWindow = Window & {
  toggleLanguage?: () => void;
};

const legacyWindow = window as LegacyWindow;

export function PrimaryNav() {
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
          aria-label="Collapse navigation"
          aria-expanded="true"
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
            <h2 data-i18n={`nav.${toI18nKey(group.heading)}`}>{group.heading}</h2>
            {group.items.map((item) => (
              <button
                key={item.route}
                className={item.active ? "menu-item active" : "menu-item"}
                type="button"
                data-route={item.route}
                data-abbr={item.abbr}
              >
                <span className="menu-icon" aria-hidden="true">
                  <NavIcon icon={item.icon} />
                </span>
                <span className="menu-label" data-i18n={`nav.${toI18nKey(item.route)}`}>
                  {item.label}
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
          aria-label="Language"
          data-short-lang="EN"
          onClick={() => legacyWindow.toggleLanguage?.()}
        >
          English
        </button>
      </div>
    </aside>
  );
}
