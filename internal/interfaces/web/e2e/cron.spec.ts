import { expect, test } from "@playwright/test";
import {
  expectComposerFocusedValue,
  expectComposerReady,
  expectComposerState,
  expectComposerValue,
} from "./helpers/asserts/composer";
import { commitIMEInput, startIMEInput } from "./helpers/interactions/ime";
import { openCronWorkspace } from "./helpers/scenarios/cron";

test.describe("Cron composer", () => {
  test("keeps IME composition text in cron prompt textarea", async ({ page }) => {
    const { cronPage } = await openCronWorkspace(page);
    const composer = cronPage.composer();
    const prompt = composer.input();
    await expectComposerReady(composer);
    await startIMEInput(prompt);
    await expectComposerState(composer, { composing: true, draft: "dirty" });

    await prompt.press("Enter");

    await expectComposerFocusedValue(composer, /ni/);
    await expect(cronPage.jobIDInput()).toHaveValue("");

    await commitIMEInput(prompt, "浣?");

    await expectComposerValue(composer, "浣?");
    await expectComposerState(composer, { composing: false, draft: "dirty" });
  });
});
