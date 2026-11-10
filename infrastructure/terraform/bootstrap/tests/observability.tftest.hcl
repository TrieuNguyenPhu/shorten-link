mock_provider "aws" {
  override_during = plan
  mock_resource "aws_s3_bucket" { defaults = { arn = "arn:aws:s3:::url-shortener-test-state" } }
  mock_data "aws_partition" { defaults = { partition = "aws" } }
  mock_resource "aws_iam_policy" { defaults = { arn = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-canary-boundary" } }
}
variables {
  aws_account_id    = "111111111111"
  state_bucket_name = "url-shortener-test-state"
}
run "operations_permissions" {
  command = plan
  assert {
    condition     = jsondecode(aws_iam_policy.canary_boundary.policy).Statement[0].Resource == ["*"] && contains(jsondecode(aws_iam_policy.canary_boundary.policy).Statement[0].Action, "lambda:InvokeFunction") && one([for s in jsondecode(aws_iam_policy.operations_deployment.policy).Statement : s if s.Sid == "PassCanaryRole"]).Condition.StringEquals["iam:PassedToService"] == "codedeploy.amazonaws.com" && alltrue([for s in jsondecode(aws_iam_policy.operations_deployment.policy).Statement : !contains(s.Resource, "*")])
    error_message = "The CodeDeploy Lambda boundary must match AWS's service policy; the operator role and PassRole scope remain pinned."
  }
  assert {
    condition     = length(aws_iam_policy.operations_deployment.policy) <= 6144 && length(aws_iam_policy.canary_boundary.policy) <= 6144 && length(aws_iam_role_policy.deployment.policy) + length(aws_iam_role_policy.state.policy) <= 10240
    error_message = "New grants must preserve managed and inline IAM policy quotas."
  }
}
