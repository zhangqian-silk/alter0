import { render } from "@testing-library/react";
import { usePageActivation } from "./usePageActivation";

function PageActivationHarness({ onActive }: { onActive: () => void }) {
  usePageActivation({ debounceMs: 0, onActive });
  return null;
}

describe("usePageActivation", () => {
  it("fires when the browser restores the page from history cache", () => {
    const onActive = vi.fn();
    render(<PageActivationHarness onActive={onActive} />);

    window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));

    expect(onActive).toHaveBeenCalledTimes(1);
  });

  it("fires when the browser comes back online while visible", () => {
    const onActive = vi.fn();
    render(<PageActivationHarness onActive={onActive} />);

    window.dispatchEvent(new Event("online"));

    expect(onActive).toHaveBeenCalledTimes(1);
  });
});
