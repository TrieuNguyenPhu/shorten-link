resource "aws_sqs_queue" "dlq" {
  fifo_queue                = false
  name                      = "${var.name_prefix}-ClickDLQ"
  message_retention_seconds = 1209600
  sqs_managed_sse_enabled   = true
}
resource "aws_sqs_queue" "clicks" {
  fifo_queue                 = false
  name                       = "${var.name_prefix}-ClickQueue"
  visibility_timeout_seconds = 30
  message_retention_seconds  = 345600
  sqs_managed_sse_enabled    = true
  # Keep three receives intentionally for compatibility with the reference.
  redrive_policy = jsonencode({ deadLetterTargetArn = aws_sqs_queue.dlq.arn, maxReceiveCount = 3 })
}
resource "aws_sqs_queue_redrive_allow_policy" "dlq" {
  queue_url = aws_sqs_queue.dlq.url
  redrive_allow_policy = jsonencode({
    redrivePermission = "byQueue"
    sourceQueueArns   = [aws_sqs_queue.clicks.arn]
  })
}
resource "aws_cloudwatch_event_rule" "clicked" {
  name           = "${var.name_prefix}-link-clicked"
  event_bus_name = var.event_bus_name
  event_pattern  = jsonencode({ source = ["urlshortener"], "detail-type" = ["link.clicked"] })
}
resource "aws_sqs_queue_policy" "clicks" {
  queue_url = aws_sqs_queue.clicks.url
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowClickRule", Effect = "Allow", Principal = { Service = "events.amazonaws.com" }
        Action = "sqs:SendMessage", Resource = aws_sqs_queue.clicks.arn
        Condition = {
          ArnEquals    = { "aws:SourceArn" = aws_cloudwatch_event_rule.clicked.arn }
          StringEquals = { "aws:SourceAccount" = split(":", var.event_bus_arn)[4] }
        }
      },
      { Sid = "DenyInsecureTransport", Effect = "Deny", Principal = "*", Action = "sqs:*", Resource = aws_sqs_queue.clicks.arn, Condition = { Bool = { "aws:SecureTransport" = "false" } } }
    ]
  })
}
resource "aws_sqs_queue_policy" "dlq" {
  queue_url = aws_sqs_queue.dlq.url
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Sid = "DenyInsecureTransport", Effect = "Deny", Principal = "*", Action = "sqs:*", Resource = aws_sqs_queue.dlq.arn, Condition = { Bool = { "aws:SecureTransport" = "false" } } }
    ]
  })
}
resource "aws_cloudwatch_event_target" "clicks" {
  target_id      = "ClickQueue"
  rule           = aws_cloudwatch_event_rule.clicked.name
  event_bus_name = var.event_bus_name
  arn            = aws_sqs_queue.clicks.arn
  # Forward the entire EventBridge envelope, including id for deduplication.
  depends_on = [aws_sqs_queue_policy.clicks]
}
locals {
  aggregator_permissions = [
    { actions = ["dynamodb:UpdateItem", "dynamodb:PutItem", "dynamodb:GetItem"], resources = [var.table_arn] },
    { actions = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes"], resources = [aws_sqs_queue.clicks.arn] },
    { actions = ["dynamodb:Query"], resources = ["${var.table_arn}/index/GSI1"] },
    { actions = ["dynamodb:DeleteItem"], resources = [var.table_arn] },
    { actions = ["execute-api:ManageConnections"], resources = [var.ws_connections_arn] }
  ]
}
module "aggregator" {
  source                = "../lambda"
  function_name         = "${var.name_prefix}-click-aggregator"
  filename              = var.aggregator_filename
  role_path             = "/url-shortener/dev/"
  runtime_boundary_arn  = var.runtime_boundary_arn
  permissions           = local.aggregator_permissions
  environment_variables = { TABLE_NAME = var.table_name, WS_ENDPOINT = var.ws_endpoint }
  log_retention_days    = var.log_retention_days
}
resource "aws_lambda_event_source_mapping" "clicks" {
  event_source_arn                   = aws_sqs_queue.clicks.arn
  function_name                      = module.aggregator.alias_arn
  batch_size                         = 10
  maximum_batching_window_in_seconds = 0
  function_response_types            = ["ReportBatchItemFailures"]
  enabled                            = true
  depends_on                         = [module.aggregator, aws_sqs_queue_redrive_allow_policy.dlq]
}
