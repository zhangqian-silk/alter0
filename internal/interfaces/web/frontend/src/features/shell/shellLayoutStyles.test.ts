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
      stylesheet.lastIndexOf("/* Codex-style terminal markdown final override */"),
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

  it("moves the mobile runtime composer with the visual-viewport-adjusted bottom offset instead of a transform layer", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-shell\s*\{[\s\S]*?bottom:\s*var\(--keyboard-composer-offset, var\(--keyboard-offset, 0px\)\);/,
    );
    expect(stylesheet).not.toMatch(/\.runtime-composer-shell\s*\{[^}]*transform:/);
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
      /\.user-message-shell \.terminal-turn-prompt \.terminal-log-text,[\s\S]*?\.user-message-shell \.terminal-log-text\s*\{[\s\S]*?font-size:\s*15px;[\s\S]*?line-height:\s*1\.45;/,
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
      /\[data-runtime-view="conversation"\] \.user-message-shell \.terminal-log-main,\s*\[data-runtime-view="terminal"\] \.user-message-shell \.terminal-log-main\s*\{[\s\S]*?max-width:\s*100%;/,
    );
    expect(stylesheet).not.toContain("background: linear-gradient(180deg, rgba(229, 242, 255, 0.98)");
  });

  it("defines shared runtime message bubbles for chat and terminal, and future runtime pages", () => {
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
      /\[data-runtime-view="conversation"\] \.chat-md-pre,\s*\[data-runtime-view="conversation"\] \.terminal-final-rendered \.chat-md-pre\s*\{[\s\S]*?border:\s*1px solid #e5e7eb;[\s\S]*?border-radius:\s*10px;[\s\S]*?background:\s*#f7f7f8;/,
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
      /\.runtime-thinking-shell\.terminal-process-shell\s*\{[\s\S]*?margin:\s*0 0 12px;[\s\S]*?padding:\s*0;[\s\S]*?border:\s*0;[\s\S]*?border-radius:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-thinking-shell\.terminal-process-shell::before\s*\{[\s\S]*?display:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-thinking-toggle\.terminal-process-toggle\s*\{[\s\S]*?display:\s*inline-flex;[\s\S]*?width:\s*auto;[\s\S]*?justify-self:\s*start;[\s\S]*?align-self:\s*start;[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-thinking-shell\.terminal-process-shell\.is-collapsed \.runtime-thinking-toggle,\s*\.runtime-thinking-toggle\.terminal-process-toggle:hover\s*\{[\s\S]*?border:\s*0;[\s\S]*?background:\s*transparent;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-thinking-toggle \.terminal-process-title\s*\{[\s\S]*?text-transform:\s*none;[\s\S]*?letter-spacing:\s*0;[\s\S]*?color:\s*#7a7f87;/,
    );
    const mobileThinkingBlocks = Array.from(stylesheet.matchAll(
      /\.runtime-thinking-shell \.terminal-process-body:not\(\[hidden\]\)\s*\{([^}]*)\}/g,
    ));
    const mobileThinkingBlock = mobileThinkingBlocks[mobileThinkingBlocks.length - 1]?.[1] || "";
    expect(mobileThinkingBlock).not.toContain("position: fixed;");
    expect(mobileThinkingBlock).not.toContain("backdrop-filter:");
    expect(mobileThinkingBlock).toContain("max-height: none;");
    expect(mobileThinkingBlock).toContain("overflow: visible;");

    expect(stylesheet).not.toMatch(
      /\[data-runtime-view="conversation"\] \.runtime-thinking-shell \.terminal-process-body:not\(\[hidden\]\)/,
    );
  });

  it("keeps terminal runtime thinking expanded inline on mobile", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const terminalThinkingBlocks = Array.from(stylesheet.matchAll(
      /\[data-runtime-view="terminal"\] \.runtime-thinking-shell \.terminal-process-body:not\(\[hidden\]\)\s*\{([^}]*)\}/g,
    ));
    const terminalThinkingBlock = terminalThinkingBlocks[terminalThinkingBlocks.length - 1]?.[1] || "";

    expect(terminalThinkingBlock).toContain("position: static;");
    expect(terminalThinkingBlock).toContain("max-height: none;");
    expect(terminalThinkingBlock).toContain("overflow: visible;");
    expect(terminalThinkingBlock).not.toContain("position: fixed;");
    expect(terminalThinkingBlock).not.toContain("backdrop-filter:");
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
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.modal-backdrop \{[\s\S]*?align-items:\s*center;[\s\S]*?padding:\s*16px;[\s\S]*?\.modal-dialog \{[\s\S]*?width:\s*min\(100%, 420px\);[\s\S]*?border-radius:\s*12px;/,
    );
  });

  it("keeps mobile app shell height at the stable keyboard baseline while only the composer consumes keyboard offset", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("--mobile-viewport-height: 100dvh;");
    expect(stylesheet).toContain("--keyboard-offset: 0px;");
    expect(stylesheet).toContain("--keyboard-composer-offset: 0px;");
    expect(stylesheet).toContain("height: calc(var(--mobile-viewport-height, 100dvh) + var(--keyboard-offset, 0px));");
    expect(stylesheet).toContain(".chat-pane:not(.page-mode) {");
    expect(stylesheet).toContain("height: min(100%, var(--mobile-viewport-height, 100dvh));");
    expect(stylesheet).toContain(".chat-pane.page-mode {");
    expect(stylesheet).toContain("height: min(100%, calc(var(--mobile-viewport-height, 100dvh) + var(--keyboard-offset, 0px)));");
    expect(stylesheet).toContain("bottom: var(--keyboard-composer-offset, var(--keyboard-offset, 0px));");
  });

  it("locks document scrolling on narrow screens so keyboard focus cannot move the whole workbench", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/root.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?html,\s*body,\s*#frontend-root\s*\{[\s\S]*?height:\s*100%;[\s\S]*?overflow:\s*hidden;[\s\S]*?overscroll-behavior:\s*none;/,
    );
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
    expect(stylesheet).toContain("[data-runtime-view=\"terminal\"] .runtime-workspace-body {");
    expect(stylesheet).toContain("grid-template-rows: auto auto minmax(0, 1fr) auto;");
  });

  it("keeps chat and terminal workspace header controls the same size", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).not.toContain("[data-runtime-view=\"conversation\"] .runtime-workspace-actions .runtime-workspace-button {");
    expect(stylesheet).toMatch(
      /\.workspace-header-actions \.runtime-workspace-button\s*\{[\s\S]*?min-height:\s*24px;[\s\S]*?padding:\s*0 8px;[\s\S]*?font-size:\s*10px;/,
    );
  });

  it("keeps terminal-runtime footer chrome aligned with the shared composer surface", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).not.toMatch(
      /\[data-runtime-view="terminal"\] \.runtime-composer-shell\s*\{[\s\S]*?background:\s*linear-gradient\(180deg, rgba\(241, 245, 249,/,
    );
    expect(stylesheet).not.toMatch(
      /\[data-runtime-view="terminal"\] \.runtime-composer-shell\s*\{[\s\S]*?padding:\s*10px 16px 14px;/,
    );
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-composer-note {");
    expect(stylesheet).toContain("[data-runtime-view=\"terminal\"] .runtime-composer-form .runtime-composer-meta {");
    expect(stylesheet).toContain("min-height: 26px;");
    expect(stylesheet).toContain("[data-runtime-view=\"terminal\"] .runtime-composer-meta[data-runtime-status=\"failed\"]");
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-composer-input {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-composer-submit {");
  });

  it("locks conversation empty states in place without disabling terminal long-press selection", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".conversation-console-panel.is-empty {");
    expect(stylesheet).toContain("overflow: hidden;");
    expect(stylesheet).toContain("[data-runtime-view=\"conversation\"] .runtime-workspace-screen.is-empty {");
    expect(stylesheet).toContain("overscroll-behavior: none;");
    expect(stylesheet).toContain("touch-action: none;");
    expect(stylesheet).toContain("-webkit-overflow-scrolling: auto;");
    expect(stylesheet).not.toMatch(/(^|\n)\.runtime-workspace-screen\.is-empty\s*\{/);
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-workspace-screen.is-empty {");
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
    expect(stylesheet).toContain("padding: var(--terminal-chat-screen-padding-top) calc(var(--terminal-chat-screen-padding-x) + 4px) 26px;");
    expect(stylesheet).toContain("border-radius: 12px;");
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
    expect(stylesheet).toContain("[data-runtime-view=\"terminal\"] {");
    expect(stylesheet).toContain(".mobile-backdrop {");
  });

  it("keeps runtime workspaces on one desktop content column after the primary nav owns the session list", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\],\s*\[data-runtime-view="terminal"\]\s*\{[\s\S]*?grid-template-columns:\s*minmax\(0, 1fr\);/,
    );
    expect(stylesheet).toMatch(
      /\.runtime-workspace-session-pane\.is-navigation-owned\s*\{[\s\S]*?display:\s*none;/,
    );
    expect(stylesheet).not.toMatch(
      /\[data-runtime-view="terminal"\]\s*\{\s*grid-template-columns:\s*minmax\(280px, 320px\) minmax\(0, 1fr\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\],\s*\[data-runtime-view="terminal"\]\s*\{[\s\S]*?grid-template-columns:\s*1fr;[\s\S]*?padding:\s*0;[\s\S]*?gap:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body,\s*\[data-runtime-view="terminal"\] \.runtime-workspace-body\s*\{[\s\S]*?grid-template-rows:\s*auto auto minmax\(0, 1fr\) auto;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-mobile-header,\s*\[data-runtime-view="terminal"\] \.runtime-workspace-body > \.runtime-workspace-mobile-header\s*\{[\s\S]*?grid-row:\s*1;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-head,\s*\[data-runtime-view="terminal"\] \.runtime-workspace-body > \.runtime-workspace-head\s*\{[\s\S]*?grid-row:\s*2;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-workspace-panel,\s*\[data-runtime-view="terminal"\] \.runtime-workspace-body > \.runtime-workspace-panel\s*\{[\s\S]*?grid-row:\s*3;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body > \.runtime-composer-shell,\s*\[data-runtime-view="terminal"\] \.runtime-workspace-body > \.runtime-composer-shell\s*\{[\s\S]*?grid-row:\s*4;[\s\S]*?align-self:\s*end;/,
    );
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
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-panel\[data-runtime-config-surface="terminal"\]\s*\{[\s\S]*?bottom:\s*calc\(env\(safe-area-inset-bottom\) \+ 120px\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-submit\s*\{[\s\S]*?width:\s*36px;[\s\S]*?min-width:\s*36px;[\s\S]*?padding:\s*0;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-tools\s*\{[\s\S]*?flex-wrap:\s*nowrap;[\s\S]*?gap:\s*10px;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="terminal"\] \.runtime-composer-form \.runtime-composer-tools\s*\{[\s\S]*?justify-content:\s*flex-end;/,
    );
  });

  it("uses one polished mobile composer tray across chat and terminal runtime pages", () => {
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
    expect(stylesheet).toContain("[data-runtime-view=\"terminal\"] .runtime-composer-form .runtime-composer-upload {");
    expect(stylesheet).toContain("width: 38px;");
    expect(stylesheet).toContain("height: 38px;");
    expect(stylesheet).toContain("align-items: center;");
    expect(stylesheet).toContain("justify-content: center;");
    expect(stylesheet).toContain("border-radius: 8px;");
    expect(stylesheet).toContain(".runtime-composer-upload-label {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-session-topline .task-summary-status");
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

    expect(stylesheet).not.toContain('[data-runtime-header-kind="terminal"]');
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

  it("aligns Settings route headings with the Chat and Terminal workspace header chrome", () => {
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

  it("keeps mobile empty-state headers on a single terminal-style row while preserving visible title space", () => {
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
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-composer-shell\s*\{[\s\S]*?border-top:\s*0;[\s\S]*?background:\s*linear-gradient\(180deg, rgba\(255, 255, 255, 0\) 0%, rgba\(250, 252, 255, 0\.94\) 48%, rgba\(250, 252, 255, 0\.98\) 100%\);/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-input\s*\{[\s\S]*?width:\s*100%;[\s\S]*?height:\s*72px;[\s\S]*?min-height:\s*72px;[\s\S]*?resize:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-submit\s*\{[\s\S]*?background:\s*#0f172a;[\s\S]*?color:\s*#fff;/,
    );
  });

  it("keeps the full page state layer for settings subpages and terminal live overlays", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("/* UI UX Pro Max full page states layer */");
    expect(stylesheet).toContain(".settings-route-content[data-settings-route-content=\"runtime\"] .settings-composite-section");
    expect(stylesheet).toContain(".settings-composite-section[data-settings-section=\"workspaces\"]");
    expect(stylesheet).toContain(".memory-tabs");
    expect(stylesheet).toContain(".runtime-composer-command-list");
    expect(stylesheet).toContain(".runtime-composer-panel[data-runtime-config-surface=\"terminal\"]");
    expect(stylesheet).toContain(".terminal-skill-section .conversation-check-item:has(input:checked)");
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

  it("anchors the narrow-screen composer to the visual-viewport-adjusted keyboard offset instead of a fixed viewport bottom", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain("@media (max-width: 760px)");
    expect(stylesheet).toContain("bottom: var(--keyboard-composer-offset, var(--keyboard-offset));");
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

  it("keeps runtime mobile shells and long terminal content from creating page-level horizontal overflow", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const rootStylesheet = readFileSync(resolve(currentDirectory, "../../styles/root.css"), "utf8");
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(rootStylesheet).toMatch(/html,\s*body,\s*#frontend-root\s*\{[\s\S]*?overflow-x:\s*hidden;/);
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.message-markdown-shell,[\s\S]*?\[data-runtime-view="terminal"\] \.message-markdown-rendered\s*\{[\s\S]*?min-width:\s*0;[\s\S]*?max-width:\s*100%;[\s\S]*?box-sizing:\s*border-box;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.terminal-turn-prompt \.terminal-log-main,\s*\[data-runtime-view="terminal"\] \.terminal-turn-prompt \.terminal-log-main\s*\{[\s\S]*?max-width:\s*min\(var\(--user-message-max-width, 80%\), calc\(100% - 24px\)\);/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.terminal-log-text,\s*\[data-runtime-view="terminal"\] \.terminal-log-text,[\s\S]*?\.chat-md-inline-code\s*\{[\s\S]*?overflow-wrap:\s*anywhere;[\s\S]*?word-break:\s*break-word;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.chat-md-table-wrap,\s*\[data-runtime-view="terminal"\] \.chat-md-table-wrap\s*\{[\s\S]*?width:\s*100%;[\s\S]*?max-width:\s*100%;[\s\S]*?overflow-x:\s*auto;[\s\S]*?-webkit-overflow-scrolling:\s*touch;/,
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

  it("styles the terminal workspace like a restrained command center while reusing shared session cards", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-session-card {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-session-card {");
    expect(stylesheet).toContain("[data-runtime-view=\"terminal\"] .runtime-workspace-screen {");
    expect(stylesheet).toContain(".runtime-workspace-body {");
    expect(stylesheet).toContain("linear-gradient(180deg, rgba(255, 255, 255, 0.99) 0%, rgba(243, 247, 251, 0.98) 100%)");
    expect(stylesheet).toContain("box-shadow: 0 34px 70px -54px rgba(15, 23, 42, 0.18);");
  });

  it("lays out the terminal composer as a single input surface with a footer tool row", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toContain(".runtime-composer-form {");
    expect(stylesheet).toContain("padding: 16px 18px 14px;");
    expect(stylesheet).toContain(".runtime-composer-input {");
    expect(stylesheet).toContain("min-height: 78px;");
    expect(stylesheet).toContain("padding: 10px 8px 8px;");
    expect(stylesheet).toContain("[data-runtime-view=\"terminal\"] .runtime-composer-form .runtime-composer-tools {");
    expect(stylesheet).toContain("justify-content: flex-end;");
    expect(stylesheet).toContain("[data-runtime-view=\"terminal\"] .runtime-composer-form .runtime-composer-meta {");
    expect(stylesheet).toContain("border: 0;");
    expect(stylesheet).toContain("background: rgba(238, 244, 249, 0.96);");
    expect(stylesheet).toContain("box-shadow: none;");
    expect(stylesheet).toContain(".runtime-composer-submit .runtime-composer-submit-icon svg {");
    expect(stylesheet).toContain("width: 18px;");
    expect(stylesheet).toContain("transform: none;");
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
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-composer-form {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"conversation\"] .runtime-composer-form {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-composer-input {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"conversation\"] .runtime-composer-input {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"terminal\"] .runtime-composer-submit {");
    expect(stylesheet).not.toContain("[data-runtime-view=\"conversation\"] .runtime-composer-submit {");
  });

  it("styles shared jump controls as one round button set across chat and terminal", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(/\.runtime-workspace-panel\s*\{[\s\S]*?position:\s*relative;/);
    expect(stylesheet).toContain(".scroll-jump-strip {");
    expect(stylesheet).toContain(".scroll-jump-control {");
    expect(stylesheet).toContain(".scroll-jump-control.is-visible {");
    expect(stylesheet).toContain(".scroll-jump-control-icon {");
    expect(stylesheet).toMatch(/\.scroll-jump-strip\s*\{[\s\S]*?position:\s*absolute;/);
    expect(stylesheet).toMatch(/\.scroll-jump-control\s*\{[\s\S]*?border-radius:\s*999px;/);
    expect(stylesheet).toMatch(/\.terminal-jump-control\s*\{[\s\S]*?border-radius:\s*999px;/);
  });

  it("keeps terminal output text selectable while preserving scroll controls", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /\[data-runtime-view="terminal"\]\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="terminal"\] \.runtime-workspace-screen\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;[\s\S]*?touch-action:\s*auto;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="terminal"\] \.runtime-workspace-panel,[\s\S]*?\[data-runtime-view="terminal"\] \.runtime-workspace\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="conversation"\] \.message-markdown-body,[\s\S]*?\[data-runtime-view="terminal"\] \.message-markdown-rendered \*\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="terminal"\] \.terminal-final-text,[\s\S]*?\[data-runtime-view="terminal"\] \.message-markdown-shell\s*\{[\s\S]*?display:\s*block;/,
    );
    expect(stylesheet).toMatch(
      /\[data-runtime-view="terminal"\] \.runtime-timeline,[\s\S]*?\[data-runtime-view="terminal"\] \.runtime-message-bubble\s*\{[\s\S]*?user-select:\s*text;[\s\S]*?-webkit-user-select:\s*text;[\s\S]*?-webkit-touch-callout:\s*default;[\s\S]*?touch-action:\s*auto;/,
    );
    expect(stylesheet).not.toMatch(/\[data-runtime-view="terminal"\] \.message-markdown-rendered \*\s*\{[\s\S]*?user-select:\s*text\s*!important;/);
    expect(stylesheet).not.toMatch(/\[data-runtime-view="terminal"\] \.runtime-message-bubble\s*\{[\s\S]*?user-select:\s*text\s*!important;/);
    expect(stylesheet).not.toContain('contenteditable="true"');
    expect(stylesheet).not.toContain("-webkit-user-modify");
    expect(stylesheet).not.toContain("caret-color: transparent");
    expect(stylesheet).not.toMatch(/\[data-runtime-view="terminal"\] \.runtime-workspace-screen\.is-empty\s*\{[\s\S]*?touch-action:\s*none;/);
    expect(stylesheet).toMatch(
      /\[data-runtime-view="terminal"\] \.message-markdown-copy\s*\{[\s\S]*?min-width:\s*40px;[\s\S]*?min-height:\s*40px;/,
    );
    expect(stylesheet).toMatch(/\.terminal-jump-cluster\s*\{[\s\S]*?pointer-events:\s*none;/);
    expect(stylesheet).toMatch(/\.terminal-jump-control\s*\{[\s\S]*?touch-action:\s*manipulation;/);
  });

  it("renders terminal markdown answers with Codex-style readable prose instead of a dark card", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");
    const codexLayer = stylesheet.slice(stylesheet.lastIndexOf("/* Codex-style terminal markdown final override */"));

    expect(codexLayer).toContain('[data-runtime-view="terminal"] .terminal-final-output,');
    expect(codexLayer).toContain("background: transparent;");
    expect(codexLayer).toContain("color: #111827;");
    expect(codexLayer).toMatch(
      /\[data-runtime-view="terminal"\] \.chat-md-inline-code\s*\{[\s\S]*?background:\s*rgba\(175, 184, 193, 0\.22\);[\s\S]*?color:\s*#24292f;/,
    );
    expect(codexLayer).toMatch(
      /\[data-runtime-view="terminal"\] \.chat-md-pre,[\s\S]*?background:\s*#f6f8fa;[\s\S]*?color:\s*#24292f;/,
    );
    expect(codexLayer).not.toContain("background: #111315;");
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

  it("keeps terminal command bubbles from collapsing into character-by-character wraps on mobile", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat-terminal.css"), "utf8");
    const block = stylesheet.match(/\.terminal-log-text\s*\{([\s\S]*?)\n\}/)?.[1] || "";

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

  it("keeps the mobile runtime composer above jump controls in the primary shell stylesheet", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-screen,\s*\[data-runtime-view="terminal"\] \.runtime-workspace-screen\s*\{[\s\S]*?height:\s*100%;[\s\S]*?max-height:\s*100%;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-shell\s*\{[\s\S]*?position:\s*relative;[\s\S]*?z-index:\s*24;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.terminal-jump-cluster\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?bottom:\s*calc\(var\(--runtime-composer-rest-inset, var\(--runtime-composer-inset, 0px\)\) \+ 24px\);[\s\S]*?right:\s*12px;/,
    );
    expect(stylesheet).toMatch(/\.terminal-jump-control\s*\{[\s\S]*?border-radius:\s*999px;/);
    expect(stylesheet).not.toContain("transition: bottom 220ms cubic-bezier(0.22, 1, 0.36, 1);");
  });

  it("keeps mobile terminal layouts on a single main surface instead of stacking nested rounded capsules", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(resolve(currentDirectory, "../../styles/shell.css"), "utf8");

    expect(stylesheet).toMatch(
      /@media \(max-width: 1100px\) \{[\s\S]*?\[data-runtime-view="conversation"\] \.runtime-workspace-body,\s*\[data-runtime-view="terminal"\] \.runtime-workspace-body\s*\{[\s\S]*?border-radius:\s*0;[\s\S]*?border-left:\s*0;[\s\S]*?border-right:\s*0;[\s\S]*?box-shadow:\s*none;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-form\s*\{[\s\S]*?border-radius:\s*26px;/,
    );
    expect(stylesheet).toMatch(
      /@media \(max-width: 760px\) \{[\s\S]*?\.runtime-composer-submit\s*\{[\s\S]*?box-shadow:\s*none;/,
    );
  });
});
