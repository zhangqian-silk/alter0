import { memo, useEffect, useState } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import {
  getLegacyRouteHeadingCopy,
  getLegacyShellCopy,
  type LegacyShellLanguage,
} from "../legacyShellCopy";
import { isLegacyShellChatRoute } from "../legacyShellState";
import {
  LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT,
  LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT,
  LEGACY_SHELL_SYNC_ROUTE_BODY_EVENT,
  type LegacyShellChatWorkspaceDetail,
  type LegacyShellMessageRegionDetail,
  type LegacyShellRouteBodyDetail,
} from "../legacyShellBridge";
import { ChatRuntimeHost } from "./ChatRuntimeHost";
import { PROMPTS } from "../legacyShellConfig";

type ChatWorkspaceProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  onCreateSession: () => void;
  onNavigate: (route: string) => void;
  onQuickPrompt: (prompt: string) => void;
  onToggleNavDrawer: () => void;
  onToggleSessionPane: () => void;
};

type ChatWorkspaceSnapshot = {
  route: string;
  heading: string;
  subheading: string;
  welcomeHeading: string;
  welcomeDescription: string;
  welcomeTargetHTML: string;
};

type MessageRegionSnapshot = {
  route: string;
  hasMessages: boolean;
  sessionId: string;
  html: string;
};

type RouteBodySnapshot = {
  route: string;
  managed: boolean;
  html: string;
};

function getDefaultChatWorkspaceSnapshot(
  currentRoute: string,
  language: LegacyShellLanguage,
): ChatWorkspaceSnapshot {
  const copy = getLegacyShellCopy(language);
  const routeCopy = getLegacyRouteHeadingCopy(language, currentRoute);

  if (!isLegacyShellChatRoute(currentRoute)) {
    return {
      route: currentRoute,
      heading: "alter0",
      subheading: copy.chatMenu,
      welcomeHeading: "Hello, how can I help you today?",
      welcomeDescription: "I am a helpful assistant that can help you with your questions.",
      welcomeTargetHTML: "",
    };
  }

  return {
    route: currentRoute,
    heading: routeCopy.title,
    subheading: routeCopy.subtitle,
    welcomeHeading: "Hello, how can I help you today?",
    welcomeDescription: "I am a helpful assistant that can help you with your questions.",
    welcomeTargetHTML: "",
  };
}

function useLegacyChatWorkspaceSnapshot(
  currentRoute: string,
  language: LegacyShellLanguage,
): ChatWorkspaceSnapshot {
  const [snapshot, setSnapshot] = useState<ChatWorkspaceSnapshot | null>(null);

  useEffect(() => {
    const handleSnapshot = (event: Event) => {
      const detail = (event as CustomEvent<LegacyShellChatWorkspaceDetail>).detail;
      if (!detail || typeof detail.route !== "string") {
        return;
      }
      setSnapshot({
        route: detail.route,
        heading: detail.heading,
        subheading: detail.subheading,
        welcomeHeading: detail.welcomeHeading,
        welcomeDescription: detail.welcomeDescription,
        welcomeTargetHTML: typeof detail.welcomeTargetHTML === "string" ? detail.welcomeTargetHTML : "",
      });
    };

    document.addEventListener(LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT, handleSnapshot as EventListener);
    return () => {
      document.removeEventListener(LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT, handleSnapshot as EventListener);
    };
  }, []);

  if (snapshot?.route === currentRoute) {
    return snapshot;
  }

  return getDefaultChatWorkspaceSnapshot(currentRoute, language);
}

function useLegacyMessageRegionSnapshot(currentRoute: string): MessageRegionSnapshot {
  const [snapshot, setSnapshot] = useState<MessageRegionSnapshot | null>(null);

  useEffect(() => {
    const handleSnapshot = (event: Event) => {
      const detail = (event as CustomEvent<LegacyShellMessageRegionDetail>).detail;
      if (!detail || typeof detail.route !== "string") {
        return;
      }
      setSnapshot({
        route: detail.route,
        hasMessages: Boolean(detail.hasMessages),
        sessionId: typeof detail.sessionId === "string" ? detail.sessionId : "",
        html: typeof detail.html === "string" ? detail.html : "",
      });
    };

    document.addEventListener(LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT, handleSnapshot as EventListener);
    return () => {
      document.removeEventListener(LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT, handleSnapshot as EventListener);
    };
  }, []);

  if (snapshot?.route === currentRoute) {
    return snapshot;
  }

  return {
    route: currentRoute,
    hasMessages: false,
    sessionId: "",
    html: "",
  };
}

function useLegacyRouteBodySnapshot(currentRoute: string): RouteBodySnapshot {
  const [snapshot, setSnapshot] = useState<RouteBodySnapshot | null>(null);

  useEffect(() => {
    const handleSnapshot = (event: Event) => {
      const detail = (event as CustomEvent<LegacyShellRouteBodyDetail>).detail;
      if (!detail || typeof detail.route !== "string") {
        return;
      }
      setSnapshot({
        route: detail.route,
        managed: Boolean(detail.managed),
        html: typeof detail.html === "string" ? detail.html : "",
      });
    };

    document.addEventListener(LEGACY_SHELL_SYNC_ROUTE_BODY_EVENT, handleSnapshot as EventListener);
    return () => {
      document.removeEventListener(LEGACY_SHELL_SYNC_ROUTE_BODY_EVENT, handleSnapshot as EventListener);
    };
  }, []);

  if (snapshot?.route === currentRoute) {
    return snapshot;
  }

  return {
    route: currentRoute,
    managed: false,
    html: "",
  };
}

const ChatHeaderTitleMount = memo(function ChatHeaderTitleMount({
  heading,
  subheading,
}: {
  heading: string;
  subheading: string;
}) {
  return (
    <div className="chat-header-copy">
      <h2 id="sessionHeading">{heading}</h2>
      <p id="sessionSubheading">{subheading}</p>
    </div>
  );
});

const ChatView = memo(function ChatView({
  currentRoute,
  hidden,
  hasMessages,
  messageRegionHTML,
  welcomeHeading,
  welcomeDescription,
  welcomeTargetHTML,
  onQuickPrompt,
}: {
  currentRoute: string;
  hidden: boolean;
  hasMessages: boolean;
  messageRegionHTML: string;
  welcomeHeading: string;
  welcomeDescription: string;
  welcomeTargetHTML: string;
  onQuickPrompt: (prompt: string) => void;
}) {
  return (
    <div className="chat-view" id="chatView" hidden={hidden}>
      <section className="welcome-screen" id={LEGACY_SHELL_IDS.welcomeScreen} hidden={hasMessages}>
        <p className="welcome-tag" data-i18n="welcome.tag">alter0 assistant</p>
        <h3 id="welcomeHeading">{welcomeHeading}</h3>
        <p id="welcomeDescription">{welcomeDescription}</p>
        <div id="welcomeTargetList" dangerouslySetInnerHTML={{ __html: welcomeTargetHTML }}></div>
        <div className="prompt-grid">
          {PROMPTS.map((item) => (
            <button
              key={item.i18n}
              className="prompt"
              type="button"
              data-prompt={item.prompt}
              onClick={() => onQuickPrompt(item.prompt)}
            >
              <span data-i18n={item.i18n}>{item.label}</span>
            </button>
          ))}
        </div>
      </section>

      <section
        className="message-area"
        id={LEGACY_SHELL_IDS.messageArea}
        aria-live="polite"
        hidden={!hasMessages}
      >
        {messageRegionHTML ? (
          <div dangerouslySetInnerHTML={{ __html: messageRegionHTML }}></div>
        ) : null}
      </section>

      <footer className="composer-shell">
        <form className="composer" id="chatForm" data-composer-form="chat-main">
          <label className="sr-only" htmlFor="composerInput">Input your message</label>
          <textarea
            id={LEGACY_SHELL_IDS.composerInput}
            maxLength={10000}
            placeholder="Input your message here..."
            rows={2}
            data-i18n="composer.placeholder"
            data-composer-input="chat-main"
          ></textarea>
          <div className="composer-actions">
            <ChatRuntimeHost currentRoute={currentRoute} />
            <div className="composer-submit-bar">
              <span id={LEGACY_SHELL_IDS.charCount} data-composer-counter="chat-main">0/10000</span>
              <button id="sendButton" type="submit" aria-label="Send message" data-i18n="composer.send" data-composer-submit="chat-main">
                Send
              </button>
            </div>
          </div>
        </form>
        <p className="composer-note" data-i18n="composer.note">Works for you, grows with you</p>
      </footer>
    </div>
  );
});

type RouteViewMountProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  hidden: boolean;
  onRouteAction: (route: string) => void;
  routeBodySnapshot: RouteBodySnapshot;
};

const RouteViewMount = memo(function RouteViewMount({
  currentRoute,
  language,
  hidden,
  onRouteAction,
  routeBodySnapshot,
}: RouteViewMountProps) {
  const routeViewClassName = currentRoute === "terminal" ? "route-view terminal-route" : "route-view";
  const routeBodyClassName = currentRoute === "terminal" ? "route-body terminal-route-body" : "route-body";
  const routeHeadingCopy = getLegacyRouteHeadingCopy(language, currentRoute);

  return (
    <section className={routeViewClassName} id={LEGACY_SHELL_IDS.routeView} data-route={currentRoute} hidden={hidden}>
      <header className="route-head">
        <div className="route-copy">
          <h3 id="routeTitle">{routeHeadingCopy.title}</h3>
          <p id="routeSubtitle">{routeHeadingCopy.subtitle}</p>
        </div>
        <button
          className="route-action"
          id="routeActionButton"
          type="button"
          hidden
          onClick={(event) => {
            const route = event.currentTarget.dataset.route;
            if (route) {
              onRouteAction(route);
            }
          }}
        >
          + Add Channel
        </button>
      </header>
      <div
        id={LEGACY_SHELL_IDS.routeBody}
        className={routeBodyClassName}
        data-route={currentRoute}
        dangerouslySetInnerHTML={routeBodySnapshot.managed ? { __html: routeBodySnapshot.html } : undefined}
      ></div>
    </section>
  );
});

export const ChatWorkspace = memo(function ChatWorkspace({
  currentRoute,
  language,
  onCreateSession,
  onNavigate,
  onQuickPrompt,
  onToggleNavDrawer,
  onToggleSessionPane,
}: ChatWorkspaceProps) {
  const copy = getLegacyShellCopy(language);
  const workspaceSnapshot = useLegacyChatWorkspaceSnapshot(currentRoute, language);
  const messageRegionSnapshot = useLegacyMessageRegionSnapshot(currentRoute);
  const routeBodySnapshot = useLegacyRouteBodySnapshot(currentRoute);
  const isChatRoute = isLegacyShellChatRoute(currentRoute);
  const isPageMode = !isChatRoute;
  const isEmptyState = isChatRoute && !messageRegionSnapshot.hasMessages;
  const isTerminalRoute = currentRoute === "terminal";
  const mobileNewChatLabel = isTerminalRoute
    ? copy.terminalNewShort
    : currentRoute === "agent-runtime"
      ? copy.sessionNewAgent
      : copy.sessionNewChat;
  const sessionToggleLabel = isTerminalRoute ? copy.terminalSessions : copy.chatSessions;
  const chatPaneClassName = [
    "chat-pane",
    isPageMode ? "page-mode" : "",
    isEmptyState ? "empty-state" : "",
  ].filter(Boolean).join(" ");

  return (
    <main className={chatPaneClassName} data-route={currentRoute}>
      <header className="chat-header">
        <button
          className="nav-toggle"
          id={LEGACY_SHELL_IDS.navToggle}
          type="button"
          aria-label={copy.chatMenu}
          data-i18n="chat.menu"
          onClick={(event) => {
            event.stopPropagation();
            onToggleNavDrawer();
          }}
        >
          {copy.chatMenu}
        </button>
        <ChatHeaderTitleMount
          heading={workspaceSnapshot.heading}
          subheading={workspaceSnapshot.subheading}
        />
        <div className="chat-header-actions">
          <button
            className="panel-toggle"
            id={LEGACY_SHELL_IDS.sessionToggle}
            type="button"
            aria-label={sessionToggleLabel}
            data-i18n="chat.sessions"
            onClick={(event) => {
              event.stopPropagation();
              onToggleSessionPane();
            }}
          >
            {sessionToggleLabel}
          </button>
          <button
            className="mobile-new-chat"
            id="mobileNewChatButton"
            type="button"
            aria-label={mobileNewChatLabel}
            data-i18n="session.new"
            onClick={onCreateSession}
          >
            {mobileNewChatLabel}
          </button>
        </div>
      </header>

      <ChatView
        currentRoute={currentRoute}
        hidden={isPageMode}
        hasMessages={messageRegionSnapshot.hasMessages}
        messageRegionHTML={messageRegionSnapshot.html}
        welcomeHeading={workspaceSnapshot.welcomeHeading}
        welcomeDescription={workspaceSnapshot.welcomeDescription}
        welcomeTargetHTML={workspaceSnapshot.welcomeTargetHTML}
        onQuickPrompt={onQuickPrompt}
      />
      <RouteViewMount
        currentRoute={currentRoute}
        language={language}
        hidden={!isPageMode}
        onRouteAction={onNavigate}
        routeBodySnapshot={routeBodySnapshot}
      />
    </main>
  );
});
