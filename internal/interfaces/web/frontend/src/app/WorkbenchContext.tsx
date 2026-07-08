import { createContext, useContext, type ComponentPropsWithoutRef, type ReactNode } from "react";
import type { LegacyShellLanguage } from "../features/shell/legacyShellCopy";

export type WorkbenchSessionRail = {
  route: string;
  countLabel: string;
  versionKey?: string;
  onPrimaryAction: () => void;
  body: ReactNode;
  primaryActionClassName?: string;
  primaryActionProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
};

export type WorkbenchContextValue = {
  route: string;
  language: LegacyShellLanguage;
  navigate: (route: string) => void;
  isMobileViewport: boolean;
  mobileNavOpen: boolean;
  mobileSessionPaneOpen: boolean;
  toggleMobileNav: () => void;
  toggleMobileSessionPane: () => void;
  openMobileSessionPane: () => void;
  closeMobileNav: () => void;
  closeMobileSessionPane: () => void;
  setRuntimeSessionRail?: (rail: WorkbenchSessionRail | null) => void;
};

export const WorkbenchContext = createContext<WorkbenchContextValue | null>(null);

export function useWorkbenchContext(): WorkbenchContextValue {
  const value = useContext(WorkbenchContext);
  if (!value) {
    throw new Error("WorkbenchContext is not available");
  }
  return value;
}
