import { ConversationRuntimeProvider } from "../../conversation-runtime/ConversationRuntimeProvider";
import { ConversationWorkspace } from "../../conversation-runtime/ConversationWorkspace";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { ReactManagedTerminalRouteBody } from "./ReactManagedTerminalRouteBody";

export function RuntimeRouteHost({
  route,
  language,
}: {
  route: "chat" | "agent-runtime" | "terminal";
  language: LegacyShellLanguage;
}) {
  if (route === "terminal") {
    return <ReactManagedTerminalRouteBody />;
  }

  return (
    <ConversationRuntimeProvider route={route} language={language}>
      <ConversationWorkspace language={language} />
    </ConversationRuntimeProvider>
  );
}
