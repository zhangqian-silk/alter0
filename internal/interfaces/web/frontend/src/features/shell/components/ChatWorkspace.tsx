import { memo } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import {
  getLegacyRouteHeadingCopy,
  getLegacyShellCopy,
  type LegacyShellLanguage,
} from "../legacyShellCopy";
import { isLegacyShellChatRoute } from "../legacyShellState";
import { ChatRuntimeHost } from "./ChatRuntimeHost";
import { PROMPTS } from "../legacyShellConfig";

type ChatWorkspaceProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
};

const ChatHeaderTitleMount = memo(function ChatHeaderTitleMount() {
  return (
    <div className="chat-header-copy">
      <h2 id="sessionHeading" data-i18n="chat.title">Chat</h2>
      <p id="sessionSubheading" data-i18n="chat.subtitle">Ready to start a new conversation</p>
    </div>
  );
});

const ChatViewMount = memo(function ChatViewMount({ hidden }: { hidden: boolean }) {
  return (
    <div className="chat-view" id="chatView" hidden={hidden}>
      <section className="welcome-screen" id={LEGACY_SHELL_IDS.welcomeScreen}>
        <p className="welcome-tag" data-i18n="welcome.tag">alter0 assistant</p>
        <h3 id="welcomeHeading" data-i18n="welcome.heading">Hello, how can I help you today?</h3>
        <p id="welcomeDescription" data-i18n="welcome.desc">I am a helpful assistant that can help you with your questions.</p>
        <div className="prompt-grid">
          {PROMPTS.map((item) => (
            <button key={item.i18n} className="prompt" type="button" data-prompt={item.prompt}>
              <span data-i18n={item.i18n}>{item.label}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="message-area" id={LEGACY_SHELL_IDS.messageArea} aria-live="polite"></section>

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
            <ChatRuntimeHost />
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
};

const RouteViewMount = memo(function RouteViewMount({
  currentRoute,
  language,
  hidden,
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
        <button className="route-action" id="routeActionButton" type="button" hidden>
          + Add Channel
        </button>
      </header>
      <div id={LEGACY_SHELL_IDS.routeBody} className={routeBodyClassName} data-route={currentRoute}></div>
    </section>
  );
});

export const ChatWorkspace = memo(function ChatWorkspace({
  currentRoute,
  language,
}: ChatWorkspaceProps) {
  const copy = getLegacyShellCopy(language);
  const isChatRoute = isLegacyShellChatRoute(currentRoute);
  const isPageMode = !isChatRoute;
  const isTerminalRoute = currentRoute === "terminal";
  const mobileNewChatLabel = isTerminalRoute
    ? copy.terminalNewShort
    : currentRoute === "agent-runtime"
      ? copy.sessionNewAgent
      : copy.sessionNewChat;
  const sessionToggleLabel = isTerminalRoute ? copy.terminalSessions : copy.chatSessions;
  const chatPaneClassName = isPageMode ? "chat-pane page-mode" : "chat-pane";

  return (
    <main className={chatPaneClassName} data-route={currentRoute}>
      <header className="chat-header">
        <button className="nav-toggle" id={LEGACY_SHELL_IDS.navToggle} type="button" aria-label={copy.chatMenu} data-i18n="chat.menu">
          {copy.chatMenu}
        </button>
        <ChatHeaderTitleMount />
        <div className="chat-header-actions">
          <button className="panel-toggle" id={LEGACY_SHELL_IDS.sessionToggle} type="button" aria-label={sessionToggleLabel} data-i18n="chat.sessions">
            {sessionToggleLabel}
          </button>
          <button className="mobile-new-chat" id="mobileNewChatButton" type="button" aria-label={mobileNewChatLabel} data-i18n="session.new">
            {mobileNewChatLabel}
          </button>
        </div>
      </header>

      <ChatViewMount hidden={isPageMode} />
      <RouteViewMount currentRoute={currentRoute} language={language} hidden={!isPageMode} />
    </main>
  );
});
