mock_provider "aws" {
  override_during = plan
  mock_resource "aws_iam_role" { defaults = { arn = "arn:aws:iam::111111111111:role/url-shortener/dev/url-shortener-dev-click-aggregator" } }
  mock_resource "aws_cloudwatch_log_group" { defaults = { arn = "arn:aws:logs:ap-southeast-1:111111111111:log-group:/aws/lambda/url-shortener-dev-click-aggregator" } }
  mock_resource "aws_lambda_function" { defaults = { arn = "arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-click-aggregator", version = "1" } }
  mock_resource "aws_lambda_alias" { defaults = { arn = "arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-click-aggregator:live" } }
  mock_resource "aws_cloudwatch_event_rule" { defaults = { arn = "arn:aws:events:ap-southeast-1:111111111111:rule/url-shortener-events/url-shortener-dev-link-clicked" } }
  override_resource {
    target = aws_sqs_queue.clicks
    values = { arn = "arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickQueue", url = "https://sqs.ap-southeast-1.amazonaws.com/111111111111/url-shortener-dev-ClickQueue" }
  }
  override_resource {
    target = aws_sqs_queue.dlq
    values = { arn = "arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickDLQ", url = "https://sqs.ap-southeast-1.amazonaws.com/111111111111/url-shortener-dev-ClickDLQ" }
  }
}
variables {
  name_prefix          = "url-shortener-dev"
  event_bus_name       = "url-shortener-events"
  event_bus_arn        = "arn:aws:events:ap-southeast-1:111111111111:event-bus/url-shortener-events"
  table_name           = "url-shortener"
  table_arn            = "arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener"
  runtime_boundary_arn = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-runtime-boundary"
  aggregator_filename  = "../../../../backend/dist/click-aggregator.zip"
  ws_endpoint          = "https://ws123.execute-api.ap-southeast-1.amazonaws.com/prod"
  ws_connections_arn   = "arn:aws:execute-api:ap-southeast-1:111111111111:ws123/prod/POST/@connections/*"
}
run "queues_and_redrive" {
  command = plan
  module { source = "../../modules/event_pipeline" }
  assert {
    condition     = aws_sqs_queue.clicks.name == "url-shortener-dev-ClickQueue" && aws_sqs_queue.dlq.name == "url-shortener-dev-ClickDLQ" && !aws_sqs_queue.clicks.fifo_queue && !aws_sqs_queue.dlq.fifo_queue && aws_sqs_queue.clicks.visibility_timeout_seconds == 30 && aws_sqs_queue.dlq.message_retention_seconds == 1209600 && aws_sqs_queue.clicks.sqs_managed_sse_enabled && aws_sqs_queue.dlq.sqs_managed_sse_enabled
    error_message = "Standard queues must preserve names, 30s visibility, DLQ 14d and SQS-managed encryption."
  }
  assert {
    condition     = jsondecode(aws_sqs_queue.clicks.redrive_policy).maxReceiveCount == 3 && jsondecode(aws_sqs_queue.clicks.redrive_policy).deadLetterTargetArn == aws_sqs_queue.dlq.arn && jsondecode(aws_sqs_queue_redrive_allow_policy.dlq.redrive_allow_policy).redrivePermission == "byQueue" && jsondecode(aws_sqs_queue_redrive_allow_policy.dlq.redrive_allow_policy).sourceQueueArns == [aws_sqs_queue.clicks.arn]
    error_message = "Only ClickQueue may redrive to ClickDLQ, after three receives."
  }
}
run "event_rule_and_queue_policy" {
  command = plan
  module { source = "../../modules/event_pipeline" }
  assert {
    condition     = aws_cloudwatch_event_rule.clicked.event_bus_name == var.event_bus_name && jsondecode(aws_cloudwatch_event_rule.clicked.event_pattern).source == ["urlshortener"] && jsondecode(aws_cloudwatch_event_rule.clicked.event_pattern)["detail-type"] == ["link.clicked"] && aws_cloudwatch_event_target.clicks.event_bus_name == var.event_bus_name && aws_cloudwatch_event_target.clicks.rule == aws_cloudwatch_event_rule.clicked.name && aws_cloudwatch_event_target.clicks.arn == aws_sqs_queue.clicks.arn && aws_cloudwatch_event_target.clicks.input == null && aws_cloudwatch_event_target.clicks.input_path == null && length(aws_cloudwatch_event_target.clicks.input_transformer) == 0 && aws_cloudwatch_event_target.clicks.role_arn == null
    error_message = "The existing custom bus must route the complete untransformed click envelope via the queue resource policy."
  }
  assert {
    condition     = jsondecode(aws_sqs_queue_policy.clicks.policy).Statement[0].Principal.Service == "events.amazonaws.com" && jsondecode(aws_sqs_queue_policy.clicks.policy).Statement[0].Action == "sqs:SendMessage" && jsondecode(aws_sqs_queue_policy.clicks.policy).Statement[0].Resource == aws_sqs_queue.clicks.arn && jsondecode(aws_sqs_queue_policy.clicks.policy).Statement[0].Condition.ArnEquals["aws:SourceArn"] == aws_cloudwatch_event_rule.clicked.arn && jsondecode(aws_sqs_queue_policy.clicks.policy).Statement[0].Condition.StringEquals["aws:SourceAccount"] == "111111111111"
    error_message = "Only the exact click rule/account may publish to ClickQueue."
  }
  assert {
    condition     = jsondecode(aws_sqs_queue_policy.clicks.policy).Statement[1].Effect == "Deny" && jsondecode(aws_sqs_queue_policy.clicks.policy).Statement[1].Condition.Bool["aws:SecureTransport"] == "false" && jsondecode(aws_sqs_queue_policy.dlq.policy).Statement[0].Condition.Bool["aws:SecureTransport"] == "false"
    error_message = "Both queues must deny insecure transport."
  }
}
run "aggregator_permissions_and_mapping" {
  command = plan
  module { source = "../../modules/event_pipeline" }
  assert {
    condition     = local.aggregator_permissions[0].actions == ["dynamodb:UpdateItem", "dynamodb:PutItem", "dynamodb:GetItem"] && local.aggregator_permissions[0].resources == [var.table_arn] && local.aggregator_permissions[1].actions == ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes"] && local.aggregator_permissions[1].resources == [aws_sqs_queue.clicks.arn] && length(local.aggregator_permissions) == 5 && module.aggregator.environment_variables == tomap({ TABLE_NAME = "url-shortener", WS_ENDPOINT = var.ws_endpoint })
    error_message = "Aggregator needs transaction actions, consistent Get, ClickQueue polling, and WebSocket push permissions; no invented TransactWriteItems IAM action."
  }
  assert {
    condition     = aws_lambda_event_source_mapping.clicks.event_source_arn == aws_sqs_queue.clicks.arn && aws_lambda_event_source_mapping.clicks.function_name == module.aggregator.alias_arn && aws_lambda_event_source_mapping.clicks.batch_size == 10 && aws_lambda_event_source_mapping.clicks.maximum_batching_window_in_seconds == 0 && aws_lambda_event_source_mapping.clicks.function_response_types == toset(["ReportBatchItemFailures"]) && aws_lambda_event_source_mapping.clicks.enabled && length(aws_lambda_event_source_mapping.clicks.filter_criteria) == 0
    error_message = "The live aggregator alias must consume batch 10 with partial failures, no destructive filtering or batching delay."
  }
}
run "reject_mismatched_bus" {
  command = plan
  module { source = "../../modules/event_pipeline" }
  variables { event_bus_arn = "arn:aws:events:ap-southeast-1:111111111111:event-bus/other" }
  expect_failures = [var.event_bus_arn]
}
