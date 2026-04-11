type LegacyShellQueryRoot = Pick<Document, "querySelector">;

export const LEGACY_SHELL_IDS = {
  appShell: "appShell",
  navCollapseButton: "navCollapseButton",
  sessionList: "sessionList",
  welcomeScreen: "welcomeScreen",
  messageArea: "messageArea",
  composerInput: "composerInput",
  charCount: "charCount",
  newChatButton: "newChatButton",
  navToggle: "navToggle",
  sessionToggle: "sessionToggle",
  mobileBackdrop: "mobileBackdrop",
  routeView: "routeView",
  routeBody: "routeBody",
  chatRuntimePanel: "chatRuntimePanel",
  chatRuntimeSheetHost: "chatRuntimeSheetHost"
} as const;

export const LEGACY_SHELL_REQUIRED_IDS = Object.values(LEGACY_SHELL_IDS);

export function getMissingLegacyShellIds(root: LegacyShellQueryRoot = document): string[] {
  return LEGACY_SHELL_REQUIRED_IDS.filter((id) => root.querySelector(`#${id}`) === null);
}

export function hasLegacyShellContract(root: LegacyShellQueryRoot = document): boolean {
  return getMissingLegacyShellIds(root).length === 0;
}
