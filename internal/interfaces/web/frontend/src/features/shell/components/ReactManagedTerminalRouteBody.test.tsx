import { render, waitFor } from "@testing-library/react";
import { ensureLegacyRuntimeScript } from "../../../bootstrap/loadLegacyRuntime";
import { ReactManagedTerminalRouteBody } from "./ReactManagedTerminalRouteBody";

vi.mock("../../../bootstrap/loadLegacyRuntime", () => ({
  ensureLegacyRuntimeScript: vi.fn(),
}));

declare global {
  interface Window {
    __alter0LegacyRuntime?: {
      mountTerminalRoute?: (container: HTMLElement) => void;
    };
  }
}

describe("ReactManagedTerminalRouteBody", () => {
  beforeEach(() => {
    delete window.__alter0LegacyRuntime;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("mounts the legacy terminal runtime into the React-managed host when available", async () => {
    const script = document.createElement("script");
    vi.mocked(ensureLegacyRuntimeScript).mockReturnValue(script);
    const mountTerminalRoute = vi.fn();
    window.__alter0LegacyRuntime = {
      mountTerminalRoute,
    };

    render(<ReactManagedTerminalRouteBody />);

    await waitFor(() => {
      expect(mountTerminalRoute).toHaveBeenCalledTimes(1);
    });

    const host = mountTerminalRoute.mock.calls[0]?.[0];
    expect(host).toBeInstanceOf(HTMLElement);
    expect(host).toHaveAttribute("data-legacy-terminal-host", "true");
    expect(host.querySelector("[data-terminal-view]")).toBeInTheDocument();
    expect(host.querySelector("[data-terminal-session-pane]")).toBeInTheDocument();
    expect(host.querySelector(".terminal-workspace")).toBeInTheDocument();
  });

  it("waits for the legacy runtime script load before mounting when the api is not ready", async () => {
    const script = document.createElement("script");
    vi.mocked(ensureLegacyRuntimeScript).mockReturnValue(script);
    const mountTerminalRoute = vi.fn();

    render(<ReactManagedTerminalRouteBody />);

    expect(mountTerminalRoute).not.toHaveBeenCalled();

    window.__alter0LegacyRuntime = {
      mountTerminalRoute,
    };
    script.dispatchEvent(new Event("load"));

    await waitFor(() => {
      expect(mountTerminalRoute).toHaveBeenCalledTimes(1);
    });
  });
});
