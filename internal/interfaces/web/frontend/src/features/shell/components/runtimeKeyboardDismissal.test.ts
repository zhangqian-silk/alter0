import {
  blurActiveEditableElement,
  isEditableRuntimeElement,
  runWithKeyboardDismissal,
  withKeyboardDismissal,
} from "./runtimeKeyboardDismissal";

describe("runtimeKeyboardDismissal", () => {
  it("recognizes editable input, textarea, and contenteditable elements", () => {
    const input = document.createElement("input");
    const textarea = document.createElement("textarea");
    const editable = document.createElement("div");
    Object.defineProperty(editable, "isContentEditable", {
      configurable: true,
      value: true,
    });

    expect(isEditableRuntimeElement(input)).toBe(true);
    expect(isEditableRuntimeElement(textarea)).toBe(true);
    expect(isEditableRuntimeElement(editable)).toBe(true);
    expect(isEditableRuntimeElement(document.createElement("button"))).toBe(false);
  });

  it("blurs active input and textarea elements", () => {
    const input = document.createElement("input");
    const textarea = document.createElement("textarea");
    document.body.append(input, textarea);

    try {
      input.focus();
      expect(blurActiveEditableElement()).toBe(true);
      expect(document.activeElement).not.toBe(input);

      textarea.focus();
      expect(blurActiveEditableElement()).toBe(true);
      expect(document.activeElement).not.toBe(textarea);
    } finally {
      input.remove();
      textarea.remove();
    }
  });

  it("returns false when no editable element owns focus", () => {
    const button = document.createElement("button");
    document.body.appendChild(button);

    try {
      button.focus();
      expect(blurActiveEditableElement()).toBe(false);
      expect(document.activeElement).toBe(button);
    } finally {
      button.remove();
    }
  });

  it("runs wrapped actions after dismissing the keyboard", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();
    const blur = vi.spyOn(input, "blur");
    const action = vi.fn();

    try {
      runWithKeyboardDismissal(action);

      expect(blur).toHaveBeenCalledTimes(1);
      expect(action).toHaveBeenCalledTimes(1);
    } finally {
      input.remove();
    }
  });

  it("passes events through dismissal-wrapped handlers", () => {
    const input = document.createElement("input");
    const event = new Event("click");
    document.body.appendChild(input);
    input.focus();
    const blur = vi.spyOn(input, "blur");
    const handler = vi.fn();

    try {
      withKeyboardDismissal(handler)(event);

      expect(blur).toHaveBeenCalledTimes(1);
      expect(handler).toHaveBeenCalledWith(event);
    } finally {
      input.remove();
    }
  });
});
