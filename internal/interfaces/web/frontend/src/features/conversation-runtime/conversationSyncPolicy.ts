export const CONVERSATION_BACKGROUND_FULL_REFRESH_MS = 5 * 60 * 1000;

export type ConversationResumePlan = {
  refreshList: boolean;
  refreshActiveDetail: boolean;
  resumeUpdates: boolean;
};

export type ConversationRequestGate = {
  begin: (key: string) => number;
  isLatest: (key: string, generation: number) => boolean;
};

export function createConversationRequestGate(): ConversationRequestGate {
  const generations = new Map<string, number>();
  return {
    begin(key) {
      const generation = (generations.get(key) || 0) + 1;
      generations.set(key, generation);
      return generation;
    },
    isLatest(key, generation) {
      return generations.get(key) === generation;
    },
  };
}

export function resolveConversationResumePlan(options: {
  hiddenAt: number;
  now: number;
  hasLocallyUnfinishedSession: boolean;
}): ConversationResumePlan {
  const hiddenDuration = Math.max(0, options.now - options.hiddenAt);
  if (options.hiddenAt > 0 && hiddenDuration >= CONVERSATION_BACKGROUND_FULL_REFRESH_MS) {
    return {
      refreshList: true,
      refreshActiveDetail: true,
      resumeUpdates: false,
    };
  }
  return {
    refreshList: false,
    refreshActiveDetail: false,
    resumeUpdates: options.hasLocallyUnfinishedSession,
  };
}

export function shouldRequestConversationDetailOnSwitch(options: {
  contentLoaded: boolean;
  summaryUpdatedAt: number;
  contentUpdatedAt: number;
  malformedCache: boolean;
  unfinished: boolean;
  updateSyncActive: boolean;
  explicitRefresh: boolean;
}): boolean {
  if (options.explicitRefresh || options.malformedCache || !options.contentLoaded) {
    return true;
  }
  if (options.summaryUpdatedAt > options.contentUpdatedAt) {
    return true;
  }
  return options.unfinished && !options.updateSyncActive;
}

export function selectConversationContentCacheSessionIDs<T extends { id: string; updatedAt: number }>(
  activeSessionID: string,
  sessions: T[],
  limit: number = 5,
): Set<string> {
  const ids = new Set<string>();
  const normalizedLimit = Math.max(1, Math.trunc(limit));
  if (activeSessionID && sessions.some((session) => session.id === activeSessionID)) {
    ids.add(activeSessionID);
  }
  [...sessions]
    .sort((left, right) => right.updatedAt - left.updatedAt || right.id.localeCompare(left.id))
    .forEach((session) => {
      if (ids.size < normalizedLimit && session.id) {
        ids.add(session.id);
      }
    });
  return ids;
}

export function filterConversationUpdatesAfter<T extends { update_id?: string | number }>(
  afterUpdateID: string | number,
  updates: T[],
): T[] {
  const after = Number(afterUpdateID);
  const lowerBound = Number.isFinite(after) ? after : 0;
  const seen = new Set<number>();
  return updates
    .map((update, index) => ({ update, index, id: Number(update.update_id) }))
    .filter(({ id }) => {
      if (!Number.isFinite(id) || id <= lowerBound || seen.has(id)) {
        return false;
      }
      seen.add(id);
      return true;
    })
    .sort((left, right) => left.id - right.id || left.index - right.index)
    .map(({ update }) => update);
}
