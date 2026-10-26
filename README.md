# Shortly — local URL shortener

Phase 01 of a URL Shortener & Analytics Platform: create short links, follow redirects, and manage links locally. The Next.js UI exports to static files; the Go API stores links in DynamoDB Local. Business logic is independent of HTTP, DynamoDB, and Lambda event types.

**Local, single-user development only. There is no authentication or ownership enforcement. Implement both before exposing management APIs publicly.** Local API and database configuration reject non-loopback endpoints. No AWS infrastructure or credentials are needed.

## Start locally

Prerequisites: Go **1.26+**, Node.js **24 LTS**, pnpm **10.33.2**, Docker with a running Linux container engine. GNU Make is optional. Run commands from the repository root.

```sh
pnpm install --frozen-lockfile
go -C services/lambda mod download
docker compose -f local/compose.yaml up -d
go -C services/lambda run ./cmd/initdb
```

The initializer waits up to 60 seconds for DynamoDB, creates the table/index if missing, and validates the existing schema without replacing the table. It is safe to rerun.

Start the API in one terminal:

```sh
go -C services/lambda run ./cmd/api
```

Start the frontend in another:

```sh
pnpm dev
```

Open **http://localhost:3000**. The API is **http://127.0.0.1:8080**, with liveness at `/health`. The form accepts a destination, an optional custom alias, and an optional expiration. The library at `/dashboard/` lists links, copies their short URLs, and enables/disables redirects.

Use `localhost:3000` for the frontend: CORS permits that exact origin by default. Both Go commands run directly on the host; only DynamoDB runs in Docker. On Windows, start Docker Desktop first. All commands above work in PowerShell as written.

## Static export

```sh
pnpm build
node local/serve-static.mjs
```

This previews `apps/web/out/` at port 3000 without a Next.js server. Stop `pnpm dev` first. Client components fetch the Go API in the browser. There are no API Routes, Server Actions, runtime SSR, runtime image optimization, or dynamically generated server pages. The local preview helper serves only build output and is not a production web server.

## Development commands

| Make target                   | Purpose                                                      |
| ----------------------------- | ------------------------------------------------------------ |
| `make install`                | Install locked frontend dependencies and download Go modules |
| `make db-up` / `make db-down` | Start/stop DynamoDB; retain the named data volume            |
| `make db-init`                | Wait for DynamoDB and initialize/validate the table          |
| `make api` / `make web`       | Start Go API / Next.js development server                    |
| `make preview`                | Serve an existing static export                              |
| `make fmt`                    | Apply gofmt and Prettier                                     |
| `make test`                   | Go unit tests and static preview server tests                |
| `make test-integration`       | Go tests including real DynamoDB Local integration           |
| `make lint`                   | Go vet, ESLint, TypeScript checks                            |
| `make build`                  | Build Go packages and frontend static export                 |
| `make check`                  | Unit tests, lint, typecheck, builds, format check            |
| `make test-e2e`               | Browser tests against the real API and exported frontend     |

Equivalent commands without Make:

```sh
go -C services/lambda test ./...
go -C services/lambda vet ./...
go -C services/lambda build ./...
node --test local/serve-static.test.mjs
pnpm lint
pnpm typecheck
pnpm build
pnpm format:check
```

Database integration requires DynamoDB running. The test creates a uniquely named temporary table and deletes only that test table. The regular test command skips this integration test unless enabled:

```powershell
# PowerShell
$env:DYNAMODB_INTEGRATION = '1'
go -C services/lambda test ./... -count=1
Remove-Item Env:DYNAMODB_INTEGRATION
```

```sh
# POSIX shell
DYNAMODB_INTEGRATION=1 go -C services/lambda test ./... -count=1
```

For browser tests, keep DynamoDB and the Go API running with defaults. Stop any server using frontend port 3000. Playwright starts its own static preview and checks desktop and mobile layouts:

```sh
pnpm exec playwright install chromium
pnpm build
pnpm test:e2e
```

These tests create identifiable `e2e-...` links and short-lived generated links in the configured local API table. They do not delete existing development links. Screenshots and failure traces go into ignored `test-results/`.

## API

All management requests use JSON. Codes and aliases are case-sensitive. Generated codes have eight URL-safe characters (48 cryptographically random bits). Custom aliases must match `[A-Za-z0-9_-]{3,32}`. Atomic conditional writes prevent overwrite; generated collisions retry five times.

| Method | Endpoint                | Behavior                                                             |
| ------ | ----------------------- | -------------------------------------------------------------------- |
| GET    | `/health`               | 200 liveness response; does not test database readiness              |
| POST   | `/api/links`            | Create; 201 with link JSON and relative `Location` header            |
| GET    | `/api/links`            | Newest-first page of up to 25 links                                  |
| GET    | `/api/links?cursor=...` | Next page using returned `nextCursor`                                |
| GET    | `/api/links/{code}`     | Link metadata, including disabled/expired links                      |
| PATCH  | `/api/links/{code}`     | Update `enabled` using `{ "enabled": false }` or `true`              |
| GET    | `/r/{code}`             | 302 with destination in `Location`; no server-side destination fetch |

GET endpoints also accept HEAD. OPTIONS supports browser preflight. Requests from an unapproved Origin return 403. Unsupported methods return 405 JSON with `Allow`; unknown routes return 404 JSON.

Create body:

```json
{
  "url": "https://example.com/articles?from=shortly#intro",
  "alias": "my-article",
  "expiresAt": "2099-12-31T00:00:00Z"
}
```

`alias` and `expiresAt` are optional. Expiry must be a future RFC 3339 timestamp with an explicit timezone. The browser converts its local date/time input to UTC. URLs must be absolute HTTP(S), at most 2048 bytes, and contain no credentials, whitespace, or control characters. JSON bodies are limited to 16 KiB, require `Content-Type: application/json`, and reject unknown fields and trailing values.

Successful create/get/update response:

```json
{
  "code": "my-article",
  "url": "https://example.com/articles?from=shortly#intro",
  "enabled": true,
  "createdAt": "2026-10-25T00:00:00Z",
  "expiresAt": "2099-12-31T00:00:00Z"
}
```

List response: `{ "links": [...], "nextCursor": "..." }`. `nextCursor` is omitted at the end; an empty list is `[]`. Treat cursors as opaque. Metadata is returned separately from the short URL; clients construct `${API_BASE_URL}/r/${code}`.

Errors have a consistent shape:

```json
{ "error": { "code": "not_found", "message": "link not found" } }
```

| Status | Error code / condition                                                   |
| ------ | ------------------------------------------------------------------------ |
| 400    | `invalid_input` or `invalid_json`; includes malformed code/cursor        |
| 403    | `forbidden_origin`                                                       |
| 404    | `not_found`                                                              |
| 405    | `method_not_allowed`                                                     |
| 409    | `conflict`; custom alias already used                                    |
| 410    | `gone`; disabled or expired redirect                                     |
| 413    | `body_too_large`                                                         |
| 415    | `unsupported_media_type`                                                 |
| 500    | `internal_error`; unexpected persistence failure, details logged locally |
| 503    | `unavailable`; random-code collision retries exhausted                   |

Redirects and API responses send `Cache-Control: no-store`. Expiry is checked on every redirect, including the exact expiry boundary. Disabled/expired metadata stays available for management. Re-enabling an expired link does not extend its expiration.

## Environment

Defaults run without creating any environment files. Examples: [`apps/web/.env.example`](apps/web/.env.example), [`services/lambda/.env.example`](services/lambda/.env.example).

| Variable                   | Default                 | Notes                                               |
| -------------------------- | ----------------------- | --------------------------------------------------- |
| `API_ADDRESS`              | `127.0.0.1:8080`        | Go local API; loopback hosts only                   |
| `WEB_ORIGIN`               | `http://localhost:3000` | Exact frontend CORS origin; loopback only           |
| `DYNAMODB_ENDPOINT`        | `http://127.0.0.1:8000` | Loopback HTTP(S) origin only                        |
| `DYNAMODB_TABLE`           | `shortlinks-local`      | Shared by initdb and API; valid DynamoDB table name |
| `NEXT_PUBLIC_API_BASE_URL` | `http://127.0.0.1:8080` | Public, embedded at frontend build time             |
| `DYNAMODB_INTEGRATION`     | unset                   | Set to `1` to run database integration tests        |

For the frontend, copy the example to `apps/web/.env.local` to override its API base URL, then restart/rebuild. Go intentionally does **not** load dotenv files: export values in the shell launching each Go command. For example, `$env:DYNAMODB_TABLE = 'shortlinks-custom'` in PowerShell or `export DYNAMODB_TABLE=shortlinks-custom` on POSIX. Use the same overrides for initialization and the API.

The **local entrypoints** configure AWS SDK v2 directly with region `us-east-1` and fixed dummy `local` credentials. They never load AWS profiles, environment credentials, or instance metadata. Do not add real credentials for local development. The separate deployed Lambda bootstrap uses execution-role credentials; see [AWS development](docs/aws-development.md).

## Repository

```text
apps/web/                         Next.js static frontend and shadcn/ui controls
services/lambda/
  cmd/api/                       Local Go HTTP entrypoint
  cmd/initdb/                    Idempotent local table initializer
  cmd/{create-link,manage-links,resolve-link}/ Native Lambda entrypoints
  internal/domain/               Link entity and application error categories
  internal/application/          Create, retrieve, resolve, list, enable/disable
  internal/ports/                Persistence contract
  internal/adapters/httpapi/     HTTP routes, JSON, status codes, CORS
  internal/adapters/dynamostore/ DynamoDB storage, pagination, table setup
  internal/localconfig/          Local-only configuration and SDK client
  internal/awsconfig/            Lambda execution-role configuration
  internal/adapters/lambdaapi/    Native API Gateway v2 transport
local/                           Docker Compose and static export preview
tests/e2e/                       Real API/browser integration tests
tests/dev/                       Strict Mode response-race regression
infra/terraform/{bootstrap,dev}/ Offline-validated AWS definitions
.github/workflows/ci.yml         Credential-free CI; no deployment
docs/                            Architecture, phase plan and verification notes
Makefile                         Common host development commands
pnpm-lock.yaml                   Reproducible frontend/tooling dependencies
```

See [architecture and request flows](docs/architecture.md), [Phase 01 verification](docs/verification.md), [Phase 02A verification](docs/phase-02-verification.md) and [Phase 02 completion review](docs/phase-02-completion.md). The [project roadmap based on the 21-part series](tasks/plan.md), [implementation checklist](tasks/todo.md), and [series review](docs/series-review.md) describe the remaining phases. Phase 02 is complete; the frontend is live at [npt-shortenlink.dev](https://npt-shortenlink.dev), backed by CloudFront/OAC and the Sydney AWS API.

## Limitations and next steps

- Authentication, ownership checks, URL editing/deletion, analytics, WebSockets, moderation, production operations and deployment are deferred to their phases. Links are validated for syntax, not safety or reputation. Private/localhost destinations are permitted; the server never visits them.
- List reads use an eventually consistent index. Newly created or updated records can briefly lag; refresh to reconcile. Redirect lookups use strongly consistent reads. The list count describes loaded records, not a total or click count.
- Data persists in the Compose named volume across `down`/`up`. Removing that volume deletes local links. No reset/delete-data command is supplied.
- The local collection index uses a single partition, suitable for this MVP. Partitioning, user ownership indexes, throttling, abuse protection, and production sizing must be addressed before a public rollout.
- ESLint 9 is retained for compatibility with Next.js's current React/accessibility/import plugins; pnpm reports its deprecation. Upgrade when those plugins support ESLint 10 together.

Phase 02 offline commands: `make build-lambdas test-packaging terraform-check` and `pnpm test:dev`. See [AWS development](docs/aws-development.md) for requirements and port conflicts. The intended domain is **npt-shortenlink.dev**; old AWS resources were removed and are not reused automatically.

For later deployments, see [deploy again](docs/aws-redeploy.md). CloudFront is already in Terraform state; no import is needed. `pnpm deploy:web` checks the Terraform account/bucket/distribution, builds for the site origin, uploads assets before HTML and requests invalidation. `pnpm test:deploy` checks its failure paths without AWS access.

**Phase 02B is complete as of 2026-10-26** in account `654822777701`, Sydney (`ap-southeast-2`), with a 4 USD monthly tracking budget. Full cloud smoke passed **29/29**, HTTP/assets **22/22**, and [GitHub Linux CI passed](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/40258572818), including Go 1.26.10 + `govulncheck`, race/integration, packaging, Terraform and 10+1 browser tests. Custom-domain HTTPS, CloudFront/OAC, security response headers and private S3 are verified; final Terraform plan has no drift. See [completion evidence](docs/phase-02-completion.md) and [safe dev pause/resume](docs/aws-dev-pause-resume.md). The budget is not a hard spending cap and has no notification recipient. Cloud management still requires AWS IAM signatures until Phase 03 adds Cognito; the exported browser UI's management 403 is expected. Local UI management continues to work.
