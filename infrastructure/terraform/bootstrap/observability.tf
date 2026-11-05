locals {
  canary_function_arn = "${local.base_arn}:lambda:${var.aws_region}:${var.aws_account_id}:function:${var.name_prefix}-resolve-link"
  canary_role_arn     = "${local.base_arn}:iam::${var.aws_account_id}:role/url-shortener/dev/${var.name_prefix}-codedeploy"
  canary_app_arn      = "${local.base_arn}:codedeploy:${var.aws_region}:${var.aws_account_id}:application:${var.name_prefix}-resolve"
  canary_group_arn    = "${local.base_arn}:codedeploy:${var.aws_region}:${var.aws_account_id}:deploymentgroup:${var.name_prefix}-resolve/${var.name_prefix}-resolve-live"
  canary_config_arn   = "${local.base_arn}:codedeploy:${var.aws_region}:${var.aws_account_id}:deploymentconfig:CodeDeployDefault.LambdaCanary10Percent5Minutes"
  alarm_arns          = [for suffix in ["resolve-errors", "resolve-throttles", "click-dlq", "click-publish-failed"] : "${local.base_arn}:cloudwatch:${var.aws_region}:${var.aws_account_id}:alarm:${var.name_prefix}-${suffix}"]
  alarm_topic_arn     = "${local.base_arn}:sns:${var.aws_region}:${var.aws_account_id}:${var.name_prefix}-alarms"
  dashboard_arn       = "${local.base_arn}:cloudwatch::${var.aws_account_id}:dashboard/${var.name_prefix}"
}
resource "aws_iam_policy" "canary_boundary" {
  name = "${var.name_prefix}-canary-boundary"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({ Version = "2012-10-17", Statement = [
    { Effect = "Allow", Action = ["lambda:GetAlias", "lambda:UpdateAlias", "lambda:GetProvisionedConcurrencyConfig"], Resource = ["${local.canary_function_arn}:live"] },
    { Effect = "Allow", Action = ["cloudwatch:DescribeAlarms"], Resource = ["*"] }
  ] })
}
resource "aws_iam_policy" "operations_deployment" {
  name = "${var.name_prefix}-operations-deploy"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({ Version = "2012-10-17", Statement = [
    { Sid = "CreateCanaryRole", Effect = "Allow", Action = ["iam:CreateRole", "iam:PutRolePermissionsBoundary"], Resource = [local.canary_role_arn], Condition = { StringEquals = { "iam:PermissionsBoundary" = aws_iam_policy.canary_boundary.arn } } },
    { Sid = "ManageCanaryRole", Effect = "Allow", Action = ["iam:GetRole", "iam:DeleteRole", "iam:UpdateRole", "iam:UpdateAssumeRolePolicy", "iam:PutRolePolicy", "iam:GetRolePolicy", "iam:DeleteRolePolicy", "iam:ListRolePolicies", "iam:ListAttachedRolePolicies", "iam:ListInstanceProfilesForRole", "iam:TagRole", "iam:UntagRole", "iam:ListRoleTags"], Resource = [local.canary_role_arn] },
    { Sid = "KeepCanaryBoundary", Effect = "Deny", Action = ["iam:DeleteRolePermissionsBoundary"], Resource = [local.canary_role_arn] },
    { Sid = "PassCanaryRole", Effect = "Allow", Action = ["iam:PassRole"], Resource = [local.canary_role_arn], Condition = { StringEquals = { "iam:PassedToService" = "codedeploy.amazonaws.com" } } },
    { Sid = "ReadCanaryBoundary", Effect = "Allow", Action = ["iam:GetPolicy", "iam:GetPolicyVersion"], Resource = [aws_iam_policy.canary_boundary.arn] },
    { Sid = "CanaryApplication", Effect = "Allow", Action = ["codedeploy:CreateApplication", "codedeploy:GetApplication", "codedeploy:DeleteApplication", "codedeploy:ListTagsForResource", "codedeploy:TagResource", "codedeploy:UntagResource", "codedeploy:RegisterApplicationRevision", "codedeploy:GetApplicationRevision"], Resource = [local.canary_app_arn] },
    { Sid = "CanaryGroup", Effect = "Allow", Action = ["codedeploy:CreateDeploymentGroup", "codedeploy:UpdateDeploymentGroup", "codedeploy:GetDeploymentGroup", "codedeploy:DeleteDeploymentGroup", "codedeploy:CreateDeployment", "codedeploy:GetDeployment", "codedeploy:ListDeployments", "codedeploy:ListTagsForResource", "codedeploy:TagResource", "codedeploy:UntagResource"], Resource = [local.canary_group_arn] },
    { Sid = "CanaryConfig", Effect = "Allow", Action = ["codedeploy:GetDeploymentConfig"], Resource = [local.canary_config_arn] },
    { Sid = "AppAlarms", Effect = "Allow", Action = ["cloudwatch:PutMetricAlarm", "cloudwatch:DeleteAlarms", "cloudwatch:DescribeAlarms", "cloudwatch:ListTagsForResource", "cloudwatch:TagResource", "cloudwatch:UntagResource"], Resource = local.alarm_arns },
    { Sid = "AppDashboard", Effect = "Allow", Action = ["cloudwatch:PutDashboard", "cloudwatch:GetDashboard", "cloudwatch:DeleteDashboards"], Resource = [local.dashboard_arn] },
    { Sid = "AlarmSubscriptions", Effect = "Allow", Action = ["sns:GetSubscriptionAttributes", "sns:SetSubscriptionAttributes", "sns:Unsubscribe"], Resource = ["${local.alarm_topic_arn}:*"] },
    { Sid = "AppAlarmTopic", Effect = "Allow", Action = ["sns:CreateTopic", "sns:DeleteTopic", "sns:GetTopicAttributes", "sns:SetTopicAttributes", "sns:ListTagsForResource", "sns:TagResource", "sns:UntagResource", "sns:Subscribe", "sns:ListSubscriptionsByTopic"], Resource = [local.alarm_topic_arn] }
  ] })
}
resource "aws_iam_role_policy_attachment" "operations" {
  role       = aws_iam_role.deployment.name
  policy_arn = aws_iam_policy.operations_deployment.arn
}
