import {
  LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT,
  LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT,
  LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT,
  LEGACY_SHELL_SYNC_SESSION_PANE_EVENT,
  type LegacyShellChatRuntimeDetail,
  type LegacyShellChatWorkspaceDetail,
  type LegacyShellMessageRegionDetail,
  type LegacyShellSessionPaneDetail,
} from "./legacyShellBridge";

export type SnapshotChannel =
  | typeof LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT
  | typeof LEGACY_SHELL_SYNC_SESSION_PANE_EVENT
  | typeof LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT
  | typeof LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT;

type SnapshotDetailMap = {
  [LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT]: LegacyShellChatWorkspaceDetail;
  [LEGACY_SHELL_SYNC_SESSION_PANE_EVENT]: LegacyShellSessionPaneDetail;
  [LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT]: LegacyShellMessageRegionDetail;
  [LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT]: LegacyShellChatRuntimeDetail;
};

type SnapshotListener = () => void;

declare global {
  interface Window {
    __alter0LegacyRuntime?: {
      publishChatWorkspaceSnapshot?: (detail: LegacyShellChatWorkspaceDetail) => boolean | void;
      publishSessionPaneSnapshot?: (detail: LegacyShellSessionPaneDetail) => boolean | void;
      publishMessageRegionSnapshot?: (detail: LegacyShellMessageRegionDetail) => boolean | void;
      publishChatRuntimeSnapshot?: (detail: LegacyShellChatRuntimeDetail) => boolean | void;
    };
  }
}

const snapshotStore = new Map<SnapshotChannel, SnapshotDetailMap[SnapshotChannel]>();
const snapshotListeners = new Map<SnapshotChannel, Set<SnapshotListener>>();

function notifySnapshotListeners(channel: SnapshotChannel) {
  const listeners = snapshotListeners.get(channel);
  if (!listeners) {
    return;
  }
  listeners.forEach((listener) => listener());
}

export function publishLegacyRuntimeSnapshot<TChannel extends SnapshotChannel>(
  channel: TChannel,
  detail: SnapshotDetailMap[TChannel],
) {
  snapshotStore.set(channel, detail);
  notifySnapshotListeners(channel);
}

export function readLegacyRuntimeSnapshot<TChannel extends SnapshotChannel>(
  channel: TChannel,
): SnapshotDetailMap[TChannel] | null {
  return (snapshotStore.get(channel) as SnapshotDetailMap[TChannel] | undefined) ?? null;
}

export function subscribeLegacyRuntimeSnapshot(
  channel: SnapshotChannel,
  listener: SnapshotListener,
): () => void {
  const listeners = snapshotListeners.get(channel) ?? new Set<SnapshotListener>();
  listeners.add(listener);
  snapshotListeners.set(channel, listeners);

  return () => {
    const current = snapshotListeners.get(channel);
    if (!current) {
      return;
    }
    current.delete(listener);
    if (current.size === 0) {
      snapshotListeners.delete(channel);
    }
  };
}

export function ensureLegacyRuntimeSnapshotBridge() {
  if (typeof window === "undefined") {
    return;
  }

  window.__alter0LegacyRuntime = Object.assign(window.__alter0LegacyRuntime || {}, {
    publishChatWorkspaceSnapshot(detail: LegacyShellChatWorkspaceDetail) {
      publishLegacyRuntimeSnapshot(LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT, detail);
      return true;
    },
    publishSessionPaneSnapshot(detail: LegacyShellSessionPaneDetail) {
      publishLegacyRuntimeSnapshot(LEGACY_SHELL_SYNC_SESSION_PANE_EVENT, detail);
      return true;
    },
    publishMessageRegionSnapshot(detail: LegacyShellMessageRegionDetail) {
      publishLegacyRuntimeSnapshot(LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT, detail);
      return true;
    },
    publishChatRuntimeSnapshot(detail: LegacyShellChatRuntimeDetail) {
      publishLegacyRuntimeSnapshot(LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT, detail);
      return true;
    },
  });
}

export function resetLegacyRuntimeSnapshotStore() {
  snapshotStore.clear();
  snapshotListeners.clear();
}
