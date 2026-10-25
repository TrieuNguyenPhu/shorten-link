# Phase 02 execution — plan: tasks/todo.md

Scope: fix the two reproduced Phase 01 defects, then implement 02.1–02.8 offline. User authorized local corrections and Phase 02 on 2026-10-25. AWS resources were removed by the user; the old GitHub deployment is not the source of truth. Intended domain: `npt-shortenlink.dev`.

Ruling: preserve the working local MVP and local data; rebuilding from scratch means a new cloud stack, not deleting useful code or database volumes. No credentials, AWS account calls, provision, deploy, push, or DNS changes in 02A.

Ruling: repository has no initial commit and all existing files are untracked. Work on `feature/phase-02a` in the provided workspace; do not invent a baseline commit or discard user files to create a worktree.

Pre-flight interfaces: native events reuse application use cases, never adapt net/http; cloud SDK bootstrap is separate from localconfig. Artifact names/runtime/architecture must match Terraform. Cloud management stays AWS_IAM until Phase 03. CloudFront forwards API requests with caching disabled; static rewrite applies only to the static behavior. Custom domain stays optional until an existing ACM certificate/DNS are explicitly configured in 02B.

- Phase 01 fixes: complete. URL boundary tests and real Strict Mode response-race test failed before the fixes and passed afterward.
- 02.1–02.2 native Lambda / cloud config: complete. Direct use cases, native/local contract tests, body/query/path/HEAD validation, execution-role client and context propagation pass.
- 02.3 packaging: complete. Three arm64 bootstrap ZIPs, deterministic SHA-256 manifest, executable/ELF checks pass on Windows.
- 02.4–02.7 infrastructure / hosting: complete offline. Both Terraform roots validate and pass mock-provider tests; static rewrite behavior passes Node tests. AWS provider 6.70.0 GSI key_schema replaces deprecated GSI hash/range arguments.
- 02.8 CI: configuration complete and actionlint passes. No remote/initial commit; real GitHub runner execution unavailable and explicitly unverified.
- Final verification / review: complete locally; see docs/phase-02-verification.md. Independent reviewer found an incorrect API Gateway permission ARN and an invalid IPv4 edge; both fixed with fail-then-pass tests. The dashboard counter was replaced with identity tokens to avoid a ref-cleanup lint warning without suppressing rules.

Ruling: use the pinned AWS build-lambda-zip tool already supplied by aws-lambda-go instead of implementing ZIP writing. Its executable Unix mode is verified; no new packaging dependency.

Ruling: dev regression and static E2E must run sequentially with no other Next dev process for the same app. An existing dev server initially blocked both; temporarily stopped it and reran the checks successfully.

This ledger records the offline 02A work. The subsequent authorized AWS deployment is recorded in [Phase 02B progress](phase-02b-progress.md) and [deployment evidence](../docs/phase-02b-verification.md): backend deployed in Sydney; CloudFront blocked by account verification. No commits/push or local data reset were performed.
