import { getMissingLegacyShellIds } from "../features/shell/legacyDomContract";

const LEGACY_RUNTIME_SCRIPT_ID = "alter0-legacy-runtime";
const LEGACY_RUNTIME_SCRIPT_SRC = "/legacy/chat.js";

export function ensureLegacyRuntimeScript(src: string = LEGACY_RUNTIME_SCRIPT_SRC): HTMLScriptElement {
  const existing = document.getElementById(LEGACY_RUNTIME_SCRIPT_ID);
  if (existing instanceof HTMLScriptElement) {
    return existing;
  }

  const missingIds = getMissingLegacyShellIds();
  if (missingIds.length > 0) {
    throw new Error(`legacy shell contract is incomplete: ${missingIds.join(", ")}`);
  }

  const script = document.createElement("script");
  script.id = LEGACY_RUNTIME_SCRIPT_ID;
  script.src = src;
  script.defer = true;
  script.dataset.runtimeBridge = "legacy";
  document.body.appendChild(script);
  return script;
}
