export async function copyTextValue(value: string): Promise<boolean> {
  const safeValue = String(value || "");
  if (!safeValue.trim()) {
    return false;
  }
  if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
    try {
      await navigator.clipboard.writeText(safeValue);
      return true;
    } catch {
      // Fall back when async clipboard writes are unavailable or denied.
    }
  }
  return fallbackCopyTextValue(safeValue);
}

function fallbackCopyTextValue(value: string): boolean {
  if (typeof document === "undefined" || !document.body) {
    return false;
  }
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "");
  textarea.setAttribute("aria-hidden", "true");
  textarea.style.position = "fixed";
  textarea.style.top = "0";
  textarea.style.left = "-9999px";
  textarea.style.opacity = "0";
  textarea.style.pointerEvents = "none";

  const activeElement = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  const selection = document.getSelection();
  const ranges = selection
    ? Array.from({ length: selection.rangeCount }, (_, index) => selection.getRangeAt(index).cloneRange())
    : [];

  document.body.appendChild(textarea);
  try {
    textarea.focus();
    textarea.select();
    textarea.setSelectionRange(0, textarea.value.length);
    const execCommand = (document as Document & { execCommand?: (command: string) => boolean }).execCommand;
    if (typeof execCommand !== "function") {
      return false;
    }
    return execCommand.call(document, "copy");
  } catch {
    return false;
  } finally {
    document.body.removeChild(textarea);
    if (selection) {
      selection.removeAllRanges();
      ranges.forEach((range) => selection.addRange(range));
    }
    activeElement?.focus?.();
  }
}
