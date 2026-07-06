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
    expect(stylesheet).toContain("--shell-radius-xl: 14px;");
    expect(stylesheet).toContain("--shell-radius-sm: 8px;");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(248, 252, 255, 0.86) 0%, rgba(233, 244, 255, 0.76) 100%);");
    expect(stylesheet).toContain("backdrop-filter: blur(20px);");
    expect(stylesheet).toContain("box-shadow: 0 24px 64px -44px rgba(8, 37, 69, 0.24);");
    expect(stylesheet).toMatch(
      /\.primary-nav,\s*\.session-pane,\s*\.chat-pane\s*\{[\s\S]*?border-radius:\s*var\(--shell-radius-xl\);/,
    );
  });

  it("applies the Gemini-style flat workbench reset after legacy rounded surfaces", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const flatReset = stylesheet.slice(stylesheet.lastIndexOf("/* Gemini-style flat workbench reset */"));

    expect(flatReset).toContain("/* Gemini-style flat workbench reset */");
    expect(stylesheet.lastIndexOf("/* Gemini-style flat workbench reset */")).toBeGreaterThan(
      stylesheet.lastIndexOf("/* Codex-style chatRuntime markdown final override */"),
    );
    expect(flatReset).toMatch(
      /\.app-shell,\s*\.workbench-main,\s*\.chat-pane,\s*\.runtime-workspace,\s*\.runtime-workspace-body,\s*\.runtime-workspace-panel,\s*\.runtime-workspace-screen,\s*\.route-view\[data-route-family="settings"\]\.workbench-route-frame\s*\{[\s\S]*?border:\s*0;[\s\S]*?border-radius:\s*0;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(flatReset).toMatch(
      /\.primary-nav\s*\{[\s\S]*?background:\s*#f7f7f7;[\s\S]*?border:\s*0;[\s\S]*?border-right:\s*1px solid #eeeeee;[\s\S]*?border-radius:\s*0;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(flatReset).toMatch(
      /\.settings-route-nav,\s*\.settings-route-content \.route-card,\s*\.settings-route-content \.route-surface,\s*\.settings-route-content \.route-data-table-wrap,\s*\.settings-route-content \.route-detail-panel,\s*\.settings-route-content \.control-task-drawer-panel,\s*\.settings-route-content \.codex-accounts-panel\s*\{[\s\S]*?border:\s*0;[\s\S]*?border-radius:\s*0;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(flatReset).toMatch(
      /\.runtime-composer-form\s*\{[\s\S]*?border:\s*0;[\s\S]*?border-radius:\s*999px;/,
    );
    expect(flatReset).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.settings-route-nav \{[\s\S]*?border-right:\s*0;[\s\S]*?background:\s*#ffffff;/,
    );
  });

  it("keeps the Settings route shell static without page-enter motion", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const flatReset = stylesheet.slice(stylesheet.lastIndexOf("/* Gemini-style flat workbench reset */"));

    expect(flatReset).toMatch(
      /\.route-view\[data-route-family="settings"\]\.workbench-route-frame,\s*\.route-view\[data-route-family="settings"\] \.route-head,\s*\.route-view\[data-route-family="settings"\] \.route-mobile-head,\s*\.route-view\[data-route-family="settings"\] \.route-body,\s*\.settings-route-body,\s*\.settings-route-nav,\s*\.settings-route-content,\s*\.settings-route-content > \*\s*\{[\s\S]*?animation:\s*none;[\s\S]*?transition:\s*none;[\s\S]*?transform:\s*none;/,
    );
  });

  it("defines shared selectors for the nav chrome, details overlay, and composer toolbar", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).not.toContain(".brand-mark {");
    expect(stylesheet).toContain(".nav-locale {");
    expect(stylesheet).toContain(".locale {");
    expect(stylesheet).toContain(".runtime-composer-body {");
    expect(stylesheet).toContain(".runtime-composer-toolbar {");
    expect(stylesheet).toContain(".runtime-composer-toolbar-start {");
    expect(stylesheet).toContain(".runtime-composer-toolbar-end {");
    expect(stylesheet).toContain(".runtime-composer-utility {");
    expect(stylesheet).toContain(".workspace-details-layer {");
    expect(stylesheet).toContain("z-index: 130;");
    expect(stylesheet).toContain(".workspace-details-panel {");
    expect(stylesheet).toContain("border-radius: 14px;");
    expect(stylesheet).toContain("background: rgba(255, 255, 255, 0.98);");
  });

  it("renders runtime session empty states as lightweight navigation copy", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.nav-session-rail \.route-empty-panel\s*\{[\s\S]*?min-height:\s*120px;[\s\S]*?padding:\s*18px 16px;[\s\S]*?border-radius:\s*12px;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.primary-nav \.nav-session-rail \.route-empty-panel\s*\{[\s\S]*?min-height:\s*88px;[\s\S]*?padding:\s*14px 10px;[\s\S]*?border-radius:\s*10px;/,
    );
  });

  it("uses one assistant-style composer surface across runtime pages", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.runtime-composer-shell\s*\{[\s\S]*?border-top:\s*0;[\s\S]*?padding:\s*10px clamp\(16px, 5vw, 42px\) 18px;[\s\S]*?background:\s*linear-gradient\(180deg, rgba\(255, 255, 255, 0\) 0%, rgba\(250, 252, 255, 0\.92\) 48%, rgba\(250, 252, 255, 0\.98\) 100%\);/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-composer-form\s*\{[\s\S]*?width:\s*min\(100%, 860px\);[\s\S]*?justify-self:\s*center;[\s\S]*?border-radius:\s*28px;[\s\S]*?background:\s*#fff;[\s\S]*?box-shadow:\s*0 18px 50px -36px rgba\(15, 23, 42, 0\.30\), 0 2px 10px rgba\(60, 64, 67, 0\.10\);/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-composer-input\s*\{[\s\S]*?min-height:\s*78px;[\s\S]*?resize:\s*none;[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-composer-submit\s*\{[\s\S]*?border-color:\s*#0f172a;[\s\S]*?background:\s*#0f172a;[\s\S]*?color:\s*#fff;/,
    );
  });

  it("keeps the mobile runtime composer in the workspace footer instead of a fixed overlay", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");

    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.runtime-workspace-body > \.runtime-composer-shell\s*\{[\s\S]*?grid-row:\s*4;[\s\S]*?position:\s*relative !important;[\s\S]*?bottom:\s*auto !important;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="chatRuntime"\] > \.runtime-composer-shell\s*\{[\s\S]*?grid-row:\s*3;/,
    );
    expect(stylesheet).not.toMatch(/\.runtime-composer-shell,\s*\.runtime-composer-portal-host \.runtime-composer-shell/);
  });

  it("sizes the mobile runtime shell with dynamic viewport CSS while only inner screens scroll", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");

    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.app-shell,\s*\.app-shell\.info-mode\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?top:\s*var\(--mobile-viewport-offset-top, 0px\);[\s\S]*?height:\s*max\(0px, calc\(var\(--mobile-viewport-height, 100dvh\) - var\(--mobile-viewport-offset-top, 0px\)\)\);[\s\S]*?transform:\s*none !important;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-workspace-body,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body,[\s\S]*?\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body > \.runtime-workspace-panel\s*\{[\s\S]*?transform:\s*none !important;[\s\S]*?transition:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.runtime-workspace-body > \.runtime-composer-shell\s*\{[\s\S]*?position:\s*relative !important;[\s\S]*?bottom:\s*auto !important;[\s\S]*?z-index:\s*80;/,
    );
    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.runtime-workspace-screen\s*\{[\s\S]*?padding-bottom:\s*20px;[\s\S]*?scroll-padding-bottom:\s*20px;/,
    );
    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-screen,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-screen\s*\{[\s\S]*?padding-bottom:\s*20px;[\s\S]*?scroll-padding-bottom:\s*20px;/,
    );
    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.runtime-workspace-body\[data-runtime-composer-interactive="false"\] > \.runtime-composer-shell\s*\{[\s\S]*?pointer-events:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.workbench-mobile-overlay-portal\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?z-index:\s*120;[\s\S]*?pointer-events:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.workbench-mobile-overlay-portal \.primary-nav,\s*\.workbench-mobile-overlay-portal \.session-pane,\s*\.runtime-workspace-session-pane\s*\{[\s\S]*?z-index:\s*130;/,
    );
    expect(stylesheet).not.toMatch(/\.runtime-workspace-body\s*\{[^}]*translate3d\(0, var\(--mobile-viewport-offset-top/);
    expect(stylesheet).not.toMatch(/\.runtime-workspace-panel\s*\{[^}]*translate3d\(0, var\(--mobile-viewport-offset-top/);
  });

  it("keeps mobile composer event handling aligned with the online viewport sync path", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");

    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.runtime-workspace-body > \.runtime-composer-shell\s*\{[\s\S]*?bottom:\s*auto !important;[\s\S]*?z-index:\s*80;/,
    );
    expect(stylesheet).not.toMatch(/\.runtime-composer-form\s*\{[\s\S]*?pointer-events:\s*none;/);
    expect(stylesheet).not.toMatch(/\.runtime-composer-shell,\s*\.runtime-composer-portal-host \.runtime-composer-shell\s*\{/);
  });

  it("keeps the mobile runtime composer in normal grid flow during viewport refreshes", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");
    const mobileComposerBlocks = Array.from(stylesheet.matchAll(
      /@media \(max-width: (?:1100|760)px\) \{[\s\S]*?\.runtime-workspace-body(?:\[data-runtime-view="(?:conversation|chatRuntime)"\])? > \.runtime-composer-shell[^{]*\{([^}]*)\}/g,
    ));
    const composerRules = mobileComposerBlocks.map((match) => match[1]).join("\n");

    expect(mobileComposerBlocks.length).toBeGreaterThan(0);
    expect(composerRules).toContain("position: relative !important;");
    expect(composerRules).toContain("bottom: auto !important;");
    expect(composerRules).toContain("transform: none !important;");
    expect(composerRules).not.toContain("bottom: 0;");
    expect(composerRules).not.toContain("var(--keyboard-offset)");
  });

  it("does not reintroduce mobile runtime composer bottom offsets in later shell rules", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const mobileRuntimeComposerBottomRules = Array.from(stylesheet.matchAll(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-shell\s*\{([^}]*)\}/g,
    )).map((match) => match[1]).join("\n");

    expect(mobileRuntimeComposerBottomRules).not.toContain("bottom: 0;");
    expect(mobileRuntimeComposerBottomRules).not.toContain("var(--keyboard-offset)");
  });

  it("keeps the mobile runtime header fixed while preserving the first workspace grid row footprint", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="chatRuntime"\]\s*\{[\s\S]*?--runtime-mobile-header-fixed-height:\s*56px;[\s\S]*?grid-template-rows:\s*var\(--runtime-mobile-header-fixed-height\) minmax\(0, 1fr\) auto;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-panel,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body > \.runtime-workspace-panel,[\s\S]*?\.runtime-workspace-body\[data-runtime-view="chatRuntime"\] > \.runtime-workspace-panel\s*\{[\s\S]*?grid-row:\s*2;[\s\S]*?\}/,
    );
    expect(stylesheet).not.toContain("height: max(0px, calc(var(--mobile-viewport-height, 100dvh) - var(--runtime-mobile-header-fixed-height) - var(--runtime-composer-height, var(--runtime-composer-rest-inset, 0px))))");
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="chatRuntime"\] > \.runtime-workspace-mobile-header\s*\{[\s\S]*?position:\s*fixed !important;[\s\S]*?top:\s*var\(--mobile-viewport-offset-top, 0px\);[\s\S]*?z-index:\s*90;[\s\S]*?transform:\s*none !important;[\s\S]*?transition:\s*none !important;[\s\S]*?animation:\s*none !important;[\s\S]*?contain:\s*layout paint style;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="chatRuntime"\] > \.runtime-workspace-head\s*\{[\s\S]*?display:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="chatRuntime"\] > \.runtime-workspace-panel\s*\{[\s\S]*?grid-row:\s*2;/,
    );
  });

  it("requests content viewport resizing for virtual keyboards where the browser supports it", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const indexHtml = readFileSync(resolve(currentDirectory, "../../../index.html"), "utf8");

    expect(indexHtml).toContain(
      '<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover, interactive-widget=resizes-content">',
    );
  });

  it("lets mobile page-level shells use the dynamic viewport instead of a locked document layer", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");

    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.app-shell,\s*\.app-shell\.info-mode\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?top:\s*var\(--mobile-viewport-offset-top, 0px\);[\s\S]*?height:\s*max\(0px, calc\(var\(--mobile-viewport-height, 100dvh\) - var\(--mobile-viewport-offset-top, 0px\)\)\);/,
    );
    expect(stylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.chat-pane\.page-mode,[\s\S]*?\.runtime-workspace-body\s*\{[\s\S]*?height:\s*100%;[\s\S]*?overflow:\s*hidden;[\s\S]*?transform:\s*none;/,
    );
    expect(stylesheet).toMatch(/Mobile runtime keyboard isolation:[\s\S]*?\.runtime-workspace-body > \.runtime-composer-shell/);
  });

  it("keeps conversation bubbles compact instead of heavy card-like gradients", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.runtime-message \.runtime-message-bubble,\s*\.conversation-message \.msg-bubble\s*\{[\s\S]*?border-color:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-message\.runtime-message-user \.runtime-message-bubble,\s*\.conversation-message\.is-user \.msg-bubble\s*\{[\s\S]*?max-width:\s*min\(72%, 560px\);[\s\S]*?padding:\s*7px 13px;[\s\S]*?border-radius:\s*18px;[\s\S]*?background:\s*#f1f1f1;/,
    );
    expect(stylesheet).toMatch(
      /\.user-message-shell \.chatRuntime-turn-prompt \.chatRuntime-log-text,[\s\S]*?\.user-message-shell \.chatRuntime-log-text\s*\{[\s\S]*?font-size:\s*15px;[\s\S]*?line-height:\s*1\.45;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-message\.runtime-message-assistant \.runtime-message-bubble,\s*\.conversation-message\.is-assistant \.msg-bubble\s*\{[\s\S]*?width:\s*min\(100%, 860px\);[\s\S]*?max-width:\s*min\(100%, 860px\);[\s\S]*?background:\s*transparent;[\s\S]*?padding:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /\.assistant-message-shell \.message-markdown-shell\s*\{[\s\S]*?display:\s*block;/,
    );
    expect(stylesheet).toMatch(
      /\.assistant-message-shell \.message-markdown-body\s*\{[\s\S]*?display:\s*block;/,
    );
    expect(stylesheet).toMatch(
      /\.assistant-message-shell \.message-markdown-toolbar\s*\{[\s\S]*?margin-top:\s*12px;[\s\S]*?margin-bottom:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /\.assistant-message-shell \.chat-md-pre\s*\{[\s\S]*?border-radius:\s*22px;[\s\S]*?background:\s*#f5f5f5;[\s\S]*?padding:\s*18px 20px;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.user-message-shell \.chatRuntime-log-main,\s*\[data-runtime-view="chatRuntime"\] \.user-message-shell \.chatRuntime-log-main\s*\{[\s\S]*?max-width:\s*100%;/,
    );
    expect(stylesheet).not.toContain("background: linear-gradient(180deg, rgba(229, 242, 255, 0.98)");
  });

  it("defines shared runtime message bubbles for chat and chatRuntime, and future runtime pages", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.runtime-message,\s*\.conversation-message\s*\{[\s\S]*?width:\s*100%;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-message \.runtime-message-bubble,\s*\.conversation-message \.msg-bubble\s*\{[\s\S]*?border-color:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-message\.runtime-message-user,\s*\.conversation-message\.is-user\s*\{[\s\S]*?justify-items:\s*end;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-message\.runtime-message-user \.runtime-message-bubble,\s*\.conversation-message\.is-user \.msg-bubble\s*\{[\s\S]*?padding:\s*7px 13px;[\s\S]*?border-radius:\s*18px;[\s\S]*?background:\s*#f1f1f1;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-message\.runtime-message-assistant \.runtime-message-bubble,\s*\.conversation-message\.is-assistant \.msg-bubble\s*\{[\s\S]*?width:\s*min\(100%, 860px\);[\s\S]*?background:\s*transparent;[\s\S]*?padding:\s*0;/,
    );
  });

  it("routes Settings through the shared runtime workspace shell instead of a separate route shell", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const alignmentLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Final shared workbench shell alignment */"));

    expect(alignmentLayer).toContain("/* Final shared workbench shell alignment */");
    expect(alignmentLayer).toMatch(
      /\.runtime-workspace-shell\[data-runtime-view="settings"\]\s*\{[\s\S]*?min-height:\s*0;[\s\S]*?height:\s*100%;[\s\S]*?display:\s*grid;[\s\S]*?grid-template-rows:\s*minmax\(0, 1fr\);[\s\S]*?padding:\s*8px 16px 18px;/,
    );
    expect(alignmentLayer).toMatch(
      /\.runtime-workspace-shell\[data-runtime-view="settings"\] > \.runtime-workspace\s*\{[\s\S]*?height:\s*100%;[\s\S]*?padding:\s*8px 0 18px;/,
    );
    expect(alignmentLayer).toMatch(
      /\.runtime-workspace-body\[data-runtime-view="settings"\]\s*\{[\s\S]*?grid-template-rows:\s*auto minmax\(0, 1fr\);[\s\S]*?overflow:\s*hidden;/,
    );
    const settingsBodyBlock = alignmentLayer.match(
      /\.runtime-workspace-body\[data-runtime-view="settings"\]\s*\{([\s\S]*?)\n\}/,
    )?.[1] || "";
    expect(settingsBodyBlock).not.toContain("scrollbar-gutter: stable;");
    expect(alignmentLayer).toMatch(
      /\.runtime-workspace-body\[data-runtime-view="settings"\] > \.settings-route-body\s*\{[\s\S]*?grid-row:\s*2;[\s\S]*?min-height:\s*0;[\s\S]*?overflow:\s*hidden;/,
    );
    expect(alignmentLayer).toMatch(
      /\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-content\s*\{[\s\S]*?width:\s*100%;[\s\S]*?max-width:\s*100%;[\s\S]*?scrollbar-gutter:\s*stable;/,
    );
  });

  it("places Chat role and time metadata in a quiet bottom row", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.runtime-message \.msg-meta,\s*\.conversation-message \.msg-meta\s*\{[\s\S]*?display:\s*flex;[\s\S]*?gap:\s*6px;[\s\S]*?margin-top:\s*6px;[\s\S]*?font-size:\s*11px;[\s\S]*?color:\s*#8a919d;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-message\.runtime-message-user \.msg-meta,\s*\.conversation-message\.is-user \.msg-meta\s*\{[\s\S]*?justify-content:\s*flex-end;[\s\S]*?padding-right:\s*4px;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-message\.runtime-message-assistant \.msg-meta,\s*\.conversation-message\.is-assistant \.msg-meta\s*\{[\s\S]*?justify-content:\s*flex-start;/,
    );
    expect(stylesheet).toContain(".msg-meta-source,");
    expect(stylesheet).toContain(".msg-meta-time {");
  });

  it("keeps the conversation message area as a borderless reading flow with polished markdown rhythm", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-workspace-screen\s*\{[\s\S]*?border:\s*0;[\s\S]*?border-radius:\s*0;[\s\S]*?background:\s*#fff;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-message\.runtime-message-assistant \.runtime-message-bubble,\s*\[data-runtime-view="conversation"\] \.conversation-message\.is-assistant \.msg-bubble\s*\{[\s\S]*?width:\s*min\(100%, 800px\);[\s\S]*?background:\s*transparent;[\s\S]*?padding:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.message-markdown-rendered\s*\{[\s\S]*?font-size:\s*15\.5px;[\s\S]*?line-height:\s*1\.72;[\s\S]*?color:\s*#1f2937;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.message-markdown-rendered h1,[\s\S]*?\[data-runtime-view="conversation"\] \.message-markdown-rendered h6\s*\{[\s\S]*?font-weight:\s*720;[\s\S]*?letter-spacing:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.chat-md-pre,\s*\[data-runtime-view="conversation"\] \.chatRuntime-final-rendered \.chat-md-pre\s*\{[\s\S]*?border:\s*1px solid #e5e7eb;[\s\S]*?border-radius:\s*10px;[\s\S]*?background:\s*#f7f7f8;/,
    );
    const conversationTableWrapBlock = stylesheet.match(
      /\[data-runtime-view="conversation"\] \.chat-md-table-wrap\s*\{([\s\S]*?)\n\}/,
    )?.[1] || "";
    expect(conversationTableWrapBlock).toContain("border: 0;");
    expect(conversationTableWrapBlock).toContain("border-radius: 0;");
    expect(conversationTableWrapBlock).toContain("background: transparent;");
    const conversationTableBlock = stylesheet.match(
      /\[data-runtime-view="conversation"\] \.chat-md-table\s*\{([\s\S]*?)\n\}/,
    )?.[1] || "";
    expect(conversationTableBlock).toContain("width: 100%;");
    expect(conversationTableBlock).toContain("min-width: 100%;");
    expect(conversationTableBlock).not.toContain("width: max-content;");
    expect(conversationTableBlock).not.toContain("min-width: 520px;");
    const conversationTableCellBlock = stylesheet.match(
      /\[data-runtime-view="conversation"\] \.chat-md-table th,\s*\[data-runtime-view="conversation"\] \.chat-md-table td\s*\{([\s\S]*?)\n\}/,
    )?.[1] || "";
    expect(conversationTableCellBlock).toContain("border: 0;");
    expect(conversationTableCellBlock).toContain("border-bottom: 1px solid #e5e7eb;");
    expect(conversationTableCellBlock).toContain("overflow-wrap: anywhere;");
    expect(conversationTableCellBlock).toContain("word-break: normal;");
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.chat-md-table td :is\(a, code\),\s*\[data-runtime-view="conversation"\] \.chat-md-table th :is\(a, code\)\s*\{[\s\S]*?white-space:\s*nowrap;[\s\S]*?overflow-wrap:\s*normal;/,
    );
    const conversationTableHeaderBlock = stylesheet.match(
      /\[data-runtime-view="conversation"\] \.chat-md-table th\s*\{([\s\S]*?)\n\}/,
    )?.[1] || "";
    expect(conversationTableHeaderBlock).toContain("background: transparent;");
    expect(conversationTableHeaderBlock).toContain("font-weight: 720;");
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.message-markdown-rendered a:not\(\.assistant-inline-image-link\)::after\s*\{[\s\S]*?content:\s*" ↗";/,
    );
  });

  it("presents runtime thinking as an inline disclosure in the current message flow", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.runtime-thinking-shell\.chatRuntime-process-shell\s*\{[\s\S]*?margin:\s*0 0 12px;[\s\S]*?padding:\s*0;[\s\S]*?border:\s*0;[\s\S]*?border-radius:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-thinking-shell\.chatRuntime-process-shell::before\s*\{[\s\S]*?display:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-thinking-toggle\.chatRuntime-process-toggle\s*\{[\s\S]*?display:\s*inline-flex;[\s\S]*?width:\s*auto;[\s\S]*?justify-self:\s*start;[\s\S]*?align-self:\s*start;[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-thinking-shell\.chatRuntime-process-shell\.is-collapsed \.runtime-thinking-toggle,\s*\.runtime-thinking-toggle\.chatRuntime-process-toggle:hover\s*\{[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-thinking-toggle \.chatRuntime-process-title\s*\{[\s\S]*?text-transform:\s*none;[\s\S]*?letter-spacing:\s*0;[\s\S]*?color:\s*#7a7f87;/,
    );
    const mobileThinkingBlocks = Array.from(stylesheet.matchAll(
      /\.runtime-thinking-shell \.chatRuntime-process-body:not\(\[hidden\]\)\s*\{([^}]*)\}/g,
    ));
    const mobileThinkingBlock = mobileThinkingBlocks[mobileThinkingBlocks.length - 1]?.[1] || "";
    expect(mobileThinkingBlock).not.toContain("position: fixed;");
    expect(mobileThinkingBlock).not.toContain("backdrop-filter:");
    expect(mobileThinkingBlock).toContain("max-height: none;");
    expect(mobileThinkingBlock).toContain("overflow: visible;");

    expect(stylesheet).not.toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-thinking-shell \.chatRuntime-process-body:not\(\[hidden\]\)/,
    );
  });

  it("keeps chatRuntime runtime thinking expanded inline on mobile", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const chatRuntimeThinkingBlocks = Array.from(stylesheet.matchAll(
      /\[data-runtime-view="chatRuntime"\] \.runtime-thinking-shell \.chatRuntime-process-body:not\(\[hidden\]\)\s*\{([^}]*)\}/g,
    ));
    const chatRuntimeThinkingBlock = chatRuntimeThinkingBlocks[chatRuntimeThinkingBlocks.length - 1]?.[1] || "";

    expect(chatRuntimeThinkingBlock).toContain("position: static;");
    expect(chatRuntimeThinkingBlock).toContain("max-height: none;");
    expect(chatRuntimeThinkingBlock).toContain("overflow: visible;");
    expect(chatRuntimeThinkingBlock).not.toContain("position: fixed;");
    expect(chatRuntimeThinkingBlock).not.toContain("backdrop-filter:");
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

  it("presents settings sections as a desktop index and a scrollable mobile control strip", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.settings-route-body\s*\{[\s\S]*?grid-template-columns:\s*minmax\(218px, 254px\) minmax\(0, 1fr\);/,
    );
    expect(stylesheet).toMatch(
      /\.route-body\[data-route="settings"\]\s*\{[\s\S]*?display:\s*block;/,
    );
    expect(stylesheet).toMatch(
      /\.settings-route-nav\s*\{[\s\S]*?position:\s*sticky;[\s\S]*?border-radius:\s*12px;/,
    );
    expect(stylesheet).toContain(".settings-route-tab-icon {");
    expect(stylesheet).toContain(".settings-route-tab-shortcut {");
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\.settings-route-body \{[\s\S]*?grid-template-columns:\s*minmax\(0, 1fr\);[\s\S]*?\.settings-route-nav-items \{[\s\S]*?overflow-x:\s*auto;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.settings-route-nav-items \{[\s\S]*?grid-template-columns:\s*repeat\(2, minmax\(0, 1fr\)\);[\s\S]*?overflow:\s*visible;/,
    );
  });

  it("applies one restrained surface system across every settings subsection", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".settings-route-content .route-card,");
    expect(stylesheet).toContain(".settings-route-content .route-surface,");
    expect(stylesheet).toContain(".settings-route-content .route-data-table-wrap,");
    expect(stylesheet).toContain(".settings-route-content .control-task-drawer-panel,");
    expect(stylesheet).toContain(".settings-route-content .codex-accounts-panel");
    expect(stylesheet).toMatch(
      /\.settings-route-content \.route-card,[\s\S]*?\.settings-route-content \.codex-accounts-panel\s*\{[\s\S]*?border-radius:\s*12px;[\s\S]*?background:\s*linear-gradient\(180deg, rgba\(255, 255, 255, 0\.98\) 0%, rgba\(248, 250, 252, 0\.94\) 100%\);/,
    );
    expect(stylesheet).toMatch(
      /\.settings-route-content \.page-filter-form\s*\{[\s\S]*?border-radius:\s*12px;[\s\S]*?background:\s*rgba\(255, 255, 255, 0\.94\);/,
    );
    expect(stylesheet).not.toContain(".task-filter-form");
    expect(stylesheet).not.toContain(".task-history-view");
    expect(stylesheet).toMatch(
      /\.settings-route-content \.route-data-table th,\s*\.settings-route-content \.route-data-table td\s*\{[\s\S]*?padding:\s*10px 12px;/,
    );
    expect(stylesheet).toMatch(
      /\.settings-route-content \.route-error\s*\{[\s\S]*?border-radius:\s*12px;[\s\S]*?background:\s*rgba\(254, 242, 242, 0\.82\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.settings-route-content \.route-card,[\s\S]*?\.settings-route-content \.codex-accounts-panel \{[\s\S]*?padding:\s*16px;/,
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
    expect(stylesheet).toContain("border-radius: 8px;");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(37, 99, 235, 0.96) 0%, rgba(29, 78, 216, 0.96) 100%);");
    expect(stylesheet).toContain(".modal-footer button[data-variant=\"secondary\"] {");
    const commitOptionsBlock = stylesheet.match(/\.codex-runtime-commit-options\s*\{([\s\S]*?)\n\}/)?.[1] || "";
    expect(commitOptionsBlock).toContain("gap: 4px;");
    expect(commitOptionsBlock).toContain("padding: 6px;");
    const commitOptionBlock = stylesheet.match(/\.codex-runtime-commit-option\s*\{([\s\S]*?)\n\}/)?.[1] || "";
    expect(commitOptionBlock).toContain("grid-template-columns: auto minmax(0, 1fr);");
    expect(commitOptionBlock).toContain("min-height: 64px;");
    expect(commitOptionBlock).not.toContain("border-bottom:");
    expect(stylesheet).toContain(".codex-runtime-commit-option.is-selected {");
    expect(stylesheet).toContain(".codex-runtime-commit-title-row {");
    expect(stylesheet).toContain(".runtime-restart-panel .modal-body {");
    expect(stylesheet).toContain(".runtime-restart-panel .modal-footer {");
    expect(stylesheet).toContain("padding: 34px 16px calc(104px + env(safe-area-inset-bottom));");
    expect(stylesheet).toContain("max-height: min(72vh, calc(var(--mobile-viewport-height, 100dvh) - 128px));");
    expect(stylesheet).toContain("max-height: min(240px, 38vh);");
    expect(stylesheet).toContain(".codex-runtime-commit-option input:checked::after {");
    expect(stylesheet).toContain("align-self: start;");
    expect(stylesheet).toContain("margin-top: 5px;");
    expect(stylesheet).not.toContain("transform: translateY(8px);");
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.modal-backdrop \{[\s\S]*?align-items:\s*center;[\s\S]*?padding:\s*16px;[\s\S]*?\.modal-dialog \{[\s\S]*?width:\s*min\(100%, 420px\);[\s\S]*?border-radius:\s*12px;/,
    );
  });

  it("keeps mobile runtime pages on dynamic viewport CSS with the composer in the workspace grid", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const isolationStylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");

    expect(stylesheet).toContain("--mobile-viewport-height: 100dvh;");
    expect(stylesheet).toContain("--mobile-viewport-offset-top: 0px;");
    expect(stylesheet).toContain("--keyboard-offset: 0px;");
    expect(isolationStylesheet).toMatch(/Mobile runtime keyboard isolation:[\s\S]*?height:\s*max\(0px, calc\(var\(--mobile-viewport-height, 100dvh\) - var\(--mobile-viewport-offset-top, 0px\)\)\);/);
    expect(stylesheet).toContain(".chat-pane:not(.page-mode) {");
    expect(stylesheet).toContain(".chat-pane.page-mode {");
    expect(isolationStylesheet).toContain("bottom: auto !important;");
    expect(isolationStylesheet).toContain(".runtime-workspace-body > .runtime-composer-shell");
    expect(isolationStylesheet).not.toContain(".runtime-composer-spacer");
    expect(isolationStylesheet).not.toContain(".runtime-composer-portal-host");
  });

  it("keeps narrow roots on dynamic viewport units without fixed document locks", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/root.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?html,\s*body,\s*#frontend-root\s*\{[\s\S]*?min-height:\s*100dvh;[\s\S]*?overflow-x:\s*hidden;[\s\S]*?overflow-x:\s*clip;[\s\S]*?overscroll-behavior-x:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?#frontend-root\s*\{[\s\S]*?min-height:\s*100dvh;/,
    );
    expect(stylesheet).not.toMatch(/#frontend-root\s*\{[\s\S]*?position:\s*fixed;/);
    expect(stylesheet).not.toMatch(/html,\s*body,\s*#frontend-root\s*\{[\s\S]*?overflow:\s*hidden;/);
    expect(stylesheet).not.toMatch(/#frontend-root\s*\{[\s\S]*?transform:\s*translate3d\(0, var\(--mobile-viewport-offset-top/);
  });

  it("keeps desktop chrome panels visible and only exposes header drawer buttons at narrow breakpoints", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(/\.nav-toggle,\s*\.panel-toggle,\s*\.mobile-new-chat,\s*\.pane-action\s*\{[\s\S]*?display:\s*none;/);
    expect(stylesheet).toContain(".nav-toggle,");
    expect(stylesheet).toContain(".panel-toggle,");
    expect(stylesheet).toContain(".mobile-new-chat,");
    expect(stylesheet).toContain(".pane-action,");
    expect(stylesheet).not.toContain(".nav-collapse");
    expect(stylesheet).not.toContain(".app-shell.nav-collapsed");
    expect(stylesheet).toContain(".app-shell.info-mode .panel-toggle {");
    expect(stylesheet).toContain(".chat-pane[data-route=\"chatRuntime\"].page-mode .panel-toggle,");
    expect(stylesheet).toContain("@media (max-width: 760px)");
    expect(stylesheet).toContain("width: min(calc(100vw - 24px), 280px);");
    expect(stylesheet).toContain("width: min(calc(100vw - 16px), 360px);");
  });

  it("aligns chatRuntime mobile header actions with the shared chatRuntime workspace control chrome", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-workspace-mobile-action {");
    expect(stylesheet).toContain(".runtime-workspace-mobile-actions {");
    expect(stylesheet).toContain("border-radius: 8px;");
    expect(stylesheet).toContain("font-size: 12px;");
    expect(stylesheet).toContain("font-weight: 700;");
    expect(stylesheet).toContain("background: rgba(248, 252, 255, 0.76);");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(239, 246, 255, 0.98) 0%, rgba(219, 234, 254, 0.92) 100%);");
    expect(stylesheet).toContain("box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.82), 0 10px 20px -18px rgba(37, 99, 235, 0.32);");
    expect(stylesheet).toContain(".runtime-workspace-body {");
    expect(stylesheet).toContain("[data-runtime-view=\"conversation\"] .runtime-workspace-body {");
    expect(stylesheet).toContain("[data-runtime-view=\"chatRuntime\"] .runtime-workspace-body {");
    expect(stylesheet).toContain("grid-template-rows: auto auto minmax(0, 1fr) auto;");
    expect(stylesheet).toContain(".runtime-workspace-body > .runtime-composer-shell {");
    expect(stylesheet).toContain("grid-row: 4;");
    expect(stylesheet).not.toContain("height: var(--runtime-composer-rest-inset, 0px);");
  });

  it("keeps chat and chatRuntime workspace header controls the same size", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).not.toContain("[data-runtime-view=\"conversation\"] .runtime-workspace-actions .runtime-workspace-button {");
    expect(stylesheet).toMatch(
      /\.workspace-header-actions \.runtime-workspace-button\s*\{[\s\S]*?min-height:\s*24px;[\s\S]*?padding:\s*0 8px;[\s\S]*?font-size:\s*10px;/,
    );
  });

  it("keeps chatRuntime-runtime footer chrome aligned with the shared composer surface", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).not.toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-composer-shell\s*\{[\s\S]*?background:\s*linear-gradient\(180deg, rgba\(241, 245, 249,/,
    );
    expect(stylesheet).not.toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-composer-shell\s*\{[\s\S]*?padding:\s*10px 16px 14px;/,
    );
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-composer-note {");
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-composer-form \.runtime-composer-meta,\s*\[data-runtime-composer-view="chatRuntime"\] \.runtime-composer-form \.runtime-composer-meta\s*\{/,
    );
    expect(stylesheet).toContain("min-height: 26px;");
    expect(stylesheet).toContain("[data-runtime-composer-view=\"chatRuntime\"] .runtime-composer-meta[data-runtime-status=\"failed\"]");
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-composer-input {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-composer-submit {");
  });

  it("locks conversation empty states in place without disabling chatRuntime long-press selection", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".conversation-console-panel.is-empty {");
    expect(stylesheet).toContain("overflow: hidden;");
    expect(stylesheet).toContain("[data-runtime-view=\"conversation\"] .runtime-workspace-screen.is-empty {");
    expect(stylesheet).toContain("overscroll-behavior: none;");
    expect(stylesheet).toContain("touch-action: none;");
    expect(stylesheet).toContain("-webkit-overflow-scrolling: auto;");
    expect(stylesheet).not.toMatch(/(^|\n)\.runtime-workspace-screen\.is-empty\s*\{/);
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-workspace-screen.is-empty {");
  });

  it("locks short mobile Chat timelines while keeping only the textarea edge square", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const guard = stylesheet.slice(stylesheet.lastIndexOf("/* Final Chat mobile scroll and composer stability guard */"));

    expect(guard).toContain("/* Final Chat mobile scroll and composer stability guard */");
    expect(guard).toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-workspace-screen\[data-runtime-scrollable="false"\]\s*\{[\s\S]*?overflow-y:\s*hidden;[\s\S]*?overscroll-behavior:\s*none;[\s\S]*?-webkit-overflow-scrolling:\s*auto;[\s\S]*?touch-action:\s*pan-x pinch-zoom;/,
    );
    expect(guard).toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-workspace-screen\[data-runtime-scrollable="true"\]\s*\{[\s\S]*?overflow-y:\s*auto;[\s\S]*?overscroll-behavior:\s*contain;[\s\S]*?-webkit-overflow-scrolling:\s*touch;[\s\S]*?touch-action:\s*pan-y;/,
    );
    expect(guard).toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-composer-form,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-form\s*\{[\s\S]*?border-radius:\s*18px !important;/,
    );
    expect(guard).toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-composer-input,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-input\s*\{[\s\S]*?border-radius:\s*0 !important;/,
    );
  });

  it("keeps the Chat composer inset and compact at the final cascade layer", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const guard = stylesheet.slice(stylesheet.lastIndexOf("/* Final Chat mobile composer compact inset guard */"));

    expect(guard).toContain("/* Final Chat mobile composer compact inset guard */");
    expect(stylesheet.lastIndexOf("/* Final Chat mobile composer compact inset guard */")).toBeGreaterThan(
      stylesheet.lastIndexOf("/* Final Chat mobile scroll and composer stability guard */"),
    );
    expect(guard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-composer-shell,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-shell\s*\{[\s\S]*?padding:\s*6px max\(16px, env\(safe-area-inset-right\)\) calc\(8px \+ env\(safe-area-inset-bottom\)\) max\(16px, env\(safe-area-inset-left\)\);[\s\S]*?box-sizing:\s*border-box;/,
    );
    expect(guard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-composer-form,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-form\s*\{[\s\S]*?min-height:\s*98px;[\s\S]*?padding:\s*10px 14px 8px;[\s\S]*?gap:\s*4px;[\s\S]*?width:\s*100%;[\s\S]*?border-radius:\s*18px !important;/,
    );
    expect(guard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-composer-input,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-input\s*\{[\s\S]*?min-height:\s*36px;[\s\S]*?padding:\s*0 2px;[\s\S]*?line-height:\s*1\.45;/,
    );
    expect(guard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-composer-toolbar,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-toolbar\s*\{[\s\S]*?min-height:\s*38px;[\s\S]*?margin-top:\s*2px;[\s\S]*?gap:\s*6px;/,
    );
    expect(guard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-composer-utility,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-utility,\s*\[data-runtime-view="conversation"\] \.runtime-composer-form \.runtime-composer-upload,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-form \.runtime-composer-upload,\s*\[data-runtime-view="conversation"\] \.runtime-composer-submit,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-submit\s*\{[\s\S]*?width:\s*38px;[\s\S]*?height:\s*38px;/,
    );
  });

  it("keeps the conversation chat viewport in its own scroll container above the composer", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-workspace-panel {");
    expect(stylesheet).toContain(".runtime-workspace-screen {");
    expect(stylesheet).toContain(".runtime-timeline {");
    expect(stylesheet).toContain("height: 100%;");
    expect(stylesheet).toContain("min-height: 0;");
    expect(stylesheet).toContain("min-height: 100%;");
    expect(stylesheet).toContain("align-content: start;");
    expect(stylesheet).toContain("grid-auto-rows: max-content;");
    expect(stylesheet).toContain("padding: var(--chatRuntime-chat-screen-padding-top) calc(var(--chatRuntime-chat-screen-padding-x) + 4px) 26px;");
    expect(stylesheet).toContain("border-radius: 12px;");
  });

  it("keeps mobile Chat thinking and user messages in a top-aligned loading flow", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const isolationStylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");
    const importedCascade = `${stylesheet}\n${isolationStylesheet}`;
    const finalGuard = importedCascade.slice(importedCascade.lastIndexOf("/* Mobile Chat loading flow isolation guard */"));

    expect(finalGuard).toContain("/* Mobile Chat loading flow isolation guard */");
    expect(isolationStylesheet.lastIndexOf("/* Mobile Chat loading flow isolation guard */")).toBeGreaterThan(
      isolationStylesheet.indexOf(".runtime-workspace-screen {"),
    );
    expect(finalGuard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-screen,\s*\.runtime-workspace-body\[data-runtime-view="conversation"\] \.runtime-workspace-screen\s*\{[\s\S]*?display:\s*block;[\s\S]*?align-content:\s*start;/,
    );
    expect(finalGuard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-timeline,\s*\.runtime-workspace-body\[data-runtime-view="conversation"\] \.runtime-timeline\s*\{[\s\S]*?min-height:\s*0;[\s\S]*?height:\s*auto;[\s\S]*?align-content:\s*start;[\s\S]*?justify-content:\s*stretch;/,
    );
    expect(finalGuard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-thinking-shell\.chatRuntime-process-shell,\s*\.runtime-workspace-body\[data-runtime-view="conversation"\] \.runtime-thinking-shell\.chatRuntime-process-shell\s*\{[\s\S]*?justify-self:\s*stretch;[\s\S]*?margin:\s*0 0 14px;/,
    );
    expect(finalGuard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="conversation"\] > \.runtime-composer-shell\s*\{[\s\S]*?align-self:\s*stretch;/,
    );
  });

  it("renders the shared header signal as a soft status dot with visible ripple pulses", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-session-signal {");
    expect(stylesheet).toContain("width: 9px;");
    expect(stylesheet).toContain("height: 9px;");
    expect(stylesheet).toContain("animation: runtime-session-signal-breathe 1.9s ease-in-out infinite;");
    expect(stylesheet).toContain(".runtime-session-signal::before,");
    expect(stylesheet).toContain(".runtime-session-signal::after {");
    expect(stylesheet).toContain("inset: -2px;");
    expect(stylesheet).toContain("border: 1px solid color-mix(in srgb, var(--runtime-session-signal-core) 62%, rgba(255, 255, 255, 0));");
    expect(stylesheet).toContain("animation: runtime-session-signal-ripple 1.9s cubic-bezier(0.22, 1, 0.36, 1) infinite;");
    expect(stylesheet).toContain("animation-delay: 0.42s;");
    expect(stylesheet).toContain("@keyframes runtime-session-signal-breathe {");
    expect(stylesheet).toContain("@keyframes runtime-session-signal-ripple {");
    expect(stylesheet).toContain("transform: scale(1.58);");
  });

  it("keeps the workspace header status as a borderless signal-only control", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".workspace-header-status {");
    expect(stylesheet).toContain("display: inline-flex;");
    expect(stylesheet).toContain("align-items: center;");
    expect(stylesheet).toContain("justify-content: center;");
    expect(stylesheet).toContain("min-width: 14px;");
    expect(stylesheet).toContain("width: 16px;");
    expect(stylesheet).toContain("height: 16px;");
    expect(stylesheet).toContain("padding: 0;");
    expect(stylesheet).toContain("border: 0;");
    expect(stylesheet).toContain("background: transparent !important;");
    expect(stylesheet).toContain("box-shadow: none;");
    expect(stylesheet).toContain("min-height: auto;");
    expect(stylesheet).toContain(".workspace-header-status .runtime-session-signal {");
    expect(stylesheet).toContain("display: block;");
  });

  it("drops blur-heavy mobile chrome so runtime surfaces stay responsive on phones", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("@media (max-width: 1100px)");
    expect(stylesheet).toContain("body::before,");
    expect(stylesheet).toContain("body::after {");
    expect(stylesheet).toContain("display: none;");
    expect(stylesheet).toContain(".runtime-workspace-session-pane-backdrop {");
    expect(stylesheet).toContain("backdrop-filter: none;");
    expect(stylesheet).toContain(".runtime-workspace-session-pane-shell {");
    expect(stylesheet).toContain("[data-runtime-view=\"conversation\"] .runtime-workspace-body {");
    expect(stylesheet).toContain("[data-runtime-view=\"chatRuntime\"] {");
    expect(stylesheet).toContain(".mobile-backdrop {");
  });

  it("keeps runtime workspaces on one desktop content column after the primary nav owns the session list", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\],\s*\[data-runtime-view="chatRuntime"\]\s*\{[\s\S]*?grid-template-columns:\s*minmax\(0, 1fr\);/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-workspace-session-pane\.is-navigation-owned\s*\{[\s\S]*?display:\s*none;/,
    );
    expect(stylesheet).not.toMatch(
      /\[data-runtime-view="chatRuntime"\]\s*\{\s*grid-template-columns:\s*minmax\(280px, 320px\) minmax\(0, 1fr\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\],\s*\[data-runtime-view="chatRuntime"\]\s*\{[\s\S]*?grid-template-columns:\s*1fr;[\s\S]*?padding:\s*0;[\s\S]*?gap:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body\s*\{[\s\S]*?grid-template-rows:\s*auto auto minmax\(0, 1fr\) auto;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-mobile-header,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body > \.runtime-workspace-mobile-header\s*\{[\s\S]*?grid-row:\s*1;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-head,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body > \.runtime-workspace-head\s*\{[\s\S]*?grid-row:\s*2;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-panel,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body > \.runtime-workspace-panel\s*\{[\s\S]*?grid-row:\s*3;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\.runtime-workspace-body > \.runtime-composer-shell\s*\{[\s\S]*?grid-row:\s*4;/,
    );
    expect(stylesheet).not.toMatch(/\.runtime-composer-spacer\s*\{[\s\S]*?height:\s*var\(--runtime-composer-rest-inset/);
  });

  it("keeps mobile runtime headers and composer controls on a compact single row instead of wrapping into oversized stacks", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-workspace-mobile-header {");
    expect(stylesheet).toContain(".runtime-composer-tools {");
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-mobile-header\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-columns:\s*auto minmax\(0, 1fr\) auto;[\s\S]*?min-height:\s*48px;[\s\S]*?padding:\s*6px 10px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-panel\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?bottom:\s*calc\(env\(safe-area-inset-bottom\) \+ 118px\);[\s\S]*?max-height:\s*min\(52vh, calc\(100dvh - 176px\)\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-command-list\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?bottom:\s*calc\(env\(safe-area-inset-bottom\) \+ 128px\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-panel\[data-runtime-config-surface="chatRuntime"\]\s*\{[\s\S]*?bottom:\s*calc\(env\(safe-area-inset-bottom\) \+ 120px\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-submit\s*\{[\s\S]*?width:\s*36px;[\s\S]*?min-width:\s*36px;[\s\S]*?padding:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-tools\s*\{[\s\S]*?flex-wrap:\s*nowrap;[\s\S]*?gap:\s*10px;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-composer-form \.runtime-composer-tools,\s*\[data-runtime-composer-view="chatRuntime"\] \.runtime-composer-form \.runtime-composer-tools\s*\{[\s\S]*?justify-content:\s*flex-end;/,
    );
  });

  it("uses one polished mobile composer tray across chat and chatRuntime runtime pages", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-shell\s*\{[\s\S]*?border-top:\s*0;[\s\S]*?padding:\s*10px 14px calc\(12px \+ env\(safe-area-inset-bottom\)\);[\s\S]*?background:\s*linear-gradient\(180deg, rgba\(255, 255, 255, 0\) 0%, rgba\(250, 252, 255, 0\.94\) 48%, rgba\(250, 252, 255, 0\.98\) 100%\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-form\s*\{[^}]*?width:\s*100%;[^}]*?align-content:\s*start;[^}]*?padding:\s*12px 14px 10px;[^}]*?border-radius:\s*26px;[^}]*?background:\s*#fff;[^}]*?box-shadow:\s*0 12px 30px -24px rgba\(15, 23, 42, 0\.28\), 0 2px 8px rgba\(60, 64, 67, 0\.10\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-body\s*\{[^}]*?align-self:\s*start;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-input\s*\{[\s\S]*?width:\s*100%;[\s\S]*?height:\s*72px;[\s\S]*?min-height:\s*72px;[\s\S]*?resize:\s*none;[\s\S]*?background:\s*transparent;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-submit\s*\{[\s\S]*?background:\s*#0f172a;[\s\S]*?color:\s*#fff;[\s\S]*?box-shadow:\s*none;/,
    );
  });

  it("collapses phone runtime chrome into one top workbar and moves details into the title control", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-mobile-header\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-columns:\s*auto minmax\(0, 1fr\) auto;[\s\S]*?padding:\s*12px 14px 10px;/,
    );
    expect(stylesheet).toContain(".runtime-workspace-mobile-title {");
    expect(stylesheet).toContain(".runtime-workspace-mobile-title-copy {");
    expect(stylesheet).toContain(".runtime-workspace-mobile-title-text {");
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-head\.is-mobile-collapsed\s*\{[\s\S]*?display:\s*none;/,
    );
  });

  it("keeps runtime session cards compact with title-only rows and busy loading", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-session-card {");
    expect(stylesheet).toContain("border-radius: 18px;");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(255, 255, 255, 0.99) 0%, rgba(250, 252, 255, 0.98) 100%);");
    expect(stylesheet).toContain(".runtime-session-delete {");
    expect(stylesheet).toContain("min-height: 30px;");
    expect(stylesheet).toContain(".runtime-session-loading {");
    expect(stylesheet).toContain("animation: runtime-session-loading-spin 0.82s linear infinite;");
    expect(stylesheet).toContain("@keyframes runtime-session-loading-spin {");
    expect(stylesheet).toContain(".runtime-workspace-session-pane-action-icon {");
    expect(stylesheet).toContain(".runtime-session-title-copy {");
    expect(stylesheet).toContain("flex: 1 1 auto;");
    expect(stylesheet).toContain(".nav-session-rail .runtime-session-summary-row {");
    expect(stylesheet).toContain("display: none;");
    expect(stylesheet).toContain(".nav-session-rail .runtime-session-context {");
    expect(stylesheet).toContain(".conversation-inspector {");
    expect(stylesheet).toContain("border-radius: 12px;");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(255, 255, 255, 0.96) 0%, rgba(241, 248, 255, 0.94) 100%);");
    expect(stylesheet).toContain(".conversation-check-item {");
    expect(stylesheet).toContain("padding: 8px 10px;");
  });

  it("keeps session drawer actions compact and aligned with the smaller mobile control size", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-workspace-session-pane-head {");
    expect(stylesheet).toContain("justify-content: space-between;");
    expect(stylesheet).toContain("flex-wrap: nowrap;");
    expect(stylesheet).toContain(".runtime-workspace-session-pane-copy {");
    expect(stylesheet).toContain("display: flex;");
    expect(stylesheet).toContain("flex-direction: column;");
    expect(stylesheet).toContain("align-items: flex-start;");
    expect(stylesheet).toContain(".runtime-workspace-session-pane-actions {");
    expect(stylesheet).toContain("margin-left: auto;");
    expect(stylesheet).toContain(".runtime-workspace-session-pane-action {");
    expect(stylesheet).toContain("min-height: 32px;");
    expect(stylesheet).toContain("border-radius: 8px;");
    expect(stylesheet).toContain("padding: 0 10px;");
    expect(stylesheet).toContain("font-size: 11px;");
    expect(stylesheet).toContain("width: min(calc(100vw - 12px), 360px);");
    expect(stylesheet).toContain(".runtime-workspace-session-pane-copy strong {");
    expect(stylesheet).toContain("font-size: 15px;");
    expect(stylesheet).toContain(".runtime-workspace-session-pane-copy span {");
    expect(stylesheet).toContain("font-size: 9.5px;");
  });

  it("renders composer toolbar controls as compact square icon buttons", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-composer-utility,");
    expect(stylesheet).toContain(".runtime-composer-submit,");
    expect(stylesheet).toContain(".runtime-composer-upload {");
    expect(stylesheet).toContain("[data-runtime-view=\"conversation\"] .runtime-composer-form .runtime-composer-upload,");
    expect(stylesheet).toContain("[data-runtime-composer-view=\"chatRuntime\"] .runtime-composer-form .runtime-composer-upload {");
    expect(stylesheet).toContain("width: 38px;");
    expect(stylesheet).toContain("height: 38px;");
    expect(stylesheet).toContain("align-items: center;");
    expect(stylesheet).toContain("justify-content: center;");
    expect(stylesheet).toContain("border-radius: 8px;");
    expect(stylesheet).toContain(".runtime-composer-upload-label {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-session-topline .task-summary-status");
  });

  it("renders the navigation-owned session rail as a quiet title-only chat sidebar with a compact actions menu", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.nav-session-rail\s*\{[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-card\s*\{[\s\S]*?grid-template-columns:\s*minmax\(0, 1fr\) 32px;[\s\S]*?padding:\s*0 4px 0 10px;[\s\S]*?border-radius:\s*9px;[\s\S]*?border-color:\s*transparent;[\s\S]*?background:\s*transparent;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-card\.is-active\s*\{[\s\S]*?border-color:\s*transparent;[\s\S]*?background:\s*rgba\(37, 99, 235, 0\.08\);[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-title\s*\{[\s\S]*?font-size:\s*14px;[\s\S]*?letter-spacing:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-actions\s*\{[\s\S]*?min-width:\s*28px;[\s\S]*?gap:\s*2px;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-action\s*\{[\s\S]*?min-width:\s*28px;[\s\S]*?min-height:\s*28px;[\s\S]*?border-radius:\s*8px;[\s\S]*?opacity:\s*1;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-session-action-menu\s*\{[\s\S]*?position:\s*absolute;[\s\S]*?right:\s*0;[\s\S]*?min-width:\s*168px;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-action-menu\s*\{[\s\S]*?top:\s*calc\(100% \+ 6px\);[\s\S]*?min-width:\s*152px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(min-width: 761px\) \{[\s\S]*?\.primary-nav\.has-session-rail > \.menu\s*\{[\s\S]*?flex:\s*0 0 clamp\(260px, 34vh, 312px\);[\s\S]*?overflow-y:\s*auto;/,
    );
  });

  it("applies the approved light-tech shell to the shared sidebar instead of the old gray chrome", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("/* Workbench shell system */");
    expect(stylesheet).toMatch(
      /\.app-shell\s*\{[\s\S]*?--shell-nav-width:\s*264px;[\s\S]*?--shell-bg:\s*#f7fbff;[\s\S]*?--shell-accent:\s*#2563eb;/,
    );
    expect(stylesheet).toMatch(
      /\.primary-nav\s*\{[\s\S]*?background:\s*color-mix\(in oklab, #ffffff, #f7fbff 18%\);[\s\S]*?box-shadow:\s*inset -1px 0 0 #edf4fb;[\s\S]*?border-right:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /\.primary-nav \.brand\s*\{[\s\S]*?min-height:\s*74px;[\s\S]*?padding:\s*4px 4px 16px;/,
    );
    expect(stylesheet).toMatch(
      /\.primary-nav \.brand-copy strong\s*\{[\s\S]*?font-size:\s*18px;[\s\S]*?font-weight:\s*800;/,
    );
    expect(stylesheet).toMatch(
      /\.primary-nav \.menu-item\.active\s*\{[\s\S]*?background:\s*#eaf2ff;[\s\S]*?color:\s*#2563eb;/,
    );
    expect(stylesheet).toMatch(
      /\.primary-nav\.has-session-rail > \.menu\s*\{[\s\S]*?flex:\s*0 0 auto;[\s\S]*?overflow:\s*visible;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail\s*\{[\s\S]*?border-top:\s*1px solid #edf4fb;[\s\S]*?background:\s*transparent;/,
    );
    expect(stylesheet).toMatch(
      /\.settings-route-tab\.is-active\s*\{[\s\S]*?background:\s*#eaf2ff;[\s\S]*?color:\s*#2563eb;[\s\S]*?box-shadow:\s*inset 3px 0 0 #2563eb;/,
    );
    expect(stylesheet).toMatch(
      /\.route-head\.workbench-title-head\.is-compact,\s*\.runtime-workspace-head\s*\{[\s\S]*?border-bottom:\s*1px solid #edf4fb;[\s\S]*?background:\s*rgba\(255, 255, 255, 0\.78\);/,
    );
  });

  it("does not define a desktop collapsed sidebar stage in the workbench shell system", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const shellLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Workbench shell system */"));

    expect(shellLayer).not.toContain(".app-shell.nav-collapsed");
    expect(shellLayer).not.toContain("--shell-nav-collapsed-width");
  });

  it("keeps the legacy public stylesheet from reintroducing the removed collapsed sidebar stage", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const legacyCoreStylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-core.css"), "utf8");
    const legacyRuntimeStylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-runtime.css"), "utf8");

    expect(legacyCoreStylesheet).not.toContain(".app-shell.nav-collapsed");
    expect(legacyCoreStylesheet).not.toContain(".nav-collapse");
    expect(legacyCoreStylesheet).not.toContain("--nav-collapsed-width");
    expect(legacyRuntimeStylesheet).not.toContain(".nav-collapse");
  });

  it("keeps long navigation session titles from changing sidebar geometry", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.nav-session-rail\s*\{[\s\S]*?width:\s*100%;[\s\S]*?min-width:\s*0;[\s\S]*?overflow:\s*hidden;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail-body\s*\{[\s\S]*?width:\s*100%;[\s\S]*?min-width:\s*0;[\s\S]*?overflow:\s*hidden;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-list\s*\{[\s\S]*?width:\s*100%;[\s\S]*?min-width:\s*0;[\s\S]*?overflow-x:\s*hidden;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-group,\s*\.nav-session-rail \.runtime-session-group-items\s*\{[\s\S]*?width:\s*100%;[\s\S]*?min-width:\s*0;[\s\S]*?max-width:\s*100%;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-card\s*\{[\s\S]*?width:\s*100%;[\s\S]*?min-width:\s*0;[\s\S]*?max-width:\s*100%;[\s\S]*?box-sizing:\s*border-box;/,
    );
  });

  it("keeps the navigation session rail steady when new sessions are inserted", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.nav-session-rail\s*\{[\s\S]*?grid-template-rows:\s*minmax\(38px, auto\) minmax\(0, 1fr\);[\s\S]*?contain:\s*layout paint;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail-head\s*\{[\s\S]*?min-height:\s*38px;[\s\S]*?flex:\s*0 0 auto;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail-body\s*\{[\s\S]*?contain:\s*layout;[\s\S]*?overflow-anchor:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-list\s*\{[\s\S]*?scrollbar-gutter:\s*stable both-edges;[\s\S]*?overflow-anchor:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-card\s*\{[\s\S]*?overflow-anchor:\s*none;/,
    );
  });

  it("prevents scrollbar threshold changes from resizing the navigation session list", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.primary-nav\.has-session-rail > \.brand\s*\{[\s\S]*?flex:\s*0 0 40px;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-list\s*\{[\s\S]*?flex:\s*1 1 auto;[\s\S]*?overflow-y:\s*scroll;[\s\S]*?scrollbar-gutter:\s*stable both-edges;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-group\.menu-group\s*\{[\s\S]*?flex:\s*0 0 auto;/,
    );
    expect(stylesheet).toMatch(
      /\.nav-session-rail \.runtime-session-group-items\s*\{[\s\S]*?display:\s*grid;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.primary-nav \.nav-session-rail \.runtime-session-list\s*\{[\s\S]*?height:\s*100%;[\s\S]*?flex:\s*1 1 auto;[\s\S]*?overflow-y:\s*scroll;[\s\S]*?overflow-x:\s*hidden;[\s\S]*?scrollbar-gutter:\s*stable both-edges;/,
    );
  });

  it("prevents the navigation-owned session list from inheriting menu grid sizing", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\.primary-nav \.nav-session-rail \.runtime-session-list\.menu\s*\{[\s\S]*?display:\s*flex;[\s\S]*?flex-direction:\s*column;[\s\S]*?align-items:\s*stretch;[\s\S]*?margin:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /\.primary-nav \.nav-session-rail \.runtime-session-group-items\s*\{[\s\S]*?align-content:\s*start;[\s\S]*?grid-auto-rows:\s*max-content;/,
    );
    expect(stylesheet).toMatch(
      /\.primary-nav \.nav-session-rail \.runtime-session-card\s*\{[\s\S]*?align-self:\s*start;[\s\S]*?min-height:\s*40px;[\s\S]*?height:\s*auto;[\s\S]*?max-height:\s*44px;/,
    );
    expect(stylesheet).toMatch(
      /\.primary-nav \.nav-session-rail \.runtime-session-select\s*\{[\s\S]*?min-height:\s*40px;[\s\S]*?height:\s*40px;[\s\S]*?display:\s*flex;[\s\S]*?align-items:\s*center;[\s\S]*?padding:\s*0;/,
    );
  });

  it("gives the mobile primary nav a purpose-built clean drawer surface", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.primary-nav\s*\{[\s\S]*?width:\s*min\(86vw, 336px\);[\s\S]*?padding:\s*18px 18px calc\(16px \+ env\(safe-area-inset-bottom\)\);[\s\S]*?background:\s*#fff;[\s\S]*?box-shadow:\s*24px 0 48px -34px rgba\(15, 23, 42, 0\.46\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.primary-nav \.brand\s*\{[\s\S]*?min-height:\s*44px;[\s\S]*?padding:\s*0 2px 8px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.primary-nav \.menu-item\s*\{[\s\S]*?min-height:\s*44px;[\s\S]*?border-radius:\s*12px;[\s\S]*?background:\s*transparent;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.primary-nav \.nav-session-rail\s*\{[\s\S]*?max-height:\s*none;[\s\S]*?padding:\s*8px 0 0;[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.primary-nav \.nav-session-rail-copy span\s*\{[^}]*?display:\s*none;/,
    );
  });

  it("keeps the mobile primary nav drawer inside the dynamic viewport", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const mobileDrawerLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Final mobile primary nav viewport guard */"));

    expect(mobileDrawerLayer).toContain("/* Final mobile primary nav viewport guard */");
    expect(mobileDrawerLayer).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.workbench-mobile-overlay-portal\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?top:\s*var\(--mobile-viewport-offset-top, 0px\);[\s\S]*?bottom:\s*auto;[\s\S]*?height:\s*max\(0px, calc\(var\(--mobile-viewport-height, 100dvh\) - var\(--mobile-viewport-offset-top, 0px\)\)\);[\s\S]*?max-height:\s*max\(0px, calc\(var\(--mobile-viewport-height, 100dvh\) - var\(--mobile-viewport-offset-top, 0px\)\)\);[\s\S]*?overflow:\s*hidden;/,
    );
    expect(mobileDrawerLayer).toMatch(
      /\.workbench-mobile-overlay-portal \.primary-nav,\s*\.app-shell\.nav-open \.primary-nav,\s*\.primary-nav\s*\{[\s\S]*?position:\s*absolute;[\s\S]*?top:\s*0;[\s\S]*?bottom:\s*auto;[\s\S]*?height:\s*max\(0px, calc\(var\(--mobile-viewport-height, 100dvh\) - var\(--mobile-viewport-offset-top, 0px\)\)\);[\s\S]*?max-height:\s*max\(0px, calc\(var\(--mobile-viewport-height, 100dvh\) - var\(--mobile-viewport-offset-top, 0px\)\)\);[\s\S]*?min-height:\s*0;[\s\S]*?overflow:\s*hidden;/,
    );
    expect(mobileDrawerLayer).toMatch(
      /\.workbench-mobile-overlay-portal \.mobile-backdrop,\s*\.mobile-backdrop\s*\{[\s\S]*?position:\s*absolute;[\s\S]*?top:\s*0;[\s\S]*?height:\s*100%;[\s\S]*?max-height:\s*100%;/,
    );
    expect(mobileDrawerLayer).toMatch(
      /\.primary-nav \.nav-session-rail,\s*\.primary-nav \.nav-utility,\s*\.primary-nav \.nav-locale\s*\{[\s\S]*?min-height:\s*0;/,
    );
  });

  it("renders attachment previews and message images with their original aspect ratios instead of square crops", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-composer-attachment-preview img {");
    expect(stylesheet).toContain("max-height: 72px;");
    expect(stylesheet).toContain("object-fit: contain;");
    expect(stylesheet).toContain(".message-image-card img {");
    expect(stylesheet).toContain("height: auto;");
    expect(stylesheet).toContain("max-height: min(420px, 70vh);");
    expect(stylesheet).not.toContain("aspect-ratio: 1.2 / 1;");
  });

  it("uses a compact single-line chat workspace header instead of the old stacked copy block", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).not.toContain('[data-runtime-header-kind="chatRuntime"]');
    expect(stylesheet).toContain(".workbench-title-head {");
    expect(stylesheet).toMatch(/\.runtime-workspace-row\.is-compact\s*\{[\s\S]*?flex-wrap:\s*nowrap;/);
    expect(stylesheet).toMatch(/\.runtime-workspace-head\.is-compact\s*\{[\s\S]*?display:\s*block;/);
    expect(stylesheet).toContain(".runtime-workspace-copy.is-compact {");
    expect(stylesheet).toContain(".runtime-workspace-copy.is-compact h4 {");
    expect(stylesheet).toContain(".runtime-workspace-row.is-compact .runtime-workspace-actions {");
    expect(stylesheet).toContain("align-items: center;");
    expect(stylesheet).toContain("font-size: 18px;");
    expect(stylesheet).toContain("text-overflow: ellipsis;");
  });

  it("aligns Settings route headings with the Chat and ChatRuntime workspace header chrome", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".route-head.workbench-title-head.is-compact {");
    expect(stylesheet).toContain(".route-title-leading.workbench-title-leading {");
    expect(stylesheet).toContain(".route-title-marker {");
    expect(stylesheet).toMatch(
      /\.route-view\[data-route-family="settings"\]\.workbench-route-frame\s*\{[\s\S]*?height:\s*100%;[\s\S]*?display:\s*flex;[\s\S]*?flex-direction:\s*column;[\s\S]*?padding:\s*0;[\s\S]*?border:\s*1px solid rgba\(203, 213, 225, 0\.88\);[\s\S]*?border-radius:\s*14px;/,
    );
    expect(stylesheet).toMatch(
      /\.route-view\[data-route-family="settings"\] \.route-body\s*\{[\s\S]*?flex:\s*1 1 auto;[\s\S]*?min-height:\s*0;[\s\S]*?padding:\s*14px 12px;[\s\S]*?overflow-y:\s*auto;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-workspace-head\.is-compact,\s*\.route-head\.workbench-title-head\.is-compact\s*\{[\s\S]*?min-height:\s*42px;[\s\S]*?padding:\s*6px 12px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.route-view\[data-route-family="settings"\] \.route-mobile-head\s*\{[\s\S]*?grid-template-columns:\s*36px minmax\(0, 1fr\) 36px;[\s\S]*?min-height:\s*48px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.route-view\[data-route-family="settings"\] \.route-mobile-head \.nav-toggle\s*\{[\s\S]*?grid-area:\s*auto;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.route-mobile-head \.conversation-mobile-action,\s*\.runtime-workspace-mobile-header \.runtime-workspace-mobile-action\s*\{[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-mobile-header \.runtime-workspace-mobile-action\.is-quiet,\s*\.runtime-workspace-mobile-header \.runtime-workspace-mobile-action\.is-primary,\s*\.route-mobile-head \.conversation-mobile-action\.is-quiet\s*\{[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.route-view\[data-route-family="settings"\] \.route-head\.workbench-title-head\.is-compact\s*\{[\s\S]*?display:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.route-head\.workbench-title-head\.is-compact\s*\{[\s\S]*?border-bottom:\s*1px solid rgba\(226, 232, 240, 0\.9\);[\s\S]*?background:\s*rgba\(255, 255, 255, 0\.98\);/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-workspace-copy\.is-compact h4,\s*\.route-view h3\s*\{[\s\S]*?font-size:\s*15px;[\s\S]*?line-height:\s*1\.15;/,
    );
    expect(stylesheet).toMatch(
      /\.route-head\.workbench-title-head\.is-compact #routeSubtitle\s*\{[\s\S]*?display:\s*none;/,
    );
    const routeTitleBlocks = Array.from(stylesheet.matchAll(/\.route-view h3\s*\{([\s\S]*?)\n\}/g));
    const routeTitleBlock = routeTitleBlocks[routeTitleBlocks.length - 1]?.[1] || "";
    expect(routeTitleBlock).not.toContain("letter-spacing: -0.04em;");
  });

  it("renders shared details panels as dense summary grids instead of loose stacked metadata", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".workspace-details-content {");
    expect(stylesheet).toContain(".workspace-details-summary {");
    expect(stylesheet).toContain("grid-template-columns: repeat(auto-fit, minmax(188px, 1fr));");
    expect(stylesheet).toContain(".workspace-details-panel-head {");
    expect(stylesheet).toContain(".workspace-details-close {");
    expect(stylesheet).toContain(".workspace-details-panel .route-field-row {");
    expect(stylesheet).toContain("grid-template-columns: minmax(62px, 84px) minmax(0, 1fr);");
    expect(stylesheet).toContain("background: rgba(255, 255, 255, 0.72);");
    expect(stylesheet).toContain(".workspace-details-panel .route-field-row > span:first-child {");
    expect(stylesheet).toContain("text-transform: uppercase;");
    expect(stylesheet).toContain(".workspace-details-panel .route-field-value.is-multiline {");
    expect(stylesheet).toContain("padding: 6px 8px;");
    expect(stylesheet).toContain(".workspace-details-panel .route-field-copy {");
    expect(stylesheet).toContain("width: 22px;");
  });

  it("renders details as a top-layer floating panel with its own scroll container and dismiss backdrop", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".workspace-details-layer {");
    expect(stylesheet).toContain("position: fixed;");
    expect(stylesheet).toContain("z-index: 130;");
    expect(stylesheet).toContain(".workspace-details-backdrop {");
    expect(stylesheet).toContain(".workspace-details-panel {");
    expect(stylesheet).toContain("border-radius: 12px;");
    expect(stylesheet).toContain("background: linear-gradient(180deg, rgba(255, 255, 255, 0.98) 0%, rgba(248, 250, 252, 0.98) 100%);");
    expect(stylesheet).toContain("max-height: min(64vh, calc(100dvh - 168px));");
    expect(stylesheet).toContain("overflow: auto;");
    expect(stylesheet).toContain("overscroll-behavior: contain;");
  });

  it("keeps mobile empty-state headers on a single chatRuntime-style row while preserving visible title space", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-workspace-head.is-mobile-empty {");
    expect(stylesheet).toContain(".runtime-workspace-row.is-mobile-empty {");
    expect(stylesheet).toContain("display: grid;");
    expect(stylesheet).toContain("grid-template-columns: minmax(0, 1fr) auto;");
    expect(stylesheet).toContain(".runtime-workspace-copy.is-mobile-empty {");
    expect(stylesheet).toContain("min-width: 0;");
    expect(stylesheet).toContain(".runtime-workspace-actions.is-mobile-empty {");
    expect(stylesheet).toContain("justify-content: flex-end;");
  });

  it("keeps mobile conversation empty states aligned with the original chat baseline", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.conversation-empty-state\s*\{[\s\S]*?margin-left:\s*0;[\s\S]*?padding:\s*24px 0 32px;[\s\S]*?align-self:\s*start;[\s\S]*?align-content:\s*start;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-composer-shell,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-shell\s*\{[\s\S]*?border-top:\s*0;[\s\S]*?background:\s*linear-gradient\(180deg, rgba\(255, 255, 255, 0\) 0%, rgba\(250, 252, 255, 0\.94\) 48%, rgba\(250, 252, 255, 0\.98\) 100%\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-input\s*\{[\s\S]*?width:\s*100%;[\s\S]*?height:\s*72px;[\s\S]*?min-height:\s*72px;[\s\S]*?resize:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-submit\s*\{[\s\S]*?background:\s*#0f172a;[\s\S]*?color:\s*#fff;/,
    );
  });

  it("keeps the full page state layer for settings subpages and chatRuntime live overlays", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("/* UI UX Pro Max full page states layer */");
    expect(stylesheet).toContain(".settings-route-content[data-settings-route-content=\"runtime\"] .settings-composite-section");
    expect(stylesheet).toContain(".settings-composite-section[data-settings-section=\"workspaces\"]");
    expect(stylesheet).toContain(".memory-tabs");
    expect(stylesheet).toContain(".runtime-composer-command-list");
    expect(stylesheet).toContain(".runtime-composer-panel[data-runtime-config-surface=\"chatRuntime\"]");
    expect(stylesheet).toContain(".chatRuntime-skill-section .conversation-check-item:has(input:checked)");
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

  it("keeps the narrow-screen legacy composer on the viewport bottom without keyboard offset layout inflation", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("@media (max-width: 760px)");
    expect(stylesheet).not.toContain("bottom: var(--keyboard-composer-offset, var(--keyboard-offset));");
    expect(stylesheet).not.toContain("calc(var(--mobile-viewport-height, 100dvh) + var(--keyboard-offset");
    expect(stylesheet).toContain("padding: 10px 12px calc(14px + env(safe-area-inset-bottom));");
  });

  it("keeps mobile composer inputs at the iOS-safe 16px text size", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.composer textarea\s*\{[^}]*font-size:\s*16px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-input\s*\{[^}]*font-size:\s*16px;/,
    );
  });

  it("keeps page-mode workbench shells stretched to the mobile viewport so chatRuntime messages can scroll", () => {
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
      /\/\* Legacy main-repo layout baseline \*\/[\s\S]*?@media \(max-width: 1100px\) \{[\s\S]*?\.nav-toggle,\s*\.panel-toggle,\s*\.mobile-new-chat,\s*\.pane-action\s*\{[\s\S]*?display:\s*inline-flex;/,
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

  it("keeps legacy chatRuntime skin scoped to chatRuntime toggles instead of conversation runtime actions", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-runtime.css"), "utf8");

    expect(stylesheet).toContain(".app-shell.info-mode .panel-toggle {");
    expect(stylesheet).toContain(".chat-pane[data-route=\"chatRuntime\"].page-mode .panel-toggle,");
    expect(stylesheet).not.toContain(".app-shell[data-workbench-route=\"chat\"] .conversation-workspace .panel-toggle,");
  });

  it("keeps runtime mobile shells and long chatRuntime content from creating page-level horizontal overflow", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const rootStylesheet = readFileSync(resolve(currentDirectory, "../../styles/root.css"), "utf8");
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const horizontalGuard = stylesheet.slice(stylesheet.lastIndexOf("/* Final mobile horizontal overflow stability guard */"));

    expect(rootStylesheet).toMatch(/html,\s*body,\s*#frontend-root\s*\{[\s\S]*?overflow-x:\s*hidden;[\s\S]*?overflow-x:\s*clip;[\s\S]*?overscroll-behavior-x:\s*none;/);
    expect(horizontalGuard).toContain("/* Final mobile horizontal overflow stability guard */");
    expect(horizontalGuard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\],\s*\.runtime-workspace-body\[data-runtime-view="conversation"\],\s*\[data-runtime-view="conversation"\] \.runtime-workspace,\s*\[data-runtime-view="conversation"\] \.runtime-workspace-body,\s*\[data-runtime-view="conversation"\] \.runtime-workspace-panel,\s*\[data-runtime-view="conversation"\] \.runtime-workspace-screen,\s*\[data-runtime-view="conversation"\] \.runtime-timeline\s*\{[\s\S]*?min-width:\s*0;[\s\S]*?max-width:\s*100%;[\s\S]*?overflow-x:\s*hidden;/,
    );
    expect(horizontalGuard).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-composer-shell,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-shell,\s*\[data-runtime-view="conversation"\] \.runtime-composer-form,\s*\[data-runtime-composer-view="conversation"\] \.runtime-composer-form\s*\{[\s\S]*?min-width:\s*0;[\s\S]*?max-width:\s*100%;[\s\S]*?overflow-x:\s*hidden;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.message-markdown-shell,[\s\S]*?\[data-runtime-view="chatRuntime"\] \.message-markdown-rendered\s*\{[\s\S]*?min-width:\s*0;[\s\S]*?max-width:\s*100%;[\s\S]*?box-sizing:\s*border-box;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.chatRuntime-turn-prompt \.chatRuntime-log-main,\s*\[data-runtime-view="chatRuntime"\] \.chatRuntime-turn-prompt \.chatRuntime-log-main\s*\{[\s\S]*?max-width:\s*min\(var\(--user-message-max-width, 80%\), calc\(100% - 24px\)\);/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.chatRuntime-log-text,\s*\[data-runtime-view="chatRuntime"\] \.chatRuntime-log-text,[\s\S]*?\.chat-md-inline-code\s*\{[\s\S]*?overflow-wrap:\s*anywhere;[\s\S]*?word-break:\s*break-word;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.chat-md-table-wrap,\s*\[data-runtime-view="chatRuntime"\] \.chat-md-table-wrap\s*\{[\s\S]*?width:\s*100%;[\s\S]*?max-width:\s*100%;[\s\S]*?overflow-x:\s*auto;[\s\S]*?-webkit-overflow-scrolling:\s*touch;/,
    );
    expect(stylesheet).toMatch(
      /\.chat-md-table th,\s*\.chat-md-table td\s*\{[\s\S]*?overflow-wrap:\s*anywhere;[\s\S]*?word-break:\s*normal;/,
    );
    expect(stylesheet).toMatch(
      /\.chat-md-table td :is\(a, code\),\s*\.chat-md-table th :is\(a, code\)\s*\{[\s\S]*?white-space:\s*nowrap;[\s\S]*?overflow-wrap:\s*normal;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-mobile-header\s*\{[\s\S]*?width:\s*100%;[\s\S]*?min-width:\s*0;[\s\S]*?max-width:\s*100%;[\s\S]*?box-sizing:\s*border-box;/,
    );
  });

  it("styles the chatRuntime workspace like a restrained command center while reusing shared session cards", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-session-card {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-session-card {");
    expect(stylesheet).toContain("[data-runtime-view=\"chatRuntime\"] .runtime-workspace-screen {");
    expect(stylesheet).toContain(".runtime-workspace-body {");
    expect(stylesheet).toContain("linear-gradient(180deg, rgba(255, 255, 255, 0.99) 0%, rgba(243, 247, 251, 0.98) 100%)");
    expect(stylesheet).toContain("box-shadow: 0 34px 70px -54px rgba(15, 23, 42, 0.18);");
  });

  it("lays out the chatRuntime composer as a single input surface with a footer tool row", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-composer-form {");
    expect(stylesheet).toContain("padding: 16px 18px 14px;");
    expect(stylesheet).toContain(".runtime-composer-input {");
    expect(stylesheet).toContain("min-height: 78px;");
    expect(stylesheet).toContain("padding: 10px 8px 8px;");
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-composer-form \.runtime-composer-tools,\s*\[data-runtime-composer-view="chatRuntime"\] \.runtime-composer-form \.runtime-composer-tools\s*\{/,
    );
    expect(stylesheet).toContain("justify-content: flex-end;");
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-composer-form \.runtime-composer-meta,\s*\[data-runtime-composer-view="chatRuntime"\] \.runtime-composer-form \.runtime-composer-meta\s*\{/,
    );
    expect(stylesheet).toContain("border: 0;");
    expect(stylesheet).toContain("background: rgba(238, 244, 249, 0.96);");
    expect(stylesheet).toContain("box-shadow: none;");
    expect(stylesheet).toContain(".runtime-composer-submit .runtime-composer-submit-icon svg {");
    expect(stylesheet).toContain("width: 18px;");
    expect(stylesheet).toContain("transform: none;");
  });

  it("uses one shared transition shell below the wide desktop breakpoint so navigation remains reachable", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const responsiveStateMachineMarker = "/* Workbench responsive state machine */";
    const responsiveStateMachineLayer = stylesheet.slice(stylesheet.lastIndexOf(responsiveStateMachineMarker));

    expect(stylesheet).toContain("@media (max-width: 1280px)");
    expect(stylesheet.match(/@media \(max-width: 1280px\)/g)).toHaveLength(1);
    expect(responsiveStateMachineLayer).toContain("@media (max-width: 1280px)");
    expect(responsiveStateMachineLayer).not.toContain("@media (max-width: 1919px)");
    expect(responsiveStateMachineLayer).not.toContain(".app-shell.nav-collapsed");
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.app-shell,\s*\.app-shell\.info-mode\s*\{[\s\S]*?display:\s*block;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\],\s*\.runtime-workspace-body\[data-runtime-view="conversation"\],\s*\.runtime-workspace-body\[data-runtime-view="chatRuntime"\]\s*\{[\s\S]*?grid-template-rows:\s*56px minmax\(0, 1fr\) auto;[\s\S]*?overflow:\s*hidden;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-shell\[data-runtime-view="settings"\] \.runtime-workspace-body,\s*\.runtime-workspace-shell\[data-runtime-view="conversation"\] \.runtime-workspace-body,\s*\.runtime-workspace-shell\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body[\s\S]*?\{[\s\S]*?grid-template-rows:\s*56px minmax\(0, 1fr\) auto;[\s\S]*?overflow:\s*hidden;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] > \.runtime-workspace-mobile-header,\s*\.runtime-workspace-body\[data-runtime-view="conversation"\] > \.runtime-workspace-mobile-header,\s*\.runtime-workspace-body\[data-runtime-view="chatRuntime"\] > \.runtime-workspace-mobile-header\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-row:\s*1;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-shell\[data-runtime-view="settings"\] \.runtime-workspace-body > \.runtime-workspace-mobile-header,\s*\.runtime-workspace-shell\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-mobile-header,\s*\.runtime-workspace-shell\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body > \.runtime-workspace-mobile-header[\s\S]*?\{[\s\S]*?display:\s*grid;[\s\S]*?grid-row:\s*1;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] > \.settings-route-body,\s*\.runtime-workspace-body\[data-runtime-view="conversation"\] > \.runtime-workspace-panel,\s*\.runtime-workspace-body\[data-runtime-view="chatRuntime"\] > \.runtime-workspace-panel\s*\{[\s\S]*?min-height:\s*0;[\s\S]*?overflow-y:\s*auto;[\s\S]*?-webkit-overflow-scrolling:\s*touch;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-shell\[data-runtime-view="settings"\] \.runtime-workspace-body > \.settings-route-body,\s*\.runtime-workspace-shell\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-panel,\s*\.runtime-workspace-shell\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body > \.runtime-workspace-panel[\s\S]*?\{[\s\S]*?min-height:\s*0;[\s\S]*?overflow-y:\s*auto;[\s\S]*?-webkit-overflow-scrolling:\s*touch;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] > \.settings-route-body\s*\{[\s\S]*?grid-template-columns:\s*200px minmax\(0, 1fr\);[\s\S]*?grid-template-rows:\s*minmax\(0, 1fr\);[\s\S]*?align-content:\s*stretch;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-nav\s*\{[\s\S]*?position:\s*sticky;[\s\S]*?grid-column:\s*1;[\s\S]*?grid-row:\s*1;[\s\S]*?align-self:\s*start;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-nav-items\s*\{[\s\S]*?display:\s*grid;[\s\S]*?overflow:\s*visible;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-tab\s*\{[\s\S]*?min-height:\s*40px;[\s\S]*?grid-template-columns:\s*20px minmax\(0, 1fr\);/,
    );
  });

  it("moves Settings subsection navigation to top tabs only at the final mobile breakpoint", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] > \.settings-route-body\s*\{[\s\S]*?grid-template-columns:\s*minmax\(0, 1fr\);[\s\S]*?grid-template-rows:\s*auto minmax\(0, 1fr\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-nav-items\s*\{[\s\S]*?display:\s*flex;[\s\S]*?overflow-x:\s*auto;[\s\S]*?touch-action:\s*pan-x;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-tab\s*\{[\s\S]*?flex:\s*1 0 72px;[\s\S]*?min-height:\s*30px;[\s\S]*?grid-template-columns:\s*minmax\(0, 1fr\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-tab-icon\s*\{[\s\S]*?display:\s*none;/,
    );
  });

  it("keeps the Settings runtime redesign aligned to the approved design draft", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".codex-runtime-studio {");
    expect(stylesheet).toContain(".codex-runtime-studio-head {");
    expect(stylesheet).toContain(".codex-runtime-quick-facts {");
    expect(stylesheet).toContain(".codex-runtime-control-grid {");
    expect(stylesheet).toContain(".codex-runtime-control-card {");
    expect(stylesheet).toContain(".codex-runtime-usage-card {");
    expect(stylesheet).toContain(".codex-runtime-provider-station {");
    expect(stylesheet).toContain(".codex-runtime-provider-station-body {");
    expect(stylesheet).not.toContain(".codex-runtime-console {");
    expect(stylesheet).not.toContain(".codex-runtime-core-grid {");
    expect(stylesheet).toContain("@media (max-width: 920px)");
  });

  it("keeps every Settings subsection on one stable content rail", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    const alignmentLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Workbench shell system */"));

    expect(alignmentLayer).toMatch(
      /\.settings-route-content\s*\{[\s\S]*?width:\s*min\(100%, 1180px\);[\s\S]*?max-width:\s*1180px;[\s\S]*?justify-self:\s*start;/,
    );
    expect(alignmentLayer).toMatch(
      /\.settings-route-content > \*,\s*\.settings-section-frame > \*,\s*\.settings-route-content \.memory-view,\s*\.settings-route-content \.route-card-grid,\s*\.settings-route-content \.control-route-grid,\s*\.settings-route-content \.memory-panel,\s*\.settings-route-content \.memory-tabs\s*\{[\s\S]*?width:\s*100%;[\s\S]*?max-width:\s*100%;[\s\S]*?box-sizing:\s*border-box;/,
    );
    expect(alignmentLayer).toMatch(
      /\.settings-section-frame\s*\{[\s\S]*?width:\s*100%;[\s\S]*?max-width:\s*100%;[\s\S]*?min-height:\s*0;[\s\S]*?display:\s*grid;[\s\S]*?align-content:\s*start;/,
    );
    expect(alignmentLayer).toMatch(
      /\.settings-route-content > \.settings-section-frame,\s*\.settings-section-frame > \*\s*\{[\s\S]*?width:\s*100%;[\s\S]*?max-width:\s*100%;[\s\S]*?box-sizing:\s*border-box;/,
    );
    expect(alignmentLayer).toMatch(
      /\.settings-route-content \.route-card-grid\s*\{[\s\S]*?grid-template-columns:\s*repeat\(auto-fit, minmax\(min\(100%, 420px\), 1fr\)\);/,
    );
    expect(alignmentLayer).toMatch(
      /\.settings-route-content \.control-route-grid\s*\{[\s\S]*?width:\s*100%;[\s\S]*?grid-template-columns:\s*repeat\(auto-fit, minmax\(min\(100%, 420px\), 1fr\)\);/,
    );
    expect(alignmentLayer).toMatch(
      /\.settings-route-content \.control-route-grid \.route-card\s*\{[\s\S]*?width:\s*100%;[\s\S]*?max-width:\s*none;/,
    );
    expect(alignmentLayer).toMatch(
      /\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-content\s*\{[\s\S]*?scrollbar-gutter:\s*stable;/,
    );
    expect(alignmentLayer).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\.settings-route-content\s*\{[\s\S]*?width:\s*100%;[\s\S]*?max-width:\s*100%;/,
    );
  });

  it("keeps shared workspace titles on one text column and removes the Settings header divider", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const alignmentLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Workbench shell system */"));

    expect(alignmentLayer).toMatch(
      /\.runtime-workspace-title-leading\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-columns:\s*18px minmax\(0, auto\);[\s\S]*?column-gap:\s*8px;/,
    );
    expect(alignmentLayer).toMatch(
      /\[data-runtime-header-signal-slot\]\s*\{[\s\S]*?width:\s*18px;[\s\S]*?min-width:\s*18px;[\s\S]*?justify-content:\s*center;/,
    );
    expect(alignmentLayer).toMatch(
      /\[data-runtime-header-signal-slot="empty"\]\s*\{[\s\S]*?visibility:\s*hidden;[\s\S]*?pointer-events:\s*none;/,
    );
    expect(alignmentLayer).toMatch(
      /\.runtime-workspace-body\[data-runtime-view="settings"\] > \.runtime-workspace-head\s*\{[\s\S]*?display:\s*block;[\s\S]*?border-bottom:\s*0;[\s\S]*?box-shadow:\s*none;/,
    );
  });

  it("keeps the runtime composer aligned to the assistant-style input surface", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("border-radius: 28px;");
    expect(stylesheet).toContain("border-color: rgba(218, 220, 224, 0.96);");
    expect(stylesheet).toContain("box-shadow: 0 18px 50px -36px rgba(15, 23, 42, 0.30), 0 2px 10px rgba(60, 64, 67, 0.10);");
    expect(stylesheet).toContain(".runtime-composer-submit {");
    expect(stylesheet).toContain("background: #0f172a;");
    expect(stylesheet).toContain("border-color: #0f172a;");
    expect(stylesheet).toContain(".runtime-composer-input:focus {");
    expect(stylesheet).toContain("box-shadow: none;");
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-composer-form {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"conversation\"] .runtime-composer-form {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-composer-input {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"conversation\"] .runtime-composer-input {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"chatRuntime\"] .runtime-composer-submit {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"conversation\"] .runtime-composer-submit {");
  });

  it("styles shared jump controls as one round button set across chat and chatRuntime", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(/\.runtime-workspace-panel\s*\{[\s\S]*?position:\s*relative;/);
    expect(stylesheet).toContain(".scroll-jump-strip {");
    expect(stylesheet).toContain(".scroll-jump-control {");
    expect(stylesheet).toContain(".scroll-jump-control.is-visible {");
    expect(stylesheet).toContain(".scroll-jump-control-icon {");
    expect(stylesheet).toMatch(/\.scroll-jump-strip\s*\{[\s\S]*?position:\s*absolute;/);
    expect(stylesheet).toMatch(/\.chatRuntime-jump-cluster\s*\{[\s\S]*?position:\s*absolute;/);
    expect(stylesheet).toMatch(/\.scroll-jump-control\s*\{[\s\S]*?display:\s*none;[\s\S]*?border-radius:\s*999px;/);
    expect(stylesheet).toMatch(/\.scroll-jump-control\.is-visible\s*\{[\s\S]*?display:\s*inline-flex;/);
    expect(stylesheet).toMatch(/\.chatRuntime-jump-control\s*\{[\s\S]*?display:\s*none;[\s\S]*?border-radius:\s*999px;/);
    expect(stylesheet).toMatch(/\.chatRuntime-jump-control\.is-visible\s*\{[\s\S]*?display:\s*inline-flex;/);
  });

  it("keeps chatRuntime output text selectable while preserving scroll controls", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\]\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-workspace-screen\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;[\s\S]*?touch-action:\s*auto;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-workspace-panel,[\s\S]*?\[data-runtime-view="chatRuntime"\] \.runtime-workspace\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.message-markdown-body,[\s\S]*?\[data-runtime-view="chatRuntime"\] \.message-markdown-rendered \*\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.chatRuntime-final-text,[\s\S]*?\[data-runtime-view="chatRuntime"\] \.message-markdown-shell\s*\{[\s\S]*?display:\s*block;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.runtime-timeline,[\s\S]*?\[data-runtime-view="chatRuntime"\] \.runtime-message-bubble\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;[\s\S]*?touch-action:\s*auto;/,
    );
    expect(stylesheet).not.toMatch(/\[data-runtime-view="chatRuntime"\] \.message-markdown-rendered \*\s*\{[\s\S]*?user-select:\s*text\s*!important;/);
    expect(stylesheet).not.toMatch(/\[data-runtime-view="chatRuntime"\] \.runtime-message-bubble\s*\{[\s\S]*?user-select:\s*text\s*!important;/);
    expect(stylesheet).not.toContain('contenteditable="true"');
    expect(stylesheet).not.toContain("-webkit-user-modify");
    expect(stylesheet).not.toContain("caret-color: transparent");
    expect(stylesheet).not.toMatch(/\[data-runtime-view="chatRuntime"\] \.runtime-workspace-screen\.is-empty\s*\{[\s\S]*?touch-action:\s*none;/);
    expect(stylesheet).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.message-markdown-copy\s*\{[\s\S]*?min-width:\s*40px;[\s\S]*?min-height:\s*40px;/,
    );
    expect(stylesheet).toMatch(/\.chatRuntime-jump-cluster\s*\{[\s\S]*?pointer-events:\s*none;/);
    expect(stylesheet).toMatch(/\.chatRuntime-jump-control\s*\{[\s\S]*?touch-action:\s*manipulation;/);
  });

  it("renders chatRuntime markdown answers with Codex-style readable prose instead of a dark card", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const codexLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Codex-style chatRuntime markdown final override */"));

    expect(codexLayer).toContain('[data-runtime-view="chatRuntime"] .chatRuntime-final-output,');
    expect(codexLayer).toContain("background: transparent;");
    expect(codexLayer).toContain("color: #111827;");
    expect(codexLayer).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.chat-md-inline-code\s*\{[\s\S]*?background:\s*rgba\(175, 184, 193, 0\.22\);[\s\S]*?color:\s*#24292f;/,
    );
    expect(codexLayer).toMatch(
      /\[data-runtime-view="chatRuntime"\] \.chat-md-pre,[\s\S]*?background:\s*#f6f8fa;[\s\S]*?color:\s*#24292f;/,
    );
    expect(codexLayer).not.toContain("background: #111315;");
  });

  it("keeps narrow chatRuntime headers on one line and preserves composer meta visibility", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-runtime.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 420px\) \{[\s\S]*?\.chatRuntime-workspace-row\s*\{[\s\S]*?flex-wrap:\s*wrap;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 420px\) \{[\s\S]*?\.chatRuntime-workspace-actions\s*\{[\s\S]*?flex-wrap:\s*wrap;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.chatRuntime-composer-meta\s*\{[\s\S]*?display:\s*block;/,
    );
  });

  it("keeps chatRuntime command bubbles from collapsing into character-by-character wraps on mobile", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-runtime.css"), "utf8");
    const block = stylesheet.match(/\.chatRuntime-log-text\s*\{([\s\S]*?)\n\}/)?.[1] || "";

    expect(block).toContain("white-space: pre-wrap;");
    expect(block).toContain("word-break: normal;");
    expect(block).toContain("overflow-wrap: break-word;");
    expect(block).not.toContain("overflow-wrap: anywhere;");
  });

  it("keeps conversation process details on a readable full-width column instead of per-character wraps", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-core.css"), "utf8");
    const bodyBlock = stylesheet.match(/\.conversation-process-step-body,\s*\.conversation-process-step-body\s*\{([\s\S]*?)\n\}/)?.[1] || "";
    const markdownBlock = stylesheet.match(/\.conversation-process-step-body\s*>\s*\.message-markdown-rendered,\s*\.conversation-process-step-body\s*>\s*\.message-markdown-rendered\s*\{([\s\S]*?)\n\}/)?.[1] || "";
    const titleBlock = stylesheet.match(/\.conversation-process-step-title,\s*\.conversation-process-step-title\s*\{([\s\S]*?)\n\}/)?.[1] || "";

    expect(bodyBlock).toContain("width: 100%;");
    expect(bodyBlock).toContain("overflow-wrap: break-word;");
    expect(bodyBlock).toContain("word-break: normal;");
    expect(bodyBlock).not.toContain("overflow-wrap: anywhere;");
    expect(markdownBlock).toContain("width: 100%;");
    expect(markdownBlock).toContain("overflow-wrap: break-word;");
    expect(markdownBlock).toContain("word-break: normal;");
    expect(titleBlock).toContain("width: 100%;");
    expect(titleBlock).toContain("display: block;");
    expect(titleBlock).toContain("overflow-wrap: break-word;");
    expect(titleBlock).toContain("word-break: normal;");
  });

  it("pins the mobile chatRuntime composer to the viewport and reserves message space above it", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-runtime.css"), "utf8");
    const mobileComposerBlock = stylesheet.match(
      /@media \(max-width: 760px\) \{[\s\S]*?\.chatRuntime-composer-shell\s*\{([\s\S]*?)\n  \}/,
    );

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.chatRuntime-composer-shell\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?left:\s*0;[\s\S]*?right:\s*0;[\s\S]*?bottom:\s*var\(--keyboard-offset\);[\s\S]*?padding:\s*0 10px calc\(10px \+ env\(safe-area-inset-bottom\)\);/,
    );
    expect(mobileComposerBlock?.[1]).toBeTruthy();
    expect(mobileComposerBlock?.[1]).not.toContain("transition: bottom");
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.chatRuntime-jump-cluster\s*\{[\s\S]*?bottom:\s*calc\(var\(--runtime-composer-rest-inset, var\(--runtime-composer-inset, 0px\)\) \+ 24px\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.chatRuntime-chat-screen\s*\{[\s\S]*?padding:\s*var\(--chatRuntime-chat-screen-padding-top\) var\(--chatRuntime-chat-screen-padding-x\) 20px;/,
    );
  });

  it("keeps mobile runtime jump controls inside the workspace panel instead of fixed to the viewport", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const isolationStylesheet = readFileSync(resolve(currentDirectory, "../../styles/runtimeKeyboardIsolation.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-screen,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-screen\s*\{[\s\S]*?height:\s*100%;[\s\S]*?max-height:\s*100%;/,
    );
    expect(stylesheet).toContain("padding-bottom: 20px;");
    expect(stylesheet).toMatch(
      /\.runtime-workspace-panel\s*\{[\s\S]*?position:\s*relative;/,
    );
    expect(isolationStylesheet).toMatch(
      /Mobile runtime keyboard isolation:[\s\S]*?\.runtime-workspace-body > \.runtime-composer-shell\s*\{[\s\S]*?position:\s*relative !important;[\s\S]*?bottom:\s*auto !important;/,
    );
    expect(stylesheet).toMatch(/\.chatRuntime-jump-cluster\s*\{[\s\S]*?position:\s*absolute;[\s\S]*?bottom:\s*18px;/);
    expect(stylesheet).not.toContain("--runtime-composer-rest-inset");
    expect(stylesheet).not.toContain("--runtime-composer-inset");
    expect(stylesheet).not.toContain("bottom: calc(var(--runtime-composer-rest-inset");
    expect(stylesheet).toMatch(/\.chatRuntime-jump-control\s*\{[\s\S]*?border-radius:\s*999px;/);
    expect(stylesheet).not.toContain("transition: bottom 220ms cubic-bezier(0.22, 1, 0.36, 1);");
  });

  it("keeps mobile chatRuntime layouts on a single main surface instead of stacking nested rounded capsules", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body,\s*\[data-runtime-view="chatRuntime"\] \.runtime-workspace-body\s*\{[\s\S]*?border-radius:\s*0;[\s\S]*?border-left:\s*0;[\s\S]*?border-right:\s*0;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-form\s*\{[\s\S]*?border-radius:\s*26px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-submit\s*\{[\s\S]*?box-shadow:\s*none;/,
    );
  });

  it("keeps shared workbench titles aligned and removes settings-only chrome seams", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const shellLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Final shared workbench shell alignment */"));

    expect(shellLayer).toMatch(
      /\.runtime-workspace-title-leading\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-columns:\s*18px minmax\(0,\s*auto\);[\s\S]*?align-items:\s*center;/,
    );
    expect(shellLayer).toMatch(
      /\[data-runtime-header-signal-slot="empty"\]\s*\{[\s\S]*?visibility:\s*hidden;[\s\S]*?pointer-events:\s*none;/,
    );
    expect(shellLayer).toMatch(
      /\.runtime-workspace-title-leading h4\s*\{[\s\S]*?grid-column:\s*2;/,
    );
    expect(shellLayer).toMatch(
      /\.runtime-workspace-title-button\s*\{[\s\S]*?appearance:\s*none;[\s\S]*?background:\s*transparent;[\s\S]*?font:\s*inherit;/,
    );
    expect(shellLayer).toMatch(
      /\.runtime-workspace-body\[data-runtime-view="settings"\] \.settings-route-content\s*\{[\s\S]*?border-bottom:\s*0;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(shellLayer).toContain(".settings-language-control");
    expect(shellLayer).toMatch(
      /\.settings-general-section\s*\{[\s\S]*?display:\s*grid;[\s\S]*?gap:\s*14px;/,
    );
    expect(shellLayer).toMatch(
      /\.settings-general-panel\s*\{[\s\S]*?display:\s*grid;[\s\S]*?gap:\s*12px;/,
    );
    expect(shellLayer).toMatch(
      /\.settings-language-control\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-columns:\s*minmax\(0, 1fr\) auto;[\s\S]*?align-items:\s*center;[\s\S]*?min-height:\s*44px;/,
    );
    expect(shellLayer).toMatch(
      /\.settings-language-label\s*\{[\s\S]*?font-size:\s*12px;[\s\S]*?font-weight:\s*720;/,
    );
    expect(shellLayer).toMatch(
      /\.settings-language-value\s*\{[\s\S]*?border-radius:\s*999px;[\s\S]*?background:\s*rgba\(37, 99, 235, 0\.08\);/,
    );
    expect(shellLayer).not.toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.settings-language-control/,
    );
  });

  it("keeps mobile Runtime settings dense enough for Codex usage to stay above provider details", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const shellLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Final shared workbench shell alignment */"));

    expect(shellLayer).toContain("/* Mobile Settings Runtime density pass */");
    expect(shellLayer).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.codex-runtime-control-grid\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-columns:\s*1fr;[\s\S]*?grid-template-areas:\s*none;[\s\S]*?gap:\s*10px;/,
    );
    expect(shellLayer).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.codex-runtime-identity-card\s*\{[\s\S]*?grid-area:\s*auto;[\s\S]*?order:\s*1;[\s\S]*?padding:\s*12px;/,
    );
    expect(shellLayer).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.codex-runtime-usage-card\s*\{[\s\S]*?grid-area:\s*auto;[\s\S]*?order:\s*2;[\s\S]*?padding:\s*12px;/,
    );
    expect(shellLayer).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.codex-runtime-device-login\s*\{[\s\S]*?grid-area:\s*auto;[\s\S]*?order:\s*3;[\s\S]*?padding:\s*10px 12px;[\s\S]*?border-radius:\s*12px;/,
    );
    expect(shellLayer).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.codex-runtime-provider-station\s*\{[\s\S]*?order:\s*4;/,
    );
    expect(shellLayer).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.codex-runtime-device-login-head p\s*\{[\s\S]*?display:\s*none;/,
    );
  });

  it("keeps desktop Runtime and General settings from inheriting the compact mobile card layout", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const shellLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Final shared workbench shell alignment */"));

    expect(shellLayer).toMatch(
      /\.codex-runtime-control-grid\s*\{[\s\S]*?grid-template-columns:\s*minmax\(280px,\s*0\.78fr\) minmax\(420px,\s*1\.22fr\);[\s\S]*?grid-template-areas:\s*"identity usage" "device usage";[\s\S]*?align-items:\s*start;/,
    );
    expect(shellLayer).toMatch(
      /\.codex-runtime-identity-card\s*\{[\s\S]*?grid-area:\s*identity;[\s\S]*?align-self:\s*start;/,
    );
    expect(shellLayer).toMatch(
      /\.codex-runtime-usage-card\s*\{[\s\S]*?grid-area:\s*usage;[\s\S]*?align-self:\s*stretch;/,
    );
    expect(shellLayer).toMatch(
      /\.codex-runtime-device-login\s*\{[\s\S]*?grid-area:\s*device;[\s\S]*?align-self:\s*start;/,
    );
    expect(shellLayer).toMatch(
      /\.settings-general-panel\s*\{[\s\S]*?max-width:\s*760px;[\s\S]*?justify-self:\s*start;[\s\S]*?grid-template-columns:\s*minmax\(0,\s*1fr\);[\s\S]*?align-content:\s*start;/,
    );
    expect(shellLayer).toMatch(
      /\.settings-general-panel \.codex-runtime-service-controls-panel\s*\{[\s\S]*?display:\s*grid;[\s\S]*?grid-template-columns:\s*minmax\(0,\s*1fr\) auto;[\s\S]*?align-items:\s*center;/,
    );
    expect(shellLayer).toMatch(
      /\.settings-general-panel \.codex-runtime-service-label,\s*\.settings-general-panel \.codex-runtime-service-copy\s*\{[\s\S]*?grid-column:\s*1;[\s\S]*?text-align:\s*left;/,
    );
    expect(shellLayer).toMatch(
      /\.settings-general-panel \.codex-runtime-service-primary-action\s*\{[\s\S]*?grid-column:\s*2;[\s\S]*?grid-row:\s*1 \/ span 2;[\s\S]*?justify-self:\s*end;/,
    );
    expect(shellLayer).toMatch(
      /@media \(max-width: 1280px\) \{[\s\S]*?\.runtime-workspace-mobile-header \.runtime-workspace-mobile-action\s*\{[\s\S]*?width:\s*auto;[\s\S]*?max-width:\s*max-content;[\s\S]*?justify-self:\s*start;/,
    );
  });

  it("defines a shared interaction polish baseline for motion, focus, numeric, and scroll behavior", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const rootStylesheet = readFileSync(resolve(currentDirectory, "../../styles/root.css"), "utf8");
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const interactionLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Interaction polish baseline */"));

    expect(rootStylesheet).toContain("--motion-scale: 1;");
    expect(rootStylesheet).toContain("--ease-out-expo: cubic-bezier(0.16, 1, 0.3, 1);");
    expect(rootStylesheet).toContain("--ease-in-exit: cubic-bezier(0.7, 0, 0.84, 0);");
    expect(rootStylesheet).toMatch(
      /@media \(prefers-reduced-motion: reduce\) \{[\s\S]*?:root\s*\{[\s\S]*?--motion-scale:\s*0;/,
    );
    expect(rootStylesheet).toMatch(
      /@media \(prefers-reduced-motion: reduce\) \{[\s\S]*?\*,\s*\*::before,\s*\*::after\s*\{[\s\S]*?scroll-behavior:\s*auto !important;/,
    );

    expect(interactionLayer).toContain("/* Interaction polish baseline */");
    expect(interactionLayer).toMatch(
      /\.menu-item,\s*\.nav-locale-button,\s*\.nav-session-rail-action,\s*\.runtime-workspace-button,\s*\.runtime-workspace-mobile-action,\s*\.route-mobile-head \.conversation-mobile-action,\s*\.runtime-composer-utility,\s*\.runtime-composer-upload,\s*\.runtime-composer-submit,\s*\.route-card-action,\s*\.modal-footer button\s*\{[\s\S]*?transition:[\s\S]*?var\(--ease-out-expo\);/,
    );
    expect(interactionLayer).toMatch(
      /\.menu-item:active,\s*\.nav-locale-button:active,\s*\.nav-session-rail-action:active,\s*\.runtime-workspace-button:active,\s*\.runtime-workspace-mobile-action:active,\s*\.route-mobile-head \.conversation-mobile-action:active,\s*\.runtime-composer-utility:active,\s*\.runtime-composer-upload:active,\s*\.runtime-composer-submit:active,\s*\.route-card-action:active,\s*\.modal-footer button:active\s*\{[\s\S]*?transform:\s*scale\(0\.96\);/,
    );
    expect(interactionLayer).toMatch(
      /\.runtime-session-card,\s*\.session-card,\s*\.welcome-target-card,\s*\.prompt,\s*\.settings-route-content \.skill-route-card,\s*\.settings-route-content \.codex-account-card\s*\{[\s\S]*?box-shadow:\s*var\(--surface-shadow-rest\);[\s\S]*?transition:[\s\S]*?var\(--ease-out-expo\);/,
    );
    expect(interactionLayer).toMatch(
      /\.runtime-session-card:hover,\s*\.session-card:hover,\s*\.welcome-target-card:hover,\s*\.prompt:hover,\s*\.settings-route-content \.skill-route-card:hover,\s*\.settings-route-content \.codex-account-card:hover\s*\{[\s\S]*?transform:\s*translateY\(-1px\);[\s\S]*?box-shadow:\s*var\(--surface-shadow-hover\);/,
    );
    expect(interactionLayer).toMatch(
      /button:focus-visible,\s*a:focus-visible,\s*textarea:focus-visible,\s*input:focus-visible,\s*select:focus-visible,\s*\[role="button"\]:focus-visible\s*\{[\s\S]*?outline:\s*none;[\s\S]*?box-shadow:\s*var\(--focus-ring\);/,
    );
    expect(interactionLayer).toMatch(
      /\.tnum,\s*\.session-card-id,\s*\.runtime-session-hash,\s*\.runtime-session-meta,\s*\.route-table-subtext,\s*\.route-data-table code,\s*\.codex-account-card-metrics,\s*\.codex-runtime-restart-status strong\s*\{[\s\S]*?font-variant-numeric:\s*tabular-nums;[\s\S]*?font-feature-settings:\s*"tnum" 1;/,
    );
    expect(interactionLayer).toMatch(
      /\.runtime-workspace-screen,\s*\.route-view,\s*\.modal-body,\s*\.workspace-details-content,\s*\.runtime-composer-command-list,\s*\.runtime-composer-panel\s*\{[\s\S]*?overscroll-behavior:\s*contain;/,
    );
    expect(interactionLayer).toMatch(
      /\.runtime-composer-command-list,\s*\.settings-route-content \.route-data-table-wrap\s*\{[\s\S]*?scrollbar-width:\s*none;/,
    );
    expect(interactionLayer).toMatch(
      /\.modal-backdrop,\s*\.workspace-details-layer,\s*\.runtime-restart-overlay\s*\{[\s\S]*?animation:\s*overlay-fade-in calc\(180ms \* var\(--motion-scale\)\) var\(--ease-out-expo\) both;/,
    );
    expect(interactionLayer).toMatch(
      /\.modal-dialog,\s*\.workspace-details-panel,\s*\.runtime-restart-panel\s*\{[\s\S]*?animation:\s*surface-enter calc\(260ms \* var\(--motion-scale\)\) var\(--ease-out-expo\) both;/,
    );
  });
});
