resource "aws_cloudwatch_log_group" "handler" {
  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = var.log_retention_days
}
resource "aws_iam_role" "execution" {
  name                 = var.function_name
  path                 = var.role_path
  permissions_boundary = var.runtime_boundary_arn
  assume_role_policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [{ Effect = "Allow", Principal = { Service = "lambda.amazonaws.com" }, Action = "sts:AssumeRole" }]
  })
}
resource "aws_iam_role_policy" "execution" {
  name = "runtime"
  role = aws_iam_role.execution.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat([
      { Sid = "WriteOwnLogs", Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = ["${aws_cloudwatch_log_group.handler.arn}:*"] },
      # These X-Ray write operations do not support resource-level permissions.
      { Sid = "WriteTraces", Effect = "Allow", Action = ["xray:PutTraceSegments", "xray:PutTelemetryRecords"], Resource = ["*"] }
    ], [for permission in var.permissions : { Effect = "Allow", Action = permission.actions, Resource = permission.resources }])
  })
}
resource "aws_lambda_function" "handler" {
  function_name    = var.function_name
  role             = aws_iam_role.execution.arn
  filename         = var.filename
  source_code_hash = filebase64sha256(var.filename)
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  memory_size      = 128
  timeout          = 5
  publish          = true
  environment { variables = var.environment_variables }
  tracing_config { mode = "Active" }
  depends_on = [aws_iam_role_policy.execution, aws_cloudwatch_log_group.handler]
}
resource "aws_lambda_alias" "live" {
  count            = var.canary_managed ? 1 : 0
  name             = "live"
  function_name    = aws_lambda_function.handler.function_name
  function_version = aws_lambda_function.handler.version
  lifecycle { ignore_changes = [function_version, routing_config] }
}
resource "aws_lambda_alias" "current" {
  count            = var.canary_managed ? 0 : 1
  name             = "live"
  function_name    = aws_lambda_function.handler.function_name
  function_version = aws_lambda_function.handler.version
}
