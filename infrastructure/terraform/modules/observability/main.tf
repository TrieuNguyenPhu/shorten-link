resource "aws_sns_topic" "alarms" {
  name = "${var.name_prefix}-alarms"
}
resource "aws_sns_topic_subscription" "email" {
  for_each  = var.alarm_email_addresses
  topic_arn = aws_sns_topic.alarms.arn
  protocol  = "email"
  endpoint  = each.value
}
# Preserve the source thresholds and function-wide dimensions, including canary traffic.
resource "aws_cloudwatch_metric_alarm" "resolve" {
  for_each            = { errors = "Errors", throttles = "Throttles" }
  alarm_name          = "${var.name_prefix}-resolve-${each.key}"
  namespace           = "AWS/Lambda"
  metric_name         = each.value
  dimensions          = { FunctionName = var.resolve_function_name }
  statistic           = "Sum"
  period              = 60
  evaluation_periods  = 1
  threshold           = 1
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alarms.arn]
}
resource "aws_cloudwatch_metric_alarm" "dlq" {
  alarm_name          = "${var.name_prefix}-click-dlq"
  namespace           = "AWS/SQS"
  metric_name         = "ApproximateNumberOfMessagesVisible"
  dimensions          = { QueueName = var.dlq_name }
  statistic           = "Maximum"
  period              = 60
  evaluation_periods  = 1
  threshold           = 1
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alarms.arn]
}
resource "aws_cloudwatch_metric_alarm" "publish" {
  alarm_name          = "${var.name_prefix}-click-publish-failed"
  namespace           = "UrlShortener"
  metric_name         = "ClickPublishFailed"
  dimensions          = { service = "resolve-link" }
  statistic           = "Sum"
  period              = 60
  evaluation_periods  = 1
  threshold           = 1
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alarms.arn]
}
resource "aws_cloudwatch_dashboard" "operations" {
  dashboard_name = var.name_prefix
  dashboard_body = jsonencode({ widgets = [
    { type = "metric", x = 0, y = 0, width = 12, height = 6, properties = {
      title   = "Resolve invocations / errors / throttles", region = var.aws_region, period = 60, stat = "Sum",
      metrics = [for metric in ["Invocations", "Errors", "Throttles"] : ["AWS/Lambda", metric, "FunctionName", var.resolve_function_name]]
    } },
    { type = "metric", x = 12, y = 0, width = 12, height = 6, properties = {
      title   = "Resolve duration p50 / p99 (ms)", region = var.aws_region, period = 60,
      metrics = [for stat in ["p50", "p99"] : ["AWS/Lambda", "Duration", "FunctionName", var.resolve_function_name, { stat = stat }]]
    } },
    { type = "metric", x = 0, y = 6, width = 12, height = 6, properties = {
      title   = "Click queue backlog / DLQ", region = var.aws_region, period = 60, stat = "Maximum",
      metrics = [for queue in [var.queue_name, var.dlq_name] : ["AWS/SQS", "ApproximateNumberOfMessagesVisible", "QueueName", queue]]
    } },
    { type = "metric", x = 12, y = 6, width = 12, height = 6, properties = {
      title   = "Click queue oldest age (seconds)", region = var.aws_region, period = 60, stat = "Maximum",
      metrics = [["AWS/SQS", "ApproximateAgeOfOldestMessage", "QueueName", var.queue_name]]
    } },
    { type = "metric", x = 0, y = 12, width = 24, height = 6, properties = {
      title = "Accepted publishes / failed publishes / discarded after deletion", region = var.aws_region, period = 60, stat = "Sum",
      metrics = [
        ["UrlShortener", "LinkResolved", "service", "resolve-link"],
        ["UrlShortener", "ClickPublishFailed", "service", "resolve-link"],
        ["UrlShortener", "ClickDiscardedDeleted", "service", "click-aggregator"]
      ]
    } }
  ] })
}

# Allow alarm delivery only from this module's exact alarms, in this account.
data "aws_caller_identity" "current" {}
resource "aws_sns_topic_policy" "alarms" {
  arn = aws_sns_topic.alarms.arn
  policy = jsonencode({ Version = "2012-10-17", Statement = [{
    Sid    = "CloudWatchAlarms", Effect = "Allow", Principal = { Service = "cloudwatch.amazonaws.com" },
    Action = "sns:Publish", Resource = aws_sns_topic.alarms.arn,
    Condition = {
      StringEquals = { "aws:SourceAccount" = data.aws_caller_identity.current.account_id }
      ArnEquals    = { "aws:SourceArn" = concat([for alarm in aws_cloudwatch_metric_alarm.resolve : alarm.arn], [aws_cloudwatch_metric_alarm.dlq.arn, aws_cloudwatch_metric_alarm.publish.arn]) }
    }
  }] })
}
