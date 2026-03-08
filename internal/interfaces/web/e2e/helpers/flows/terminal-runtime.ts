import { type Page } from "@playwright/test";
import { expectComposerReady } from "../asserts/composer";
import { createTerminalPage } from "../pages/terminal";

export async function waitForTerminalRepaint(page: Page, timeout = 5000): Promise<void> {
  const terminalPage = createTerminalPage(page);
  const composer = terminalPage.composer();
  const previousInput = await composer.input().elementHandle();
  if (!previousInput) {
    await expectComposerReady(composer, timeout);
    return;
  }
  try {
    await page.waitForFunction((node) => {
      const current = document.querySelector('[data-composer-input="terminal-runtime"]');
      return !node || !node.isConnected || current !== node;
    }, previousInput, { timeout });
  } finally {
    await previousInput.dispose();
  }
  await expectComposerReady(composer, timeout);
}

export async function waitForTerminalPoll(page: Page, sessionID: string, timeout = 5000): Promise<void> {
  const encodedSessionID = encodeURIComponent(sessionID);
  const matchesSessionState = (url: string) => {
    if (!url.includes(`/api/terminal/sessions/${encodedSessionID}`)) {
      return false;
    }
    return !url.includes("/entries?") && !url.includes("/steps/");
  };
  await page.waitForResponse((response) => response.request().method() === "GET" && response.ok() && matchesSessionState(response.url()), { timeout });
}

export async function waitForTerminalPollAndRepaint(page: Page, sessionID: string, timeout = 5000): Promise<void> {
  await Promise.all([
    waitForTerminalPoll(page, sessionID, timeout),
    waitForTerminalRepaint(page, timeout)
  ]);
}
