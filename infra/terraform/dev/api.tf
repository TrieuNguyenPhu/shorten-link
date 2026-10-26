locals {
  routes = {
    "POST /api/links"          = { function = "create-link", auth = "JWT", scopes = ["shortly-api/links.write"] }
    "GET /api/links"           = { function = "manage-links", auth = "JWT", scopes = ["shortly-api/links.read"] }
    "HEAD /api/links"          = { function = "manage-links", auth = "JWT", scopes = ["shortly-api/links.read"] }
    "GET /api/links/{code}"    = { function = "manage-links", auth = "JWT", scopes = ["shortly-api/links.read"] }
    "HEAD /api/links/{code}"   = { function = "manage-links", auth = "JWT", scopes = ["shortly-api/links.read"] }
    "PATCH /api/links/{code}"  = { function = "manage-links", auth = "JWT", scopes = ["shortly-api/links.write"] }
    "DELETE /api/links/{code}" = { function = "manage-links", auth = "JWT", scopes = ["shortly-api/links.write"] }
    "GET /r/{code}"            = { function = "resolve-link", auth = "NONE", scopes = [] }
    "HEAD /r/{code}"           = { function = "resolve-link", auth = "NONE", scopes = [] }
    "GET /health"              = { function = "manage-links", auth = "NONE", scopes = [] }
    "HEAD /health"             = { function = "manage-links", auth = "NONE", scopes = [] }
  }
}
resource "aws_apigatewayv2_api" "http" {
  name          = local.name
  protocol_type = "HTTP"
  cors_configuration {
    allow_origins  = [var.web_origin]
    allow_methods  = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST"]
    allow_headers  = ["Content-Type", "Authorization"]
    expose_headers = ["Location", "WWW-Authenticate"]
    max_age        = 300
  }
}
resource "aws_apigatewayv2_integration" "lambda" {
  for_each               = local.functions
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  integration_uri        = aws_lambda_alias.live[each.key].invoke_arn
  payload_format_version = "2.0"
  timeout_milliseconds   = 12000
}
resource "aws_apigatewayv2_route" "route" {
  for_each             = local.routes
  api_id               = aws_apigatewayv2_api.http.id
  route_key            = each.key
  target               = "integrations/${aws_apigatewayv2_integration.lambda[each.value.function].id}"
  authorization_type   = each.value.auth
  authorizer_id        = each.value.auth == "JWT" ? aws_apigatewayv2_authorizer.jwt.id : null
  authorization_scopes = each.value.auth == "JWT" ? each.value.scopes : null
}
resource "aws_lambda_permission" "gateway" {
  for_each      = local.routes
  statement_id  = "gateway-${substr(sha1(each.key), 0, 12)}"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.function[each.value.function].function_name
  qualifier     = aws_lambda_alias.live[each.value.function].name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/${replace(replace(each.key, " ", ""), "{code}", "*")}"
}
resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
  default_route_settings {
    throttling_rate_limit  = var.throttle_rate
    throttling_burst_limit = var.throttle_burst
  }
  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.gateway.arn
    format = jsonencode({
      requestId        = "$context.requestId", route = "$context.routeKey", status = "$context.status",
      integrationError = "$context.integrationErrorMessage", latencyMs = "$context.responseLatency"
    })
  }
}
