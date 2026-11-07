import "./helpers/register-tsx.mjs";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { JSDOM } from "jsdom";

async function source(path) {
  return readFile(new URL(`../src/${path}`, import.meta.url), "utf8");
}

test("landing page has product copy and one Google login entry point", async () => {
  const page = await source("app/page.tsx");
  const navbar = await source("components/Navbar.tsx");
  const layout = await source("app/layout.tsx");

  assert.match(page, /short link|rút gọn link/i);
  assert.match(page, /realtime|theo dõi lượt click/i);
  assert.match(`${page}\n${navbar}`, /\/login\//);
  assert.doesNotMatch(page, /To get started|Deploy Now|Documentation|vercel\.com/i);
  assert.doesNotMatch(page, /your\.link\/hello|>24</i);
  assert.doesNotMatch(`${page}\n${layout}`, /password|sign up|signup/i);
  assert.match(`${page}\n${layout}`, /aria-|focus|nav|main/i);
  assert.match(page, /Shrnk/);
  assert.match(page, /Source Code/);
  assert.match(page, /className="sketch-button" href="\/login\/">Đăng nhập/);
  assert.match(page, /className="sketch-button sketch-button-secondary" href="https:\/\/github\.com\/TrieuNguyenPhu\/shorten-link"[^>]*>Source Code/);
  assert.doesNotMatch(page, /Xem cách hoạt động|→|My Links/);
});

test("login and callback routes exist with no password form and require explicit privacy consent", async () => {
  const login = await source("app/login/page.tsx");
  const button = await source("components/GoogleLoginButton.tsx");
  const callbackComponent = await source("components/AuthCallback.tsx");
  const footer = await source("components/Footer.tsx");
  const privacy = await source("app/privacy/page.tsx");

  assert.match(button, /Tiếp tục với Google/);
  assert.doesNotMatch(`${login}\n${button}`, /type=["'](?:password|email|text)|password|đăng ký|signup/i);
  assert.match(button, /type="checkbox"/);
  assert.doesNotMatch(button, /defaultChecked/);
  assert.match(button, /\/privacy\//);
  assert.match(footer, /\/privacy\//);
  assert.match(privacy, /Chính sách bảo vệ dữ liệu cá nhân/);
  assert.match(privacy, /91\/2025\/QH15/);
  assert.match(privacy, /356\/2025\/NĐ-CP/);
  assert.match(privacy, /2 ngày làm việc/);
  assert.match(privacy, /Singapore/);
  assert.match(callbackComponent, /Đang đăng nhập/i);
  assert.match(callbackComponent, /history\.replaceState/);
});

test("navigation and dashboard copy support signed-in landing flow", async () => {
  const navbar = await source("components/Navbar.tsx");
  const workspace = await source("features/links/LinksWorkspace.tsx");

  assert.match(navbar, /getAccessToken/);
  assert.match(navbar, /LogoutButton/);
  assert.match(workspace, /CreateLinkForm/);
  assert.doesNotMatch(workspace, /My Links/);
});

test("GoogleLoginButton requires explicit unchecked consent before starting OAuth redirect", async (t) => {
  const { GoogleLoginButton } = await import("../src/components/GoogleLoginButton.tsx");
  const { readPrivacyConsent } = await import("../src/lib/auth.ts");
  const dom = new JSDOM("<div id='root'></div>", { url: "https://shortn.dev/login/" });
  const assigned = [];
  const location = {
    href: dom.window.location.href,
    origin: dom.window.location.origin,
    pathname: "/login/",
    assign: (url) => assigned.push(url),
  };
  const globals = { window: globalThis.window, self: globalThis.self, document: globalThis.document, IS_REACT_ACT_ENVIRONMENT: globalThis.IS_REACT_ACT_ENVIRONMENT };
  const environment = { ...process.env };
  Object.assign(process.env, {
    NEXT_PUBLIC_COGNITO_DOMAIN: "https://auth-shortn.auth.ap-southeast-1.amazoncognito.com",
    NEXT_PUBLIC_COGNITO_CLIENT_ID: "client-123",
    NEXT_PUBLIC_AUTH_REDIRECT_URI: "https://shortn.dev/auth/callback/",
    NEXT_PUBLIC_AUTH_LOGOUT_URI: "https://shortn.dev/",
  });
  globalThis.window = new Proxy(dom.window, { get: (target, key) => key === "location" ? location : Reflect.get(target, key, target) });
  globalThis.document = dom.window.document;
  globalThis.self = globalThis.window;
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  const root = createRoot(document.getElementById("root"));
  t.after(async () => {
    await act(async () => root.unmount());
    Object.assign(globalThis, globals);
    process.env = environment;
    dom.window.close();
  });

  await act(async () => root.render(createElement(GoogleLoginButton)));
  const checkbox = document.querySelector('input[type="checkbox"]');
  const button = document.querySelector("button");
  assert.ok(checkbox);
  assert.equal(checkbox.checked, false);

  await act(async () => button.click());
  assert.deepEqual(assigned, []);
  assert.equal(readPrivacyConsent(dom.window.sessionStorage), null);
  assert.match(document.body.textContent, /Chính sách bảo vệ dữ liệu cá nhân/);

  await act(async () => checkbox.click());
  assert.equal(checkbox.checked, true);

  await act(async () => button.click());
  assert.equal(assigned.length, 1);
  assert.match(assigned[0], /scope=openid\+email/);
  assert.equal(readPrivacyConsent(dom.window.sessionStorage)?.policyVersion, "1.0");
});

test("HomeCreateLink degrades gracefully without throwing if public config is missing", async (t) => {
  const { HomeCreateLink } = await import("../src/components/HomeCreateLink.tsx");
  const { AUTH_STORAGE_KEYS } = await import("../src/lib/auth.ts");
  const dom = new JSDOM("<div id='root'></div>", { url: "https://shortn.dev/" });
  const globals = { window: globalThis.window, self: globalThis.self, document: globalThis.document, IS_REACT_ACT_ENVIRONMENT: globalThis.IS_REACT_ACT_ENVIRONMENT };
  const environment = { ...process.env };
  delete process.env.NEXT_PUBLIC_API_BASE_URL;
  Object.assign(globalThis, { window: dom.window, self: dom.window, document: dom.window.document, IS_REACT_ACT_ENVIRONMENT: true });
  dom.window.sessionStorage.setItem(AUTH_STORAGE_KEYS.accessToken, "token-123");
  dom.window.sessionStorage.setItem(AUTH_STORAGE_KEYS.accessTokenExpiresAt, String(Date.now() + 60_000));
  const root = createRoot(document.getElementById("root"));
  t.after(async () => {
    await act(async () => root.unmount());
    Object.assign(globalThis, globals);
    process.env = environment;
    dom.window.close();
  });

  await act(async () => root.render(createElement(HomeCreateLink)));
  assert.equal(document.getElementById("create-link"), null);
});
