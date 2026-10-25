output "site_url" { value = "https://${var.site_domain == "" ? aws_cloudfront_distribution.web.domain_name : var.site_domain}" }
output "api_url" {
  description = "Use this origin for developer SigV4 management calls until Cognito is added."
  value       = aws_apigatewayv2_api.http.api_endpoint
}
output "web_bucket" { value = aws_s3_bucket.web.id }
output "distribution_id" { value = aws_cloudfront_distribution.web.id }
output "links_table" { value = aws_dynamodb_table.links.name }
