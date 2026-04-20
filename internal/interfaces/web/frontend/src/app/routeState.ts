import { useEffect, useState } from "react";
import { NAV_GROUPS } from "../features/shell/legacyShellConfig";

export const DEFAULT_WORKBENCH_ROUTE = "chat";

const KNOWN_ROUTES = new Set(
  NAV_GROUPS.flatMap((group) => group.items.map((item) => item.route)),
);

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
