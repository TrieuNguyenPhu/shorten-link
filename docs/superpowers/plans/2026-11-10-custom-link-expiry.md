# Custom link names and expiry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let authenticated clients choose an optional safe custom slug and an optional 1–365 day lifetime, while preserving generated permanent links.

**Architecture:** The API accepts `alias` and `expiryDays`; the domain validates the alias and expiry, the service computes an optional Unix expiry timestamp, and DynamoDB stores it in the existing `ttl` attribute while exposing it as `expiresAt`. Resolution checks expiry synchronously before redirecting, and the frontend sends/validates the new fields and displays the expiry.

**Tech Stack:** Go 1.26, AWS Lambda/API Gateway, DynamoDB single-table repository, Next.js 16/React 19/TypeScript, Node test runner, Terraform-managed DynamoDB TTL.

**Spec:** `docs/superpowers/specs/2026-11-10-custom-link-expiry-design.md`

## Global Constraints

- Alias is trimmed, optional, and must match `[A-Za-z0-9-]{3,32}`; it cannot start/end with `-`, contain `--`, or use the reserved route name `links`.
- `expiryDays` is optional; omitted/blank means permanent, otherwise it is an integer from 1 through 365.
- Existing requests containing only `url` and existing stored links remain valid.
- Expiry is enforced by the resolve service even if DynamoDB TTL cleanup is delayed.
- Collision writes must never overwrite an existing link and return HTTP 409.
- Use TDD: each behavior gets a failing test before production code.

## Review Focus

- Alias collisions and reserved route names must not overwrite or shadow `/links`; cover repository conditional failure and HTTP 409.
- Expired links must return 404 at resolve time even when the record still exists; cover boundary timestamps with an injected clock.
- Permanent links must omit the TTL/expiry field and continue resolving; cover the old `url`-only request.
- Invalid alias characters/length and expiry values outside 1–365 must return 400 before storage; cover both API and service validation.
- List/create responses and optimistic frontend state must carry the optional expiry without breaking links that omit it; cover response validators and form submission.

## File Map

- Modify `backend/internal/domain/models.go`: add optional expiry metadata to `Link`.
- Modify `backend/internal/domain/shortcode.go` and `shortcode_test.go`: alias and expiry validation helpers/tests.
- Modify `backend/internal/service/link_service.go` and `link_service_test.go`: accept alias/expiry inputs, calculate expiry, choose generated/custom code, and enforce expiration.
- Modify `backend/internal/httpapi/handler.go` and `handler_test.go`: parse new request fields, map validation/collision errors, and serialize `expiresAt`.
- Modify `backend/internal/repository/dynamodb.go` and `repository_test.go`: preserve TTL mapping and recognize custom-code collisions.
- Modify `frontend/src/lib/api.ts`: extend request/response types and `createLink` payload.
- Modify `frontend/src/features/links/CreateLinkForm.tsx` and `frontend/tests/create-link-form.test.mjs`: add alias/expiry controls and client validation.
- Modify `frontend/src/features/links/LinksWorkspace.tsx` and `frontend/src/features/links/LinksTable.tsx`: retain and display expiry metadata.
- Update relevant API/frontend tests and README API contract after implementation.

### Task 1: Domain validation and link model

**Files:**
- Modify: `backend/internal/domain/models.go`
- Modify: `backend/internal/domain/shortcode.go`
- Test: `backend/internal/domain/shortcode_test.go`

**Interfaces:**
- Produces `ValidateAlias(string) (string, error)` and `ValidateExpiryDays(int) error` for service/handler use.
- `domain.Link.ExpiresAt int64` is optional (`json:"expiresAt,omitempty"`, DynamoDB `ttl,omitempty`).

- [ ] **Step 1: Write failing tests** for accepted aliases (`abc`, `my-link-2`, length 32), blank alias, rejected aliases (length <3/>32, leading/trailing/consecutive hyphens, punctuation, `links`), accepted expiry days 1 and 365, and rejected 0/366/negative values.
- [ ] **Step 2: Run `go -C backend test ./internal/domain -run 'TestValidateAlias|TestValidateExpiryDays' -v` and confirm the new tests fail because helpers/model behavior is absent.
- [ ] **Step 3: Implement the validators and optional model field with canonical trimming/lowercase-preserving behavior (aliases remain case-sensitive for lookup, but reserved-name comparison is case-insensitive).**
- [ ] **Step 4: Run the focused domain tests and confirm they pass.

### Task 2: Service creation and expiry enforcement

**Files:**
- Modify: `backend/internal/service/link_service.go`
- Test: `backend/internal/service/link_service_test.go`

**Interfaces:**
- Replace the internal creation signature with `Create(ctx context.Context, owner, rawURL, alias string, expiryDays *int) (domain.Link, error)`.
- Add service errors for invalid alias/expiry and alias collision as needed; retain generated-code collision retry behavior.

- [ ] **Step 1: Write failing tests** for custom code persistence, generated code when alias is blank, computed `ExpiresAt` from an injected clock, permanent links with zero expiry, invalid alias/expiry rejection before repository writes, alias collision propagation, and resolve returning not-found for expired links while accepting an unexpired boundary.
- [ ] **Step 2: Run `go -C backend test ./internal/service -run 'TestCreate|TestResolve' -v` and confirm the new cases fail for the missing signature/behavior.
- [ ] **Step 3: Implement minimal service behavior: validate inputs, choose alias or generator, calculate `now + days*24h`, retry only generated-code collisions, and reject expired links before status/target redirect.
- [ ] **Step 4: Run the focused service tests and confirm they pass.

### Task 3: HTTP API and DynamoDB contract

**Files:**
- Modify: `backend/internal/httpapi/handler.go`
- Modify: `backend/internal/repository/dynamodb.go`
- Test: `backend/internal/httpapi/handler_test.go`
- Test: `backend/internal/repository/repository_test.go`

**Interfaces:**
- POST `/links` accepts `{url:string, alias?:string, expiryDays?:number|null}` and returns `code`, `shortUrl`, `target`, `status`, and optional `expiresAt`.
- Invalid values return 400; conditional alias collision returns 409; old `{url}` remains 201.

- [ ] **Step 1: Write failing handler/repository tests** for parsing/validation, 400 responses, 409 collision, response expiry serialization, old request compatibility, and DynamoDB `ttl` item mapping.
- [ ] **Step 2: Run `go -C backend test ./internal/httpapi ./internal/repository -run 'TestCreate|Test.*TTL|Test.*Collision' -v` and confirm failure.
- [ ] **Step 3: Implement request decoding and error mapping, pass parsed values to the service, add `ErrAliasTaken` handling, and ensure repository writes the optional `ttl` attribute without emitting it for permanent links.
- [ ] **Step 4: Run focused backend tests and confirm they pass.

### Task 4: Frontend API and creation form

**Files:**
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/features/links/CreateLinkForm.tsx`
- Test: `frontend/tests/create-link-form.test.mjs`

**Interfaces:**
- `createLink(config, token, target, options?: { alias?: string; expiryDays?: number })` sends optional fields only when supplied.
- `CreatedLink` and `Link` expose optional `expiresAt`.

- [ ] **Step 1: Write failing frontend tests** for rendering alias/expiry controls, sending `{url, alias, expiryDays}`, blank expiry omission, client-side range/alias errors, and preserving the created result/copy flow.
- [ ] **Step 2: Run `corepack pnpm@10.33.2 --dir frontend test -- create-link-form.test.mjs` and confirm failure.
- [ ] **Step 3: Implement controlled inputs, Vietnamese labels/help text, client validation matching backend limits, and the expanded API payload/response guards.
- [ ] **Step 4: Run the focused frontend test and typecheck.

### Task 5: Dashboard display and documentation

**Files:**
- Modify: `frontend/src/features/links/LinksWorkspace.tsx`
- Modify: `frontend/src/features/links/LinksTable.tsx`
- Modify: `README.md`
- Test: `frontend/tests/dashboard-lifecycle.test.mjs` and relevant existing tests

**Interfaces:**
- Optimistic created links preserve `expiresAt`; table shows a localized expiry date or `Vĩnh viễn`.

- [ ] **Step 1: Write failing dashboard test** for expiry propagation and permanent-link display.
- [ ] **Step 2: Run the focused dashboard test and confirm failure.**
- [ ] **Step 3: Implement the display and update the API contract documentation.
- [ ] **Step 4: Run frontend tests, lint, typecheck, and static build.

### Task 6: Conflict resolution, full verification, GitHub, and AWS

**Files:**
- Modify: all files above plus plan/spec docs as needed.

- [ ] **Step 1: Run the full repository checks (`make check`) and inspect `git diff`/`git status`; preserve unrelated untracked audit files.
- [ ] **Step 2: Fetch `origin/main`, rebase or merge it into the current PR branch, resolve any conflicts in favor of the tested implementation, rerun checks, and commit the conflict resolution plus feature.
- [ ] **Step 3: Push the branch to `origin` and verify PR #97 reports no merge conflict and the new commit is its head.
- [ ] **Step 4: Use the repository's guarded GitHub Actions `Deploy dev` workflow to create a plan, review its resource actions, then apply the reviewed plan at the same commit; do not bypass `scripts/deploy-app.sh` or deploy from a non-main ref.
- [ ] **Step 5: Verify the deployed frontend/API smoke path and report GitHub commit, PR status, workflow run, and AWS deployment result with fresh command evidence.
