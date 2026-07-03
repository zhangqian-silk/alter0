import { ConversationRuntimeProvider } from "../../conversation-runtime/ConversationRuntimeProvider";
import { ConversationWorkspace } from "../../conversation-runtime/ConversationWorkspace";
import type { LegacyShellLanguage } from "../legacyShellCopy";

export function RuntimeRouteHost({
  route,
  language,
}: {
  route: "chat";
  language: LegacyShellLanguage;
}) {
  return (
    <ConversationRuntimeProvider key={route} route={route} language={language}>
      <ConversationWorkspace language={language} />
    </ConversationRuntimeProvider>
  );
}
