export type RuntimeSessionRoute = "chat";

export const RUNTIME_SESSION_HISTORY_PAGE_TURN_LIMIT = 20;

export type RuntimeSessionDetailOptions = {
  turnBefore?: string;
  turnLimit?: number;
};

export function runtimeSessionEndpoint(
  route: RuntimeSessionRoute,
  path: string = "",
  query: Record<string, string | number | boolean | undefined> = {},
): string {
  const normalizedPath = path.trim().replace(/^\/+/, "");
  const suffix = normalizedPath ? `/${normalizedPath}` : "";
  const search = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    if (typeof value === "undefined" || value === "") {
      return;
    }
    search.set(key, String(value));
  });
  const queryString = search.toString();
  return `/api/${route}/sessions${suffix}${queryString ? `?${queryString}` : ""}`;
}

export function runtimeSessionCollectionEndpoint(route: RuntimeSessionRoute): string {
  return runtimeSessionEndpoint(route);
}

export function runtimeSessionRecoverEndpoint(route: RuntimeSessionRoute): string {
  return runtimeSessionEndpoint(route, "recover");
}

export function runtimeSessionDetailEndpoint(
  route: RuntimeSessionRoute,
  sessionID: string,
  options: RuntimeSessionDetailOptions = {},
): string {
  return runtimeSessionEndpoint(route, encodeURIComponent(sessionID), {
    turn_before: normalizeRuntimeSessionQueryText(options.turnBefore),
    turn_limit: options.turnLimit,
  });
}

export function runtimeSessionInputEndpoint(route: RuntimeSessionRoute, sessionID: string): string {
  return runtimeSessionEndpoint(route, `${encodeURIComponent(sessionID)}/input`);
}

export function runtimeSessionPinEndpoint(route: RuntimeSessionRoute, sessionID: string): string {
  return runtimeSessionEndpoint(route, `${encodeURIComponent(sessionID)}/pin`);
}

export function runtimeSessionAttachmentsEndpoint(route: RuntimeSessionRoute, sessionID: string): string {
  return runtimeSessionEndpoint(route, `${encodeURIComponent(sessionID)}/attachments`);
}

export function runtimeSessionEventDetailEndpoint(
  route: RuntimeSessionRoute,
  sessionID: string,
  turnID: string,
  eventID: string,
): string {
  return runtimeSessionEndpoint(
    route,
    `${encodeURIComponent(sessionID)}/turns/${encodeURIComponent(turnID)}/events/${encodeURIComponent(eventID)}`,
  );
}

function normalizeRuntimeSessionQueryText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
