resource "aws_cloudwatch_log_group" "function" {
  for_each          = local.functions
  name              = "/aws/lambda/${local.name}-${each.key}"
  retention_in_days = var.log_retention_days
}
resource "aws_cloudwatch_log_group" "gateway" {
  name              = "/aws/apigateway/${local.name}"
  retention_in_days = var.log_retention_days
}
resource "aws_cloudwatch_log_metric_filter" "failures" {
  for_each       = local.functions
  name           = "${local.name}-${each.key}-failures"
  log_group_name = aws_cloudwatch_log_group.function[each.key].name
  pattern        = "{ $.status >= 500 }"
  metric_transformation {
    name          = "${each.key}-Failures"
    namespace     = "Shortly/${var.environment}"
    value         = "1"
    default_value = "0"
  }
}
