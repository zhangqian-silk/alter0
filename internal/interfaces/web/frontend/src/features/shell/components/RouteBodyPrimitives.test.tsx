import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RouteFieldRow } from "./RouteBodyPrimitives";

describe("RouteFieldRow", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
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
});
