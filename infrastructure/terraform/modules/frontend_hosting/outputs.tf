output "bucket_name" { value = aws_s3_bucket.site.bucket }
output "distribution_id" { value = aws_cloudfront_distribution.site.id }
output "url" { value = var.domain_name == null ? "https://${aws_cloudfront_distribution.site.domain_name}" : "https://${var.domain_name}" }
