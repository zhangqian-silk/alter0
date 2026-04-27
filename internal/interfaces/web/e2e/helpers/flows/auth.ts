import { expect, type APIRequestContext } from "@playwright/test";
import { resolveE2ELoginPassword } from "../support/e2e-login-password";

export function resolveWebLoginPassword(): string {
  const password = resolveE2ELoginPassword();
  if (!password) {
    throw new Error("ALTER0_WEB_LOGIN_PASSWORD is required for protected web E2E routes");
  }
  return password;
}

export async function authenticateWebRequest(request: APIRequestContext): Promise<void> {
  const response = await request.post("/login", {
    maxRedirects: 0,
    form: {
      password: resolveWebLoginPassword(),
    },
  });
  expect([200, 302, 303, 307, 308]).toContain(response.status());
}
