output "id" { value = aws_apigatewayv2_api.websocket.id }
output "endpoint" { value = "${aws_apigatewayv2_api.websocket.api_endpoint}/${aws_apigatewayv2_stage.prod.name}" }
output "management_endpoint" { value = "${replace(aws_apigatewayv2_api.websocket.api_endpoint, "wss://", "https://")}/${aws_apigatewayv2_stage.prod.name}" }
output "connections_arn" { value = "${aws_apigatewayv2_api.websocket.execution_arn}/${aws_apigatewayv2_stage.prod.name}/POST/@connections/*" }
