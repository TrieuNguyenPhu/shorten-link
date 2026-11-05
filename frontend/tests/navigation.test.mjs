import "./helpers/register-tsx.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { PathnameContext } from "next/dist/shared/lib/hooks-client-context.shared-runtime.js";
import { JSDOM } from "jsdom";
import { AUTH_STORAGE_KEYS } from "../src/lib/auth.ts";

const { Navbar } = await import("../src/components/Navbar.tsx");

test("shared navigation follows the route and session without remounting", async (t) => {
  const dom = new JSDOM("<div id='root'></div>", { url: "https://shortn.dev/" });
  const globals = { window: globalThis.window, self: globalThis.self, document: globalThis.document, IS_REACT_ACT_ENVIRONMENT: globalThis.IS_REACT_ACT_ENVIRONMENT };
  Object.assign(globalThis, { window: dom.window, self: dom.window, document: dom.window.document, IS_REACT_ACT_ENVIRONMENT: true });
  const root = createRoot(document.getElementById("root"));
  t.after(async () => { await act(async () => root.unmount()); Object.assign(globalThis, globals); dom.window.close(); });
  const render = async (path) => act(async () => root.render(createElement(PathnameContext.Provider, { value: path }, createElement(Navbar))));
  const labels = () => [...document.querySelectorAll("nav a")].map((a) => a.textContent);
  await render("/");
  assert.deepEqual(labels(), ["Trang chủ", "Kiến trúc dự án"]);
  assert.equal(document.querySelector('nav [aria-current="page"]').textContent, "Trang chủ");
  window.sessionStorage.setItem(AUTH_STORAGE_KEYS.accessToken, "fixture-token");
  window.sessionStorage.setItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt, String(Date.now() + 900_000));
  await render("/dashboard/");
  assert.deepEqual(labels(), ["Trang chủ", "Dashboard", "Kiến trúc dự án"]);
  assert.equal(document.querySelector('nav [aria-current="page"]').textContent, "Dashboard");
  assert.match(document.querySelector("header").textContent, /Đăng xuất/);
  await render("/architecture");
  assert.equal(document.querySelector('nav [aria-current="page"]').textContent, "Kiến trúc dự án");
  window.sessionStorage.setItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt, "1");
  await act(async () => window.dispatchEvent(new window.Event("focus")));
  assert.deepEqual(labels(), ["Trang chủ", "Kiến trúc dự án"]);
  assert.match(document.querySelector("header").textContent, /Đăng nhập/);
});
