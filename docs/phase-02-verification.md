# Phase 02A verification — 2026-10-25

This report records the original offline checkpoint. The subsequent [AWS/CI follow-up](phase-02b-verification.md) includes a passing GitHub Linux run, live backend evidence and current hosting blockers.

Scope: the owner authorized correcting the local MVP and continuing Phase 02. The old AWS resources have been removed. Work remains local/offline; `npt-shortenlink.dev` is the intended domain, not evidence of a live deployment. No old state or GitHub deployment was imported.

## Corrections and regression evidence

1. Destination validation now rejects out-of-range ports, forbidden hostname characters and invalid numeric IPv4 quads. The new application test first failed for port 99999 and `<`/`>` hostnames, then passed after the fix. Independent review found `256.256.256.256`; tests failed for it (also with a trailing dot), then passed after IPv4 validation. Valid IPv6, IPv4 loopback, port 65535 and escaped path spaces remain accepted.
2. Link library requests use identity tokens. Only the latest load can update data/error/loading state; effect cleanup invalidates it. A real Chromium test against Next dev/Strict Mode first reproduced Disabled → Active after a delayed stale list response; it passes with the fix. No lint rule is suppressed.
3. Fresh review found a double slash in generated API Gateway invocation-permission ARNs. An exact ARN assertion failed before correcting the separator and passed afterward. Permission remains restricted to the published alias and route.

## Checks executed on this host

| Check                                                              | Result                                                                                                                                                                                          |
| ------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DYNAMODB_INTEGRATION=1 go -C services/lambda test ./... -count=1` | Pass; includes actual DynamoDB Local concurrency/pagination/schema preservation and native/local contracts                                                                                      |
| `go -C services/lambda vet ./...`, `build ./...`, gofmt            | Pass                                                                                                                                                                                            |
| Native payload-v2 tests                                            | Pass; create/get/list/update/resolve/health, collisions, disabled/expired/missing links, malformed JSON/base64/body limits, query decoding, missing path parameter, HEAD and downstream failure |
| AWS config tests                                                   | Pass with injected fake execution-role env and an in-memory HTTP transport; no profiles/metadata/AWS network                                                                                    |
| `make check`                                                       | Pass; Go/Node tests, vet, ESLint, typecheck, static build and formatting. A ref-cleanup warning from the first run was removed; subsequent lint is clean                                        |
| `pnpm build`                                                       | Pass; Next static export includes home and dashboard                                                                                                                                            |
| `pnpm test:dev`                                                    | 1/1 pass on Chromium with Strict Mode                                                                                                                                                           |
| `pnpm test:e2e`                                                    | 10/10 pass on static export, desktop/mobile, real local API/database                                                                                                                            |
| `make test-packaging build-lambdas`                                | Pass; ZIP entry count/name/Unix executable bits, 64-bit little-endian ARM ELF, SHA-256 and identical digests across two builds                                                                  |
| Static CloudFront rewrite Node test                                | Pass; directory routes, assets, Next exported payloads and API/redirect/query preservation                                                                                                      |
| `make terraform-check`                                             | Both roots fmt/init backend-disabled/validate/mock-provider tests pass                                                                                                                          |
| `pnpm audit --prod --audit-level high`                             | No known vulnerabilities reported                                                                                                                                                               |
| `actionlint v1.7.12`                                               | Pass for CI workflow (shellcheck unavailable on this Windows host)                                                                                                                              |

Two browser runs initially failed because an existing Next dev process held port 3000/the application's dev lock. The identified local dev server was temporarily stopped; the Strict Mode/build/static E2E sequence then passed. The log-color environment warning from Playwright is tooling output, not an application error.

Artifacts are in ignored `dist/lambdas/`: `create-link.zip`, `manage-links.zip`, `resolve-link.zip`, `manifest.json`. Rebuild after changing code or Go toolchain. Tests compare repeated builds on the current toolchain; they do not assert identical binaries across different Go versions.

## Review and limits

A separate read-only reviewer inspected native handlers, cloud bootstrap, packaging, Terraform, CloudFront and CI. Both actionable findings (permission ARN and IPv4 validation) were corrected with regression evidence. Final self-review retains the small application/repository boundaries and local-only safety rules.

CI configuration is present and statically validated. This local repository has no remote and no initial commit, so no workflow was pushed or run on a clean GitHub runner. Linux CI/race-detector execution remains to be confirmed there; it is not claimed as a completed run.

No AWS credentials were inspected; no AWS API, real-account plan/apply, deployment, DNS change, cloud smoke test or production load test ran. Terraform mock tests validate definitions, not real service authorization or behavior. Cloud management requires IAM and does not work from the unsigned browser client until Phase 03. Phase 02B, Cognito, analytics and operational rollout remain separate work.
