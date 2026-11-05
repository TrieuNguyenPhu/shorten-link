import assert from "node:assert/strict";
import test from "node:test";

const { createPkceTransaction, readPkceTransaction, validateCallback, clearAuthSession, getAccessToken, AUTH_STORAGE_KEYS } =
  await import("../src/lib/auth.ts");

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

test("creates and reloads a PKCE transaction from session storage", async () => {
  const storage = new MemoryStorage();
  const transaction = await createPkceTransaction(storage, "https://example.test/auth/callback/");

  assert.match(transaction.state, /^[A-Za-z0-9_-]{32,}$/);
  assert.match(transaction.nonce, /^[A-Za-z0-9_-]{32,}$/);
  assert.match(transaction.verifier, /^[A-Za-z0-9_-]{43,}$/);
  assert.match(transaction.challenge, /^[A-Za-z0-9_-]{43,}$/);
  assert.equal(transaction.redirectUri, "https://example.test/auth/callback/");
  assert.equal(readPkceTransaction(storage)?.state, transaction.state);
});

test("validates the callback state and code exactly once", () => {
  const storage = new MemoryStorage();
  storage.setItem(
    AUTH_STORAGE_KEYS.pkce,
    JSON.stringify({ state: "state-1", nonce: "nonce-1", verifier: "verifier-1", challenge: "challenge-1", redirectUri: "https://example.test/auth/callback/" }),
  );

  assert.deepEqual(validateCallback(storage, new URLSearchParams("code=abc&state=state-1")), {
    code: "abc",
    state: "state-1",
  });
  assert.throws(() => validateCallback(storage, new URLSearchParams("code=abc&state=wrong")), /state/i);
  assert.throws(() => validateCallback(storage, new URLSearchParams("state=state-1")), /code/i);
});

test("clears access token and PKCE material on logout", () => {
  const storage = new MemoryStorage();
  storage.setItem(AUTH_STORAGE_KEYS.accessToken, "access-token");
  storage.setItem(AUTH_STORAGE_KEYS.pkce, "pending");

  clearAuthSession(storage);

  assert.equal(storage.getItem(AUTH_STORAGE_KEYS.accessToken), null);
  assert.equal(storage.getItem(AUTH_STORAGE_KEYS.pkce), null);
});

test("a signed-out Navbar token check preserves the pending OAuth callback transaction", async () => {
  const storage = new MemoryStorage();
  const transaction = await createPkceTransaction(storage, "https://shortn.dev/auth/callback/");

  assert.equal(getAccessToken(storage), null);

  assert.deepEqual(validateCallback(storage, new URLSearchParams({ code: "returned-code", state: transaction.state })), {
    code: "returned-code",
    state: transaction.state,
  });
  assert.equal(readPkceTransaction(storage)?.verifier, transaction.verifier);
  assert.throws(() => validateCallback(storage, new URLSearchParams({ code: "returned-code", state: "wrong-state" })), /state mismatch/);
});

test("an expired token is removed without deleting a new login transaction", async () => {
  const storage = new MemoryStorage();
  storage.setItem(AUTH_STORAGE_KEYS.accessToken, "expired-token");
  storage.setItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt, "1000");
  const transaction = await createPkceTransaction(storage, "https://shortn.dev/auth/callback/");

  assert.equal(getAccessToken(storage, 1000), null);
  assert.equal(storage.getItem(AUTH_STORAGE_KEYS.accessToken), null);
  assert.equal(storage.getItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt), null);
  assert.equal(readPkceTransaction(storage)?.state, transaction.state);
});
