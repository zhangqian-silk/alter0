import { useState, type ReactNode } from "react";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { SETTINGS_DEFAULT_SECTION_ROUTE, SETTINGS_ROUTE_GROUPS, toI18nKey } from "../legacyShellConfig";
import { getLegacyShellCopy } from "../legacyShellCopy";
import {
  isReactManagedRouteBody,
} from "../reactManagedRouteContract";
import { ReactManagedCodexAccountsRouteBody } from "./ReactManagedCodexAccountsRouteBody";
import { ReactManagedControlRouteBody } from "./ReactManagedControlRouteBody";
import { NavIcon } from "./NavIcon";

type RouteBodyRenderer = (props: { language: LegacyShellLanguage }) => React.JSX.Element;

const SETTINGS_ROUTE_BODY_RENDERERS: Record<string, RouteBodyRenderer> = {
  runtime: ({ language }) => <RuntimeSettingsSection language={language} />,
  schedules: ({ language }) => <ReactManagedControlRouteBody route="cron-jobs" language={language} />,
  skills: ({ language }) => <ReactManagedControlRouteBody route="skills" language={language} />,
};

export {
  getReactManagedRouteBodyRoutes,
  isReactManagedRouteBody,
} from "../reactManagedRouteContract";

export function ReactManagedRouteBody({
  route,
  language,
  onToggleLanguage,
}: {
  route: string;
  language: LegacyShellLanguage;
  onToggleLanguage: () => void;
}) {
  if (!isReactManagedRouteBody(route)) {
    return null;
  }

  return <SettingsRouteBody language={language} onToggleLanguage={onToggleLanguage} />;
}

function RuntimeSettingsSection({ language }: { language: LegacyShellLanguage }) {
  return (
    <div className="settings-composite-section" data-settings-section="runtime">
      <ReactManagedCodexAccountsRouteBody language={language} />
    </div>
  );
}

function GeneralSettingsSection({
  language,
  onToggleLanguage,
}: {
  language: LegacyShellLanguage;
  onToggleLanguage: () => void;
}) {
  const copy = getLegacyShellCopy(language);

  return (
    <section className="settings-general-section" data-settings-section="general" aria-label={copy.routes.general}>
      <div className="settings-general-panel">
        <div className="settings-general-heading">
          <h4>{copy.routes.general}</h4>
          <p>{copy.routeSubtitles.general}</p>
        </div>
        <button
          className="settings-language-control"
          type="button"
          aria-label={`${copy.localeAriaLabel} ${copy.localeButton}`}
          data-short-lang={copy.localeShort}
          onClick={onToggleLanguage}
        >
          <span className="settings-language-label">{copy.localeAriaLabel}</span>
          <span className="settings-language-value">{copy.localeButton}</span>
        </button>
        <ReactManagedCodexAccountsRouteBody language={language} mode="serviceControls" />
      </div>
    </section>
  );
}

function SettingsRouteBody({
  language,
  onToggleLanguage,
}: {
  language: LegacyShellLanguage;
  onToggleLanguage: () => void;
}) {
  const copy = getLegacyShellCopy(language);
  const [selectedRoute, setSelectedRoute] = useState(SETTINGS_DEFAULT_SECTION_ROUTE);
  const renderSelectedRouteBody = SETTINGS_ROUTE_BODY_RENDERERS[selectedRoute] ?? SETTINGS_ROUTE_BODY_RENDERERS[SETTINGS_DEFAULT_SECTION_ROUTE];
  const children: ReactNode = selectedRoute === "general"
    ? <GeneralSettingsSection language={language} onToggleLanguage={onToggleLanguage} />
    : renderSelectedRouteBody({ language });

  return (
    <section className="settings-route-body" data-settings-route={selectedRoute}>
      <nav className="settings-route-nav" aria-label={copy.settingsSectionsLabel}>
        {SETTINGS_ROUTE_GROUPS.map((group) => (
          <section className="settings-route-nav-group" key={group.heading} data-settings-route-group={toI18nKey(group.heading)}>
            <div className="settings-route-nav-items">
              {group.items.map((item) => (
                <button
                  key={item.route}
                  className={item.route === selectedRoute ? "settings-route-tab is-active" : "settings-route-tab"}
                  type="button"
                  aria-current={item.route === selectedRoute ? "page" : undefined}
                  onClick={() => setSelectedRoute(item.route)}
                >
                  <span className="settings-route-tab-icon" aria-hidden="true">
                    <NavIcon icon={item.icon} />
                  </span>
                  <span className="settings-route-tab-label">{copy.routes[item.route] ?? item.label}</span>
                  <span className="settings-route-tab-shortcut" aria-hidden="true">{item.abbr}</span>
                </button>
              ))}
            </div>
          </section>
        ))}
      </nav>
      <div className="settings-route-content" data-settings-route-content={selectedRoute}>
        <div className="settings-section-frame" data-settings-section-frame={selectedRoute}>
          {children}
        </div>
      </div>
    </section>
  );
}
