import { expect, test, type Page } from "@playwright/test";
import { expectComposerReady } from "./helpers/asserts/composer";
import { openTerminalRoute } from "./helpers/flows/routes";
import { createTerminalPage } from "./helpers/pages/terminal";
import { installVisualViewportMock, setVisualViewport } from "./helpers/support/visual-viewport";

async function readTerminalViewportGap(page: Page) {
  return page.evaluate(() => {
    const screen = document.querySelector("[data-runtime-screen='terminal']");
    const composer = document.querySelector(".runtime-composer-shell");
    if (!(screen instanceof HTMLElement) || !(composer instanceof HTMLElement)) {
      return null;
    }
    const screenRect = screen.getBoundingClientRect();
    const composerRect = composer.getBoundingClientRect();
    return {
      screenBottom: screenRect.bottom,
      composerTop: composerRect.top,
      gap: composerRect.top - screenRect.bottom,
    };
  });
}

test.describe("Terminal compatibility route", () => {
  test("mounts the shared runtime with Terminal owner selectors", async ({ page }) => {
    await openTerminalRoute(page);
    const terminalPage = createTerminalPage(page);

    await expect(page.locator("[data-runtime-view='terminal']")).toHaveAttribute("data-runtime-route", "terminal");
    await expect(terminalPage.workspace()).toHaveAttribute("data-runtime-route", "terminal");
    await expect(terminalPage.chatScreen()).toBeVisible();
    await expect(terminalPage.composer().input()).toHaveAttribute("data-composer-input", "terminal");
    await expect(terminalPage.createButton()).toBeVisible();
  });

  test("creates and submits a Terminal-owned conversation session", async ({ page }) => {
    await openTerminalRoute(page);
    const terminalPage = createTerminalPage(page);

    await terminalPage.createButton().click();
    await expectComposerReady(terminalPage.composer());

    await terminalPage.composer().input().fill("terminal compatibility prompt");
    await terminalPage.composer().submitButton().click();

    await expect(page.locator(".msg.user .msg-bubble").last()).toContainText("terminal compatibility prompt");
    await expect(terminalPage.composer().input()).toHaveValue("");
    await expectComposerReady(terminalPage.composer(), 60000);
  });

  test("keeps the Terminal viewport above the composer on desktop and mobile", async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 960 });
    await openTerminalRoute(page);

    let metrics = await readTerminalViewportGap(page);
    expect(metrics).not.toBeNull();
    expect(metrics?.gap ?? -1).toBeGreaterThanOrEqual(0);

    await installVisualViewportMock(page);
    await page.setViewportSize({ width: 430, height: 932 });
    await openTerminalRoute(page);
    const terminalPage = createTerminalPage(page);
    await terminalPage.composer().input().click();
    await setVisualViewport(page, { width: 430, height: 620, offsetTop: 0 });
    await expect.poll(async () => page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset").trim()
    )).toBe("312px");

    metrics = await readTerminalViewportGap(page);
    expect(metrics).not.toBeNull();
    expect(metrics?.gap ?? -1).toBeGreaterThanOrEqual(0);
  });
});
