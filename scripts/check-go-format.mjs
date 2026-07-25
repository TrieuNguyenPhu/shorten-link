import { spawnSync } from "node:child_process";

const result = spawnSync("gofmt", ["-l", "services/shortener-api"], {
  encoding: "utf8",
});

if (result.error) {
  console.error(`Could not run gofmt: ${result.error.message}`);
  process.exit(1);
}

if (result.status !== 0) {
  process.stderr.write(result.stderr);
  process.exit(result.status ?? 1);
}

const unformatted = result.stdout.trim();
if (unformatted) {
  console.error("The following Go files are not formatted:");
  console.error(unformatted);
  console.error("Run `make format-api` and commit the result.");
  process.exit(1);
}
