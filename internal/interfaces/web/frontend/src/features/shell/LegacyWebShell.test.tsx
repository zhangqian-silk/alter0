import { render, screen } from "@testing-library/react";
import { LegacyWebShell } from "./LegacyWebShell";
import { LEGACY_SHELL_IDS } from "./legacyDomContract";

describe("LegacyWebShell", () => {
  it("renders the legacy shell contract for the runtime bridge", () => {
    render(<LegacyWebShell />);

    expect(document.getElementById(LEGACY_SHELL_IDS.appShell)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.navCollapseButton)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.sessionList)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.messageArea)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.chatRuntimePanel)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.chatRuntimeSheetHost)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.mobileBackdrop)).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-controls-root]")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-note-root]")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-sheet-root]")).toBeInTheDocument();
  });

  it("renders the workspace navigation and composer entrypoints", () => {
    render(<LegacyWebShell />);

    expect(screen.getByRole("button", { name: "Chat" })).toHaveAttribute("data-route", "chat");
    expect(screen.getByRole("button", { name: "Agent" })).toHaveAttribute("data-route", "agent-runtime");
    expect(screen.getByRole("button", { name: "Terminal" })).toHaveAttribute("data-route", "terminal");
    expect(screen.getByRole("button", { name: "Send message" })).toHaveAttribute("data-composer-submit", "chat-main");
    expect(screen.getByLabelText("Input your message")).toHaveAttribute("data-composer-input", "chat-main");
  });
});
