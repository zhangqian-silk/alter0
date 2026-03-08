import { expect, type Locator, type Page } from "@playwright/test";

export type ConfirmAction = "accept" | "dismiss";

export async function handleNextUnsavedDialog(page: Page, action: ConfirmAction): Promise<void> {
  await expect(page.locator("body")).toHaveAttribute("data-composer-unsaved-state", "dirty");
  page.once("dialog", async (dialog) => {
    expect(dialog.type()).toBe("confirm");
    if (action === "accept") {
      await dialog.accept();
      return;
    }
    await dialog.dismiss();
  });
}

export async function clickWithUnsavedDialog(page: Page, target: Locator, action: ConfirmAction): Promise<void> {
  await handleNextUnsavedDialog(page, action);
  await target.click();
  await expect(page.locator("body")).toHaveAttribute(
    "data-composer-unsaved-confirm",
    action === "accept" ? "accepted" : "dismissed"
  );
}
