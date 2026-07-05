import type { WorkbenchSessionRail } from "../../../app/WorkbenchContext";
import { NAV_GROUPS, SETTINGS_WORKBENCH_ROUTE, toI18nKey } from "../legacyShellConfig";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../legacyShellCopy";
import { NavIcon } from "./NavIcon";

type PrimaryNavProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  sessionRail?: WorkbenchSessionRail | null;
  onNavigate: (route: string) => void;
  onDismiss?: () => void;
};

export function PrimaryNav({
  currentRoute,
  language,
  sessionRail,
  onNavigate,
  onDismiss,
}: PrimaryNavProps) {
  const copy = getLegacyShellCopy(language);
  const dismissLabel = language === "zh" ? "关闭导航" : "Close navigation";
  const sessionRailTitle = copy.chatSessions;
  const sessionRailPrimaryActionLabel = copy.chatNewShort;

  return (
    <aside
      className={sessionRail ? "primary-nav has-session-rail" : "primary-nav"}
      data-shell-design="light-tech"
    >
      <div className="brand">
        <div className="brand-copy">
          <strong>Alter0</strong>
        </div>
        {onDismiss ? (
          <button
            className="nav-dismiss"
            type="button"
            aria-label={dismissLabel}
            onClick={onDismiss}
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
        ) : null}
      </div>

      <nav className="menu" aria-label={copy.primaryNavLabel}>
        {NAV_GROUPS.map((group) => (
          <section
            key={group.heading}
            className={group.bottom ? "menu-group menu-group-bottom" : "menu-group"}
          >
            {group.items.map((item) => (
              <button
                key={item.route}
                className={item.route === currentRoute ? "menu-item active" : "menu-item"}
                type="button"
                data-route={item.route}
                data-abbr={item.abbr}
                aria-label={copy.routes[item.route] ?? item.label}
                onClick={() => onNavigate(item.route)}
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

      {sessionRail ? (
        <section className="nav-session-rail" data-nav-session-rail={sessionRail.route}>
          <div className="nav-session-rail-head">
            <div className="nav-session-rail-copy">
              <strong>{sessionRailTitle}</strong>
              <span>{sessionRail.countLabel}</span>
            </div>
            <button
              className={[
                "nav-session-rail-action",
                sessionRail.primaryActionClassName,
              ].filter(Boolean).join(" ")}
              type="button"
              onClick={sessionRail.onPrimaryAction}
              {...sessionRail.primaryActionProps}
            >
              <span className="nav-session-rail-action-icon" aria-hidden="true">
                <svg viewBox="0 0 20 20" fill="none" focusable="false">
                  <path d="M10 5.25v9.5" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
                  <path d="M5.25 10h9.5" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
                </svg>
              </span>
              <span>{sessionRailPrimaryActionLabel}</span>
            </button>
          </div>
          <div className="nav-session-rail-body">
            {sessionRail.body}
          </div>
        </section>
      ) : null}

      <div className="nav-locale">
        <button
          className={[
            "locale",
            "nav-locale-button",
            "nav-settings-shortcut",
            currentRoute === SETTINGS_WORKBENCH_ROUTE ? "active" : "",
          ].filter(Boolean).join(" ")}
          type="button"
          aria-label={copy.routes[SETTINGS_WORKBENCH_ROUTE] ?? "Settings"}
          aria-current={currentRoute === SETTINGS_WORKBENCH_ROUTE ? "page" : undefined}
          data-route={SETTINGS_WORKBENCH_ROUTE}
          onClick={() => onNavigate(SETTINGS_WORKBENCH_ROUTE)}
        >
          <span className="menu-icon" aria-hidden="true">
            <NavIcon icon="settings" />
          </span>
          <span>{copy.routes[SETTINGS_WORKBENCH_ROUTE] ?? "Settings"}</span>
        </button>
      </div>
    </aside>
  );
}
