import "./helpers/register-tsx.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { act, createElement, Fragment, StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { JSDOM } from "jsdom";
import { AUTH_STORAGE_KEYS, createPkceTransaction, getAccessToken } from "../src/lib/auth.ts";

const { AuthCallback } = await import("../src/components/AuthCallback.tsx");
const { Navbar } = await import("../src/components/Navbar.tsx");

for (const scenario of [
  { name: "Navbar and callback complete one token exchange", strict: false },
  { name: "Navbar and callback complete one token exchange under StrictMode effect replay", strict: true },
  { name: "StrictMode callback rejects the wrong state without exchanging a code", strict: true, wrongState: true },
  { name: "StrictMode callback rejects the wrong nonce without leaving the callback", strict: true, wrongNonce: true },
]) {
  test(scenario.name, async (t) => {
    const dom = new JSDOM("<div id='root'></div>", { url: "https://shortn.dev/auth/callback/" });
    const storage = dom.window.sessionStorage;
    const transaction = await createPkceTransaction(storage, "https://shortn.dev/auth/callback/");
    const location = {
      href: dom.window.location.href,
      origin: dom.window.location.origin,
      pathname: "/auth/callback/",
      search: `?${new URLSearchParams({ code: "returned-code", state: scenario.wrongState ? "wrong-state" : transaction.state })}`,
      replace: (url) => redirects.push(url),
    };
    const redirects = [];
    const redactions = [];
    const history = { replaceState: (_state, _title, url) => { redactions.push(url); location.search = ""; } };
    let exchanges = 0;
    let releaseResponse;
    const responseReady = new Promise((resolve) => { releaseResponse = resolve; });
    const globals = { window: globalThis.window, self: globalThis.self, document: globalThis.document, fetch: globalThis.fetch, IS_REACT_ACT_ENVIRONMENT: globalThis.IS_REACT_ACT_ENVIRONMENT };
    const environment = { ...process.env };
    Object.assign(process.env, {
      NEXT_PUBLIC_COGNITO_DOMAIN: "https://auth-shortn.auth.ap-southeast-1.amazoncognito.com",
      NEXT_PUBLIC_COGNITO_CLIENT_ID: "client-123",
      NEXT_PUBLIC_AUTH_REDIRECT_URI: transaction.redirectUri,
      NEXT_PUBLIC_AUTH_LOGOUT_URI: "https://shortn.dev/",
    });
    globalThis.window = new Proxy(dom.window, { get: (target, key) => key === "location" ? location : key === "history" ? history : ["addEventListener", "removeEventListener"].includes(key) ? target[key].bind(target) : Reflect.get(target, key, target) });
    globalThis.document = dom.window.document;
    globalThis.self = globalThis.window;
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    globalThis.fetch = async (url, options) => {
      exchanges += 1;
      assert.equal(url, "https://auth-shortn.auth.ap-southeast-1.amazoncognito.com/oauth2/token");
      assert.equal(options.method, "POST");
      assert.equal(options.body.get("code"), "returned-code");
      assert.equal(options.body.get("code_verifier"), transaction.verifier);
      assert.equal(options.body.get("redirect_uri"), transaction.redirectUri);
      await responseReady;
      return new Response(JSON.stringify({
        access_token: "fixture-access-token",
        expires_in: 900,
        id_token: `header.${Buffer.from(JSON.stringify({ nonce: scenario.wrongNonce ? "wrong-nonce" : transaction.nonce })).toString("base64url")}.signature`,
      }), { status: 200 });
    };
    const root = createRoot(document.getElementById("root"));
    t.after(async () => {
      releaseResponse();
      await act(async () => root.unmount());
      Object.assign(globalThis, globals);
      process.env = environment;
      dom.window.close();
    });

    await act(async () => root.render(createElement(scenario.strict ? StrictMode : Fragment, null, createElement(Navbar), createElement(AuthCallback))));
    assert.equal(exchanges, scenario.wrongState ? 0 : 1);
    assert.equal(storage.getItem(AUTH_STORAGE_KEYS.pkce) !== null, !scenario.wrongState);
    await act(async () => { releaseResponse(); await responseReady; });

    assert.deepEqual(redactions, ["/auth/callback/"]);
    assert.equal(storage.getItem(AUTH_STORAGE_KEYS.pkce), null);
    if (scenario.wrongState || scenario.wrongNonce) {
      assert.deepEqual(redirects, []);
      assert.equal(getAccessToken(storage), null);
      assert.match(document.body.textContent, scenario.wrongState ? /state mismatch/ : /nonce mismatch/);
    } else {
      assert.deepEqual(redirects, ["/"]);
      assert.equal(getAccessToken(storage), "fixture-access-token");
      assert.doesNotMatch(document.body.textContent, /state mismatch|missing code|Chưa thể đăng nhập/);
    }
  });
}
