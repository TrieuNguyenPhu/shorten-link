import type { PublicConfig } from "./config";

export type Link = {
  code: string;
  target: string;
  clicks: number;
  createdAt: string;
  shortUrl?: string;
  realtimeTicket?: string;
};

export type LinkPage = { links: Link[]; nextCursor?: string };
export type CreatedLink = { code: string; shortUrl: string; target: string };

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

type FetchLike = typeof fetch;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isCreatedLink(value: unknown): value is CreatedLink {
  return isRecord(value) && typeof value.code === "string" && typeof value.target === "string" && typeof value.shortUrl === "string";
}

function isLinkPage(value: unknown): value is LinkPage {
  return isRecord(value) && Array.isArray(value.links) && value.links.every((link) => isRecord(link) && typeof link.code === "string" && typeof link.target === "string" && typeof link.clicks === "number" && typeof link.createdAt === "string");
}

async function readResponse<T>(response: Response, validate?: (value: unknown) => value is T): Promise<T> {
  let payload: unknown = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  if (!response.ok) {
    const message = typeof payload === "object" && payload && "error" in payload && typeof payload.error === "string" ? payload.error : `API request failed (${response.status})`;
    throw new ApiError(response.status, message);
  }
  if (validate && !validate(payload)) throw new ApiError(response.status, "API response was invalid");
  return payload as T;
}

function headers(token: string, idempotencyKey?: string): HeadersInit {
  return { Authorization: `Bearer ${token}`, "Content-Type": "application/json", ...(idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}) };
}

async function request(fetchImpl: FetchLike, url: string, init: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 8_000);
  if (init.signal) {
    if (init.signal.aborted) controller.abort();
    else init.signal.addEventListener("abort", () => controller.abort(), { once: true });
  }
  try {
    return await fetchImpl(url, { ...init, signal: controller.signal });
  } catch (error) {
    if (controller.signal.aborted) throw new ApiError(408, "API request timed out");
    throw new ApiError(0, error instanceof Error ? error.message : "API request failed");
  } finally {
    clearTimeout(timeout);
  }
}

export async function createLink(config: PublicConfig, token: string, target: string, fetchImpl: FetchLike = fetch): Promise<CreatedLink> {
  const response = await request(fetchImpl, `${config.apiBaseUrl}/links`, { method: "POST", headers: headers(token, crypto.randomUUID()), body: JSON.stringify({ url: target }) });
  return readResponse<CreatedLink>(response, isCreatedLink);
}

export async function listLinks(config: PublicConfig, token: string, options: { limit?: number; cursor?: string } = {}, fetchImpl: FetchLike = fetch): Promise<LinkPage> {
  const params = new URLSearchParams();
  if (options.limit) params.set("limit", String(options.limit));
  if (options.cursor) params.set("cursor", options.cursor);
  const query = params.toString();
  const response = await request(fetchImpl, `${config.apiBaseUrl}/links${query ? `?${query}` : ""}`, { method: "GET", headers: headers(token) });
  return readResponse<LinkPage>(response, isLinkPage);
}

export async function listAllLinks(config: PublicConfig, token: string, fetchImpl: FetchLike = fetch): Promise<Link[]> {
  const all: Link[] = [];
  let cursor: string | undefined;
  const seen = new Set<string>();
  do {
    const page = await listLinks(config, token, { limit: 100, cursor }, fetchImpl);
    all.push(...page.links);
    cursor = page.nextCursor;
    if (cursor && seen.has(cursor)) throw new ApiError(500, "API pagination repeated a cursor");
    if (cursor) seen.add(cursor);
  } while (cursor);
  return all;
}

export async function deleteLink(config: PublicConfig, token: string, code: string, fetchImpl: FetchLike = fetch): Promise<{ deleted: string }> {
  const response = await request(fetchImpl, `${config.apiBaseUrl}/links/${encodeURIComponent(code)}`, { method: "DELETE", headers: headers(token) });
  return readResponse<{ deleted: string }>(response, (value): value is { deleted: string } => isRecord(value) && value.deleted === code);
}
