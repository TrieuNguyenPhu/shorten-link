# Shortener API

Go/Gin backend for `npt-shortenlink.dev`. The application core depends on small
ports; Gin, DynamoDB, clocks, and code generation are outer adapters.

## Run locally

The local default uses the in-memory repository and listens on port `8080`.

```bash
go run ./cmd/api
```

Environment variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `STORAGE_DRIVER` | `memory` locally, `dynamodb` on Lambda | `memory` or `dynamodb` |
| `LINKS_TABLE_NAME` | — | Required for DynamoDB |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | Canonical base used in `short_url` |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Comma-separated exact origins |
| `HTTP_ADDR` | `:8080` | Local listen address |
| `PORT` | `8080` | Used when `HTTP_ADDR` is unset |

The DynamoDB table uses a string partition key named `code`. Expiring links also
store `ttl` as Unix epoch seconds for DynamoDB TTL cleanup.

## HTTP API

- `POST /api/v1/links`
- `GET /api/v1/links/:code`
- `GET /link/:code` (`302 Found`)
- `GET /healthz`

Example create body:

```json
{
  "url": "https://example.com/article",
  "custom_alias": "my-article",
  "expires_in_days": 30
}
```

## Verify

```bash
go test ./...
go vet ./...
```

For an arm64 Lambda build:

```bash
GOOS=linux GOARCH=arm64 go build -o bootstrap ./cmd/api
```
