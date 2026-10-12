import assert from "node:assert/strict";
import { test } from "node:test";
import { assertDeploymentPermissions, verifyLinkFlow } from "./deploy-aws.mjs";

test("organization denial prevents deployment even when other actions are allowed", () => {
  assert.throws(
    () => assertDeploymentPermissions([
      { EvalActionName: "cloudfront:CreateDistribution", EvalDecision: "allowed" },
      { EvalActionName: "lambda:CreateFunction", EvalDecision: "explicitDeny", OrganizationsDecisionDetail: { AllowedByOrganizations: false } },
    ]),
    /lambda:CreateFunction.*AWS Organizations/,
  );
});

test("deployment smoke test creates, reads and redirects a short link", async (context) => {
  const requests = [];
  const responses = [
    new Response(JSON.stringify({ data: { code: "smoke123" } }), { status: 201 }),
    new Response(JSON.stringify({ data: { code: "smoke123", status: "active" } }), { status: 200 }),
    new Response(null, { status: 302, headers: { Location: "https://example.com/" } }),
  ];
  context.mock.method(globalThis, "fetch", async (...args) => {
    requests.push(args);
    return responses.shift();
  });
  await verifyLinkFlow("https://distribution.cloudfront.net");
  assert.deepEqual(requests.map(([url]) => url), [
    "https://distribution.cloudfront.net/api/v1/links",
    "https://distribution.cloudfront.net/api/v1/links/smoke123",
    "https://distribution.cloudfront.net/link/smoke123",
  ]);
  assert.equal(JSON.parse(requests[0][1].body).expires_in_days, 1);
  assert.equal(requests[2][1].redirect, "manual");
});

test("deployment smoke test rejects a redirect to the wrong destination", async (context) => {
  const responses = [
    new Response(JSON.stringify({ data: { code: "smoke123" } }), { status: 201 }),
    new Response(JSON.stringify({ data: { code: "smoke123", status: "active" } }), { status: 200 }),
    new Response(null, { status: 302, headers: { Location: "https://wrong.example/" } }),
  ];
  context.mock.method(globalThis, "fetch", async () => responses.shift());
  await assert.rejects(verifyLinkFlow("https://distribution.cloudfront.net"), /redirect/);
});

test("allowed operations pass; malformed permission responses fail closed", () => {
  assert.doesNotThrow(() => assertDeploymentPermissions([{ EvalActionName: "lambda:CreateFunction", EvalDecision: "allowed" }]));
  assert.throws(() => assertDeploymentPermissions([]), /empty/);
  assert.throws(() => assertDeploymentPermissions([{}]), /malformed/);
  assert.throws(() => assertDeploymentPermissions([{ EvalActionName: "s3:CreateBucket", EvalDecision: "implicitDeny", OrganizationsDecisionDetail: { AllowedByOrganizations: false } }]), /s3:CreateBucket/);
});
