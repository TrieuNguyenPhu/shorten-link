locals {
  moderation_alias_arn   = "${local.base_arn}:lambda:${var.aws_region}:${var.aws_account_id}:function:${var.name_prefix}-moderate-check:live"
  moderation_role_arn    = "${local.base_arn}:iam::${var.aws_account_id}:role/url-shortener/dev/${var.name_prefix}-moderation-sfn"
  moderation_machine_arn = "${local.base_arn}:states:${var.aws_region}:${var.aws_account_id}:stateMachine:${var.name_prefix}-moderation"
}
resource "aws_iam_policy" "moderation_boundary" {
  name = "${var.name_prefix}-moderation-boundary"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Effect = "Allow", Action = ["lambda:InvokeFunction"], Resource = [local.moderation_alias_arn] },
      { Effect = "Allow", Action = ["dynamodb:UpdateItem"], Resource = [local.table_arn] }
    ]
  })
}
resource "aws_iam_policy" "realtime_deployment" {
  name = "${var.name_prefix}-realtime-deploy"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Sid = "CreateModerationRole", Effect = "Allow", Action = ["iam:CreateRole", "iam:PutRolePermissionsBoundary"], Resource = [local.moderation_role_arn], Condition = { StringEquals = { "iam:PermissionsBoundary" = aws_iam_policy.moderation_boundary.arn } } },
      { Sid = "ManageModerationRole", Effect = "Allow", Action = ["iam:GetRole", "iam:DeleteRole", "iam:UpdateRole", "iam:UpdateAssumeRolePolicy", "iam:PutRolePolicy", "iam:GetRolePolicy", "iam:DeleteRolePolicy", "iam:ListRolePolicies", "iam:ListAttachedRolePolicies", "iam:ListInstanceProfilesForRole", "iam:TagRole", "iam:UntagRole", "iam:ListRoleTags"], Resource = [local.moderation_role_arn] },
      { Sid = "KeepModerationBoundary", Effect = "Deny", Action = ["iam:DeleteRolePermissionsBoundary"], Resource = [local.moderation_role_arn] },
      { Sid = "PassModerationRole", Effect = "Allow", Action = ["iam:PassRole"], Resource = [local.moderation_role_arn], Condition = { StringEquals = { "iam:PassedToService" = "states.amazonaws.com" } } },
      { Sid = "ReadModerationBoundary", Effect = "Allow", Action = ["iam:GetPolicy", "iam:GetPolicyVersion"], Resource = [aws_iam_policy.moderation_boundary.arn] },
      { Sid = "AppModeration", Effect = "Allow", Action = ["states:CreateStateMachine", "states:DescribeStateMachine", "states:ListStateMachineVersions", "states:UpdateStateMachine", "states:DeleteStateMachine", "states:ListTagsForResource", "states:TagResource", "states:UntagResource"], Resource = [local.moderation_machine_arn] },
      # This plan-time validation API has no resource-level permissions.
      { Sid = "ValidateModerationDefinition", Effect = "Allow", Action = ["states:ValidateStateMachineDefinition"], Resource = ["*"], Condition = { StringEquals = { "aws:RequestedRegion" = var.aws_region } } }
    ]
  })
}
resource "aws_iam_role_policy_attachment" "realtime" {
  role       = aws_iam_role.deployment.name
  policy_arn = aws_iam_policy.realtime_deployment.arn
}
