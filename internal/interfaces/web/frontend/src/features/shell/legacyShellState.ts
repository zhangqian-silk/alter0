import { useEffect, useState } from "react";
import { NAV_GROUPS } from "./legacyShellConfig";
import { MOBILE_VIEWPORT_BREAKPOINT_PX } from "../../shared/viewport/mobileViewport";

export const LEGACY_SHELL_DEFAULT_ROUTE = "chat";
export const LEGACY_SESSION_HISTORY_STORAGE_KEY = "alter0.web.session-history-panel.v1";
export const LEGACY_SHELL_MOBILE_BREAKPOINT_PX = MOBILE_VIEWPORT_BREAKPOINT_PX;

const LEGACY_CHAT_ROUTES = new Set(["chat", "agent-runtime"]);
const LEGACY_SHELL_ROUTES = new Set(
  NAV_GROUPS.flatMap((group) => group.items.map((item) => item.route)),
);

export function parseLegacyShellPathRoute(pathname: string = window.location.pathname): string {
  const normalized = pathname.replace(/^\/?/, "").replace(/\/+$/, "").trim().toLowerCase();
  if (!normalized) {
    return LEGACY_SHELL_DEFAULT_ROUTE;
  }
  return LEGACY_SHELL_ROUTES.has(normalized) ? normalized : LEGACY_SHELL_DEFAULT_ROUTE;
}

export function isLegacyShellChatRoute(route: string): boolean {
  return LEGACY_CHAT_ROUTES.has(route);
}

export function isLegacyShellMobileViewport(): boolean {
  if (typeof window.matchMedia !== "function") {
    return false;
  }
  return window.matchMedia(`(max-width: ${LEGACY_SHELL_MOBILE_BREAKPOINT_PX}px)`).matches;
}

export function useLegacyShellRoute(): string {
  const [route, setRoute] = useState(() => parseLegacyShellPathRoute());

  useEffect(() => {
    const syncRoute = () => {
      setRoute(parseLegacyShellPathRoute());
    };

    window.addEventListener("popstate", syncRoute);
    return () => window.removeEventListener("popstate", syncRoute);
  }, []);

  return route;
}

function getLegacyShellSessionStorage(): Storage | null {
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}

export function loadLegacySessionHistoryCollapsed(): boolean {
  const storage = getLegacyShellSessionStorage();
  if (!storage) {
    return false;
  }

  const raw = storage.getItem(LEGACY_SESSION_HISTORY_STORAGE_KEY);
  if (!raw) {
    return false;
  }

  try {
    const parsed = JSON.parse(raw);
    return Boolean(parsed?.collapsed_state);
  } catch {
    return raw === "1";
  }
}

export function persistLegacySessionHistoryCollapsed(collapsed: boolean): void {
  const storage = getLegacyShellSessionStorage();
  if (!storage) {
    return;
  }

  try {
    storage.setItem(
      LEGACY_SESSION_HISTORY_STORAGE_KEY,
      JSON.stringify({ collapsed_state: collapsed }),
    );
  } catch {
  }
}
