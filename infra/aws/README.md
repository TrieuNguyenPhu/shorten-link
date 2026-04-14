# AWS infrastructure

This directory contains the AWS SAM/CloudFormation stack for `npt-shortenlink.dev`.

> This runbook documents an operator-controlled deployment. Repository
> validation and pull requests never execute `sam deploy`, mutate DNS, upload
> static files, or invalidate a production distribution.

The stack provisions:

- one Go Lambda (`provided.al2023`, `arm64`) built from `services/shortener-api`;
- an API Gateway HTTP API for link creation, lookup, redirect, and health routes;
- a DynamoDB on-demand table with point-in-time recovery and TTL on the numeric `ttl` attribute;
- a private, versioned S3 bucket exposed only through CloudFront Origin Access Control;
- CloudFront behaviors for `/api/*`, `/link/*`, and `/healthz` plus the static frontend origin;
- Route 53 IPv4/IPv6 aliases for the custom domain.

## Prerequisites

- AWS CLI + credentials/profile có quyền deploy stack và upload frontend.
- AWS SAM CLI, Go `1.26.5+` và GNU Make để build Lambda qua Makefile.
- Node.js `22.20+` và pnpm `10.33+` để build static frontend.
- An existing public Route 53 hosted zone containing `npt-shortenlink.dev`.
- An issued ACM certificate in **us-east-1** that covers the domain. CloudFront requires its viewer certificate in that region even when the stack is deployed elsewhere.
- The backend Makefile target `build-ShortenerFunction`. SAM invokes it from `services/shortener-api` and expects the `bootstrap` artifact in `$(ARTIFACTS_DIR)`.

## Validate and build

Run from the repository root:

```bash
sam validate --lint --template-file infra/aws/template.yaml
sam build --template-file infra/aws/template.yaml
```

The root Makefile also exposes the mutation-free end-to-end plan:

```bash
make deploy-dry-run CERTIFICATE_ARN=arn:aws:acm:us-east-1:123456789012:certificate/replace-me HOSTED_ZONE_ID=ZREPLACE_ME
```

## Recommended deployment

From a fresh clone, the supported operator workflow is one command:

```bash
make deploy CERTIFICATE_ARN=arn:aws:acm:us-east-1:123456789012:certificate/replace-me HOSTED_ZONE_ID=ZREPLACE_ME
```

It performs these stages in order and stops on the first failure:

1. validate tool versions, required parameters, and AWS identity;
2. install the frozen pnpm lockfile and run frontend/backend verification;
3. run dependency, vulnerability, OpenAPI, and SAM validation;
4. build and deploy the CloudFormation stack;
5. read bucket/distribution/site values from stack outputs;
6. publish immutable assets before mutable HTML;
7. invalidate CloudFront and wait for completion;
8. smoke-test the frontend and `/healthz`.

The following Make variables have defaults and may be overridden on the same
command:

| Variable | Default |
|---|---|
| `AWS_REGION` | `ap-southeast-1` |
| `STACK_NAME` | `npt-shortenlink-prod` |
| `ENVIRONMENT_NAME` | `prod` |
| `DOMAIN_NAME` | `npt-shortenlink.dev` |
| `CORS_ALLOWED_ORIGINS` | `https://<DOMAIN_NAME>` |
| `AWS_PROFILE` | Uses the standard AWS credential chain |

`CERTIFICATE_ARN` and `HOSTED_ZONE_ID` intentionally have no defaults. This
prevents a clone from mutating an unintended account or DNS zone.

## Manual SAM deployment

Replace both placeholder values before running:

```bash
sam deploy \
  --stack-name npt-shortenlink-prod \
  --region ap-southeast-1 \
  --resolve-s3 \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides \
    EnvironmentName=prod \
    DomainName=npt-shortenlink.dev \
    CertificateArn=arn:aws:acm:us-east-1:123456789012:certificate/replace-me \
    HostedZoneId=ZREPLACE_ME \
    CorsAllowedOrigins=https://npt-shortenlink.dev
```

The manual command only deploys CloudFormation. It does not publish the
frontend, wait for a CloudFront invalidation, or run public smoke tests; prefer
`make deploy` for a complete release.

CloudFormation retains the DynamoDB table, frontend bucket, and log groups when the stack is deleted or those resources are replaced. This protects link data and logs, but they must be cleaned up explicitly when they are no longer needed.

## Publish the static frontend

Production uses relative `/api/*` requests through the same CloudFront domain, so no public API base URL is required at build time. Run `pnpm build:web` to produce the static export in `apps/web/out`, then read `FrontendBucketName` and `CloudFrontDistributionId` from the stack outputs. Upload content-hashed Next.js assets first with a one-year immutable policy, then upload HTML and other mutable files with immediate revalidation:

`make deploy` performs this sequence automatically. The commands below are the
manual recovery/reference procedure:

```bash
aws s3 sync apps/web/out/_next/static "s3://$FRONTEND_BUCKET/_next/static" \
  --cache-control "public,max-age=31536000,immutable"

aws s3 sync apps/web/out "s3://$FRONTEND_BUCKET" \
  --delete \
  --exclude "_next/static/*" \
  --cache-control "public,max-age=0,must-revalidate"

aws cloudfront create-invalidation --distribution-id "$DISTRIBUTION_ID" --paths "/*"
```

Do not delete old content-hashed assets during the upload: a browser or edge may still hold HTML from the previous release that references them. A lifecycle policy can remove old object versions after the chosen rollback window.

The default CloudFront behavior uses AWS's managed `CachingOptimized` policy. Its minimum TTL is one second, so that lower bound can override `no-cache`, `no-store`, or `max-age=0` from S3; the release invalidation above is therefore still required for immediately visible HTML changes. The bucket has no public access, and CloudFront signs origin requests through Origin Access Control.

## Rollback

CloudFormation stack operations must use a reviewed change set. If a stack
update fails, inspect the CloudFormation events and allow the service to finish
its automatic rollback before starting another operation. Do not delete the
stack as a rollback mechanism: the table, bucket, and log groups are retained
and require explicit lifecycle decisions.

For a frontend rollback:

1. identify the last known-good S3 object versions;
2. restore those versions without deleting content-hashed assets still
   referenced by cached HTML;
3. invalidate `/*` on the same distribution;
4. run the `POST → metadata → 302` smoke flow before declaring recovery.

For an API rollback, redeploy the last known-good SAM artifact and parameter
set, then verify `/healthz`, structured logs, DynamoDB reads/writes, and the
public redirect path. DNS is not changed during an application rollback.
