import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  isReactManagedControlRoute,
  ReactManagedControlRouteBody,
} from "./ReactManagedControlRouteBody";
import { ReactManagedSessionsRouteBody } from "./ReactManagedSessionsRouteBody";

export function isReactManagedRouteBody(route: string) {
  return route === "sessions" || isReactManagedControlRoute(route);
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
  if (route === "sessions") {
    return <ReactManagedSessionsRouteBody language={language} />;
  }
  return null;
}
