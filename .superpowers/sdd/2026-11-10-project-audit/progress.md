# SDD ledger — plan: docs/superpowers/plans/2026-11-10-project-audit.md

Pre-flight: shared interfaces — F01 produces owner-scoped WebSocket identity consumed by F07/F09/F10; F02 metric names consumed by observability/deploy checks; F04 URL canonicalization consumed by F05 and config validation; F08 session manager consumed by F18/F20/F22; F12 clickId contract consumed by F13/F14/F23. Preserve these interfaces while sequencing tasks.

Task 1: complete — F01 realtime tickets/owner-scoped WebSocket. Tests: go test -race ./...; go test -tags=integration ./tools/local-http -run '^$'; frontend test/lint/typecheck; terraform fmt/validate/test (21/21). Commit pending.
Task 2: complete — F02 handled resolve 5xx emits ResolveFailed and canary alarms include it. Verified go httpapi test and Terraform observability tests (2/2).
Task 3: complete — F03 removes wildcard API child-resource fallback and adds regression coverage. Verified bootstrap tests (24/24).
Task 4: complete — F04 rejects legacy decimal/hex/octal IPv4 forms, mapped/private addresses, and URL credentials. Verified domain tests and backend race suite.
Task 5: complete — F05 create starts pending moderation workflow, resolve requires active, moderation status/reason is surfaced, and workflow failures are recorded. Verified backend race suite, bootstrap/dev Terraform tests, frontend tests/lint/typecheck. Commit pending.
Task 6: complete — F06 mutation-safe dashboard snapshot merge preserves pending creates and deleted links. Frontend test suite 32/32.
Task 9: complete — F09 focus/online/30s reconciliation keeps realtime counts current after dropped pushes.
Task 10: complete — F10 jittered reconnect backoff, stable reset, focus/online recovery, and cleanup.
Task 11: complete — F11 EventBridge publication budget reduced to 250ms with best-effort redirect contract.
Task 12: complete — F12 stable clickId dedupe across envelopes and 15-day marker horizon.
Task 13: complete — F13 EventBridge target delivery DLQ, retry policy, queue policy, and alarm.
Task 16: complete — F16 CloudFront HTML security headers policy.
Task 17: complete — F17 optional custom domain/certificate aliases and canonical publish URL in Terraform/scripts.
Task 18: complete — F18 API timeout, abort, shape validation, and create idempotency key.
Task 20: complete — F20 pending delete, clipboard feedback, and full target links.
Task 21: complete — F21 CI high/critical development dependency audit gate.
Task 22: partial — F22 privacy placeholders/log fields/retention wording updated; server-side consent evidence not implemented.
Task 23: partial — F23 typed status/clickId/realtime ticket contracts added; broader API schema/OpenAPI and interface split remain.
Tasks 7, 8, 14, 15, 19: partial/deferred — larger redesigns (one owner socket/pagination, full refresh manager, hot-counter sharding, quotas, frontend release rollback) remain for a follow-up.
