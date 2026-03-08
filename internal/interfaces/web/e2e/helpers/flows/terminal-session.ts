import { expect, type APIRequestContext, type Page } from "@playwright/test";
import { authenticateWebRequest } from "./auth";
import { createTerminalPage } from "../pages/terminal";

const TERMINAL_CLIENT_STORAGE_KEY = "alter0.web.terminal.client.v1";
const TERMINAL_SESSIONS_STORAGE_KEY = "alter0.web.terminal.sessions.v2";

export type TerminalSessionRecord = {
  id: string;
  [key: string]: unknown;
};

export function createTerminalClientID(scope: string): string {
  return `playwright-terminal-${scope}-${Date.now()}`;
}

export async function bindTerminalClient(page: Page, clientID: string): Promise<void> {
  await page.addInitScript(([storageKey, value]) => {
    window.sessionStorage.setItem(storageKey, value);
  }, [TERMINAL_CLIENT_STORAGE_KEY, clientID]);
}

export async function seedTerminalSessions(page: Page, sessions: unknown[]): Promise<void> {
  await page.addInitScript(([storageKey, value]) => {
    window.localStorage.setItem(storageKey, value);
  }, [TERMINAL_SESSIONS_STORAGE_KEY, JSON.stringify(sessions)]);
}

export async function createTerminalSession(request: APIRequestContext, clientID: string): Promise<TerminalSessionRecord> {
  await authenticateWebRequest(request);
  const response = await request.post("/api/terminal/sessions", {
    headers: {
      "X-Alter0-Terminal-Client": clientID,
    },
    data: {},
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  return payload?.session;
}

export async function selectTerminalSession(page: Page, sessionID: string): Promise<void> {
  const terminalPage = createTerminalPage(page);
  const sessionCard = terminalPage.sessionList().itemByValue?.(sessionID);
  if (!sessionCard) {
    throw new Error("terminal session list does not support selecting by id");
  }
  await sessionCard.click();
  await expect(terminalPage.workspace()).toContainText(sessionID);
}
