import { afterEach, describe, expect, it, vi } from "vitest";
import { groupSessionListItems } from "./sessionListGroups";

describe("groupSessionListItems", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps pinned sessions in their own group before Today", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-23T12:00:00Z"));

    const today = Date.parse("2026-04-23T08:00:00Z");
    const yesterday = Date.parse("2026-04-22T08:00:00Z");

    const groups = groupSessionListItems(
      [
        { id: "today", at: today, pinned: false },
        { id: "pinned", at: yesterday, pinned: true },
        { id: "yesterday", at: yesterday, pinned: false },
      ],
      {
        language: "en",
        getTimestamp: (item) => item.at,
        getPinned: (item) => item.pinned,
      },
    );

    expect(groups.map((group) => group.key)).toEqual(["pinned", "today", "yesterday"]);
    expect(groups.map((group) => group.label)).toEqual(["Pinned", "Today", "Yesterday"]);
    expect(groups[0].items.map((item) => item.id)).toEqual(["pinned"]);
  });

  it("places unpinned sessions within two days of cleanup in the expiring soon group", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-23T12:00:00Z"));

    const groups = groupSessionListItems(
      [
        { id: "five-days-old", at: Date.parse("2026-04-18T12:00:00Z"), pinned: false },
        { id: "almost-expired", at: Date.parse("2026-04-16T13:00:00Z"), pinned: false },
        { id: "already-expired", at: Date.parse("2026-04-16T11:59:00Z"), pinned: false },
        { id: "pinned-old", at: Date.parse("2026-04-17T12:00:00Z"), pinned: true },
      ],
      {
        language: "en",
        getTimestamp: (item) => item.at,
        getPinned: (item) => item.pinned,
      },
    );

    expect(groups.map((group) => group.key)).toEqual(["pinned", "expiringSoon", "earlier"]);
    expect(groups.map((group) => group.label)).toEqual(["Pinned", "Expiring Soon", "Earlier"]);
    expect(groups.find((group) => group.key === "expiringSoon")?.items.map((item) => item.id)).toEqual([
      "five-days-old",
      "almost-expired",
    ]);
  });
});
