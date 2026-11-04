mock_provider "aws" {
  override_during = plan
  mock_resource "aws_s3_bucket" { defaults = { arn = "arn:aws:s3:::url-shortener-test-state" } }
  mock_resource "aws_iam_openid_connect_provider" { defaults = { arn = "arn:aws:iam::111111111111:oidc-provider/token.actions.githubusercontent.com" } }
  mock_data "aws_partition" { defaults = { partition = "aws" } }
  mock_resource "aws_iam_policy" { defaults = { arn = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-runtime-boundary" } }
}
variables {
  aws_account_id    = "111111111111"
  state_bucket_name = "url-shortener-test-state"
}
run "protected_environment_and_state" {
  command = plan
  assert {
    condition     = jsondecode(aws_iam_role.deployment.assume_role_policy).Statement[0].Condition.StringEquals["token.actions.githubusercontent.com:aud"] == "sts.amazonaws.com" && jsondecode(aws_iam_role.deployment.assume_role_policy).Statement[0].Condition.StringEquals["token.actions.githubusercontent.com:sub"] == "repo:TrieuNguyenPhu/shorten-link:environment:dev"
    error_message = "OIDC must pin the audience and exact protected environment subject."
  }
  assert {
    condition     = aws_s3_bucket_versioning.state.versioning_configuration[0].status == "Enabled" && one(one(aws_s3_bucket_server_side_encryption_configuration.state.rule).apply_server_side_encryption_by_default).sse_algorithm == "AES256" && aws_s3_bucket_public_access_block.state.block_public_acls && aws_s3_bucket_public_access_block.state.block_public_policy && aws_s3_bucket_public_access_block.state.ignore_public_acls && aws_s3_bucket_public_access_block.state.restrict_public_buckets && !aws_s3_bucket.state.force_destroy
    error_message = "State must be versioned, private, encrypted and protected from force destruction."
  }
  assert {
    condition     = jsondecode(aws_iam_role_policy.state.policy).Statement[1].Action == ["s3:GetObject", "s3:PutObject"] && jsondecode(aws_iam_role_policy.state.policy).Statement[2].Action == ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"] && endswith(jsondecode(aws_iam_role_policy.state.policy).Statement[2].Resource[0], "dev/terraform.tfstate.tflock")
    error_message = "Delete must be allowed on the lock object only, not state."
  }
  assert {
    condition     = jsondecode(aws_iam_role_policy.deployment.policy).Statement[0].Condition.StringEquals["iam:PermissionsBoundary"] == aws_iam_policy.runtime_boundary.arn && jsondecode(aws_iam_role_policy.deployment.policy).Statement[3].Condition.StringEquals["iam:PassedToService"] == "lambda.amazonaws.com"
    error_message = "App roles require a boundary and PassRole must be restricted to Lambda."
  }
}
run "deployment_limits" {
  command = plan
  assert {
    condition     = length(aws_iam_role_policy.deployment.policy) + length(aws_iam_role_policy.state.policy) <= 10240 && jsondecode(aws_s3_bucket_policy.state_tls.policy).Statement[0].Condition.Bool["aws:SecureTransport"] == "false"
    error_message = "Role inline policies must fit the IAM quota; the bucket must deny non-TLS access."
  }
}
run "main_branch_subject" {
  command = plan
  variables { github_environment = null }
  assert {
    condition     = jsondecode(aws_iam_role.deployment.assume_role_policy).Statement[0].Condition.StringEquals["token.actions.githubusercontent.com:sub"] == "repo:TrieuNguyenPhu/shorten-link:ref:refs/heads/main"
    error_message = "Branch mode must allow main exactly; no PR, fork, wildcard or alternate ref."
  }
}
run "reject_wildcard_repository" {
  command = plan
  variables { github_repository = "TrieuNguyenPhu/*" }
  expect_failures = [var.github_repository]
}

run "pin_known_api_ids" {
  command = plan
  variables { http_api_ids = ["api123"] }
  assert {
    condition     = alltrue([for arn in local.api_child_arns : strcontains(arn, "/apis/api123/")])
    error_message = "Known API IDs must scope child resources without tag inheritance assumptions."
  }
}
run "reject_unscoped_state_key" {
  command = plan
  variables { state_key = "bootstrap/terraform.tfstate" }
  expect_failures = [var.state_key]
}
run "reject_wildcard_environment" {
  command = plan
  variables { github_environment = "*" }
  expect_failures = [var.github_environment]
}
run "http_api_tag_permissions" {
  command = plan
  variables { http_api_ids = ["api123"] }
  assert {
    condition     = alltrue([for arn in local.api_tag_arns : strcontains(arn, "api123")]) && one([for statement in jsondecode(aws_iam_role_policy.deployment.policy).Statement : statement if statement.Sid == "ProtectApiOwnershipTags"]).Condition["ForAnyValue:StringEqualsIgnoreCase"]["aws:TagKeys"] == ["Project", "Environment"]
    error_message = "API tag endpoints need ID/path scope and protection against changing ownership tags."
  }
}

run "cognito_has_no_provisioning_or_retag_escape" {
  command = plan
  assert {
    condition     = alltrue([for s in jsondecode(aws_iam_role_policy.deployment.policy).Statement : !contains(s.Action, "cognito-idp:CreateUserPool") && (!contains(s.Action, "cognito-idp:TagResource") || !contains(s.Resource, "*"))])
    error_message = "Deployment must not provision arbitrary pools or claim foreign pools through request tags."
  }
  assert {
    condition     = contains(one([for s in jsondecode(aws_iam_role_policy.deployment.policy).Statement : s if s.Sid == "AppFunctions"]).Action, "lambda:GetFunctionCodeSigningConfig")
    error_message = "The pinned AWS provider requires GetFunctionCodeSigningConfig when refreshing ZIP Lambdas."
  }
}
run "cognito_maintenance_is_pinned" {
  command = plan
  variables { cognito_user_pool_ids = ["ap-southeast-1_owned123"] }
  assert {
    condition     = one([for s in jsondecode(aws_iam_role_policy.deployment.policy).Statement : s if s.Sid == "AppUserPool"]).Resource == ["arn:aws:cognito-idp:ap-southeast-1:111111111111:userpool/ap-southeast-1_owned123"]
    error_message = "Cognito maintenance must use the exact operator-provisioned pool ARN."
  }
  assert {
    condition     = one([for s in jsondecode(aws_iam_role_policy.deployment.policy).Statement : s if s.Sid == "ProtectCognitoOwnershipTags"]).Condition["ForAnyValue:StringEqualsIgnoreCase"]["aws:TagKeys"] == ["Project", "Environment"]
    error_message = "The deployment role must not rewrite ownership tags on its pool."
  }
}
