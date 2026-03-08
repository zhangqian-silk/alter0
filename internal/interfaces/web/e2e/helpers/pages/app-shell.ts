import { type Locator, type Page } from "@playwright/test";

export function createAppShellPage(page: Page): {
  routeMenuItem(route: string): Locator;
} {
  return {
    routeMenuItem: (route: string) => page.locator(`.menu-item[data-route="${route}"]`),
  };
}
