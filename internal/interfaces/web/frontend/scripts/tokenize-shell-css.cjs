#!/usr/bin/env node
/**
 * One-shot helper: tokenize hardcoded colors in shell.css.
 *
 * Reads the file, applies a series of context-aware replacements,
 * and writes the result back.  Safe to re-run — already-tokenized
 * values (containing `var(`) are left alone.
 *
 * Usage:
 *   node scripts/tokenize-shell-css.js [path/to/shell.css]
 */

const fs = require("fs");
const path = require("path");

const DEFAULT_PATH = path.join(
  __dirname,
  "..",
  "internal",
  "interfaces",
  "web",
  "frontend",
  "src",
  "styles",
  "shell.css",
);

/* ── Replacement rules ────────────────────────────────────── */

/**
 * Each rule:
 *   pattern:   regex to match (must include the full value to replace)
 *   token:     the CSS variable to use
 *   context:   optional regex that the surrounding line must match
 *              (used for context-dependent values like #ffffff)
 */

// ── Unambiguous rgba replacements ──
const rgbaRules = [
  // Borders
  { from: /rgba\(15,\s*23,\s*42,\s*0\.0[6-9]\)/g, to: "var(--border-subtle)" },
  { from: /rgba\(15,\s*23,\s*42,\s*0\.1[0-5]\)/g, to: "var(--border-default)" },
  { from: /rgba\(15,\s*23,\s*42,\s*0\.1[6-9]\)/g, to: "var(--border-strong)" },

  // Accent subtle
  { from: /rgba\(15,\s*159,\s*143,\s*0\.0[6-9]\)/g, to: "var(--accent-subtle)" },
  { from: /rgba\(15,\s*159,\s*143,\s*0\.1[0-5]\)/g, to: "var(--accent-subtle-strong)" },
  { from: /rgba\(15,\s*159,\s*143,\s*0\.38\)/g, to: "var(--accent-ring)" },

  // Elevated surfaces
  { from: /rgba\(255,\s*255,\s*255,\s*0\.9[6-9]\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(255,\s*255,\s*255,\s*0\.78\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(255,\s*255,\s*255,\s*0\.9[0-5]\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(255,\s*255,\s*255,\s*0\.9\)/g, to: "var(--bg-elevated)" },

  // Overlay
  { from: /rgba\(248,\s*250,\s*252,\s*0\.92\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(226,\s*232,\s*240,\s*0\.86\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(226,\s*232,\s*240,\s*0\.96\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(239,\s*246,\s*255,\s*0\.98\)/g, to: "var(--bg-elevated)" },

  // Status error bg
  { from: /rgba\(180,\s*35,\s*24,\s*0\.0[6-9]\)/g, to: "var(--status-error-bg)" },

  // Status info bg
  { from: /rgba\(37,\s*99,\s*235,\s*0\.0[6-9]\)/g, to: "var(--status-info-bg)" },
  { from: /rgba\(37,\s*99,\s*235,\s*0\.1[0-9]\)/g, to: "var(--status-info-bg)" },
  { from: /rgba\(37,\s*99,\s*235,\s*0\.2[0-9]\)/g, to: "var(--status-info-bg)" },

  // Shadow rgba (for the shadow tokens — these are complex, skip for now)

  // Additional border/line rgba patterns
  { from: /rgba\(15,\s*23,\s*42,\s*0\.0[4-5]\)/g, to: "var(--border-subtle)" },
  { from: /rgba\(15,\s*23,\s*42,\s*0\.2[0-9]\)/g, to: "var(--border-strong)" },
  { from: /rgba\(15,\s*23,\s*42,\s*0\.3\)/g, to: "var(--border-strong)" },

  // Accent-related rgba
  { from: /rgba\(15,\s*159,\s*143,\s*0\.2[0-9]\)/g, to: "var(--accent-ring)" },
  { from: /rgba\(15,\s*159,\s*143,\s*0\.3[0-9]\)/g, to: "var(--accent-ring)" },
  { from: /rgba\(15,\s*159,\s*143,\s*0\.1[4-9]\)/g, to: "var(--accent-subtle-strong)" },

  // White-based overlays and glass effects
  { from: /rgba\(255,\s*255,\s*255,\s*0\.7[0-9]\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(255,\s*255,\s*255,\s*0\.8[0-9]\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(255,\s*255,\s*255,\s*0\.6[0-9]\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(255,\s*255,\s*255,\s*0\.5\)/g, to: "var(--bg-elevated)" },
  { from: /rgba\(255,\s*255,\s*255,\s*0\.4[0-9]\)/g, to: "var(--bg-overlay)" },

  // Status error rgba
  { from: /rgba\(180,\s*35,\s*24,\s*0\.1[0-9]\)/g, to: "var(--status-error-bg)" },
  { from: /rgba\(220,\s*38,\s*38,\s*0\.[0-9]+\)/g, to: "var(--status-error-bg)" },
  { from: /rgba\(239,\s*68,\s*68,\s*0\.[0-9]+\)/g, to: "var(--status-error-bg)" },

  // Status success rgba
  { from: /rgba\(22,\s*163,\s*74,\s*0\.[0-9]+\)/g, to: "var(--status-success-bg)" },
  { from: /rgba\(101,\s*163,\s*13,\s*0\.[0-9]+\)/g, to: "var(--status-success-bg)" },

  // Status warning rgba
  { from: /rgba\(245,\s*158,\s*11,\s*0\.[0-9]+\)/g, to: "var(--status-warning-bg)" },
  { from: /rgba\(217,\s*119,\s*6,\s*0\.[0-9]+\)/g, to: "var(--status-warning-bg)" },

  // Status info / blue rgba
  { from: /rgba\(96,\s*165,\s*250,\s*0\.[0-9]+\)/g, to: "var(--status-info-bg)" },
  { from: /rgba\(59,\s*130,\s*246,\s*0\.[0-9]+\)/g, to: "var(--status-info-bg)" },

  // Generic slate/ink rgba (used in shadows, borders)
  { from: /rgba\(15,\s*23,\s*42,\s*0\.[4-9]\)/g, to: "var(--shadow-lg)" },
  { from: /rgba\(15,\s*23,\s*42,\s*0\.3[0-9]\)/g, to: "var(--shadow-md)" },
];

// ── Hex replacements (unambiguous) ──
const hexRules = [
  // Accent colors
  { from: /#0f9f8f/gi, to: "var(--accent)" },
  { from: /#087f73/gi, to: "var(--accent-hover)" },

  // Error
  { from: /#b42318/gi, to: "var(--status-error)" },

  // Text colors
  { from: /#111827/gi, to: "var(--text-primary)" },
  { from: /#0f172a/gi, to: "var(--text-primary)" },
  { from: /#101828/gi, to: "__CONTEXT_TEXT_OR_BTN__" }, // needs context
  { from: /#101827/gi, to: "var(--text-primary)" },
  { from: /#344054/gi, to: "var(--text-secondary)" },
  { from: /#334155/gi, to: "var(--text-secondary)" },
  { from: /#667085/gi, to: "var(--text-muted)" },
  { from: /#64748b/gi, to: "var(--text-muted)" },
  { from: /#475467/gi, to: "var(--text-secondary)" },
  { from: /#475569/gi, to: "var(--text-muted)" },

  // Backgrounds
  { from: /#f6f7f9/gi, to: "var(--bg-page)" },
  { from: /#eef2f5/gi, to: "var(--bg-page-deep)" },
  { from: /#f8fafb/gi, to: "var(--bg-nav)" },
  { from: /#f8fafc/gi, to: "var(--bg-surface-hover)" },
  { from: /#f4f7fb/gi, to: "var(--bg-page)" },

  // Assistant bubble
  { from: /#111315/gi, to: "var(--bg-assistant)" },
  { from: /#0b0d0f/gi, to: "var(--bg-assistant)" },

  // Code / soft grays
  { from: /#eeeeee/gi, to: "__CONTEXT_BORDER_OR_CODE__" }, // needs context
  { from: /#ececec/gi, to: "var(--border-subtle)" },
  { from: /#e5e7eb/gi, to: "__CONTEXT_BORDER_OR_TEXT__" }, // needs context

  // Old accent blue
  { from: /#2563eb/gi, to: "var(--status-info)" },
  { from: /#1d4ed8/gi, to: "var(--status-info)" },

  // Misc text
  { from: /#1f2937/gi, to: "var(--text-primary)" },
  { from: /#1d2939/gi, to: "var(--text-primary)" },
  { from: /#24292f/gi, to: "var(--text-primary)" },
  { from: /#202124/gi, to: "var(--text-primary)" },
  { from: /#94a3b8/gi, to: "var(--text-muted)" },
  { from: /#6b7280/gi, to: "var(--text-muted)" },
  { from: /#4b5563/gi, to: "var(--text-secondary)" },
  { from: /#374151/gi, to: "var(--text-secondary)" },

  // Misc backgrounds
  { from: /#f1f3f4/gi, to: "var(--bg-code-inline)" },
  { from: /#edf4fb/gi, to: "var(--bg-code-block)" },
  { from: /#f2f8ff/gi, to: "var(--status-info-bg)" },
  { from: /#eaf2ff/gi, to: "var(--status-info-bg)" },
  { from: /#f8f9fa/gi, to: "var(--bg-surface-hover)" },
  { from: /#f7f7f7/gi, to: "var(--bg-surface-hover)" },
  { from: /#f5f5f5/gi, to: "var(--bg-surface-hover)" },
  { from: /#f4f4f4/gi, to: "var(--bg-page-deep)" },
  { from: /#f3f4f6/gi, to: "var(--bg-page-deep)" },
  { from: /#f1f1f1/gi, to: "var(--bg-surface-hover)" },
  { from: /#f2f4f7/gi, to: "var(--bg-surface-hover)" },
  { from: /#f1f5f9/gi, to: "var(--bg-surface-hover)" },
  { from: /#f5f7f8/gi, to: "var(--bg-page)" },

  // ── Second pass: remaining high-impact colors ──

  // Status colors (Tailwind palette values used in legacy code)
  { from: /#dc2626/gi, to: "var(--status-error)" },
  { from: /#b91c1c/gi, to: "var(--status-error)" },
  { from: /#991b1b/gi, to: "var(--status-error)" },
  { from: /#16a34a/gi, to: "var(--status-success)" },
  { from: /#15803d/gi, to: "var(--status-success)" },
  { from: /#166534/gi, to: "var(--status-success)" },
  { from: /#65a30d/gi, to: "var(--status-success)" },
  { from: /#b45309/gi, to: "var(--status-warning)" },
  { from: /#d97706/gi, to: "var(--status-warning)" },
  { from: /#ca8a04/gi, to: "var(--status-warning)" },
  { from: /#eab308/gi, to: "var(--status-warning)" },
  { from: /#1e3a8a/gi, to: "var(--status-info)" },
  { from: /#0891b2/gi, to: "var(--status-info)" },
  { from: /#0f766e/gi, to: "var(--accent-hover)" },
  { from: /#14b8a6/gi, to: "var(--accent)" },
  { from: /#f97316/gi, to: "var(--status-warning)" },

  // Old blue-theme ink colors (used in legacy/control panels)
  { from: /#072843/gi, to: "var(--text-primary)" },
  { from: /#16324f/gi, to: "var(--text-secondary)" },
  { from: /#1c4768/gi, to: "var(--text-secondary)" },
  { from: /#0f3f68/gi, to: "var(--text-secondary)" },
  { from: /#355776/gi, to: "var(--text-secondary)" },
  { from: /#37526d/gi, to: "var(--text-secondary)" },
  { from: /#587693/gi, to: "var(--text-muted)" },
  { from: /#66758a/gi, to: "var(--text-muted)" },
  { from: /#6a86a3/gi, to: "var(--text-muted)" },
  { from: /#758195/gi, to: "var(--text-muted)" },
  { from: /#7794b2/gi, to: "var(--text-muted)" },
  { from: /#7891aa/gi, to: "var(--text-muted)" },
  { from: /#8a919d/gi, to: "var(--text-muted)" },
  { from: /#91a1b6/gi, to: "var(--text-muted)" },
  { from: /#98a2b3/gi, to: "var(--text-muted)" },
  { from: /#b0b7c2/gi, to: "var(--text-muted)" },
  { from: /#7a7f87/gi, to: "var(--text-muted)" },
  { from: /#4f5661/gi, to: "var(--text-secondary)" },

  // Remaining soft backgrounds
  { from: /#f6f8fa/gi, to: "var(--bg-code-block)" },
  { from: /#f1f2f4/gi, to: "var(--bg-surface-hover)" },
  { from: /#f7f7f8/gi, to: "var(--bg-surface-hover)" },
  { from: /#eef2f7/gi, to: "var(--bg-page-deep)" },
  { from: /#e5edf7/gi, to: "var(--bg-page-deep)" },
  { from: /#dfeeff/gi, to: "var(--status-info-bg)" },
  { from: /#dbe8f6/gi, to: "var(--bg-page-deep)" },
  { from: /#c9d9ec/gi, to: "var(--border-default)" },
  { from: /#d1d5db/gi, to: "var(--border-default)" },

  // Light text on dark backgrounds (used in status labels, etc.)
  { from: /#eff9ff/gi, to: "var(--text-on-accent)" },
  { from: /#effaff/gi, to: "var(--text-on-accent)" },
  { from: /#eff6ff/gi, to: "var(--text-on-accent)" },
  { from: /#fff5f4/gi, to: "var(--text-on-accent)" },
  { from: /#f8fbff/gi, to: "var(--text-on-accent)" },
  { from: /#f7fbff/gi, to: "var(--text-on-accent)" },
  { from: /#f2f7ff/gi, to: "var(--status-info-bg)" },
  { from: /#eef3f8/gi, to: "var(--bg-surface-hover)" },
];

/* ── Main ────────────────────────────────────────────────── */

function main() {
  const filePath = process.argv[2] || DEFAULT_PATH;
  let css = fs.readFileSync(filePath, "utf8");
  const original = css;

  console.log(`Tokenizing: ${filePath}`);
  console.log(`Original size: ${css.length} bytes`);

  // ── Step 1: Remove all :root blocks (now in tokens.css) ──
  // Match from :root { to its closing }
  // We need to handle nested braces carefully
  css = removeRootBlocks(css);

  // ── Step 2: Apply rgba rules ──
  for (const rule of rgbaRules) {
    css = css.replace(rule.from, rule.to);
  }

  // ── Step 3: Apply hex rules (unambiguous) ──
  for (const rule of hexRules) {
    if (rule.to.startsWith("__CONTEXT")) continue; // handled in step 4
    css = css.replace(rule.from, rule.to);
  }

  // ── Step 4: Context-dependent replacements ──

  // #101828: in color: → text-primary; in background/border → bg-btn-primary
  css = css.replace(
    /(\bcolor\s*:\s*)#101828\b/gi,
    "$1var(--text-primary)",
  );
  css = css.replace(
    /(\bbackground(?:-color)?\s*:\s*)#101828\b/gi,
    "$1var(--bg-btn-primary)",
  );
  css = css.replace(
    /(\bborder(?:-color)?\s*:\s*)#101828\b/gi,
    "$1var(--bg-btn-primary)",
  );
  // Any remaining #101828 (e.g., in gradients) — treat as text
  css = css.replace(/#101828/gi, "var(--text-primary)");

  // #eeeeee: in background → bg-code-inline; in border → border-subtle
  css = css.replace(
    /(\bbackground(?:-color)?\s*:\s*)#eeeeee\b/gi,
    "$1var(--bg-code-inline)",
  );
  css = css.replace(
    /(\bborder(?:-color)?\s*:\s*)#eeeeee\b/gi,
    "$1var(--border-subtle)",
  );
  css = css.replace(/#eeeeee/gi, "var(--border-subtle)");

  // #e5e7eb: in color (on dark bg) → text-on-assistant; in border → border-default
  css = css.replace(
    /(\bcolor\s*:\s*)#e5e7eb\b/gi,
    "$1var(--text-on-assistant)",
  );
  css = css.replace(
    /(\bborder(?:-color)?\s*:\s*)#e5e7eb\b/gi,
    "$1var(--border-default)",
  );
  css = css.replace(/#e5e7eb/gi, "var(--border-default)");

  // #ffffff / #fff: this is tricky. Most are bg-surface, but some are text-on-accent.
  // Strategy: in color: property → text-on-accent or text-on-primary-btn (if parent is accent/btn)
  // In background: → bg-surface
  // For simplicity, replace all #ffffff/#fff in backgrounds with bg-surface,
  // and in color properties with text-on-accent.
  // Actually, #fff in color is very rare — most are backgrounds.
  css = css.replace(
    /(\bbackground(?:-color)?\s*:\s*)#ffffff\b/gi,
    "$1var(--bg-surface)",
  );
  css = css.replace(
    /(\bbackground(?:-color)?\s*:\s*)#fff\b/gi,
    "$1var(--bg-surface)",
  );
  css = css.replace(
    /(\bcolor\s*:\s*)#ffffff\b/gi,
    "$1var(--text-on-accent)",
  );
  css = css.replace(
    /(\bcolor\s*:\s*)#fff\b/gi,
    "$1var(--text-on-accent)",
  );
  // Remaining #fff (in gradients, box-shadow 2px white ring, etc.)
  // In box-shadow: 0 0 0 2px #ffffff → keep as is for focus ring (it's the white inner ring)
  // Actually, let's replace remaining with bg-surface since that's the most common
  css = css.replace(/#ffffff/gi, "var(--bg-surface)");
  css = css.replace(/#fff\b/gi, "var(--bg-surface)");

  // ── Step 5: Replace hardcoded 8px radii with token ──
  // Only replace standalone border-radius values, not inside calc() or other contexts
  // Actually this is risky — let's skip radii for now since they're already tokenized
  // via the --shell-radius-* aliases

  // ── Step 6: Clean up ──
  // Remove any leftover __CONTEXT_* markers (should be none)
  css = css.replace(/__CONTEXT_\w+__/g, (m) => {
    console.warn(`Unresolved context marker: ${m}`);
    return m;
  });

  // ── Report ──
  const remainingHex = (css.match(/#[0-9a-fA-F]{3,8}\b/g) || []).length;
  const remainingRgba = (css.match(/rgba?\(/g) || []).length;
  const diff = original.length - css.length;

  console.log(`\nResults:`);
  console.log(`  New size: ${css.length} bytes (${diff > 0 ? "-" : "+"}${Math.abs(diff)})`);
  console.log(`  Remaining hex colors: ${remainingHex}`);
  console.log(`  Remaining rgba() calls: ${remainingRgba}`);

  fs.writeFileSync(filePath, css);
  console.log(`\nWritten to: ${filePath}`);
}

/**
 * Remove all `:root { ... }` blocks from the CSS.
 * Handles nested braces by tracking depth.
 */
function removeRootBlocks(css) {
  let result = "";
  let i = 0;
  let removed = 0;

  while (i < css.length) {
    // Look for :root {
    const match = css.substring(i).match(/^:root\s*\{/m);
    if (!match) {
      result += css.substring(i);
      break;
    }

    const startIdx = i + (match.index || 0);
    // Add everything before this :root
    result += css.substring(i, startIdx);

    // Find the matching closing brace
    let depth = 0;
    let j = startIdx + match[0].length;
    while (j < css.length) {
      if (css[j] === "{") depth++;
      else if (css[j] === "}") {
        if (depth === 0) {
          j++; // skip the closing }
          break;
        }
        depth--;
      }
      j++;
    }

    // Skip this block (don't add to result)
    i = j;
    removed++;
  }

  if (removed > 0) {
    console.log(`  Removed ${removed} :root blocks`);
  }

  return result;
}

main();
