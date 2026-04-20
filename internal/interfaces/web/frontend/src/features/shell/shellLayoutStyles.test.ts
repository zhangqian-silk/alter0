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

  it("keeps the shell aligned to the legacy full-width workbench instead of a centered capsule canvas", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("width: 100%;");
    expect(stylesheet).toContain("max-width: 100%;");
    expect(stylesheet).toContain("border-radius: 0;");
    expect(stylesheet).toContain("box-shadow: none;");
    expect(stylesheet).toContain("@media (max-width: 1100px)");
  });

  it("keeps desktop shell surfaces calm and avoids glass-heavy workbench chrome", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("background: #f4f7fb;");
    expect(stylesheet).toContain("backdrop-filter: none;");
    expect(stylesheet).toContain("background: rgba(255, 255, 255, 0.96);");
    expect(stylesheet).toContain("box-shadow: 0 10px 30px -26px rgba(15, 23, 42, 0.16);");
  });

  it("keeps shared route pages on the same restrained workbench surface system", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".route-card,");
    expect(stylesheet).toContain("border-color: rgba(15, 23, 42, 0.08);");
    expect(stylesheet).toContain("background: rgba(255, 255, 255, 0.94);");
    expect(stylesheet).toContain("background: rgba(248, 250, 252, 0.92);");
    expect(stylesheet).toContain("background: rgba(239, 246, 255, 0.92);");
  });

  it("binds mobile viewport css vars into chat and page-mode layouts", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("--mobile-viewport-height: 100dvh;");
    expect(stylesheet).toContain("--keyboard-offset: 0px;");
    expect(stylesheet).toContain(".chat-pane:not(.page-mode) {");
    expect(stylesheet).toContain("height: min(100%, var(--mobile-viewport-height, 100dvh));");
    expect(stylesheet).toContain(".chat-pane.page-mode {");
    expect(stylesheet).toContain("height: min(100%, calc(var(--mobile-viewport-height, 100dvh) + var(--keyboard-offset, 0px)));");
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

  it("anchors narrow shell drawers to the viewport edges instead of floating them inside the desktop canvas", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\.primary-nav,\s*\.session-pane\s*\{[\s\S]*?top:\s*0;[\s\S]*?bottom:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\.session-pane\s*\{[\s\S]*?left:\s*0;[\s\S]*?transform:\s*translateX\(-102%\);/,
    );
    expect(stylesheet).toContain(".app-shell.panel-open:not(.info-mode) .session-pane {");
  });

  it("keeps narrow navigation drawers vertically scrollable so small viewports can reach every menu item", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?\.menu\s*\{[\s\S]*?min-height:\s*0;[\s\S]*?overflow-y:\s*auto;[\s\S]*?-webkit-overflow-scrolling:\s*touch;[\s\S]*?overscroll-behavior-y:\s*contain;/,
    );
  });

  it("anchors the narrow-screen composer to the keyboard offset instead of a fixed viewport bottom", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("@media (max-width: 760px)");
    expect(stylesheet).toContain("bottom: var(--keyboard-offset);");
    expect(stylesheet).toContain("padding: 10px 12px calc(14px + env(safe-area-inset-bottom));");
  });

  it("keeps page-mode workbench shells stretched to the mobile viewport so terminal messages can scroll", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\.workbench-main,\s*\.workbench-pane-shell\s*\{[\s\S]*?min-height:\s*0;[\s\S]*?height:\s*100%;/,
    );
  });

  it("restores legacy narrow-screen header controls and trims welcome spacing in the final mobile overrides", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 1100px\) \{[\s\S]*?\.nav-toggle,\s*\.panel-toggle,\s*\.mobile-new-chat,\s*\.pane-action,\s*\.nav-collapse\s*\{[\s\S]*?display:\s*inline-flex;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 1100px\) \{[\s\S]*?\.app-shell,\s*\.app-shell\.info-mode\s*\{[\s\S]*?display:\s*block;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 1100px\) \{[\s\S]*?\.chat-header\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-areas:[\s\S]*?"menu actions"[\s\S]*?"title title";/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 760px\) \{[\s\S]*?\.welcome-screen\s*\{[\s\S]*?margin:\s*12px auto 16px;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 1100px\) \{[\s\S]*?\.chat-pane\.empty-state \.chat-header-copy\s*\{[\s\S]*?display:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?\.chat-pane\.empty-state \.chat-view\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-rows:\s*minmax\(0, 1fr\) auto;[\s\S]*?min-height:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 1100px\) \{[\s\S]*?\.chat-pane\.empty-state \.composer-shell\s*\{[\s\S]*?margin-top:\s*0;[\s\S]*?align-self:\s*end;[\s\S]*?width:\s*100%;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?\.chat-pane\.empty-state \.welcome-screen\s*\{[\s\S]*?display:\s*flex;[\s\S]*?flex-direction:\s*column;[\s\S]*?align-items:\s*center;[\s\S]*?align-self:\s*center;[\s\S]*?text-align:\s*center;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?\.chat-pane\.empty-state \.prompt-grid\s*\{[\s\S]*?justify-content:\s*center;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?\.chat-pane\.empty-state \.welcome-tag\s*\{[\s\S]*?margin:\s*0 0 6px;[\s\S]*?line-height:\s*1\.1;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 1100px\) \{[\s\S]*?\.chat-pane\.empty-state \.welcome-screen\s*\{[\s\S]*?margin:\s*0 auto;[\s\S]*?align-self:\s*center;/,
    );
    expect(stylesheet).toMatch(
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 760px\) \{[\s\S]*?\.chat-pane\.empty-state \.welcome-screen\s*\{[\s\S]*?margin:\s*14px auto 0;[\s\S]*?align-self:\s*start;/,
    );
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
