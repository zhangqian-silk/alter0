import { useEffect, useRef } from "react";
import { ensureLegacyRuntimeScript } from "../../../bootstrap/loadLegacyRuntime";

declare global {
  interface Window {
    __alter0LegacyRuntime?: {
      mountTerminalRoute?: (container: HTMLElement) => void;
    };
  }
}

export function ReactManagedTerminalRouteBody() {
  const hostRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) {
      return;
    }

    const mount = () => {
      const runtime = window.__alter0LegacyRuntime;
      if (typeof runtime?.mountTerminalRoute !== "function") {
        return false;
      }
      runtime.mountTerminalRoute(host);
      return true;
    };

    const script = ensureLegacyRuntimeScript();
    if (mount()) {
      return;
    }

    const handleLoad = () => {
      mount();
    };

    script.addEventListener("load", handleLoad);
    return () => {
      script.removeEventListener("load", handleLoad);
    };
  }, []);

  return (
    <div ref={hostRef} data-legacy-terminal-host="true">
      <section className="terminal-view" data-terminal-view>
        <aside className="terminal-session-pane" data-terminal-session-pane></aside>
        <section className="terminal-workspace"></section>
      </section>
    </div>
  );
}
