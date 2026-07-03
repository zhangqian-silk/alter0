import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

describe("message markdown rendering contract", () => {
  it("uses one shared generic markdown shell for chat", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const sourceRoot = resolve(currentDirectory, "../..");
    const files = [
      "features/shell/components/RuntimeTimeline.tsx",
      "features/shell/components/ChatMessageRegion.tsx",
      "features/shell/components/MessageMarkdown.ts",
      "features/shell/components/MessageMarkdownShell.tsx",
      "features/shell/shellLayoutStyles.test.ts",
      "styles/shell.css",
    ];
    const source = files.map((file) => readFileSync(resolve(sourceRoot, file), "utf8")).join("\n");

    expect(source).not.toContain("dialog-markdown");
    expect(source).not.toContain("runtime-dialog-markdown");
    expect(source).not.toContain("RuntimeDialogMarkdown");
    expect(source).not.toContain("RuntimeMarkdown");
    expect(source).not.toContain("runtime-markdown");
    expect(existsSync(resolve(sourceRoot, "features/shell/components/RuntimeDialogMarkdown.tsx"))).toBe(false);
    expect(existsSync(resolve(sourceRoot, "features/shell/components/RuntimeMarkdown.ts"))).toBe(false);
  });
});
