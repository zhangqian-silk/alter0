import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app/App";
import { installClickDiagnostics, isClickDiagnosticsEnabled } from "./shared/debug/clickDiagnostics";
import "./styles/root.css";
import "./styles/shell.css";
import "./styles/runtimeKeyboardIsolation.css";

const container = document.getElementById("frontend-root");

if (!container) {
  throw new Error("missing frontend bootstrap container");
}

if (isClickDiagnosticsEnabled()) {
  installClickDiagnostics();
}

createRoot(container).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
