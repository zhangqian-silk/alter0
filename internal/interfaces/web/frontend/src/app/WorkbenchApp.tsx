import { useCallback, useEffect, useMemo, useState } from "react";
import { WorkbenchContext, type WorkbenchSessionRail } from "./WorkbenchContext";
import { isConversationRoute, useWorkbenchRoute } from "./routeState";
import {
  getLegacyRouteHeadingCopy,
  getLegacyShellCopy,
  normalizeLegacyShellLanguage,
  type LegacyShellLanguage,
} from "../features/shell/legacyShellCopy";
import { isLegacyShellMobileViewport } from "../features/shell/legacyShellState";
import { PrimaryNav } from "../features/shell/components/PrimaryNav";
import { ReactManagedRouteBody } from "../features/shell/components/ReactManagedRouteBody";
import { RuntimeRouteHost } from "../features/shell/components/RuntimeRouteHost";
import { createMobileViewportSyncController } from "../shared/viewport/mobileViewportSync";

function RouteMobileMenuIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true" data-route-mobile-icon="menu">
      <path d="M4.5 6.25h11" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M4.5 10h11" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M4.5 13.75h11" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}

export function WorkbenchApp() {
  /* Source contract markers:
     const [navOpen, setNavOpen] = useState(false);
     setNavOpen(false);
     setNavOpen((current) => !current);
     route === "terminal" ? "route-body terminal-route-body" : "route-body"
  */
  const [route, navigate] = useWorkbenchRoute();
  const [language, setLanguage] = useState<LegacyShellLanguage>(() =>
    normalizeLegacyShellLanguage(document.documentElement.lang),
  );
  const [isMobileViewport, setIsMobileViewport] = useState(() => isLegacyShellMobileViewport());
  const [navCollapsed, setNavCollapsed] = useState(false);
  const [mobilePanel, setMobilePanel] = useState<"nav" | "sessions" | null>(null);
  const [runtimeSessionRail, setRuntimeSessionRailState] = useState<WorkbenchSessionRail | null>(null);
  const setRuntimeSessionRail = useCallback((rail: WorkbenchSessionRail | null) => {
    if (!rail) {
      return;
    }
    setRuntimeSessionRailState((current) => {
      if (current === rail) {
        return current;
      }
      return rail;
    });
  }, []);
  const runtimeRouteActive = isConversationRoute(route) || route === "terminal";
  const fallbackSessionRail = useMemo<WorkbenchSessionRail | null>(() => {
    if (!runtimeRouteActive) {
      return null;
    }
    const newLabel = getLegacyShellCopy(language).terminalNewShort;
    return {
      route,
      countLabel: language === "zh" ? "1 个会话" : "1 session",
      onPrimaryAction: () => undefined,
      primaryActionProps: {
        disabled: true,
        "aria-disabled": true,
      },
      body: (
        <div className="runtime-session-list" data-runtime-session-list-placeholder="true">
          <section className="runtime-session-group menu-group">
            <div className="runtime-session-group-items" role="list">
              <div role="listitem" className="runtime-session-card is-active">
                <button className="runtime-session-select active" type="button" disabled>
                  <span className="runtime-session-main">
                    <span className="runtime-session-title-row">
                      <span className="runtime-session-title-copy">
                        <span className="runtime-session-title">{newLabel}</span>
                      </span>
                    </span>
                  </span>
                </button>
              </div>
            </div>
          </section>
        </div>
      ),
    };
  }, [language, route, runtimeRouteActive]);
  const visibleSessionRail = runtimeRouteActive ? (runtimeSessionRail ?? fallbackSessionRail) : null;
  const runtimeSessionsUseNav = Boolean(visibleSessionRail);
  const navOpen = mobilePanel === "nav";
  const sessionPaneOpen = !runtimeSessionsUseNav && mobilePanel === "sessions";

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
    toggleMobileSessionPane: () => setMobilePanel((current) => {
      if (runtimeSessionsUseNav) {
        return current === "nav" ? null : "nav";
      }
      return current === "sessions" ? null : "sessions";
    }),
    openMobileSessionPane: () => setMobilePanel(runtimeSessionsUseNav ? "nav" : "sessions"),
    closeMobileNav: () => setMobilePanel((current) => current === "nav" ? null : current),
    closeMobileSessionPane: () => setMobilePanel((current) => {
      if (runtimeSessionsUseNav) {
        return current === "nav" ? null : current;
      }
      return current === "sessions" ? null : current;
    }),
    setRuntimeSessionRail,
  }), [route, language, navigate, isMobileViewport, navOpen, sessionPaneOpen, runtimeSessionsUseNav]);

  return (
    <WorkbenchContext.Provider value={contextValue}>
      <div className={shellClassName} data-workbench-route={route}>
        <PrimaryNav
          currentRoute={route}
          language={language}
          navCollapsed={navCollapsed}
          sessionRail={visibleSessionRail}
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
            {isConversationRoute(route) || route === "terminal" ? (
              <RuntimeRouteHost route={route} language={language} />
            ) : (
              <RoutePageFrame
                route={route}
                language={language}
                isMobileViewport={isMobileViewport}
                mobileNavOpen={navOpen}
                onToggleMobileNav={() => setMobilePanel((current) => current === "nav" ? null : "nav")}
              />
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
  isMobileViewport,
  mobileNavOpen,
  onToggleMobileNav,
}: {
  route: string;
  language: LegacyShellLanguage;
  isMobileViewport: boolean;
  mobileNavOpen: boolean;
  onToggleMobileNav: () => void;
}) {
  const isSettingsRoute = route === "settings" || route === "management";
  const routeHeadingCopy = getLegacyRouteHeadingCopy(language, isSettingsRoute ? "settings" : route);
  const shellCopy = getLegacyShellCopy(language);

  return (
    <section
      className={isSettingsRoute ? "route-view workbench-route-frame" : "route-view"}
      data-route={route}
      data-route-family={isSettingsRoute ? "management" : undefined}
    >
      {isMobileViewport ? (
        <header className="route-mobile-head" data-route-mobile-head>
          <button
            className="nav-toggle conversation-mobile-action is-quiet"
            type="button"
            aria-expanded={mobileNavOpen}
            onClick={onToggleMobileNav}
          >
            <RouteMobileMenuIcon />
            <span className="route-mobile-action-label sr-only">{shellCopy.chatMenu}</span>
          </button>
          <div className="route-mobile-title workbench-title-leading">
            <span className="route-title-marker" aria-hidden="true"></span>
            <h3>{routeHeadingCopy.title}</h3>
          </div>
          <span className="route-mobile-head-spacer" aria-hidden="true"></span>
        </header>
      ) : null}
      <header
        className="route-head workbench-title-head is-compact"
        data-workbench-title-head="route"
      >
        <div className="route-copy workbench-title-copy is-compact">
          <div className="route-title-leading workbench-title-leading">
            <span className="route-title-marker" aria-hidden="true"></span>
            <h3>{routeHeadingCopy.title}</h3>
          </div>
          <p id="routeSubtitle">{routeHeadingCopy.subtitle}</p>
        </div>
      </header>
      <div className="route-body" data-route={route}>
        <ReactManagedRouteBody route={route} language={language} />
      </div>
    </section>
  );
}
