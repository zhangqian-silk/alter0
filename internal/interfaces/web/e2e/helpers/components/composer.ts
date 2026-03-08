import { type Locator, type Page } from "@playwright/test";

export type ComposerComponent = {
  name: string;
  form(): Locator;
  input(): Locator;
  submitButton(): Locator;
  counter(): Locator;
};

export function createComposerComponent(
  page: Page,
  name: string,
  options: {
    hasSubmitButton?: boolean;
    hasCounter?: boolean;
  } = {}
): ComposerComponent {
  const hasSubmitButton = options.hasSubmitButton !== false;
  const hasCounter = Boolean(options.hasCounter);

  const missingControl = (control: string): Locator => {
    throw new Error(`composer "${name}" is missing ${control}`);
  };

  return {
    name,
    form: () => page.locator(`[data-composer-form="${name}"]`),
    input: () => page.locator(`[data-composer-input="${name}"]`),
    submitButton: () => (hasSubmitButton ? page.locator(`[data-composer-submit="${name}"]`) : missingControl("submit button")),
    counter: () => (hasCounter ? page.locator(`[data-composer-counter="${name}"]`) : missingControl("counter")),
  };
}
