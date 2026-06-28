export function isEditableRuntimeElement(value: Element | null): value is HTMLElement {
  if (!(value instanceof HTMLElement)) {
    return false;
  }
  return (
    value instanceof HTMLInputElement
    || value instanceof HTMLTextAreaElement
    || Boolean(value.isContentEditable)
  );
}

export function blurActiveEditableElement(ownerDocument: Document = document): boolean {
  const activeElement = ownerDocument.activeElement;
  if (!isEditableRuntimeElement(activeElement)) {
    return false;
  }
  activeElement.blur();
  return true;
}

export function runWithKeyboardDismissal(action?: () => void): void {
  blurActiveEditableElement();
  action?.();
}

export function withKeyboardDismissal<TEvent>(
  handler?: (event: TEvent) => void,
): (event: TEvent) => void {
  return (event) => {
    blurActiveEditableElement();
    handler?.(event);
  };
}
