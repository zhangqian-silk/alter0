import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  isReactManagedControlRoute,
  isReactManagedRouteBody,
} from "../reactManagedRouteContract";
import { ReactManagedAgentRouteBody } from "./ReactManagedAgentRouteBody";
import { ReactManagedControlRouteBody } from "./ReactManagedControlRouteBody";
import { ReactManagedMemoryRouteBody } from "./ReactManagedMemoryRouteBody";
import { ReactManagedProductsRouteBody } from "./ReactManagedProductsRouteBody";
import { ReactManagedSessionsRouteBody } from "./ReactManagedSessionsRouteBody";
import { ReactManagedTasksRouteBody } from "./ReactManagedTasksRouteBody";

export { isReactManagedRouteBody } from "../reactManagedRouteContract";

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
  if (route === "products") {
    return <ReactManagedProductsRouteBody language={language} />;
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
