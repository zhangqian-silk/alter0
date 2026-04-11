import {
  FRONTEND_DISPLAY_TIME_ZONE,
  formatDateTime,
  formatTimeLabel
} from "./format";

describe("shared time format", () => {
  it("keeps the frontend display timezone fixed to Asia/Shanghai", () => {
    expect(FRONTEND_DISPLAY_TIME_ZONE).toBe("Asia/Shanghai");
  });

  it("formats absolute timestamps in Beijing time", () => {
    expect(formatDateTime("2026-04-11T01:05:06Z")).toBe("2026-04-11 09:05:06");
  });

  it("formats time labels in Beijing time with h23 hour cycle", () => {
    expect(formatTimeLabel("2026-04-10T20:05:06Z")).toBe("04:05");
  });

  it("returns the original text for invalid absolute timestamps", () => {
    expect(formatDateTime("not-a-time")).toBe("not-a-time");
  });

  it("returns fallback markers for empty or invalid time labels", () => {
    expect(formatDateTime("")).toBe("-");
    expect(formatTimeLabel("not-a-time")).toBe("-");
  });
});
