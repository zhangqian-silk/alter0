import path from "node:path";

export const terminalRepoRoot = path.resolve(__dirname, "../../../../../..");
export function terminalSessionWorkspace(sessionID: string): string {
  return path.join(terminalRepoRoot, ".alter0", "workspaces", "terminal", "sessions", sessionID);
}
export const terminalCommandPreview = "pwd";
export const interruptedTerminalPrompt = process.platform === "win32"
  ? `PS ${terminalRepoRoot}>`
  : `${terminalRepoRoot} $`;
