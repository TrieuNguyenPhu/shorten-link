locals {
  name        = "${var.name_prefix}-moderation"
  table_parts = split(":", var.table_arn)
  arn         = "arn:${local.table_parts[1]}:states:${local.table_parts[3]}:${local.table_parts[4]}:stateMachine:${local.name}"
}
resource "aws_iam_role" "execution" {
  name                 = "${var.name_prefix}-moderation-sfn"
  path                 = "/url-shortener/dev/"
  permissions_boundary = var.moderation_boundary_arn
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{ Effect = "Allow", Action = "sts:AssumeRole", Principal = { Service = "states.amazonaws.com" }, Condition = {
      StringEquals = { "aws:SourceAccount" = local.table_parts[4] }
      ArnEquals    = { "aws:SourceArn" = local.arn }
    } }]
  })
}
resource "aws_iam_role_policy" "execution" {
  name = "moderation"
  role = aws_iam_role.execution.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Effect = "Allow", Action = ["lambda:InvokeFunction"], Resource = [var.moderate_check_arn] },
      { Effect = "Allow", Action = ["dynamodb:UpdateItem"], Resource = [var.table_arn] }
    ]
  })
}
resource "aws_sfn_state_machine" "moderation" {
  name       = local.name
  type       = "STANDARD"
  role_arn   = aws_iam_role.execution.arn
  definition = templatefile("${path.module}/../../../../statemachine/moderation.asl.json", { ModerateCheckArn = var.moderate_check_arn, TableName = var.table_name })
  depends_on = [aws_iam_role_policy.execution]
}
