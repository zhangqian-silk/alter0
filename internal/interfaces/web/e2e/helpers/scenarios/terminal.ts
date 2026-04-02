import { expectComposerReady } from "../asserts/composer";
import { openTerminalRoute } from "../flows/routes";
import {
  bindTerminalClient,
  clearTerminalSessions,
  createTerminalClientID,
  createTerminalSession,
  seedTerminalSessions,
  type TerminalSessionRecord,
} from "../flows/terminal-session";
import { waitForTerminalPollAndRepaint } from "../flows/terminal-runtime";
import { createTerminalPage } from "../pages/terminal";
import { type APIRequestContext, type Page } from "@playwright/test";
import { interruptedTerminalPrompt, terminalSessionWorkspace } from "../support/terminal-env";

export async function openTerminalWorkspace(
  page: Page,
  options: {
    scope: string;
  }
): Promise<{
  clientID: string;
  terminalPage: ReturnType<typeof createTerminalPage>;
}> {
  const clientID = createTerminalClientID(options.scope);
  await bindTerminalClient(page, clientID);
  await openTerminalRoute(page);
  const terminalPage = createTerminalPage(page);
  return { clientID, terminalPage };
}

export async function openTerminalWorkspaceWithSessions(
  page: Page,
  request: APIRequestContext,
  options: {
    scope: string;
    count?: number;
  }
): Promise<{
  clientID: string;
  sessions: TerminalSessionRecord[];
  terminalPage: ReturnType<typeof createTerminalPage>;
}> {
  const clientID = createTerminalClientID(options.scope);
  await clearTerminalSessions(request, clientID);
  await bindTerminalClient(page, clientID);
  const sessionCount = options.count ?? 1;
  const sessions: TerminalSessionRecord[] = [];
  for (let index = 0; index < sessionCount; index += 1) {
    sessions.push(await createTerminalSession(request, clientID));
  }
  await openTerminalRoute(page);
  const terminalPage = createTerminalPage(page);
  return { clientID, sessions, terminalPage };
}

export async function openReadyTerminalWorkspace(
  page: Page,
  request: APIRequestContext,
  options: {
    scope: string;
  }
): Promise<{
  clientID: string;
  session: TerminalSessionRecord;
  terminalPage: ReturnType<typeof createTerminalPage>;
}> {
  const { clientID, sessions, terminalPage } = await openTerminalWorkspaceWithSessions(page, request, options);
  const [session] = sessions;
  await expectComposerReady(terminalPage.composer());
  await waitForTerminalPollAndRepaint(page, session.id);
  return { clientID, session, terminalPage };
}

export async function openInterruptedTerminalWorkspace(
  page: Page,
  options: {
    scope: string;
    now?: number;
  }
): Promise<{
  clientID: string;
  terminalPage: ReturnType<typeof createTerminalPage>;
  session: TerminalSessionRecord;
}> {
  const clientID = createTerminalClientID(options.scope);
  const now = options.now ?? Date.now();
  const session: TerminalSessionRecord = {
    id: "terminal-stale-session",
    title: "terminal-stale-session",
    terminal_session_id: "terminal-stale-session",
    status: "ready",
    shell: "codex exec",
    working_dir: terminalSessionWorkspace("terminal-stale-session"),
    created_at: now - 1000,
    updated_at: now,
    entry_cursor: 1,
    disconnected_notice: false,
    entries: [
      {
        id: "entry-1",
        role: "output",
        text: interruptedTerminalPrompt,
        at: now - 500,
        kind: "stdout",
        stream: "stdout",
        cursor: 1,
      },
    ],
  };
  await bindTerminalClient(page, clientID);
  await seedTerminalSessions(page, [session]);
  await openTerminalRoute(page);
  const terminalPage = createTerminalPage(page);
  return { clientID, terminalPage, session };
}
