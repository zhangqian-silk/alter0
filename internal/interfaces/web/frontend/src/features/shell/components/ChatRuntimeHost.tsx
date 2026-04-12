import { memo } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";

export function ChatRuntimeHost() {
  return (
    <div className="composer-runtime-bar" id={LEGACY_SHELL_IDS.chatRuntimePanel}>
      <div data-runtime-controls-root></div>
      <div data-runtime-note-root></div>
    </div>
  );
}

export const ChatRuntimeSheetHost = memo(function ChatRuntimeSheetHost() {
  return (
    <div className="runtime-sheet-host" id={LEGACY_SHELL_IDS.chatRuntimeSheetHost}>
      <div data-runtime-sheet-root></div>
    </div>
  );
});
