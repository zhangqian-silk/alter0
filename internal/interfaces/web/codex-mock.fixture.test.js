const test = require("node:test");
const assert = require("node:assert/strict");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

const shellScript = path.join(__dirname, "e2e/fixtures/codex-mock.sh");

function runMock(args) {
  const stdout = execFileSync("sh", [shellScript, ...args], {
    cwd: __dirname,
    encoding: "utf8",
  });
  return stdout.trim().split("\n").map((line) => JSON.parse(line));
}

test("codex mock preserves recovered thread ids for resume flows after exec options", () => {
  const events = runMock([
    "exec",
    "--enable",
    "bwrap",
    "resume",
    "--json",
    "--skip-git-repo-check",
    "thread-recovered",
    "Reply with exactly: resumed",
  ]);
  const reasoningEvent = events.find((event) => event?.item?.type === "reasoning");
  const messageEvent = events.find((event) => event?.item?.type === "agent_message");

  assert.equal(events[0]?.type, "thread.started");
  assert.equal(events[0]?.thread_id, "thread-recovered");
  assert.equal(reasoningEvent?.item?.text, "Resume prior Codex CLI context.");
  assert.equal(messageEvent?.item?.text, "resumed");
});
