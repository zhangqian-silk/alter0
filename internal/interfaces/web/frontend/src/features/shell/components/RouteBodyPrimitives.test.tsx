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
});
