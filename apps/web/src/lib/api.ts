export type ShortLink = {
  code: string;
  url: string;
  enabled: boolean;
  createdAt: string;
  expiresAt?: string;
};

export type LinkPage = { links: ShortLink[]; nextCursor?: string };

const baseURL = (
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080"
).replace(/\/$/, "");

export const shortURL = (code: string) =>
  `${baseURL}/r/${encodeURIComponent(code)}`;

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${baseURL}${path}`, {
      ...init,
      cache: "no-store",
      signal: AbortSignal.timeout(10000),
    });
  } catch {
    throw new Error(
      "Cannot reach the local API. Check that it is running, then try again.",
    );
  }
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new Error(
      body?.error?.message ||
        `Request failed (${response.status}). Please try again.`,
    );
  }
  return response.json() as Promise<T>;
}

export const api = {
  create: (input: { url: string; alias?: string; expiresAt?: string }) =>
    request<ShortLink>("/api/links", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    }),
  list: (cursor?: string) =>
    request<LinkPage>(
      `/api/links${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`,
    ),
  setEnabled: (code: string, enabled: boolean) =>
    request<ShortLink>(`/api/links/${encodeURIComponent(code)}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    }),
};
