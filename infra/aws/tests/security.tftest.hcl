mock_provider "aws" {
  mock_resource "aws_cloudwatch_log_group" {
    defaults = { arn = "arn:aws:logs:ap-southeast-2:123456789012:log-group:shortener" }
  }
  mock_data "aws_caller_identity" {
    defaults = { account_id = "123456789012" }
  }
  mock_resource "aws_iam_role" {
    defaults = { arn = "arn:aws:iam::123456789012:role/shortener" }
  }
  mock_resource "aws_lambda_function" {
    defaults = {
      arn        = "arn:aws:lambda:ap-southeast-2:123456789012:function:shortener"
      invoke_arn = "arn:aws:apigateway:ap-southeast-2:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-southeast-2:123456789012:function:shortener/invocations"
    }
  }
  mock_resource "aws_apigatewayv2_api" {
    defaults = { execution_arn = "arn:aws:execute-api:ap-southeast-2:123456789012:testapi" }
  }
}

mock_provider "archive" {
}

run "secure_same_origin_deployment" {
  command = apply
  variables {
    certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/00000000-0000-0000-0000-000000000000"
  }
  assert {
    condition     = aws_s3_bucket_public_access_block.frontend.block_public_acls && aws_s3_bucket_public_access_block.frontend.block_public_policy && aws_s3_bucket_public_access_block.frontend.ignore_public_acls && aws_s3_bucket_public_access_block.frontend.restrict_public_buckets
    error_message = "The frontend bucket must remain private."
  }
  assert {
    condition     = aws_s3_bucket_versioning.frontend.versioning_configuration[0].status == "Enabled" && aws_dynamodb_table.links.point_in_time_recovery[0].enabled
    error_message = "Link data and frontend rollback protection must be enabled."
  }
  assert {
    condition     = toset(keys(aws_lambda_function.shortener.environment[0].variables)) == toset(["CORS_ALLOWED_ORIGINS", "LINKS_TABLE_NAME", "LOG_LEVEL", "PUBLIC_BASE_URL", "STORAGE_DRIVER"])
    error_message = "Lambda configuration must not contain operator access keys."
  }
  assert {
    condition     = jsondecode(aws_iam_role_policy.shortener.policy).Statement[1].Action == ["dynamodb:GetItem", "dynamodb:PutItem"]
    error_message = "Lambda must only read/write individual links."
  }
  assert {
    condition     = aws_apigatewayv2_api.shortener.cors_configuration[0].allow_origins == toset(["https://npt-shortenlink.dev"])
    error_message = "CORS must be limited to the website origin."
  }
  assert {
    condition     = alltrue([for behavior in aws_cloudfront_distribution.frontend.ordered_cache_behavior : behavior.cache_policy_id == "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"])
    error_message = "API responses and redirects must not be cached."
  }
  assert {
    condition     = output.dns_record.name == "npt-shortenlink.dev" && output.dns_record.type == "ANAME" && output.dns_record.value == aws_cloudfront_distribution.frontend.domain_name
    error_message = "The apex domain target must be exported for Name.com DNS."
  }
  assert {
    condition     = aws_lambda_permission.api["GET /link/{code}"].source_arn == "arn:aws:execute-api:ap-southeast-2:123456789012:testapi/*/GET/link/*"
    error_message = "API Gateway must be permitted to invoke the redirect route with a concrete code."
  }
}
