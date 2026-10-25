# Phase 01 verification

Verified locally during the October 24–25, 2026 implementation session. No AWS resources or real AWS credentials were used.

Environment: Windows / PowerShell, Go 1.26.5, Node.js 24.18.0, pnpm 10.33.2, Docker 29.4.3 with DynamoDB Local 3.3.0. Frontend: Next.js 16.4.3, React 19.3.1, Tailwind CSS 4.3.3.

| Executed command                                                                                       | Result                                                                                                         |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------- |
| `pnpm install --frozen-lockfile`                                                                       | Passed; lockfile up to date                                                                                    |
| `docker compose -f local/compose.yaml up -d`                                                           | Database running on loopback with a named volume                                                               |
| `go -C services/lambda run ./cmd/initdb` (twice)                                                       | Both passed; table retained                                                                                    |
| `docker compose -f local/compose.yaml restart dynamodb` followed by initialization and a metadata read | Existing link retained after container restart                                                                 |
| `go -C services/lambda fmt ./...` / `gofmt -l services/lambda`                                         | Formatted; final listing empty                                                                                 |
| `pnpm format` / `pnpm format:check`                                                                    | Passed                                                                                                         |
| `make check`                                                                                           | Passed: Go tests, preview server test, Go vet, ESLint, TypeScript, Go build, static frontend build, formatting |
| `make test-integration`                                                                                | Passed against real DynamoDB Local, including concurrent conditional writes and incompatible-schema rejection  |
| `pnpm test:e2e`                                                                                        | 10 passed across desktop Chromium and emulated mobile Chromium                                                 |
| `pnpm audit --prod`                                                                                    | No known vulnerabilities reported                                                                              |
| `git diff --check`                                                                                     | Passed                                                                                                         |

Business tests cover HTTP(S) validation, credential-bearing/invalid destinations, aliases, UTC conversion, past and exact-boundary expiration, generated-code retries/exhaustion, custom alias conflicts, disabled links, missing links, and persistence/generator failures. HTTP tests cover response/status contracts, consistent errors, redirect headers, CORS, methods/routes, body limits, malformed JSON and unsupported fields.

DynamoDB tests use their own temporary tables and verify one winner among 12 concurrent creates, strong reads, persisted updates/expiration, missing-record handling, newest-first pagination over 28 records, malformed cursors, idempotent initialization, and preserving an incompatible existing table. Test tables are removed after execution.

Browser tests exercise the statically exported application. Create/get/302/disable/410/enable/reload and generated-code expiration use the real Go API and DynamoDB. Empty/loading/network errors, pagination/deduplication, and failed-update UI states use deliberate network fixtures. Desktop/mobile screenshots and main-region accessibility snapshots were inspected; overflow checks passed. No page JavaScript errors were observed in the real create/manage flow.

Regression sensitivity was checked by temporarily changing expiration to permit redirects at the exact expiry timestamp: `TestResolveAndEnable` failed as expected. The correct comparison was restored and the suite passed. The schema-type regression test also failed before the initializer fix and passed afterward.

The static export contains `/`, `/dashboard/`, the not-found page, and an icon. Runtime Next.js server features are not required. The preview server's own test covers static routes/assets, traversal rejection, malformed encoding, missing files, methods and HEAD responses.

Remaining limits: Chromium coverage does not establish Safari/Firefox behavior; no load test, penetration test, AWS deployment, CI pipeline, authentication, analytics, or moderation was attempted. ESLint 9's installation deprecation is documented in the README; lint itself passes. Playwright emitted host color-environment warnings, which do not affect its test results. Local browser-test links remain in the development table.
