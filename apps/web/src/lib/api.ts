import {
  baseURL,
  forceRefreshAccessToken,
  getValidAccessToken,
  isLocalAPI,
  setAuthErrorNotice,
} from "@/lib/auth";

export { baseURL, isLocalAPI };

export type ShortLink = {
  code: string;
  url: string;
  enabled: boolean;
  version?: number;
  createdAt: string;
  updatedAt?: string;
  expiresAt?: string;
};

export type LinkPage = { links: ShortLink[]; nextCursor?: string };

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(message: string, status: number, code: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export const shortURL = (code: string) =>
  `${baseURL}/r/${encodeURIComponent(code)}`;

async function fetchWithToken(
  path: string,
  init: RequestInit | undefined,
  token: string | null,
): Promise<Response> {
  const headers = new Headers(init?.headers);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  return fetch(`${baseURL}${path}`, {
    ...init,
    headers,
    cache: "no-store",
    signal: AbortSignal.timeout(10000),
  });
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = await getValidAccessToken();
  let response: Response;
  try {
    response = await fetchWithToken(path, init, token);
    if (response.status === 401 && token) {
      const refreshed = await forceRefreshAccessToken();
      if (refreshed) {
        response = await fetchWithToken(path, init, refreshed);
      }
    }
  } catch {
    throw new Error(
      isLocalAPI
        ? "Cannot reach the local API. Check that it is running, then try again."
        : "Cannot reach the API. Check your connection, then try again.",
    );
  }
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    const appCode: string | undefined = body?.error?.code;
    const appMessage: string | undefined = body?.error?.message;
    const gatewayMessage: string | undefined = body?.message;

    if (response.status === 401) {
      const msg =
        appMessage ||
        "Authentication required. Please sign in to manage your links.";
      setAuthErrorNotice(msg);
      throw new ApiError(msg, 401, appCode || "unauthorized");
    }
    if (response.status === 403) {
      const msg =
        appMessage ||
        "Access denied. Your session does not have permission for this action.";
      throw new ApiError(msg, 403, appCode || "forbidden");
    }
    throw new ApiError(
      appMessage ||
        gatewayMessage ||
        `Request failed (${response.status}). Please try again.`,
      response.status,
      appCode || (response.status === 409 ? "conflict" : "request_failed"),
    );
  }
  if (response.status === 204) {
    return undefined as T;
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
  get: (code: string) =>
    request<ShortLink>(`/api/links/${encodeURIComponent(code)}`),
  setEnabled: (code: string, enabled: boolean, version = 1) =>
    request<ShortLink>(`/api/links/${encodeURIComponent(code)}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled, version }),
    }),
  update: (
    code: string,
    input: {
      version: number;
      url?: string;
      enabled?: boolean;
      expiresAt?: string | null;
    },
  ) =>
    request<ShortLink>(`/api/links/${encodeURIComponent(code)}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    }),
  remove: (code: string, version?: number) =>
    request<void>(
      `/api/links/${encodeURIComponent(code)}${version ? `?version=${encodeURIComponent(String(version))}` : ""}`,
      {
        method: "DELETE",
      },
    ),
};
