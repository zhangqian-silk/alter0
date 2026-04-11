import { render } from "@testing-library/react";
import { LegacyRuntimeBridge } from "./LegacyRuntimeBridge";
import { ensureLegacyRuntimeScript } from "./loadLegacyRuntime";

describe("ensureLegacyRuntimeScript", () => {
  beforeEach(() => {
    document.head.innerHTML = "";
    document.body.innerHTML = '<div id="frontend-root"></div>';
  });

  it("appends the legacy runtime script once", () => {
    const first = ensureLegacyRuntimeScript("/legacy/chat.js");
    const second = ensureLegacyRuntimeScript("/legacy/chat.js");

    expect(first).toBe(second);
    expect(document.querySelectorAll("#alter0-legacy-runtime")).toHaveLength(1);
    expect(first.getAttribute("src")).toBe("/legacy/chat.js");
    expect(first.dataset.runtimeBridge).toBe("legacy");
  });

  it("boots the legacy runtime from the React bridge", () => {
    render(<LegacyRuntimeBridge />);

    const script = document.getElementById("alter0-legacy-runtime");
    expect(script).toBeInstanceOf(HTMLScriptElement);
    expect(script).toHaveAttribute("src", "/legacy/chat.js");
  });
});
