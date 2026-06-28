export type RuntimeSessionRoute = "chat" | "terminal";

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
