export const MAX_COMPOSER_IMAGE_ATTACHMENTS = 5;
export const MAX_COMPOSER_IMAGE_BYTES = 5 * 1024 * 1024;
export const MAX_COMPOSER_IMAGE_DIMENSION_PX = 1600;
export const MAX_COMPOSER_IMAGE_DATA_URL_BYTES = 1_500_000;
export const MAX_COMPOSER_IMAGE_PREVIEW_DIMENSION_PX = 240;

export type ComposerImageAttachment = {
  id: string;
  name: string;
  contentType: string;
  size: number;
  dataURL?: string;
  previewDataURL?: string;
  assetURL?: string;
  previewURL?: string;
};

export async function readComposerImageFiles(files: FileList | File[]): Promise<ComposerImageAttachment[]> {
  const selected = Array.from(files || []);
  const attachments: ComposerImageAttachment[] = [];
  for (const file of selected) {
    if (!file.type.startsWith("image/")) {
      throw new Error("Only image files are supported.");
    }
    if (file.size > MAX_COMPOSER_IMAGE_BYTES) {
      throw new Error("Each image must be 5 MB or smaller.");
    }
    const optimized = await buildOptimizedImagePayload(file);
    attachments.push({
      id: `composer-image-${Math.random().toString(36).slice(2, 10)}`,
      name: file.name || "image",
      contentType: optimized.contentType,
      size: optimized.size,
      dataURL: optimized.dataURL,
      previewDataURL: optimized.previewDataURL,
    });
  }
  return attachments;
}

export function resolveComposerAttachmentPreviewURL(attachment: ComposerImageAttachment): string {
  return attachment.previewURL
    || attachment.assetURL
    || attachment.previewDataURL
    || attachment.dataURL
    || "";
}

async function buildOptimizedImagePayload(file: File) {
  if (!shouldRasterizeImage(file.type)) {
    const dataURL = await readFileAsDataURL(file);
    return {
      contentType: file.type || "image/*",
      size: estimateDataURLBytes(dataURL),
      dataURL,
      previewDataURL: dataURL,
    };
  }

  const sourceImage = await loadImageFromFile(file);
  const { width, height } = fitWithinBounds(sourceImage.width, sourceImage.height, MAX_COMPOSER_IMAGE_DIMENSION_PX);
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Failed to read image file.");
  }
  context.drawImage(sourceImage.element, 0, 0, width, height);

  const preferredContentType = resolvePreferredContentType(file.type);
  let dataURL = canvas.toDataURL(preferredContentType, 0.82);
  let size = estimateDataURLBytes(dataURL);
  if (size > MAX_COMPOSER_IMAGE_DATA_URL_BYTES) {
    for (const quality of [0.74, 0.66, 0.58]) {
      dataURL = canvas.toDataURL(preferredContentType, quality);
      size = estimateDataURLBytes(dataURL);
      if (size <= MAX_COMPOSER_IMAGE_DATA_URL_BYTES) {
        break;
      }
    }
  }
  return {
    contentType: extractDataURLContentType(dataURL) || preferredContentType,
    size,
    dataURL,
    previewDataURL: await buildPreviewDataURL(dataURL, extractDataURLContentType(dataURL) || preferredContentType),
  };
}

async function buildPreviewDataURL(dataURL: string, contentType: string) {
  try {
    const image = await loadImageFromURL(dataURL);
    const { width, height } = fitWithinBounds(
      image.width,
      image.height,
      MAX_COMPOSER_IMAGE_PREVIEW_DIMENSION_PX,
    );
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d");
    if (!context) {
      return dataURL;
    }
    context.drawImage(image.element, 0, 0, width, height);
    const previewType = resolvePreferredContentType(contentType);
    const previewDataURL = canvas.toDataURL(previewType, 0.7);
    if (estimateDataURLBytes(previewDataURL) >= estimateDataURLBytes(dataURL)) {
      return dataURL;
    }
    return previewDataURL;
  } catch {
    return dataURL;
  }
}

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(new Error("Failed to read image file."));
    reader.onload = () => {
      if (typeof reader.result !== "string" || reader.result.trim() === "") {
        reject(new Error("Failed to read image file."));
        return;
      }
      resolve(reader.result);
    };
    reader.readAsDataURL(file);
  });
}

function shouldRasterizeImage(contentType: string) {
  const normalized = contentType.trim().toLowerCase();
  return normalized !== "image/gif" && normalized !== "image/svg+xml";
}

function resolvePreferredContentType(contentType: string) {
  const normalized = contentType.trim().toLowerCase();
  if (normalized === "image/png") {
    return "image/webp";
  }
  if (normalized === "image/webp" || normalized === "image/jpeg") {
    return normalized;
  }
  return "image/jpeg";
}

function fitWithinBounds(width: number, height: number, maxDimension: number) {
  if (width <= maxDimension && height <= maxDimension) {
    return { width, height };
  }
  const scale = Math.min(maxDimension / Math.max(width, 1), maxDimension / Math.max(height, 1));
  return {
    width: Math.max(1, Math.round(width * scale)),
    height: Math.max(1, Math.round(height * scale)),
  };
}

async function loadImageFromFile(file: File): Promise<{ element: HTMLImageElement; width: number; height: number }> {
  const objectURL = typeof URL.createObjectURL === "function"
    ? URL.createObjectURL(file)
    : await readFileAsDataURL(file);
  return loadImageFromURL(objectURL, objectURL.startsWith("blob:"));
}

function loadImageFromURL(sourceURL: string, revokeAfterLoad = false): Promise<{ element: HTMLImageElement; width: number; height: number }> {
  const objectURL = sourceURL;
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onerror = () => {
      if (revokeAfterLoad && typeof URL.revokeObjectURL === "function") {
        URL.revokeObjectURL(objectURL);
      }
      reject(new Error("Failed to read image file."));
    };
    image.onload = () => {
      const width = image.naturalWidth || image.width;
      const height = image.naturalHeight || image.height;
      if (revokeAfterLoad && typeof URL.revokeObjectURL === "function") {
        URL.revokeObjectURL(objectURL);
      }
      if (!width || !height) {
        reject(new Error("Failed to read image file."));
        return;
      }
      resolve({ element: image, width, height });
    };
    image.src = objectURL;
  });
}

function estimateDataURLBytes(dataURL: string) {
  const markerIndex = dataURL.indexOf(",");
  if (markerIndex < 0) {
    return dataURL.length;
  }
  const payload = dataURL.slice(markerIndex + 1);
  const padding = payload.endsWith("==") ? 2 : payload.endsWith("=") ? 1 : 0;
  return Math.max(0, Math.floor((payload.length * 3) / 4) - padding);
}

function extractDataURLContentType(dataURL: string) {
  const match = /^data:([^;,]+)[;,]/i.exec(dataURL);
  return match?.[1]?.trim().toLowerCase() || "";
}
