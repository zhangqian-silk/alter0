import { render } from "@testing-library/react";
import { LegacyWebShell } from "./LegacyWebShell";
import {
  LEGACY_SHELL_REQUIRED_IDS,
  getMissingLegacyShellIds,
  hasLegacyShellContract
} from "./legacyDomContract";

describe("legacyDomContract", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
  });

  it("reports the full contract as missing before the shell renders", () => {
    expect(hasLegacyShellContract()).toBe(false);
    expect(getMissingLegacyShellIds()).toEqual(LEGACY_SHELL_REQUIRED_IDS);
  });

  it("reports the contract as ready after the React shell renders", () => {
    render(<LegacyWebShell />);

    expect(hasLegacyShellContract()).toBe(true);
    expect(getMissingLegacyShellIds()).toEqual([]);
  });
});
