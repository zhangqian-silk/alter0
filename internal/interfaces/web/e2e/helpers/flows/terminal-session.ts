import { expect, type APIRequestContext, type Page } from "@playwright/test";
import { authenticateWebRequest } from "./auth";
import { createTerminalPage } from "../pages/terminal";

const TERMINAL_CLIENT_STORAGE_KEY = "alter0.web.terminal.client.v1";
const TERMINAL_SESSIONS_STORAGE_KEY = "alter0.web.terminal.sessions.v2";
const trackedTerminalClientIDs = new Set<string>();

export type TerminalSessionRecord = {
  id: string;
  [key: string]: unknown;
};

export function createTerminalClientID(scope: string): string {
  const clientID = `playwright-terminal-${scope}-${Date.now()}`;
  trackedTerminalClientIDs.add(clientID);
  return clientID;
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

export async function clearTerminalSessions(request: APIRequestContext, clientID: string): Promise<void> {
  await authenticateWebRequest(request);
  const listResponse = await request.get("/api/terminal/sessions", {
    headers: {
      "X-Alter0-Terminal-Client": clientID,
    },
  });
  expect(listResponse.ok()).toBeTruthy();
  const payload = await listResponse.json();
  const items = Array.isArray(payload?.items) ? payload.items : [];
  for (const item of items) {
    const sessionID = typeof item?.id === "string" ? item.id : "";
    if (!sessionID) {
      continue;
    }
    const closeResponse = await request.delete(`/api/terminal/sessions/${encodeURIComponent(sessionID)}`, {
      headers: {
        "X-Alter0-Terminal-Client": clientID,
      },
    });
  }
}

export async function closeTrackedTerminalSessions(request: APIRequestContext): Promise<void> {
  const clientIDs = Array.from(trackedTerminalClientIDs);
  for (const clientID of clientIDs) {
    await clearTerminalSessions(request, clientID);
    trackedTerminalClientIDs.delete(clientID);
  }
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
