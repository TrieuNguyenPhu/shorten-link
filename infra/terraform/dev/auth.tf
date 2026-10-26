resource "aws_cognito_user_pool" "users" {
  name                     = "${local.name}-users"
  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]
  deletion_protection      = "ACTIVE"

  user_attribute_update_settings {
    attributes_require_verification_before_update = ["email"]
  }

  password_policy {
    minimum_length                   = 12
    require_lowercase                = true
    require_numbers                  = true
    require_symbols                  = true
    require_uppercase                = true
    temporary_password_validity_days = 7
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }
}

resource "aws_cognito_user_pool_domain" "auth" {
  domain       = "${local.name}-${substr(sha256(aws_dynamodb_table.links.arn), 0, 12)}"
  user_pool_id = aws_cognito_user_pool.users.id
}

resource "aws_cognito_resource_server" "api" {
  identifier   = "shortly-api"
  name         = "${local.name}-api"
  user_pool_id = aws_cognito_user_pool.users.id

  scope {
    scope_name        = "links.read"
    scope_description = "Read own short links and metadata"
  }

  scope {
    scope_name        = "links.write"
    scope_description = "Create, update and delete own short links"
  }
}

resource "aws_cognito_user_pool_client" "web" {
  name                                 = "${local.name}-web"
  user_pool_id                         = aws_cognito_user_pool.users.id
  generate_secret                      = false
  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = sort(concat(["openid", "email"], tolist(aws_cognito_resource_server.api.scope_identifiers)))
  supported_identity_providers         = ["COGNITO"]
  callback_urls                        = ["${var.web_origin}/auth/callback/"]
  logout_urls                          = ["${var.web_origin}/"]
  default_redirect_uri                 = "${var.web_origin}/auth/callback/"
  explicit_auth_flows                  = ["ALLOW_REFRESH_TOKEN_AUTH", "ALLOW_USER_SRP_AUTH"]
  enable_token_revocation              = true
  prevent_user_existence_errors        = "ENABLED"
  access_token_validity                = 60
  id_token_validity                    = 60
  refresh_token_validity               = 30

  token_validity_units {
    access_token  = "minutes"
    id_token      = "minutes"
    refresh_token = "days"
  }
}

resource "aws_apigatewayv2_authorizer" "jwt" {
  api_id           = aws_apigatewayv2_api.http.id
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]
  name             = "${local.name}-cognito-jwt"

  jwt_configuration {
    audience = [aws_cognito_user_pool_client.web.id]
    issuer   = "https://cognito-idp.${var.region}.amazonaws.com/${aws_cognito_user_pool.users.id}"
  }
}
