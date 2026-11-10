# Project Audit Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close the actionable findings in `docs/audits/2026-11-07/PROJECT-AUDIT.md`, with a regression test, review, and deploy gate for each task.

**Architecture:** Preserve the current Go Lambda, Next.js static export, DynamoDB, API Gateway, EventBridge, and Terraform boundaries. Prefer narrow contract changes and shared helpers over broad rewrites; keep asynchronous analytics best-effort while making identity, state, and deployment behavior explicit.

**Tech Stack:** Go 1.26, AWS SDK for Go v2, Next.js/React/TypeScript, Terraform 1.17, GitHub Actions, AWS Lambda/API Gateway/DynamoDB/EventBridge/S3/CloudFront.

**Spec:** `docs/audits/2026-11-07/PROJECT-AUDIT.md`

## Global Constraints

- Every behavior change must have a regression test written and observed failing before implementation.
- Private operations derive identity from verified authorization context; never trust client owner fields.
- Redirects remain best-effort and must not wait on unbounded asynchronous work.
- Terraform must pass format, validate, and test checks without AWS access.
- Deployment must use the repository's saved-plan and canary gates; no direct destructive apply.

## Review Focus

- WebSocket identity, owner isolation, ticket replay/expiry, and reconnect lifecycle.
- Stale eventual-consistency snapshots and concurrent frontend mutations.
- Token expiry/401 recovery without losing form state or duplicating mutations.
- Duplicate analytics envelopes and replay beyond queue retention.
- Terraform IAM wildcard scope, delivery DLQs, headers, domain aliases, and rollback state.

### Task 1: WebSocket authorization and owner isolation (F01)

Add a short-lived, one-use WebSocket subscription ticket issued by an authenticated HTTP route; require and consume it during `$connect`, persist owner identity with the connection, and filter broadcasts by owner/code. Add backend, frontend, and Terraform contract tests.

### Task 2: Canary HTTP failure metrics (F02)

Emit a `ResolveFailed` metric for handled 5xx responses and add a CloudWatch alarm to the CodeDeploy canary alarm set. Test a 500 response with a nil Lambda error.

### Task 3: Remove conditional IAM wildcard fallback (F03)

Make bootstrap child-resource permissions empty until exact HTTP and WebSocket API IDs are pinned. Add tests for empty, partial, and exact ID sets.

### Task 4: Canonical public URL validation (F04)

Reject legacy decimal/hex/octal IPv4, ambiguous hosts, credentials, and private/mapped addresses with a shared corpus and Go tests.

### Task 5: Moderation lifecycle contract (F05)

Create links as `pending`, start the moderation workflow through an explicit integration, resolve only `active`, and expose status/reason in list/create responses. Add state-machine and service tests.

### Task 6: Mutation-safe dashboard snapshots (F06)

Normalize link state by code, preserve unobserved creates, suppress stale deletes, and ignore updates after unmount. Add jsdom lifecycle tests.

### Task 7: One owner WebSocket and paged dashboard (F07)

Use one owner-scoped socket, page link data, and reconcile once per connection recovery. Add frontend request/socket-count tests.

### Task 8: Shared session expiry and 401 recovery (F08)

Introduce a session-aware API wrapper with expiry checks and single-flight refresh/relogin behavior while preserving form input and return URL. Add auth/API tests.

### Task 9: Realtime reconciliation (F09)

Add bounded focus/online/interval reconciliation and last-updated state without retrying committed analytics. Add hook tests.

### Task 10: Jittered reconnect and recovery controls (F10)

Add jittered exponential backoff, stable-connection reset, online/focus retry, and explicit reconnect UI. Add deterministic timer tests.

### Task 11: Redirect publication budget (F11)

Bound EventBridge publication with a small configurable budget, emit dropped-publication metrics, and retain the 302 contract. Add latency/failure tests.

### Task 12: Stable click idempotency horizon (F12)

Generate stable click IDs before publish, dedupe on click ID, and retain markers beyond queue/DLQ replay horizons. Add cross-envelope replay tests and Terraform TTL assertions.

### Task 13: EventBridge delivery DLQ and alarms (F13)

Add a dedicated target DLQ, retry policy, queue policy, and delivery failure alarms/runbook. Extend Terraform tests.

### Task 14: Analytics contention measurement and controls (F14)

Add hot-code benchmark instrumentation, bounded batch/notification budgets, and operational metrics before considering counter sharding. Add load-oriented tests.

### Task 15: Owner quotas and abuse controls (F15)

Add atomic per-owner create quota/rate limits, connection caps, differentiated API throttles, and explicit 429/Retry-After responses. Add service/IaC tests.

### Task 16: HTML security headers (F16)

Configure CloudFront response headers for CSP, frame protection, nosniff, referrer policy, and HSTS with tests that preserve static build/OAuth/WebSocket behavior.

### Task 17: Custom domain as source of truth (F17)

Declare domain/certificate variables, CloudFront aliases, canonical origin, and publish-script validation. Add Terraform and script tests.

### Task 18: Cancellable, validated API client (F18)

Add timeout/AbortSignal/request IDs, response-shape validation, deliberate GET retry, and idempotency keys for create. Add frontend tests for hangs, HTML 200, 429, unmount, and lost responses.

### Task 19: Frontend release manifest and rollback (F19)

Publish versioned artifacts with a manifest and known-good pointer, enable bucket versioning, add smoke-gated promotion and rollback command, and test injected publish failures.

### Task 20: UX safety and accessibility (F20)

Add per-row delete pending/undo semantics, clipboard error feedback, session-aware CTA, full target access, and theme contrast checks.

### Task 21: Dependency/tracing maintenance (F21)

Run full dependency audits in CI, update compatible packages, document any bounded exceptions, and add an OpenTelemetry migration note for X-Ray instrumentation.

### Task 22: Privacy content and consent evidence (F22)

Replace placeholders with configured contact/retention values, align policy with access logs/TTL semantics, and persist versioned consent evidence through an authenticated endpoint.

### Task 23: API/domain contracts (F23)

Introduce typed status/event/error contracts, stable click IDs/timestamps, focused repository interfaces, and contract tests shared by Go/TypeScript/ASL.

