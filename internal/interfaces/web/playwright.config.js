const path = require("path");

const port = process.env.ALTER0_PLAYWRIGHT_PORT || "18188";
const baseURL = `http://127.0.0.1:${port}`;

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
    command: `go run ./cmd/alter0 -web-addr 127.0.0.1:${port}`,
    cwd: path.resolve(__dirname, "../../.."),
    url: `${baseURL}/chat`,
    reuseExistingServer: false,
    timeout: 120_000,
  },
};
