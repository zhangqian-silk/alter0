import { useEffect, useState } from "react";
import { TOP_LEVEL_WORKBENCH_ROUTES } from "../features/shell/legacyShellConfig";
import { sessionRouteToken } from "../shared/session/sessionHash";

export const DEFAULT_WORKBENCH_ROUTE = "chat";
export type WorkbenchSessionRoute = "chat" | "terminal";

const KNOWN_ROUTES = new Set<string>(TOP_LEVEL_WORKBENCH_ROUTES);
const LEGACY_ROUTE_ALIASES: Record<string, string> = {
  "agent-runtime": "chat",
  management: "settings",
};
const SESSION_QUERY_KEY = "session_id";

export function parseWorkbenchRoute(
  pathname: string = window.location.pathname,
): string {
  const route = normalizePathname(pathname).replace(/^\//, "");
  if (LEGACY_ROUTE_ALIASES[route]) {
    return LEGACY_ROUTE_ALIASES[route];
  }
  return KNOWN_ROUTES.has(route) ? route : DEFAULT_WORKBENCH_ROUTE;
}

export function navigateWorkbenchRoute(route: string): void {
  const aliased = LEGACY_ROUTE_ALIASES[route] ?? route;
  const normalized = KNOWN_ROUTES.has(aliased) ? aliased : DEFAULT_WORKBENCH_ROUTE;
  const currentRoute = parseWorkbenchRoute();
  const nextURL = new URL(window.location.href);
  nextURL.pathname = `/${normalized}`;
  nextURL.hash = "";
  if (normalized !== currentRoute || normalized === "chat") {
    nextURL.searchParams.delete(SESSION_QUERY_KEY);
  }
  const nextLocation = `${nextURL.pathname}${nextURL.search}${nextURL.hash}`;
  const currentLocation = `${window.location.pathname}${window.location.search}${window.location.hash}`;
  if (currentLocation !== nextLocation) {
    window.history.pushState(window.history.state, "", nextLocation);
    window.dispatchEvent(new PopStateEvent("popstate"));
    return;
  }
  window.dispatchEvent(new PopStateEvent("popstate"));
}

export function useWorkbenchRoute(): [string, (route: string) => void] {
  const [route, setRoute] = useState(() => parseWorkbenchRoute());

  useEffect(() => {
    const syncRoute = () => setRoute(parseWorkbenchRoute());
    window.addEventListener("popstate", syncRoute);
    return () => {
      window.removeEventListener("popstate", syncRoute);
    };
  }, []);

  return [route, navigateWorkbenchRoute];
}

export function isConversationRoute(route: string): route is "chat" {
  return route === "chat";
}

export function readWorkbenchRouteSessionID(
  route: WorkbenchSessionRoute,
  search: string = window.location.search,
): string {
  void route;
  return normalizeSessionQueryValue(new URLSearchParams(search).get(SESSION_QUERY_KEY));
}

export function writeWorkbenchRouteSessionID(
  route: WorkbenchSessionRoute,
  sessionID: string,
): void {
  const normalized = sessionRouteToken(normalizeSessionQueryValue(sessionID));
  const url = new URL(window.location.href);
  const current = normalizeSessionQueryValue(url.searchParams.get(SESSION_QUERY_KEY));
  url.pathname = `/${route}`;
  url.hash = "";
  if (normalized) {
    url.searchParams.set(SESSION_QUERY_KEY, normalized);
  } else {
    url.searchParams.delete(SESSION_QUERY_KEY);
  }
  const nextLocation = `${url.pathname}${url.search}${url.hash}`;
  const currentLocation = `${window.location.pathname}${window.location.search}${window.location.hash}`;
  if (current === normalized && currentLocation === nextLocation) {
    return;
  }
  window.history.replaceState(window.history.state, "", `${url.pathname}${url.search}${url.hash}`);
}

function normalizeSessionQueryValue(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizePathname(value: string): string {
  const normalized = value.trim().replace(/\/+$/, "");
  return normalized || "/";
}
