import { type Page } from "@playwright/test";
import { openCronRoute } from "../flows/routes";
import { createCronPage } from "../pages/cron";

export async function openCronWorkspace(page: Page): Promise<{
  cronPage: ReturnType<typeof createCronPage>;
}> {
  await openCronRoute(page);
  const cronPage = createCronPage(page);
  return { cronPage };
}
