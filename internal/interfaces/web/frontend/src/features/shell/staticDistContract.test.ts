import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

describe("static dist contract", () => {
  it("keeps the committed web assets aligned with shared markdown table rendering", () => {
    const currentDirectory = dirname(fileURLToPath(import.meta.url));
    const distDirectory = resolve(currentDirectory, "../../../../static/dist");
    const indexHTML = readFileSync(resolve(distDirectory, "index.html"), "utf8");
    const scriptPath = indexHTML.match(/src="\/(assets\/index-[^"]+\.js)"/)?.[1];
    const stylesheetPath = indexHTML.match(/href="\/(assets\/index-[^"]+\.css)"/)?.[1];

    expect(scriptPath).toBeTruthy();
    expect(stylesheetPath).toBeTruthy();

    const script = readFileSync(resolve(distDirectory, scriptPath || ""), "utf8");
    const stylesheet = readFileSync(resolve(distDirectory, stylesheetPath || ""), "utf8");

    expect(script).toContain("chat-md-table-wrap");
    expect(script).toContain("chat-md-table");
    expect(stylesheet).toContain(".chat-md-table-wrap");
    expect(stylesheet).toContain(".chat-md-table");
  });
});
