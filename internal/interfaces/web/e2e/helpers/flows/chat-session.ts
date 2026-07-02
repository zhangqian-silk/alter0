import { expect, type Page } from "@playwright/test";
import { createChatPage } from "../pages/chat";

export async function createNewChatSession(page: Page): Promise<void> {
  const chatPage = createChatPage(page);
  await chatPage.newChatButton().click();
}

export async function switchChatSession(page: Page, index: number): Promise<void> {
  const chatPage = createChatPage(page);
  await chatPage.sessionList().itemAt(index).click();
}

export async function removeChatSession(page: Page, index: number): Promise<void> {
  const chatPage = createChatPage(page);
  const sessionCard = chatPage.sessionList().itemAt(index);
  await sessionCard.getByRole("button", { name: "Session actions" }).click();
  const deleteButton = page.getByRole("menuitem", { name: "Delete" });
  page.once("dialog", async (dialog) => {
    await dialog.accept();
  });
  await deleteButton.click();
}

export async function expectActiveChatSession(page: Page, index: number): Promise<void> {
  const chatPage = createChatPage(page);
  await expect(chatPage.sessionList().itemAt(index)).toHaveClass(/active/);
}
