output "user_pool_id" { value = aws_cognito_user_pool.pool.id }
output "user_pool_arn" { value = aws_cognito_user_pool.pool.arn }
output "client_id" { value = aws_cognito_user_pool_client.web.id }
output "issuer" { value = "https://${aws_cognito_user_pool.pool.endpoint}" }
output "login_domain" { value = "https://${aws_cognito_user_pool_domain.login.domain}.auth.${data.aws_region.current.region}.amazoncognito.com" }
output "google_redirect_uri" { value = "https://${aws_cognito_user_pool_domain.login.domain}.auth.${data.aws_region.current.region}.amazoncognito.com/oauth2/idpresponse" }
data "aws_region" "current" {}
