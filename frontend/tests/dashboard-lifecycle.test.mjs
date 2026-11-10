import "./helpers/register-tsx.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { JSDOM } from "jsdom";
import { AUTH_STORAGE_KEYS } from "../src/lib/auth.ts";

const { LinksWorkspace, mergeLinkSnapshot } = await import("../src/features/links/LinksWorkspace.tsx");

test("stale snapshots preserve pending creates and never restore deleted links", () => {
  const current = [
    { code: "New0001", target: "https://new.example", clicks: 0, createdAt: "2026-11-05T00:00:00Z" },
    { code: "Deleted1", target: "https://deleted.example", clicks: 0, createdAt: "2026-11-04T00:00:00Z" },
  ];
  const merged = mergeLinkSnapshot(current, [], new Set(["New0001"]), new Set(["Deleted1"]));
  assert.deepEqual(merged.links.map((link) => link.code), ["New0001"]);
  assert.deepEqual([...merged.pendingCreated], ["New0001"]);
});

test("dashboard keeps its table and sockets mounted while reconciling HTTP snapshots", async (t) => {
  const dom = new JSDOM("<div id='root'></div>", { url: "https://shortn.dev/dashboard/" });
  const keys = ["window", "self", "document", "fetch", "WebSocket", "IS_REACT_ACT_ENVIRONMENT"];
  const globals = Object.fromEntries(keys.map((key) => [key, globalThis[key]]));
  const environment = { ...process.env };
  Object.assign(process.env, {
    NEXT_PUBLIC_API_BASE_URL: "https://api.shortn.dev",
    NEXT_PUBLIC_WS_URL: "wss://ws.shortn.dev/prod",
    NEXT_PUBLIC_COGNITO_DOMAIN: "https://auth.shortn.dev",
    NEXT_PUBLIC_COGNITO_CLIENT_ID: "client",
    NEXT_PUBLIC_AUTH_REDIRECT_URI: "https://shortn.dev/auth/callback/",
    NEXT_PUBLIC_AUTH_LOGOUT_URI: "https://shortn.dev/",
  });
  Object.assign(globalThis, { window: dom.window, self: dom.window, document: dom.window.document, IS_REACT_ACT_ENVIRONMENT: true });
  window.sessionStorage.setItem(AUTH_STORAGE_KEYS.accessToken, "fixture-token");
  window.sessionStorage.setItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt, String(Date.now() + 900_000));
  const sockets = [];
  globalThis.WebSocket = class {
    constructor(url) { this.url = url; this.closed = false; sockets.push(this); }
    close() { this.closed = true; this.onclose?.(); }
  };
  const links = [{ code: "Abc1234", target: "https://example.org", clicks: 3, createdAt: "2026-11-05T00:00:00Z", realtimeTicket: "ticket-1" }];
  let calls = 0;
  let completeRefresh;
  globalThis.fetch = async (_url, options) => {
    assert.equal(options.headers.Authorization, "Bearer fixture-token");
    if (options.method === "DELETE") return Response.json({ deleted: "Abc1234" });
    calls += 1;
    if (calls === 1) return Response.json({ links });
    return new Promise((resolve) => { completeRefresh = resolve; });
  };
  const root = createRoot(document.getElementById("root"));
  t.after(async () => {
    completeRefresh?.(Response.json({ links }));
    await act(async () => root.unmount());
    Object.assign(globalThis, globals);
    process.env = environment;
    dom.window.close();
  });
  await act(async () => root.render(createElement(LinksWorkspace)));
  assert.equal(sockets.length, 1);
  const table = document.querySelector("table");
  assert.ok(table);
  await act(async () => sockets[0].onopen());
  assert.equal(calls, 2);
  assert.equal(document.querySelector("table"), table, "background reconciliation must retain the table");
  assert.equal(sockets[0].closed, false, "HTTP refresh must not disconnect realtime");
  await act(async () => sockets[0].onmessage({ data: JSON.stringify({ code: "Abc1234", clicks: 9 }) }));
  assert.equal(table.querySelector("strong").textContent, "9");
  await act(async () => completeRefresh(Response.json({ links })));
  assert.equal(table.querySelector("strong").textContent, "9", "an older snapshot must not undo a realtime count");
  assert.equal(sockets.length, 1, "snapshot completion must not recreate the socket");
  await act(async () => sockets[0].onopen());
  await act(async () => completeRefresh(Response.json({ error: "unavailable" }, { status: 503 })));
  assert.equal(document.querySelector("table"), table, "failed background refresh must retain existing data");
  assert.equal(sockets.length, 1);
  assert.match(document.body.textContent, /Dịch vụ đang gặp sự cố/);
  await act(async () => {
    sockets[0].onclose();
    await new Promise((resolve) => setTimeout(resolve, 300));
  });
  assert.equal(sockets.length, 2, "a real disconnect reconnects once");
  await act(async () => sockets[1].onopen());
  await act(async () => completeRefresh(Response.json({ links: [{ ...links[0], clicks: 12 }] })));
  assert.equal(document.querySelector("table"), table);
  assert.equal(table.querySelector("strong").textContent, "12");
  assert.equal(sockets.length, 2, "reconnect snapshot must not trigger another connection");
  await act(async () => sockets[1].onopen());
  const remove = [...document.querySelectorAll("button")].find((button) => button.textContent === "Xóa");
  await act(async () => remove.click());
  assert.ok(!document.querySelector("table"));
  await act(async () => completeRefresh(Response.json({ links })));
  assert.ok(!document.querySelector("table"), "a stale in-flight snapshot must not restore a successfully deleted link");
  assert.equal(sockets.length, 2, "a deleted link must not reconnect");
});
