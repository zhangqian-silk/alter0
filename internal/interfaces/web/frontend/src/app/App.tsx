import { ReactRuntimeFacade } from "../bootstrap/ReactRuntimeFacade";
import { LegacyWebShell } from "../features/shell/LegacyWebShell";

export function App() {
  return (
    <>
      <LegacyWebShell />
      <ReactRuntimeFacade />
    </>
  );
}
