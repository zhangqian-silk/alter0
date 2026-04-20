import { createContext, useContext } from "react";
import type { LegacyShellLanguage } from "../features/shell/legacyShellCopy";

export type WorkbenchContextValue = {
  route: string;
  language: LegacyShellLanguage;
  navigate: (route: string) => void;
  isMobileViewport: boolean;
  mobileNavOpen: boolean;
  toggleMobileNav: () => void;
  closeMobileNav: () => void;
};

export const WorkbenchContext = createContext<WorkbenchContextValue | null>(null);

export function useWorkbenchContext(): WorkbenchContextValue {
  const value = useContext(WorkbenchContext);
  if (!value) {
    throw new Error("WorkbenchContext is not available");
  }
  return value;
}
