import { useEffect, useMemo, useState } from "react";
import { LEGACY_SHELL_IDS } from "./legacyDomContract";
import { useLegacyShellLanguage } from "./legacyShellCopy";
import {
  isLegacyShellChatRoute,
  isLegacyShellMobileViewport,
  loadLegacySessionHistoryCollapsed,
  persistLegacySessionHistoryCollapsed,
  useLegacyShellRoute,
} from "./legacyShellState";
import { ChatRuntimeSheetHost } from "./components/ChatRuntimeHost";
import { ChatWorkspace } from "./components/ChatWorkspace";
import { PrimaryNav } from "./components/PrimaryNav";
import { SessionPane } from "./components/SessionPane";

const LEGACY_TRANSIENT_SHELL_CLASSES = [
  "nav-open",
  "panel-open",
  "overlay-open",
  "runtime-sheet-open",
] as const;

type LegacyTransientShellClass = (typeof LEGACY_TRANSIENT_SHELL_CLASSES)[number];

function readLegacyTransientShellClasses(root: HTMLElement | null): LegacyTransientShellClass[] {
  if (!root) {
    return [];
  }

  return LEGACY_TRANSIENT_SHELL_CLASSES.filter((className) => root.classList.contains(className));
}

export function LegacyWebShell() {
  const currentRoute = useLegacyShellRoute();
  const language = useLegacyShellLanguage();
  const [navCollapsed, setNavCollapsed] = useState(false);
  const [sessionHistoryCollapsed, setSessionHistoryCollapsed] = useState(() =>
    loadLegacySessionHistoryCollapsed(),
  );
  const [legacyTransientClasses, setLegacyTransientClasses] = useState<LegacyTransientShellClass[]>([]);

  useEffect(() => {
    const syncViewport = () => {
      if (isLegacyShellMobileViewport()) {
        setNavCollapsed(false);
      }
    };

    window.addEventListener("resize", syncViewport);
    return () => window.removeEventListener("resize", syncViewport);
  }, []);

  useEffect(() => {
    const root = document.getElementById(LEGACY_SHELL_IDS.appShell);
    if (!root) {
      return;
    }

    const syncLegacyClasses = () => {
      setLegacyTransientClasses(readLegacyTransientShellClasses(root));
    };

    syncLegacyClasses();

    const observer = new MutationObserver((mutations) => {
      if (mutations.some((mutation) => mutation.attributeName === "class")) {
        syncLegacyClasses();
      }
    });

    observer.observe(root, {
      attributes: true,
      attributeFilter: ["class"],
    });

    return () => observer.disconnect();
  }, [currentRoute, language, navCollapsed, sessionHistoryCollapsed]);

  const shellClassName = useMemo(() => {
    const nextClassNames = ["app-shell"];
    if (navCollapsed) {
      nextClassNames.push("nav-collapsed");
    }
    if (!isLegacyShellChatRoute(currentRoute)) {
      nextClassNames.push("info-mode");
    }
    nextClassNames.push(...legacyTransientClasses);
    return nextClassNames.join(" ");
  }, [currentRoute, legacyTransientClasses, navCollapsed]);

  return (
    <div className={shellClassName} id={LEGACY_SHELL_IDS.appShell}>
      <PrimaryNav
        currentRoute={currentRoute}
        language={language}
        navCollapsed={navCollapsed}
        onToggleNavCollapsed={() => {
          if (isLegacyShellMobileViewport()) {
            return;
          }
          setNavCollapsed((collapsed) => !collapsed);
        }}
      />
      <SessionPane
        currentRoute={currentRoute}
        language={language}
        sessionHistoryCollapsed={sessionHistoryCollapsed}
        onToggleSessionHistoryCollapsed={() => {
          setSessionHistoryCollapsed((collapsed) => {
            const next = !collapsed;
            persistLegacySessionHistoryCollapsed(next);
            return next;
          });
        }}
      />
      <ChatWorkspace currentRoute={currentRoute} language={language} />

      <ChatRuntimeSheetHost />
      <button className="mobile-backdrop" id={LEGACY_SHELL_IDS.mobileBackdrop} type="button" aria-label="Close panels"></button>
    </div>
  );
}
