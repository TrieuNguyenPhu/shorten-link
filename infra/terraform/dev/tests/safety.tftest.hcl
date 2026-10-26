mock_provider "aws" {
  override_during = plan
  mock_resource "aws_cloudfront_distribution" {
    defaults = { domain_name = "offline-web.cloudfront.net" }
  }
  mock_resource "aws_dynamodb_table" {
    defaults = { arn = "arn:aws:dynamodb:ap-southeast-1:123456789012:table/shortly-dev-links" }
  }
  mock_resource "aws_cloudwatch_log_group" {
    defaults = { arn = "arn:aws:logs:ap-southeast-1:123456789012:log-group:test" }
  }
  mock_resource "aws_s3_bucket" {
    defaults = { arn = "arn:aws:s3:::offline-web-test", bucket_regional_domain_name = "offline-web-test.s3.ap-southeast-1.amazonaws.com" }
  }
  mock_resource "aws_apigatewayv2_api" {
    defaults = {
      execution_arn = "arn:aws:execute-api:ap-southeast-1:123456789012:offline-test"
      api_endpoint  = "https://offline-test.execute-api.ap-southeast-1.amazonaws.com"
    }
  }
  mock_resource "aws_cognito_user_pool" {
    defaults = {
      id  = "ap-southeast-2_TestPool"
      arn = "arn:aws:cognito-idp:ap-southeast-2:123456789012:userpool/ap-southeast-2_TestPool"
    }
  }
  mock_resource "aws_cognito_resource_server" {
    defaults = {
      scope_identifiers = ["shortly-api/links.read", "shortly-api/links.write"]
    }
  }
  mock_resource "aws_cognito_user_pool_client" {
    defaults = {
      id = "test-cognito-web-client-id"
    }
  }
  mock_resource "aws_apigatewayv2_authorizer" {
    defaults = {
      id = "jwt-authorizer-id"
    }
  }
}

override_resource {
  target          = aws_cloudfront_cache_policy.api
  override_during = plan
  values          = { id = "api-no-cache" }
}
override_resource {
  target          = aws_cloudfront_cache_policy.static
  override_during = plan
  values          = { id = "static-cache" }
}
override_resource {
  target          = aws_cloudfront_response_headers_policy.static
  override_during = plan
  values          = { id = "static-security-headers" }
}

run "offline_safety" {
  command = plan

  assert {
    condition     = aws_budgets_budget.dev.limit_amount == "4" && aws_budgets_budget.dev.limit_unit == "USD" && aws_budgets_budget.dev.time_unit == "MONTHLY"
    error_message = "The dev budget must track the agreed 4 USD monthly target."
  }

  assert {
    condition = length(aws_apigatewayv2_route.route) == 11 && alltrue([
      for route in aws_apigatewayv2_route.route :
      contains(["GET /r/{code}", "HEAD /r/{code}", "GET /health", "HEAD /health"], route.route_key) ? (
        route.authorization_type == "NONE" && route.authorizer_id == null
        ) : (
        route.authorization_type == "JWT" &&
        route.authorizer_id == aws_apigatewayv2_authorizer.jwt.id &&
        route.authorization_scopes == toset(
          contains(["GET /api/links", "HEAD /api/links", "GET /api/links/{code}", "HEAD /api/links/{code}"], route.route_key) ? ["shortly-api/links.read"] : ["shortly-api/links.write"]
        )
      )
    ])
    error_message = "Every management route must require Cognito JWT with explicit read/write access-token scopes; redirect and health stay public."
  }
  assert {
    condition = (
      !aws_cognito_user_pool_client.web.generate_secret &&
      aws_cognito_user_pool_client.web.allowed_oauth_flows_user_pool_client &&
      aws_cognito_user_pool_client.web.allowed_oauth_flows == toset(["code"]) &&
      aws_cognito_user_pool_client.web.callback_urls == toset(["https://offline-web.cloudfront.net/auth/callback/"]) &&
      aws_cognito_user_pool_client.web.logout_urls == toset(["https://offline-web.cloudfront.net/"]) &&
      aws_cognito_user_pool_client.web.default_redirect_uri == "${output.site_url}/auth/callback/" &&
      aws_cognito_user_pool_client.web.access_token_validity == 60 &&
      aws_cognito_user_pool_client.web.id_token_validity == 60 &&
      aws_cognito_user_pool_client.web.refresh_token_validity == 30 &&
      aws_apigatewayv2_authorizer.jwt.authorizer_type == "JWT" &&
      aws_apigatewayv2_authorizer.jwt.identity_sources == toset(["$request.header.Authorization"]) &&
      aws_apigatewayv2_authorizer.jwt.jwt_configuration[0].audience == toset([aws_cognito_user_pool_client.web.id]) &&
      aws_apigatewayv2_authorizer.jwt.jwt_configuration[0].issuer == "https://cognito-idp.ap-southeast-2.amazonaws.com/ap-southeast-2_TestPool"
    )
    error_message = "Cognito client must enforce PKCE code flow, explicit HTTPS callback/logout URLs, agreed token validities, and matching HTTP API JWT authorizer."
  }
  assert {
    condition     = alltrue([for f in aws_lambda_function.function : f.runtime == "provided.al2023" && length(f.architectures) == 1 && contains(f.architectures, "arm64") && f.publish && f.environment[0].variables.DYNAMODB_TABLE == aws_dynamodb_table.links.name])
    error_message = "Lambda runtime, artifact architecture and table wiring must agree."
  }
  assert {
    condition     = alltrue([for integration in aws_apigatewayv2_integration.lambda : integration.payload_format_version == "2.0"]) && alltrue([for permission in aws_lambda_permission.gateway : permission.qualifier == "live" && !endswith(permission.source_arn, "/*/*")])
    error_message = "Invoke only published aliases with payload v2 and route-scoped permissions."
  }
  assert {
    condition     = aws_lambda_permission.gateway["POST /api/links"].source_arn == "arn:aws:execute-api:ap-southeast-1:123456789012:offline-test/*/POST/api/links" && aws_lambda_permission.gateway["GET /r/{code}"].source_arn == "arn:aws:execute-api:ap-southeast-1:123456789012:offline-test/*/GET/r/*"
    error_message = "API Gateway permissions must match real method/path execution ARNs exactly."
  }
  assert {
    condition     = jsondecode(aws_iam_role_policy.function["resolve-link"].policy).Statement[0].Action == ["dynamodb:GetItem"] && jsondecode(aws_iam_role_policy.function["create-link"].policy).Statement[0].Action == ["dynamodb:PutItem"]
    error_message = "Redirect cannot write data, and create cannot read/list/delete data."
  }
  assert {
    condition     = aws_dynamodb_table.links.deletion_protection_enabled && one(aws_dynamodb_table.links.point_in_time_recovery).enabled && aws_dynamodb_table.links.hash_key == "PK" && aws_dynamodb_table.links.range_key == "SK" && one(aws_dynamodb_table.links.global_secondary_index).projection_type == "ALL"
    error_message = "Preserve the single-table contract and protect data from deletion."
  }
  assert {
    condition     = aws_s3_bucket_public_access_block.web.block_public_acls && aws_s3_bucket_public_access_block.web.block_public_policy && !aws_s3_bucket.web.force_destroy && aws_cloudfront_origin_access_control.web.signing_behavior == "always"
    error_message = "Web storage must remain private and use signed OAC requests."
  }
  assert {
    condition     = alltrue([for behavior in aws_cloudfront_distribution.web.ordered_cache_behavior : behavior.cache_policy_id == aws_cloudfront_cache_policy.api.id && behavior.origin_request_policy_id == "b689b0a8-53d0-40ab-baf2-68738e2966ac" && length(behavior.function_association) == 0]) && aws_cloudfront_cache_policy.api.min_ttl == 0 && aws_cloudfront_cache_policy.api.default_ttl == 0 && aws_cloudfront_cache_policy.api.max_ttl == 0
    error_message = "Never cache or rewrite API and redirect requests; forward viewer fields except Host."
  }
  assert {
    condition     = toset([for response in aws_cloudfront_distribution.web.custom_error_response : response.error_code if response.error_caching_min_ttl == 0 && (response.response_page_path == null || response.response_page_path == "")]) == toset([400, 403, 404, 405, 414, 500, 501, 502, 503, 504])
    error_message = "Disable CloudFront error caching too; never replace API errors with an HTML page."
  }
  assert {
    condition     = one(aws_cloudfront_distribution.web.default_cache_behavior).response_headers_policy_id == aws_cloudfront_response_headers_policy.static.id && one(aws_cloudfront_response_headers_policy.static.security_headers_config).frame_options[0].frame_option == "DENY" && one(aws_cloudfront_response_headers_policy.static.security_headers_config).content_security_policy[0].content_security_policy == "frame-ancestors 'none'" && one(aws_cloudfront_response_headers_policy.static.security_headers_config).content_type_options[0].override && one(aws_cloudfront_response_headers_policy.static.security_headers_config).strict_transport_security[0].access_control_max_age_sec >= 31536000
    error_message = "Static hosting must send anti-framing, nosniff and HSTS security headers."
  }
}

run "custom_domain_auth_urls" {
  command = plan
  variables {
    site_domain = "short.example"
    acm_certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/offline-test"
  }
  assert {
    condition = (
      output.site_url == "https://short.example" &&
      aws_cognito_user_pool_client.web.callback_urls == toset(["https://short.example/auth/callback/"]) &&
      aws_cognito_user_pool_client.web.logout_urls == toset(["https://short.example/"]) &&
      aws_cognito_user_pool_client.web.default_redirect_uri == "https://short.example/auth/callback/"
    )
    error_message = "Cognito must follow the actual site domain, independently of the direct API CORS origin."
  }
}
