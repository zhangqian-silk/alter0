import { renderRuntimeMarkdownToHTML } from "./RuntimeMarkdown";

describe("renderRuntimeMarkdownToHTML", () => {
  it("renders markdown images, links, and fenced code through the shared runtime contract", () => {
    const html = renderRuntimeMarkdownToHTML([
      "# Runtime",
      "",
      "![Diagram](https://cdn.example.com/runtime-diagram.png)",
      "",
      "Visit [workspace](/chat).",
      "",
      "```ts",
      "const ready = true;",
      "```",
    ].join("\n"));

    expect(html).toContain('class="assistant-inline-image-link"');
    expect(html).toContain('class="assistant-inline-image"');
    expect(html).toContain('href="/chat"');
    expect(html).toContain('<pre class="chat-md-pre"><code class="language-ts">');
    expect(html).toContain("const ready = true;");
  });

  it("drops unsafe urls while preserving readable fallback content", () => {
    const html = renderRuntimeMarkdownToHTML([
      "[bad](javascript:alert(1))",
      "",
      "![oops](javascript:alert(2))",
    ].join("\n"));

    expect(html).not.toContain("javascript:");
    expect(html).toContain("<p>bad</p>");
    expect(html).toContain("<p>oops</p>");
  });
});
