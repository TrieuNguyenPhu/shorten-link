import { afterEach, describe, expect, it, vi } from "vitest";

import { createShortLink, ShortenerApiError } from "./shortener-api";

const link = {
  code: "abc1234",
  short_url: "https://npt-shortenlink.dev/link/abc1234",
  target_url: "https://example.com/docs",
  status: "active" as const,
  created_at: "2026-07-26T00:00:00Z",
  expires_at: null,
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("createShortLink", () => {
  it("sends the v2 request and returns a validated link envelope", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: link }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      createShortLink({
        url: "https://example.com/docs",
        custom_alias: "abc1234",
        expires_in_days: 30,
      }),
    ).resolves.toEqual(link);

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/links", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        url: "https://example.com/docs",
        custom_alias: "abc1234",
        expires_in_days: 30,
      }),
      signal: undefined,
    });
  });

  it("preserves an application error code and status", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: {
              code: "custom_alias_conflict",
              message: "custom_alias is already in use",
            },
          }),
          { status: 409 },
        ),
      ),
    );

    await expect(
      createShortLink({ url: "https://example.com" }),
    ).rejects.toMatchObject({
      name: "ShortenerApiError",
      code: "custom_alias_conflict",
      status: 409,
    } satisfies Partial<ShortenerApiError>);
  });

  it("rejects a malformed success envelope", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: { ...link, short_url: "javascript:alert(1)" },
          }),
          { status: 201 },
        ),
      ),
    );

    await expect(
      createShortLink({ url: "https://example.com" }),
    ).rejects.toMatchObject({
      name: "ShortenerApiError",
      code: "invalid_response",
      status: 201,
    } satisfies Partial<ShortenerApiError>);
  });
});
