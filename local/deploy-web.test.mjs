import assert from "node:assert/strict";
import test from "node:test";
import { deployWeb } from "./deploy-web.mjs";

function fixture(
  overrides = {},
  failAt = "",
  distributionOverrides = {},
  clientOverrides = {},
) {
  process.env.npm_execpath ||= "pnpm.cjs";
  const calls = [];
  const values = {
    site_url: "https://short.example",
    api_url: "https://api.example",
    web_bucket: "shortly-dev-web-test",
    distribution_id: "E123456789TEST",
    aws_account_id: "123456789012",
    aws_region: "ap-southeast-2",
    cognito_client_id: "cognitoclient123",
    cognito_user_pool_id: "ap-southeast-2_test123",
    cognito_domain_url:
      "https://shortly-dev.auth.ap-southeast-2.amazoncognito.com",
    cognito_issuer:
      "https://cognito-idp.ap-southeast-2.amazonaws.com/ap-southeast-2_test123",
    ...overrides,
  };
  const execute = (command, args, options) => {
    calls.push({ command, args, options });
    if (failAt && args.includes(failAt))
      throw new Error("simulated command failure");
    if (command === "terraform") {
      return JSON.stringify(
        Object.fromEntries(
          Object.entries(values).map(([key, value]) => [key, { value }]),
        ),
      );
    }
    if (args.includes("get-caller-identity"))
      return JSON.stringify({ Account: "123456789012" });
    if (args.includes("describe-user-pool-client")) {
      return JSON.stringify({
        UserPoolClient: {
          UserPoolId: "ap-southeast-2_test123",
          ClientId: "cognitoclient123",
          CallbackURLs: ["https://short.example/auth/callback/"],
          LogoutURLs: ["https://short.example/"],
          ...clientOverrides,
        },
      });
    }
    if (args.includes("get-distribution")) {
      return JSON.stringify({
        Distribution: {
          ARN: "arn:aws:cloudfront::123456789012:distribution/E123456789TEST",
          Status: "Deployed",
          DomainName: "d123.cloudfront.net",
          DistributionConfig: {
            Enabled: true,
            Aliases: { Items: ["short.example"] },
            Origins: {
              Items: [
                {
                  DomainName:
                    "shortly-dev-web-test.s3.ap-southeast-2.amazonaws.com",
                },
                { DomainName: "api.example" },
              ],
            },
          },
          ...distributionOverrides,
        },
      });
    }
    return "";
  };
  return { execute, calls };
}

test("deploy verifies ownership, builds with the site origin, uploads assets before HTML and invalidates last", () => {
  const { execute, calls } = fixture();
  deployWeb(execute);
  const build = calls.find((call) => call.command === process.execPath);
  assert.equal(
    build.options.env.NEXT_PUBLIC_API_BASE_URL,
    "https://short.example",
  );
  assert.equal(
    build.options.env.NEXT_PUBLIC_COGNITO_CLIENT_ID,
    "cognitoclient123",
  );
  assert.equal(
    build.options.env.NEXT_PUBLIC_COGNITO_DOMAIN,
    "https://shortly-dev.auth.ap-southeast-2.amazoncognito.com",
  );
  assert.equal(
    build.options.env.NEXT_PUBLIC_COGNITO_ISSUER,
    "https://cognito-idp.ap-southeast-2.amazonaws.com/ap-southeast-2_test123",
  );
  const ownerCheck = calls.findIndex((call) =>
    call.args.includes("head-bucket"),
  );
  assert.ok(ownerCheck > 0 && ownerCheck < calls.indexOf(build));
  assert.ok(calls[ownerCheck].args.includes("--expected-bucket-owner"));
  const uploads = calls.filter((call) => call.args.includes("sync"));
  assert.equal(uploads.length, 3);
  assert.ok(uploads[0].args.includes("public,max-age=31536000,immutable"));
  assert.ok(uploads[1].args.includes("*.html"));
  assert.ok(uploads[2].args.includes("text/html"));
  assert.ok(
    uploads.every(
      (call) =>
        !call.args.includes("--delete") &&
        call.args.includes("--no-follow-symlinks"),
    ),
  );
  assert.ok(calls.at(-1).args.includes("create-invalidation"));
});

test("missing hosting outputs, wrong identity, invalid URLs and mismatched origins stop before build or writes", () => {
  for (const overrides of [
    { distribution_id: null },
    { aws_account_id: "999999999999" },
    { site_url: "http://short.example" },
    { site_url: "https://short.example/extra" },
    { web_bucket: "another-project-bucket" },
    { api_url: "https://wrong-api.example" },
    { site_url: "https://another.example" },
    { web_bucket: "s3://invalid-bucket" },
    { distribution_id: "--invalid-id" },
    { aws_region: "invalid-region" },
    { cognito_client_id: "" },
    { cognito_user_pool_id: "" },
    { cognito_domain_url: "http://shortly-dev.auth.example" },
    { cognito_issuer: "http://cognito-idp.example/pool" },
  ]) {
    const { execute, calls } = fixture(overrides);
    assert.throws(() => deployWeb(execute));
    assert.ok(
      calls.every(
        (call) =>
          call.command !== process.execPath &&
          !call.args.includes("sync") &&
          !call.args.includes("create-invalidation"),
      ),
    );
  }
});

test("a disabled or still-deploying distribution stops before build or writes", () => {
  for (const state of [
    { Status: "InProgress" },
    { DistributionConfig: { Enabled: false } },
  ]) {
    const { execute, calls } = fixture({}, "", state);
    assert.throws(() => deployWeb(execute), /CloudFront is not deployed/);
    assert.ok(
      calls.every(
        (call) =>
          call.command !== process.execPath && !call.args.includes("sync"),
      ),
    );
  }
});

test("a bucket permission failure stops before build and a build failure never uploads", () => {
  for (const failAt of ["head-bucket", "describe-user-pool-client", "build"]) {
    const { execute, calls } = fixture({}, failAt);
    assert.throws(() => deployWeb(execute), /simulated command failure/);
    assert.ok(
      calls.every(
        (call) =>
          !call.args.includes("sync") &&
          !call.args.includes("create-invalidation"),
      ),
    );
  }
});

test("Cognito callback/logout mismatches stop before build or uploads", () => {
  for (const client of [
    { CallbackURLs: ["https://another.example/auth/callback/"] },
    { LogoutURLs: ["https://another.example/"] },
    { CallbackURLs: undefined },
    { UserPoolId: "ap-southeast-2_other" },
    { ClientId: "another-client" },
  ]) {
    const { execute, calls } = fixture({}, "", {}, client);
    assert.throws(() => deployWeb(execute), /Cognito callback\/logout URLs/);
    assert.ok(
      calls.every(
        (call) =>
          call.command !== process.execPath &&
          !call.args.includes("sync") &&
          !call.args.includes("create-invalidation"),
      ),
    );
  }
});

test("an upload failure stops before invalidation", () => {
  const { execute, calls } = fixture({}, "sync");
  assert.throws(() => deployWeb(execute), /simulated command failure/);
  assert.equal(calls.filter((call) => call.args.includes("sync")).length, 1);
  assert.ok(calls.every((call) => !call.args.includes("create-invalidation")));
});
