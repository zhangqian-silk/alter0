import { useMemo, useSyncExternalStore } from "react";
import {
  ensureLegacyRuntimeSnapshotBridge,
  readLegacyRuntimeSnapshot,
  subscribeLegacyRuntimeSnapshot,
  type SnapshotChannel,
} from "./legacyRuntimeSnapshotStore";

type RoutedSnapshot = {
  route: string;
};

type RoutedDetail = {
  route?: string | null;
};

type LegacyShellSnapshotOptions<TDetail extends RoutedDetail, TSnapshot extends RoutedSnapshot> = {
  currentRoute: string;
  eventName: SnapshotChannel;
  fallback: () => TSnapshot;
  normalizeDetail: (detail: TDetail) => TSnapshot | null;
};

export function useLegacyShellSnapshot<TDetail extends RoutedDetail, TSnapshot extends RoutedSnapshot>({
  currentRoute,
  eventName,
  fallback,
  normalizeDetail,
}: LegacyShellSnapshotOptions<TDetail, TSnapshot>): TSnapshot {
  ensureLegacyRuntimeSnapshotBridge();

  const detail = useSyncExternalStore(
    (listener) => subscribeLegacyRuntimeSnapshot(eventName, listener),
    () => readLegacyRuntimeSnapshot(eventName) as TDetail | null,
    () => null,
  );
  const snapshot = useMemo(() => {
    if (!detail) {
      return null;
    }
    return normalizeDetail(detail);
  }, [detail, normalizeDetail]);

  if (snapshot?.route === currentRoute) {
    return snapshot;
  }

  return fallback();
}
