import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  isReactManagedRouteBody,
  type ReactManagedRouteBodyRoute,
} from "../reactManagedRouteContract";
import { ReactManagedAgentRouteBody } from "./ReactManagedAgentRouteBody";
import { ReactManagedControlRouteBody } from "./ReactManagedControlRouteBody";
import { ReactManagedMemoryRouteBody } from "./ReactManagedMemoryRouteBody";
import { ReactManagedProductsRouteBody } from "./ReactManagedProductsRouteBody";
import { ReactManagedSessionsRouteBody } from "./ReactManagedSessionsRouteBody";
import { ReactManagedTerminalRouteBody } from "./ReactManagedTerminalRouteBody";
import { ReactManagedTasksRouteBody } from "./ReactManagedTasksRouteBody";

type RouteBodyRenderer = (props: { language: LegacyShellLanguage }) => React.JSX.Element;

const REACT_MANAGED_ROUTE_BODY_RENDERERS: Record<ReactManagedRouteBodyRoute, RouteBodyRenderer> = {
  agent: ({ language }) => <ReactManagedAgentRouteBody language={language} />,
  terminal: () => <ReactManagedTerminalRouteBody />,
  products: ({ language }) => <ReactManagedProductsRouteBody language={language} />,
  memory: ({ language }) => <ReactManagedMemoryRouteBody language={language} />,
  sessions: ({ language }) => <ReactManagedSessionsRouteBody language={language} />,
  tasks: ({ language }) => <ReactManagedTasksRouteBody language={language} />,
  channels: ({ language }) => <ReactManagedControlRouteBody route="channels" language={language} />,
  skills: ({ language }) => <ReactManagedControlRouteBody route="skills" language={language} />,
  mcp: ({ language }) => <ReactManagedControlRouteBody route="mcp" language={language} />,
  models: ({ language }) => <ReactManagedControlRouteBody route="models" language={language} />,
  environments: ({ language }) => <ReactManagedControlRouteBody route="environments" language={language} />,
  "cron-jobs": ({ language }) => <ReactManagedControlRouteBody route="cron-jobs" language={language} />,
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

  return REACT_MANAGED_ROUTE_BODY_RENDERERS[route]({ language });
}
