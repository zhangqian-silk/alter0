import { type Locator, type Page } from "@playwright/test";

export type SessionListComponent = {
  items(): Locator;
  itemAt(index: number): Locator;
  itemByValue?(value: string): Locator;
  deleteButtons?(): Locator;
  deleteButtonAt?(index: number): Locator;
};

export function createSessionListComponent(
  page: Page,
  selectors: {
    items: string;
    itemByValue?: (value: string) => string;
    deleteButtons?: string;
  }
): SessionListComponent {
  const items = () => page.locator(selectors.items);

  return {
    items,
    itemAt: (index: number) => items().nth(index),
    ...(selectors.itemByValue
      ? {
          itemByValue: (value: string) => page.locator(selectors.itemByValue(value)),
        }
      : {}),
    ...(selectors.deleteButtons
      ? {
          deleteButtons: () => page.locator(selectors.deleteButtons),
          deleteButtonAt: (index: number) => page.locator(selectors.deleteButtons).nth(index),
        }
      : {}),
  };
}
