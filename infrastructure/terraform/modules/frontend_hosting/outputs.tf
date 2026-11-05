output "bucket_name" { value = aws_s3_bucket.site.bucket }
output "distribution_id" { value = aws_cloudfront_distribution.site.id }
output "distribution_domain_name" { value = aws_cloudfront_distribution.site.domain_name }
output "url" { value = "https://${var.domain_name != "" ? var.domain_name : aws_cloudfront_distribution.site.domain_name}" }
