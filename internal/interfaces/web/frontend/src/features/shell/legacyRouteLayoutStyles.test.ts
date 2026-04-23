import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

describe("legacy route layout stylesheet", () => {
  it("drops the agent runtime split panes to a single column before the shell switches to drawer mode", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-routes.css"),
      "utf8",
    );

    expect(stylesheet).toContain("@media (max-width: 1100px) {");
    expect(stylesheet).toContain(".agent-route-layout {");
    expect(stylesheet).toContain("grid-template-columns: minmax(0, 1fr);");
    expect(stylesheet).toContain(".agent-workspace {");
    expect(stylesheet).toContain("grid-template-rows: auto minmax(260px, 1fr) auto;");
  });

  it("keeps agent route primitives on the restrained workbench surface system", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const stylesheet = readFileSync(
      resolve(currentDirectory, "../../../public/legacy/chat-routes.css"),
      "utf8",
    );

    expect(stylesheet).toContain(".agent-route-card {");
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
});
