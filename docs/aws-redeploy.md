# Adopt CloudFront and deploy again

Current account `654822777701`, region `ap-southeast-2`, domain `npt-shortenlink.dev`. AWS backend and private S3 already exist. The owner has submitted an account-verification Support case; no distribution exists yet. Do not retry creation repeatedly while AWS holds the account. This guide prepares the remaining work; it is not evidence of a completed import or public frontend deployment.

## 1. Reconnect an existing checkout

Install the pinned Go/Node/pnpm/Terraform versions from [AWS development](aws-development.md), then run `pnpm install --frozen-lockfile`. Select the existing CLI profile using PowerShell:

```powershell
$env:AWS_PROFILE = 'default'
aws sts get-caller-identity
```

Confirm the intended account. Keep `infra/terraform/dev/backend.tfbackend` and `dev.tfvars` private/ignored. On a new checkout, copy their example files and restore the **existing** state bucket/key and actual dev values. For this stack, backend bucket is `shortly-dev-tfstate-654822777701-ap-southeast-2`, key `shortly/dev/terraform.tfstate`, region Sydney, encryption/lockfile enabled. Do not rerun bootstrap or replace the state bucket when reconnecting.

```powershell
terraform '-chdir=infra/terraform/dev' init '-backend-config=backend.tfbackend' -input=false
pnpm build:lambdas
```

Use the canonical domain/issued us-east-1 certificate, 4 USD tracking budget and 3-day logs in dev tfvars. The current certificate ARN is recorded in [deployment evidence](phase-02b-verification.md); account/domain-specific inputs belong in ignored configuration. The local bootstrap state and its backup are documented in [pause/resume](aws-dev-pause-resume.md#bootstrap-state-backup).

## 2. Adopt a distribution created in the console — one time only

Once creation succeeds, inspect its ID, account, origins, OAC, bucket policy, aliases/certificate, WAF association and pricing-plan subscription before changing state. The web origin must use `shortly-dev-web-aefb0bbd1464560878eb5dc7e7.s3.ap-southeast-2.amazonaws.com`, not the Terraform state bucket or an S3 website endpoint.

Back up the current dev state to ignored local evidence before importing. Run from repository root:

```powershell
New-Item -ItemType Directory -Force dist/evidence | Out-Null
$beforeImport = terraform '-chdir=infra/terraform/dev' state pull
if ($LASTEXITCODE -ne 0) { throw 'Could not back up existing state' }
[System.IO.File]::WriteAllText((Join-Path (Get-Location) ('dist/evidence/dev-before-import-' + (Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssZ') + '.tfstate')), ($beforeImport -join [Environment]::NewLine), (New-Object System.Text.UTF8Encoding($false)))
```

Confirm `aws_cloudfront_distribution.web` is not already bound in `terraform state list`, replace the example ID below, then import:

```powershell
$distributionId = 'REPLACE_WITH_CREATED_DISTRIBUTION_ID'
if ($distributionId -notmatch '^E[A-Z0-9]+$') { throw 'Set the actual distribution ID first' }
terraform '-chdir=infra/terraform/dev' import '-var-file=dev.tfvars' aws_cloudfront_distribution.web $distributionId
if ($LASTEXITCODE -ne 0) { throw 'Distribution import failed; inspect state before continuing' }
```

If the console already created the web bucket policy and it is absent from state, import that policy too, using the bucket name as ID:

```powershell
terraform '-chdir=infra/terraform/dev' import '-var-file=dev.tfvars' aws_s3_bucket_policy.web shortly-dev-web-aefb0bbd1464560878eb5dc7e7
```

Never bind one remote object to two Terraform addresses. Existing Terraform OAC/function/cache policies already have state entries: do not import a different console-created object over those bindings. Reconcile the actual distribution with the intended HCL, reusing existing project resources where appropriate. Leave unrelated resources alone.

**Free flat-rate plan:** preserve the user's selected tier and its mandatory WAF association. The current baseline uses custom cache policies, which are not supported by Free. Reconcile these to appropriate AWS-managed policies and represent/preserve the plan lifecycle before applying; never silently upgrade the plan or detach WAF. A successful distribution import alone does not finish plan/WAF adoption. See [AWS plan restrictions](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/flat-rate-pricing-plan.html).

After reconciliation, verify the intended S3/API origins, static rewrite function and `/api/*`, `/r/*`, `/health` behaviors; API/redirect cache TTLs must remain zero. Check that the private bucket policy permits only the adopted distribution and denies insecure transport. Plan first:

```powershell
terraform '-chdir=infra/terraform/dev' plan '-var-file=dev.tfvars' '-out=adopt.tfplan' -input=false
terraform '-chdir=infra/terraform/dev' show adopt.tfplan
```

Review every action. Stop on unexpected replacement/deletion, data changes, paid-plan transitions or WAF removal. Import does not apply configuration. Apply the saved plan only after its actual changes have been reviewed; then wait for CloudFront deployment and aim for a fresh plan with no unintended changes.

Configure Name.com root ANAME to the resulting CloudFront hostname. Keep the ACM CNAME and existing nameservers. Check TLS/domain before publishing a build that embeds that site origin.

## 3. Repeat an infrastructure/backend deployment

After changing Go/infra code, run the relevant tests and rebuild the Lambda archives. Create/review a new saved plan, then apply exactly that saved plan:

```powershell
pnpm build:lambdas
terraform '-chdir=infra/terraform/dev' plan '-var-file=dev.tfvars' '-out=deploy.tfplan' -input=false
terraform '-chdir=infra/terraform/dev' show deploy.tfplan
```

After review, `terraform '-chdir=infra/terraform/dev' apply deploy.tfplan` updates the managed stack. Regenerate plans after any state/import/configuration changes; do not reuse a stale saved plan. This workflow does not recreate deleted resources blindly or perform an automatic destroy.

## 4. Repeat a frontend deployment

Once the hosting stack is reconciled, applied and Deployed:

```powershell
pnpm deploy:web
```

This command reads actual Terraform outputs, checks the selected CLI account, verifies bucket ownership and the distribution's site/S3/API origins, and refuses incomplete or mismatched targets. It builds Next with `NEXT_PUBLIC_API_BASE_URL` set to the site's HTTPS origin, uploads fingerprinted assets first, other assets next and HTML last, then requests a CloudFront invalidation. It retains old assets and does not make the bucket public, run Terraform apply or change DNS/plan settings. It works with `AWS_PROFILE`; no credentials are embedded in the script or frontend.

Check the invalidation status in CloudFront and verify home, direct dashboard load, JS/CSS and `/health`. Run the full smoke command from [AWS development](aws-development.md#deployment-smoke-command), substituting `api_url`, `site_url`, `web_bucket` and region from Terraform outputs. Management remains AWS_IAM until Phase 03.

The export in `apps/web/out` now targets the cloud. Run `pnpm build` without the cloud environment override before using local static preview again. On upload/invalidation failure, inspect the failed command and rerun after fixing it; the command does not automatically roll back or promise atomic multi-page publication. Versioned S3 objects/retained assets support manual recovery.

## Verification while account verification is pending

`pnpm test:deploy` exercises the deploy command with fake process responses: wrong/missing targets, ownership failure, build/upload failure and upload ordering. CI runs these tests without AWS credentials. Real upload/invalidation and import acceptance remain blocked until a distribution exists; `pnpm deploy:web` deliberately exits before building/uploading if Terraform hosting outputs are incomplete.
