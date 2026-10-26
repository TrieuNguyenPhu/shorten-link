import { createServer } from "node:http";
import { readFile, stat } from "node:fs/promises";
import { extname, resolve, sep } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const mime = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json",
  ".txt": "text/plain; charset=utf-8",
  ".svg": "image/svg+xml",
  ".ico": "image/x-icon",
  ".png": "image/png",
  ".woff2": "font/woff2",
};

export function createStaticServer(directory) {
  const root = resolve(directory);
  return createServer(async (req, res) => {
    const reply = (status, body) => {
      res.writeHead(status, { "Content-Type": "text/plain; charset=utf-8" });
      res.end(body);
    };
    if (req.method !== "GET" && req.method !== "HEAD") {
      res.setHeader("Allow", "GET, HEAD");
      reply(405, "Method not allowed");
      return;
    }
    let file;
    try {
      file = resolve(
        root,
        "." + decodeURIComponent(new URL(req.url, "http://localhost").pathname),
      );
    } catch {
      reply(400, "Bad request");
      return;
    }
    if (file !== root && !file.startsWith(root + sep)) {
      reply(403, "Forbidden");
      return;
    }
    try {
      if ((await stat(file)).isDirectory()) file = resolve(file, "index.html");
      const body = await readFile(file);
      res.writeHead(200, {
        "Content-Type": mime[extname(file)] || "application/octet-stream",
        "X-Content-Type-Options": "nosniff",
        "X-Frame-Options": "DENY",
        "Content-Security-Policy": "frame-ancestors 'none'",
      });
      res.end(req.method === "HEAD" ? undefined : body);
    } catch (err) {
      if (err.code === "ENOENT" || err.code === "ENOTDIR")
        reply(404, "Not found");
      else {
        console.error("Static preview failed:", err.message);
        reply(500, "Unable to read export");
      }
    }
  });
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  const root = fileURLToPath(new URL("../apps/web/out/", import.meta.url));
  try {
    await stat(resolve(root, "index.html"));
  } catch {
    console.error("Static export not found. Run pnpm build first.");
    process.exit(1);
  }
  const server = createStaticServer(root);
  server.on("error", (err) => {
    console.error(err.message);
    process.exitCode = 1;
  });
  server.listen(3000, "localhost", () =>
    console.log("Static export: http://localhost:3000"),
  );
}
