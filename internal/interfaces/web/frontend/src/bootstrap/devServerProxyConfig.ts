import type { ProxyOptions } from "vite";

export const DEFAULT_BACKEND_PROXY_TARGET = "http://127.0.0.1:18088";

export const DEV_BACKEND_PROXY_PATHS = [
  "/api",
  "/login",
  "/logout",
  "/healthz",
  "/readyz",
  "/metrics"
] as const;

export function normalizeBackendProxyTarget(rawTarget: string | undefined): string {
  const trimmedTarget = String(rawTarget || "").trim();
  if (!trimmedTarget) {
    return DEFAULT_BACKEND_PROXY_TARGET;
  }

  try {
    const parsed = new URL(trimmedTarget);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return DEFAULT_BACKEND_PROXY_TARGET;
    }
    parsed.hash = "";
    parsed.search = "";
    return parsed.toString().replace(/\/$/, "");
  } catch {
    return DEFAULT_BACKEND_PROXY_TARGET;
  }
}

export function resolveBackendProxyTarget(rawTarget: string | undefined = process.env.ALTER0_WEB_BACKEND_ORIGIN): string {
  return normalizeBackendProxyTarget(rawTarget);
}

export function resolveDevServerProxy(rawTarget?: string): Record<string, ProxyOptions> {
  const target = resolveBackendProxyTarget(rawTarget);
  const proxyEntries = DEV_BACKEND_PROXY_PATHS.map((pathname) => [
    pathname,
    {
      target,
      changeOrigin: true
    } satisfies ProxyOptions
  ]);
  return Object.fromEntries(proxyEntries);
}
