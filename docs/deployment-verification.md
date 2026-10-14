# Production deployment verification

Verified on **2026-10-12** at **https://npt-shortenlink.dev/**.

## Deployment

Terraform created 26 resources in the AWS project's selected Sydney region
(`ap-southeast-2`). Name.com hosts the apex ANAME and ACM DNS validation record;
ACM supplies the CloudFront certificate from `us-east-1`.

The frontend is a Next.js static export in a private S3 bucket, accessed through
CloudFront OAC. The Go Lambda uses its IAM execution role for DynamoDB access.
The deployment retained the existing account plan and did not modify AWS-managed
organization policies.

## Evidence

| Check | Result |
| --- | --- |
| Frontend and `/healthz` on the canonical HTTPS domain | HTTP 200; health status `ok` |
| Create -> metadata -> redirect | HTTP 201 -> HTTP 200 -> HTTP 302 to the submitted URL |
| HTTP entrypoint | HTTP 301 to HTTPS |
| Security headers | HSTS and `X-Content-Type-Options: nosniff` present |
| Untrusted browser origin | No wildcard or matching allow-origin header |
| Anonymous direct S3 access | HTTP 403 |
| Lambda package | Executable Linux ARM64 `bootstrap` in ZIP, mode 0755 |
| Frontend/Go validation | Tests, ESLint, TypeScript/static build and Go vet passed |
| Infrastructure and deployment tests | Terraform mocked security test and four Node tests passed |
| Dependency checks | Production pnpm audit clean; no reachable Go vulnerability findings |

An exact-match scan checked the local AWS credential values against source,
frontend export, Lambda binary, Terraform state/decoded plan, live homepage and
eight public JavaScript assets. No matches were found, and credential values
were not printed or committed.

## Scope and limitations

These checks establish the deployed release's critical flow and selected
security controls. They are not a penetration test, load test or ongoing SLA.
The API remains publicly usable without authentication, as defined by the MVP.
AWS throttling and Lambda concurrency limits are configured; WAF and advanced
analytics remain outside the deployed scope.

Local Windows race tests were unavailable because cgo/a C compiler was absent;
the repository's Linux CI runs them. The unused OpenPGP module advisory and
review date are documented in the [deployment runbook](../infra/aws/README.md).

Protect and back up the local Terraform state before future changes; state and
generated artifacts remain excluded from Git. See the runbook for DNS, deployment
and rollback commands.
