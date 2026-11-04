# Keep pipeline deployment grants outside the role's 10 KiB inline quota.
# Bootstrap is operator-owned; the application role cannot edit this policy.
resource "aws_iam_policy" "event_pipeline_deployment" {
  name = "${var.name_prefix}-event-pipeline-deploy"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat([
      { Sid = "AppClickRule", Effect = "Allow", Action = ["events:PutRule", "events:DescribeRule", "events:DeleteRule", "events:PutTargets", "events:RemoveTargets", "events:ListTargetsByRule", "events:ListTagsForResource", "events:TagResource", "events:UntagResource"], Resource = [local.click_rule_arn] },
      { Sid = "AppClickQueues", Effect = "Allow", Action = ["sqs:CreateQueue", "sqs:GetQueueUrl", "sqs:GetQueueAttributes", "sqs:SetQueueAttributes", "sqs:DeleteQueue", "sqs:ListQueueTags", "sqs:TagQueue", "sqs:UntagQueue"], Resource = local.click_queue_arns },
      ], [for statement in [
        # UUIDs are unknown before creation. Operator provisions once, then pins IDs;
        # the app role cannot claim other ESMs with wildcard TagResource permissions.
        { Sid = "AppClickMapping", Effect = "Allow", Action = ["lambda:GetEventSourceMapping", "lambda:UpdateEventSourceMapping", "lambda:DeleteEventSourceMapping"], Resource = local.click_mapping_arns, Condition = { ArnEquals = { "lambda:FunctionArn" = local.aggregator_alias_arn } } },
        { Sid = "AppClickMappingTags", Effect = "Allow", Action = ["lambda:ListTags", "lambda:TagResource", "lambda:UntagResource"], Resource = local.click_mapping_arns },
        { Sid = "ProtectMappingOwnershipTags", Effect = "Deny", Action = ["lambda:TagResource", "lambda:UntagResource"], Resource = local.click_mapping_arns, Condition = { "ForAnyValue:StringEqualsIgnoreCase" = { "aws:TagKeys" = ["Project", "Environment"] } } }
    ] : statement if length(local.click_mapping_arns) > 0])
  })
}
resource "aws_iam_role_policy_attachment" "event_pipeline" {
  role       = aws_iam_role.deployment.name
  policy_arn = aws_iam_policy.event_pipeline_deployment.arn
}
