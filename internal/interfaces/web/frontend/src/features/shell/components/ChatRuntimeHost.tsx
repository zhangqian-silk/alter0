import { memo, useEffect, useLayoutEffect, useRef, useState } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import {
  LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT,
  type LegacyShellChatRuntimeDetail,
} from "../legacyShellBridge";

type ChatRuntimeSnapshot = {
  route: string;
  controlsHTML: string;
  noteHTML: string;
  sheetHTML: string;
  scrollPopover: string;
  scrollTop: number;
};

function useLegacyChatRuntimeSnapshot(currentRoute: string): ChatRuntimeSnapshot {
  const [snapshot, setSnapshot] = useState<ChatRuntimeSnapshot | null>(null);

  useEffect(() => {
    const handleSnapshot = (event: Event) => {
      const detail = (event as CustomEvent<LegacyShellChatRuntimeDetail>).detail;
      if (!detail || typeof detail.route !== "string") {
        return;
      }

      setSnapshot({
        route: detail.route,
        controlsHTML: typeof detail.controlsHTML === "string" ? detail.controlsHTML : "",
        noteHTML: typeof detail.noteHTML === "string" ? detail.noteHTML : "",
        sheetHTML: typeof detail.sheetHTML === "string" ? detail.sheetHTML : "",
        scrollPopover: typeof detail.scrollPopover === "string" ? detail.scrollPopover : "",
        scrollTop: Number.isFinite(detail.scrollTop) ? Math.max(Number(detail.scrollTop), 0) : 0,
      });
    };

    document.addEventListener(LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT, handleSnapshot as EventListener);
    return () => {
      document.removeEventListener(LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT, handleSnapshot as EventListener);
    };
  }, []);

  if (snapshot?.route === currentRoute) {
    return snapshot;
  }

  return {
    route: currentRoute,
    controlsHTML: "",
    noteHTML: "",
    sheetHTML: "",
    scrollPopover: "",
    scrollTop: 0,
  };
}

function restoreRuntimeScrollPosition(root: HTMLDivElement | null, scrollPopover: string, scrollTop: number) {
  if (!root || !scrollPopover) {
    return;
  }

  const container = [...root.querySelectorAll<HTMLElement>("[data-runtime-scroll-container]")]
    .find((node) => node.getAttribute("data-runtime-scroll-container") === scrollPopover);

  if (!container) {
    return;
  }

  container.scrollTop = scrollTop;
}

const RuntimeHTMLMount = memo(function RuntimeHTMLMount({
  html,
  scrollPopover,
  scrollTop,
  className,
  id,
  runtimeRoot,
}: {
  html: string;
  scrollPopover: string;
  scrollTop: number;
  className?: string;
  id?: string;
  runtimeRoot: "controls" | "note" | "sheet";
}) {
  const ref = useRef<HTMLDivElement | null>(null);

  useLayoutEffect(() => {
    restoreRuntimeScrollPosition(ref.current, scrollPopover, scrollTop);
  }, [html, scrollPopover, scrollTop]);

  return (
    <div
      ref={ref}
      className={className}
      id={id}
      data-runtime-controls-root={runtimeRoot === "controls" ? "" : undefined}
      data-runtime-note-root={runtimeRoot === "note" ? "" : undefined}
      data-runtime-sheet-root={runtimeRoot === "sheet" ? "" : undefined}
      dangerouslySetInnerHTML={{ __html: html }}
    ></div>
  );
});

export function ChatRuntimeHost({ currentRoute }: { currentRoute: string }) {
  const snapshot = useLegacyChatRuntimeSnapshot(currentRoute);

  return (
    <div className="composer-runtime-bar" id={LEGACY_SHELL_IDS.chatRuntimePanel}>
      <RuntimeHTMLMount
        html={snapshot.controlsHTML}
        runtimeRoot="controls"
        scrollPopover={snapshot.scrollPopover}
        scrollTop={snapshot.scrollTop}
      />
      <RuntimeHTMLMount
        html={snapshot.noteHTML}
        runtimeRoot="note"
        scrollPopover={snapshot.scrollPopover}
        scrollTop={snapshot.scrollTop}
      />
    </div>
  );
}

export const ChatRuntimeSheetHost = memo(function ChatRuntimeSheetHost({
  currentRoute,
}: {
  currentRoute: string;
}) {
  const snapshot = useLegacyChatRuntimeSnapshot(currentRoute);

  return (
    <div className="runtime-sheet-host" id={LEGACY_SHELL_IDS.chatRuntimeSheetHost}>
      <RuntimeHTMLMount
        html={snapshot.sheetHTML}
        runtimeRoot="sheet"
        scrollPopover={snapshot.scrollPopover}
        scrollTop={snapshot.scrollTop}
      />
    </div>
  );
});
