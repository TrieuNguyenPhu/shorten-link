resource "aws_iam_role" "function" {
  for_each = local.functions
  name     = "${local.name}-${each.key}"
  assume_role_policy = jsonencode({ Version = "2012-10-17", Statement = [{
    Effect = "Allow", Principal = { Service = "lambda.amazonaws.com" }, Action = "sts:AssumeRole"
  }] })
}
resource "aws_iam_role_policy" "function" {
  for_each = local.functions
  name     = "runtime"
  role     = aws_iam_role.function[each.key].id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat([
      { Effect = "Allow", Action = each.value, Resource = [aws_dynamodb_table.links.arn] },
      { Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = ["${aws_cloudwatch_log_group.function[each.key].arn}:*"] }
      ], each.key == "manage-links" ? [
      { Effect = "Allow", Action = ["dynamodb:Query"], Resource = ["${aws_dynamodb_table.links.arn}/index/GSI1"] }
    ] : [])
  })
}
