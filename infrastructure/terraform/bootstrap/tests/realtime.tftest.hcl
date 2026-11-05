mock_provider "aws" {
  override_during = plan
  mock_resource "aws_s3_bucket" { defaults = { arn = "arn:aws:s3:::url-shortener-test-state" } }
  mock_resource "aws_iam_policy" { defaults = { arn = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-runtime-boundary" } }
  mock_data "aws_partition" { defaults = { partition = "aws" } }
}
variables {
  aws_account_id    = "111111111111"
  state_bucket_name = "url-shortener-test-state"
  http_api_ids      = []
  websocket_api_ids = []
}
run "realtime_boundaries_and_passrole" {
  command = plan
  variables {
    http_api_ids      = ["http123"]
    websocket_api_ids = ["ws123"]
  }
  assert {
    condition     = length(local.handlers) == 8 && alltrue([for h in ["ws-connect", "ws-disconnect", "moderate-check"] : contains(local.handlers, h)]) && one([for s in jsondecode(aws_iam_policy.runtime_boundary.policy).Statement : s if contains(s.Action, "execute-api:ManageConnections")]).Resource == ["arn:aws:execute-api:ap-southeast-1:111111111111:ws123/prod/POST/@connections/*"]
    error_message = "All eight Lambdas need bounded deployment; push boundary must pin API/prod/POST only."
  }
  assert {
    condition     = length(jsondecode(aws_iam_policy.moderation_boundary.policy).Statement) == 2 && jsondecode(aws_iam_policy.moderation_boundary.policy).Statement[0].Resource == ["arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-moderate-check:live"] && jsondecode(aws_iam_policy.moderation_boundary.policy).Statement[1].Action == ["dynamodb:UpdateItem"] && jsondecode(aws_iam_policy.moderation_boundary.policy).Statement[1].Resource == [local.table_arn]
    error_message = "Dedicated moderation boundary must not authorize other functions, tables or services."
  }
  assert {
    condition     = one([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : s if s.Sid == "PassModerationRole"]).Resource == [local.moderation_role_arn] && one([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : s if s.Sid == "PassModerationRole"]).Condition.StringEquals["iam:PassedToService"] == "states.amazonaws.com" && one([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : s if s.Sid == "CreateModerationRole"]).Condition.StringEquals["iam:PermissionsBoundary"] == aws_iam_policy.moderation_boundary.arn
    error_message = "Step Functions role creation and PassRole must use a distinct boundary/service."
  }
  assert {
    condition     = alltrue([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : !contains(s.Action, "states:StartExecution") && !contains(s.Action, "iam:CreatePolicyVersion") && (!contains(s.Resource, "*") || s.Sid == "ValidateModerationDefinition")]) && one([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : s if s.Sid == "AppModeration"]).Resource == ["arn:aws:states:ap-southeast-1:111111111111:stateMachine:url-shortener-dev-moderation"]
    error_message = "Deployment may manage only the exact machine; execution remains operator-owned."
  }
}
run "unpinned_websocket_has_no_push_grant" {
  command = plan
  assert {
    condition     = alltrue([for s in jsondecode(aws_iam_policy.runtime_boundary.policy).Statement : !contains(s.Action, "execute-api:ManageConnections")])
    error_message = "An unknown WebSocket API ID must not turn into wildcard push permission."
  }
}
run "pinned_all_services_fit_quotas" {
  command = plan
  variables {
    http_api_ids                   = ["http123"]
    websocket_api_ids              = ["ws123"]
    cognito_user_pool_ids          = ["ap-southeast-1_owned123"]
    click_event_source_mapping_ids = ["12345678-1234-1234-1234-123456789012"]
  }
  assert {
    condition     = length(aws_iam_role_policy.deployment.policy) + length(aws_iam_role_policy.state.policy) <= 10240 && alltrue([for p in [aws_iam_policy.lambda_deployment.policy, aws_iam_policy.api_deployment.policy, aws_iam_policy.realtime_deployment.policy, aws_iam_policy.event_pipeline_deployment.policy, aws_iam_policy.runtime_boundary.policy, aws_iam_policy.moderation_boundary.policy] : length(p) <= 6144])
    error_message = "All inline and managed policies must fit IAM quotas with every service ID pinned."
  }
}
run "reject_websocket_wildcard" {
  command = plan
  variables { websocket_api_ids = ["*"] }
  expect_failures = [var.websocket_api_ids]
}
run "provider_definition_validation_and_refresh" {
  command = plan
  assert {
    condition     = contains(one([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : s if s.Sid == "AppModeration"]).Action, "states:ListStateMachineVersions") && one([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : s if s.Sid == "ValidateModerationDefinition"]).Action == ["states:ValidateStateMachineDefinition"] && one([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : s if s.Sid == "ValidateModerationDefinition"]).Resource == ["*"] && one([for s in jsondecode(aws_iam_policy.realtime_deployment.policy).Statement : s if s.Sid == "ValidateModerationDefinition"]).Condition.StringEquals["aws:RequestedRegion"] == var.aws_region
    error_message = "Provider 6.71.0 requires definition validation during plan and version listing during refresh."
  }
}
