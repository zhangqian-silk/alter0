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
});
