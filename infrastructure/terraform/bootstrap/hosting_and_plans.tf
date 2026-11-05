variable "frontend_resource_ids" {
  type = object({
    distribution_id            = string
    origin_access_control_id   = string
    cache_policy_ids           = set(string)
    response_headers_policy_id = string
  })
  default     = null
  description = "Exact CloudFront IDs provisioned by the operator; null grants no CloudFront maintenance."
  validation {
    condition = var.frontend_resource_ids == null ? true : (
      can(regex("^[A-Z0-9]+$", var.frontend_resource_ids.distribution_id)) &&
      can(regex("^[A-Z0-9]+$", var.frontend_resource_ids.origin_access_control_id)) &&
      length(var.frontend_resource_ids.cache_policy_ids) == 2 &&
      alltrue([for id in setunion(var.frontend_resource_ids.cache_policy_ids, [var.frontend_resource_ids.response_headers_policy_id]) : can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", id))])
    )
    error_message = "Pin the distribution/OAC and exact UUIDs of two cache policies and the response headers policy."
  }
}
locals {
  frontend_bucket_arn = "${local.base_arn}:s3:::${var.name_prefix}-${var.aws_account_id}-frontend"
  frontend_cf_base    = "${local.base_arn}:cloudfront::${var.aws_account_id}"
}
resource "aws_iam_policy" "hosting_deployment" {
  name = "${var.name_prefix}-hosting-deploy"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({ Version = "2012-10-17", Statement = concat([
    { Sid = "ReviewSavedPlans", Effect = "Allow", Action = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"], Resource = ["${aws_s3_bucket.state.arn}/dev/plans/*"] },
    { Sid = "FrontendBucket", Effect = "Allow", Action = ["s3:ListBucket", "s3:GetBucketLocation", "s3:GetBucketPolicy", "s3:PutBucketPolicy", "s3:GetBucketPublicAccessBlock", "s3:PutBucketPublicAccessBlock", "s3:GetBucketOwnershipControls", "s3:PutBucketOwnershipControls", "s3:GetBucketTagging", "s3:PutBucketTagging", "s3:GetBucketAcl", "s3:GetBucketCors", "s3:GetBucketWebsite", "s3:GetBucketVersioning", "s3:GetAccelerateConfiguration", "s3:GetBucketRequestPayment", "s3:GetBucketLogging", "s3:GetLifecycleConfiguration", "s3:GetReplicationConfiguration", "s3:GetEncryptionConfiguration", "s3:GetBucketObjectLockConfiguration"], Resource = [local.frontend_bucket_arn] },
    # Release preserves old hashed bundles; it cannot delete frontend objects.
    { Sid = "PublishFrontend", Effect = "Allow", Action = ["s3:GetObject", "s3:PutObject"], Resource = ["${local.frontend_bucket_arn}/*"] }
    ], var.frontend_resource_ids == null ? [] : [
    { Sid = "FrontendDistribution", Effect = "Allow", Action = ["cloudfront:GetDistribution", "cloudfront:GetDistributionConfig", "cloudfront:UpdateDistribution", "cloudfront:ListTagsForResource", "cloudfront:TagResource", "cloudfront:UntagResource", "cloudfront:CreateInvalidation", "cloudfront:GetInvalidation"], Resource = ["${local.frontend_cf_base}:distribution/${var.frontend_resource_ids.distribution_id}"] },
    { Sid = "FrontendRewrite", Effect = "Allow", Action = ["cloudfront:DescribeFunction", "cloudfront:GetFunction", "cloudfront:UpdateFunction", "cloudfront:PublishFunction", "cloudfront:ListTagsForResource", "cloudfront:TagResource", "cloudfront:UntagResource"], Resource = ["${local.frontend_cf_base}:function/${var.name_prefix}-frontend-rewrite"] },
    { Sid = "FrontendOAC", Effect = "Allow", Action = ["cloudfront:GetOriginAccessControl", "cloudfront:UpdateOriginAccessControl"], Resource = ["${local.frontend_cf_base}:origin-access-control/${var.frontend_resource_ids.origin_access_control_id}"] },
    { Sid = "FrontendCache", Effect = "Allow", Action = ["cloudfront:GetCachePolicy", "cloudfront:UpdateCachePolicy"], Resource = [for id in var.frontend_resource_ids.cache_policy_ids : "${local.frontend_cf_base}:cache-policy/${id}"] },
    { Sid = "FrontendHeaders", Effect = "Allow", Action = ["cloudfront:GetResponseHeadersPolicy", "cloudfront:UpdateResponseHeadersPolicy"], Resource = ["${local.frontend_cf_base}:response-headers-policy/${var.frontend_resource_ids.response_headers_policy_id}"] }
  ]) })
}
resource "aws_iam_role_policy_attachment" "hosting" {
  role       = aws_iam_role.deployment.name
  policy_arn = aws_iam_policy.hosting_deployment.arn
}
# Expire sensitive saved plans without affecting state or its version history.
resource "aws_s3_bucket_lifecycle_configuration" "plans" {
  bucket = aws_s3_bucket.state.id
  rule {
    id     = "expire-reviewed-plans"
    status = "Enabled"
    filter { prefix = "dev/plans/" }
    expiration { days = 2 }
    noncurrent_version_expiration { noncurrent_days = 2 }
    abort_incomplete_multipart_upload { days_after_initiation = 1 }
  }
}
