export type SessionListGroup<T> = {
  key: "today" | "yesterday" | "earlier";
  label: string;
  items: T[];
};

export function groupSessionListItems<T>(
  items: T[],
  options: {
    language: "en" | "zh";
    getTimestamp: (item: T) => number;
  },
): SessionListGroup<T>[] {
  const labels = options.language === "zh"
    ? {
        today: "今天",
        yesterday: "昨天",
        earlier: "更早",
      }
    : {
        today: "Today",
        yesterday: "Yesterday",
        earlier: "Earlier",
      };

  const now = new Date();
  const todayStart = new Date(now);
  todayStart.setHours(0, 0, 0, 0);
  const yesterdayStart = new Date(todayStart);
  yesterdayStart.setDate(todayStart.getDate() - 1);
  const todayStartTime = todayStart.getTime();
  const yesterdayStartTime = yesterdayStart.getTime();

  const groups: SessionListGroup<T>[] = [
    { key: "today", label: labels.today, items: [] },
    { key: "yesterday", label: labels.yesterday, items: [] },
    { key: "earlier", label: labels.earlier, items: [] },
  ];

  items.forEach((item) => {
    const timestamp = options.getTimestamp(item);
    if (Number.isFinite(timestamp) && timestamp >= todayStartTime) {
      groups[0].items.push(item);
      return;
    }
    if (Number.isFinite(timestamp) && timestamp >= yesterdayStartTime) {
      groups[1].items.push(item);
      return;
    }
    groups[2].items.push(item);
  });

  return groups.filter((group) => group.items.length > 0);
}
