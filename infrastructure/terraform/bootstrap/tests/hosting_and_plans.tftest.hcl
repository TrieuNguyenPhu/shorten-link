mock_provider "aws" {
  override_during = plan
  mock_data "aws_partition" { defaults = { partition = "aws" } }
  mock_resource "aws_s3_bucket" { defaults = { arn = "arn:aws:s3:::url-shortener-test-state" } }
}
variables {
  aws_account_id    = "111111111111"
  state_bucket_name = "url-shortener-test-state"
}
run "unpinned_frontend_cannot_modify_cloudfront" {
  command = plan
  assert {
    condition     = alltrue([for s in jsondecode(aws_iam_policy.hosting_deployment.policy).Statement : alltrue([for a in s.Action : !startswith(a, "cloudfront:")])])
    error_message = "Unknown CloudFront resource IDs must not become wildcard grants."
  }
  assert {
    condition     = jsondecode(aws_iam_policy.hosting_deployment.policy).Statement[0].Resource == ["arn:aws:s3:::url-shortener-test-state/dev/plans/*"] && one(aws_s3_bucket_lifecycle_configuration.plans.rule).filter[0].prefix == "dev/plans/" && one(aws_s3_bucket_lifecycle_configuration.plans.rule).noncurrent_version_expiration[0].noncurrent_days == 2
    error_message = "Sensitive plans need a separate prefix and expiry for both current and old versions."
  }
}
run "pinned_hosting_permissions_and_quota" {
  command = plan
  variables {
    frontend_resource_ids = {
      distribution_id            = "EDIST123"
      origin_access_control_id   = "EOAC123"
      cache_policy_ids           = ["11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"]
      response_headers_policy_id = "33333333-3333-3333-3333-333333333333"
    }
  }
  assert {
    condition     = length(aws_iam_policy.hosting_deployment.policy) <= 6144 && one([for s in jsondecode(aws_iam_policy.hosting_deployment.policy).Statement : s if s.Sid == "FrontendDistribution"]).Resource == ["arn:aws:cloudfront::111111111111:distribution/EDIST123"] && alltrue([for s in jsondecode(aws_iam_policy.hosting_deployment.policy).Statement : !contains(s.Resource, "*") && !contains(s.Action, "cloudfront:CreateDistribution") && !contains(s.Action, "s3:DeleteBucket")])
    error_message = "Hosting maintenance must pin existing resources, omit destructive provisioning, and fit IAM quota."
  }
}
run "reject_wildcard_frontend" {
  command = plan
  variables {
    frontend_resource_ids = {
      distribution_id            = "*"
      origin_access_control_id   = "EOAC123"
      cache_policy_ids           = ["11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"]
      response_headers_policy_id = "33333333-3333-3333-3333-333333333333"
    }
  }
  expect_failures = [var.frontend_resource_ids]
}
