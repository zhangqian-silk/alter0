import { LEGACY_SHELL_IDS } from "./legacyDomContract";
import { ChatRuntimeSheetHost } from "./components/ChatRuntimeHost";
import { ChatWorkspace } from "./components/ChatWorkspace";
import { PrimaryNav } from "./components/PrimaryNav";
import { SessionPane } from "./components/SessionPane";

export function LegacyWebShell() {
  return (
    <div className="app-shell" id={LEGACY_SHELL_IDS.appShell}>
      <PrimaryNav />
      <SessionPane />
      <ChatWorkspace />

      <ChatRuntimeSheetHost />
      <button className="mobile-backdrop" id={LEGACY_SHELL_IDS.mobileBackdrop} type="button" aria-label="Close panels"></button>
    </div>
  );
}
