mock_provider "aws" {
  override_during = plan
  mock_resource "aws_s3_bucket" { defaults = { arn = "arn:aws:s3:::url-shortener-test-state" } }
  mock_resource "aws_iam_policy" { defaults = { arn = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-runtime-boundary" } }
  mock_data "aws_partition" { defaults = { partition = "aws" } }
}
variables {
  aws_account_id    = "111111111111"
  state_bucket_name = "url-shortener-test-state"
}
run "aggregator_role_and_boundary" {
  command = plan
  assert {
    condition     = contains(local.handlers, "click-aggregator") && length(local.handlers) == 5 && contains(local.role_arns, "arn:aws:iam::111111111111:role/url-shortener/dev/url-shortener-dev-click-aggregator")
    error_message = "Only the five implemented Lambda roles may be created/passed."
  }
  assert {
    condition     = contains(jsondecode(aws_iam_policy.runtime_boundary.policy).Statement[0].Action, "dynamodb:UpdateItem") && one([for s in jsondecode(aws_iam_policy.runtime_boundary.policy).Statement : s if contains(s.Action, "sqs:ReceiveMessage")]).Resource == ["arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickQueue"]
    error_message = "The boundary must allow atomic counter writes and polling only ClickQueue."
  }
}
run "pipeline_deployment_scopes" {
  command = plan
  assert {
    condition     = one([for s in jsondecode(aws_iam_policy.event_pipeline_deployment.policy).Statement : s if s.Sid == "AppClickRule"]).Resource == ["arn:aws:events:ap-southeast-1:111111111111:rule/url-shortener-events/url-shortener-dev-link-clicked"] && one([for s in jsondecode(aws_iam_policy.event_pipeline_deployment.policy).Statement : s if s.Sid == "AppClickQueues"]).Resource == ["arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickQueue", "arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickDLQ"]
    error_message = "Rule and queue maintenance must use exact names in the target account/region."
  }
}
run "pinned_mapping_maintenance" {
  command = plan
  variables { click_event_source_mapping_ids = ["12345678-1234-1234-1234-123456789012"] }
  assert {
    condition     = one([for s in jsondecode(aws_iam_policy.event_pipeline_deployment.policy).Statement : s if s.Sid == "AppClickMapping"]).Resource == ["arn:aws:lambda:ap-southeast-1:111111111111:event-source-mapping:12345678-1234-1234-1234-123456789012"] && one([for s in jsondecode(aws_iam_policy.event_pipeline_deployment.policy).Statement : s if s.Sid == "AppClickMapping"]).Condition.ArnEquals["lambda:FunctionArn"] == "arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-click-aggregator:live"
    error_message = "ESM maintenance must pin both operator-provisioned UUID and aggregator live function."
  }
  assert {
    condition     = alltrue([for s in jsondecode(aws_iam_policy.event_pipeline_deployment.policy).Statement : !contains(s.Action, "lambda:CreateEventSourceMapping")]) && one([for s in jsondecode(aws_iam_policy.event_pipeline_deployment.policy).Statement : s if s.Sid == "AppClickMappingTags"]).Resource == ["arn:aws:lambda:ap-southeast-1:111111111111:event-source-mapping:12345678-1234-1234-1234-123456789012"] && one([for s in jsondecode(aws_iam_policy.event_pipeline_deployment.policy).Statement : s if s.Sid == "ProtectMappingOwnershipTags"]).Condition["ForAnyValue:StringEqualsIgnoreCase"]["aws:TagKeys"] == ["Project", "Environment"]
    error_message = "Initial ESM provisioning is operator-owned; mapping tagging must not claim arbitrary mappings."
  }
  assert {
    condition     = length(aws_iam_role_policy.deployment.policy) + length(aws_iam_role_policy.state.policy) <= 10240 && length(aws_iam_policy.event_pipeline_deployment.policy) <= 6144 && aws_iam_role_policy_attachment.event_pipeline.policy_arn == aws_iam_policy.event_pipeline_deployment.arn && aws_iam_role_policy_attachment.event_pipeline.role == aws_iam_role.deployment.name
    error_message = "Fully pinned deployment policies must fit the IAM inline policy quota."
  }
}
run "reject_mapping_wildcard" {
  command = plan
  variables { click_event_source_mapping_ids = ["*"] }
  expect_failures = [var.click_event_source_mapping_ids]
}
