locals {
  function_arn_parts   = split(":", var.function_arn)
  deployment_group_arn = "arn:${local.function_arn_parts[1]}:codedeploy:${local.function_arn_parts[3]}:${local.function_arn_parts[4]}:deploymentgroup:${var.name_prefix}-resolve/${var.name_prefix}-resolve-live"
}
resource "aws_codedeploy_app" "resolve" {
  name             = "${var.name_prefix}-resolve"
  compute_platform = "Lambda"
}
resource "aws_iam_role" "service" {
  name                 = "${var.name_prefix}-codedeploy"
  path                 = "/url-shortener/dev/"
  permissions_boundary = var.boundary_arn
  # Derive the exact group ARN without a role/group dependency cycle.
  assume_role_policy = jsonencode({ Version = "2012-10-17", Statement = [{
    Effect = "Allow", Action = "sts:AssumeRole", Principal = { Service = "codedeploy.amazonaws.com" },
    Condition = {
      StringEquals = { "aws:SourceAccount" = local.function_arn_parts[4] }
      ArnEquals    = { "aws:SourceArn" = local.deployment_group_arn }
    }
  }] })
}
resource "aws_iam_role_policy" "service" {
  name = "canary"
  role = aws_iam_role.service.id
  policy = jsonencode({ Version = "2012-10-17", Statement = [
    # AWS CodeDeploy's Lambda service integration evaluates these control-plane actions against Resource *; the role trust policy and deployment group still constrain who can assume/use the role.
    { Effect = "Allow", Action = ["lambda:GetAlias", "lambda:UpdateAlias", "lambda:GetProvisionedConcurrencyConfig", "lambda:PutProvisionedConcurrencyConfig", "lambda:DeleteProvisionedConcurrencyConfig", "lambda:ListProvisionedConcurrencyConfigs", "lambda:GetFunction", "lambda:GetFunctionConfiguration", "lambda:PublishVersion", "lambda:InvokeFunction", "sns:Publish"], Resource = ["*"] },
    # DescribeAlarms requires wildcard resources to include composite alarms.
    { Effect = "Allow", Action = ["cloudwatch:DescribeAlarms"], Resource = ["*"] }
  ] })
}
resource "aws_iam_role_policy_attachment" "service_managed" {
  role       = aws_iam_role.service.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSCodeDeployRoleForLambda"
}
resource "aws_codedeploy_deployment_group" "resolve" {
  app_name               = aws_codedeploy_app.resolve.name
  deployment_group_name  = "${var.name_prefix}-resolve-live"
  service_role_arn       = aws_iam_role.service.arn
  deployment_config_name = "CodeDeployDefault.LambdaCanary10Percent5Minutes"
  deployment_style {
    deployment_type   = "BLUE_GREEN"
    deployment_option = "WITH_TRAFFIC_CONTROL"
  }
  alarm_configuration {
    enabled                   = true
    ignore_poll_alarm_failure = false
    alarms                    = var.alarm_names
  }
  auto_rollback_configuration {
    enabled = true
    events  = ["DEPLOYMENT_FAILURE", "DEPLOYMENT_STOP_ON_ALARM", "DEPLOYMENT_STOP_ON_REQUEST"]
  }
  depends_on = [aws_iam_role_policy.service, aws_iam_role_policy_attachment.service_managed]
}
