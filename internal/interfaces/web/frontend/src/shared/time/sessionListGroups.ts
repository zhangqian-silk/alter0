export type SessionListGroup<T> = {
  key: "pinned" | "expiringSoon" | "today" | "yesterday" | "earlier";
  label: string;
  items: T[];
};

const inactiveCleanupDurationMs = 7 * 24 * 60 * 60 * 1000;
const expiringSoonWindowMs = 2 * 24 * 60 * 60 * 1000;

export function groupSessionListItems<T>(
  items: T[],
  options: {
    language: "en" | "zh";
    getTimestamp: (item: T) => number;
    getPinned?: (item: T) => boolean;
  },
): SessionListGroup<T>[] {
  const labels = options.language === "zh"
    ? {
        pinned: "置顶",
        expiringSoon: "即将清理",
        today: "今天",
        yesterday: "昨天",
        earlier: "更早",
      }
    : {
        pinned: "Pinned",
        expiringSoon: "Expiring Soon",
        today: "Today",
        yesterday: "Yesterday",
        earlier: "Earlier",
      };

  const now = new Date();
  const nowTime = now.getTime();
  const todayStart = new Date(now);
  todayStart.setHours(0, 0, 0, 0);
  const yesterdayStart = new Date(todayStart);
  yesterdayStart.setDate(todayStart.getDate() - 1);
  const todayStartTime = todayStart.getTime();
  const yesterdayStartTime = yesterdayStart.getTime();
  const expiringSoonStartTime = nowTime - (inactiveCleanupDurationMs - expiringSoonWindowMs);
  const inactiveCleanupCutoffTime = nowTime - inactiveCleanupDurationMs;

  const groups: SessionListGroup<T>[] = [
    { key: "pinned", label: labels.pinned, items: [] },
    { key: "expiringSoon", label: labels.expiringSoon, items: [] },
    { key: "today", label: labels.today, items: [] },
    { key: "yesterday", label: labels.yesterday, items: [] },
    { key: "earlier", label: labels.earlier, items: [] },
  ];

  items.forEach((item) => {
    if (options.getPinned?.(item)) {
      groups[0].items.push(item);
      return;
    }
    const timestamp = options.getTimestamp(item);
    if (
      Number.isFinite(timestamp)
      && timestamp <= expiringSoonStartTime
      && timestamp >= inactiveCleanupCutoffTime
    ) {
      groups[1].items.push(item);
      return;
    }
    if (Number.isFinite(timestamp) && timestamp >= todayStartTime) {
      groups[2].items.push(item);
      return;
    }
    if (Number.isFinite(timestamp) && timestamp >= yesterdayStartTime) {
      groups[3].items.push(item);
      return;
    }
    groups[4].items.push(item);
  });

  return groups.filter((group) => group.items.length > 0);
}
