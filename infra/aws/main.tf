data "aws_caller_identity" "current" {}

data "archive_file" "lambda" {
  type             = "zip"
  source_file      = "${path.module}/.build/bootstrap"
  output_path      = "${path.module}/.build/lambda.zip"
  output_file_mode = "0755"
}

resource "aws_dynamodb_table" "links" {
  name         = "${var.deployment_name}-prod-links"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "code"
  attribute {
    name = "code"
    type = "S"
  }
  point_in_time_recovery { enabled = true }
  server_side_encryption { enabled = true }
  ttl {
    attribute_name = "ttl"
    enabled        = true
  }
  lifecycle { prevent_destroy = true }
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${var.deployment_name}-shortener-api"
  retention_in_days = 30
  lifecycle { prevent_destroy = true }
}

resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/http-api/${var.deployment_name}"
  retention_in_days = 30
  lifecycle { prevent_destroy = true }
}

resource "aws_iam_role" "shortener" {
  name = "${var.deployment_name}-shortener"
  assume_role_policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [{ Effect = "Allow", Principal = { Service = "lambda.amazonaws.com" }, Action = "sts:AssumeRole" }]
  })
}

resource "aws_iam_role_policy" "shortener" {
  name = "shortener-execution"
  role = aws_iam_role.shortener.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "${aws_cloudwatch_log_group.lambda.arn}:*"
      },
      {
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem"]
        Resource = aws_dynamodb_table.links.arn
      },
      {
        Effect   = "Allow"
        Action   = ["xray:PutTelemetryRecords", "xray:PutTraceSegments"]
        Resource = "*"
      }
    ]
  })
}

resource "aws_lambda_function" "shortener" {
  function_name                  = "${var.deployment_name}-shortener-api"
  role                           = aws_iam_role.shortener.arn
  runtime                        = "provided.al2023"
  architectures                  = ["arm64"]
  handler                        = "bootstrap"
  filename                       = data.archive_file.lambda.output_path
  source_code_hash               = data.archive_file.lambda.output_base64sha256
  memory_size                    = 256
  timeout                        = 10
  reserved_concurrent_executions = 20
  environment {
    variables = {
      LINKS_TABLE_NAME     = aws_dynamodb_table.links.name
      PUBLIC_BASE_URL      = "https://${var.domain_name}"
      STORAGE_DRIVER       = "dynamodb"
      CORS_ALLOWED_ORIGINS = "https://${var.domain_name}"
      LOG_LEVEL            = "INFO"
    }
  }
  tracing_config { mode = "Active" }
  logging_config {
    log_format            = "JSON"
    application_log_level = "INFO"
    system_log_level      = "INFO"
    log_group             = aws_cloudwatch_log_group.lambda.name
  }
  depends_on = [aws_iam_role_policy.shortener]
}

resource "aws_apigatewayv2_api" "shortener" {
  name          = "${var.deployment_name}-http-api"
  protocol_type = "HTTP"
  cors_configuration {
    allow_credentials = false
    allow_headers     = ["Content-Type", "Authorization", "X-Request-Id"]
    allow_methods     = ["GET", "POST", "OPTIONS"]
    allow_origins     = ["https://${var.domain_name}"]
    expose_headers    = ["Location", "X-Request-Id"]
    max_age           = 600
  }
}

resource "aws_apigatewayv2_integration" "shortener" {
  api_id                 = aws_apigatewayv2_api.shortener.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.shortener.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "shortener" {
  for_each  = toset(["POST /api/v1/links", "GET /api/v1/links/{code}", "GET /link/{code}", "GET /healthz"])
  api_id    = aws_apigatewayv2_api.shortener.id
  route_key = each.key
  target    = "integrations/${aws_apigatewayv2_integration.shortener.id}"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.shortener.id
  name        = "$default"
  auto_deploy = true
  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api.arn
    format = jsonencode({
      requestId = "$context.requestId", httpMethod = "$context.httpMethod", routeKey = "$context.routeKey",
      status    = "$context.status", responseLength = "$context.responseLength", integrationError = "$context.integrationErrorMessage"
    })
  }
  default_route_settings {
    detailed_metrics_enabled = true
    throttling_burst_limit   = 50
    throttling_rate_limit    = 25
  }
}

resource "aws_lambda_permission" "api" {
  for_each      = aws_apigatewayv2_route.shortener
  statement_id  = "HttpApi-${substr(sha256(each.key), 0, 16)}"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.shortener.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.shortener.execution_arn}/*/${replace(replace(each.key, " /", "/"), "/{code}", "/*")}"
}

resource "aws_s3_bucket" "frontend" {
  bucket = "${var.deployment_name}-frontend-${data.aws_caller_identity.current.account_id}"
  lifecycle { prevent_destroy = true }
}

resource "aws_s3_bucket_public_access_block" "frontend" {
  bucket                  = aws_s3_bucket.frontend.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "frontend" {
  bucket = aws_s3_bucket.frontend.id
  rule { object_ownership = "BucketOwnerEnforced" }
}

resource "aws_s3_bucket_versioning" "frontend" {
  bucket = aws_s3_bucket.frontend.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "frontend" {
  bucket = aws_s3_bucket.frontend.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}

resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = "${var.deployment_name}-frontend-oac"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_response_headers_policy" "security" {
  name = "${var.deployment_name}-security-headers"
  security_headers_config {
    content_type_options { override = true }
    frame_options {
      frame_option = "SAMEORIGIN"
      override     = true
    }
    referrer_policy {
      referrer_policy = "strict-origin-when-cross-origin"
      override        = true
    }
    strict_transport_security {
      access_control_max_age_sec = 86400
      include_subdomains         = false
      preload                    = false
      override                   = true
    }
    xss_protection {
      mode_block = true
      protection = true
      override   = true
    }
  }
}

resource "aws_cloudfront_distribution" "frontend" {
  enabled             = true
  is_ipv6_enabled     = true
  aliases             = [var.domain_name]
  default_root_object = "index.html"
  http_version        = "http2and3"
  price_class         = "PriceClass_100"
  origin {
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_id                = "frontend-s3"
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }
  origin {
    domain_name = trimprefix(aws_apigatewayv2_api.shortener.api_endpoint, "https://")
    origin_id   = "shortener-http-api"
    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }
  default_cache_behavior {
    allowed_methods            = ["GET", "HEAD", "OPTIONS"]
    cached_methods             = ["GET", "HEAD"]
    target_origin_id           = "frontend-s3"
    viewer_protocol_policy     = "redirect-to-https"
    cache_policy_id            = "658327ea-f89d-4fab-a63d-7e88639e58f6"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.security.id
    compress                   = true
  }
  dynamic "ordered_cache_behavior" {
    for_each = ["/api/*", "/link/*", "/healthz"]
    content {
      path_pattern               = ordered_cache_behavior.value
      allowed_methods            = ordered_cache_behavior.value == "/api/*" ? ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"] : ["GET", "HEAD", "OPTIONS"]
      cached_methods             = ["GET", "HEAD"]
      target_origin_id           = "shortener-http-api"
      viewer_protocol_policy     = "redirect-to-https"
      cache_policy_id            = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
      origin_request_policy_id   = "b689b0a8-53d0-40ab-baf2-68738e2966ac"
      response_headers_policy_id = aws_cloudfront_response_headers_policy.security.id
      compress                   = true
    }
  }
  restrictions {
    geo_restriction { restriction_type = "none" }
  }
  viewer_certificate {
    acm_certificate_arn      = var.certificate_arn
    minimum_protocol_version = "TLSv1.2_2021"
    ssl_support_method       = "sni-only"
  }
}

resource "aws_s3_bucket_policy" "frontend" {
  bucket = aws_s3_bucket.frontend.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "cloudfront.amazonaws.com" }
      Action    = "s3:GetObject"
      Resource  = "${aws_s3_bucket.frontend.arn}/*"
      Condition = { StringEquals = { "AWS:SourceArn" = aws_cloudfront_distribution.frontend.arn } }
    }]
  })
  depends_on = [aws_s3_bucket_public_access_block.frontend]
}
