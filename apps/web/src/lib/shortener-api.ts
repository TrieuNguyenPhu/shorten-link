export type LinkStatus = "active" | "expired" | "disabled";

export interface ShortLink {
  code: string;
  short_url: string;
  target_url: string;
  status: LinkStatus;
  created_at: string;
  expires_at: string | null;
}

export interface CreateShortLinkInput {
  url: string;
  custom_alias?: string;
  expires_in_days?: number;
}

interface LinkEnvelope {
  data: ShortLink;
}

interface ErrorEnvelope {
  error?: {
    code?: string;
    message?: string;
  };
}

function isHTTPURL(value: unknown): value is string {
  if (typeof value !== "string") {
    return false;
  }
  try {
    const parsed = new URL(value);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

function isDateTime(value: unknown): value is string {
  return typeof value === "string" && !Number.isNaN(Date.parse(value));
}

function isShortLink(value: unknown): value is ShortLink {
  if (!value || typeof value !== "object") {
    return false;
  }
  const link = value as Partial<ShortLink>;
  return (
    typeof link.code === "string" &&
    /^[a-z0-9-]{4,32}$/.test(link.code) &&
    isHTTPURL(link.short_url) &&
    isHTTPURL(link.target_url) &&
    (link.status === "active" ||
      link.status === "expired" ||
      link.status === "disabled") &&
    isDateTime(link.created_at) &&
    (link.expires_at === null || isDateTime(link.expires_at))
  );
}

// Production is intentionally same-origin through CloudFront. The public env
// override is development-only so a local .env file cannot leak localhost into
// the static production bundle.
const API_BASE_URL =
  process.env.NODE_ENV === "development"
    ? (process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080").replace(
        /\/+$/,
        "",
      )
    : "";

export class ShortenerApiError extends Error {
  constructor(
    public readonly code: string,
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ShortenerApiError";
  }
}

export async function createShortLink(
  input: CreateShortLinkInput,
  signal?: AbortSignal,
): Promise<ShortLink> {
  const response = await fetch(`${API_BASE_URL}/api/v1/links`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
    signal,
  });

  const payload = (await response.json().catch(() => null)) as
    | LinkEnvelope
    | ErrorEnvelope
    | null;

  if (!response.ok) {
    const error = payload && "error" in payload ? payload.error : undefined;
    throw new ShortenerApiError(
      error?.code ?? "request_failed",
      response.status,
      error?.message ?? "The API returned an invalid error response.",
    );
  }

  if (!payload || !("data" in payload) || !isShortLink(payload.data)) {
    throw new ShortenerApiError(
      "invalid_response",
      response.status,
      "The API returned an invalid success response.",
    );
  }

  return payload.data;
}
