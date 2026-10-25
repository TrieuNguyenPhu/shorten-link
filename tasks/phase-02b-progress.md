# Phase 02B deployment ledger — 2026-10-25

**Current status: backend deployed and verified; full Phase 02 is blocked by AWS account verification for CloudFront.** See [deployment evidence](../docs/phase-02b-verification.md).

The user requested completion of Phase 02 and approved account `654822777701`, profile `default`, Sydney `ap-southeast-2`, budget target **4 USD/month**. This authorizes the project dev deployment and smoke tests. The follow-up request to finish the reviewed CI gap is being executed on a separate branch of the matching existing GitHub repository; main/history remain preserved. No unrelated resource deletion, support message or Phase 03 functionality is included.

## Completed execution

- Confirmed CLI identity and inventoried retained resources. The original Singapore proposal was superseded by the user's Sydney choice. Region-policy checks were read-only; no permissions or account policies were changed.
- Reviewed and applied bootstrap: **5 added, 0 changed, 0 destroyed**. State bucket `shortly-dev-tfstate-654822777701-ap-southeast-2` is private, versioned, encrypted and TLS-only. Bootstrap state remains local/ignored; preserve it. Dev uses the S3 backend and native lockfiles.
- Reviewed dev saved plan: **56 creates**, no changes/deletes. Apply created **54 resources**, including the backend, private web storage and budget. A new plan after the failed apply confirms only **2 creates remain**: CloudFront distribution and its S3 bucket policy. The original saved plan is stale; regenerate before resuming.
- Created account-wide `shortly-dev-monthly-guard`, **4 USD/month**, without changing the existing 20 USD budget. Console tracking is not a hard spending cap; no notification recipient was invented. Logs retained 3 days; no NAT, paid WAF, provisioned concurrency, dev PITR or new Route 53 zone.
- Built frontend with the live direct API origin, uploaded fingerprinted assets first and HTML last, and checked Content-Type/cache headers. Saved cloud export under ignored `dist/web-aws/`; rebuilt local output with local defaults afterward. Direct S3 access remains 403.
- Real AWS backend smoke: **20/20 passed**. Checked signed management, unsigned denials, redirect/disabled/expired/missing behavior, invalid destination and private S3. The client never follows redirect destinations. Two tiny disabled/expired smoke fixtures remain.
- Verified deployed Lambda aliases/version, JSON gateway/Lambda request correlation, actual IAM redirect-role read-only permissions and CloudWatch metric-filter matching. Raw evidence is ignored under `dist/evidence/`; credentials remain in memory only.
- Independent code review found no code blocker in the optional backend-only smoke mode or budget configuration. Backend-only checks explicitly exclude frontend/CloudFront acceptance.

## External blockers and resumption

CloudFront CreateDistributionWithTags failed with 403 AccessDenied: account verification is required through AWS Support. Request ID `bc0180e2-34d7-4a15-ab61-5fdba53b1312`. A [support request draft and resumption steps](../docs/aws-support-cloudfront.md) are prepared locally and **have not been sent**. No bypass, public-S3 fallback or replacement account was attempted.

ACM certificate is CLI-managed in us-east-1, separately from Terraform: `arn:aws:acm:us-east-1:654822777701:certificate/c1e7ead4-422f-4b0e-ad5d-3cb0a1e1dd70`. The owner added the CNAME at Name.com and ACM now reports **ISSUED**, validation **SUCCESS**. The old ignored `dist/evidence/acm-validation-status.json` captures an earlier pending check, not current status. This does not resolve the separate account-verification restriction. Keep authoritative nameservers unchanged. The dev configuration includes the issued certificate; once CloudFront exists, point the root ANAME at its actual hostname.

After verification: regenerate/review/apply the remaining Terraform plan, finish OAC delivery, rebuild/upload using the live site origin, then run full smoke with `-site` and browser/deep-link/static-asset checks. Preserve the current backend and state. Do not mark Phase 02 complete before these checks pass.

## Follow-up completion attempt

- ACM now reports **ISSUED** for `npt-shortenlink.dev`; validation succeeded. Updated ignored dev tfvars to attach this domain and its issued us-east-1 certificate.
- Regenerated/reviewed saved plan: exactly two creates, no updates/deletes; inspected exact alias/certificate before apply. CloudFront creation still fails with account-verification 403, new request ID `4001d2c9-d748-4662-a108-2fd2f2e9c8fe`. No further retries or bypasses are appropriate until AWS removes the restriction.
- Added [safe pause/resume instructions](../docs/aws-dev-pause-resume.md), including account/resource checks, temporary zero concurrency, exact baseline restoration, data/health/auth verification and cost/drift limits. Read-only preflight verified all three functions are unreserved; no outage drill was executed.
- GitHub CLI is authenticated. Found `https://github.com/TrieuNguyenPhu/shorten-link` with matching historical project/CI. Asked for the intended URL; after allowing time for an answer, proceeded with this matching repository on isolated branch `phase-02/completion-20261025` to verify the current code. The local uncommitted workspace is preserved. A preparation clone remains under ignored `dist/phase-02-ci`; the actual source snapshot is under ignored `dist/phase-02-snapshot`. No force push or main update is used. A successful run of the old code would not prove this local implementation.
- Published a source-only snapshot from ignored `dist/phase-02-snapshot` to that branch, with existing main as commit parent; no filesystem deletion was needed. State/tfvars/backend/plan/cache/output files and credential patterns were checked before push. Commit `fdf10a31602b511fa871c2299012dff3fbfbf439`; real CI run [40160153532](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/40160153532) **passed**, including Linux Go race/integration, Lambda packaging, both Terraform roots and browser tests (10 E2E + 1 Strict Mode). Raw run JSON/log evidence is saved under ignored `dist/evidence/`.
- Fresh independent follow-up review found no new code/security blocker; corrected its single stale ACM-status documentation finding. No production code changed in this follow-up.
- Verified off-machine bootstrap-state backup in the existing private/versioned state bucket, key `shortly/bootstrap/backups/bootstrap-20261025T040007Z.tfstate`, AES256, matching SHA-256 metadata and version ID. Canonical local state is preserved. Raw evidence: ignored `dist/evidence/bootstrap-backup.json`.
