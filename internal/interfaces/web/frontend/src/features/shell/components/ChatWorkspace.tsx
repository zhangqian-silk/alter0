import { memo, useRef } from "react";
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
  type LegacyShellChatWorkspaceDetail,
  type LegacyShellMessageProcessStepDetail,
  type LegacyShellMessageSnapshotDetail,
  type LegacyShellChatWorkspaceTargetDetail,
  type LegacyShellMessageRegionDetail,
} from "../legacyShellBridge";
import { useLegacyShellSnapshot } from "../legacyShellSnapshot";
import { ChatRuntimeHost } from "./ChatRuntimeHost";
import { ScrollJumpStrip } from "./ScrollJumpStrip";
import { PROMPTS } from "../legacyShellConfig";
import {
  isReactManagedRouteBody,
  ReactManagedRouteBody,
} from "./ReactManagedRouteBody";
import {
  ChatMessageRegion,
  type ChatMessageProcessStepSnapshot,
  type ChatMessageSnapshot,
} from "./ChatMessageRegion";

type ChatWorkspaceProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  onCreateSession: () => void;
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
  welcomeTargets: WelcomeTargetSnapshot[];
  welcomeTargetError: string;
};

type MessageRegionSnapshot = {
  route: string;
  hasMessages: boolean;
  sessionId: string;
  messages: ChatMessageSnapshot[];
};

type WelcomeTargetSnapshot = {
  type: string;
  id: string;
  name: string;
  active: boolean;
  interactive: boolean;
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
      welcomeTargets: [],
      welcomeTargetError: "",
    };
  }

  return {
    route: currentRoute,
    heading: routeCopy.title,
    subheading: routeCopy.subtitle,
    welcomeHeading: "Hello, how can I help you today?",
    welcomeDescription: "I am a helpful assistant that can help you with your questions.",
    welcomeTargets: [],
    welcomeTargetError: "",
  };
}

function useLegacyChatWorkspaceSnapshot(
  currentRoute: string,
  language: LegacyShellLanguage,
): ChatWorkspaceSnapshot {
  return useLegacyShellSnapshot<LegacyShellChatWorkspaceDetail, ChatWorkspaceSnapshot>({
    currentRoute,
    eventName: LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT,
    fallback: () => getDefaultChatWorkspaceSnapshot(currentRoute, language),
    normalizeDetail: normalizeChatWorkspaceSnapshot,
  });
}

function useLegacyMessageRegionSnapshot(currentRoute: string): MessageRegionSnapshot {
  return useLegacyShellSnapshot<LegacyShellMessageRegionDetail, MessageRegionSnapshot>({
    currentRoute,
    eventName: LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT,
    fallback: () => ({
      route: currentRoute,
      hasMessages: false,
      sessionId: "",
      messages: [],
    }),
    normalizeDetail: normalizeMessageRegionSnapshot,
  });
}

function normalizeChatWorkspaceSnapshot(
  detail: LegacyShellChatWorkspaceDetail,
): ChatWorkspaceSnapshot | null {
  if (!detail || typeof detail.route !== "string") {
    return null;
  }

  return {
    route: detail.route,
    heading: detail.heading,
    subheading: detail.subheading,
    welcomeHeading: detail.welcomeHeading,
    welcomeDescription: detail.welcomeDescription,
    welcomeTargets: Array.isArray(detail.welcomeTargets)
      ? detail.welcomeTargets.map(normalizeChatWorkspaceTarget).filter((item) => item !== null)
      : [],
    welcomeTargetError:
      typeof detail.welcomeTargetError === "string" ? detail.welcomeTargetError : "",
  };
}

function normalizeChatWorkspaceTarget(
  detail: LegacyShellChatWorkspaceTargetDetail,
): WelcomeTargetSnapshot | null {
  if (!detail) {
    return null;
  }

  const type = typeof detail.type === "string" ? detail.type : "model";
  const id = typeof detail.id === "string" ? detail.id : "";
  const name = typeof detail.name === "string" ? detail.name : "";
  if (!id && !name) {
    return null;
  }

  return {
    type,
    id,
    name,
    active: Boolean(detail.active),
    interactive: detail.interactive !== false,
  };
}

function normalizeMessageRegionSnapshot(
  detail: LegacyShellMessageRegionDetail,
): MessageRegionSnapshot | null {
  if (!detail || typeof detail.route !== "string") {
    return null;
  }

  return {
    route: detail.route,
    hasMessages: Boolean(detail.hasMessages),
    sessionId: typeof detail.sessionId === "string" ? detail.sessionId : "",
    messages: Array.isArray(detail.messages)
      ? detail.messages.map(normalizeMessageSnapshot).filter((item) => item !== null)
      : [],
  };
}

function normalizeMessageSnapshot(
  detail: LegacyShellMessageSnapshotDetail,
): ChatMessageSnapshot | null {
  if (!detail || typeof detail.id !== "string" || !detail.id.trim()) {
    return null;
  }

  const role = detail.role === "assistant" ? "assistant" : "user";
  const text = typeof detail.text === "string" ? detail.text : "";
  const processSteps = Array.isArray(detail.process_steps)
    ? detail.process_steps.map(normalizeProcessStepSnapshot).filter((item) => item !== null)
    : [];
  if (!text.trim() && !processSteps.length) {
    return null;
  }

  return {
    id: detail.id,
    role,
    text,
    route: typeof detail.route === "string" ? detail.route : "",
    source: typeof detail.source === "string" ? detail.source : "",
    error: Boolean(detail.error),
    status: typeof detail.status === "string" && detail.status ? detail.status : "done",
    at: Number.isFinite(detail.at) ? Number(detail.at) : Date.now(),
    processSteps,
    agentProcessCollapsed:
      typeof detail.agent_process_collapsed === "boolean"
        ? detail.agent_process_collapsed
        : undefined,
  };
}

function normalizeProcessStepSnapshot(
  detail: LegacyShellMessageProcessStepDetail,
): ChatMessageProcessStepSnapshot | null {
  if (!detail) {
    return null;
  }
  const title = typeof detail.title === "string" ? detail.title.trim() : "";
  const detailText = typeof detail.detail === "string" ? detail.detail.trim() : "";
  if (!title && !detailText) {
    return null;
  }
  return {
    id: typeof detail.id === "string" ? detail.id : "",
    kind: typeof detail.kind === "string" ? detail.kind : "",
    title,
    detail: detailText,
    status: typeof detail.status === "string" ? detail.status : "",
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
  messageRegion,
  language,
  welcomeHeading,
  welcomeDescription,
  welcomeTargets,
  welcomeTargetError,
  onQuickPrompt,
}: {
  currentRoute: string;
  hidden: boolean;
  hasMessages: boolean;
  messageRegion: MessageRegionSnapshot;
  language: LegacyShellLanguage;
  welcomeHeading: string;
  welcomeDescription: string;
  welcomeTargets: WelcomeTargetSnapshot[];
  welcomeTargetError: string;
  onQuickPrompt: (prompt: string) => void;
}) {
  const copy = getLegacyShellCopy(language);
  const messageAreaRef = useRef<HTMLElement | null>(null);
  const showMessages = hasMessages || messageRegion.messages.length > 0;

  return (
    <div className="chat-view" id="chatView" hidden={hidden}>
      <section className="welcome-screen" id={LEGACY_SHELL_IDS.welcomeScreen} hidden={showMessages}>
        <p className="welcome-tag" data-i18n="welcome.tag">alter0 assistant</p>
        <h3 id="welcomeHeading">{welcomeHeading}</h3>
        <p id="welcomeDescription">{welcomeDescription}</p>
        <div className="welcome-target-list" id="welcomeTargetList">
          {welcomeTargets.map((item) =>
            item.interactive ? (
              <button
                key={`${item.type}:${item.id}:${item.name}`}
                className={item.active ? "welcome-target-card active" : "welcome-target-card"}
                type="button"
                data-chat-target-type={item.type}
                data-chat-target-id={item.id}
                data-chat-target-name={item.name}
              >
                <strong>{item.name}</strong>
                <span>{item.id}</span>
              </button>
            ) : (
              <div
                key={`${item.type}:${item.id}:${item.name}`}
                className={item.active ? "welcome-target-card active is-static" : "welcome-target-card is-static"}
              >
                <strong>{item.name}</strong>
                <span>{item.id}</span>
              </div>
            ),
          )}
          {welcomeTargetError ? <p className="welcome-target-error">{welcomeTargetError}</p> : null}
        </div>
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
        hidden={!showMessages}
        ref={messageAreaRef}
      >
        {messageRegion.messages.length ? (
          <ChatMessageRegion
            sessionId={messageRegion.sessionId}
            messages={messageRegion.messages}
            language={language}
          />
        ) : null}
        {currentRoute === "agent-runtime" ? (
          <ScrollJumpStrip
            scope="agent"
            language={language}
            containerRef={messageAreaRef}
            itemSelector="[data-message-id]"
            itemAttribute="data-message-id"
          />
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
            <ChatRuntimeHost currentRoute={currentRoute} language={language} />
            <div className="composer-submit-bar">
              <span id={LEGACY_SHELL_IDS.charCount} data-composer-counter="chat-main">0/10000</span>
              <button id="sendButton" type="submit" aria-label="Send message" data-i18n="composer.send" data-composer-submit="chat-main">
                Send
              </button>
            </div>
          </div>
        </form>
        <p className="composer-note" data-i18n="composer.note">{copy.composerNote}</p>
      </footer>
    </div>
  );
});

type RouteViewMountProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  hidden: boolean;
};

function LegacyManagedRouteHost({ route }: { route: string }) {
  return null;
}

const RouteViewMount = memo(function RouteViewMount({
  currentRoute,
  language,
  hidden,
}: RouteViewMountProps) {
  const routeViewRef = useRef<HTMLElement | null>(null);
  const routeViewClassName = currentRoute === "terminal" ? "route-view terminal-route" : "route-view";
  const routeBodyClassName = currentRoute === "terminal" ? "route-body terminal-route-body" : "route-body";
  const routeHeadingCopy = getLegacyRouteHeadingCopy(language, currentRoute);
  const reactManagedRoute = isReactManagedRouteBody(currentRoute) ? currentRoute : null;

  return (
    <section
      className={routeViewClassName}
      id={LEGACY_SHELL_IDS.routeView}
      data-route={currentRoute}
      hidden={hidden}
      ref={routeViewRef}
    >
      <header className="route-head">
        <div className="route-copy">
          <h3 id="routeTitle">{routeHeadingCopy.title}</h3>
          <p id="routeSubtitle">{routeHeadingCopy.subtitle}</p>
        </div>
      </header>
      <div
        id={LEGACY_SHELL_IDS.routeBody}
        className={routeBodyClassName}
        data-route={currentRoute}
        data-react-managed-route={reactManagedRoute ? "true" : "false"}
      >
        {reactManagedRoute ? (
          <ReactManagedRouteBody route={reactManagedRoute} language={language} />
        ) : (
          <LegacyManagedRouteHost route={currentRoute} />
        )}
      </div>
    </section>
  );
});

export const ChatWorkspace = memo(function ChatWorkspace({
  currentRoute,
  language,
  onCreateSession,
  onQuickPrompt,
  onToggleNavDrawer,
  onToggleSessionPane,
}: ChatWorkspaceProps) {
  const copy = getLegacyShellCopy(language);
  const workspaceSnapshot = useLegacyChatWorkspaceSnapshot(currentRoute, language);
  const messageRegionSnapshot = useLegacyMessageRegionSnapshot(currentRoute);
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
        {!isPageMode ? (
          <ChatHeaderTitleMount
            heading={workspaceSnapshot.heading}
            subheading={workspaceSnapshot.subheading}
          />
        ) : null}
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
        messageRegion={messageRegionSnapshot}
        language={language}
        welcomeHeading={workspaceSnapshot.welcomeHeading}
        welcomeDescription={workspaceSnapshot.welcomeDescription}
        welcomeTargets={workspaceSnapshot.welcomeTargets}
        welcomeTargetError={workspaceSnapshot.welcomeTargetError}
        onQuickPrompt={onQuickPrompt}
      />
      <RouteViewMount
        currentRoute={currentRoute}
        language={language}
        hidden={!isPageMode}
      />
    </main>
  );
});
