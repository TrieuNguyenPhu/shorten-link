mock_provider "aws" {
  override_during = plan
  mock_data "aws_caller_identity" { defaults = { account_id = "111111111111" } }
  mock_resource "aws_sns_topic" { defaults = { arn = "arn:aws:sns:ap-southeast-1:111111111111:url-shortener-dev-alarms" } }
  mock_resource "aws_cloudwatch_metric_alarm" { defaults = { arn = "arn:aws:cloudwatch:ap-southeast-1:111111111111:alarm:test" } }
}
run "alarm_contract" {
  command = plan
  module { source = "../../modules/observability" }
  variables {
    name_prefix           = "url-shortener-dev"
    aws_region            = "ap-southeast-1"
    resolve_function_name = "url-shortener-dev-resolve-link"
    queue_name            = "url-shortener-dev-ClickQueue"
    dlq_name              = "url-shortener-dev-ClickDLQ"
  }
  assert {
    condition     = alltrue([for a in aws_cloudwatch_metric_alarm.resolve : a.statistic == "Sum" && a.period == 60 && a.evaluation_periods == 1 && a.threshold == 1 && a.treat_missing_data == "notBreaching" && a.dimensions.FunctionName == var.resolve_function_name])
    error_message = "Preserve both source alarms, Sum >= 1 per minute, on the real resolve function."
  }
  assert {
    condition     = aws_cloudwatch_metric_alarm.dlq.metric_name == "ApproximateNumberOfMessagesVisible" && aws_cloudwatch_metric_alarm.dlq.statistic == "Maximum" && aws_cloudwatch_metric_alarm.dlq.dimensions.QueueName == var.dlq_name && aws_cloudwatch_metric_alarm.publish.dimensions.service == "resolve-link" && aws_cloudwatch_metric_alarm.publish.namespace == "UrlShortener" && length(aws_sns_topic_subscription.email) == 0
    error_message = "DLQ uses a gauge; EMF uses its emitted service dimension; no implicit email recipient."
  }
  assert {
    condition     = jsondecode(aws_sns_topic_policy.alarms.policy).Statement[0].Principal.Service == "cloudwatch.amazonaws.com" && strcontains(aws_cloudwatch_dashboard.operations.dashboard_body, "ClickDiscardedDeleted") && strcontains(aws_cloudwatch_dashboard.operations.dashboard_body, "LinkResolved") && strcontains(aws_cloudwatch_dashboard.operations.dashboard_body, "p99") && strcontains(aws_cloudwatch_dashboard.operations.dashboard_body, "ApproximateAgeOfOldestMessage")
    error_message = "Dashboard must expose latency, backlog and both custom pipeline signals."
  }
}
run "canary_contract" {
  command = plan
  module { source = "../../modules/codedeploy_canary" }
  variables {
    name_prefix  = "url-shortener-dev"
    function_arn = "arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-resolve-link"
    boundary_arn = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-canary-boundary"
    alarm_names  = ["resolve-errors", "resolve-throttles"]
  }
  assert {
    condition     = aws_codedeploy_app.resolve.compute_platform == "Lambda" && aws_codedeploy_deployment_group.resolve.deployment_config_name == "CodeDeployDefault.LambdaCanary10Percent5Minutes" && aws_codedeploy_deployment_group.resolve.alarm_configuration[0].enabled && !aws_codedeploy_deployment_group.resolve.alarm_configuration[0].ignore_poll_alarm_failure && toset(aws_codedeploy_deployment_group.resolve.alarm_configuration[0].alarms) == toset(var.alarm_names) && aws_codedeploy_deployment_group.resolve.auto_rollback_configuration[0].enabled && contains(aws_codedeploy_deployment_group.resolve.auto_rollback_configuration[0].events, "DEPLOYMENT_STOP_ON_ALARM")
    error_message = "Canary must shift 10 percent for five minutes and fail closed with alarm rollback."
  }
  assert {
    condition     = jsondecode(aws_iam_role_policy.service.policy).Statement[0].Resource == ["${var.function_arn}:live"] && aws_iam_role.service.permissions_boundary == var.boundary_arn && jsondecode(aws_iam_role.service.assume_role_policy).Statement[0].Principal.Service == "codedeploy.amazonaws.com"
    error_message = "CodeDeploy may manage only the live resolve alias, using a bounded service role."
  }
  assert {
    condition     = try(jsondecode(aws_iam_role.service.assume_role_policy).Statement[0].Condition.StringEquals["aws:SourceAccount"], "") == "111111111111" && try(jsondecode(aws_iam_role.service.assume_role_policy).Statement[0].Condition.ArnEquals["aws:SourceArn"], "") == "arn:aws:codedeploy:ap-southeast-1:111111111111:deploymentgroup:url-shortener-dev-resolve/url-shortener-dev-resolve-live"
    error_message = "Only the exact deployment group in this account may assume the CodeDeploy role."
  }

}
