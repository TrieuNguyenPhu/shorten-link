import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import test from "node:test";

const output = new URL("../out/", import.meta.url);

test("export includes standalone product routes and error HTML pages", async () => {
  for (const name of ["index.html", "login/index.html", "auth/callback/index.html", "dashboard/index.html", "architecture/index.html", "404.html"]) {
    const html = await readFile(new URL(name, output), "utf8");
    assert.match(html, /<html[\s>]/);
    assert.match(html, /<body[\s>]/);
  }
});

test("export contains product copy without starter or password flow", async () => {
  const html = await readFile(new URL("index.html", output), "utf8");
  assert.match(html, /Shrnk|rút gọn link/i);
  assert.doesNotMatch(html, /To get started|Deploy Now|vercel\.com|next\.js\/learn/i);
  assert.doesNotMatch(html, /password|sign up|signup/i);
});

test("home references existing static assets without an image server", async () => {
  const html = await readFile(new URL("index.html", output), "utf8");
  const assets = [...html.matchAll(/<(?:script|link|img)\b[^>]*?\b(?:src|href)="([^"#]+)"/g)]
    .map((match) => match[1].replaceAll("&amp;", "&"))
    .filter((href) => href.startsWith("/") && !href.startsWith("//"));
  assert.ok(assets.some((href) => href.includes("/_next/static/") && href.endsWith(".js")));
  assert.ok(assets.some((href) => href.includes("/_next/static/") && href.endsWith(".css")));
  for (const href of assets) {
    assert.ok(!href.startsWith("/_next/image"), `Image server required: ${href}`);
    const path = new URL("." + new URL(href, "https://static.invalid").pathname, output);
    await access(fileURLToPath(path));
  }
});
