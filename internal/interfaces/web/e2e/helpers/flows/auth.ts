import { expect, type APIRequestContext } from "@playwright/test";

export function resolveWebLoginPassword(): string {
  const password = String(process.env.ALTER0_WEB_LOGIN_PASSWORD || "").trim();
  if (!password) {
    throw new Error("ALTER0_WEB_LOGIN_PASSWORD is required for protected web E2E routes");
  }
  return password;
}

export async function authenticateWebRequest(request: APIRequestContext): Promise<void> {
  const response = await request.post("/login", {
    form: {
      password: resolveWebLoginPassword(),
    },
  });
  expect(response.ok()).toBeTruthy();
}
