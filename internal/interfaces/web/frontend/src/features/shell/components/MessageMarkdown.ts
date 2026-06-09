import MarkdownIt from "markdown-it";

const markdownRenderer = createMessageMarkdownRenderer();

export function renderMessageMarkdownToHTML(value: string) {
  const normalized = normalizeMessageMarkdownInput(value);
  if (!normalized.trim()) {
    return "";
  }
  return markdownRenderer.render(prepareMessageMarkdownForRendering(normalized)).trim();
}

function createMessageMarkdownRenderer() {
  const renderer = new MarkdownIt({
    breaks: true,
    html: false,
    linkify: true,
  });
  const defaultValidateLink = renderer.validateLink.bind(renderer);

  renderer.validateLink = (url: string) => {
    const normalized = normalizeMarkdownURL(url);
    if (/^data:image\//i.test(normalized)) {
      return true;
    }
    return defaultValidateLink(url);
  };

  renderer.renderer.rules.fence = (tokens: MarkdownIt.Token[], index: number) => {
    const token = tokens[index];
    const language = normalizeMarkdownCodeLanguage(token.info);
    const languageClass = language ? ` class="language-${escapeHTML(language)}"` : "";
    return `<pre class="chat-md-pre"><code${languageClass}>${escapeHTML(decodeHTMLEntities(token.content).replace(/\n$/, ""))}</code></pre>`;
  };

  renderer.renderer.rules.code_block = (tokens: MarkdownIt.Token[], index: number) =>
    `<pre class="chat-md-pre"><code>${escapeHTML(decodeHTMLEntities(tokens[index].content).replace(/\n$/, ""))}</code></pre>`;

  renderer.renderer.rules.code_inline = (tokens: MarkdownIt.Token[], index: number) =>
    `<code class="chat-md-inline-code">${escapeHTML(decodeHTMLEntities(tokens[index].content))}</code>`;

  renderer.renderer.rules.s_open = () => "<del>";
  renderer.renderer.rules.s_close = () => "</del>";

  renderer.renderer.rules.link_open = (tokens: MarkdownIt.Token[], index: number, options, env, self) => {
    const token = tokens[index];
    const href = sanitizeMarkdownURL(token.attrGet("href") || "");
    if (!href) {
      token.attrSet("href", "#");
    } else {
      token.attrSet("href", href);
    }
    token.attrSet("target", "_blank");
    token.attrSet("rel", "noreferrer noopener");
    return self.renderToken(tokens, index, options);
  };

  renderer.renderer.rules.image = (tokens: MarkdownIt.Token[], index: number) => {
    const token = tokens[index];
    const src = sanitizeMarkdownImageURL(token.attrGet("src") || "");
    const alt = token.content.trim() || token.attrGet("alt") || "Generated image";
    if (!src) {
      return escapeHTML(alt);
    }
    const safeAlt = escapeHTML(alt);
    return [
      `<a class="assistant-inline-image-link" href="${src}" target="_blank" rel="noreferrer noopener">`,
      `<img class="assistant-inline-image" src="${src}" alt="${safeAlt}" loading="lazy" decoding="async" />`,
      "</a>",
    ].join("");
  };

  renderer.renderer.rules.table_open = () => '<div class="chat-md-table-wrap"><table class="chat-md-table">';
  renderer.renderer.rules.table_close = () => "</table></div>";

  return renderer;
}

function prepareMessageMarkdownForRendering(value: string) {
  return stripTaskListMarkers(dropUnsafeMarkdownLinks(decodeHTMLEntities(value)));
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

function stripTaskListMarkers(value: string) {
  return String(value || "").replace(/^([ \t]*[-*+]\s+)\[(?:x|X| )\]\s+/gm, "$1");
}

function dropUnsafeMarkdownLinks(value: string) {
  return String(value || "")
    .replace(/!\[([^\]]*)\]\(((?:[^()]|\([^)]*\))+)\)/g, (match, altText: string, url: string) =>
      sanitizeMarkdownImageURL(url) ? match : altText,
    )
    .replace(
      /\[([^\]]+)\]\(((?:[^()]|\([^)]*\))+)\)/g,
      (match, label: string, url: string, offset: number, input: string) => {
        if (input[offset - 1] === "!") {
          return match;
        }
        return sanitizeMarkdownURL(url) ? match : label;
      },
    );
}

function normalizeMarkdownCodeLanguage(value: string) {
  return String(value || "")
    .trim()
    .split(/\s+/)[0]
    .toLowerCase()
    .replace(/[^a-z0-9_-]/g, "");
}

function sanitizeMarkdownURL(rawURL: string) {
  const normalized = normalizeMarkdownURL(rawURL);
  if (/^(https?:|mailto:)/i.test(normalized) || normalized.startsWith("/") || normalized.startsWith("#")) {
    return normalized;
  }
  return "";
}

function sanitizeMarkdownImageURL(rawURL: string) {
  const normalized = normalizeMarkdownURL(rawURL);
  if (/^data:image\//i.test(normalized)) {
    return normalized;
  }
  return sanitizeMarkdownURL(normalized);
}

function normalizeMarkdownURL(rawURL: string) {
  return String(rawURL || "").trim().replace(/^<|>$/g, "");
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
