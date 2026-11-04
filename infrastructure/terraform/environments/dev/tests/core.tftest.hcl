mock_provider "aws" {
  override_during = plan
  mock_resource "aws_apigatewayv2_api" { defaults = { api_endpoint = "https://test.execute-api.ap-southeast-1.amazonaws.com", execution_arn = "arn:aws:execute-api:ap-southeast-1:111111111111:test" } }
  mock_data "aws_caller_identity" { defaults = { account_id = "111111111111" } }
  mock_resource "aws_iam_role" { defaults = { arn = "arn:aws:iam::111111111111:role/url-shortener/dev/test" } }
  mock_resource "aws_lambda_function" { defaults = { arn = "arn:aws:lambda:ap-southeast-1:111111111111:function:test", version = "1" } }
  mock_resource "aws_dynamodb_table" { defaults = { arn = "arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener" } }
  mock_resource "aws_cloudwatch_log_group" { defaults = { arn = "arn:aws:logs:ap-southeast-1:111111111111:log-group:test" } }
  mock_resource "aws_cloudwatch_event_bus" { defaults = { arn = "arn:aws:events:ap-southeast-1:111111111111:event-bus/url-shortener-events" } }
  mock_resource "aws_lambda_event_source_mapping" { defaults = { uuid = "12345678-1234-1234-1234-123456789012" } }
  mock_resource "aws_sqs_queue" { defaults = { arn = "arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickQueue", url = "https://sqs.ap-southeast-1.amazonaws.com/111111111111/url-shortener-dev-ClickQueue" } }
  mock_resource "aws_lambda_alias" { defaults = { arn = "arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-click-aggregator:live" } }
  mock_resource "aws_cognito_user_pool" { defaults = { endpoint = "cognito-idp.ap-southeast-1.amazonaws.com/ap-southeast-1_test" } }
}
variables {
  aws_account_id          = "111111111111"
  moderation_boundary_arn = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-moderation-boundary"
  runtime_boundary_arn    = "arn:aws:iam::111111111111:policy/url-shortener/bootstrap/url-shortener-dev-runtime-boundary"
  cognito_domain_prefix   = "url-shortener-test"
  google_client_id        = "test-client.apps.googleusercontent.com"
  google_client_secret    = "test-only-not-a-real-secret"
  callback_urls           = ["http://localhost:3000/auth/callback/"]
  logout_urls             = ["http://localhost:3000/"]
  cors_origins            = ["http://localhost:3000"]
}
run "core_wiring" {
  command = plan
  assert {
    condition     = toset(keys(module.functions)) == toset(["create-link", "resolve-link", "list-links", "delete-link"]) && module.functions["resolve-link"].environment_variables.EVENT_BUS == "url-shortener-events" && module.functions["create-link"].environment_variables.TABLE_NAME == "url-shortener"
    error_message = "The four real handlers must receive the real table and custom event bus."
  }
  assert {
    condition     = module.functions["create-link"].environment_variables.PUBLIC_REDIRECT_BASE_URL == module.http_api.endpoint
    error_message = "Short URLs must use the redirect API endpoint, not a frontend domain."
  }
}
run "pipeline_root_wiring" {
  command = plan
  assert {
    condition     = module.event_pipeline.event_bus_name == module.event_bus.name && module.event_pipeline.event_bus_arn == module.event_bus.arn && toset(keys(output.lambda_roles)) == toset(["create-link", "resolve-link", "list-links", "delete-link", "click-aggregator", "ws-connect", "ws-disconnect", "moderate-check"]) && output.lambda_aliases["click-aggregator"] == module.event_pipeline.aggregator_alias_arn && output.click_queue_arn == module.event_pipeline.click_queue_arn && output.click_event_source_mapping_id == module.event_pipeline.event_source_mapping_id
    error_message = "The existing bus and eight Lambdas must be exposed without changing the four HTTP targets."
  }
}
run "per_handler_permissions" {
  command = plan
  assert {
    condition     = local.permissions["create-link"][0].actions == ["dynamodb:PutItem"] && local.permissions["list-links"][0].actions == ["dynamodb:Query"] && local.permissions["list-links"][0].resources == ["arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener/index/GSI1"] && local.permissions["delete-link"][0].actions == ["dynamodb:DeleteItem"] && local.permissions["resolve-link"][1].actions == ["events:PutEvents"] && length(local.permissions["resolve-link"]) == 2
    error_message = "Business permissions must match each handler; only resolve can publish clicks."
  }
}
run "dynamodb_contract" {
  command = plan
  module { source = "../../modules/dynamodb" }
  variables { table_name = "url-shortener" }
  assert {
    condition     = aws_dynamodb_table.links.billing_mode == "PAY_PER_REQUEST" && aws_dynamodb_table.links.hash_key == "PK" && aws_dynamodb_table.links.range_key == "SK" && one(aws_dynamodb_table.links.global_secondary_index).name == "GSI1" && one(aws_dynamodb_table.links.global_secondary_index).projection_type == "ALL" && one(aws_dynamodb_table.links.global_secondary_index).key_schema[0].attribute_name == "GSI1PK" && one(aws_dynamodb_table.links.global_secondary_index).key_schema[1].attribute_name == "GSI1SK" && aws_dynamodb_table.links.ttl[0].attribute_name == "ttl" && aws_dynamodb_table.links.ttl[0].enabled
    error_message = "Single table keys, GSI1, TTL and on-demand billing must match the contract."
  }
}
run "google_only_auth" {
  command = plan
  module { source = "../../modules/cognito" }
  variables { name = "url-shortener-dev" }
  assert {
    condition     = aws_cognito_user_pool_client.web.supported_identity_providers == toset(["Google"]) && aws_cognito_user_pool_client.web.allowed_oauth_flows == toset(["code"]) && !aws_cognito_user_pool_client.web.generate_secret && aws_cognito_user_pool_client.web.explicit_auth_flows == toset(["ALLOW_REFRESH_TOKEN_AUTH"]) && aws_cognito_user_pool.pool.admin_create_user_config[0].allow_admin_create_user_only
    error_message = "Only Google + authorization code for a public client; no password or self-signup flow."
  }
  assert {
    condition     = aws_cognito_user_pool_client.web.allowed_oauth_scopes == toset(["openid", "email", "profile"]) && aws_cognito_identity_provider.google.attribute_mapping["email"] == "email"
    error_message = "OAuth scopes and Google attribute mapping must support browser federation."
  }
}
run "lambda_contract" {
  command = plan
  module { source = "../../modules/lambda" }
  variables {
    function_name         = "url-shortener-dev-resolve-link"
    role_path             = "/url-shortener/dev/"
    filename              = "../../../../backend/dist/resolve-link.zip"
    environment_variables = { TABLE_NAME = "url-shortener", EVENT_BUS = "url-shortener-events" }
    permissions           = [{ actions = ["dynamodb:GetItem"], resources = ["arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener"] }]
    canary_managed        = true
  }
  assert {
    condition     = aws_lambda_function.handler.runtime == "provided.al2023" && aws_lambda_function.handler.architectures == tolist(["arm64"]) && aws_lambda_function.handler.handler == "bootstrap" && aws_lambda_function.handler.memory_size == 128 && aws_lambda_function.handler.timeout == 5 && aws_lambda_function.handler.publish && aws_lambda_function.handler.tracing_config[0].mode == "Active" && aws_iam_role.execution.permissions_boundary == var.runtime_boundary_arn && aws_lambda_alias.live[0].name == "live"
    error_message = "Lambda must publish arm64 AL2023 bootstrap with bounded IAM and a live alias."
  }
}
run "http_contract" {
  command = plan
  module { source = "../../modules/http_api" }
  variables {
    name         = "url-shortener-dev"
    jwt_issuer   = "https://cognito-idp.ap-southeast-1.amazonaws.com/ap-southeast-1_test"
    jwt_audience = "test-client"
    lambda_targets = {
      create-link  = { function_name = "url-shortener-dev-create-link", alias_invoke_arn = "arn:aws:apigateway:ap-southeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-create-link:live/invocations" }
      resolve-link = { function_name = "url-shortener-dev-resolve-link", alias_invoke_arn = "arn:aws:apigateway:ap-southeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-resolve-link:live/invocations" }
      list-links   = { function_name = "url-shortener-dev-list-links", alias_invoke_arn = "arn:aws:apigateway:ap-southeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-list-links:live/invocations" }
      delete-link  = { function_name = "url-shortener-dev-delete-link", alias_invoke_arn = "arn:aws:apigateway:ap-southeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-delete-link:live/invocations" }
    }
  }
  assert {
    condition     = length(aws_apigatewayv2_route.routes) == 4 && aws_apigatewayv2_route.routes["resolve-link"].authorization_type == "NONE" && alltrue([for key in ["create-link", "list-links", "delete-link"] : aws_apigatewayv2_route.routes[key].authorization_type == "JWT" && aws_apigatewayv2_route.routes[key].authorization_scopes == toset(["openid"])])
    error_message = "Resolve is public; all three owner routes must require Cognito access-token scope."
  }
  assert {
    condition     = alltrue([for value in aws_lambda_permission.api : value.qualifier == "live" && value.principal == "apigateway.amazonaws.com"]) && alltrue([for value in aws_apigatewayv2_integration.lambda : strcontains(value.integration_uri, ":live/invocations") && value.payload_format_version == "2.0"]) && aws_apigatewayv2_stage.default.name == "$default" && aws_apigatewayv2_stage.default.auto_deploy
    error_message = "API integrations and permissions must use qualified live aliases and HTTP payload v2."
  }
}
run "reject_remote_http_callback" {
  command = plan
  variables { callback_urls = ["http://example.org/auth/callback/"] }
  expect_failures = [var.callback_urls]
}
run "reject_wildcard_cors" {
  command = plan
  variables { cors_origins = ["*"] }
  expect_failures = [var.cors_origins]
}
