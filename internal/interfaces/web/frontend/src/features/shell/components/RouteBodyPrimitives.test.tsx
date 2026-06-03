import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CopyValueButton, RouteFieldRow } from "./RouteBodyPrimitives";
import { MessageMarkdownShell } from "./MessageMarkdownShell";

describe("RouteFieldRow", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it("falls back to document copy when navigator.clipboard is unavailable", async () => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: undefined,
    });
    const execCommand = vi.fn().mockReturnValue(true);
    Object.defineProperty(document, "execCommand", {
      configurable: true,
      value: execCommand,
    });

    render(
      <RouteFieldRow
        label="Guide HTML URL"
        value="https://travel-wu42aa18.alter0.cn"
        copyLabel="Copy value"
        copyable
        mono
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Copy value" }));

    await waitFor(() => {
      expect(execCommand).toHaveBeenCalledWith("copy");
    });
  });

  it("renders markdown field values through the shared safe renderer", () => {
    const { container } = render(
      <RouteFieldRow
        label="Description"
        value={"## Runtime notes\n- **fast** path\n- [Guide](/docs)\n- [bad](javascript:alert(1))"}
        copyLabel="Copy value"
        markdown
        multiline
      />,
    );

    expect(container.querySelector(".route-field-value.is-markdown h2")).toHaveTextContent("Runtime notes");
    expect(container.querySelector(".route-field-value.is-markdown strong")).toHaveTextContent("fast");
    expect(container.querySelector(".route-field-value.is-markdown a")).toHaveAttribute("href", "/docs");
    expect(container.querySelector(".route-field-value.is-markdown")).not.toContainHTML("javascript:");
    expect(screen.getByText("bad")).toBeInTheDocument();
  });

  it("keeps long copy payloads out of DOM attributes while preserving clipboard writes", async () => {
    const longValue = "terminal output line\n".repeat(512);
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(
      <CopyValueButton
        value={longValue}
        label="Copy output"
      />,
    );

    const button = screen.getByRole("button", { name: "Copy output" });
    expect(button).not.toHaveAttribute("data-copy-value");

    fireEvent.click(button);

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(longValue);
    });
  });

  it("renders markdown body before the copy toolbar so browser text selection follows visual order", () => {
    const { container } = render(
      <MessageMarkdownShell
        markdown="selectable terminal output"
        copyValue="selectable terminal output"
        copyLabel="Copy output"
      />,
    );

    const shell = container.querySelector(".message-markdown-shell");
    expect(shell?.children.item(0)).toHaveClass("message-markdown-body");
    expect(shell?.children.item(1)).toHaveClass("message-markdown-toolbar");
  });

  it("renders markdown output as static selectable text without entering edit mode", () => {
    const { container } = render(
      <MessageMarkdownShell
        markdown="selectable terminal output"
        copyValue="selectable terminal output"
        copyLabel="Copy output"
      />,
    );

    const rendered = container.querySelector(".message-markdown-rendered");
    expect(rendered).not.toHaveAttribute("contenteditable");
    expect(rendered).not.toHaveAttribute("aria-readonly");
    expect(rendered).not.toHaveAttribute("inputmode");
    expect(rendered).not.toHaveAttribute("tabindex");
  });

  it("does not install scripted touch selection controls on markdown output", () => {
    vi.useFakeTimers();
    const { container } = render(
      <MessageMarkdownShell
        markdown="selectable terminal output"
        copyValue="selectable terminal output"
        copyLabel="Copy output"
      />,
    );

    const rendered = container.querySelector(".message-markdown-rendered") as HTMLElement;
    const touch = { clientX: 12, clientY: 18 };
    const touchStart = new Event("touchstart", { bubbles: true, cancelable: true });
    Object.defineProperty(touchStart, "touches", { value: [touch] });
    Object.defineProperty(touchStart, "targetTouches", { value: [touch] });
    Object.defineProperty(touchStart, "changedTouches", { value: [touch] });
    fireEvent(rendered, touchStart);

    expect(vi.getTimerCount()).toBe(0);
    expect(container.querySelector(".runtime-touch-copy")).not.toBeInTheDocument();
    expect(rendered).not.toHaveClass("is-touch-selected");
    expect(rendered).not.toHaveAttribute("contenteditable");
    expect(rendered).not.toHaveAttribute("inputmode");
  });
});
