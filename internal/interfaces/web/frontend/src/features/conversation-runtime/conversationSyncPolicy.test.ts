import { describe, expect, it } from "vitest";
import {
  CONVERSATION_BACKGROUND_FULL_REFRESH_MS,
  createConversationRequestGate,
  filterConversationUpdatesAfter,
  resolveConversationResumePlan,
  selectConversationContentCacheSessionIDs,
  shouldRequestConversationDetailOnSwitch,
} from "./conversationSyncPolicy";

describe("conversationSyncPolicy", () => {
  it("resumes only unfinished update sync after a short mobile background gap", () => {
    expect(resolveConversationResumePlan({
      hiddenAt: 1_000,
      now: 1_000 + CONVERSATION_BACKGROUND_FULL_REFRESH_MS - 1,
      hasLocallyUnfinishedSession: true,
    })).toEqual({
      refreshList: false,
      refreshActiveDetail: false,
      resumeUpdates: true,
    });
  });

  it("refreshes the lightweight list and active detail after five minutes", () => {
    expect(resolveConversationResumePlan({
      hiddenAt: 1_000,
      now: 1_000 + CONVERSATION_BACKGROUND_FULL_REFRESH_MS,
      hasLocallyUnfinishedSession: false,
    })).toEqual({
      refreshList: true,
      refreshActiveDetail: true,
      resumeUpdates: false,
    });
  });

  it("requests detail only when the selected content cannot be trusted", () => {
    const stable = {
      contentLoaded: true,
      summaryUpdatedAt: 20,
      contentUpdatedAt: 20,
      malformedCache: false,
      unfinished: false,
      updateSyncActive: false,
      explicitRefresh: false,
    };

    expect(shouldRequestConversationDetailOnSwitch(stable)).toBe(false);
    expect(shouldRequestConversationDetailOnSwitch({ ...stable, summaryUpdatedAt: 21 })).toBe(true);
    expect(shouldRequestConversationDetailOnSwitch({ ...stable, contentLoaded: false })).toBe(true);
    expect(shouldRequestConversationDetailOnSwitch({ ...stable, malformedCache: true })).toBe(true);
    expect(shouldRequestConversationDetailOnSwitch({ ...stable, unfinished: true, updateSyncActive: true })).toBe(false);
    expect(shouldRequestConversationDetailOnSwitch({ ...stable, unfinished: true, updateSyncActive: false })).toBe(true);
    expect(shouldRequestConversationDetailOnSwitch({ ...stable, explicitRefresh: true })).toBe(true);
  });

  it("rejects a response from an older request generation", () => {
    const gate = createConversationRequestGate();
    const first = gate.begin("detail:session-1");
    const second = gate.begin("detail:session-1");

    expect(gate.isLatest("detail:session-1", first)).toBe(false);
    expect(gate.isLatest("detail:session-1", second)).toBe(true);
  });

  it("keeps content only for the active and four most recently updated sessions", () => {
    const sessions = Array.from({ length: 7 }, (_, index) => ({
      id: `session-${index + 1}`,
      updatedAt: index + 1,
    }));

    expect(Array.from(selectConversationContentCacheSessionIDs("session-1", sessions))).toEqual([
      "session-1",
      "session-7",
      "session-6",
      "session-5",
      "session-4",
    ]);
  });

  it("drops replayed update ids before they reach the conversation merge", () => {
    expect(filterConversationUpdatesAfter("7", [
      { update_id: 7, value: "replayed" },
      { update_id: "9", value: "newest" },
      { update_id: 8, value: "newer" },
      { update_id: 8, value: "duplicate" },
    ])).toEqual([
      { update_id: 8, value: "newer" },
      { update_id: "9", value: "newest" },
    ]);
  });
});
