import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app/App";
import "./styles/bridge.css";
import "./styles/shell.css";

const container = document.getElementById("frontend-root");

if (!container) {
  throw new Error("missing frontend bootstrap container");
}

createRoot(container).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
