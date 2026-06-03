import { renderMessageMarkdownToHTML } from "./MessageMarkdown";

describe("renderMessageMarkdownToHTML", () => {
  it("renders markdown images, links, and fenced code through the shared message markdown contract", () => {
    const html = renderMessageMarkdownToHTML([
      "# Message",
      "",
      "![Diagram](https://cdn.example.com/message-diagram.png)",
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
    const html = renderMessageMarkdownToHTML([
      "[bad](javascript:alert(1))",
      "",
      "![oops](javascript:alert(2))",
    ].join("\n"));

    expect(html).not.toContain("javascript:");
    expect(html).toContain("<p>bad</p>");
    expect(html).toContain("<p>oops</p>");
  });

  it("decodes html entities into readable text before rendering", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML("Chat &gt; Details &gt; Model &amp; Quota");

    expect(container.textContent).toContain("Chat > Details > Model & Quota");
    expect(container.textContent).not.toContain("&gt;");
  });

  it("strips invisible break characters that would otherwise split process text at every glyph", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML("下\u200B面\u200B给\u200B出\u200B一\u200B条\u200B接\u200B入\u200B路\u200B径");

    expect(container.textContent).toContain("下面给出一条接入路径");
    expect(container.textContent).not.toContain("\u200B");
  });

  it("collapses pathological single-glyph line breaks back into a readable paragraph", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML([
      "下",
      "面",
      "给",
      "出",
      "一",
      "条",
      "可",
      "落",
      "地",
      "的",
      "接",
      "入",
      "路",
      "径",
    ].join("\n"));

    expect(container.textContent).toContain("下面给出一条可落地的接入路径");
    expect(container.querySelectorAll("p")).toHaveLength(1);
  });
});
