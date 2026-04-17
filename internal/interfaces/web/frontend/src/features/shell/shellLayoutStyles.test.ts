import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

describe("shell layout stylesheet", () => {
  it("allows desktop shell panels to shrink within the grid", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.primary-nav,\s*\.session-pane,\s*\.chat-pane\s*\{[\s\S]*?min-width:\s*0;/,
    );
  });

  it("keeps the shell columns adaptive while chat content stays on a shared reading width", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("--shell-nav-width: clamp(");
    expect(stylesheet).toContain("--shell-session-width: clamp(");
    expect(stylesheet).toContain("--shell-layout-max-width:");
    expect(stylesheet).toContain("width: min(100%, var(--shell-layout-max-width));");
    expect(stylesheet).toContain("@media (max-width: 1100px)");
  });

  it("binds mobile viewport css vars into chat and page-mode layouts", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("--mobile-viewport-height: 100dvh;");
    expect(stylesheet).toContain("--keyboard-offset: 0px;");
    expect(stylesheet).toContain(".chat-pane:not(.page-mode) {");
    expect(stylesheet).toContain("height: min(100%, calc(var(--mobile-viewport-height, 100dvh) - 14px));");
    expect(stylesheet).toContain(".chat-pane.page-mode {");
    expect(stylesheet).toContain("height: min(100%, calc(var(--mobile-viewport-height, 100dvh) + var(--keyboard-offset, 0px) - 28px));");
  });

  it("keeps desktop chrome panels visible and only exposes header drawer buttons at narrow breakpoints", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(/\.nav-toggle,\s*\.panel-toggle,\s*\.mobile-new-chat,\s*\.pane-action\s*\{[\s\S]*?display:\s*none;/);
    expect(stylesheet).toContain(".nav-toggle,");
    expect(stylesheet).toContain(".panel-toggle,");
    expect(stylesheet).toContain(".mobile-new-chat,");
    expect(stylesheet).toContain(".pane-action,");
    expect(stylesheet).toContain(".nav-collapse {");
    expect(stylesheet).toContain(".app-shell.info-mode .panel-toggle {");
    expect(stylesheet).toContain(".chat-pane[data-route=\"terminal\"].page-mode .panel-toggle,");
    expect(stylesheet).toContain("@media (max-width: 760px)");
    expect(stylesheet).toContain("width: min(calc(100vw - 24px), 280px);");
    expect(stylesheet).toContain("width: min(calc(100vw - 16px), 360px);");
  });

  it("anchors the narrow-screen composer to the keyboard offset instead of a fixed viewport bottom", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("@media (max-width: 760px)");
    expect(stylesheet).toContain("bottom: var(--keyboard-offset);");
    expect(stylesheet).toContain("padding: 10px 12px calc(14px + env(safe-area-inset-bottom));");
  });

  it("styles shared jump controls as arrow clusters for dialog surfaces", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".scroll-jump-strip {");
    expect(stylesheet).toContain(".scroll-jump-control {");
    expect(stylesheet).toContain(".scroll-jump-control.is-visible {");
    expect(stylesheet).toContain(".scroll-jump-control-icon {");
  });
});
