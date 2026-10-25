# Phase 01 implementation plan

**Goal:** Deliver and verify only the local MVP described in [the design](phase-01-design.md).

**Architecture:** Static Next.js client → Go HTTP adapter → application use cases → repository port → DynamoDB Local. Future Lambda transports reuse the application layer.

**Stack:** Go standard HTTP library, AWS SDK for Go v2, Next.js App Router, TypeScript, Tailwind CSS, shadcn/ui, pnpm, Docker Compose, Playwright.

## Scope and verification

- [x] Backend: create `services/lambda/{cmd,internal}` for domain, application, repository port, HTTP and DynamoDB adapters, local configuration, API entrypoint and initialization tool. Write and run business tests before implementation, then HTTP and database integration tests. Verify with `go test ./...`, `go vet ./...`, and `go build ./...`.
- [x] Local runtime: create `local/compose.yaml`, environment examples and root Makefile. Start DynamoDB Local; run initialization twice to prove idempotency. Retain data across ordinary restarts.
- [x] Frontend: create pnpm workspace and `apps/web`, static export config, shadcn controls, home form and dashboard. Write browser acceptance tests, implement API client and interactions, then run lint, typecheck and static build.
- [x] Integration: exercise create → retrieve → 302 → disable → 410 → enable, duplicate aliases, pagination, invalid input, unknown routes/methods, unavailable services, and mobile overflow. Use an isolated Playwright browser against the exported site and real API/database.
- [x] Documentation and review: document setup on Windows and POSIX, API contracts, environment, Mermaid architecture, limitations and phases 02–06. Recheck all acceptance requirements, run formatting and the full relevant checks, report actual results and remaining limits. Stop at Phase 01.

All files are new: no existing callers or data migrations. Data safety concerns are failed conditional writes, accidental table deletion, stale redirects, and accidental public management access. Initialization must preserve existing tables; local entrypoints must reject non-loopback operation. No Terraform, paid services, AWS credential discovery, fake authentication, or future-phase placeholders.
