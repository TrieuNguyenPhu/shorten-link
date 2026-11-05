export type PublicConfig = {
  apiBaseUrl: string;
  websocketUrl: string;
  cognitoDomain: string;
  cognitoClientId: string;
  redirectUri: string;
  logoutUri: string;
};

export type ConfigEnvironment = Partial<Record<
  | "NEXT_PUBLIC_API_BASE_URL"
  | "NEXT_PUBLIC_WS_URL"
  | "NEXT_PUBLIC_COGNITO_DOMAIN"
  | "NEXT_PUBLIC_COGNITO_CLIENT_ID"
  | "NEXT_PUBLIC_AUTH_REDIRECT_URI"
  | "NEXT_PUBLIC_AUTH_LOGOUT_URI",
  string | undefined
>>;

const requiredKeys = [
  "NEXT_PUBLIC_API_BASE_URL",
  "NEXT_PUBLIC_WS_URL",
  "NEXT_PUBLIC_COGNITO_DOMAIN",
  "NEXT_PUBLIC_COGNITO_CLIENT_ID",
  "NEXT_PUBLIC_AUTH_REDIRECT_URI",
  "NEXT_PUBLIC_AUTH_LOGOUT_URI",
] as const;

function buildRuntimeEnvironment(): ConfigEnvironment {
  return {
    NEXT_PUBLIC_API_BASE_URL: process.env.NEXT_PUBLIC_API_BASE_URL,
    NEXT_PUBLIC_WS_URL: process.env.NEXT_PUBLIC_WS_URL,
    NEXT_PUBLIC_COGNITO_DOMAIN: process.env.NEXT_PUBLIC_COGNITO_DOMAIN,
    NEXT_PUBLIC_COGNITO_CLIENT_ID: process.env.NEXT_PUBLIC_COGNITO_CLIENT_ID,
    NEXT_PUBLIC_AUTH_REDIRECT_URI: process.env.NEXT_PUBLIC_AUTH_REDIRECT_URI,
    NEXT_PUBLIC_AUTH_LOGOUT_URI: process.env.NEXT_PUBLIC_AUTH_LOGOUT_URI,
  };
}

function assertNoPlaceholder(parsed: URL, name: string): void {
  const hostname = parsed.hostname.toLowerCase();
  if (hostname === "localhost" || hostname === "127.0.0.1" || hostname === "[::1]") return;
  if (
    hostname.endsWith(".test") ||
    /(?:placeholder|example|dummy|changeme|replace-me)/i.test(`${hostname}${parsed.pathname}`)
  ) {
    throw new Error(`${name} must not use a placeholder host`);
  }
}

function parseUrl(
  value: string | undefined,
  name: string,
  protocols: readonly string[],
  path: "origin" | "callback" | "logout",
): string {
  if (!value?.trim()) throw new Error(`${name} is required`);
  let parsed: URL;
  try {
    parsed = new URL(value.trim());
  } catch {
    throw new Error(`${name} must be a valid URL`);
  }

  const local = ["localhost", "127.0.0.1", "[::1]"].includes(parsed.hostname);
  if (local ? !protocols.includes(parsed.protocol) : parsed.protocol !== protocols[0]) {
    throw new Error(local ? `${name} must use ${protocols.join(" or ")} for local development` : `${name} must use ${protocols[0]} outside local development`);
  }
  if (parsed.username || parsed.password || parsed.search || parsed.hash) {
    throw new Error(`${name} must not include credentials, query, or fragment`);
  }
  assertNoPlaceholder(parsed, name);

  if (path === "origin" && parsed.pathname !== "/") throw new Error(`${name} must be an origin URL without a path`);
  if (path === "callback" && !/^\/auth\/callback\/?$/.test(parsed.pathname)) {
    throw new Error(`${name} must point to /auth/callback/`);
  }
  if (path === "logout" && parsed.pathname !== "/") throw new Error(`${name} must point to the site root`);

  const origin = parsed.origin;
  if (path === "callback") return `${origin}/auth/callback/`;
  return origin + (path === "logout" ? "/" : "");
}

export function getPublicConfig(environment: ConfigEnvironment = buildRuntimeEnvironment()): PublicConfig {
  for (const key of requiredKeys) {
    if (!environment[key]?.trim()) throw new Error(`${key} is required`);
  }

  const clientId = environment.NEXT_PUBLIC_COGNITO_CLIENT_ID!.trim();
  if (/(?:placeholder|example|dummy|changeme|replace-me)/i.test(clientId)) {
    throw new Error("Cognito client ID is a placeholder");
  }

  return {
    apiBaseUrl: parseUrl(environment.NEXT_PUBLIC_API_BASE_URL, requiredKeys[0], ["https:", "http:"], "origin"),
    websocketUrl: parseUrl(environment.NEXT_PUBLIC_WS_URL, requiredKeys[1], ["wss:", "ws:"], "origin"),
    cognitoDomain: parseUrl(environment.NEXT_PUBLIC_COGNITO_DOMAIN, requiredKeys[2], ["https:", "http:"], "origin"),
    cognitoClientId: clientId,
    redirectUri: parseUrl(environment.NEXT_PUBLIC_AUTH_REDIRECT_URI, requiredKeys[4], ["https:", "http:"], "callback"),
    logoutUri: parseUrl(environment.NEXT_PUBLIC_AUTH_LOGOUT_URI, requiredKeys[5], ["https:", "http:"], "logout"),
  };
}
