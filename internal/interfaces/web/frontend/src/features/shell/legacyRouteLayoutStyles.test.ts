import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

describe("legacy route layout stylesheet", () => {
  it("drops the chat split panes to a single column before the shell switches to drawer mode", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-routes.css"),
      "utf8",
    );

    expect(stylesheet).toContain("@media (max-width: 1100px) {");
    expect(stylesheet).toContain(".skill-route-layout {");
    expect(stylesheet).toContain("grid-template-columns: minmax(0, 1fr);");
    expect(stylesheet).toContain(".skill-workspace {");
    expect(stylesheet).toContain("grid-template-rows: auto minmax(260px, 1fr) auto;");
  });

  it("keeps skill route primitives on the restrained workbench surface system", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-routes.css"),
      "utf8",
    );

    expect(stylesheet).toContain(".skill-route-card {");
    expect(stylesheet).toContain("border: 1px solid rgba(15, 23, 42, 0.08);");
    expect(stylesheet).toContain("background: rgba(255, 255, 255, 0.94);");
  });

  it("keeps task follow-up image attachments inside a stacked composer row", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-routes.css"),
      "utf8",
    );

    expect(stylesheet).toContain(".control-task-terminal-input {");
    expect(stylesheet).toContain("flex-direction: column;");
    expect(stylesheet).toContain(".control-task-terminal-input-row {");
    expect(stylesheet).toContain(".control-task-terminal-upload {");
  });

  it("styles shared runtime markdown shells in chat-core without relying on assistant-specific wrappers", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-core.css"),
      "utf8",
    );

    expect(stylesheet).toContain(".message-markdown-shell {");
    expect(stylesheet).toContain(".message-markdown-toolbar {");
    expect(stylesheet).toContain(".message-markdown-copy {");
    expect(stylesheet).toContain(".message-markdown-body {");
    expect(stylesheet).toContain(".conversation-process-step-body > .message-markdown-rendered > :first-child,");
    expect(stylesheet).toContain(".conversation-process-answer > .message-markdown-rendered > :last-child {");
  });

  it("keeps skill process detail content readable on narrow screens", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-core.css"),
      "utf8",
    );

    expect(stylesheet).toContain(".conversation-process-step-head > span:last-child,");
    expect(stylesheet).toContain(".conversation-process-step-head > span:last-child {");
    expect(stylesheet).toContain(".conversation-process-step-body > .message-markdown-rendered,");
    expect(stylesheet).toContain(".conversation-process-step-body > .message-markdown-rendered {");
    expect(stylesheet).toContain("@media (max-width: 760px) {");
    expect(stylesheet).toContain(".conversation-process-step-body {");
    expect(stylesheet).toContain("width: 100%;");
  });

  it("keeps conversation process step indices vertically centered with titles", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-core.css"),
      "utf8",
    );
    const headBlock = stylesheet.match(/\.conversation-process-step-head,\s*\.conversation-process-step-head\s*\{([\s\S]*?)\n\}/)?.[1] || "";
    const metaSlotBlock = stylesheet.match(/\.conversation-process-step-head\s*>\s*span:first-child,\s*\.conversation-process-step-head\s*>\s*span:first-child\s*\{([\s\S]*?)\n\}/)?.[1] || "";
    const mobileBlock = stylesheet.match(/@media \(max-width: 760px\) \{[\s\S]*?\.conversation-process-step-head,\s*\.conversation-process-step-head\s*\{([\s\S]*?)\n  \}/)?.[1] || "";

    expect(headBlock).toContain("align-items: center;");
    expect(metaSlotBlock).toContain("display: inline-flex;");
    expect(metaSlotBlock).toContain("align-items: center;");
    expect(metaSlotBlock).toContain("justify-content: center;");
    expect(mobileBlock).toContain("align-items: center;");
    expect(mobileBlock).not.toContain("align-items: flex-start;");
  });

  it("styles terminal final output through runtime markdown wrappers", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-terminal.css"),
      "utf8",
    );

    expect(stylesheet).toContain(".terminal-final-text .message-markdown-toolbar,");
    expect(stylesheet).toContain(".terminal-final-rendered .message-markdown-rendered > :first-child {");
    expect(stylesheet).toContain(".terminal-final-rendered .message-markdown-rendered > :last-child {");
  });

  it("version-busts legacy runtime stylesheets so repeated preview deploys cannot reuse stale process CSS", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const html = readFileSync(resolve(currentDirectory, "../../../index.html"), "utf8");
    const legacyEntry = readFileSync(resolve(currentDirectory, "../../../public/legacy/chat.css"), "utf8");

    expect(html).toContain('/legacy/chat.css?v=20260429-terminal-process-wrap');
    expect(legacyEntry).toContain('@import url("./chat-core.css?v=20260429-terminal-process-wrap");');
    expect(legacyEntry).toContain('@import url("./chat-routes.css?v=20260429-terminal-process-wrap");');
    expect(legacyEntry).toContain('@import url("./chat-terminal.css?v=20260429-terminal-process-wrap");');
  });
});
