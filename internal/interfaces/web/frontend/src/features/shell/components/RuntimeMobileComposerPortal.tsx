import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { runtimeMobileLayoutSuspendsComposer, type RuntimeMobileLayoutState } from "./runtimeMobileLayout";

type RuntimeMobileComposerPortalProps = {
  isMobileViewport: boolean;
  route: string;
  composerNode: ReactNode;
  mobileLayoutState?: RuntimeMobileLayoutState;
};

export function RuntimeMobileComposerPortal({
  isMobileViewport,
  route,
  composerNode,
  mobileLayoutState,
}: RuntimeMobileComposerPortalProps) {
  if (!isMobileViewport || !composerNode || typeof document === "undefined") {
    return null;
  }
  const composerPortalView = route === "terminal" ? "terminal" : "conversation";
  const composerSuspended = runtimeMobileLayoutSuspendsComposer(mobileLayoutState);
  return createPortal(
    <div
      className="runtime-composer-portal-host"
      data-runtime-composer-portal-host={route}
      data-runtime-composer-view={composerPortalView}
      data-runtime-composer-suspended={composerSuspended ? "true" : "false"}
      aria-hidden={composerSuspended ? "true" : undefined}
    >
      {composerNode}
    </div>,
    document.body,
  );
}
