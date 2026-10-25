import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtemp, mkdir, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createStaticServer } from "./serve-static.mjs";

test("serves exported routes and assets without exposing files outside the export", async () => {
  const root = await mkdtemp(join(tmpdir(), "shortly-static-"));
  const server = createStaticServer(root);
  try {
    await writeFile(join(root, "index.html"), "home");
    await mkdir(join(root, "dashboard"));
    await writeFile(join(root, "dashboard", "index.html"), "library");
    await writeFile(join(root, "style.css"), "body {}");
    await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
    const base = `http://127.0.0.1:${server.address().port}`;
    assert.equal(await (await fetch(base)).text(), "home");
    assert.equal(await (await fetch(`${base}/dashboard/`)).text(), "library");
    assert.match(
      (await fetch(`${base}/style.css`)).headers.get("content-type"),
      /text\/css/,
    );
    assert.equal((await fetch(`${base}/missing`)).status, 404);
    assert.equal((await fetch(`${base}/%2e%2e%2foutside`)).status, 403);
    assert.equal((await fetch(`${base}/%FF`)).status, 400);
    assert.equal((await fetch(base, { method: "POST" })).status, 405);
    assert.equal(await (await fetch(base, { method: "HEAD" })).text(), "");
  } finally {
    await new Promise((resolve) => server.close(resolve));
    assert.ok(root.startsWith(join(tmpdir(), "shortly-static-")));
    await rm(root, { recursive: true, force: true });
  }
});
