import type { AuthConfig } from "./config";

export const PRIVACY_POLICY_VERSION = "1.0";

export const AUTH_STORAGE_KEYS = {
  accessToken: "shorten-link.access-token",
  accessTokenExpiresAt: "shorten-link.access-token-expires-at",
  pkce: "shorten-link.pkce",
  privacyConsent: "shorten-link.privacy-consent",
  pendingTarget: "shorten-link.pending-target",
} as const;

export type PrivacyConsentRecord = {
  policyVersion: string;
  consentedAt: string;
  purposes: ["authentication"];
};

export type StorageLike = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
};

export type PkceTransaction = {
  state: string;
  nonce: string;
  verifier: string;
  challenge: string;
  redirectUri: string;
};

function storageAvailable(): StorageLike | null {
  return typeof window === "undefined" ? null : window.sessionStorage;
}

function encode(bytes: Uint8Array): string {
  let value = "";
  for (const byte of bytes) value += String.fromCharCode(byte);
  return btoa(value).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

function randomToken(bytes = 32): string {
  const data = new Uint8Array(bytes);
  crypto.getRandomValues(data);
  return encode(data);
}

async function sha256(value: string): Promise<string> {
  const data = new TextEncoder().encode(value);
  return encode(new Uint8Array(await crypto.subtle.digest("SHA-256", data)));
}

export async function createPkceTransaction(storage: StorageLike, redirectUri: string): Promise<PkceTransaction> {
  const verifier = randomToken(32);
  const transaction: PkceTransaction = {
    state: randomToken(),
    nonce: randomToken(),
    verifier,
    challenge: await sha256(verifier),
    redirectUri,
  };
  storage.setItem(AUTH_STORAGE_KEYS.pkce, JSON.stringify(transaction));
  return transaction;
}

export function readPkceTransaction(storage: StorageLike): PkceTransaction | null {
  const raw = storage.getItem(AUTH_STORAGE_KEYS.pkce);
  if (!raw) return null;
  try {
    const value = JSON.parse(raw) as Partial<PkceTransaction>;
    if (!value.state || !value.nonce || !value.verifier || !value.challenge || !value.redirectUri) return null;
    return value as PkceTransaction;
  } catch {
    return null;
  }
}

export function validateCallback(storage: StorageLike, params: URLSearchParams): { code: string; state: string } {
  const code = params.get("code");
  const state = params.get("state");
  const transaction = readPkceTransaction(storage);
  if (!code) throw new Error("OAuth callback is missing code");
  if (!state || !transaction || state !== transaction.state) throw new Error("OAuth callback state mismatch");
  return { code, state };
}

export function recordPrivacyConsent(
  storage: StorageLike = storageAvailable()!,
  now = Date.now(),
): PrivacyConsentRecord {
  const record: PrivacyConsentRecord = {
    policyVersion: PRIVACY_POLICY_VERSION,
    consentedAt: new Date(now).toISOString(),
    purposes: ["authentication"],
  };
  storage.setItem(AUTH_STORAGE_KEYS.privacyConsent, JSON.stringify(record));
  return record;
}

export function readPrivacyConsent(storage: StorageLike = storageAvailable()!): PrivacyConsentRecord | null {
  const raw = storage?.getItem(AUTH_STORAGE_KEYS.privacyConsent) ?? null;
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<PrivacyConsentRecord>;
    if (
      typeof parsed.policyVersion !== "string" ||
      typeof parsed.consentedAt !== "string" ||
      !Array.isArray(parsed.purposes) ||
      parsed.purposes[0] !== "authentication"
    ) {
      return null;
    }
    return parsed as PrivacyConsentRecord;
  } catch {
    return null;
  }
}

export function clearAuthSession(storage: StorageLike): void {
  storage.removeItem(AUTH_STORAGE_KEYS.accessToken);
  storage.removeItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt);
  storage.removeItem(AUTH_STORAGE_KEYS.pkce);
  storage.removeItem(AUTH_STORAGE_KEYS.privacyConsent);
}

export async function buildGoogleAuthorizationUrl(config: AuthConfig, storage: StorageLike): Promise<string> {
  const transaction = await createPkceTransaction(storage, config.redirectUri);
  const url = new URL(`${config.cognitoDomain}/oauth2/authorize`);
  url.search = new URLSearchParams({
    identity_provider: "Google",
    response_type: "code",
    client_id: config.cognitoClientId,
    redirect_uri: config.redirectUri,
    scope: "openid email",
    state: transaction.state,
    nonce: transaction.nonce,
    code_challenge_method: "S256",
    code_challenge: transaction.challenge,
  }).toString();
  return url.toString();
}

export async function startGoogleLogin(config: AuthConfig): Promise<void> {
  const storage = storageAvailable() ?? window.sessionStorage;
  recordPrivacyConsent(storage);
  const target = await buildGoogleAuthorizationUrl(config, storage);
  window.location.assign(target);
}

function decodeIdTokenNonce(idToken: string): string | null {
  const encodedPayload = idToken.split(".")[1];
  if (!encodedPayload) return null;
  try {
    const normalized = encodedPayload.replaceAll("-", "+").replaceAll("_", "/").padEnd(Math.ceil(encodedPayload.length / 4) * 4, "=");
    const payload = JSON.parse(atob(normalized)) as { nonce?: unknown };
    return typeof payload.nonce === "string" ? payload.nonce : null;
  } catch {
    return null;
  }
}

export async function exchangeCodeForToken(
  config: AuthConfig,
  code: string,
  storage: StorageLike,
  fetchImpl: typeof fetch = fetch,
  now = Date.now(),
): Promise<string> {
  const transaction = readPkceTransaction(storage);
  if (!transaction) throw new Error("PKCE transaction expired");
  const response = await fetchImpl(`${config.cognitoDomain}/oauth2/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      client_id: config.cognitoClientId,
      code,
      redirect_uri: transaction.redirectUri,
      code_verifier: transaction.verifier,
    }),
  });
  if (!response.ok) {
    storage.removeItem(AUTH_STORAGE_KEYS.pkce);
    throw new Error("Cognito token exchange failed");
  }
  let payload: { access_token?: string; expires_in?: number; id_token?: string };
  try {
    payload = (await response.json()) as typeof payload;
  } catch {
    storage.removeItem(AUTH_STORAGE_KEYS.pkce);
    throw new Error("Cognito token response was invalid");
  }
  if (!payload.access_token) {
    storage.removeItem(AUTH_STORAGE_KEYS.pkce);
    throw new Error("Cognito response did not include an access token");
  }
  const expiresIn = Number(payload.expires_in);
  if (!payload.id_token || decodeIdTokenNonce(payload.id_token) !== transaction.nonce) {
    storage.removeItem(AUTH_STORAGE_KEYS.pkce);
    throw new Error("Cognito id token nonce mismatch");
  }
  if (!Number.isFinite(expiresIn) || expiresIn <= 0) {
    storage.removeItem(AUTH_STORAGE_KEYS.pkce);
    throw new Error("Cognito response did not include a valid expiry");
  }
  storage.setItem(AUTH_STORAGE_KEYS.accessToken, payload.access_token);
  storage.setItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt, String(now + expiresIn * 1000));
  storage.removeItem(AUTH_STORAGE_KEYS.pkce);
  return payload.access_token;
}

export function getAccessToken(storage: StorageLike = storageAvailable()!, now = Date.now()): string | null {
  const token = storage?.getItem(AUTH_STORAGE_KEYS.accessToken) ?? null;
  const expiresAt = Number(storage?.getItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt));
  if (!token || !Number.isFinite(expiresAt) || now >= expiresAt) {
    if (storage) {
      storage.removeItem(AUTH_STORAGE_KEYS.accessToken);
      storage.removeItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt);
    }
    return null;
  }
  return token;
}

export function logout(config: AuthConfig, storage: StorageLike = storageAvailable()!): void {
  clearAuthSession(storage);
  const url = new URL(`${config.cognitoDomain}/logout`);
  url.search = new URLSearchParams({ client_id: config.cognitoClientId, logout_uri: config.logoutUri }).toString();
  window.location.assign(url.toString());
}
