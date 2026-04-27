const path = require("path");
const { buildPlaywrightEnv } = require("./playwright.config.shared");

const port = process.env.ALTER0_PLAYWRIGHT_PORT || "18188";
const baseURL = `http://127.0.0.1:${port}`;
const isWindows = process.platform === "win32";
const codexMockShell = isWindows ? path.resolve(__dirname, "e2e/fixtures/codex-mock.cmd") : "sh";
const codexMockShellArgs = isWindows ? "" : path.resolve(__dirname, "e2e/fixtures/codex-mock.sh");
const terminalShellFlags = isWindows
  ? `-task-terminal-shell "${codexMockShell}"`
  : `-task-terminal-shell "${codexMockShell}" -task-terminal-shell-args "${codexMockShellArgs}"`;
const playwrightEnv = buildPlaywrightEnv(process.env);

process.env.ALTER0_WEB_LOGIN_PASSWORD = playwrightEnv.ALTER0_WEB_LOGIN_PASSWORD;
process.env.ALTER0_PLAYWRIGHT_BROWSERS_PATH = playwrightEnv.ALTER0_PLAYWRIGHT_BROWSERS_PATH;
process.env.ALTER0_PLAYWRIGHT_PASSWORD_FILE = playwrightEnv.ALTER0_PLAYWRIGHT_PASSWORD_FILE;
process.env.GOCACHE = playwrightEnv.GOCACHE;
process.env.PLAYWRIGHT_BROWSERS_PATH = playwrightEnv.PLAYWRIGHT_BROWSERS_PATH;
process.env.XDG_CACHE_HOME = playwrightEnv.XDG_CACHE_HOME;

/** @type {import('@playwright/test').PlaywrightTestConfig} */
module.exports = {
  testDir: "./e2e",
  outputDir: "../../../output/playwright/test-results",
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL,
    headless: true,
    trace: "on-first-retry",
  },
  webServer: {
    command: `go run ./cmd/alter0 -web-addr 127.0.0.1:${port} ${terminalShellFlags}`,
    cwd: path.resolve(__dirname, "../../.."),
    env: playwrightEnv,
    url: `${baseURL}/chat`,
    reuseExistingServer: false,
    timeout: 120_000,
  },
};
