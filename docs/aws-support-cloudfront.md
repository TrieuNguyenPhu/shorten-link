# AWS Support draft — CloudFront account verification

**The owner has submitted an AWS Support case.** The Case ID has not been provided in this chat; no case was submitted by the assistant. Track the existing case in [AWS Support Center](https://console.aws.amazon.com/support/home#/) and provide any requested verification there. The prepared text below is retained for reference; do not open duplicate cases for the same restriction.

**Subject:** Verify account 654822777701 for CloudFront distribution creation

**Body to review and submit:**

We are deploying a small development URL shortener in AWS account 654822777701, primarily in ap-southeast-2 (Sydney). CloudFront distribution creation is blocked by an account-verification requirement.

Operation: `CloudFront: CreateDistributionWithTags`

Latest request ID: `4001d2c9-d748-4662-a108-2fd2f2e9c8fe` (earlier attempt: `bc0180e2-34d7-4a15-ab61-5fdba53b1312`).

Error returned:

```text
AccessDenied: Your account must be verified before you can add new CloudFront resources. To verify your account, please contact AWS Support (https://console.aws.amazon.com/support/home#/) and include this error message.
```

The intended workload is a low-traffic development project with a 4 USD/month budget target, private S3 static hosting using Origin Access Control, API Gateway HTTP API and Go Lambda functions. Management routes require AWS IAM authentication; only redirects and health are public. The intended domain is npt-shortenlink.dev. Existing AWS resources were removed before this fresh deployment.

Please advise which account-verification steps are required and enable CloudFront distribution creation after verification.

## Current project state

The backend and private web storage are already deployed and have been smoke-tested. No public S3 website or permission bypass was introduced. Terraform keeps the two pending resources (`aws_cloudfront_distribution.web` and its S3 read policy) in configuration; do not delete local/remote Terraform state or recreate the working backend.

After AWS confirms verification, regenerate and review the saved dev plan before applying it. It should create only those pending resources if the configuration has not changed. Then rebuild the frontend with the resulting site origin, upload assets before HTML, and run the full smoke command including `-site`.

The separate ACM certificate is now **ISSUED**, validation **SUCCESS**; the owner added its CNAME at Name.com. The current dev configuration includes the custom domain and certificate. This domain check is separate from the unresolved CloudFront account-verification restriction. Certificate ARN: `arn:aws:acm:us-east-1:654822777701:certificate/c1e7ead4-422f-4b0e-ad5d-3cb0a1e1dd70`.

| Type  | Host                                | Value                                                              |
| ----- | ----------------------------------- | ------------------------------------------------------------------ |
| CNAME | `_3c762ff95660ec5fd7828efd75433dd4` | `_ca4ec022b2b069d1375fee3df9101151.wzccmgtwzk.acm-validations.aws` |

Leave DNS authority at Name.com. After the certificate is issued and CloudFront is available, set the custom-domain variables and configure the root ANAME target using the actual distribution hostname; do not guess an IP or change name servers. Keep the ACM validation CNAME for renewal.
