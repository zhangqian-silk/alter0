import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  getReactManagedRouteBodyRoutes,
  isReactManagedRouteBody,
  ReactManagedRouteBody,
} from "./ReactManagedRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders settings as the single settings route and switches compact sections without changing paths", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/api/control/runtime/restart") {
        return Promise.resolve(jsonResponse({ status: "idle" }));
      }
      return Promise.resolve(jsonResponse({ items: [] }));
    });

    expect(isReactManagedRouteBody("settings")).toBe(true);
    expect(isReactManagedRouteBody("overview")).toBe(false);
    expect(isReactManagedRouteBody("memory")).toBe(false);

    window.history.replaceState({}, "", "/settings");
    const onToggleLanguage = vi.fn();
    const { container } = render(
      <ReactManagedRouteBody route="settings" language="en" onToggleLanguage={onToggleLanguage} />,
    );

    expect(container.querySelector(".settings-route-body")).toHaveAttribute("data-settings-route", "runtime");
    expect(container.querySelectorAll("[data-settings-route-group]")).toHaveLength(1);
    const settingsTabs = screen.getAllByRole("button").filter((button) => button.classList.contains("settings-route-tab"));
    expect(settingsTabs.map((button) => button.textContent?.trim())).toEqual([
      "RuntimeRU",
      "SkillsSK",
      "SchedulesSC",
      "GeneralGE",
    ]);
    expect(screen.getByRole("button", { name: "Runtime" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("button", { name: "Runtime" }).querySelector(".settings-route-tab-icon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Runtime" }).querySelector(".settings-route-tab-shortcut")).toHaveTextContent("RU");
    expect(container.querySelector(".settings-section-frame")).toHaveAttribute("data-settings-section-frame", "runtime");
    expect(screen.queryByRole("button", { name: "Language" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Restart service" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "General" }));

    await waitFor(() => {
      expect(container.querySelector(".settings-route-body")).toHaveAttribute("data-settings-route", "general");
    });
    expect(container.querySelector(".settings-section-frame")).toHaveAttribute("data-settings-section-frame", "general");
    const languageButton = screen.getByRole("button", { name: "Language English" });
    expect(languageButton).toBeInTheDocument();
    expect(languageButton).toHaveClass("settings-language-control");
    expect(languageButton.querySelector(".settings-language-label")).toHaveTextContent("Language");
    expect(languageButton.querySelector(".settings-language-value")).toHaveTextContent("English");
    expect(screen.getByRole("button", { name: "Restart service" })).toBeInTheDocument();
    fireEvent.click(languageButton);
    expect(onToggleLanguage).toHaveBeenCalledTimes(1);

    expect(window.location.pathname).toBe("/settings");
    expect(screen.queryByRole("button", { name: "Memory" })).not.toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/api/memory/"), expect.anything());
  });

  it("keeps the service restart flow reachable from general settings", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url === "/api/control/llm/providers") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/control/runtime/restart" && init?.method === "POST") {
        return Promise.resolve(
          jsonResponse(
            {
              code: "runtime_restart_discard_confirmation_required",
              error: "tracked changes exist",
            },
            { status: 409 },
          ),
        );
      }
      if (url === "/api/control/runtime/restart") {
        return Promise.resolve(jsonResponse({ status: "idle" }));
      }
      if (url === "/api/control/runtime/restart-candidates") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/codex/accounts") {
        return Promise.resolve(jsonResponse({
          current: {
            account_name: "local",
            runtime_ready: true,
            runtime_model: "gpt-5.5",
            runtime_reasoning: "high",
          },
          accounts: [],
          models: [{ id: "gpt-5.5", name: "GPT-5.5", enabled: true }],
          reasoning_modes: [{ id: "high", name: "High", enabled: true }],
        }));
      }
      return Promise.resolve(jsonResponse({ items: [] }));
    }));

    window.history.replaceState({}, "", "/settings");
    render(<ReactManagedRouteBody route="settings" language="en" onToggleLanguage={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Runtime" })).toHaveAttribute("aria-current", "page");
    });
    expect(screen.queryByRole("button", { name: "Restart service" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "General" }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "General" })).toHaveAttribute("aria-current", "page");
      expect(screen.getByRole("button", { name: "Restart service" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Restart service" }));

    expect(screen.getByRole("dialog", { name: "Restart service?" })).toBeInTheDocument();
    const updateCheckbox = screen.getByRole("checkbox");
    expect(updateCheckbox).toBeChecked();
    await waitFor(() => {
      expect(screen.getByText("No master commits available.")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Restart" }));

    await waitFor(() => {
      expect(screen.getByRole("dialog", { name: "Discard local tracked changes?" })).toBeInTheDocument();
    });
  });

  it("does not treat old settings subpages as react-managed route bodies", () => {
    expect(isReactManagedRouteBody("tasks")).toBe(false);
    expect(isReactManagedRouteBody("skill")).toBe(false);
    expect(isReactManagedRouteBody("codex-accounts")).toBe(false);
  });

  it("tracks the full set of routes now owned by React", () => {
    expect(getReactManagedRouteBodyRoutes()).toEqual([
      "settings",
    ]);
    expect(isReactManagedRouteBody("settings")).toBe(true);
    expect(isReactManagedRouteBody("overview")).toBe(false);
    expect(isReactManagedRouteBody("skill")).toBe(false);
    expect(isReactManagedRouteBody("memory")).toBe(false);
    expect(isReactManagedRouteBody("products")).toBe(false);
    expect(isReactManagedRouteBody("chatRuntime")).toBe(false);
  });
});
