# Phase 02 — cloud-ready implementation

The previous AWS resources were deleted by the owner. This repository is the new baseline; old resource IDs/state/deployment configuration were not imported. The intended domain is **npt-shortenlink.dev**. The user approved Phase 02B in account **654822777701**, Sydney **ap-southeast-2**, budget target **4 USD/month**. The backend is deployed and verified; CloudFront is blocked by AWS account verification. See [current deployment evidence](phase-02b-verification.md) and [resumption steps](aws-support-cloudfront.md) before running deployment commands.

## Implemented in 02A

- Native API Gateway payload-v2 adapters call the existing application service directly. `cmd/create-link`, `cmd/manage-links` and `cmd/resolve-link` start the Lambda runtime, never an HTTP server.
- `internal/awsconfig` requires region, table and temporary execution-role credentials injected by Lambda. It ignores local profiles, metadata and custom endpoint variables. The SDK client/service are initialized once per execution environment; each invocation passes its context to storage. Local entrypoints still use loopback endpoints and dummy credentials.
- Three deterministic Linux/arm64 ZIPs contain executable `bootstrap` binaries for `provided.al2023`. The AWS packaging tool is pinned through go.mod. A SHA-256 manifest accompanies the archives.
- Terraform defines state bootstrap, DynamoDB, scoped execution roles, published Lambda versions with `live` aliases, HTTP API, JSON logs/5xx metrics, private S3/OAC, CloudFront and static route rewriting.
- CI runs offline quality gates and local integration/browser tests without an AWS role or secrets. Actions use immutable commit references and the checkout token is not persisted.

No Cognito, ownership, analytics, moderation, OIDC deployment or CodeDeploy rollout is implemented in this phase.

## Local verification

Prerequisites: Go 1.26.5, Node 24, pnpm 10.33.2, Terraform 1.15.5 and Docker. AWS provider 6.70.0 is pinned in both roots and lockfiles. Provider downloads need the public registry, not AWS credentials.

```sh
pnpm install --frozen-lockfile
make db-up db-init
make test-integration lint
make build-lambdas test-packaging
node --test infra/terraform/dev/static-rewrite.test.mjs
make terraform-check
pnpm test:dev
pnpm build
```

`terraform-check` runs fmt, `init -backend=false`, validate and tests with a **mock AWS provider**. The test's plan operation never contacts AWS or writes remote state. It checks that management requires IAM, alias permissions are route scoped, redirect cannot write data, S3 is private, API caching is disabled and no static rewrite is attached to API behaviors. These assertions do not prove real IAM, DNS or CloudFront behavior.

`pnpm test:dev` starts its own dev server on port 3001 to exercise React Strict Mode. Stop another Next dev server for this same application first: Next holds a per-app dev lock even on different ports. Static E2E runs separately on port 3000 against a running local Go API/DynamoDB (`make api` in another terminal, then `pnpm test:e2e`).

## Routes and temporary access model

| Routes                             | Integration       | Authorization         |
| ---------------------------------- | ----------------- | --------------------- |
| POST `/api/links`                  | create-link:live  | AWS_IAM               |
| GET/HEAD `/api/links`              | manage-links:live | AWS_IAM               |
| GET/HEAD/PATCH `/api/links/{code}` | manage-links:live | AWS_IAM               |
| GET/HEAD `/r/{code}`               | resolve-link:live | Public                |
| GET/HEAD `/health`                 | manage-links:live | Public; liveness only |

Application errors preserve the local JSON envelope and 302/400/404/409/410/413/415/500/503 behavior. API Gateway owns authorization failures, unknown routes/methods, throttling and CORS; its error bodies can differ. HEAD responses have no body. Lambda responses always disable caching. Malformed base64 and bodies over 16 KiB are rejected before use cases run.

The browser client does not sign AWS requests. **Cloud management from the UI will return 403 until Phase 03 adds Cognito/JWT.** Local UI management remains functional. For developer SigV4 calls in 02B, use Terraform's direct `api_url` output, with an IAM principal granted only the required `execute-api:Invoke` routes. Do not sign a request for the CloudFront hostname and expect that signature to survive its Host rewrite. CloudFront forwards Authorization, cookies and query strings for future JWT integration; this does not itself provide authentication.

## Fresh AWS deployment procedure

These are the fresh-stack steps, already executed through backend deployment for the current account. Preserve existing state/resources and follow the resumption steps above for the current partial stack. Sydney supersedes the original Singapore proposal.

1. Confirm the AWS account, region (current default `ap-southeast-2`), budget and deployment identity. Inventory any retained resources/state before reusing names. An old GitHub workflow or state file is not proof a resource still exists. Do not run the old deployment workflow blindly.
2. Choose a new globally unique state bucket. Review and apply the separate `infra/terraform/bootstrap` root using local state. Back up its state securely; bootstrap stays outside the application's lifecycle. The bucket has versioning, AES256 encryption, public blocking and a TLS-only policy; `prevent_destroy` protects it.
3. Copy `dev/backend.hcl.example` to an ignored `dev/backend.tfbackend`, set the new bucket, and initialize dev with `-backend-config=backend.tfbackend`. It uses S3 native `use_lockfile`; no DynamoDB lock table. Grant deployment identity Get/List/Put access only to the application's state key/prefix, plus Get/Put/Delete on the matching `.tflock` object. Retain bootstrap state separately; do not overwrite it with the app state.
4. Build ZIPs with `pnpm build:lambdas`. Copy `dev.tfvars.example` to ignored `dev.tfvars`. Initially leave `site_domain` empty to use the assigned CloudFront hostname; set `web_origin` to the exact intended browser origin before enabling browser auth in Phase 03. Review a saved Terraform plan, especially create/replace/delete operations, before applying.
5. For `npt-shortenlink.dev`, explicitly set `site_domain` and an **issued existing ACM certificate in us-east-1** covering that hostname. Verify ownership and Route 53/external DNS before changing records. Neither certificate issuance nor DNS mutation is automatic here. The old domain may still point to deleted resources.
6. Rebuild frontend with `NEXT_PUBLIC_API_BASE_URL` set to the HTTPS site origin (for example `https://npt-shortenlink.dev`). This is a build-time value; the local default must never be published as the cloud API URL. Upload assets before HTML, retain old versioned assets for rollback, and set HTML cache headers appropriately. Terraform provisions hosting; it does not upload the frontend.
7. Smoke-test `/`, a direct `/dashboard/` load, static assets and `/health`; verify direct S3 access is denied. Use a SigV4 developer call at `api_url` to create/manage; unsigned management must fail. Verify public redirect 302, disabled/expired 410 and missing 404 through both API and CloudFront, without cached redirects.
8. Inspect gateway access logs and Lambda logs using `gatewayRequestId`/`lambdaRequestId`. Application 5xx responses increment `Shortly/<environment>` failure metrics through log filters; Lambda's built-in Errors metric alone does not count a returned HTTP 500. Alarms and operational runbooks are finalized in Phase 06.

DynamoDB deletion protection is on, `prevent_destroy` is set, and PITR is enabled for `prod`. State/web buckets do not force-delete objects; both have versioning. There is no automatic destroy or local-data reset command. Terraform currently owns the `live` alias; explicitly revise that ownership when CodeDeploy is introduced in Phase 06. Do not deploy this Phase 02 dev root as a production system.

## Deployment smoke command

For a reversible dev outage without deleting data/state, follow [pause and resume](aws-dev-pause-resume.md). This pauses execution; it does not guarantee zero AWS charges.

After the dev stack and static frontend are deployed, run the following from the repository root, substituting the actual Terraform outputs:

```sh
go -C services/lambda run ./cmd/smoke -api https://API_ID.execute-api.REGION.amazonaws.com -site https://DISTRIBUTION.cloudfront.net -bucket WEB_BUCKET -region REGION -profile default
```

The command resolves the existing AWS CLI profile in memory and signs management requests using the SDK's SigV4 signer. It never prints credentials or follows redirects to destinations. It checks the exported home/dashboard, API/CloudFront health, unsigned management denial, signed create/get/list/update, redirects before and after disable/re-enable/expiry, invalid URL rejection, missing links and private S3 access denial. JSON output records check names and HTTP statuses. It leaves two `smoke-*` fixtures: one disabled, one expired; no deletion endpoint exists yet.

This is a live-cloud smoke command and is deliberately excluded from credential-free CI. Its local transport/signing/redirect checks run with `go test ./cmd/smoke` against an in-memory test server.

Omit `-site` for backend-only verification while CloudFront is unavailable. This explicitly skips frontend/CloudFront delivery checks and cannot satisfy the full Phase 02 acceptance criteria. The API and private-S3 checks still run.

## Sources checked during implementation

- [Go Lambda ZIP deployment and bootstrap naming](https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html).
- [Lambda runtime environment variables and execution-role credentials](https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html).
- [HTTP API payload v2 format](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-develop-integrations-lambda.html).
- [CloudFront AllViewerExceptHostHeader](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-origin-request-policies.html).
- [Terraform mock providers](https://developer.hashicorp.com/terraform/language/tests/mocking) and [S3 native state locking](https://developer.hashicorp.com/terraform/language/backend/s3).
- [AWS provider 6.70.0 DynamoDB schema](https://github.com/hashicorp/terraform-provider-aws/blob/v6.70.0/website/docs/r/dynamodb_table.html.markdown).
- [GitHub Actions security guidance](https://docs.github.com/en/actions/reference/security/secure-use).
