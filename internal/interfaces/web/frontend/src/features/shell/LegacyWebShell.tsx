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
import {
  requestLegacyShellLanguageToggle,
  requestLegacyShellQuickPrompt,
  requestLegacyShellRouteNavigation,
  requestLegacyShellSessionCreation,
  syncLegacyShellNavCollapsed,
  syncLegacyShellSessionHistoryCollapsed,
} from "./legacyShellBridge";
import {
  dismissLegacyTransientPanels,
  readLegacyTransientShellClasses,
  toggleLegacyNavDrawer,
  toggleLegacySessionPane,
  type LegacyTransientShellClass,
} from "./legacyTransientShell";
import { ChatRuntimeSheetHost } from "./components/ChatRuntimeHost";
import { ChatWorkspace } from "./components/ChatWorkspace";
import { PrimaryNav } from "./components/PrimaryNav";
import { SessionPane } from "./components/SessionPane";

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
        setNavCollapsed((collapsed) => {
          if (!collapsed) {
            return collapsed;
          }
          syncLegacyShellNavCollapsed(false);
          return false;
        });
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

  const getShellRoot = () => document.getElementById(LEGACY_SHELL_IDS.appShell);

  return (
    <div className={shellClassName} id={LEGACY_SHELL_IDS.appShell}>
      <PrimaryNav
        currentRoute={currentRoute}
        language={language}
        navCollapsed={navCollapsed}
        onNavigate={requestLegacyShellRouteNavigation}
        onToggleLanguage={requestLegacyShellLanguageToggle}
        onToggleNavCollapsed={() => {
          if (isLegacyShellMobileViewport()) {
            return;
          }
          setNavCollapsed((collapsed) => {
            const next = !collapsed;
            syncLegacyShellNavCollapsed(next);
            return next;
          });
        }}
      />
      <SessionPane
        currentRoute={currentRoute}
        language={language}
        sessionHistoryCollapsed={sessionHistoryCollapsed}
        onCreateSession={requestLegacyShellSessionCreation}
        onClosePane={() => {
          const root = getShellRoot();
          if (!root) {
            return;
          }
          dismissLegacyTransientPanels(root);
        }}
        onToggleSessionHistoryCollapsed={() => {
          setSessionHistoryCollapsed((collapsed) => {
            const next = !collapsed;
            persistLegacySessionHistoryCollapsed(next);
            syncLegacyShellSessionHistoryCollapsed(next);
            return next;
          });
        }}
      />
      <ChatWorkspace
        currentRoute={currentRoute}
        language={language}
        onCreateSession={requestLegacyShellSessionCreation}
        onNavigate={requestLegacyShellRouteNavigation}
        onQuickPrompt={requestLegacyShellQuickPrompt}
        onToggleNavDrawer={() => {
          const root = getShellRoot();
          if (!root) {
            return;
          }
          toggleLegacyNavDrawer(root);
        }}
        onToggleSessionPane={() => {
          const root = getShellRoot();
          if (!root) {
            return;
          }
          toggleLegacySessionPane(root, currentRoute);
        }}
      />

      <ChatRuntimeSheetHost currentRoute={currentRoute} />
      <button
        className="mobile-backdrop"
        id={LEGACY_SHELL_IDS.mobileBackdrop}
        type="button"
        aria-label="Close panels"
        onClick={() => {
          const root = getShellRoot();
          if (!root) {
            return;
          }
          dismissLegacyTransientPanels(root);
        }}
      ></button>
    </div>
  );
}
