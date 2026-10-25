# Phase 02 review — 2026-10-25

**Follow-up:** findings 4 (clean-runner CI) and 5 (pause/resume guide) below are now resolved; ACM is issued and custom-domain inputs are configured. Bootstrap state has a verified encrypted/versioned off-machine backup. See [current deployment evidence](phase-02b-verification.md) for the passing GitHub run and remaining AWS account-verification blocker. The original review below is retained as the checkpoint that prompted those fixes.

**Verdict: backend and local implementation pass review; full Phase 02 is not complete.** The remaining work is cloud hosting/domain delivery, final end-to-end acceptance, clean-runner CI evidence and a safe dev pause/resume guide. No new blocking backend code defect was found at the agreed low-traffic dev scope.

This review inspects the current working tree, not a commit diff: the local repository has no initial commit or configured remote. Product code and AWS resources were not changed. A read-only Terraform plan was saved to ignored `infra/terraform/dev/phase-02-review.tfplan` and was not applied.

## Completion by task

| Task                             | Implemented / verified                                                                                                                                                                                                            | Remaining                                                                                                           |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| 02.1 Native Lambda transport     | Three native payload-v2 entrypoints call application use cases. Tests cover create/get/list/update/resolve, error mapping, body/base64 limits, HEAD and route isolation.                                                          | No new code gap found.                                                                                              |
| 02.2 Cloud bootstrap/config      | Lambda execution-role credentials and regional DynamoDB client are separate from local dummy credentials; client reused and request context propagated.                                                                           | No new code gap found.                                                                                              |
| 02.3 Packaging                   | Three executable Linux/arm64 bootstrap ZIPs; tests verify ELF, ZIP permissions, SHA-256 and repeated-build equality.                                                                                                              | Linux runner execution remains unverified.                                                                          |
| 02.4 Terraform state             | Separate bootstrap, private encrypted/versioned state bucket, S3 backend/native locking, pinned provider/toolchain; both roots validate and pass mock tests.                                                                      | Preserve local bootstrap state; off-machine backup is not verified by this review.                                  |
| 02.5 DynamoDB/Lambda/IAM/logging | Backend deployed. DynamoDB/GSI ACTIVE, on-demand billing, deletion protection; live Lambda aliases, scoped roles, logs and metric-filter definitions. Prior deployment evidence includes log correlation and IAM negative checks. | No new backend deployment gap found. Full metric ingestion/alert operations are not claimed.                        |
| 02.6 HTTP API                    | All ten live routes match intended auth: six management routes AWS_IAM, four redirect/health routes public. CORS/throttle configured.                                                                                             | CloudFront forwarding still needs real-path acceptance.                                                             |
| 02.7 Static hosting              | Private S3, OAC, cache policies and rewrite function defined; static export uploaded. Direct S3 access is 403; dashboard HTML has correct type/no-cache.                                                                          | Distribution and bucket policy are absent. Frontend delivery, custom domain and cloud browser tests are incomplete. |
| 02.8 CI                          | Credential-free workflow has pinned action references, Go/race/integration, packaging, frontend/browser and Terraform gates. Local checks pass.                                                                                   | No GitHub runner execution, Linux/race result or clean-runner proof.                                                |
| 02.9 AWS acceptance              | Approved account/region/budget; backend deployed; prior 20/20 backend smoke results retained. Fresh read-only endpoint checks pass.                                                                                               | Full smoke with `-site`, real CloudFront/OAC delivery and safe pause/resume instructions remain incomplete.         |

## Findings and unfinished acceptance

1. **Hosting is the remaining deployment blocker.** Fresh inventory returns no CloudFront distribution; Terraform plans exactly `aws_cloudfront_distribution.web` and `aws_s3_bucket_policy.web`: **2 add, 0 change, 0 destroy**. The last creation attempt failed with AWS account-verification AccessDenied. This read-only review did not retry creation, so it cannot establish whether AWS has since removed that restriction. Complete account verification, then regenerate/review a plan before applying. Without these resources the private export cannot be served to users. See [hosting.tf](../infra/terraform/dev/hosting.tf) and [support draft](aws-support-cloudfront.md).

2. **The certificate is ready, but the current plan does not attach the domain.** ACM now reports **ISSUED**, validation **SUCCESS**, `InUseBy: []` for `npt-shortenlink.dev`. This supersedes the earlier pending-validation reports. Actual plan inputs remain `site_domain = ""`, `acm_certificate_arn = null`, so applying that plan would use the CloudFront hostname/default certificate. Set the intended domain and issued us-east-1 ARN before preparing the custom-domain plan; configure Name.com root ANAME using the actual distribution hostname afterward. Keep the validation CNAME. Otherwise the intended public domain remains unfinished. See [variables.tf](../infra/terraform/dev/variables.tf) and [hosting.tf](../infra/terraform/dev/hosting.tf).

3. **Cloud end-to-end acceptance is missing.** Existing 20/20 results explicitly cover only the backend. Home/deep-link HTML, JS/CSS delivery, TLS/domain, OAC origin access, API forwarding and uncached redirect state changes through CloudFront have not passed live checks. Run full `cmd/smoke` with `-site` plus browser/static-asset checks after hosting is available. Browser management remains intentionally 403 until Phase 03; do not treat it as an unfinished Phase 02 authentication feature. See [smoke command](aws-development.md#deployment-smoke-command) and [02.9 checklist](../tasks/todo.md).

4. **CI is configured, not proven on GitHub.** There is no local initial commit or remote, and no real run was produced. Keep workflow execution separate from configuration completion. Connect the intended repository through the normal reviewed Git workflow and obtain a passing clean Linux run; do not imply that Windows/local checks prove the race-detector or Linux runner result. See [ci.yml](../.github/workflows/ci.yml).

5. **The Phase 02B safe dev pause/resume guide is missing.** The checklist explicitly requires a way to stop dev activity without deleting out-of-scope data. Existing documentation warns against destroy and preserves state/data, but does not provide a concrete traffic-disable/resume procedure and validation. Add the short guide before closing this acceptance item; distinguish disabling traffic from deleting resources and from stopping all charges. See [02.9 checklist](../tasks/todo.md) and [AWS development](aws-development.md).

## Fresh checks in this review

| Check                                                             | Result                                                             |
| ----------------------------------------------------------------- | ------------------------------------------------------------------ |
| Go tests with `DYNAMODB_INTEGRATION=1`, `-count=1`                | Pass, using the running DynamoDB Local container                   |
| Go vet/build/gofmt                                                | Pass                                                               |
| Node static-server, Lambda-packaging and CloudFront-rewrite tests | 3/3 pass                                                           |
| Terraform fmt/validate                                            | Both roots pass                                                    |
| Terraform mock safety tests                                       | Both roots pass                                                    |
| Real AWS Terraform refresh/plan                                   | Only two pending creates; no updates/deletes                       |
| pnpm lint/typecheck/format/build                                  | Pass; home/dashboard static export succeeds                        |
| Production dependency audit                                       | No known vulnerabilities reported                                  |
| API health                                                        | 200, no-store                                                      |
| Unsigned management list                                          | 403                                                                |
| Missing public redirect                                           | 404, no-store                                                      |
| Direct private S3 index                                           | 403                                                                |
| ACM                                                               | ISSUED / SUCCESS / not attached                                    |
| AWS Budget                                                        | 4 USD MONTHLY, account-wide; no notifications                      |
| Ignore rules                                                      | State, tfvars, backend config, saved plan and raw evidence ignored |

The prior 20-result live backend smoke file was checked for presence/count, not rerun: this review avoided creating more cloud fixtures. Previous local browser results (10 static E2E and one Strict Mode regression) remain historical evidence; this pass did not rerun browsers. `actionlint` is unavailable in PATH in this pass; the earlier static validation result is recorded in [02A verification](phase-02-verification.md). No load test, paid-traffic test, new AWS resource, support submission or DNS mutation occurred.

## Scope and cost limits

Cognito/JWT, ownership, URL editing/deletion, analytics, WebSocket, moderation, OIDC deployment, canary/rollback and production alarms are later-phase work, not missing Phase 02 features.

The 4 USD budget is console tracking, not a hard spending cap, and currently has no notifications. A reported spend of zero is delayed billing data, not proof of no charges. Bootstrap state is local and must be preserved; its secure external backup is unverified. This is a restricted development stack, not a production-readiness approval.

Close findings 1–5 and obtain the full hosting acceptance evidence before marking Phase 02 complete. Phase 03 design can proceed against the existing backend, but should not hide the unfinished Phase 02 hosting/verification gates.
