locals {
  site_url = "https://${var.site_domain == "" ? aws_cloudfront_distribution.web.domain_name : var.site_domain}"
}

output "site_url" { value = local.site_url }
output "api_url" {
  description = "Direct HTTP API Gateway origin."
  value       = aws_apigatewayv2_api.http.api_endpoint
}
output "web_bucket" { value = aws_s3_bucket.web.id }
output "distribution_id" { value = aws_cloudfront_distribution.web.id }
output "links_table" { value = aws_dynamodb_table.links.name }
output "aws_account_id" { value = split(":", aws_dynamodb_table.links.arn)[4] }
output "aws_region" { value = var.region }
output "cognito_user_pool_id" { value = aws_cognito_user_pool.users.id }
output "cognito_client_id" { value = aws_cognito_user_pool_client.web.id }
output "cognito_domain_url" { value = "https://${aws_cognito_user_pool_domain.auth.domain}.auth.${var.region}.amazoncognito.com" }
output "cognito_issuer" { value = "https://cognito-idp.${var.region}.amazonaws.com/${aws_cognito_user_pool.users.id}" }
