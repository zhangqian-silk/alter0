import { useEffect, useMemo, useState } from "react";
import { WorkbenchContext } from "./WorkbenchContext";
import { isConversationRoute, useWorkbenchRoute } from "./routeState";
import { ConversationRuntimeProvider } from "../features/conversation-runtime/ConversationRuntimeProvider";
import { ConversationWorkspace } from "../features/conversation-runtime/ConversationWorkspace";
import {
  getLegacyRouteHeadingCopy,
  normalizeLegacyShellLanguage,
  type LegacyShellLanguage,
} from "../features/shell/legacyShellCopy";
import { isLegacyShellMobileViewport } from "../features/shell/legacyShellState";
import { PrimaryNav } from "../features/shell/components/PrimaryNav";
import { ReactManagedRouteBody } from "../features/shell/components/ReactManagedRouteBody";
import { createMobileViewportSyncController } from "../shared/viewport/mobileViewportSync";

export function WorkbenchApp() {
  const [route, navigate] = useWorkbenchRoute();
  const [language, setLanguage] = useState<LegacyShellLanguage>(() =>
    normalizeLegacyShellLanguage(document.documentElement.lang),
  );
  const [isMobileViewport, setIsMobileViewport] = useState(() => isLegacyShellMobileViewport());
  const [navCollapsed, setNavCollapsed] = useState(false);
  const [mobilePanel, setMobilePanel] = useState<"nav" | "sessions" | null>(null);
  const navOpen = mobilePanel === "nav";
  const sessionPaneOpen = mobilePanel === "sessions";

  useEffect(() => {
    document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
  }, [language]);

  useEffect(() => {
    const syncViewport = () => {
      const mobile = isLegacyShellMobileViewport();
      setIsMobileViewport(mobile);
      if (!mobile) {
        setMobilePanel(null);
      }
    };
    syncViewport();
    window.addEventListener("resize", syncViewport);
    return () => window.removeEventListener("resize", syncViewport);
  }, []);

  useEffect(() => {
    const controller = createMobileViewportSyncController();
    return () => controller.destroy();
  }, []);

  const shellClassName = useMemo(() => {
    const classNames = ["app-shell", "info-mode"];
    if (navCollapsed) {
      classNames.push("nav-collapsed");
    }
    if (navOpen) {
      classNames.push("nav-open", "overlay-open");
    } else if (sessionPaneOpen) {
      classNames.push("overlay-open");
    }
    return classNames.join(" ");
  }, [navCollapsed, navOpen, sessionPaneOpen]);

  const contextValue = useMemo(() => ({
    route,
    language,
    navigate,
    isMobileViewport,
    mobileNavOpen: navOpen,
    mobileSessionPaneOpen: sessionPaneOpen,
    toggleMobileNav: () => setMobilePanel((current) => current === "nav" ? null : "nav"),
    toggleMobileSessionPane: () => setMobilePanel((current) => current === "sessions" ? null : "sessions"),
    closeMobileNav: () => setMobilePanel((current) => current === "nav" ? null : current),
    closeMobileSessionPane: () => setMobilePanel((current) => current === "sessions" ? null : current),
  }), [route, language, navigate, isMobileViewport, navOpen, sessionPaneOpen]);

  return (
    <WorkbenchContext.Provider value={contextValue}>
      <div className={shellClassName} data-workbench-route={route}>
        <PrimaryNav
          currentRoute={route}
          language={language}
          navCollapsed={navCollapsed}
          onNavigate={(nextRoute) => {
            navigate(nextRoute);
            if (isMobileViewport) {
              setMobilePanel(null);
            }
          }}
          onToggleLanguage={() => setLanguage((current) => current === "zh" ? "en" : "zh")}
          onToggleNavCollapsed={() => {
            if (isMobileViewport) {
              setMobilePanel((current) => current === "nav" ? null : "nav");
              return;
            }
            setNavCollapsed((current) => !current);
          }}
        />
        <main className="workbench-main">
          <div className="chat-pane page-mode workbench-pane-shell" data-route={route} data-workbench-pane-shell>
            {isConversationRoute(route) ? (
              <ConversationRuntimeProvider route={route} language={language}>
                <ConversationWorkspace language={language} />
              </ConversationRuntimeProvider>
            ) : route === "terminal" ? (
              <ReactManagedRouteBody route={route} language={language} />
            ) : (
              <RoutePageFrame route={route} language={language} />
            )}
          </div>
        </main>
        <button
          className="mobile-backdrop"
          type="button"
          aria-label="Close panels"
          onClick={() => setMobilePanel(null)}
        ></button>
      </div>
    </WorkbenchContext.Provider>
  );
}

function RoutePageFrame({
  route,
  language,
}: {
  route: string;
  language: LegacyShellLanguage;
}) {
  const routeHeadingCopy = getLegacyRouteHeadingCopy(language, route);

  return (
    <section className="route-view" data-route={route}>
      <header className="route-head">
        <div className="route-copy">
          <h3>{routeHeadingCopy.title}</h3>
          <p id="routeSubtitle">{routeHeadingCopy.subtitle}</p>
        </div>
      </header>
      <div className="route-body" data-route={route}>
        <ReactManagedRouteBody route={route} language={language} />
      </div>
    </section>
  );
}
