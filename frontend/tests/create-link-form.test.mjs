import "./helpers/register-tsx.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { act, createElement } from "react";
import { JSDOM } from "jsdom";

const { CreateLinkForm } = await import("../src/features/links/CreateLinkForm.tsx");

test("creating a link exposes the API result and supports copying it", async (t) => {
  const dom = new JSDOM("<div id='root'></div>", { url: "https://shortn.dev/" });
  const keys = ["window", "self", "document", "fetch", "IS_REACT_ACT_ENVIRONMENT"];
  const globals = Object.fromEntries(keys.map((key) => [key, globalThis[key]]));
  Object.assign(globalThis, { window: dom.window, self: dom.window, document: dom.window.document, IS_REACT_ACT_ENVIRONMENT: true });
  const copied = [];
  Object.defineProperty(window.navigator, "clipboard", { value: { writeText: async (value) => copied.push(value) } });
  const shortUrl = "https://api.shortn.dev/Abc1234";
  let calls = 0;
  globalThis.fetch = async (url, options) => {
    calls++;
    assert.equal(url, "https://api.shortn.dev/links");
    assert.equal(options.headers.Authorization, "Bearer fixture-token");
    assert.deepEqual(JSON.parse(options.body), { url: "https://example.org/article" });
    return Response.json({ code: "Abc1234", target: "https://example.org/article", shortUrl }, { status: 201 });
  };
  const { createRoot } = await import("react-dom/client");
  const root = createRoot(document.getElementById("root"));
  t.after(async () => { await act(async () => root.unmount()); Object.assign(globalThis, globals); dom.window.close(); });
  await act(async () => root.render(createElement(CreateLinkForm, { config: { apiBaseUrl: "https://api.shortn.dev" }, token: "fixture-token", onCreated: () => undefined })));
  const input = document.querySelector("input");
  const setInput = (value) => {
    Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new window.Event("input", { bubbles: true }));
  };
  await act(async () => setInput("https://example.org/article"));
  await act(async () => document.querySelector("form").dispatchEvent(new window.Event("submit", { bubbles: true, cancelable: true })));
  assert.equal(calls, 1);
  const result = document.querySelector('a[href="' + shortUrl + '"]');
  assert.ok(result, "the new short link must be visible after creation");
  assert.equal(input.value, "");
  const copyButton = [...document.querySelectorAll("button")].find((button) => button.textContent === "Copy");
  assert.ok(copyButton);
  await act(async () => copyButton.click());
  assert.deepEqual(copied, [shortUrl]);
  assert.equal(copyButton.textContent, "Đã copy");
});

test("creating a link sends a custom alias and expiry days", async (t) => {
  const dom = new JSDOM("<div id='root'></div>", { url: "https://shortn.dev/" });
  const keys = ["window", "self", "document", "fetch", "IS_REACT_ACT_ENVIRONMENT"];
  const globals = Object.fromEntries(keys.map((key) => [key, globalThis[key]]));
  Object.assign(globalThis, { window: dom.window, self: dom.window, document: dom.window.document, IS_REACT_ACT_ENVIRONMENT: true });
  let body;
  globalThis.fetch = async (_url, options) => {
    body = JSON.parse(options.body);
    return Response.json({ code: "my-link", target: body.url, shortUrl: "https://api.shortn.dev/my-link", expiresAt: 1794000000 }, { status: 201 });
  };
  const { createRoot } = await import("react-dom/client");
  const root = createRoot(document.getElementById("root"));
  t.after(async () => { await act(async () => root.unmount()); Object.assign(globalThis, globals); dom.window.close(); });
  await act(async () => root.render(createElement(CreateLinkForm, { config: { apiBaseUrl: "https://api.shortn.dev" }, token: "fixture-token", onCreated: () => undefined })));
  const setInput = (selector, value) => {
    const input = document.querySelector(selector);
    Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new window.Event("input", { bubbles: true }));
  };
  await act(async () => setInput("#target-url", "https://example.org/article"));
  await act(async () => setInput("#link-alias", "my-link"));
  await act(async () => setInput("#expiry-days", "14"));
  await act(async () => document.querySelector("form").dispatchEvent(new window.Event("submit", { bubbles: true, cancelable: true })));
  assert.deepEqual(body, { url: "https://example.org/article", alias: "my-link", expiryDays: 14 });
});

test("creating a link rejects an expiry outside the supported range", async (t) => {
  const dom = new JSDOM("<div id='root'></div>", { url: "https://shortn.dev/" });
  const keys = ["window", "self", "document", "fetch", "IS_REACT_ACT_ENVIRONMENT"];
  const globals = Object.fromEntries(keys.map((key) => [key, globalThis[key]]));
  Object.assign(globalThis, { window: dom.window, self: dom.window, document: dom.window.document, IS_REACT_ACT_ENVIRONMENT: true });
  let calls = 0;
  globalThis.fetch = async () => { calls++; return Response.json({}, { status: 500 }); };
  const { createRoot } = await import("react-dom/client");
  const root = createRoot(document.getElementById("root"));
  t.after(async () => { await act(async () => root.unmount()); Object.assign(globalThis, globals); dom.window.close(); });
  await act(async () => root.render(createElement(CreateLinkForm, { config: { apiBaseUrl: "https://api.shortn.dev" }, token: "fixture-token", onCreated: () => undefined })));
  const setInput = (selector, value) => {
    const input = document.querySelector(selector);
    Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new window.Event("input", { bubbles: true }));
  };
  await act(async () => setInput("#target-url", "https://example.org/article"));
  await act(async () => setInput("#expiry-days", "366"));
  await act(async () => document.querySelector("form").dispatchEvent(new window.Event("submit", { bubbles: true, cancelable: true })));
  assert.equal(calls, 0);
  assert.match(document.body.textContent, /1 đến 365/);
});
