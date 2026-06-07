export function renderMessageMarkdownToHTML(value: string) {
  const normalized = normalizeMessageMarkdownInput(value);
  if (!normalized.trim()) {
    return "";
  }
  const tokens: Array<{ type: "markdown" | "code"; content: string; language?: string }> = [];
  const fencePattern = /```([a-zA-Z0-9_-]+)?\n([\s\S]*?)```/g;
  let cursor = 0;
  let match = fencePattern.exec(normalized);
  while (match) {
    if (match.index > cursor) {
      tokens.push({ type: "markdown", content: normalized.slice(cursor, match.index) });
    }
    tokens.push({
      type: "code",
      language: String(match[1] || "").trim().toLowerCase(),
      content: String(match[2] || "").replace(/\n$/, ""),
    });
    cursor = match.index + match[0].length;
    match = fencePattern.exec(normalized);
  }
  if (cursor < normalized.length) {
    tokens.push({ type: "markdown", content: normalized.slice(cursor) });
  }
  return tokens
    .map((token) => {
      if (token.type === "code") {
        const languageClass = token.language ? ` class="language-${escapeHTML(token.language)}"` : "";
        return `<pre class="chat-md-pre"><code${languageClass}>${escapeHTML(decodeHTMLEntities(token.content))}</code></pre>`;
      }
      return renderMarkdownBlocks(token.content);
    })
    .join("");
}

function normalizeMessageMarkdownInput(value: string) {
  const normalized = String(value ?? "")
    .replace(/\r\n?/g, "\n")
    .replace(/[\u200B\u200C\u200D\u2060\uFEFF\u00AD]/g, "");

  return normalized
    .split(/\n{2,}/)
    .map((paragraph) => collapseSingleGlyphLineBreaks(paragraph))
    .join("\n\n");
}

function collapseSingleGlyphLineBreaks(paragraph: string) {
  const lines = String(paragraph || "").split("\n");
  const trimmedNonEmptyLines = lines.map((line) => line.trim()).filter(Boolean);
  if (trimmedNonEmptyLines.length < 6) {
    return paragraph;
  }
  const singleGlyphLines = trimmedNonEmptyLines.filter(isLikelySingleGlyphLine);
  if (singleGlyphLines.length / trimmedNonEmptyLines.length < 0.8) {
    return paragraph;
  }
  return trimmedNonEmptyLines.join("");
}

function isLikelySingleGlyphLine(line: string) {
  const trimmed = line.trim();
  if (!trimmed) {
    return false;
  }
  if (/^[-*+#>]/.test(trimmed) || /^\d+\./.test(trimmed)) {
    return false;
  }
  return Array.from(trimmed).length === 1;
}

type MarkdownTableBlock = {
  header: string[];
  rows: string[][];
  lineCount: number;
};

function renderMarkdownBlocks(content: string) {
  const lines = String(content || "").split("\n");
  const html: string[] = [];
  let paragraphLines: string[] = [];
  let quoteLines: string[] = [];
  let listType = "";
  let listItems: string[] = [];

  const flushParagraph = () => {
    if (!paragraphLines.length) {
      return;
    }
    html.push(`<p>${paragraphLines.map((line) => renderMarkdownInline(line)).join("<br>")}</p>`);
    paragraphLines = [];
  };

  const flushQuote = () => {
    if (!quoteLines.length) {
      return;
    }
    html.push(`<blockquote>${renderMarkdownBlocks(quoteLines.join("\n"))}</blockquote>`);
    quoteLines = [];
  };

  const flushList = () => {
    if (!listType || !listItems.length) {
      listType = "";
      listItems = [];
      return;
    }
    html.push(
      `<${listType}>${listItems
        .map((item) => `<li>${renderMarkdownInline(item)}</li>`)
        .join("")}</${listType}>`,
    );
    listType = "";
    listItems = [];
  };

  const flushAll = () => {
    flushParagraph();
    flushQuote();
    flushList();
  };

  for (let lineIndex = 0; lineIndex < lines.length; lineIndex += 1) {
    const rawLine = lines[lineIndex];
    const trimmed = rawLine.trim();
    if (!trimmed) {
      flushAll();
      continue;
    }

    if (/^>\s?/.test(trimmed)) {
      flushParagraph();
      flushList();
      quoteLines.push(trimmed.replace(/^>\s?/, ""));
      continue;
    }
    flushQuote();

    const unorderedMatch = /^[-*+]\s+(.+)$/.exec(trimmed);
    if (unorderedMatch) {
      flushParagraph();
      if (listType && listType !== "ul") {
        flushList();
      }
      listType = "ul";
      listItems.push(unorderedMatch[1]);
      continue;
    }

    const orderedMatch = /^(\d+)\.\s+(.+)$/.exec(trimmed);
    if (orderedMatch) {
      flushParagraph();
      if (listType && listType !== "ol") {
        flushList();
      }
      listType = "ol";
      listItems.push(orderedMatch[2]);
      continue;
    }

    flushList();

    const tableBlock = parseMarkdownTableBlock(lines, lineIndex);
    if (tableBlock) {
      flushParagraph();
      html.push(renderMarkdownTableBlock(tableBlock));
      lineIndex += tableBlock.lineCount - 1;
      continue;
    }

    const headingMatch = /^(#{1,6})\s+(.+)$/.exec(trimmed);
    if (headingMatch) {
      flushParagraph();
      const level = headingMatch[1].length;
      html.push(`<h${level}>${renderMarkdownInline(headingMatch[2])}</h${level}>`);
      continue;
    }

    if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph();
      html.push("<hr>");
      continue;
    }

    paragraphLines.push(trimmed);
  }

  flushAll();
  return html.join("");
}

function parseMarkdownTableBlock(lines: string[], startIndex: number): MarkdownTableBlock | null {
  const header = parseMarkdownTableRow(lines[startIndex]);
  const separator = parseMarkdownTableRow(lines[startIndex + 1] || "");
  if (!header || !separator || !isMarkdownTableSeparator(separator)) {
    return null;
  }

  const rows: string[][] = [];
  let cursor = startIndex + 2;
  while (cursor < lines.length) {
    const row = parseMarkdownTableRow(lines[cursor]);
    if (!row || isMarkdownTableSeparator(row)) {
      break;
    }
    rows.push(row);
    cursor += 1;
  }

  return {
    header,
    rows,
    lineCount: cursor - startIndex,
  };
}

function parseMarkdownTableRow(line: string) {
  const value = String(line || "").trim();
  if (!value.includes("|")) {
    return null;
  }

  const cells: string[] = [];
  let cell = "";
  for (let index = 0; index < value.length; index += 1) {
    const char = value[index];
    const nextChar = value[index + 1];
    if (char === "\\" && nextChar === "|") {
      cell += "|";
      index += 1;
      continue;
    }
    if (char === "|") {
      cells.push(cell.trim());
      cell = "";
      continue;
    }
    cell += char;
  }
  cells.push(cell.trim());

  if (cells[0] === "") {
    cells.shift();
  }
  if (cells[cells.length - 1] === "") {
    cells.pop();
  }

  return cells.length >= 2 ? cells : null;
}

function isMarkdownTableSeparator(cells: string[]) {
  return cells.length >= 2 && cells.every((cell) => /^:?-{3,}:?$/.test(cell.trim()));
}

function renderMarkdownTableBlock(table: MarkdownTableBlock) {
  const columnCount = table.header.length;
  const renderCell = (cell: string, tagName: "th" | "td") => `<${tagName}>${renderMarkdownInline(cell)}</${tagName}>`;
  const normalizeRow = (row: string[]) => Array.from({ length: columnCount }, (_, index) => row[index] || "");
  const header = normalizeRow(table.header).map((cell) => `<th scope="col">${renderMarkdownInline(cell)}</th>`).join("");
  const body = table.rows
    .map((row) => `<tr>${normalizeRow(row).map((cell) => renderCell(cell, "td")).join("")}</tr>`)
    .join("");

  return [
    '<div class="chat-md-table-wrap">',
    '<table class="chat-md-table">',
    `<thead><tr>${header}</tr></thead>`,
    `<tbody>${body}</tbody>`,
    "</table>",
    "</div>",
  ].join("");
}

function renderMarkdownInline(content: string) {
  let rendered = decodeHTMLEntities(String(content ?? ""));
  const placeholders: string[] = [];
  const markdownLinkPattern = /\[([^\]]+)\]\(((?:[^()]|\([^)]*\))+)\)/g;
  const markdownImagePattern = /!\[([^\]]*)\]\(((?:[^()]|\([^)]*\))+)\)/g;
  const reserve = (html: string) => {
    const token = `\u0000${placeholders.length}\u0000`;
    placeholders.push(html);
    return token;
  };

  rendered = rendered.replace(markdownImagePattern, (_, altText: string, url: string) => {
    const src = sanitizeMarkdownImageURL(url);
    if (!src) {
      return renderMarkdownInline(altText);
    }
    const alt = escapeHTML(altText.trim() || "Generated image");
    return reserve(
      `<a class="assistant-inline-image-link" href="${src}" target="_blank" rel="noreferrer noopener">`
      + `<img class="assistant-inline-image" src="${src}" alt="${alt}" loading="lazy" decoding="async" />`
      + "</a>",
    );
  });
  rendered = rendered.replace(/`([^`]+)`/g, (_, code: string) =>
    reserve(`<code class="chat-md-inline-code">${escapeHTML(code)}</code>`),
  );
  rendered = rendered.replace(markdownLinkPattern, (_, label: string, url: string) => {
    const href = sanitizeMarkdownURL(url);
    if (!href) {
      return renderMarkdownInline(label);
    }
    return reserve(
      `<a href="${href}" target="_blank" rel="noreferrer noopener">${renderMarkdownInline(label)}</a>`,
    );
  });
  rendered = rendered
    .split(/(\u0000\d+\u0000)/g)
    .map((part) => (/^\u0000\d+\u0000$/.test(part) ? part : escapeHTML(part)))
    .join("");
  rendered = rendered.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  rendered = rendered.replace(/__([^_]+)__/g, "<strong>$1</strong>");
  rendered = rendered.replace(/(^|[\s(>])\*([^*\n]+)\*(?=$|[\s).,!?:;<])/g, "$1<em>$2</em>");
  rendered = rendered.replace(/(^|[\s(>])_([^_\n]+)_(?=$|[\s).,!?:;<])/g, "$1<em>$2</em>");

  return rendered.replace(/\u0000(\d+)\u0000/g, (_, index: string) => placeholders[Number(index)] || "");
}

function sanitizeMarkdownURL(rawURL: string) {
  const value = String(rawURL || "").trim();
  if (!value) {
    return "";
  }
  const normalized = value.replace(/^<|>$/g, "");
  if (/^(https?:|mailto:)/i.test(normalized) || normalized.startsWith("/") || normalized.startsWith("#")) {
    return escapeHTML(normalized);
  }
  return "";
}

function sanitizeMarkdownImageURL(rawURL: string) {
  const value = String(rawURL || "").trim();
  if (!value) {
    return "";
  }
  const normalized = value.replace(/^<|>$/g, "");
  if (/^data:image\//i.test(normalized)) {
    return escapeHTML(normalized);
  }
  return sanitizeMarkdownURL(normalized);
}

function escapeHTML(value: string) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => {
    if (char === "&") return "&amp;";
    if (char === "<") return "&lt;";
    if (char === ">") return "&gt;";
    if (char === '"') return "&quot;";
    return "&#39;";
  });
}

function decodeHTMLEntities(value: string) {
  return String(value ?? "").replace(/&(#\d+|#x[0-9a-fA-F]+|[a-zA-Z]+);/g, (entity, token: string) => {
    const normalized = String(token || "").toLowerCase();
    switch (normalized) {
      case "amp":
        return "&";
      case "lt":
        return "<";
      case "gt":
        return ">";
      case "quot":
        return '"';
      case "apos":
      case "#39":
        return "'";
      case "nbsp":
        return " ";
      default:
        break;
    }
    if (normalized.startsWith("#x")) {
      const codePoint = Number.parseInt(normalized.slice(2), 16);
      return Number.isFinite(codePoint) ? String.fromCodePoint(codePoint) : entity;
    }
    if (normalized.startsWith("#")) {
      const codePoint = Number.parseInt(normalized.slice(1), 10);
      return Number.isFinite(codePoint) ? String.fromCodePoint(codePoint) : entity;
    }
    return entity;
  });
}
