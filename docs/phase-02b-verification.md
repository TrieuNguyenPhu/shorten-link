# Phase 02B deployment evidence — 2026-10-25

**Status: backend deployed and verified; full Phase 02 remains incomplete because AWS account verification blocks CloudFront.**

The user approved account **654822777701**, profile `default`, region **ap-southeast-2 (Sydney)**, budget target **4 USD/month**. No AWS credentials were printed or committed. No existing resources or policies outside the project were deleted or replaced.

## Deployment outcome

- Bootstrap: **5 resources added, 0 changed, 0 destroyed**. Private/versioned/encrypted/TLS-only state bucket `shortly-dev-tfstate-654822777701-ap-southeast-2`. Bootstrap state remains local and ignored; dev state uses this S3 backend with native lockfiles.
- Bootstrap off-machine backup verified in that same private/versioned bucket at `shortly/bootstrap/backups/bootstrap-20261025T040007Z.tfstate`: AES256, matching SHA-256 metadata, nonempty S3 version ID. Local state remains canonical. Recovery/limits are documented in [pause/resume](aws-dev-pause-resume.md#bootstrap-state-backup).
- Dev saved plan: **56 creates, 0 updates/deletes**. Apply created **54** resources, including DynamoDB, three Lambda functions/published versions/live aliases, scoped execution roles, HTTP API/routes, logging/metrics, private web storage and budget.
- CloudFront CreateDistributionWithTags returned **403 AccessDenied**, explicitly requiring account verification through AWS Support. Initial request ID: `bc0180e2-34d7-4a15-ab61-5fdba53b1312`. A fresh reviewed apply with the issued custom-domain certificate still failed with the same restriction, request ID `4001d2c9-d748-4662-a108-2fd2f2e9c8fe`.
- Subsequent real-account plan confirms only **two pending creates**: `aws_cloudfront_distribution.web`, `aws_s3_bucket_policy.web`. No updates, replacements or deletes. Do not reapply the original stale plan; regenerate after verification.

## Live endpoints and configuration

| Item           | Value / evidence                                                                                                                                |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| API            | `https://ujgb995hc5.execute-api.ap-southeast-2.amazonaws.com`                                                                                   |
| Health         | GET `/health` returns 200 and `{"status":"ok"}`                                                                                                 |
| DynamoDB       | `shortly-dev-links`, ACTIVE, PAY_PER_REQUEST, GSI1 ACTIVE, deletion protection enabled                                                          |
| Lambda runtime | provided.al2023, arm64; live aliases point to published versions                                                                                |
| Logs           | `/aws/lambda/shortly-dev-*` and `/aws/apigateway/shortly-dev`, 3-day retention                                                                  |
| Web bucket     | `shortly-dev-web-aefb0bbd1464560878eb5dc7e7`; all four public-access blocks enabled                                                             |
| Frontend       | Static build with public API origin uploaded assets first, HTML last; home and dashboard HTML Content-Type text/html and Cache-Control no-cache |
| Direct S3      | Unauthenticated GET index.html returns 403                                                                                                      |
| AWS Budget     | `shortly-dev-monthly-guard`, MONTHLY, 4 USD; account-wide console tracking with no invented notification recipients                             |

The cloud frontend is **not publicly accessible yet**. Management from an unsigned browser still waits for Phase 03 Cognito; the live management API is intentionally restricted by AWS_IAM.

## GitHub CI follow-up

[Run 40160153532 passed](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/40160153532) on Ubuntu 24.04 for source snapshot commit `fdf10a31602b511fa871c2299012dff3fbfbf439`, branch `phase-02/completion-20261025`. It ran frozen installation, Go formatting/vet/race tests with real DynamoDB Local, three Node/packaging/rewrite tests, Lambda artifact build, both Terraform mock safety gates, frontend lint/typecheck/audit/static export, **one Strict Mode test and ten browser E2E tests**. Artifacts uploaded successfully. The existing `main` branch/history were preserved; the original local working tree was not committed/reset. Subsequent evidence-only documentation updates do not change the verified application/infra source.

The runner warned that three pinned actions use a Node 20 entrypoint and were forced to Node 24; all steps passed. Updating those pins is deferred maintenance, not evidence of a failed gate.

## Real AWS smoke results

Command (backend-only, explicitly excludes CloudFront/static delivery checks):

```sh
go -C services/lambda run ./cmd/smoke -api https://ujgb995hc5.execute-api.ap-southeast-2.amazonaws.com -bucket shortly-dev-web-aefb0bbd1464560878eb5dc7e7 -region ap-southeast-2 -profile default
```

**20/20 checks passed:** health; unsigned create/list/get/update → 403; SigV4 create → 201; signed metadata/list/update → 200; active redirect → 302 with exact destination and no-store; disabled → 410; re-enabled → 302; missing → 404; invalid destination → 400; expiring link → 302 then 410 after expiry; direct S3 → 403; final fixture disabled. The HTTP client never followed destinations. Two `smoke-*` records remain (disabled/expired).

IAM simulation against the deployed redirect role and table confirms `dynamodb:GetItem` allowed and `PutItem`/`DeleteItem` denied. It supplements the live smoke evidence; it does not replace testing real resource calls.

Actual API Gateway and Lambda JSON logs show request IDs, route, status and latency for live invocations. AWS CloudWatch `TestMetricFilter` with `{ $.status >= 500 }` matched only the 500 fixture, excluding 302; input and response are saved separately as `metric-filter-test.json` and `metric-filter-result.json`. This verifies filter matching without injecting a production failure or publishing a synthetic metric.

Final local checks after deployment changes: Go tests (`-count=1`), vet and build passed; both Terraform roots passed fmt, validate and mock-provider safety tests. Frontend cloud build/upload and the subsequent local-default build passed. Independent review found no new code blocker. Linux CI subsequently reran local browser tests successfully; browser tests against the unavailable CloudFront site remain blocked.

Raw evidence is ignored under `dist/evidence/` (`backend-smoke.json`, dev/remaining/finish plan and apply logs). Terraform state/plan/tfvars/backend files are ignored. GitHub CI follow-up is recorded in the [deployment ledger](../tasks/phase-02b-progress.md).

## Budget assumptions and limits

Expected order of magnitude **1–2 USD/month** at ≤50,000 API requests/month, ≤100 MiB log ingestion, small table/index/storage and no paid WAF/NAT/provisioned concurrency/dev PITR. This estimate excludes domain renewal/tax and assumes low dev use, not continuous traffic at the throttle limit.

Regional AWS price-list checks: HTTP API Sydney **1.29 USD/million requests** (first tier), custom metrics Sydney **0.30 USD/metric-month** (three metrics approximately 0.90 USD/month before allowances). Source: [API regional price list](https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonApiGateway/current/ap-southeast-2/index.json), [CloudWatch regional price list](https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonCloudWatch/current/ap-southeast-2/index.json).

AWS Budget is a tracking threshold, not a hard spending cap; it includes prior/other account costs and is delayed. Its current reported zero is not proof that no charges have accrued. No email/SNS alerts were configured because no recipient was supplied. See [AWS Budgets behavior](https://docs.aws.amazon.com/cost-management/latest/userguide/budgets-managing-costs.html).

## Remaining external steps

1. Account owner completes AWS verification via Support. A [reviewable support request](aws-support-cloudfront.md) is prepared locally and **has not been sent**.
2. ACM is now **ISSUED**, DNS validation **SUCCESS**. The ignored dev configuration now specifies `npt-shortenlink.dev` and its issued us-east-1 certificate. CloudFront cannot attach it until account verification is resolved; the root ANAME must target the actual resulting distribution hostname.
3. Resume the reviewed Terraform plan, finish S3/OAC policy, rebuild/upload with the live site origin, run full smoke with `-site` and verify home/deep links/assets in a browser.

No public-S3 fallback, alternate AWS account, account-policy modification or verification bypass was attempted. Backend/storage are preserved for resumption.

The [safe dev pause/resume guide](aws-dev-pause-resume.md) is complete, including account/resource preflight, data preservation, normal-baseline restoration and cost/Terraform-drift limits. The concurrency baseline was checked read-only; an outage drill was not executed.
