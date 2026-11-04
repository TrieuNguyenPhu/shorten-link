mock_provider "aws" {
  override_during = plan
  mock_resource "aws_cloudfront_distribution" {
    defaults = { arn = "arn:aws:cloudfront::111111111111:distribution/ETEST", domain_name = "dtest.cloudfront.net" }
  }
  mock_resource "aws_s3_bucket" {
    defaults = { arn = "arn:aws:s3:::test-frontend", bucket_regional_domain_name = "test-frontend.s3.ap-southeast-1.amazonaws.com" }
  }
  mock_resource "aws_cloudfront_origin_access_control" {
    defaults = { id = "ETESTOAC" }
  }
  mock_resource "aws_cloudfront_response_headers_policy" {
    defaults = { id = "11111111-1111-4111-8111-111111111111" }
  }
  mock_resource "aws_cloudfront_function" {
    defaults = { arn = "arn:aws:cloudfront::111111111111:function/test-rewrite" }
  }
}

run "static_hosting_contract" {
  command = plan
  module { source = "../../modules/frontend_hosting" }
  variables {
    name_prefix = "test"
    bucket_name = "test-frontend"
  }
  override_resource {
    target          = aws_cloudfront_cache_policy.site["pages"]
    override_during = plan
    values          = { id = "22222222-2222-4222-8222-222222222222" }
  }
  override_resource {
    target          = aws_cloudfront_cache_policy.site["assets"]
    override_during = plan
    values          = { id = "33333333-3333-4333-8333-333333333333" }
  }
  assert {
    condition = (
      aws_s3_bucket_public_access_block.site.block_public_acls &&
      aws_s3_bucket_public_access_block.site.ignore_public_acls &&
      aws_s3_bucket_public_access_block.site.block_public_policy &&
      aws_s3_bucket_public_access_block.site.restrict_public_buckets &&
      aws_s3_bucket_ownership_controls.site.rule[0].object_ownership == "BucketOwnerEnforced" &&
      aws_cloudfront_origin_access_control.site.signing_behavior == "always" &&
      aws_cloudfront_origin_access_control.site.signing_protocol == "sigv4" &&
      one(aws_cloudfront_distribution.site.origin).domain_name == aws_s3_bucket.site.bucket_regional_domain_name &&
      one(aws_cloudfront_distribution.site.origin).origin_access_control_id == aws_cloudfront_origin_access_control.site.id
    )
    error_message = "Only a private S3 regional origin signed by OAC should serve the site."
  }
  assert {
    condition = jsondecode(aws_s3_bucket_policy.site.policy).Statement[0] == {
      Sid       = "ReadFromDistribution", Effect = "Allow", Principal = { Service = "cloudfront.amazonaws.com" },
      Action    = "s3:GetObject", Resource = "arn:aws:s3:::test-frontend/*",
      Condition = { StringEquals = { "AWS:SourceArn" = aws_cloudfront_distribution.site.arn }, Bool = { "aws:SecureTransport" = "true" } }
    } && length(jsondecode(aws_s3_bucket_policy.site.policy).Statement) == 1
    error_message = "Bucket policy must allow only HTTPS object reads by this distribution."
  }
  assert {
    condition = (
      length(aws_cloudfront_distribution.site.custom_error_response) == 0 &&
      one(aws_cloudfront_distribution.site.default_cache_behavior).viewer_protocol_policy == "redirect-to-https" &&
      one(one(aws_cloudfront_distribution.site.default_cache_behavior).function_association).function_arn == aws_cloudfront_function.rewrite.arn &&
      one(one(aws_cloudfront_distribution.site.default_cache_behavior).function_association).event_type == "viewer-request" &&
      one(aws_cloudfront_distribution.site.default_cache_behavior).cache_policy_id == aws_cloudfront_cache_policy.site["pages"].id &&
      aws_cloudfront_cache_policy.site["pages"].max_ttl <= 300 &&
      one(aws_cloudfront_distribution.site.ordered_cache_behavior).path_pattern == "_next/static/*" &&
      one(aws_cloudfront_distribution.site.ordered_cache_behavior).cache_policy_id == aws_cloudfront_cache_policy.site["assets"].id &&
      one(aws_cloudfront_distribution.site.ordered_cache_behavior).response_headers_policy_id == aws_cloudfront_response_headers_policy.assets.id &&
      aws_cloudfront_cache_policy.site["assets"].default_ttl == 31536000 &&
      one(one(aws_cloudfront_response_headers_policy.assets.custom_headers_config).items).value == "public,max-age=31536000,immutable" &&
      alltrue([for p in aws_cloudfront_cache_policy.site : one(one(p.parameters_in_cache_key_and_forwarded_to_origin).query_strings_config).query_string_behavior == "none"])
    )
    error_message = "Exported pages need viewer rewrites, short cache and native errors. Hashed assets must cache for one year."
  }
}
