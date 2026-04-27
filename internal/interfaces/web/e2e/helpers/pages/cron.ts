import { type Locator, type Page } from "@playwright/test";

export function createCronPage(page: Page): {
  routeGrid(): Locator;
  card(title: string): Locator;
} {
  return {
    routeGrid: () => page.locator('[data-control-route-grid="cron-jobs"]'),
    card: (title: string) => page.locator(".route-card").filter({ hasText: title }).first(),
  };
}
