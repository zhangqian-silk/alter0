import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { LegacyRuntimeBridge } from "./bootstrap/LegacyRuntimeBridge";
import "./styles/bridge.css";

const container = document.getElementById("frontend-root");

if (!container) {
  throw new Error("missing frontend bootstrap container");
}

createRoot(container).render(
  <StrictMode>
    <LegacyRuntimeBridge />
  </StrictMode>,
);
