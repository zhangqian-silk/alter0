import {
  DEFAULT_BACKEND_PROXY_TARGET,
  DEV_BACKEND_PROXY_PATHS,
  normalizeBackendProxyTarget,
  resolveBackendProxyTarget,
  resolveDevServerProxy
} from "./devServerProxyConfig";

describe("devServerProxyConfig", () => {
  it("uses the default Go backend target when no override is provided", () => {
    expect(resolveBackendProxyTarget()).toBe(DEFAULT_BACKEND_PROXY_TARGET);
  });

  it("normalizes backend proxy targets by trimming whitespace and trailing slash", () => {
    expect(normalizeBackendProxyTarget(" http://127.0.0.1:19090/ ")).toBe("http://127.0.0.1:19090");
  });

  it("falls back to the default target for unsupported schemes", () => {
    expect(normalizeBackendProxyTarget("ws://127.0.0.1:19090")).toBe(DEFAULT_BACKEND_PROXY_TARGET);
  });

  it("resolves the full proxy table for the Go backend paths", () => {
    const proxy = resolveDevServerProxy("http://127.0.0.1:19090/");

    expect(Object.keys(proxy)).toEqual(DEV_BACKEND_PROXY_PATHS);
    expect(proxy["/api"]).toMatchObject({
      target: "http://127.0.0.1:19090",
      changeOrigin: true
    });
    expect(proxy["/healthz"]).toMatchObject({
      target: "http://127.0.0.1:19090"
    });
  });
});
