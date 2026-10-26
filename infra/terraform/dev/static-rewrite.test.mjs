import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";
import test from "node:test";

test("CloudFront rewrites exported pages while preserving API, assets and query data", async () => {
  const source = await readFile(
    new URL("./static-rewrite.js", import.meta.url),
    "utf8",
  );
  for (const [uri, expected] of [
    ["/", "/index.html"],
    ["/dashboard/", "/dashboard/index.html"],
    ["/dashboard", "/dashboard/index.html"],
    ["/_next/static/chunk.js", "/_next/static/chunk.js"],
    ["/dashboard/__next.dashboard.txt", "/dashboard/__next.dashboard.txt"],
    ["/api/links", "/api/links"],
    ["/r/abc", "/r/abc"],
    ["/health", "/health"],
    ["/icon.svg", "/icon.svg"],
  ]) {
    const event = {
      request: {
        uri,
        method: "GET",
        querystring: { cursor: { value: "abc" } },
      },
    };
    const result = runInNewContext(source + ";handler(event)", { event });
    assert.equal(result.uri, expected);
    assert.equal(result.querystring.cursor.value, "abc");
    assert.equal(result.method, "GET");
  }
});
