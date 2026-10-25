import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const repository = fileURLToPath(new URL("../", import.meta.url));

function go(args, env = process.env) {
  const result = spawnSync("go", args, {
    cwd: join(repository, "services/lambda"),
    env,
    encoding: "utf8",
  });
  if (result.error || result.status !== 0) {
    throw new Error(
      result.error?.message || result.stderr || "Go build failed",
    );
  }
}

export async function buildLambdas(
  directory = join(repository, "dist/lambdas"),
) {
  const output = resolve(directory);
  await mkdir(output, { recursive: true });
  const scratch = await mkdtemp(join(tmpdir(), "shortly-bootstrap-"));
  const manifest = {};
  try {
    for (const name of ["create-link", "manage-links", "resolve-link"]) {
      const bootstrap = join(scratch, "bootstrap");
      go(
        [
          "build",
          "-trimpath",
          "-buildvcs=false",
          "-tags=lambda.norpc",
          "-ldflags=-s -w -buildid=",
          "-o",
          bootstrap,
          `./cmd/${name}`,
        ],
        {
          ...process.env,
          GOOS: "linux",
          GOARCH: "arm64",
          CGO_ENABLED: "0",
        },
      );
      const archive = join(output, `${name}.zip`);
      // AWS's pinned packaging tool preserves executable permissions on Windows too.
      go([
        "run",
        "github.com/aws/aws-lambda-go/cmd/build-lambda-zip",
        "-o",
        archive,
        bootstrap,
      ]);
      manifest[name] = createHash("sha256")
        .update(await readFile(archive))
        .digest("hex");
    }
    await writeFile(
      join(output, "manifest.json"),
      JSON.stringify(manifest, null, 2) + "\n",
    );
    return manifest;
  } finally {
    await rm(scratch, { recursive: true, force: true });
  }
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  try {
    console.log(await buildLambdas());
  } catch (err) {
    console.error(err.message);
    process.exitCode = 1;
  }
}
