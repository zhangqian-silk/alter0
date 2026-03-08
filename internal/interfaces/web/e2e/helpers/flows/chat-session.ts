import { expect, type Page } from "@playwright/test";
import { createChatPage } from "../pages/chat";
import { clickWithUnsavedDialog, type ConfirmAction } from "../guards/unsaved";

export async function createNewChatSession(page: Page, confirmUnsaved = true): Promise<void> {
  const chatPage = createChatPage(page);
  if (confirmUnsaved) {
    await clickWithUnsavedDialog(page, chatPage.newChatButton(), "accept");
    return;
  }
  await chatPage.newChatButton().click();
}

export async function switchChatSession(page: Page, index: number, action: ConfirmAction = "accept"): Promise<void> {
  const chatPage = createChatPage(page);
  await clickWithUnsavedDialog(page, chatPage.sessionList().itemAt(index), action);
}

export async function removeChatSession(page: Page, index: number, confirmUnsaved = true): Promise<void> {
  const chatPage = createChatPage(page);
  const deleteButton = chatPage.sessionList().deleteButtonAt?.(index);
  if (!deleteButton) {
    throw new Error("chat session list is missing delete controls");
  }
  if (confirmUnsaved) {
    await clickWithUnsavedDialog(page, deleteButton, "accept");
    return;
  }
  await deleteButton.click();
}

export async function expectActiveChatSession(page: Page, index: number): Promise<void> {
  const chatPage = createChatPage(page);
  await expect(chatPage.sessionList().itemAt(index)).toHaveClass(/active/);
}
