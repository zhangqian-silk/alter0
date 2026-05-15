import { useEffect, useState } from "react";
import { NAV_GROUPS } from "../features/shell/legacyShellConfig";

export const DEFAULT_WORKBENCH_ROUTE = "chat";
export type WorkbenchSessionRoute = "chat" | "agent-runtime" | "terminal";

const KNOWN_ROUTES = new Set(
  NAV_GROUPS.flatMap((group) => group.items.map((item) => item.route)),
);
const ROUTE_SESSION_QUERY_KEYS: Record<WorkbenchSessionRoute, string> = {
  chat: "chat_session_id",
  "agent-runtime": "agent_session_id",
  terminal: "terminal_session_id",
};

export function parseWorkbenchHashRoute(hash: string = window.location.hash): string {
  const normalized = hash.replace(/^#\/?/, "").trim().toLowerCase();
  if (!normalized) {
    return DEFAULT_WORKBENCH_ROUTE;
  }
  return KNOWN_ROUTES.has(normalized) ? normalized : DEFAULT_WORKBENCH_ROUTE;
}

export function navigateWorkbenchRoute(route: string): void {
  const normalized = KNOWN_ROUTES.has(route) ? route : DEFAULT_WORKBENCH_ROUTE;
  if (window.location.hash !== `#${normalized}`) {
    window.location.hash = `#${normalized}`;
    return;
  }
  window.dispatchEvent(new HashChangeEvent("hashchange"));
}

export function useWorkbenchRoute(): [string, (route: string) => void] {
  const [route, setRoute] = useState(() => parseWorkbenchHashRoute());

  useEffect(() => {
    const syncRoute = () => setRoute(parseWorkbenchHashRoute());
    window.addEventListener("hashchange", syncRoute);
    return () => window.removeEventListener("hashchange", syncRoute);
  }, []);

  return [route, navigateWorkbenchRoute];
}

export function isConversationRoute(route: string): route is "chat" | "agent-runtime" {
  return route === "chat" || route === "agent-runtime";
}

export function readWorkbenchRouteSessionID(
  route: WorkbenchSessionRoute,
  search: string = window.location.search,
): string {
  return normalizeSessionQueryValue(new URLSearchParams(search).get(ROUTE_SESSION_QUERY_KEYS[route]));
}

export function writeWorkbenchRouteSessionID(
  route: WorkbenchSessionRoute,
  sessionID: string,
): void {
  const normalized = normalizeSessionQueryValue(sessionID);
  const url = new URL(window.location.href);
  const key = ROUTE_SESSION_QUERY_KEYS[route];
  const current = normalizeSessionQueryValue(url.searchParams.get(key));
  if (current === normalized) {
    return;
  }
  if (normalized) {
    url.searchParams.set(key, normalized);
  } else {
    url.searchParams.delete(key);
  }
  window.history.replaceState(window.history.state, "", `${url.pathname}${url.search}${url.hash}`);
}

function normalizeSessionQueryValue(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
