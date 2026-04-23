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

  it("gives the environments runtime header a grouped control bar instead of a flat strip of equal-weight buttons", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".environment-toolbar-head {");
    expect(stylesheet).toContain(".environment-toolbar-copy h3 {");
    expect(stylesheet).toContain(".environment-toolbar-panel {");
    expect(stylesheet).toContain("grid-template-columns: minmax(0, 1.25fr) minmax(280px, 0.75fr);");
    expect(stylesheet).toContain(".environment-toolbar-action-group-primary {");
    expect(stylesheet).toContain(".environment-toolbar-button[data-variant=\"primary\"] {");
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.environment-runtime-meta \{[\s\S]*?grid-template-columns:\s*1fr;[\s\S]*?\.environment-toolbar-action-group \{[\s\S]*?width:\s*100%;/,
    );
  });

  it("renders restart confirmation as a centered popup on narrow screens instead of dropping to a bottom sheet", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".modal-backdrop {");
    expect(stylesheet).toContain(".modal-dialog {");
    expect(stylesheet).toContain("width: min(100%, 460px);");
    expect(stylesheet).toContain("grid-template-rows: auto minmax(0, 1fr) auto;");
    expect(stylesheet).toContain(".modal-footer button {");
    expect(stylesheet).toContain("border-radius: 999px;");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(37, 99, 235, 0.96) 0%, rgba(29, 78, 216, 0.96) 100%);");
    expect(stylesheet).toContain(".modal-footer button[data-variant=\"secondary\"] {");
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.modal-backdrop \{[\s\S]*?align-items:\s*center;[\s\S]*?padding:\s*16px;[\s\S]*?\.modal-dialog \{[\s\S]*?width:\s*min\(100%, 420px\);[\s\S]*?border-radius:\s*24px;/,
    );
  });

  it("binds mobile viewport css vars into chat and page-mode layouts", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("--mobile-viewport-height: 100dvh;");
    expect(stylesheet).toContain("--keyboard-offset: 0px;");
    expect(stylesheet).toContain("height: calc(var(--mobile-viewport-height, 100dvh) + var(--keyboard-offset, 0px));");
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

  it("aligns terminal mobile header actions with the shared terminal workspace control chrome", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".conversation-mobile-action {");
    expect(stylesheet).toContain(".terminal-mobile-header .conversation-mobile-action {");
    expect(stylesheet).toContain("border-radius: 999px;");
    expect(stylesheet).toContain("font-size: 12px;");
    expect(stylesheet).toContain("font-weight: 700;");
    expect(stylesheet).toContain("background: rgba(248, 252, 255, 0.76);");
    expect(stylesheet).toContain(".terminal-mobile-header .conversation-mobile-action.is-primary {");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(239, 246, 255, 0.98) 0%, rgba(219, 234, 254, 0.92) 100%);");
    expect(stylesheet).toContain("box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.82), 0 10px 20px -18px rgba(37, 99, 235, 0.32);");
    expect(stylesheet).toContain(".conversation-workspace-body {");
    expect(stylesheet).toContain("[data-conversation-view] .conversation-workspace-body {");
    expect(stylesheet).toContain("grid-template-rows: auto auto minmax(0, 1fr) auto;");
  });

  it("keeps conversation runtime header controls denser than terminal workspace tools", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("[data-conversation-view] .conversation-workspace-actions .terminal-inline-button {");
    expect(stylesheet).toContain("min-height: 32px;");
    expect(stylesheet).toContain("padding: 0 10px;");
    expect(stylesheet).toContain("font-size: 11px;");
  });

  it("gives terminal-runtime pages a richer composer surface instead of the default terminal footer slab", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".terminal-runtime-view .terminal-composer-shell.conversation-composer-shell {");
    expect(stylesheet).toContain("padding: 10px 16px 14px;");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(241, 245, 249, 0) 0%, rgba(241, 245, 249, 0.78) 18%, rgba(241, 245, 249, 0.96) 100%);");
    expect(stylesheet).toContain(".terminal-runtime-view .terminal-chat-form.conversation-chat-form {");
    expect(stylesheet).toContain("border-radius: 28px;");
    expect(stylesheet).toContain("background: rgba(255, 255, 255, 0.96);");
    expect(stylesheet).toContain(".terminal-runtime-view .terminal-composer-input.conversation-composer-input {");
    expect(stylesheet).toContain("min-height: 88px;");
    expect(stylesheet).toContain("font-size: 15px;");
    expect(stylesheet).toContain(".terminal-runtime-view .terminal-chat-form .terminal-composer-meta {");
    expect(stylesheet).toContain("min-height: 26px;");
    expect(stylesheet).toContain(".terminal-runtime-view .terminal-chat-submit {");
    expect(stylesheet).toContain("width: 40px;");
    expect(stylesheet).toContain("height: 40px;");
    expect(stylesheet).toContain("background: linear-gradient(180deg, #1f6feb 0%, #1747a6 100%);");
    expect(stylesheet).toContain("color: #f8fbff;");
  });

  it("locks conversation empty states in place so mobile overscroll cannot drag the header away", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".conversation-console-panel.is-empty {");
    expect(stylesheet).toContain("overflow: hidden;");
    expect(stylesheet).toContain(".terminal-runtime-view .conversation-chat-screen.is-empty {");
    expect(stylesheet).toContain("overscroll-behavior: none;");
    expect(stylesheet).toContain("touch-action: none;");
    expect(stylesheet).toContain("-webkit-overflow-scrolling: auto;");
  });

  it("keeps the conversation chat viewport in its own scroll container above the composer", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".conversation-chat-screen {");
    expect(stylesheet).toContain("height: 100%;");
    expect(stylesheet).toContain("min-height: 0;");
    expect(stylesheet).toContain("padding: 18px 20px 18px;");
  });

  it("drops blur-heavy mobile chrome so runtime surfaces stay responsive on phones", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("@media (max-width: 1100px)");
    expect(stylesheet).toContain("body::before,");
    expect(stylesheet).toContain("body::after {");
    expect(stylesheet).toContain("display: none;");
    expect(stylesheet).toContain(".conversation-session-pane-backdrop {");
    expect(stylesheet).toContain("backdrop-filter: none;");
    expect(stylesheet).toContain(".conversation-session-pane-shell {");
    expect(stylesheet).toContain("[data-conversation-view] .conversation-workspace-body {");
    expect(stylesheet).toContain(".mobile-backdrop {");
  });

  it("polishes conversation runtime session cards and inspector panels to match the calmer workbench system", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".terminal-runtime-view .conversation-session-card {");
    expect(stylesheet).toContain("border-radius: 14px;");
    expect(stylesheet).toContain(".terminal-runtime-view .conversation-session-card.is-active {");
    expect(stylesheet).toContain("background: linear-gradient(180deg, #16385b 0%, #0f2740 100%);");
    expect(stylesheet).toContain("box-shadow: 0 18px 30px -24px rgba(15, 39, 64, 0.58);");
    expect(stylesheet).toContain(".conversation-session-select.menu-item.active .conversation-session-title {");
    expect(stylesheet).toContain("color: #f8fbff;");
    expect(stylesheet).toContain(".terminal-runtime-view .conversation-session-delete {");
    expect(stylesheet).toContain("min-height: auto;");
    expect(stylesheet).toContain(".terminal-runtime-view .conversation-inspector {");
    expect(stylesheet).toContain("border-radius: 26px;");
    expect(stylesheet).toContain("box-shadow: 0 18px 36px -30px rgba(8, 37, 69, 0.2);");
    expect(stylesheet).toContain(".terminal-runtime-view .conversation-check-item {");
    expect(stylesheet).toContain("padding: 10px 12px;");
  });

  it("keeps the image upload control readable inside the composer tool row", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".conversation-runtime-view .terminal-chat-form .conversation-chat-upload {");
    expect(stylesheet).toContain("width: auto;");
    expect(stylesheet).toContain("min-width: fit-content;");
    expect(stylesheet).toContain("white-space: nowrap;");
  });

  it("uses a compact single-line chat workspace header instead of the old stacked copy block", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(/\.conversation-workspace-row\.is-compact\s*\{[\s\S]*?flex-wrap:\s*nowrap;/);
    expect(stylesheet).toMatch(/\.conversation-workspace-head\.is-compact\s*\{[\s\S]*?display:\s*block;/);
    expect(stylesheet).toContain(".conversation-workspace-copy.is-compact {");
    expect(stylesheet).toContain(".conversation-workspace-copy.is-compact h4 {");
    expect(stylesheet).toContain(".conversation-workspace-row.is-compact .conversation-workspace-actions {");
    expect(stylesheet).toContain("align-items: center;");
    expect(stylesheet).toContain("font-size: 18px;");
    expect(stylesheet).toContain("text-overflow: ellipsis;");
  });

  it("keeps mobile empty-state headers on a single terminal-style row while preserving visible title space", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".conversation-workspace-head.is-mobile-empty {");
    expect(stylesheet).toContain(".conversation-workspace-row.is-mobile-empty {");
    expect(stylesheet).toContain("display: grid;");
    expect(stylesheet).toContain("grid-template-columns: minmax(0, 1fr) auto;");
    expect(stylesheet).toContain(".conversation-workspace-copy.is-mobile-empty {");
    expect(stylesheet).toContain("min-width: 0;");
    expect(stylesheet).toContain(".conversation-workspace-actions.is-mobile-empty {");
    expect(stylesheet).toContain("justify-content: flex-end;");
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

  it("keeps legacy terminal skin scoped to terminal toggles instead of conversation runtime actions", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-terminal.css"), "utf8");

    expect(stylesheet).toContain(".app-shell.info-mode .panel-toggle {");
    expect(stylesheet).toContain(".chat-pane[data-route=\"terminal\"].page-mode .panel-toggle,");
    expect(stylesheet).not.toContain(".app-shell[data-workbench-route=\"chat\"] .conversation-workspace .panel-toggle,");
  });

  it("styles the terminal workspace like a restrained command center with layered session and composer surfaces", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-terminal.css"), "utf8");

    expect(stylesheet).toContain(".terminal-session-card.active {");
    expect(stylesheet).toContain("background: transparent;");
    expect(stylesheet).toContain(".terminal-workspace-body,");
    expect(stylesheet).toContain("linear-gradient(180deg, rgba(255, 255, 255, 0.99) 0%, rgba(243, 247, 251, 0.98) 100%)");
    expect(stylesheet).toContain("box-shadow: 0 30px 70px -54px rgba(15, 23, 42, 0.26), inset 0 1px 0 rgba(255, 255, 255, 0.82);");
    expect(stylesheet).toContain(".terminal-chat-form {");
    expect(stylesheet).toContain("border-radius: 26px;");
    expect(stylesheet).toContain("background: rgba(255, 255, 255, 0.9);");
  });

  it("lays out the terminal composer as a single input surface with a footer tool row", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-terminal.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.terminal-chat-form\s*\{[\s\S]*?grid-template-columns:\s*minmax\(0, 1fr\);/,
    );
    expect(stylesheet).toContain(".terminal-composer-tools {");
    expect(stylesheet).toContain("justify-content: space-between;");
    expect(stylesheet).toContain(".terminal-chat-form .terminal-composer-meta {");
    expect(stylesheet).toContain("border: 1px solid rgba(148, 163, 184, 0.28);");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(239, 246, 255, 0.98) 0%, rgba(219, 234, 254, 0.98) 100%);");
    expect(stylesheet).toContain("box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.86), 0 8px 16px -16px rgba(37, 99, 235, 0.36);");
    expect(stylesheet).toContain(".terminal-chat-form-button-icon svg {");
    expect(stylesheet).toContain("width: 20px;");
    expect(stylesheet).toContain("transform: translateY(0.75px);");
  });

  it("styles shared jump controls as arrow clusters for dialog surfaces", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".scroll-jump-strip {");
    expect(stylesheet).toContain(".scroll-jump-control {");
    expect(stylesheet).toContain(".scroll-jump-control.is-visible {");
    expect(stylesheet).toContain(".scroll-jump-control-icon {");
  });

  it("keeps narrow terminal headers on one line and preserves composer meta visibility", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-terminal.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 420px\) \{[\s\S]*?\.terminal-workspace-row\s*\{[\s\S]*?flex-wrap:\s*wrap;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 420px\) \{[\s\S]*?\.terminal-workspace-actions\s*\{[\s\S]*?flex-wrap:\s*wrap;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-composer-meta\s*\{[\s\S]*?display:\s*block;/,
    );
  });

  it("pins the mobile terminal composer to the viewport and reserves message space above it", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-terminal.css"), "utf8");
    const mobileComposerBlock = stylesheet.match(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-composer-shell\s*\{([\s\S]*?)\n  \}/,
    );

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-composer-shell\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?left:\s*0;[\s\S]*?right:\s*0;[\s\S]*?bottom:\s*var\(--keyboard-offset\);[\s\S]*?padding:\s*0 10px calc\(10px \+ env\(safe-area-inset-bottom\)\);/,
    );
    expect(mobileComposerBlock?.[1]).toBeTruthy();
    expect(mobileComposerBlock?.[1]).not.toContain("transition: bottom");
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-jump-cluster\s*\{[\s\S]*?bottom:\s*calc\(var\(--runtime-composer-rest-inset, var\(--runtime-composer-inset, 0px\)\) \+ 24px\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-chat-screen\s*\{[\s\S]*?padding:\s*var\(--terminal-chat-screen-padding-top\) var\(--terminal-chat-screen-padding-x\) 20px;/,
    );
  });

  it("keeps the mobile terminal jump controls above the fixed composer in the primary shell stylesheet", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-conversation-view\] \.conversation-chat-screen,\s*\[data-terminal-view\] \.terminal-chat-screen\s*\{[\s\S]*?height:\s*calc\(100% - var\(--runtime-composer-inset, 0px\)\);[\s\S]*?max-height:\s*calc\(100% - var\(--runtime-composer-inset, 0px\)\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-jump-cluster\s*\{[\s\S]*?bottom:\s*calc\(var\(--runtime-composer-rest-inset, var\(--runtime-composer-inset, 0px\)\) \+ 24px\);[\s\S]*?right:\s*12px;/,
    );
    expect(stylesheet).not.toContain("transition: bottom 220ms cubic-bezier(0.22, 1, 0.36, 1);");
  });

  it("keeps mobile terminal layouts on a single main surface instead of stacking nested rounded capsules", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-terminal.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-workspace-body(?:\.conversation-workspace-body)?\s*\{[\s\S]*?border-radius:\s*0;[\s\S]*?border-left:\s*0;[\s\S]*?border-right:\s*0;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-chat-screen\s*\{[\s\S]*?border:\s*0;[\s\S]*?border-radius:\s*0;[\s\S]*?background:\s*transparent;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-chat-form\s*\{[\s\S]*?border-radius:\s*22px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-chat-form\s*\{[\s\S]*?backdrop-filter:\s*none;[\s\S]*?box-shadow:\s*0 12px 24px -20px rgba\(15, 23, 42, 0.12\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-chat-form button\s*\{[\s\S]*?box-shadow:\s*none;/,
    );
  });
});
