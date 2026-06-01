import { describe, expect, it } from "vitest";

import { CODEX_SLASH_COMMANDS } from "./codexSlashCommands";

describe("CODEX_SLASH_COMMANDS", () => {
  it("keeps the Web-safe Codex slash command list in grouped help order", () => {
    expect(CODEX_SLASH_COMMANDS.map((item) => item.command)).toEqual([
      "/apps",
      "/plugins",
      "/hooks",
      "/compact",
      "/diff",
      "/memories",
      "/skills",
      "/init",
      "/mcp",
      "/mention",
      "/model",
      "/fast",
      "/plan",
      "/goal",
      "/personality",
      "/ps",
      "/stop",
      "/review",
      "/status",
    ]);
  });

  it("does not expose permission or TUI-only commands in Web composers", () => {
    expect(CODEX_SLASH_COMMANDS.map((item) => item.command)).not.toEqual(
      expect.arrayContaining([
        "/permissions",
        "/approve",
        "/sandbox-add-read-dir",
        "/keymap",
        "/vim",
        "/copy",
        "/quit",
        "/exit",
      ]),
    );
  });

  it("uses concise command labels for candidate rows", () => {
    for (const item of CODEX_SLASH_COMMANDS) {
      expect(item.label.en.length).toBeLessThanOrEqual(34);
      expect(item.label.zh.length).toBeLessThanOrEqual(14);
    }
  });
});
