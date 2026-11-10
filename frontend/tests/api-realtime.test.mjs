import assert from "node:assert/strict";
import test from "node:test";

const { ApiError, createLink, deleteLink, listLinks } = await import("../src/lib/api.ts");
const { mergeClickCount, calculateReconnectDelay } = await import("../src/hooks/useRealtimeClicks.ts");

const config = {
  apiBaseUrl: "https://api.shortn.dev",
  websocketUrl: "wss://ws.shortn.dev/prod",
  cognitoDomain: "https://auth.shortn.dev",
  cognitoClientId: "client",
  redirectUri: "https://shortn.dev/auth/callback/",
  logoutUri: "https://shortn.dev/",
};

function response(body, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });
}

test("API client sends Cognito bearer token and maps a link page", async () => {
  const calls = [];
  const fetchImpl = async (url, options) => {
    calls.push({ url, options });
    if (url.endsWith("/links") && options.method === "POST") return response({ code: "Abc1234", target: "https://example.org", shortUrl: "https://api.shortn.dev/Abc1234" }, 201);
    return response({ links: [{ code: "Abc1234", target: "https://example.org", clicks: 3, createdAt: "2026-11-05T00:00:00.000Z" }], nextCursor: "next" });
  };

  const created = await createLink(config, "access-token", "https://example.org", fetchImpl);
  const page = await listLinks(config, "access-token", { limit: 25, cursor: "cursor-1" }, fetchImpl);

  assert.equal(created.code, "Abc1234");
  assert.equal(page.links[0].clicks, 3);
  assert.equal(calls[0].options.headers.Authorization, "Bearer access-token");
  assert.match(calls[1].url, /limit=25&cursor=cursor-1/);
});

test("API client exposes HTTP status for delete failures", async () => {
  await assert.rejects(
    deleteLink(config, "access-token", "Abc1234", async () => response({ error: "missing" }, 404)),
    (error) => error instanceof ApiError && error.status === 404,
  );
});

test("API client rejects HTML success bodies and aborts hung requests", async () => {
  await assert.rejects(
    createLink(config, "access-token", "https://example.org", async () => new Response("<html>", { status: 200 })),
    (error) => error instanceof ApiError && /invalid/i.test(error.message),
  );
  await assert.rejects(
    listLinks(config, "access-token", {}, (_url, options) => new Promise((_resolve, reject) => {
      options.signal.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")));
    })),
    (error) => error instanceof ApiError && /timed out|aborted/i.test(error.message),
  );
});

test("realtime merge never moves a click count backwards", () => {
  assert.equal(mergeClickCount(7, 4), 7);
  assert.equal(mergeClickCount(7, 9), 9);
  assert.equal(mergeClickCount(0, 0), 0);
});

test("reconnect delay applies bounded jitter to exponential backoff", () => {
  assert.equal(calculateReconnectDelay(1, () => 0), 125);
  assert.equal(calculateReconnectDelay(3, () => 1), 1000);
  assert.ok(calculateReconnectDelay(6, () => 0.5) <= 8000);
});
