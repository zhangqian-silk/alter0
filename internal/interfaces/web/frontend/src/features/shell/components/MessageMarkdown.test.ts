import { renderMessageMarkdownToHTML } from "./MessageMarkdown";
import { conversationMarkdownSyntaxFixture } from "./MessageMarkdownSyntaxFixture";

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

  it("renders pipe-delimited markdown tables as structured table blocks", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML([
      "四个产品的用户记忆管理可以归成四种风格：",
      "",
      "| 产品 | 记忆模型 | 管理方式 | 关键取法 |",
      "| --- | --- | --- | --- |",
      "| ChatGPT | 平台托管的用户记忆 + 历史引用 | `Saved Memories` 是长期记忆；`Reference Chat History` 从过往对话中取相关上下文 | 用户可查看、删除、关闭临时聊天绕过记忆 |",
      "| Claude Code | 本地 Markdown 指令 + 自动项目记忆 | `CLAUDE.md`、`.claude/rules/`、`~/.claude/CLAUDE.md` 管长期规则 | `/memory` 可审计 |",
      "",
      "表格后继续渲染普通段落。",
    ].join("\n"));

    const table = container.querySelector(".chat-md-table");
    expect(table).not.toBeNull();
    expect(container.querySelector(".chat-md-table-wrap")).toContainElement(table);
    expect(container.querySelectorAll("thead th")).toHaveLength(4);
    expect(container.querySelectorAll("tbody tr")).toHaveLength(2);
    expect(container.querySelector("tbody td code")).toHaveTextContent("Saved Memories");
    expect(container.textContent).toContain("表格后继续渲染普通段落。");
    expect(container.innerHTML).not.toContain("| --- | --- |");
  });

  it("preserves nested list hierarchy from markdown indentation", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML([
      "- Maintenance",
      "  - Memory maintenance",
      "  - Session cleanup",
      "- Safety",
      "  1. Pinned sessions excluded",
      "  2. Cleanup remains auditable",
    ].join("\n"));

    const topLevelList = container.querySelector(":scope > ul");
    expect(topLevelList).not.toBeNull();
    expect(topLevelList?.children).toHaveLength(2);
    expect(topLevelList?.children.item(0)?.firstChild?.textContent?.trim()).toBe("Maintenance");
    expect(topLevelList?.children.item(1)?.firstChild?.textContent?.trim()).toBe("Safety");

    const maintenanceSublist = topLevelList?.children.item(0)?.querySelector(":scope > ul");
    expect(maintenanceSublist).not.toBeNull();
    expect(maintenanceSublist?.children).toHaveLength(2);
    expect(maintenanceSublist?.children.item(0)).toHaveTextContent("Memory maintenance");
    expect(maintenanceSublist?.children.item(1)).toHaveTextContent("Session cleanup");

    const safetySublist = topLevelList?.children.item(1)?.querySelector(":scope > ol");
    expect(safetySublist).not.toBeNull();
    expect(safetySublist?.children).toHaveLength(2);
    expect(safetySublist?.children.item(0)).toHaveTextContent("Pinned sessions excluded");
    expect(safetySublist?.children.item(1)).toHaveTextContent("Cleanup remains auditable");
  });

  it("renders reference markdown semantics used by ChatGPT-style answers", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML([
      "Setext 一级标题",
      "===",
      "",
      "Setext 二级标题",
      "---",
      "",
      "普通段落包含 ~~删除线~~ 和 https://example.com 与 test@example.com。",
      "",
      "| 左对齐 | 居中对齐 | 右对齐 |",
      "| :--- | :---: | ---: |",
      "| A1 | B1 | C1 |",
      "",
      "1. 一级有序项",
      "   - 无序子项",
      "   - 另一个无序子项，包含 [链接](https://example.com/docs)",
      "     > 子项中的引用",
      "     ```python",
      "     print(\"nested markdown\")",
      "     ```",
      "2. 第二个有序项",
      "",
      "- [x] 已完成任务",
      "- [ ] 未完成任务",
    ].join("\n"));

    expect(container.querySelector("h1")).toHaveTextContent("Setext 一级标题");
    expect(container.querySelector("h2")).toHaveTextContent("Setext 二级标题");
    expect(container.querySelector("del")).toHaveTextContent("删除线");
    expect(container.querySelector('a[href="https://example.com"]')).toHaveTextContent("https://example.com");
    expect(container.querySelector('a[href="mailto:test@example.com"]')).toHaveTextContent("test@example.com");

    const table = container.querySelector(".chat-md-table") as HTMLTableElement | null;
    expect(table).not.toBeNull();
    expect(table?.querySelector("th:nth-child(1)")).toHaveStyle({ textAlign: "left" });
    expect(table?.querySelector("th:nth-child(2)")).toHaveStyle({ textAlign: "center" });
    expect(table?.querySelector("th:nth-child(3)")).toHaveStyle({ textAlign: "right" });
    expect(table?.querySelector("td:nth-child(3)")).toHaveStyle({ textAlign: "right" });

    const topOrderedList = container.querySelector(":scope > ol");
    expect(topOrderedList?.children).toHaveLength(2);
    const nestedUnorderedList = topOrderedList?.children.item(0)?.querySelector(":scope > ul");
    expect(nestedUnorderedList?.children).toHaveLength(2);
    expect(nestedUnorderedList?.children.item(1)?.querySelector("blockquote")).toHaveTextContent("子项中的引用");
    expect(nestedUnorderedList?.children.item(1)?.querySelector(".chat-md-pre code.language-python")).toHaveTextContent(
      'print("nested markdown")',
    );

    const taskList = Array.from(container.querySelectorAll(":scope > ul")).find((list) =>
      list.textContent?.includes("已完成任务"),
    );
    expect(taskList).not.toBeUndefined();
    expect(taskList).toHaveTextContent("已完成任务");
    expect(taskList).toHaveTextContent("未完成任务");
    expect(taskList?.textContent).not.toContain("[x]");
    expect(taskList?.textContent).not.toContain("[ ]");
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

  it("renders the comprehensive conversation markdown syntax fixture", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML(conversationMarkdownSyntaxFixture.markdown);

    for (const headingLevel of [1, 2, 3, 4, 5, 6]) {
      expect(container.querySelector(`h${headingLevel}`)).toHaveTextContent(
        conversationMarkdownSyntaxFixture.expected.headings[headingLevel - 1],
      );
    }

    expect(container.querySelectorAll("p")).toHaveLength(conversationMarkdownSyntaxFixture.expected.paragraphCount);
    expect(container.innerHTML).toContain("<br>");
    expect(container.querySelectorAll("strong")).toHaveLength(conversationMarkdownSyntaxFixture.expected.strongCount);
    expect(container.querySelectorAll("em")).toHaveLength(conversationMarkdownSyntaxFixture.expected.emphasisCount);
    expect(container.querySelectorAll(".chat-md-inline-code")).toHaveLength(
      conversationMarkdownSyntaxFixture.expected.inlineCodeCount,
    );
    expect(container.querySelectorAll("a[href]")).toHaveLength(conversationMarkdownSyntaxFixture.expected.linkCount);
    expect(container.querySelectorAll(".assistant-inline-image")).toHaveLength(
      conversationMarkdownSyntaxFixture.expected.imageCount,
    );
    expect(container.querySelectorAll("blockquote")).toHaveLength(conversationMarkdownSyntaxFixture.expected.blockquoteCount);
    expect(container.querySelectorAll("ul")).toHaveLength(conversationMarkdownSyntaxFixture.expected.unorderedListCount);
    expect(container.querySelectorAll("ol")).toHaveLength(conversationMarkdownSyntaxFixture.expected.orderedListCount);
    expect(container.querySelectorAll("hr")).toHaveLength(conversationMarkdownSyntaxFixture.expected.horizontalRuleCount);
    expect(container.querySelectorAll(".chat-md-pre")).toHaveLength(conversationMarkdownSyntaxFixture.expected.codeBlockCount);
    expect(container.querySelectorAll(".chat-md-table")).toHaveLength(conversationMarkdownSyntaxFixture.expected.tableCount);
    expect(Array.from(container.querySelectorAll(".chat-md-table tbody td"), (cell) => cell.textContent)).toContain("A | B");
    expect(container.textContent).toContain("Raw HTML stays text: <mark>not trusted</mark>");
    expect(container.innerHTML).not.toContain("<mark>not trusted</mark>");
  });

  it("includes deployed markdown demo data for nested list and mixed content validation", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML(conversationMarkdownSyntaxFixture.markdown);

    expect(container.querySelector("h1")).toHaveTextContent("H1 一级标题");
    expect(Array.from(container.querySelectorAll("h2"), (heading) => heading.textContent)).toContain("Setext 二级标题");

    const unorderedList = Array.from(container.querySelectorAll(":scope > ul")).find((list) =>
      list.textContent?.includes("项目 B"),
    );
    expect(unorderedList).not.toBeUndefined();
    expect(unorderedList?.querySelector(":scope > li:nth-child(2) > ul > li:nth-child(2) > ul > li")).toHaveTextContent(
      "更深层项目",
    );

    const mixedOrderedList = Array.from(container.querySelectorAll(":scope > ol")).find((list) =>
      list.textContent?.includes("一级有序项"),
    );
    expect(mixedOrderedList).not.toBeUndefined();
    expect(mixedOrderedList?.querySelector("blockquote")).toHaveTextContent("子项中的引用");
    expect(mixedOrderedList?.querySelector(".chat-md-pre code.language-python")).toHaveTextContent(
      'print("nested markdown")',
    );

    const htmlCodeBlocks = Array.from(container.querySelectorAll(".chat-md-pre code.language-html"));
    expect(htmlCodeBlocks[0]?.textContent).toBe("<details>\n<summary>点击展开</summary>");
    expect(htmlCodeBlocks[1]?.textContent).toBe("</details>");
    expect(Array.from(container.querySelectorAll("p"), (paragraph) => paragraph.textContent)).not.toContain("<details>");

    expect(container.querySelector(".chat-md-table th:nth-child(2)")).toHaveStyle({ textAlign: "center" });
    expect(container.querySelector('a[href="https://example.com"]')).toHaveTextContent("https://example.com");
  });

  it("includes deployed markdown demo table data for short, long, and overflow-prone cells", () => {
    const container = document.createElement("div");
    container.innerHTML = renderMessageMarkdownToHTML(conversationMarkdownSyntaxFixture.markdown);

    expect(container.textContent).toContain("短字符表格：");
    expect(container.textContent).toContain("长中文表格：");
    expect(container.textContent).toContain("长 URL 与代码表格：");

    const tables = Array.from(container.querySelectorAll(".chat-md-table"));
    expect(tables).toHaveLength(conversationMarkdownSyntaxFixture.expected.tableCount);
    expect(tables.some((table) => table.textContent?.includes("短"))).toBe(true);
    expect(tables.some((table) => table.textContent?.includes("这是一段较长的中文单元格内容"))).toBe(true);
    expect(tables.some((table) => table.textContent?.includes("/api/conversation-runtime/sessions"))).toBe(true);
    expect(container.querySelector(".chat-md-table code")).toHaveTextContent("session_id=abcd1234&markdown_demo=1");
    expect(container.querySelector('a[href="https://example.com/docs/very/long/path?query=markdown-table-demo"]')).toHaveTextContent(
      "长链接",
    );
  });
});
