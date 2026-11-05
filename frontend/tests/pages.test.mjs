import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

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
  assert.doesNotMatch(`${page}\n${layout}`, /password|sign up|signup|innerHTML/i);
  assert.match(`${page}\n${layout}`, /aria-|focus|nav|main/i);
  assert.match(page, /Shrnk/);
  assert.match(page, /Source Code/);
  assert.match(page, /Kiến trúc dự án/);
  assert.doesNotMatch(page, /Xem cách hoạt động|→|My Links/);
});

test("login and callback routes exist with no password form", async () => {
  const login = await source("app/login/page.tsx");
  const button = await source("components/GoogleLoginButton.tsx");
  const callbackComponent = await source("components/AuthCallback.tsx");

  assert.match(button, /Tiếp tục với Google/);
  assert.doesNotMatch(`${login}\n${button}`, /<input|password|đăng ký|signup/i);
  assert.match(callbackComponent, /Đang đăng nhập/i);
  assert.match(callbackComponent, /history\.replaceState/);
});

test("navigation and dashboard copy support signed-in landing flow", async () => {
  const navbar = await source("components/Navbar.tsx");
  const workspace = await source("features/links/LinksWorkspace.tsx");
  const architecture = await source("app/architecture/page.tsx");

  assert.match(navbar, /getAccessToken/);
  assert.match(navbar, /LogoutButton/);
  assert.doesNotMatch(workspace, /CreateLinkForm/);
  assert.doesNotMatch(workspace, /My Links/);
  assert.match(architecture, /Kiến trúc dự án/);
});
