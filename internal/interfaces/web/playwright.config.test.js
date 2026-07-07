const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const {
  buildPlaywrightEnv,
  generateRandomE2EPassword,
  resolveE2ELoginPassword,
  resolveScopedTempPath,
} = require("./playwright.config.shared");

function tempPasswordFile(name) {
  return path.join(os.tmpdir(), `alter0-playwright-config-test-${name}.txt`);
}

test("resolveE2ELoginPassword reuses configured password", () => {
  const configuredValue = "configured-login-value";
  const password = resolveE2ELoginPassword({
    ALTER0_WEB_LOGIN_PASSWORD: configuredValue,
  });
  assert.equal(password, configuredValue);
});

test("resolveE2ELoginPassword generates random password when env is missing", () => {
  const filePath = tempPasswordFile("generated");
  const seed = Buffer.from("0123456789abcdef", "utf8");
  fs.rmSync(filePath, { force: true });
  const password = resolveE2ELoginPassword({
    ALTER0_PLAYWRIGHT_PASSWORD_FILE: filePath,
  }, () => seed);
  assert.equal(password, `alter0-e2e-${seed.toString("hex")}`);
  fs.rmSync(filePath, { force: true });
});

test("generateRandomE2EPassword uses the expected prefix", () => {
  const seed = Buffer.from("beef", "utf8");
  const password = generateRandomE2EPassword(() => seed);
  assert.equal(password, `alter0-e2e-${seed.toString("hex")}`);
});

test("buildPlaywrightEnv injects only the resolved login password", () => {
  const filePath = tempPasswordFile("env");
  const seed = Buffer.from("seed-seed-seed!!", "utf8");
  fs.rmSync(filePath, { force: true });
  const env = buildPlaywrightEnv(
    {
      PATH: "/usr/bin",
      ALTER0_PLAYWRIGHT_PASSWORD_FILE: filePath,
    },
    () => seed,
  );
  assert.equal(env.PATH, "/usr/bin");
  assert.equal(env.ALTER0_RUNTIME_ROOT, resolveScopedTempPath("alter0-playwright-runtime"));
  assert.equal(env.ALTER0_STORAGE_DIR, undefined);
  assert.equal(env.ALTER0_WEB_LOGIN_PASSWORD, `alter0-e2e-${seed.toString("hex")}`);
  assert.equal(env.ALTER0_PLAYWRIGHT_BROWSERS_PATH, resolveScopedTempPath("alter0-playwright-browsers"));
  assert.equal(env.ALTER0_PLAYWRIGHT_PASSWORD_FILE, filePath);
  assert.equal(env.GOCACHE, resolveScopedTempPath("alter0-playwright-go-build"));
  assert.equal(env.PLAYWRIGHT_BROWSERS_PATH, resolveScopedTempPath("alter0-playwright-browsers"));
  assert.equal(env.XDG_CACHE_HOME, resolveScopedTempPath("alter0-playwright-cache"));
  assert.equal(fs.statSync(env.GOCACHE).isDirectory(), true);
  assert.equal(fs.statSync(env.ALTER0_RUNTIME_ROOT).isDirectory(), true);
  assert.equal(fs.statSync(env.PLAYWRIGHT_BROWSERS_PATH).isDirectory(), true);
  assert.equal(fs.statSync(env.XDG_CACHE_HOME).isDirectory(), true);
  fs.rmSync(filePath, { force: true });
});

test("buildPlaywrightEnv clears the default runtime root between e2e runs", () => {
  const runtimeRoot = resolveScopedTempPath("alter0-playwright-runtime");
  const sentinel = path.join(runtimeRoot, "storage", "stale-session.json");
  fs.mkdirSync(path.dirname(sentinel), { recursive: true });
  fs.writeFileSync(sentinel, "stale");

  const env = buildPlaywrightEnv({});

  assert.equal(env.ALTER0_RUNTIME_ROOT, runtimeRoot);
  assert.equal(fs.existsSync(sentinel), false);
  assert.equal(fs.statSync(runtimeRoot).isDirectory(), true);
});

test("buildPlaywrightEnv preserves an explicitly configured runtime root", () => {
  const runtimeRoot = path.join(os.tmpdir(), "alter0-playwright-explicit-runtime-test");
  const sentinel = path.join(runtimeRoot, "storage", "keep-session.json");
  fs.rmSync(runtimeRoot, { recursive: true, force: true });
  fs.mkdirSync(path.dirname(sentinel), { recursive: true });
  fs.writeFileSync(sentinel, "keep");

  const env = buildPlaywrightEnv({ ALTER0_RUNTIME_ROOT: runtimeRoot });

  assert.equal(env.ALTER0_RUNTIME_ROOT, runtimeRoot);
  assert.equal(env.ALTER0_STORAGE_DIR, undefined);
  assert.equal(fs.readFileSync(sentinel, "utf8"), "keep");
  fs.rmSync(runtimeRoot, { recursive: true, force: true });
});
