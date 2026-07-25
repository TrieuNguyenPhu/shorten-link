import { afterEach, describe, expect, it, vi } from "vitest";

import { createShortLink, ShortenerApiError } from "./shortener-api";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("createShortLink", () => {
  it("uses the legacy URL-only request and response contract", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ short_url_code: "abc123" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      createShortLink("https://example.com/tai-lieu"),
    ).resolves.toEqual({
      code: "abc123",
      path: "/link/abc123",
    });

    expect(fetchMock).toHaveBeenCalledOnce();
    expect(fetchMock).toHaveBeenCalledWith("/api/generate-short-url", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ url: "https://example.com/tai-lieu" }),
    });
  });

  it("normalizes an HTTP failure without changing the response contract", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("Unavailable", { status: 503 })),
    );

    await expect(createShortLink("https://example.com")).rejects.toMatchObject({
      name: "ShortenerApiError",
      kind: "http",
      status: 503,
    } satisfies Partial<ShortenerApiError>);
  });

  it("rejects a successful non-JSON response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("not json", {
          status: 200,
          headers: { "Content-Type": "text/plain" },
        }),
      ),
    );

    await expect(createShortLink("https://example.com")).rejects.toMatchObject({
      name: "ShortenerApiError",
      kind: "invalid-response",
    } satisfies Partial<ShortenerApiError>);
  });
});
