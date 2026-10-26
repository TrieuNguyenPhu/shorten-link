# Shortly — Serverless URL Shortener

Phase 03 of a URL Shortener & Analytics Platform: authenticate with OAuth 2.0 Authorization Code + PKCE (`S256`), create short links, edit destinations/expirations with optimistic concurrency (`version`), enable/disable redirects, delete links with tombstone semantics, and follow public redirects locally or on AWS (`ap-southeast-2`, live at [npt-shortenlink.dev](https://npt-shortenlink.dev)). The Next.js UI exports to static files served via CloudFront + private S3 (OAC); the Go backend runs both locally and on native `provided.al2023` (`arm64`) Lambda functions backed by a DynamoDB single-table store. Business logic is independent of HTTP, DynamoDB, and Lambda event types.

**Cloud management routes are protected by Amazon Cognito + API Gateway HTTP API JWT Authorizer (`shortly-api/links.read` and `shortly-api/links.write` scopes) and enforce per-owner (`USER#<sub_id>`) isolation at the application and DynamoDB layers.** Local development runs a zero-Docker in-memory store with a local PKCE OAuth2 loopback simulator and never requires AWS credentials.

## Start locally

Prerequisites: Go **1.26+**, Node.js **24 LTS**, pnpm **10.33.2**. GNU Make is optional. Run commands from the repository root.

```sh
pnpm install --frozen-lockfile
go -C services/lambda mod download
```

Start the API in one terminal:

```sh
go -C services/lambda run ./cmd/api
```

Start the frontend in another:

```sh
pnpm dev
```

Open **http://localhost:3000**. The API is **http://127.0.0.1:8080**, with liveness at `/health`. Click **Sign in** to complete the local PKCE OAuth 2.0 flow. The form accepts a destination, an optional custom alias, and an optional expiration. The library at `/dashboard/` lists your links, edits destinations/expirations, enables/disables redirects, and permanently deletes links with confirmation.

## Static export

```sh
pnpm build
node local/serve-static.mjs
```

This previews `apps/web/out/` at port 3000 without a Next.js server. Stop `pnpm dev` first. Client components fetch the Go API in the browser. There are no API Routes, Server Actions, runtime SSR, runtime image optimization, or dynamically generated server pages.

## Development commands

| Make target           | Purpose                                                      |
| --------------------- | ------------------------------------------------------------ |
| `make install`        | Install locked frontend dependencies and download Go modules |
| `make api`/`make web` | Start Go API / Next.js development server                    |
| `make preview`        | Serve an existing static export                              |
| `make fmt`            | Apply gofmt and Prettier                                     |
| `make test`           | Go unit/contract tests, static preview & deploy script tests |
| `make lint`           | Go vet, ESLint, TypeScript checks                            |
| `make build`          | Build Go packages and frontend static export                 |
| `make check`          | Unit tests, lint, typecheck, builds, format check            |
| `make test-e2e`       | Browser tests against the real API and exported frontend     |

Equivalent commands without Make:

```sh
go -C services/lambda test ./...
go -C services/lambda vet ./...
go -C services/lambda build ./...
node --test local/serve-static.test.mjs local/deploy-web.test.mjs
pnpm lint
pnpm typecheck
pnpm build
pnpm format:check
```

For browser E2E tests, start `cmd/api` and run Playwright:

```sh
pnpm exec playwright install chromium
pnpm build
pnpm test:e2e
```

## API

All management requests require a valid OAuth 2.0 `Bearer` access token (`shortly-api/links.read` for `GET`/`HEAD`, `shortly-api/links.write` for `POST`/`PATCH`/`DELETE`) and use JSON. Codes and aliases are case-sensitive. Generated codes have eight URL-safe characters (48 cryptographically random bits). Custom aliases must match `[A-Za-z0-9_-]{3,32}`. Atomic conditional writes prevent overwrite and enforce optimistic concurrency (`version`).

| Method | Endpoint                        | Scope                     | Behavior                                                              |
| ------ | ------------------------------- | ------------------------- | --------------------------------------------------------------------- |
| GET    | `/health`                       | Public                    | 200 liveness response                                                 |
| POST   | `/api/links`                    | `shortly-api/links.write` | Create; 201 with link JSON (`version: 1`) and relative `Location`     |
| GET    | `/api/links`                    | `shortly-api/links.read`  | Newest-first page of up to 25 links owned by caller (`USER#sub`)      |
| GET    | `/api/links?cursor=...`         | `shortly-api/links.read`  | Next page using returned `nextCursor` (verified against caller)       |
| GET    | `/api/links/{code}`             | `shortly-api/links.read`  | Link metadata if owned by caller; 404 otherwise                       |
| PATCH  | `/api/links/{code}`             | `shortly-api/links.write` | Update `url`, `enabled`, `expiresAt`, `clearExpiresAt` with `version` |
| DELETE | `/api/links/{code}?version=...` | `shortly-api/links.write` | 204 tombstone delete; removes from index and retires short code       |
| GET    | `/r/{code}`                     | Public                    | 302 with destination in `Location`; 410 if disabled/expired/deleted   |

| Status | Error code / condition                                                 |
| ------ | ---------------------------------------------------------------------- |
| 400    | `invalid_input` or `invalid_json`; includes malformed/tampered cursor  |
| 401    | `unauthorized`; missing, malformed, or expired Bearer access token     |
| 403    | `forbidden` or `forbidden_origin`; insufficient token scope or origin  |
| 404    | `not_found`; missing, deleted, or owned by another user                |
| 405    | `method_not_allowed`                                                   |
| 409    | `conflict` (alias in use / retired) or `version_conflict` (stale edit) |
| 410    | `gone`; disabled, expired, or deleted redirect                         |
| 413    | `body_too_large`                                                       |
| 415    | `unsupported_media_type`                                               |
| 500    | `internal_error`; unexpected persistence failure                       |
| 503    | `unavailable`; random-code collision retries exhausted                 |

## Environment

Defaults run without creating any environment files. Examples: [`apps/web/.env.example`](apps/web/.env.example), [`services/lambda/.env.example`](services/lambda/.env.example).

| Variable                        | Default                              | Notes                                        |
| ------------------------------- | ------------------------------------ | -------------------------------------------- |
| `API_ADDRESS`                   | `127.0.0.1:8080`                     | Go local API; loopback hosts only            |
| `WEB_ORIGIN`                    | `http://localhost:3000`              | Exact frontend CORS origin; loopback only    |
| `NEXT_PUBLIC_API_BASE_URL`      | `http://127.0.0.1:8080`              | Public API origin embedded at frontend build |
| `NEXT_PUBLIC_COGNITO_CLIENT_ID` | `local-web-client`                   | Public Cognito PKCE client ID                |
| `NEXT_PUBLIC_COGNITO_DOMAIN`    | `http://127.0.0.1:8080/oauth2-local` | Cognito Hosted UI base URL                   |
| `NEXT_PUBLIC_COGNITO_ISSUER`    | `http://127.0.0.1:8080/oauth2-local` | Cognito User Pool OIDC issuer URL            |

## AWS packaging & deployment (Phase 03)

Offline verification commands (no AWS credentials required):

```sh
make build-lambdas test-packaging terraform-check
pnpm test:deploy
pnpm test:dev
```

For redeployments to the `ap-southeast-2` stack:

- `pnpm build:lambdas` builds deterministic `linux/arm64` Lambda archives into `dist/lambdas/`.
- `pnpm deploy:web` verifies the Terraform account/bucket/distribution/Cognito outputs, builds the static export against `site_url` and Cognito settings, syncs immutable assets before HTML, and creates a CloudFront invalidation.
- `go -C services/lambda run ./cmd/smoke -api <api_url> -site <site_url> -bucket <web_bucket> -region ap-southeast-2 -user-pool-id <pool_id> -client-id <client_id> -cognito-domain <cognito_domain>` runs the 50-check multi-user Cognito PKCE cloud smoke suite.

See [authentication design](docs/authentication.md) and [Phase 03 completion report](docs/phase-03-completion.md).
