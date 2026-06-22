import { useState, type ReactNode } from "react";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { SETTINGS_DEFAULT_SECTION_ROUTE, SETTINGS_ROUTE_GROUPS, toI18nKey } from "../legacyShellConfig";
import { getLegacyShellCopy } from "../legacyShellCopy";
import {
  isReactManagedRouteBody,
} from "../reactManagedRouteContract";
import { ReactManagedCodexAccountsRouteBody } from "./ReactManagedCodexAccountsRouteBody";
import { ReactManagedControlRouteBody } from "./ReactManagedControlRouteBody";
import { ReactManagedMemoryRouteBody } from "./ReactManagedMemoryRouteBody";
import { ReactManagedTerminalRouteBody } from "./ReactManagedTerminalRouteBody";
import { NavIcon } from "./NavIcon";

type RouteBodyRenderer = (props: { language: LegacyShellLanguage }) => React.JSX.Element;

const SETTINGS_ROUTE_BODY_RENDERERS: Record<string, RouteBodyRenderer> = {
  runtime: ({ language }) => <RuntimeSettingsSection language={language} />,
  memory: ({ language }) => <ReactManagedMemoryRouteBody language={language} />,
  schedules: ({ language }) => <ReactManagedControlRouteBody route="cron-jobs" language={language} />,
  skills: ({ language }) => <ReactManagedControlRouteBody route="skills" language={language} />,
};

const REACT_MANAGED_ROUTE_BODY_RENDERERS: Record<"terminal", RouteBodyRenderer> = {
  terminal: () => <ReactManagedTerminalRouteBody />,
};

export {
  getReactManagedRouteBodyRoutes,
  isReactManagedRouteBody,
} from "../reactManagedRouteContract";

export function ReactManagedRouteBody({
  route,
  language,
}: {
  route: string;
  language: LegacyShellLanguage;
}) {
  if (!isReactManagedRouteBody(route)) {
    return null;
  }

  if (route === "terminal") {
    return REACT_MANAGED_ROUTE_BODY_RENDERERS[route]({ language });
  }

  return <SettingsRouteBody language={language} />;
}

function RuntimeSettingsSection({ language }: { language: LegacyShellLanguage }) {
  return (
    <div className="settings-composite-section" data-settings-section="runtime">
      <ReactManagedCodexAccountsRouteBody language={language} />
    </div>
  );
}

function SettingsRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  const [selectedRoute, setSelectedRoute] = useState(SETTINGS_DEFAULT_SECTION_ROUTE);
  const renderSelectedRouteBody = SETTINGS_ROUTE_BODY_RENDERERS[selectedRoute] ?? SETTINGS_ROUTE_BODY_RENDERERS[SETTINGS_DEFAULT_SECTION_ROUTE];
  const children: ReactNode = renderSelectedRouteBody({ language });

  return (
    <section className="settings-route-body" data-settings-route={selectedRoute}>
      <nav className="settings-route-nav" aria-label={copy.settingsSectionsLabel}>
        {SETTINGS_ROUTE_GROUPS.map((group) => (
          <section className="settings-route-nav-group" key={group.heading} data-settings-route-group={toI18nKey(group.heading)}>
            <h4 data-i18n={`nav.${toI18nKey(group.heading)}`}>{copy.headings[group.heading] ?? group.heading}</h4>
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
        {children}
      </div>
    </section>
  );
}
