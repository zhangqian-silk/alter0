import { type Locator, type Page } from "@playwright/test";

export function createLoginPage(page: Page): {
  passwordInput(): Locator;
  submitButton(): Locator;
} {
  return {
    passwordInput: () => page.locator('input[name="password"]'),
    submitButton: () => page.locator('button[type="submit"]'),
  };
}
