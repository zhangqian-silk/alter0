import { useEffect } from "react";
import { ensureLegacyRuntimeScript } from "./loadLegacyRuntime";

export function LegacyRuntimeBridge() {
  useEffect(() => {
    ensureLegacyRuntimeScript();
  }, []);

  return null;
}
