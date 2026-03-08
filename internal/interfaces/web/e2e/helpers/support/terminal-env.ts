import path from "node:path";

export const terminalRepoRoot = path.resolve(__dirname, "../../../../../..");
export const terminalCommandPreview = "pwd";
export const interruptedTerminalPrompt = process.platform === "win32"
  ? `PS ${terminalRepoRoot}>`
  : `${terminalRepoRoot} $`;
