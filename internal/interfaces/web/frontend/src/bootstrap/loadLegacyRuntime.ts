const LEGACY_RUNTIME_SCRIPT_ID = "alter0-legacy-runtime";
const LEGACY_RUNTIME_SCRIPT_SRC = "/legacy/chat.js";

export function ensureLegacyRuntimeScript(src: string = LEGACY_RUNTIME_SCRIPT_SRC): HTMLScriptElement {
  const existing = document.getElementById(LEGACY_RUNTIME_SCRIPT_ID);
  if (existing instanceof HTMLScriptElement) {
    return existing;
  }

  const script = document.createElement("script");
  script.id = LEGACY_RUNTIME_SCRIPT_ID;
  script.src = src;
  script.defer = true;
  script.dataset.runtimeBridge = "legacy";
  document.body.appendChild(script);
  return script;
}
