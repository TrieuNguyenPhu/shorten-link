output "site_url" { value = "https://${var.domain_name}" }
output "frontend_bucket_name" { value = aws_s3_bucket.frontend.id }
output "cloudfront_distribution_id" { value = aws_cloudfront_distribution.frontend.id }
output "cloudfront_domain_name" { value = aws_cloudfront_distribution.frontend.domain_name }
output "http_api_endpoint" { value = aws_apigatewayv2_api.shortener.api_endpoint }
output "links_table_name" { value = aws_dynamodb_table.links.name }
output "dns_record" {
  description = "Add this apex ANAME record at Name.com; do not change nameservers."
  value       = { type = "ANAME", name = var.domain_name, value = aws_cloudfront_distribution.frontend.domain_name }
}
