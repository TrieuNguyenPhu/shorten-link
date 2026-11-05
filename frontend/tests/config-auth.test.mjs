import assert from "node:assert/strict";
import test from "node:test";

const { getAuthConfig, getPublicConfig } = await import("../src/lib/config.ts");
const {
  AUTH_STORAGE_KEYS,
  PRIVACY_POLICY_VERSION,
  buildGoogleAuthorizationUrl,
  createPkceTransaction,
  exchangeCodeForToken,
  getAccessToken,
  readPrivacyConsent,
  recordPrivacyConsent,
} = await import("../src/lib/auth.ts");

class MemoryStorage {
  #values = new Map();

  getItem(key) {
    return this.#values.get(key) ?? null;
  }

  setItem(key, value) {
    this.#values.set(key, value);
  }

  removeItem(key) {
    this.#values.delete(key);
  }
}

const validEnvironment = {
  NEXT_PUBLIC_API_BASE_URL: "https://api.shortn.dev",
  NEXT_PUBLIC_WS_URL: "wss://ws.shortn.dev",
  NEXT_PUBLIC_COGNITO_DOMAIN: "https://auth-shortn.auth.ap-southeast-1.amazoncognito.com",
  NEXT_PUBLIC_COGNITO_CLIENT_ID: "client-123",
  NEXT_PUBLIC_AUTH_REDIRECT_URI: "https://shortn.dev/auth/callback",
  NEXT_PUBLIC_AUTH_LOGOUT_URI: "https://shortn.dev",
};

test("builds auth configuration without unrelated API and WebSocket endpoints", () => {
  const authEnvironment = { ...validEnvironment };
  delete authEnvironment.NEXT_PUBLIC_API_BASE_URL;
  delete authEnvironment.NEXT_PUBLIC_WS_URL;
  const config = getAuthConfig(authEnvironment);

  assert.equal(config.cognitoDomain, "https://auth-shortn.auth.ap-southeast-1.amazoncognito.com");
  assert.equal(config.cognitoClientId, "client-123");
  assert.equal(config.redirectUri, "https://shortn.dev/auth/callback/");
  assert.equal(config.logoutUri, "https://shortn.dev/");
});

test("preserves the API Gateway WebSocket stage when loading deployed configuration", () => {
  const config = getPublicConfig({
    ...validEnvironment,
    NEXT_PUBLIC_WS_URL: "wss://a1b2c3.execute-api.ap-southeast-1.amazonaws.com/prod",
  });

  assert.equal(config.websocketUrl, "wss://a1b2c3.execute-api.ap-southeast-1.amazonaws.com/prod");
  assert.equal(
    getPublicConfig({ ...validEnvironment, NEXT_PUBLIC_WS_URL: "ws://localhost:8080/realtime" }).websocketUrl,
    "ws://localhost:8080/realtime",
  );
  for (const url of ["wss://ws.shortn.dev/prod?token=secret", "wss://ws.shortn.dev/prod#fragment", "wss://user:secret@ws.shortn.dev/prod", "ws://ws.shortn.dev/prod"]) {
    assert.throws(() => getPublicConfig({ ...validEnvironment, NEXT_PUBLIC_WS_URL: url }));
  }
});

function idTokenWithNonce(nonce) {
  const payload = Buffer.from(JSON.stringify({ nonce })).toString("base64url");
  return `header.${payload}.signature`;
}

test("validates production public URLs and rejects placeholder endpoints", () => {
  const config = getPublicConfig(validEnvironment);

  assert.equal(config.apiBaseUrl, "https://api.shortn.dev");
  assert.equal(config.websocketUrl, "wss://ws.shortn.dev");
  assert.equal(config.redirectUri, "https://shortn.dev/auth/callback/");
  assert.equal(config.logoutUri, "https://shortn.dev/");

  assert.throws(
    () => getPublicConfig({ ...validEnvironment, NEXT_PUBLIC_API_BASE_URL: "https://api.example.test" }),
    /placeholder|example/i,
  );
  assert.throws(
    () => getPublicConfig({ ...validEnvironment, NEXT_PUBLIC_AUTH_REDIRECT_URI: "https://shortn.dev/login" }),
    /callback/i,
  );
  assert.throws(
    () => getPublicConfig({ ...validEnvironment, NEXT_PUBLIC_WS_URL: "https://ws.shortn.dev/path" }),
    /websocket|origin|URL/i,
  );
  assert.throws(
    () => getPublicConfig({ ...validEnvironment, NEXT_PUBLIC_WS_URL: "ws://ws.shortn.dev" }),
    /wss|local/i,
  );
  assert.throws(
    () => getPublicConfig({ ...validEnvironment, NEXT_PUBLIC_API_BASE_URL: "http://api.shortn.dev" }),
    /https|local/i,
  );
});

test("builds Google Cognito PKCE authorization URL with minimal scopes and stores consent evidence", async () => {
  const storage = new MemoryStorage();
  const config = getPublicConfig(validEnvironment);
  const consent = recordPrivacyConsent(storage, 1_762_300_000_000);
  const url = new URL(await buildGoogleAuthorizationUrl(config, storage));

  assert.equal(consent.policyVersion, PRIVACY_POLICY_VERSION);
  assert.deepEqual(consent.purposes, ["authentication"]);
  assert.equal(readPrivacyConsent(storage)?.consentedAt, new Date(1_762_300_000_000).toISOString());
  assert.equal(url.pathname, "/oauth2/authorize");
  assert.equal(url.searchParams.get("identity_provider"), "Google");
  assert.equal(url.searchParams.get("response_type"), "code");
  assert.equal(url.searchParams.get("scope"), "openid email");
  assert.equal(url.searchParams.get("code_challenge_method"), "S256");
  assert.ok(url.searchParams.get("state"));
  assert.ok(url.searchParams.get("nonce"));
  assert.ok(url.searchParams.get("code_challenge"));
  assert.ok(storage.getItem(AUTH_STORAGE_KEYS.pkce));
});

test("exchanges code only when the Cognito nonce matches and stores expiry", async () => {
  const storage = new MemoryStorage();
  const config = getPublicConfig(validEnvironment);
  const transaction = await createPkceTransaction(storage, config.redirectUri);
  const now = 1_700_000_000_000;
  const token = await exchangeCodeForToken(
    config,
    "auth-code",
    storage,
    async () =>
      new Response(
        JSON.stringify({ access_token: "cognito-access", expires_in: 3600, id_token: idTokenWithNonce(transaction.nonce) }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    now,
  );

  assert.equal(token, "cognito-access");
  assert.equal(getAccessToken(storage, now + 3_599_000), "cognito-access");
  assert.equal(getAccessToken(storage, now + 3_601_000), null);
  assert.equal(storage.getItem(AUTH_STORAGE_KEYS.pkce), null);
});

test("rejects non-success token responses and a mismatched nonce", async () => {
  const config = getPublicConfig(validEnvironment);
  const storage = new MemoryStorage();
  await createPkceTransaction(storage, config.redirectUri);

  await assert.rejects(
    exchangeCodeForToken(
      config,
      "auth-code",
      storage,
      async () => new Response(JSON.stringify({ error: "invalid_grant" }), { status: 400 }),
    ),
    /token exchange/i,
  );

  const retryTransaction = await createPkceTransaction(storage, config.redirectUri);
  await assert.rejects(
    exchangeCodeForToken(
      config,
      "auth-code",
      storage,
      async () =>
        new Response(JSON.stringify({ access_token: "wrong", expires_in: 3600, id_token: idTokenWithNonce("wrong") }), { status: 200 }),
    ),
    /nonce/i,
  );
  assert.equal(storage.getItem(AUTH_STORAGE_KEYS.accessToken), null);
  assert.equal(storage.getItem(AUTH_STORAGE_KEYS.pkce), null);
  assert.equal(retryTransaction.redirectUri, config.redirectUri);
});
