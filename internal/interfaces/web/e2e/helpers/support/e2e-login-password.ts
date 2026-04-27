import { createHash, randomBytes } from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

function resolvePasswordCachePath(): string {
  const rootDir = path.resolve(__dirname, "../../..");
  const scope = createHash("sha256").update(rootDir).digest("hex").slice(0, 16);
  return path.join(os.tmpdir(), `alter0-playwright-login-password-${scope}`);
}

function readCachedPassword(filePath: string): string {
  try {
    return String(fs.readFileSync(filePath, "utf8") || "").trim();
  } catch {
    return "";
  }
}

function writeCachedPassword(filePath: string, password: string) {
  try {
    fs.writeFileSync(filePath, `${password}\n`, { mode: 0o600 });
  } catch {
  }
}

export function resolveE2ELoginPassword(): string {
  const direct = String(process.env.ALTER0_WEB_LOGIN_PASSWORD || "").trim();
  if (direct) {
    return direct;
  }
  const cachePath = String(process.env.ALTER0_PLAYWRIGHT_PASSWORD_FILE || "").trim() || resolvePasswordCachePath();
  const cached = readCachedPassword(cachePath);
  if (cached) {
    return cached;
  }
  const generated = `alter0-e2e-${randomBytes(16).toString("hex")}`;
  writeCachedPassword(cachePath, generated);
  return generated;
}
