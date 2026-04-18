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
});
