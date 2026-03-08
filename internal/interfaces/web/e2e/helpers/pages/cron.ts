import { type Locator, type Page } from "@playwright/test";
import { createComposerComponent, type ComposerComponent } from "../components/composer";

export function createCronPage(page: Page): {
  composer(): ComposerComponent;
  jobIDInput(): Locator;
} {
  const composer = createComposerComponent(page, "cron-prompt", {
    hasSubmitButton: false,
  });
  return {
    composer: () => composer,
    jobIDInput: () => page.locator('[data-cron-form] input[name="job_id"]'),
  };
}
