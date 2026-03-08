import { expect, type Page } from "@playwright/test";
import { createLoginPage } from "../pages/login";
import { resolveWebLoginPassword } from "../flows/auth";

export async function loginIfNeeded(page: Page): Promise<void> {
  const loginPage = createLoginPage(page);
  const passwordInput = loginPage.passwordInput();
  const needsLogin = await passwordInput.isVisible().catch(() => false);
  if (!needsLogin) {
    return;
  }
  await passwordInput.fill(resolveWebLoginPassword());
  await loginPage.submitButton().click();
  await expect(passwordInput).toBeHidden({ timeout: 10000 });
}
