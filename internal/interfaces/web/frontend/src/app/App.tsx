import { LegacyRuntimeBridge } from "../bootstrap/LegacyRuntimeBridge";
import { LegacyWebShell } from "../features/shell/LegacyWebShell";

export function App() {
  return (
    <>
      <LegacyWebShell />
      <LegacyRuntimeBridge />
    </>
  );
}
