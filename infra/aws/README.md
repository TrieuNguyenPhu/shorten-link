# Terraform AWS deployment

This module provisions the Go Lambda (`provided.al2023`, arm64), HTTP API,
DynamoDB, CloudWatch logs, IAM execution role, private S3 bucket and CloudFront.
Name.com remains the DNS provider. CloudFormation and Route 53 are not required.

## Prerequisites

- Terraform 1.15.5+, AWS CLI, Go 1.26.9+, Node 22.20+ and pnpm 10.33.2.
- An issued ACM certificate in **us-east-1** covering the deployment domain.
- AWS credentials permitted to manage the resources declared in this module.
- Access to the authoritative DNS provider for the domain.

The production project uses **ap-southeast-2 (Sydney)**. AWS Projects in the new
sign-up experience restrict regional resources to their selected region. Check
AWS Settings -> Project -> Additional Info before deploying to another project.
A region listed as enabled in the account does not prove a project permits it.
See [AWS project regions](https://docs.aws.amazon.com/accounts/latest/reference/project-regions.html).

Provider versions and Windows/Linux checksums are committed in
`.terraform.lock.hcl`. CI uses mocked providers; it never applies infrastructure,
changes DNS or uploads the frontend.

## Credentials and state

Use the existing local AWS credential chain or `AWS_PROFILE`. Never put access
keys in Terraform variables, tfvars, `NEXT_PUBLIC_*`, frontend files or uploads.
The Lambda uses its execution role; DynamoDB permissions are limited to GetItem
and PutItem on its own table. Production frontend calls relative `/api/*` URLs.

Terraform state, saved plans, `.terraform/`, `.build/` and real tfvars are ignored
by Git. State is local plaintext: back it up securely and never place it in the
frontend bucket. Keep the existing state when deploying updates. Multiple
operators must configure a shared backend with locking before concurrent changes.

Only `apps/web/out` is published. Old content-hashed assets are retained so
browsers with cached HTML can continue loading the previous release.

## Plan and deploy

Run from the repository root, supplying the issued certificate's public ARN:

```bash
make deploy-dry-run CERTIFICATE_ARN=<issued-acm-certificate-arn>
make deploy-plan CERTIFICATE_ARN=<issued-acm-certificate-arn>
make deploy CERTIFICATE_ARN=<issued-acm-certificate-arn>
```

- `deploy-dry-run` prints the workflow without builds or cloud mutations.
- `deploy-plan` verifies/builds and saves `infra/aws/production.tfplan` without
  applying resources or publishing the frontend.
- `deploy` verifies account/certificate, installs the frozen lockfile with scripts
  disabled, runs application/security/contract checks, builds Lambda, validates
  and tests Terraform, plans, checks explicit AWS permission denials, applies the
  saved plan, publishes frontend and invalidates CloudFront. It smoke-tests
  frontend, health and create -> metadata -> redirect on the distribution hostname.

Defaults: `AWS_REGION=ap-southeast-2`, `STACK_NAME=npt-shortenlink-prod`,
`DOMAIN_NAME=npt-shortenlink.dev`. Override these or set `AWS_PROFILE` as needed.
`STACK_NAME` is a resource-name prefix retained for command compatibility; it
is not a CloudFormation stack. CORS is exactly `https://<DOMAIN_NAME>`.
The former `HOSTED_ZONE_ID`, `ENVIRONMENT_NAME` and `CORS_ALLOWED_ORIGINS`
deployment overrides are no longer consumed.

The script stops before apply when IAM user permission simulation reports an
explicit denial. Regional actions use the deployment region; global IAM and
CloudFront actions use us-east-1. Simulation cannot evaluate assumed-role sessions
or guarantee every resource-level permission; AWS enforces authorization during
apply. Failed applies preserve Terraform state for repair and retry.

If resources already exist outside this state, import them before applying.
Do not manage the same resources with both SAM and Terraform.

## Name.com DNS and HTTPS

After apply, read the target from deployment output or:

```bash
terraform -chdir=infra/aws output -json dns_record
```

At Name.com, add **ANAME**, leave **Host empty** for the apex, set **Answer** to
the exported CloudFront hostname and use TTL 300. Preserve unrelated DNS records
and the ACM validation CNAME so certificate renewal continues. Do not pin
CloudFront IP addresses, change nameservers or use URL forwarding.
See [Name.com ANAME records](https://www.name.com/support/articles/115010493967-adding-an-aname-alias-record).

After propagation, verify the same create/metadata/302 flow on the canonical
domain and confirm HTTP redirects to HTTPS. Smoke links point to example.com
and expire after one day.

## Validation and rollback

```bash
node --test scripts/deploy-aws.test.mjs
terraform -chdir=infra/aws init -backend=false -input=false -lockfile=readonly
terraform -chdir=infra/aws fmt -check -recursive
terraform -chdir=infra/aws validate
terraform -chdir=infra/aws test
```

Tests mock both providers and create no AWS resources. CI builds the executable
Linux ARM64 artifact before validation. Table, bucket and logs have
`prevent_destroy`; DynamoDB PITR and S3 versioning are enabled.

For application rollback, rebuild/reapply the previous artifact with the same
state, restore prior S3 object versions and invalidate CloudFront. Do not use
`terraform destroy` or delete state as a rollback mechanism. Avoid renaming
persistent resources, which would require replacement.

## Security audit scope

Production pnpm audit reports no known vulnerabilities. Go symbol/package
scanning reports no reachable findings after the dependency updates.
GO-2026-5932 remains at module level for unused `x/crypto/openpgp`, with no fix;
reassess on 2026-11-12 or before adding OpenPGP functionality. Go race tests need
cgo and a C compiler and run in Linux CI.

See [production verification](../../docs/deployment-verification.md) for the live
release checks, their scope and remaining limitations.
