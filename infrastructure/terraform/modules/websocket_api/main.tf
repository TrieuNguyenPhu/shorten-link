locals {
  routes = { ws-connect = "$connect", ws-disconnect = "$disconnect" }
}
resource "aws_apigatewayv2_api" "websocket" {
  name                       = "${var.name}-websocket"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}
resource "aws_apigatewayv2_integration" "lambda" {
  for_each             = local.routes
  api_id               = aws_apigatewayv2_api.websocket.id
  integration_type     = "AWS_PROXY"
  integration_method   = "POST"
  integration_uri      = var.lambda_targets[each.key].alias_invoke_arn
  timeout_milliseconds = 6000
  # WebSocket uses its native requestContext.connectionId envelope, not HTTP v2.
}
resource "aws_apigatewayv2_route" "routes" {
  for_each           = local.routes
  api_id             = aws_apigatewayv2_api.websocket.id
  route_key          = each.value
  authorization_type = "NONE"
  target             = "integrations/${aws_apigatewayv2_integration.lambda[each.key].id}"
}
resource "aws_apigatewayv2_stage" "prod" {
  api_id      = aws_apigatewayv2_api.websocket.id
  name        = "prod"
  auto_deploy = true
  default_route_settings {
    throttling_burst_limit = 20
    throttling_rate_limit  = 10
  }
}
resource "aws_lambda_permission" "api" {
  for_each       = local.routes
  statement_id   = "AllowWebSocketApi"
  action         = "lambda:InvokeFunction"
  function_name  = var.lambda_targets[each.key].function_name
  qualifier      = "live"
  principal      = "apigateway.amazonaws.com"
  source_account = var.aws_account_id
  # API Gateway invokes connect against the stage; route is the final ARN part.
  source_arn = "${aws_apigatewayv2_api.websocket.execution_arn}/${aws_apigatewayv2_stage.prod.name}/${each.value}"
}
