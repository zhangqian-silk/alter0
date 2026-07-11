const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");

function generateRandomE2EPassword(randomBytes = crypto.randomBytes) {
  return `alter0-e2e-${randomBytes(16).toString("hex")}`;
}

function resolvePasswordCachePath(rootDir = __dirname) {
  const scope = crypto.createHash("sha256").update(path.resolve(rootDir)).digest("hex").slice(0, 16);
  return path.join(os.tmpdir(), `alter0-playwright-login-password-${scope}`);
}

function resolveScopedTempPath(prefix, rootDir = __dirname) {
  const scope = crypto.createHash("sha256").update(path.resolve(rootDir)).digest("hex").slice(0, 16);
  return path.join(os.tmpdir(), `${prefix}-${scope}`);
}

function resolvePlaywrightRuntimeRoot(env = process.env, randomBytes = crypto.randomBytes) {
  const explicitTestRoot = String(env.ALTER0_PLAYWRIGHT_RUNTIME_ROOT || "").trim();
  if (explicitTestRoot) {
    const resolvedRoot = path.resolve(explicitTestRoot);
    const systemTempRoot = path.resolve(os.tmpdir());
    if (
      path.dirname(resolvedRoot) !== systemTempRoot
      || !path.basename(resolvedRoot).startsWith("alter0-playwright-")
    ) {
      throw new Error("ALTER0_PLAYWRIGHT_RUNTIME_ROOT must be a direct alter0-playwright-* child of the system temp directory");
    }
    return resolvedRoot;
  }
  const isolatedSuffix = randomBytes(8).toString("hex");
  return `${resolveScopedTempPath("alter0-playwright-runtime")}-${isolatedSuffix}`;
}

function readCachedPassword(filePath) {
  try {
    return String(fs.readFileSync(filePath, "utf8") || "").trim();
  } catch {
    return "";
  }
}

function writeCachedPassword(filePath, password) {
  try {
    fs.writeFileSync(filePath, `${password}\n`, { mode: 0o600 });
  } catch {
  }
}

function resolveE2ELoginPassword(env = process.env, randomBytes = crypto.randomBytes) {
  const existing = String(env.ALTER0_WEB_LOGIN_PASSWORD || "").trim();
  if (existing) {
    return existing;
  }
  const cachePath = String(env.ALTER0_PLAYWRIGHT_PASSWORD_FILE || "").trim() || resolvePasswordCachePath();
  const cached = readCachedPassword(cachePath);
  if (cached) {
    return cached;
  }
  const generated = generateRandomE2EPassword(randomBytes);
  writeCachedPassword(cachePath, generated);
  return generated;
}

function buildPlaywrightEnv(env = process.env, randomBytes = crypto.randomBytes) {
  const cachePath = String(env.ALTER0_PLAYWRIGHT_PASSWORD_FILE || "").trim() || resolvePasswordCachePath();
  const loginPassword = resolveE2ELoginPassword(env, randomBytes);
  const goCacheDir = String(env.GOCACHE || "").trim() || resolveScopedTempPath("alter0-playwright-go-build");
  const xdgCacheHome = String(env.XDG_CACHE_HOME || "").trim() || resolveScopedTempPath("alter0-playwright-cache");
  const browserCacheDir = String(env.ALTER0_PLAYWRIGHT_BROWSERS_PATH || "").trim() || resolveScopedTempPath("alter0-playwright-browsers");
  const runtimeRoot = resolvePlaywrightRuntimeRoot(env, randomBytes);
  fs.rmSync(runtimeRoot, { recursive: true, force: true });
  fs.mkdirSync(goCacheDir, { recursive: true });
  fs.mkdirSync(xdgCacheHome, { recursive: true });
  fs.mkdirSync(browserCacheDir, { recursive: true });
  fs.mkdirSync(runtimeRoot, { recursive: true });
  const resolvedEnv = {
    ...env,
    ALTER0_RUNTIME_ROOT: runtimeRoot,
    ALTER0_STORAGE_DIR: "",
    ALTER0_CODEX_WORKSPACE_ROOT: "",
    ALTER0_PLAYWRIGHT_RUNTIME_ROOT: "",
    ALTER0_WEB_LOGIN_PASSWORD: loginPassword,
    ALTER0_PLAYWRIGHT_BROWSERS_PATH: browserCacheDir,
    ALTER0_PLAYWRIGHT_PASSWORD_FILE: cachePath,
    GOCACHE: goCacheDir,
    PLAYWRIGHT_BROWSERS_PATH: browserCacheDir,
    XDG_CACHE_HOME: xdgCacheHome,
  };
  return resolvedEnv;
}

module.exports = {
  buildPlaywrightEnv,
  generateRandomE2EPassword,
  resolvePasswordCachePath,
  resolvePlaywrightRuntimeRoot,
  resolveScopedTempPath,
  resolveE2ELoginPassword,
};
