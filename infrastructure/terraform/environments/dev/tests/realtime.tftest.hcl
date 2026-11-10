mock_provider "aws" {
  override_during = plan
  mock_resource "aws_apigatewayv2_api" { defaults = { id = "ws123", api_endpoint = "wss://ws123.execute-api.ap-southeast-1.amazonaws.com", execution_arn = "arn:aws:execute-api:ap-southeast-1:111111111111:ws123" } }
  mock_resource "aws_sfn_state_machine" { defaults = { arn = "arn:aws:states:ap-southeast-1:111111111111:stateMachine:url-shortener-dev-moderation" } }
  mock_resource "aws_iam_role" { defaults = { arn = "arn:aws:iam::111111111111:role/url-shortener/dev/url-shortener-dev-moderation-sfn" } }
  mock_resource "aws_cloudwatch_log_group" { defaults = { arn = "arn:aws:logs:ap-southeast-1:111111111111:log-group:test" } }
  mock_resource "aws_lambda_function" { defaults = { arn = "arn:aws:lambda:ap-southeast-1:111111111111:function:test", version = "1" } }
  mock_resource "aws_lambda_alias" { defaults = { arn = "arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-moderate-check:live" } }
  mock_resource "aws_dynamodb_table" { defaults = { arn = "arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener" } }
  mock_resource "aws_cloudwatch_event_bus" { defaults = { arn = "arn:aws:events:ap-southeast-1:111111111111:event-bus/url-shortener-events" } }
  mock_resource "aws_sqs_queue" { defaults = { arn = "arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickQueue", url = "https://sqs.ap-southeast-1.amazonaws.com/111111111111/url-shortener-dev-ClickQueue" } }
  mock_resource "aws_cognito_user_pool" { defaults = { endpoint = "cognito-idp.ap-southeast-1.amazonaws.com/ap-southeast-1_test" } }
  mock_data "aws_caller_identity" { defaults = { account_id = "111111111111" } }
}
variables {
  aws_account_id          = "111111111111"
  runtime_boundary_arn    = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-runtime-boundary"
  moderation_boundary_arn = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-moderation-boundary"
  cognito_domain_prefix   = "url-shortener-test"
  google_client_id        = "test.apps.googleusercontent.com"
  google_client_secret    = "test-only-not-a-real-secret"
  callback_urls           = ["http://localhost:3000/auth/callback/"]
  logout_urls             = ["http://localhost:3000/"]
  cors_origins            = ["http://localhost:3000"]
}
run "websocket_contract" {
  command = plan
  module { source = "../../modules/websocket_api" }
  variables {
    name = "url-shortener-dev"
    lambda_targets = {
      ws-connect    = { function_name = "url-shortener-dev-ws-connect", alias_invoke_arn = "arn:aws:apigateway:ap-southeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-ws-connect:live/invocations" }
      ws-disconnect = { function_name = "url-shortener-dev-ws-disconnect", alias_invoke_arn = "arn:aws:apigateway:ap-southeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-ws-disconnect:live/invocations" }
    }
  }
  assert {
    condition     = aws_apigatewayv2_api.websocket.protocol_type == "WEBSOCKET" && aws_apigatewayv2_api.websocket.route_selection_expression == "$request.body.action" && aws_apigatewayv2_stage.prod.name == "prod" && aws_apigatewayv2_stage.prod.auto_deploy && toset([for r in aws_apigatewayv2_route.routes : r.route_key]) == toset(["$connect", "$disconnect"]) && alltrue([for r in aws_apigatewayv2_route.routes : r.authorization_type == "NONE"])
    error_message = "Public WebSocket connect/disconnect and auto-deployed prod stage must preserve the contract."
  }
  assert {
    condition     = alltrue([for i in aws_apigatewayv2_integration.lambda : i.integration_type == "AWS_PROXY" && i.integration_method == "POST" && strcontains(i.integration_uri, ":live/invocations") && i.payload_format_version != "2.0"]) && alltrue([for key, p in aws_lambda_permission.api : p.qualifier == "live" && p.source_account == var.aws_account_id && p.source_arn == "arn:aws:execute-api:ap-southeast-1:111111111111:ws123/prod/${local.routes[key]}"])
    error_message = "Proxy integrations must preserve the WebSocket envelope and exact route/live alias invoke permissions."
  }
  assert {
    condition     = output.endpoint == "wss://ws123.execute-api.ap-southeast-1.amazonaws.com/prod" && output.management_endpoint == "https://ws123.execute-api.ap-southeast-1.amazonaws.com/prod" && output.connections_arn == "arn:aws:execute-api:ap-southeast-1:111111111111:ws123/prod/POST/@connections/*"
    error_message = "Client and management URLs must use the real prod stage; push permission must allow POST only."
  }
}
run "moderation_contract" {
  command = plan
  module { source = "../../modules/moderation_sfn" }
  variables {
    name_prefix        = "url-shortener-dev"
    table_name         = "url-shortener"
    table_arn          = "arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener"
    moderate_check_arn = "arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-moderate-check:live"
  }
  assert {
    condition     = aws_sfn_state_machine.moderation.type == "STANDARD" && jsondecode(aws_sfn_state_machine.moderation.definition).States.CheckSafety.Resource == var.moderate_check_arn && jsondecode(aws_sfn_state_machine.moderation.definition).States.Activate.Parameters.TableName == var.table_name && jsondecode(aws_sfn_state_machine.moderation.definition).States.MarkError.Type == "Task" && jsondecode(aws_sfn_state_machine.moderation.definition).States.Reject.Parameters.ExpressionAttributeNames["#r"] == "statusReason"
    error_message = "Use the real rendered ASL with direct Lambda alias, existing table and Fail branch."
  }
  assert {
    condition     = length(jsondecode(aws_iam_role_policy.execution.policy).Statement) == 2 && jsondecode(aws_iam_role_policy.execution.policy).Statement[0].Action == ["lambda:InvokeFunction"] && jsondecode(aws_iam_role_policy.execution.policy).Statement[0].Resource == [var.moderate_check_arn] && jsondecode(aws_iam_role_policy.execution.policy).Statement[1].Action == ["dynamodb:UpdateItem"] && jsondecode(aws_iam_role_policy.execution.policy).Statement[1].Resource == [var.table_arn] && aws_iam_role.execution.permissions_boundary == var.moderation_boundary_arn
    error_message = "State machine business role must be bounded to one alias and one table."
  }
  assert {
    condition     = jsondecode(aws_iam_role.execution.assume_role_policy).Statement[0].Principal.Service == "states.amazonaws.com" && jsondecode(aws_iam_role.execution.assume_role_policy).Statement[0].Condition.StringEquals["aws:SourceAccount"] == "111111111111" && jsondecode(aws_iam_role.execution.assume_role_policy).Statement[0].Condition.ArnEquals["aws:SourceArn"] == "arn:aws:states:ap-southeast-1:111111111111:stateMachine:url-shortener-dev-moderation"
    error_message = "Step Functions trust must constrain the account and exact state machine ARN without a dependency cycle."
  }
}
run "realtime_root_wiring" {
  command = plan
  assert {
    condition     = length(module.functions) == 4 && toset(keys(module.realtime_functions)) == toset(["ws-connect", "ws-disconnect", "moderate-check"]) && length(output.lambda_aliases) == 8 && module.event_pipeline.aggregator_environment_variables.WS_ENDPOINT == module.websocket_api.management_endpoint && output.websocket_endpoint == module.websocket_api.endpoint && output.moderation_state_machine_arn == module.moderation_sfn.arn
    error_message = "Expose all eight handlers while wiring the real WebSocket management URL into the aggregator."
  }
  assert {
    condition     = local.realtime_permissions["ws-connect"][0].actions == ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:DeleteItem"] && local.realtime_permissions["ws-disconnect"][0].actions == ["dynamodb:DeleteItem"] && length(local.realtime_permissions["moderate-check"]) == 0 && length(local.permissions["create-link"]) == 2 && length(local.permissions["resolve-link"]) == 2
    error_message = "Each new Lambda must receive only its own permissions; create/redirect must not start moderation."
  }
}

run "aggregator_push_permissions" {
  command = plan
  module { source = "../../modules/event_pipeline" }
  variables {
    name_prefix         = "url-shortener-dev"
    event_bus_name      = "url-shortener-events"
    event_bus_arn       = "arn:aws:events:ap-southeast-1:111111111111:event-bus/url-shortener-events"
    table_name          = "url-shortener"
    table_arn           = "arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener"
    aggregator_filename = "../../../../backend/dist/click-aggregator.zip"
    ws_endpoint         = "https://ws123.execute-api.ap-southeast-1.amazonaws.com/prod"
    ws_connections_arn  = "arn:aws:execute-api:ap-southeast-1:111111111111:ws123/prod/POST/@connections/*"
  }
  assert {
    condition     = one([for p in local.aggregator_permissions : p if contains(p.actions, "execute-api:ManageConnections")]).resources == [var.ws_connections_arn] && one([for p in local.aggregator_permissions : p if contains(p.actions, "dynamodb:Query")]).resources == ["${var.table_arn}/index/GSI1"] && one([for p in local.aggregator_permissions : p if contains(p.actions, "dynamodb:DeleteItem")]).resources == [var.table_arn]
    error_message = "Aggregator needs only the configured API POST, connection cleanup and the sparse GSI query."
  }
}
run "reject_cross_account_push" {
  command = plan
  module { source = "../../modules/event_pipeline" }
  variables {
    name_prefix         = "url-shortener-dev"
    event_bus_name      = "url-shortener-events"
    event_bus_arn       = "arn:aws:events:ap-southeast-1:111111111111:event-bus/url-shortener-events"
    table_name          = "url-shortener"
    table_arn           = "arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener"
    aggregator_filename = "../../../../backend/dist/click-aggregator.zip"
    ws_endpoint         = "https://ws123.execute-api.ap-southeast-1.amazonaws.com/prod"
    ws_connections_arn  = "arn:aws:execute-api:ap-southeast-1:222222222222:ws123/prod/POST/@connections/*"
  }
  expect_failures = [var.ws_endpoint]
}
