locals {
  routes = {
    create-link  = { key = "POST /links", method = "POST", path = "links", private = true }
    resolve-link = { key = "GET /{code}", method = "GET", path = "*", private = false }
    list-links   = { key = "GET /links", method = "GET", path = "links", private = true }
    delete-link  = { key = "DELETE /links/{code}", method = "DELETE", path = "links/*", private = true }
  }
}
resource "aws_apigatewayv2_api" "http" {
  name          = var.name
  protocol_type = "HTTP"
  cors_configuration {
    allow_origins = var.cors_origins
    allow_headers = ["authorization", "content-type"]
    allow_methods = ["GET", "POST", "DELETE", "OPTIONS"]
    max_age       = 300
  }
}
resource "aws_apigatewayv2_authorizer" "cognito" {
  api_id           = aws_apigatewayv2_api.http.id
  name             = "cognito-access-token"
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]
  jwt_configuration {
    issuer   = var.jwt_issuer
    audience = [var.jwt_audience]
  }
}
resource "aws_apigatewayv2_integration" "lambda" {
  for_each               = local.routes
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  integration_uri        = var.lambda_targets[each.key].alias_invoke_arn
  payload_format_version = "2.0"
  timeout_milliseconds   = 6000
}
resource "aws_apigatewayv2_route" "routes" {
  for_each             = local.routes
  api_id               = aws_apigatewayv2_api.http.id
  route_key            = each.value.key
  target               = "integrations/${aws_apigatewayv2_integration.lambda[each.key].id}"
  authorization_type   = each.value.private ? "JWT" : "NONE"
  authorizer_id        = each.value.private ? aws_apigatewayv2_authorizer.cognito.id : null
  authorization_scopes = each.value.private ? ["openid"] : []
}
resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/apigateway/${var.name}"
  retention_in_days = var.log_retention_days
}
resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
  default_route_settings {
    throttling_burst_limit = 20
    throttling_rate_limit  = 10
  }
  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api.arn
    format          = jsonencode({ requestId = "$context.requestId", routeKey = "$context.routeKey", status = "$context.status", integrationError = "$context.integrationErrorMessage" })
  }
}
resource "aws_lambda_permission" "api" {
  for_each       = local.routes
  statement_id   = "AllowHttpApi"
  action         = "lambda:InvokeFunction"
  function_name  = var.lambda_targets[each.key].function_name
  qualifier      = "live"
  principal      = "apigateway.amazonaws.com"
  source_account = data.aws_caller_identity.current.account_id
  source_arn     = "${aws_apigatewayv2_api.http.execution_arn}/*/${each.value.method}/${each.value.path}"
}
data "aws_caller_identity" "current" {}
