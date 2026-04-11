import { render } from "@testing-library/react";
import { LegacyRuntimeBridge } from "./LegacyRuntimeBridge";
import { ensureLegacyRuntimeScript } from "./loadLegacyRuntime";
import { LegacyWebShell } from "../features/shell/LegacyWebShell";

describe("ensureLegacyRuntimeScript", () => {
  beforeEach(() => {
    document.head.innerHTML = "";
    document.body.innerHTML = "";
  });

  it("appends the legacy runtime script once", () => {
    render(<LegacyWebShell />);

    const first = ensureLegacyRuntimeScript("/legacy/chat.js");
    const second = ensureLegacyRuntimeScript("/legacy/chat.js");

    expect(first).toBe(second);
    expect(document.querySelectorAll("#alter0-legacy-runtime")).toHaveLength(1);
    expect(first.getAttribute("src")).toBe("/legacy/chat.js");
    expect(first.dataset.runtimeBridge).toBe("legacy");
  });

  it("refuses to boot the legacy runtime before the shell contract exists", () => {
    expect(() => ensureLegacyRuntimeScript("/legacy/chat.js")).toThrow(
      "legacy shell contract is incomplete",
    );
    expect(document.querySelector("#alter0-legacy-runtime")).toBeNull();
  });

  it("boots the legacy runtime from the React bridge", () => {
    render(
      <>
        <LegacyWebShell />
        <LegacyRuntimeBridge />
      </>,
    );

    const script = document.getElementById("alter0-legacy-runtime");
    expect(script).toBeInstanceOf(HTMLScriptElement);
    expect(script).toHaveAttribute("src", "/legacy/chat.js");
  });
});
