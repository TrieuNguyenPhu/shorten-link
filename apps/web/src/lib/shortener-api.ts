const generateShortURLPath = "/api/generate-short-url";

interface LegacyShortenResponse {
  short_url_code: string;
}

export type ShortenerApiErrorKind = "http" | "invalid-response";

export interface ShortLinkResult {
  code: string;
  path: string;
}

export class ShortenerApiError extends Error {
  constructor(
    public readonly kind: ShortenerApiErrorKind,
    public readonly status?: number,
  ) {
    super(
      kind === "http"
        ? `Legacy API request failed with status ${status}.`
        : "Legacy API returned an invalid response.",
    );
    this.name = "ShortenerApiError";
  }
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
    throw new ShortenerApiError("http", response.status);
  }

  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    throw new ShortenerApiError("invalid-response");
  }

  if (!isLegacyShortenResponse(payload)) {
    throw new ShortenerApiError("invalid-response");
  }

  const code = payload.short_url_code.trim();
  return {
    code,
    path: `/link/${encodeURIComponent(code)}`,
  };
}
