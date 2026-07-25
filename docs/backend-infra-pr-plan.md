# Backend and infrastructure pull-request plan

## Why this rollout is being rebuilt

The first v2 rollout placed the Go backend in one 2,314-line commit and the
AWS infrastructure in one 620-line commit. The resulting source tree is
valid, but those commit boundaries are too large for focused review.

This plan temporarily removes only the Go backend and AWS infrastructure.
The reviewed Next.js application, OpenAPI contract, design evidence, and
legacy-source deletion remain intact. No Python/React application is restored.

Every implementation pull request must:

- represent one independently explainable capability;
- contain its implementation and directly relevant tests;
- pass every check that exists at that point in the rollout;
- document dependencies on an earlier pull request;
- avoid deployment, DNS mutation, or static-site upload;
- avoid a final aggregate pull request that bypasses the individual reviews.

## Merge discipline

The pull requests are prepared as a dependency stack so their diffs remain
small. Only the first pull request targets `main`. Each later pull request
initially targets its immediate predecessor. After a predecessor is approved
and merged, the next pull request must be retargeted to `main`, updated from
the new `origin/main`, and validated again before it can be merged.

## Backend sequence

| Order | Branch | Capability |
|---:|---|---|
| B01 | `feature/api-go-domain-model` | Go module, link states, domain errors, and application ports |
| B02 | `feature/api-create-short-link` | strict create-link validation, aliases, expiration, collision retry, and generator |
| B03 | `feature/api-memory-repository` | concurrency-safe local repository and conflict semantics |
| B04 | `feature/api-http-create-link` | health and create-link Gin endpoints with the OpenAPI error envelope |
| B05 | `feature/api-dynamodb-repository` | conditional writes, consistent reads, serialization, and repository tests |
| B06 | `feature/api-resolve-redirect` | active-link resolution and HTTP 302 behavior |
| B07 | `feature/api-link-metadata` | active, expired, and disabled metadata states |
| B08 | `feature/api-gin-lambda-runtime` | local server, Lambda HTTP API v2 adapter, configuration, and build target |
| B09 | `feature/api-security-boundaries` | CORS allowlist, request IDs, content type, body limit, and trusted-proxy policy |
| B10 | `feature/api-observability` | structured request logs and safe panic recovery |
| B11 | `test/api-concurrent-alias` | concurrent custom-alias regression coverage |
| B12 | `fix/api-lambda-public-config` | explicit Lambda public URL and CORS configuration |

## Infrastructure sequence

| Order | Branch | Capability |
|---:|---|---|
| I01 | `feature/infra-dynamodb-table` | links table, billing mode, encryption, and outputs |
| I02 | `feature/infra-lambda-http-api` | Lambda, least-privilege IAM, HTTP API routes, and logs |
| I03 | `feature/infra-static-web-edge` | private S3 origin, OAC, CloudFront distribution, and API behaviors |
| I04 | `feature/infra-domain-routing` | ACM input, Route 53 alias records, and canonical-domain output |
| I05 | `feature/infra-edge-security-observability` | response headers, access logs, throttling, retention, and concurrency controls |
| I06 | `feature/infra-dynamodb-lifecycle` | TTL and point-in-time recovery |
| I07 | `docs/infra-build-deploy-runbook` | local validation, build, publish, cache policy, and rollback instructions |

## Cross-cutting gates

After B12 and I07 are approved, separate CI and toolchain pull requests add:

1. Go formatting, vet, race tests, and reachable-vulnerability scanning.
2. SAM lint/build validation.
3. Supported Node, pnpm, Go dependency, GitHub Action, and SAM versions.

The legacy archive branch and tag are recovery references and must not be
deleted. AWS deployment remains out of scope for this rollout.
