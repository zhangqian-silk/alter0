import { APIClientError, createAPIClient } from "./client";

describe("api client", () => {
  it("sends JSON requests and returns parsed JSON payloads", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ item: { id: "provider-1" } }), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      }),
    );

    const client = createAPIClient({
      baseURL: "http://127.0.0.1:18088",
      fetchImpl: fetchMock
    });

    const result = await client.post<{ item: { id: string } }>("/api/control/llm/providers", {
      name: "OpenAI"
    });

    expect(result).toEqual({ item: { id: "provider-1" } });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [requestURL, requestInit] = fetchMock.mock.calls[0];
    const headers = requestInit?.headers as Headers;

    expect(requestURL).toBe("http://127.0.0.1:18088/api/control/llm/providers");
    expect(requestInit?.method).toBe("POST");
    expect(requestInit?.body).toBe(JSON.stringify({ name: "OpenAI" }));
    expect(headers.get("Accept")).toBe("application/json");
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  it("returns undefined for 204 responses", async () => {
    const client = createAPIClient({
      fetchImpl: vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    });

    await expect(client.delete("/api/sessions/session-1")).resolves.toBeUndefined();
  });

  it("calls the unauthorized hook for 401 responses", async () => {
    const onUnauthorized = vi.fn();
    const client = createAPIClient({
      fetchImpl: vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "authentication required" }), {
          status: 401,
          headers: { "Content-Type": "application/json" }
        }),
      ),
      onUnauthorized
    });

    await expect(client.get("/api/sessions")).rejects.toBeInstanceOf(APIClientError);
    expect(onUnauthorized).toHaveBeenCalledTimes(1);
  });

  it("surfaces structured API errors with status and code", async () => {
    const client = createAPIClient({
      fetchImpl: vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "provider not found", code: "provider_missing" }), {
          status: 404,
          headers: { "Content-Type": "application/json" }
        }),
      )
    });

    await expect(client.get("/api/control/llm/providers/missing")).rejects.toMatchObject({
      name: "APIClientError",
      message: "provider not found",
      status: 404,
      code: "provider_missing"
    });
  });
});
