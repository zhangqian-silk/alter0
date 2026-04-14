import type { LegacyShellLanguage } from "../legacyShellCopy";
import { ReactManagedAgentRouteBody } from "./ReactManagedAgentRouteBody";
import {
  isReactManagedControlRoute,
  ReactManagedControlRouteBody,
} from "./ReactManagedControlRouteBody";
import { ReactManagedMemoryRouteBody } from "./ReactManagedMemoryRouteBody";
import { ReactManagedSessionsRouteBody } from "./ReactManagedSessionsRouteBody";
import { ReactManagedTasksRouteBody } from "./ReactManagedTasksRouteBody";

export function isReactManagedRouteBody(route: string) {
  return (
    route === "agent" ||
    route === "memory" ||
    route === "sessions" ||
    route === "tasks" ||
    isReactManagedControlRoute(route)
  );
}

export function ReactManagedRouteBody({
  route,
  language,
}: {
  route: string;
  language: LegacyShellLanguage;
}) {
  if (isReactManagedControlRoute(route)) {
    return <ReactManagedControlRouteBody route={route} language={language} />;
  }
  if (route === "agent") {
    return <ReactManagedAgentRouteBody language={language} />;
  }
  if (route === "memory") {
    return <ReactManagedMemoryRouteBody language={language} />;
  }
  if (route === "sessions") {
    return <ReactManagedSessionsRouteBody language={language} />;
  }
  if (route === "tasks") {
    return <ReactManagedTasksRouteBody language={language} />;
  }
  return null;
}
