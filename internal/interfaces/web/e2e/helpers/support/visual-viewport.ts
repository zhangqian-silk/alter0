import type { Page } from "@playwright/test";

export type VisualViewportShape = {
  width?: number;
  height?: number;
  offsetTop?: number;
  offsetLeft?: number;
};

export async function installVisualViewportMock(page: Page): Promise<void> {
  await page.addInitScript(() => {
    class MockVisualViewport extends EventTarget {
      width: number;
      height: number;
      offsetTop: number;
      offsetLeft: number;
      pageTop: number;
      pageLeft: number;
      scale: number;

      constructor() {
        super();
        this.width = window.innerWidth;
        this.height = window.innerHeight;
        this.offsetTop = 0;
        this.offsetLeft = 0;
        this.pageTop = 0;
        this.pageLeft = 0;
        this.scale = 1;
      }
    }

    const mock = new MockVisualViewport();
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: mock,
    });
    Object.defineProperty(window, "__alter0SetVisualViewport", {
      configurable: true,
      value: (next: VisualViewportShape) => {
        if (!next || typeof next !== "object") {
          return;
        }
        if (typeof next.width === "number") {
          mock.width = next.width;
        }
        if (typeof next.height === "number") {
          mock.height = next.height;
        }
        if (typeof next.offsetTop === "number") {
          mock.offsetTop = next.offsetTop;
          mock.pageTop = next.offsetTop;
        }
        if (typeof next.offsetLeft === "number") {
          mock.offsetLeft = next.offsetLeft;
          mock.pageLeft = next.offsetLeft;
        }
        mock.dispatchEvent(new Event("resize"));
        mock.dispatchEvent(new Event("scroll"));
      },
    });
  });
}

export async function setVisualViewport(
  page: Page,
  next: VisualViewportShape
): Promise<void> {
  await page.evaluate((value) => {
    const setter = (window as typeof window & {
      __alter0SetVisualViewport?: (payload: typeof value) => void;
    }).__alter0SetVisualViewport;
    setter?.(value);
  }, next);
}
