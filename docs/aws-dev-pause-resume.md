# Pause and resume the Phase 02 dev backend

Target: account `654822777701`, profile `default`, Sydney `ap-southeast-2`. This procedure temporarily stops the three project's Lambda functions from starting new work while preserving the functions, aliases, API, DynamoDB data, S3 objects and Terraform state. It is not a zero-cost shutdown: API/edge requests, storage and monitoring can still incur charges.

AWS documents [reserved concurrency zero](https://docs.aws.amazon.com/lambda/latest/dg/configuration-concurrency.html) as the way to stop a function processing events; [delete-function-concurrency](https://docs.aws.amazon.com/cli/latest/reference/lambda/delete-function-concurrency.html) removes that limit when resuming.

## Preflight — PowerShell at the repository root

Stop local cloud smoke/load clients first. Never delete state or run `terraform destroy` to pause this stack. This procedure is for the current baseline with **no reserved concurrency configured**; if any function has a configured limit, record it and restore that exact value instead of using the baseline resume command.

```powershell
$devAccount = aws sts get-caller-identity --profile default --query Account --output text
if ($LASTEXITCODE -ne 0 -or $devAccount -ne '654822777701') { throw 'Wrong or unavailable AWS identity' }
$devFunctions = @('shortly-dev-create-link', 'shortly-dev-manage-links', 'shortly-dev-resolve-link')
foreach ($devFunction in $devFunctions) {
  Write-Output $devFunction
  aws lambda get-function-concurrency --profile default --region ap-southeast-2 --function-name $devFunction
  if ($LASTEXITCODE -ne 0) { throw "Cannot inspect $devFunction" }
}
```

An absent `ReservedConcurrentExecutions` field means the baseline is unreserved. Save the output with the maintenance notes before proceeding. Record current link count/known smoke code if a data-preservation drill is required; do not print destination URLs or credentials to shared logs.

## Pause

```powershell
foreach ($devFunction in $devFunctions) {
  aws lambda put-function-concurrency --profile default --region ap-southeast-2 --function-name $devFunction --reserved-concurrent-executions 0
  if ($LASTEXITCODE -ne 0) { throw "Pause incomplete at $devFunction; inspect all three functions before continuing" }
}
```

Allow already-running invocations to finish (the current function timeout is 10 seconds). Check each function with `get-function-concurrency`; all three must show `ReservedConcurrentExecutions: 0`. Backend `/health` should no longer return 200. API Gateway can return an integration/throttling error; do not require a specific status code for this maintenance state. This deliberately makes the dev application unavailable.

After CloudFront is deployed, its static content can still be served while the backend is paused. To pause edge delivery as well, identify only this project's distribution using `terraform -chdir=infra/terraform/dev output -raw distribution_id`, disable it in the CloudFront console and wait for deployment. Record its prior enabled state. Do not delete the distribution, certificate, DNS records or web bucket. Direct API requests can still reach API Gateway and incur request/log charges even with Lambda paused.

## Resume the current unreserved baseline

If edge delivery was disabled, re-enable that same distribution and wait for deployment. Then remove the temporary Lambda limits:

```powershell
foreach ($devFunction in $devFunctions) {
  aws lambda delete-function-concurrency --profile default --region ap-southeast-2 --function-name $devFunction
  if ($LASTEXITCODE -ne 0) { throw "Resume incomplete at $devFunction; inspect all three functions before continuing" }
}
```

`delete-function-concurrency` deletes only the concurrency setting, not the Lambda function. Verify that the field is absent again, direct `/health` returns 200, unsigned management still returns 403 and an existing link retains its enabled/expiry behavior. Verify DynamoDB remains ACTIVE, deletion protection remains enabled and known data is still present. Re-run the full smoke command from [AWS development](aws-development.md#deployment-smoke-command) once CloudFront is available; it creates two tiny smoke fixtures.

These are temporary operational overrides outside Terraform. Do not apply an unrelated Terraform plan while intentionally paused: the configured normal baseline can undo the pause. After resuming, regenerate a plan and verify no unintended updates/deletes; any still-pending CloudFront/policy creates are separate deployment work. Preserve bootstrap state and the S3 dev state throughout.

## Evidence and limits

### Bootstrap state backup

The canonical bootstrap state remains local at `infra/terraform/bootstrap/terraform.tfstate`; do not lose or commit it. A verified encrypted/versioned backup is stored in the private state bucket:

- Bucket: `shortly-dev-tfstate-654822777701-ap-southeast-2`
- Key: `shortly/bootstrap/backups/bootstrap-20261025T040007Z.tfstate`
- SHA-256: `685e0a5ee7be8a0dc0fccc9f6ab34d9840b8a6eec17a76d0e10df978780834d0`

For recovery, download the recorded S3 version to an ignored scratch file, compare its SHA-256 with `dist/evidence/bootstrap-backup.json` and the value above, then review the recovered resource IDs/serial before restoring the canonical local state. Do not overwrite an existing newer state or copy bootstrap state onto the dev application's state key. This backup is in the same AWS account/bucket, so it does not protect against losing that account or deleting the bucket; versioning/public-block/encryption remain enabled. Refresh the backup after a deliberate future bootstrap change.

Read-only preflight verified the identity and unreserved baseline for all three deployed functions. The pause/resume commands follow AWS documentation; an actual outage drill has not been executed. The 4 USD monthly budget tracks spend and has no notification recipient; it does not stop charges. Local development can continue separately with DynamoDB Local and the host API.
