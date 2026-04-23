import { afterEach, describe, expect, it, vi } from "vitest";
import {
  MAX_COMPOSER_IMAGE_BYTES,
  readComposerImageFiles,
} from "./composerImageAttachments";

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("readComposerImageFiles", () => {
  it("reads svg image files into composer attachments without rasterizing", async () => {
    const file = new File(['<svg xmlns="http://www.w3.org/2000/svg"></svg>'], "diagram.svg", { type: "image/svg+xml" });

    const attachments = await readComposerImageFiles([file]);

    expect(attachments).toHaveLength(1);
    expect(attachments[0]).toMatchObject({
      name: "diagram.svg",
      contentType: "image/svg+xml",
    });
    expect(attachments[0].dataURL).toContain("data:image/svg+xml;base64,");
  });

  it("downscales raster images before storing them in attachments", async () => {
    const createObjectURL = vi.fn().mockReturnValue("blob:test-image");
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", class extends URL {
      static createObjectURL = createObjectURL;
      static revokeObjectURL = revokeObjectURL;
    });
    const drawImage = vi.fn();
    const toDataURL = vi.fn().mockReturnValue("data:image/webp;base64,QUJDRA==");
    const getContext = vi.fn().mockReturnValue({ drawImage });
    const realCreateElement = document.createElement.bind(document);
    vi.spyOn(document, "createElement").mockImplementation(((tagName: string) => {
      if (tagName === "canvas") {
        return {
          width: 0,
          height: 0,
          getContext,
          toDataURL,
        } as unknown as HTMLCanvasElement;
      }
      return realCreateElement(tagName);
    }) as typeof document.createElement);

    class MockImage {
      onload: null | (() => void) = null;
      onerror: null | (() => void) = null;
      naturalWidth = 3200;
      naturalHeight = 1800;
      width = 3200;
      height = 1800;

      set src(_value: string) {
        queueMicrotask(() => this.onload?.());
      }
    }

    vi.stubGlobal("Image", MockImage);

    const file = new File(["hello"], "diagram.png", { type: "image/png" });
    const attachments = await readComposerImageFiles([file]);

    expect(attachments).toHaveLength(1);
    expect(attachments[0]).toMatchObject({
      name: "diagram.png",
      contentType: "image/webp",
    });
    expect(attachments[0].dataURL).toBe("data:image/webp;base64,QUJDRA==");
    expect(attachments[0].size).toBeGreaterThan(0);
    expect(createObjectURL).toHaveBeenCalledWith(file);
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:test-image");
    expect(drawImage).toHaveBeenCalled();
    expect(toDataURL).toHaveBeenCalledWith("image/webp", 0.82);
  });

  it("rejects non-image files and oversized files", async () => {
    await expect(readComposerImageFiles([
      new File(["hello"], "notes.txt", { type: "text/plain" }),
    ])).rejects.toThrow("Only image files are supported.");

    const oversized = new File(["x"], "huge.png", { type: "image/png" });
    vi.spyOn(oversized, "size", "get").mockReturnValue(MAX_COMPOSER_IMAGE_BYTES + 1);

    await expect(readComposerImageFiles([oversized])).rejects.toThrow("Each image must be 5 MB or smaller.");
  });
});
