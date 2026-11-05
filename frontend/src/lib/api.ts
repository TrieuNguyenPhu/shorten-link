import type { PublicConfig } from "./config";

export type Link = {
  code: string;
  target: string;
  clicks: number;
  createdAt: string;
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

async function readResponse<T>(response: Response): Promise<T> {
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
  return payload as T;
}

function headers(token: string): HeadersInit {
  return { Authorization: `Bearer ${token}`, "Content-Type": "application/json" };
}

export async function createLink(config: PublicConfig, token: string, target: string, fetchImpl: FetchLike = fetch): Promise<CreatedLink> {
  const response = await fetchImpl(`${config.apiBaseUrl}/links`, { method: "POST", headers: headers(token), body: JSON.stringify({ url: target }) });
  return readResponse<CreatedLink>(response);
}

export async function listLinks(config: PublicConfig, token: string, options: { limit?: number; cursor?: string } = {}, fetchImpl: FetchLike = fetch): Promise<LinkPage> {
  const params = new URLSearchParams();
  if (options.limit) params.set("limit", String(options.limit));
  if (options.cursor) params.set("cursor", options.cursor);
  const query = params.toString();
  const response = await fetchImpl(`${config.apiBaseUrl}/links${query ? `?${query}` : ""}`, { method: "GET", headers: headers(token) });
  return readResponse<LinkPage>(response);
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
  const response = await fetchImpl(`${config.apiBaseUrl}/links/${encodeURIComponent(code)}`, { method: "DELETE", headers: headers(token) });
  return readResponse<{ deleted: string }>(response);
}
