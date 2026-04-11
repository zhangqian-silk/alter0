export type APIClientFetch = typeof fetch;

export type APIClientOptions = {
  baseURL?: string;
  fetchImpl?: APIClientFetch;
  onUnauthorized?: () => void;
};

export type APIRequestOptions = Omit<RequestInit, "body"> & {
  body?: BodyInit | Record<string, unknown> | null;
};

type ErrorPayload = {
  code?: string;
  error?: string;
  message?: string;
};

export class APIClientError extends Error {
  status: number;
  code: string;
  payload: unknown;

  constructor(message: string, status: number, code: string = "", payload: unknown = null) {
    super(message);
    this.name = "APIClientError";
    this.status = status;
    this.code = code;
    this.payload = payload;
  }
}

export function createAPIClient(options: APIClientOptions = {}) {
  const baseURL = normalizeBaseURL(options.baseURL);
  const fetchImpl = options.fetchImpl ?? fetch;
  const onUnauthorized = options.onUnauthorized;

  async function request<T>(path: string, init: APIRequestOptions = {}): Promise<T> {
    const response = await fetchImpl(resolveURL(baseURL, path), {
      ...init,
      headers: buildHeaders(init.headers, init.body),
      body: serializeRequestBody(init.body)
    });

    if (response.status === 204) {
      return undefined as T;
    }

    const payload = await readResponsePayload(response);
    if (!response.ok) {
      if (response.status === 401) {
        onUnauthorized?.();
      }
      throw toAPIClientError(response.status, response.statusText, payload);
    }

    return payload as T;
  }

  return {
    request,
    get<T>(path: string, init: Omit<APIRequestOptions, "method"> = {}) {
      return request<T>(path, { ...init, method: "GET" });
    },
    post<T>(path: string, body?: APIRequestOptions["body"], init: Omit<APIRequestOptions, "method" | "body"> = {}) {
      return request<T>(path, { ...init, method: "POST", body });
    },
    put<T>(path: string, body?: APIRequestOptions["body"], init: Omit<APIRequestOptions, "method" | "body"> = {}) {
      return request<T>(path, { ...init, method: "PUT", body });
    },
    patch<T>(path: string, body?: APIRequestOptions["body"], init: Omit<APIRequestOptions, "method" | "body"> = {}) {
      return request<T>(path, { ...init, method: "PATCH", body });
    },
    delete<T>(path: string, init: Omit<APIRequestOptions, "method"> = {}) {
      return request<T>(path, { ...init, method: "DELETE" });
    }
  };
}

function normalizeBaseURL(baseURL?: string): string {
  return String(baseURL || "").trim().replace(/\/$/, "");
}

function resolveURL(baseURL: string, path: string): string {
  if (!baseURL) {
    return path;
  }
  if (/^https?:\/\//i.test(path)) {
    return path;
  }
  return `${baseURL}${path.startsWith("/") ? path : `/${path}`}`;
}

function buildHeaders(headers: HeadersInit | undefined, body: APIRequestOptions["body"]): Headers {
  const nextHeaders = new Headers(headers);
  if (!nextHeaders.has("Accept")) {
    nextHeaders.set("Accept", "application/json");
  }
  if (isJSONBody(body) && !nextHeaders.has("Content-Type")) {
    nextHeaders.set("Content-Type", "application/json");
  }
  return nextHeaders;
}

function serializeRequestBody(body: APIRequestOptions["body"]): BodyInit | undefined {
  if (body == null) {
    return undefined;
  }
  if (isJSONBody(body)) {
    return JSON.stringify(body);
  }
  return body;
}

function isJSONBody(body: APIRequestOptions["body"]): body is Record<string, unknown> {
  if (body == null) {
    return false;
  }
  if (typeof body !== "object") {
    return false;
  }
  if (body instanceof FormData || body instanceof Blob || body instanceof URLSearchParams || ArrayBuffer.isView(body) || body instanceof ArrayBuffer) {
    return false;
  }
  return true;
}

async function readResponsePayload(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) {
    return undefined;
  }
  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) {
    try {
      return JSON.parse(text);
    } catch {
      return text;
    }
  }
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function toAPIClientError(status: number, statusText: string, payload: unknown): APIClientError {
  const normalizedPayload = payload as ErrorPayload | string | undefined;
  if (typeof normalizedPayload === "string") {
    return new APIClientError(normalizedPayload || statusText || "request failed", status, "", payload);
  }
  const message = normalizedPayload?.error || normalizedPayload?.message || statusText || "request failed";
  const code = normalizedPayload?.code || "";
  return new APIClientError(message, status, code, payload);
}
