import { useState, type ReactNode } from "react";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { MANAGEMENT_DEFAULT_SECTION_ROUTE, MANAGEMENT_ROUTE_GROUPS, toI18nKey } from "../legacyShellConfig";
import { getLegacyShellCopy } from "../legacyShellCopy";
import {
  isReactManagedRouteBody,
} from "../reactManagedRouteContract";
import { ReactManagedAgentRouteBody } from "./ReactManagedAgentRouteBody";
import { ReactManagedCodexAccountsRouteBody } from "./ReactManagedCodexAccountsRouteBody";
import { ReactManagedControlRouteBody } from "./ReactManagedControlRouteBody";
import { ReactManagedMemoryRouteBody } from "./ReactManagedMemoryRouteBody";
import { ReactManagedSessionsRouteBody } from "./ReactManagedSessionsRouteBody";
import { ReactManagedTerminalRouteBody } from "./ReactManagedTerminalRouteBody";
import { ReactManagedTasksRouteBody } from "./ReactManagedTasksRouteBody";
import { NavIcon } from "./NavIcon";

type RouteBodyRenderer = (props: { language: LegacyShellLanguage }) => React.JSX.Element;

const MANAGEMENT_ROUTE_BODY_RENDERERS: Record<string, RouteBodyRenderer> = {
  agent: ({ language }) => <ReactManagedAgentRouteBody language={language} />,
  memory: ({ language }) => <ReactManagedMemoryRouteBody language={language} />,
  sessions: ({ language }) => <ReactManagedSessionsRouteBody language={language} />,
  tasks: ({ language }) => <ReactManagedTasksRouteBody language={language} />,
  channels: ({ language }) => <ReactManagedControlRouteBody route="channels" language={language} />,
  skills: ({ language }) => <ReactManagedControlRouteBody route="skills" language={language} />,
  mcp: ({ language }) => <ReactManagedControlRouteBody route="mcp" language={language} />,
  models: ({ language }) => <ReactManagedControlRouteBody route="models" language={language} />,
  environments: ({ language }) => <ReactManagedControlRouteBody route="environments" language={language} />,
  "cron-jobs": ({ language }) => <ReactManagedControlRouteBody route="cron-jobs" language={language} />,
  "codex-accounts": ({ language }) => <ReactManagedCodexAccountsRouteBody language={language} />,
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

  return <ManagementRouteBody language={language} />;
}

function ManagementRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  const [selectedRoute, setSelectedRoute] = useState(MANAGEMENT_DEFAULT_SECTION_ROUTE);
  const renderSelectedRouteBody = MANAGEMENT_ROUTE_BODY_RENDERERS[selectedRoute] ?? MANAGEMENT_ROUTE_BODY_RENDERERS[MANAGEMENT_DEFAULT_SECTION_ROUTE];
  const children: ReactNode = renderSelectedRouteBody({ language });

  return (
    <section className="management-route-body" data-management-route={selectedRoute}>
      <nav className="management-route-nav" aria-label={copy.managementSectionsLabel}>
        {MANAGEMENT_ROUTE_GROUPS.map((group) => (
          <section className="management-route-nav-group" key={group.heading} data-management-route-group={toI18nKey(group.heading)}>
            <h4 data-i18n={`nav.${toI18nKey(group.heading)}`}>{copy.headings[group.heading] ?? group.heading}</h4>
            <div className="management-route-nav-items">
              {group.items.map((item) => (
                <button
                  key={item.route}
                  className={item.route === selectedRoute ? "management-route-tab is-active" : "management-route-tab"}
                  type="button"
                  aria-current={item.route === selectedRoute ? "page" : undefined}
                  onClick={() => setSelectedRoute(item.route)}
                >
                  <span className="management-route-tab-icon" aria-hidden="true">
                    <NavIcon icon={item.icon} />
                  </span>
                  <span className="management-route-tab-label">{copy.routes[item.route] ?? item.label}</span>
                  <span className="management-route-tab-shortcut" aria-hidden="true">{item.abbr}</span>
                </button>
              ))}
            </div>
          </section>
        ))}
      </nav>
      <div className="management-route-content" data-management-route-content={selectedRoute}>
        {children}
      </div>
    </section>
  );
}
