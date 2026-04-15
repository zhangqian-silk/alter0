import { useEffect, useState } from "react";

type RoutedSnapshot = {
  route: string;
};

type RoutedDetail = {
  route?: string | null;
};

type LegacyShellSnapshotOptions<TDetail extends RoutedDetail, TSnapshot extends RoutedSnapshot> = {
  currentRoute: string;
  eventName: string;
  fallback: () => TSnapshot;
  normalizeDetail: (detail: TDetail) => TSnapshot | null;
};

export function useLegacyShellSnapshot<TDetail extends RoutedDetail, TSnapshot extends RoutedSnapshot>({
  currentRoute,
  eventName,
  fallback,
  normalizeDetail,
}: LegacyShellSnapshotOptions<TDetail, TSnapshot>): TSnapshot {
  const [snapshot, setSnapshot] = useState<TSnapshot | null>(null);

  useEffect(() => {
    const handleSnapshot = (event: Event) => {
      const detail = (event as CustomEvent<TDetail>).detail;
      const nextSnapshot = normalizeDetail(detail);
      if (nextSnapshot) {
        setSnapshot(nextSnapshot);
      }
    };

    document.addEventListener(eventName, handleSnapshot as EventListener);
    return () => {
      document.removeEventListener(eventName, handleSnapshot as EventListener);
    };
  }, [eventName, normalizeDetail]);

  if (snapshot?.route === currentRoute) {
    return snapshot;
  }

  return fallback();
}
