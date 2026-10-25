import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { mkdtemp, readFile, readdir, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { inflateRawSync } from "node:zlib";
import test from "node:test";
import { buildLambdas } from "./build-lambdas.mjs";

test("Lambda archives contain one executable Linux arm64 bootstrap and repeatable digests", async () => {
  const directory = await mkdtemp(join(tmpdir(), "shortly-lambdas-"));
  try {
    const first = await buildLambdas(directory);
    assert.deepEqual((await readdir(directory)).sort(), [
      "create-link.zip",
      "manage-links.zip",
      "manifest.json",
      "resolve-link.zip",
    ]);
    for (const name of ["create-link", "manage-links", "resolve-link"]) {
      const zip = await readFile(join(directory, `${name}.zip`));
      assert.equal(createHash("sha256").update(zip).digest("hex"), first[name]);
      const end = zip.length - 22;
      assert.equal(zip.readUInt32LE(end), 0x06054b50);
      assert.equal(zip.readUInt16LE(end + 10), 1);
      const central = zip.readUInt32LE(end + 16);
      assert.equal(zip.readUInt32LE(central), 0x02014b50);
      assert.equal(zip[central + 5], 3, "ZIP must advertise Unix permissions");
      assert.equal((zip.readUInt32LE(central + 38) >>> 16) & 0o111, 0o111);
      const nameLength = zip.readUInt16LE(central + 28);
      assert.equal(
        zip.subarray(central + 46, central + 46 + nameLength).toString(),
        "bootstrap",
      );
      const local = zip.readUInt32LE(central + 42);
      const dataStart =
        local +
        30 +
        zip.readUInt16LE(local + 26) +
        zip.readUInt16LE(local + 28);
      const data = zip.subarray(
        dataStart,
        dataStart + zip.readUInt32LE(central + 20),
      );
      const elf =
        zip.readUInt16LE(central + 10) === 8 ? inflateRawSync(data) : data;
      assert.equal(elf.subarray(0, 4).toString(), "\x7fELF");
      assert.equal(elf[4], 2, "64-bit ELF");
      assert.equal(elf[5], 1, "little-endian ELF");
      assert.equal(elf.readUInt16LE(18), 183, "AArch64 machine");
    }
    assert.deepEqual(
      await buildLambdas(directory),
      first,
      "identical sources must yield identical ZIPs",
    );
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});
