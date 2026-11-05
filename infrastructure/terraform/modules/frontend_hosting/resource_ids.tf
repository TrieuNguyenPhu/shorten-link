output "resource_ids" {
  value = {
    distribution_id            = aws_cloudfront_distribution.site.id
    origin_access_control_id   = aws_cloudfront_origin_access_control.site.id
    cache_policy_ids           = toset([for policy in aws_cloudfront_cache_policy.site : policy.id])
    response_headers_policy_id = aws_cloudfront_response_headers_policy.assets.id
  }
}
