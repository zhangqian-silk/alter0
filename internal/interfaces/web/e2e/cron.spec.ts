import { expect, test } from "@playwright/test";
import { openCronWorkspace } from "./helpers/scenarios/cron";

test.describe("Cron jobs route", () => {
  test("renders cron jobs from the scheduler control API", async ({ page }) => {
    await page.route("**/api/control/cron/jobs", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          items: [
            {
              id: "job-daily",
              name: "Daily Summary",
              enabled: true,
              schedule_mode: "daily",
              cron_expression: "30 9 * * *",
              timezone: "Asia/Shanghai",
              task_config: {
                input: "summarize latest tasks",
                retry_limit: 2,
              },
            },
          ],
        }),
      });
    });

    const { cronPage } = await openCronWorkspace(page);

    await expect(cronPage.routeGrid()).toBeVisible();
    await expect(cronPage.card("Daily Summary")).toContainText("30 9 * * *");
    await expect(cronPage.card("Daily Summary")).toContainText("Asia/Shanghai");
    await expect(cronPage.card("Daily Summary")).toContainText("summarize latest tasks");
  });
});
