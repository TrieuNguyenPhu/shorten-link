const generateShortURLPath = "/api/generate-short-url";

interface LegacyShortenResponse {
  short_url_code: string;
}

export interface ShortLinkResult {
  code: string;
  path: string;
}

function isLegacyShortenResponse(
  value: unknown,
): value is LegacyShortenResponse {
  if (!value || typeof value !== "object") {
    return false;
  }

  const response = value as Record<string, unknown>;
  return (
    typeof response.short_url_code === "string" &&
    response.short_url_code.trim().length > 0
  );
}

export async function createShortLink(url: string): Promise<ShortLinkResult> {
  const response = await fetch(generateShortURLPath, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ url }),
  });

  if (!response.ok) {
    throw new Error("Legacy API request failed.");
  }

  const payload: unknown = await response.json();
  if (!isLegacyShortenResponse(payload)) {
    throw new Error("Legacy API returned an invalid response.");
  }

  const code = payload.short_url_code.trim();
  return {
    code,
    path: `/link/${encodeURIComponent(code)}`,
  };
}
