import {
  CLICK_DIAGNOSTICS_STORAGE_KEY,
  installClickDiagnostics,
  isClickDiagnosticsEnabled,
} from "./clickDiagnostics";

describe("shared debug clickDiagnostics", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    document.body.innerHTML = "";
    localStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("enables diagnostics from query parameters or local storage", () => {
    expect(isClickDiagnosticsEnabled("?debug_clicks=1", localStorage)).toBe(true);
    expect(isClickDiagnosticsEnabled("?debugClicks=true", localStorage)).toBe(true);
    expect(isClickDiagnosticsEnabled("?debug_clicks=0", localStorage)).toBe(false);

    localStorage.setItem(CLICK_DIAGNOSTICS_STORAGE_KEY, "on");
    expect(isClickDiagnosticsEnabled("", localStorage)).toBe(true);
  });

  it("logs the post-dispatch click target, blocking layers, and prevented state", () => {
    const logger = { debug: vi.fn(), warn: vi.fn() };
    const shell = document.createElement("div");
    shell.className = "app-shell overlay-open";
    const backdrop = document.createElement("button");
    backdrop.className = "mobile-backdrop";
    const action = document.createElement("button");
    action.className = "runtime-composer-submit";
    action.ariaLabel = "Send";
    action.addEventListener("click", (event) => event.preventDefault());
    shell.append(backdrop, action);
    document.body.append(shell);
    document.elementFromPoint = vi.fn(() => backdrop);

    const cleanup = installClickDiagnostics({ logger });

    action.dispatchEvent(new MouseEvent("click", {
      bubbles: true,
      cancelable: true,
      clientX: 24,
      clientY: 32,
    }));
    vi.runOnlyPendingTimers();

    expect(logger.debug).toHaveBeenCalledWith("[alter0:click]", expect.objectContaining({
      type: "click",
      defaultPrevented: true,
      target: expect.stringContaining("button.runtime-composer-submit"),
      topElement: expect.stringContaining("button.mobile-backdrop"),
      shellClassName: "app-shell overlay-open",
      blockingLayers: expect.arrayContaining([
        expect.stringContaining("button.mobile-backdrop"),
      ]),
    }));

    cleanup();
  });
});
