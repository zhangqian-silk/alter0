import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
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
import { RuntimeWorkspaceHeader } from "../features/shell/components/RuntimeWorkspaceHeader";
import { RuntimeWorkspaceShell } from "../features/shell/components/RuntimeWorkspaceShell";
import { RuntimeRouteHost } from "../features/shell/components/RuntimeRouteHost";
import { ConversationRuntimeProvider } from "../features/conversation-runtime/ConversationRuntimeProvider";
import { ConversationSessionRailBridge } from "../features/conversation-runtime/ConversationSessionRailBridge";
import { createMobileViewportSyncController } from "../shared/viewport/mobileViewportSync";

function blurActiveRuntimeInput() {
  const active = document.activeElement;
  if (active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement) {
    active.blur();
  }
}

function MobileOverlayPortal({
  active,
  children,
}: {
  active: boolean;
  children: ReactNode;
}) {
  if (!active || typeof document === "undefined") {
    return null;
  }
  return createPortal(
    <div
      className={[
        "workbench-mobile-overlay-portal",
        "nav-open overlay-open",
      ].filter(Boolean).join(" ")}
      data-workbench-mobile-overlay-portal="true"
    >
      {children}
    </div>,
    document.body,
  );
}

export function WorkbenchApp() {
  /* Source contract markers:
     const [navOpen, setNavOpen] = useState(false);
     setNavOpen(false);
     setNavOpen((current) => !current);
  */
  const [route, navigate] = useWorkbenchRoute();
  const [language, setLanguage] = useState<LegacyShellLanguage>(() =>
    normalizeLegacyShellLanguage(document.documentElement.lang),
  );
  const [isMobileViewport, setIsMobileViewport] = useState(() => isLegacyShellMobileViewport());
  const [mobilePanel, setMobilePanel] = useState<"nav" | "sessions" | null>(null);
  const [runtimeSessionRail, setRuntimeSessionRailState] = useState<WorkbenchSessionRail | null>(null);
  const [settingsSessionRailHydrationActive, setSettingsSessionRailHydrationActive] = useState(false);
  const setRuntimeSessionRail = useCallback((rail: WorkbenchSessionRail | null) => {
    if (!rail) {
      return;
    }
    setRuntimeSessionRailState((current) => {
      if (current === rail) {
        return current;
      }
      if (rail.versionKey && current?.route === rail.route && current.versionKey === rail.versionKey) {
        return current;
      }
      return rail;
    });
  }, []);
  const runtimeRouteActive = isConversationRoute(route);
  const settingsRouteActive = route === "settings";
  useEffect(() => {
    if (!settingsRouteActive) {
      setSettingsSessionRailHydrationActive(false);
      return;
    }
    if (!runtimeSessionRail) {
      setSettingsSessionRailHydrationActive(true);
    }
  }, [runtimeSessionRail, settingsRouteActive]);
  const fallbackSessionRail = useMemo<WorkbenchSessionRail | null>(() => {
    const fallbackSelected = runtimeRouteActive;
    const newLabel = language === "zh" ? "新对话" : "New";
    return {
      route: "chat",
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
              <div role="listitem" className={fallbackSelected ? "runtime-session-card is-active" : "runtime-session-card"}>
                <button className={fallbackSelected ? "runtime-session-select active" : "runtime-session-select"} type="button" disabled>
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
  }, [language, runtimeRouteActive]);
  const visibleSessionRail = runtimeRouteActive
    ? (runtimeSessionRail ?? fallbackSessionRail)
    : settingsRouteActive
      ? runtimeSessionRail
      : null;
  const runtimeSessionsUseNav = Boolean(visibleSessionRail);
  const navOpen = mobilePanel === "nav";
  const sessionPaneOpen = !runtimeSessionsUseNav && mobilePanel === "sessions";
  const toggleMobileNav = useCallback(() => {
    setMobilePanel((current) => {
      const opening = current !== "nav";
      if (opening) {
        blurActiveRuntimeInput();
      }
      return opening ? "nav" : null;
    });
  }, []);
  const toggleMobileSessionPane = useCallback(() => {
    setMobilePanel((current) => {
      const nextPanel = runtimeSessionsUseNav ? "nav" : "sessions";
      const opening = current !== nextPanel;
      if (opening) {
        blurActiveRuntimeInput();
      }
      return opening ? nextPanel : null;
    });
  }, [runtimeSessionsUseNav]);
  const openMobileSessionPane = useCallback(() => {
    blurActiveRuntimeInput();
    setMobilePanel(runtimeSessionsUseNav ? "nav" : "sessions");
  }, [runtimeSessionsUseNav]);

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
    if (navOpen) {
      classNames.push("nav-open", "overlay-open");
    } else if (sessionPaneOpen) {
      classNames.push("overlay-open");
    }
    return classNames.join(" ");
  }, [navOpen, sessionPaneOpen]);

  const contextValue = useMemo(() => ({
    route,
    language,
    navigate,
    isMobileViewport,
    mobileNavOpen: navOpen,
    mobileSessionPaneOpen: sessionPaneOpen,
    toggleMobileNav,
    toggleMobileSessionPane,
    openMobileSessionPane,
    closeMobileNav: () => setMobilePanel((current) => current === "nav" ? null : current),
    closeMobileSessionPane: () => setMobilePanel((current) => {
      if (runtimeSessionsUseNav) {
        return current === "nav" ? null : current;
      }
      return current === "sessions" ? null : current;
    }),
    setRuntimeSessionRail,
  }), [
    route,
    language,
    navigate,
    isMobileViewport,
    navOpen,
    sessionPaneOpen,
    runtimeSessionsUseNav,
    toggleMobileNav,
    toggleMobileSessionPane,
    openMobileSessionPane,
  ]);
  const primaryNav = (
    <PrimaryNav
      currentRoute={route}
      language={language}
      sessionRail={visibleSessionRail}
      onNavigate={(nextRoute) => {
        navigate(nextRoute);
        if (isMobileViewport) {
          setMobilePanel(null);
        }
      }}
      onDismiss={isMobileViewport ? () => setMobilePanel(null) : undefined}
    />
  );
  const mobileBackdrop = (
    <button
      className="mobile-backdrop"
      type="button"
      aria-label="Close panels"
      onClick={() => setMobilePanel(null)}
    ></button>
  );

  return (
    <WorkbenchContext.Provider value={contextValue}>
      <div className={shellClassName} data-workbench-route={route}>
        {settingsRouteActive && settingsSessionRailHydrationActive ? (
          <ConversationRuntimeProvider route="chat" language={language}>
            <ConversationSessionRailBridge language={language} />
          </ConversationRuntimeProvider>
        ) : null}
        {isMobileViewport ? null : primaryNav}
        <main className="workbench-main">
          <div className="chat-pane page-mode workbench-pane-shell" data-route={route} data-workbench-pane-shell>
            {isConversationRoute(route) ? (
              <RuntimeRouteHost route={route} language={language} />
            ) : (
              <RoutePageFrame
                route={route}
                language={language}
                isMobileViewport={isMobileViewport}
                mobileNavOpen={navOpen}
                onToggleMobileNav={toggleMobileNav}
                onToggleLanguage={() => setLanguage((current) => current === "zh" ? "en" : "zh")}
              />
            )}
          </div>
        </main>
        {isMobileViewport ? null : mobileBackdrop}
      </div>
      {isMobileViewport ? (
        <MobileOverlayPortal active={navOpen || sessionPaneOpen}>
          {primaryNav}
          {mobileBackdrop}
        </MobileOverlayPortal>
      ) : null}
    </WorkbenchContext.Provider>
  );
}

function RoutePageFrame({
  route,
  language,
  isMobileViewport,
  mobileNavOpen,
  onToggleMobileNav,
  onToggleLanguage,
}: {
  route: string;
  language: LegacyShellLanguage;
  isMobileViewport: boolean;
  mobileNavOpen: boolean;
  onToggleMobileNav: () => void;
  onToggleLanguage: () => void;
}) {
  const isSettingsRoute = route === "settings";
  const routeHeadingCopy = getLegacyRouteHeadingCopy(language, isSettingsRoute ? "settings" : route);
  const shellCopy = getLegacyShellCopy(language);

  return (
    <RuntimeWorkspaceShell
      rootClassName="runtime-workspace-view"
      rootProps={{
        "data-route": route,
        "data-route-family": isSettingsRoute ? "settings" : undefined,
        "data-runtime-view": isSettingsRoute ? "settings" : route,
        "data-runtime-workspace-page": isSettingsRoute ? "settings" : undefined,
      }}
      sessionPanePlacement="navigation"
      sessionPaneProps={{
        "data-runtime-session-pane": route,
      }}
      sessionPaneBackdrop={{
        ariaLabel: "Close panels",
        onClick: () => undefined,
      }}
      sessionPaneTitle={routeHeadingCopy.title}
      sessionPaneCountLabel=""
      sessionPanePrimaryActionLabel={routeHeadingCopy.title}
      onSessionPanePrimaryAction={() => undefined}
      workspaceProps={{
        "data-route": route,
      }}
      workspaceBodyProps={{
        "data-route": route,
        "data-runtime-view": isSettingsRoute ? "settings" : route,
      }}
      mobileHeaderPlacement={isMobileViewport ? "body" : undefined}
      mobileNavButtonLabel={shellCopy.chatMenu}
      mobileNavButtonProps={{
        "aria-expanded": mobileNavOpen,
      }}
      onMobileNav={onToggleMobileNav}
      mobileTitleButtonLabel={routeHeadingCopy.title}
      mobileTitleButtonProps={{
        disabled: true,
      }}
      workspaceHeader={(
        <RuntimeWorkspaceHeader
          title={routeHeadingCopy.title}
          statusLabel={routeHeadingCopy.title}
          statusTone="ready"
          detailsLabel={routeHeadingCopy.title}
          detailsOpen={false}
          onToggleDetails={() => undefined}
          detailsDisabled
          showStatusSignal={false}
          showDetailsAction={false}
          headerProps={{
            className: "route-head",
            "data-workbench-title-head": "route",
          }}
        />
      )}
      workspaceContent={<ReactManagedRouteBody route={route} language={language} onToggleLanguage={onToggleLanguage} />}
    />
  );
}
