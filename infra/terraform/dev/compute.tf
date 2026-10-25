locals {
  functions = {
    create-link  = ["dynamodb:PutItem"]
    resolve-link = ["dynamodb:GetItem"]
    manage-links = ["dynamodb:GetItem", "dynamodb:UpdateItem"]
  }
}
resource "aws_lambda_function" "function" {
  for_each         = local.functions
  function_name    = "${local.name}-${each.key}"
  role             = aws_iam_role.function[each.key].arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = "${path.module}/../../../dist/lambdas/${each.key}.zip"
  source_code_hash = filebase64sha256("${path.module}/../../../dist/lambdas/${each.key}.zip")
  memory_size      = 128
  timeout          = 10
  publish          = true
  environment { variables = { DYNAMODB_TABLE = aws_dynamodb_table.links.name } }
  depends_on = [aws_iam_role_policy.function, aws_cloudwatch_log_group.function]
}
resource "aws_lambda_alias" "live" {
  for_each         = local.functions
  name             = "live"
  function_name    = aws_lambda_function.function[each.key].function_name
  function_version = aws_lambda_function.function[each.key].version
}
