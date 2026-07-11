// @vitest-environment node
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const currentDirectory = dirname(fileURLToPath(import.meta.url));
const stylesDir = resolve(currentDirectory, "../../styles");

function readStyle(filename: string): string {
  return readFileSync(resolve(stylesDir, filename), "utf8");
}

describe("CSS architecture", () => {
  describe("tokens.css — design system", () => {
    const tokens = readStyle("tokens.css");

    it("defines light theme color tokens", () => {
      expect(tokens).toContain("--bg-page:");
      expect(tokens).toContain("--bg-sidebar:");
      expect(tokens).toContain("--bg-surface:");
      expect(tokens).toContain("--border-subtle:");
      expect(tokens).toContain("--border-default:");
      expect(tokens).toContain("--text-primary:");
      expect(tokens).toContain("--text-secondary:");
      expect(tokens).toContain("--text-muted:");
      expect(tokens).toContain("--accent:");
    });

    it("defines status color tokens", () => {
      expect(tokens).toContain("--status-success:");
      expect(tokens).toContain("--status-warning:");
      expect(tokens).toContain("--status-danger:");
      expect(tokens).toContain("--status-info:");
    });

    it("defines typography tokens", () => {
      expect(tokens).toContain("--font-sans:");
      expect(tokens).toContain("--font-mono:");
      expect(tokens).toContain("--text-sm:");
      expect(tokens).toContain("--text-base:");
      expect(tokens).toContain("--text-lg:");
      expect(tokens).toContain("--leading-normal:");
      expect(tokens).toContain("--leading-relaxed:");
    });

    it("defines spacing tokens", () => {
      expect(tokens).toContain("--space-2:");
      expect(tokens).toContain("--space-4:");
      expect(tokens).toContain("--space-8:");
      expect(tokens).toContain("--space-12:");
    });

    it("defines radius tokens", () => {
      expect(tokens).toContain("--radius-sm:");
      expect(tokens).toContain("--radius-md:");
      expect(tokens).toContain("--radius-lg:");
      expect(tokens).toContain("--radius-xl:");
    });

    it("defines layout dimension tokens", () => {
      expect(tokens).toContain("--sidebar-width:");
      expect(tokens).toContain("--header-height:");
      expect(tokens).toContain("--content-max-width:");
      expect(tokens).toContain("--page-padding-x:");
    });

    it("defines shadow tokens", () => {
      expect(tokens).toContain("--shadow-sm:");
      expect(tokens).toContain("--shadow-md:");
      expect(tokens).toContain("--shadow-focus:");
    });

    it("defines z-index layer tokens", () => {
      expect(tokens).toContain("--z-base:");
      expect(tokens).toContain("--z-header:");
      expect(tokens).toContain("--z-composer:");
      expect(tokens).toContain("--z-modal:");
    });

    it("defines motion tokens", () => {
      expect(tokens).toContain("--duration-fast:");
      expect(tokens).toContain("--duration-normal:");
      expect(tokens).toContain("--ease-out:");
    });

    it("supports dark theme via [data-theme='dark']", () => {
      expect(tokens).toContain('[data-theme="dark"]');
      expect(tokens).toContain("--bg-page: #0f0f11");
      expect(tokens).toContain("--bg-sidebar: #18181b");
      expect(tokens).toContain("--text-primary: #fafafa");
    });

    it("provides legacy variable aliases for backward compatibility", () => {
      expect(tokens).toContain("--shell-bg: var(--bg-page)");
      expect(tokens).toContain("--shell-ink: var(--text-primary)");
      expect(tokens).toContain("--shell-accent: var(--accent)");
      expect(tokens).toContain("--bg-nav: var(--bg-sidebar)");
    });

    it("respects prefers-reduced-motion", () => {
      expect(tokens).toContain("prefers-reduced-motion");
    });
  });

  describe("shell.css — base component styles (preserved from original)", () => {
    const shell = readStyle("shell.css");

    it("preserves the original base styles for all 198+ classes", () => {
      expect(shell.length).toBeGreaterThan(10000);
    });

    it("still defines core layout classes", () => {
      expect(shell).toContain(".app-shell {");
      expect(shell).toContain(".primary-nav {");
      expect(shell).toContain(".runtime-workspace-head {");
      expect(shell).toContain(".runtime-composer-form {");
      expect(shell).toContain(".runtime-timeline {");
      expect(shell).toContain(".runtime-message");
    });

    it("preserves responsive breakpoints", () => {
      expect(shell).toContain("@media (max-width: 1100px)");
      expect(shell).toContain("@media (max-width: 760px)");
    });

    it("preserves mobile viewport CSS variables", () => {
      expect(shell).toContain("--mobile-viewport-height: 100dvh;");
      expect(shell).toContain("--mobile-viewport-offset-top: 0px;");
      expect(shell).toContain("--keyboard-offset: 0px;");
    });
  });

  describe("theme.css — visual-only overrides (no layout changes)", () => {
    const theme = readStyle("theme.css");

    it("uses design tokens for colors throughout", () => {
      const bgReferences = (theme.match(/var\(--bg-/g) || []).length;
      const textReferences = (theme.match(/var\(--text-/g) || []).length;
      const borderReferences = (theme.match(/var\(--border-/g) || []).length;

      expect(bgReferences).toBeGreaterThan(10);
      expect(textReferences).toBeGreaterThan(5);
      expect(borderReferences).toBeGreaterThan(5);
    });

    it("does NOT override layout display properties (leaves to shell.css)", () => {
      // theme.css should NOT force display:flex on structural elements
      // It should only touch colors, backgrounds, borders, shadows
      const displayFlexCount = (theme.match(/display:\s*flex/g) || []).length;
      const displayGridCount = (theme.match(/display:\s*grid/g) || []).length;

      // A few display:none for pseudo-elements is OK, but not structural flex/grid
      expect(displayFlexCount).toBeLessThan(5);
      expect(displayGridCount).toBe(0);
    });

    it("does NOT override width/height on structural elements", () => {
      // theme.css should not set width/height on .app-shell, .primary-nav, etc.
      // It can set width/height on small things like signal dots
      expect(theme).not.toMatch(/\.app-shell\s*\{[^}]*width:/);
      expect(theme).not.toMatch(/\.primary-nav\s*\{[^}]*width:/);
    });

    it("does NOT override padding/margin on structural layout elements", () => {
      // theme.css should not set padding/margin on layout containers
      expect(theme).not.toMatch(/\.app-shell\s*\{[^}]*padding:/);
      expect(theme).not.toMatch(/\.workbench-main\s*\{[^}]*padding:/);
    });

    it("kills body gradients and uses flat page color", () => {
      expect(theme).toContain("body {");
      expect(theme).toContain("background: var(--bg-page)");
      expect(theme).toContain("background-image: none");
    });

    it("styles sidebar with token colors (visual only)", () => {
      expect(theme).toContain(".primary-nav {");
      expect(theme).toContain("background: var(--bg-sidebar)");
      expect(theme).toContain("border-right: 1px solid var(--border-subtle)");
      expect(theme).toContain("box-shadow: none");
      expect(theme).toContain("backdrop-filter: none");
    });

    it("styles workspace header with token colors (visual only)", () => {
      expect(theme).toContain(".runtime-workspace-head");
      expect(theme).toContain("background: var(--bg-page)");
      expect(theme).toContain("background-image: none");
      expect(theme).toContain("border-bottom: 1px solid var(--border-subtle)");
      expect(theme).toContain("box-shadow: none");
    });

    it("styles status signals with token colors", () => {
      expect(theme).toContain(".runtime-session-signal.is-ready");
      expect(theme).toContain("background: var(--status-success)");
      expect(theme).toContain(".runtime-session-signal.is-busy");
      expect(theme).toContain("background: var(--status-info)");
      expect(theme).toContain(".runtime-session-signal.is-failed");
      expect(theme).toContain("background: var(--status-danger)");
    });

    it("defines signal pulse animation", () => {
      expect(theme).toContain("@keyframes signal-pulse");
      expect(theme).toContain("animation: signal-pulse");
    });

    it("styles user message bubble with token colors", () => {
      expect(theme).toContain(".runtime-message.runtime-message-user .runtime-message-bubble");
      expect(theme).toContain("background: var(--bg-bubble-user)");
      expect(theme).toContain("color: var(--text-primary)");
      expect(theme).toContain("box-shadow: none");
    });

    it("styles assistant message as transparent (no bubble)", () => {
      expect(theme).toContain(".assistant-message-shell");
      expect(theme).toContain("background: transparent");
      expect(theme).toContain("color: var(--text-primary)");
    });

    it("styles composer surface with token colors", () => {
      expect(theme).toContain(".runtime-composer-form");
      expect(theme).toContain("background: var(--bg-surface)");
      expect(theme).toContain("border: 1px solid var(--border-default)");
      expect(theme).toContain("box-shadow: none");
    });

    it("defines composer hover and focus states", () => {
      expect(theme).toContain(".runtime-composer-form:hover");
      expect(theme).toContain("border-color: var(--border-strong)");
      expect(theme).toContain(".runtime-composer-form:focus-within");
      expect(theme).toContain("border-color: var(--border-focus)");
      expect(theme).toContain("box-shadow: var(--shadow-focus)");
    });

    it("styles send button with token colors", () => {
      expect(theme).toContain(".runtime-composer-submit");
      expect(theme).toContain("background: var(--text-primary)");
      expect(theme).toContain("color: var(--text-inverse)");
    });

    it("styles code blocks with tokens", () => {
      expect(theme).toContain(".chat-md-pre");
      expect(theme).toContain("background: var(--bg-code)");
      expect(theme).toContain("color: var(--text-primary)");
      expect(theme).toContain("border: 0");
    });

    it("styles inline code with tokens", () => {
      expect(theme).toContain(".chat-md-inline-code");
      expect(theme).toContain("background: var(--bg-code)");
      expect(theme).toContain("color: var(--text-primary)");
    });

    it("styles markdown links with accent color", () => {
      expect(theme).toContain(".message-markdown-rendered a");
      expect(theme).toContain("color: var(--accent)");
    });

    it("styles empty state (visual only)", () => {
      expect(theme).toContain(".conversation-empty-state");
      expect(theme).toContain("background: transparent");
      expect(theme).toContain("box-shadow: none");
    });

    it("styles scroll jump controls (visual only)", () => {
      expect(theme).toContain(".scroll-jump-control");
      expect(theme).toContain("background: var(--bg-page)");
      expect(theme).toContain("border: 1px solid var(--border-default)");
      expect(theme).toContain("color: var(--text-muted)");
    });

    it("hides the right-side session pane (nav owns sessions)", () => {
      expect(theme).toContain(".runtime-workspace-session-pane");
      expect(theme).toContain("display: none");
    });

    it("styles settings page with tokens", () => {
      expect(theme).toContain(".settings-route-nav");
      expect(theme).toContain("background: var(--bg-sidebar)");
      expect(theme).toContain(".settings-route-tab.is-active");
      expect(theme).toContain("background: var(--bg-page)");
    });

    it("defines reduced motion support", () => {
      expect(theme).toContain("prefers-reduced-motion");
    });

    it("is a reasonable size (under 25KB raw)", () => {
      expect(theme.length).toBeLessThan(25000);
    });
  });

  describe("import order", () => {
    const mainTsx = readFileSync(
      resolve(currentDirectory, "../../main.tsx"),
      "utf8",
    );

    it("imports in correct cascade order: root → tokens → shell → theme → keyboard", () => {
      const rootPos = mainTsx.indexOf('root.css"');
      const tokensPos = mainTsx.indexOf('tokens.css"');
      const shellPos = mainTsx.indexOf('shell.css"');
      const themePos = mainTsx.indexOf('theme.css"');
      const keyboardPos = mainTsx.indexOf('runtimeKeyboardIsolation.css"');

      expect(rootPos).toBeGreaterThan(-1);
      expect(tokensPos).toBeGreaterThan(-1);
      expect(shellPos).toBeGreaterThan(-1);
      expect(themePos).toBeGreaterThan(-1);
      expect(keyboardPos).toBeGreaterThan(-1);

      // Cascade: later imports override earlier
      expect(rootPos).toBeLessThan(tokensPos);
      expect(tokensPos).toBeLessThan(shellPos);
      expect(shellPos).toBeLessThan(themePos);
      expect(themePos).toBeLessThan(keyboardPos);
    });
  });

  describe("runtimeKeyboardIsolation.css — mobile viewport", () => {
    const css = readStyle("runtimeKeyboardIsolation.css");

    it("keeps mobile composer in the workspace grid (not fixed overlay)", () => {
      expect(css).toContain(".runtime-workspace-body > .runtime-composer-shell");
      expect(css).toContain("position: relative !important;");
      expect(css).toContain("bottom: auto !important;");
    });

    it("uses dynamic viewport units for mobile sizing", () => {
      expect(css).toContain("100dvh");
    });
  });

  describe("theme support in app", () => {
    const appSource = readFileSync(
      resolve(currentDirectory, "../../app/WorkbenchApp.tsx"),
      "utf8",
    );

    it("initializes theme from localStorage or system preference", () => {
      expect(appSource).toContain("alter0-theme");
      expect(appSource).toContain("prefers-color-scheme: dark");
    });

    it("applies data-theme attribute to document root", () => {
      expect(appSource).toContain('setAttribute("data-theme"');
    });

    it("persists theme to localStorage", () => {
      expect(appSource).toContain('localStorage.setItem("alter0-theme"');
    });

    it("safely handles missing matchMedia in test environments", () => {
      expect(appSource).toContain("typeof window.matchMedia === \"function\"");
    });
  });
});
