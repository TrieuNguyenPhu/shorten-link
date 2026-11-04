resource "aws_s3_bucket" "state" {
  bucket        = var.state_bucket_name
  force_destroy = false
  lifecycle { prevent_destroy = true }
}
resource "aws_s3_bucket_versioning" "state" {
  bucket = aws_s3_bucket.state.id
  versioning_configuration { status = "Enabled" }
}
resource "aws_s3_bucket_server_side_encryption_configuration" "state" {
  bucket = aws_s3_bucket.state.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}
resource "aws_s3_bucket_public_access_block" "state" {
  bucket                  = aws_s3_bucket.state.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
resource "aws_s3_bucket_policy" "state_tls" {
  bucket = aws_s3_bucket.state.id
  policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [{ Sid = "DenyInsecureTransport", Effect = "Deny", Principal = "*", Action = "s3:*", Resource = [aws_s3_bucket.state.arn, "${aws_s3_bucket.state.arn}/*"], Condition = { Bool = { "aws:SecureTransport" = "false" } } }]
  })
}
locals {
  api_ids = length(var.http_api_ids) > 0 ? var.http_api_ids : toset(["*"])
  api_resource_arns = flatten([for id in local.api_ids : [
    "arn:${data.aws_partition.current.partition}:apigateway:${var.aws_region}::/apis/${id}",
    "arn:${data.aws_partition.current.partition}:apigateway:${var.aws_region}::/apis/${id}/stages/*"
  ]])
  api_tag_arns = flatten([for arn in local.api_resource_arns : [
    "arn:${data.aws_partition.current.partition}:apigateway:${var.aws_region}::/tags/${arn}",
    "arn:${data.aws_partition.current.partition}:apigateway:${var.aws_region}::/tags/${replace(urlencode(arn), "%2A", "*")}"
  ]])
  api_child_arns = flatten([for id in local.api_ids : [for child in ["authorizers", "integrations", "routes", "stages"] : [
    "arn:${data.aws_partition.current.partition}:apigateway:${var.aws_region}::/apis/${id}/${child}",
    "arn:${data.aws_partition.current.partition}:apigateway:${var.aws_region}::/apis/${id}/${child}/*"
  ]]])
  partition = data.aws_partition.current.partition
  base_arn  = "arn:${local.partition}"
  table_arn = "${local.base_arn}:dynamodb:${var.aws_region}:${var.aws_account_id}:table/${var.table_name}"
  bus_arn   = "${local.base_arn}:events:${var.aws_region}:${var.aws_account_id}:event-bus/${var.event_bus_name}"
  handlers  = ["create-link", "resolve-link", "list-links", "delete-link"]
  role_arns = [for handler in local.handlers : "${local.base_arn}:iam::${var.aws_account_id}:role/url-shortener/dev/${var.name_prefix}-${handler}"]
  function_arns = flatten([for handler in local.handlers : [
    "${local.base_arn}:lambda:${var.aws_region}:${var.aws_account_id}:function:${var.name_prefix}-${handler}",
    "${local.base_arn}:lambda:${var.aws_region}:${var.aws_account_id}:function:${var.name_prefix}-${handler}:*"
  ]])
  log_arns = ["${local.base_arn}:logs:${var.aws_region}:${var.aws_account_id}:log-group:/aws/lambda/${var.name_prefix}-*", "${local.base_arn}:logs:${var.aws_region}:${var.aws_account_id}:log-group:/aws/apigateway/${var.name_prefix}*"]
}
data "aws_partition" "current" {}
resource "aws_iam_policy" "runtime_boundary" {
  name = "${var.name_prefix}-runtime-boundary"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Effect = "Allow", Action = ["dynamodb:PutItem", "dynamodb:GetItem", "dynamodb:DeleteItem"], Resource = [local.table_arn] },
      { Effect = "Allow", Action = ["dynamodb:Query"], Resource = ["${local.table_arn}/index/GSI1"] },
      { Effect = "Allow", Action = ["events:PutEvents"], Resource = [local.bus_arn] },
      { Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = ["${local.base_arn}:logs:${var.aws_region}:${var.aws_account_id}:log-group:/aws/lambda/${var.name_prefix}-*:*"] },
      { Effect = "Allow", Action = ["xray:PutTraceSegments", "xray:PutTelemetryRecords"], Resource = ["*"] }
    ]
  })
}
resource "aws_iam_role_policy" "state" {
  name = "application-state"
  role = aws_iam_role.deployment.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Sid = "ListStatePrefix", Effect = "Allow", Action = ["s3:ListBucket"], Resource = [aws_s3_bucket.state.arn], Condition = { StringLike = { "s3:prefix" = ["dev/*"] } } },
      { Sid = "ReadWriteState", Effect = "Allow", Action = ["s3:GetObject", "s3:PutObject"], Resource = ["${aws_s3_bucket.state.arn}/${var.state_key}"] },
      { Sid = "ManageLock", Effect = "Allow", Action = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"], Resource = ["${aws_s3_bucket.state.arn}/${var.state_key}.tflock"] }
    ]
  })
}
resource "aws_iam_role_policy" "deployment" {
  name = "application-infrastructure"
  role = aws_iam_role.deployment.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Sid = "CreateBoundedRoles", Effect = "Allow", Action = ["iam:CreateRole", "iam:PutRolePermissionsBoundary"], Resource = local.role_arns, Condition = { StringEquals = { "iam:PermissionsBoundary" = aws_iam_policy.runtime_boundary.arn } } },
      { Sid = "ManageAppRoles", Effect = "Allow", Action = ["iam:GetRole", "iam:DeleteRole", "iam:UpdateRole", "iam:UpdateAssumeRolePolicy", "iam:PutRolePolicy", "iam:GetRolePolicy", "iam:DeleteRolePolicy", "iam:ListRolePolicies", "iam:ListAttachedRolePolicies", "iam:ListInstanceProfilesForRole", "iam:TagRole", "iam:UntagRole", "iam:ListRoleTags"], Resource = local.role_arns },
      { Sid = "KeepBoundary", Effect = "Deny", Action = ["iam:DeleteRolePermissionsBoundary"], Resource = local.role_arns },
      { Sid = "PassAppRoles", Effect = "Allow", Action = ["iam:PassRole"], Resource = local.role_arns, Condition = { StringEquals = { "iam:PassedToService" = "lambda.amazonaws.com" } } },
      { Sid = "ReadBoundary", Effect = "Allow", Action = ["iam:GetPolicy", "iam:GetPolicyVersion"], Resource = [aws_iam_policy.runtime_boundary.arn] },
      { Sid = "AppFunctions", Effect = "Allow", Action = ["lambda:CreateFunction", "lambda:GetFunction", "lambda:GetFunctionConfiguration", "lambda:UpdateFunctionCode", "lambda:UpdateFunctionConfiguration", "lambda:DeleteFunction", "lambda:PublishVersion", "lambda:ListVersionsByFunction", "lambda:CreateAlias", "lambda:GetAlias", "lambda:UpdateAlias", "lambda:DeleteAlias", "lambda:ListAliases", "lambda:AddPermission", "lambda:RemovePermission", "lambda:GetPolicy", "lambda:TagResource", "lambda:UntagResource", "lambda:ListTags"], Resource = local.function_arns },
      { Sid = "AppTable", Effect = "Allow", Action = ["dynamodb:CreateTable", "dynamodb:DescribeTable", "dynamodb:UpdateTable", "dynamodb:DeleteTable", "dynamodb:DescribeContinuousBackups", "dynamodb:UpdateContinuousBackups", "dynamodb:DescribeTimeToLive", "dynamodb:UpdateTimeToLive", "dynamodb:ListTagsOfResource", "dynamodb:TagResource", "dynamodb:UntagResource"], Resource = [local.table_arn] },
      { Sid = "AppEventBus", Effect = "Allow", Action = ["events:CreateEventBus", "events:DescribeEventBus", "events:UpdateEventBus", "events:DeleteEventBus", "events:ListTagsForResource", "events:TagResource", "events:UntagResource"], Resource = [local.bus_arn] },
      { Sid = "AppLogs", Effect = "Allow", Action = ["logs:CreateLogGroup", "logs:DeleteLogGroup", "logs:PutRetentionPolicy", "logs:DeleteRetentionPolicy", "logs:ListTagsForResource", "logs:TagResource", "logs:UntagResource"], Resource = local.log_arns },
      { Sid = "DescribeLogs", Effect = "Allow", Action = ["logs:DescribeLogGroups"], Resource = ["*"] },
      { Sid = "CreateTaggedUserPool", Effect = "Allow", Action = ["cognito-idp:CreateUserPool", "cognito-idp:TagResource"], Resource = ["*"], Condition = { StringEquals = { "aws:RequestTag/Project" = "url-shortener", "aws:RequestTag/Environment" = "dev" } } },
      { Sid = "AppUserPool", Effect = "Allow", Action = ["cognito-idp:DescribeUserPool", "cognito-idp:UpdateUserPool", "cognito-idp:DeleteUserPool", "cognito-idp:CreateUserPoolClient", "cognito-idp:DescribeUserPoolClient", "cognito-idp:UpdateUserPoolClient", "cognito-idp:DeleteUserPoolClient", "cognito-idp:CreateUserPoolDomain", "cognito-idp:UpdateUserPoolDomain", "cognito-idp:DeleteUserPoolDomain", "cognito-idp:CreateIdentityProvider", "cognito-idp:DescribeIdentityProvider", "cognito-idp:UpdateIdentityProvider", "cognito-idp:DeleteIdentityProvider", "cognito-idp:ListTagsForResource", "cognito-idp:TagResource", "cognito-idp:UntagResource"], Resource = ["${local.base_arn}:cognito-idp:${var.aws_region}:${var.aws_account_id}:userpool/*"], Condition = { StringEquals = { "aws:ResourceTag/Project" = "url-shortener", "aws:ResourceTag/Environment" = "dev" } } },
      { Sid = "DescribeUserPoolDomain", Effect = "Allow", Action = ["cognito-idp:DescribeUserPoolDomain"], Resource = ["*"], Condition = { StringEquals = { "aws:RequestedRegion" = var.aws_region } } },
      { Sid = "CreateTaggedHttpApi", Effect = "Allow", Action = ["apigateway:POST"], Resource = ["${local.base_arn}:apigateway:${var.aws_region}::/apis"], Condition = { StringEquals = { "aws:RequestTag/Project" = "url-shortener", "aws:RequestTag/Environment" = "dev" } } },
      { Sid = "AppHttpApi", Effect = "Allow", Action = ["apigateway:GET", "apigateway:POST", "apigateway:PUT", "apigateway:PATCH", "apigateway:DELETE"], Resource = ["${local.base_arn}:apigateway:${var.aws_region}::/apis/*"], Condition = { StringEquals = { "aws:ResourceTag/Project" = "url-shortener", "aws:ResourceTag/Environment" = "dev" } } },
      # API Gateway v2 child resources do not inherit parent tags. Scope them by IDs/path instead.
      { Sid = "AppHttpApiChildren", Effect = "Allow", Action = ["apigateway:GET", "apigateway:POST", "apigateway:PUT", "apigateway:PATCH", "apigateway:DELETE"], Resource = local.api_child_arns },
      { Sid = "AppApiTags", Effect = "Allow", Action = ["apigateway:GET", "apigateway:PUT", "apigateway:DELETE"], Resource = local.api_tag_arns },
      { Sid = "ProtectApiOwnershipTags", Effect = "Deny", Action = ["apigateway:PUT", "apigateway:DELETE"], Resource = local.api_tag_arns, Condition = { "ForAnyValue:StringEqualsIgnoreCase" = { "aws:TagKeys" = ["Project", "Environment"] } } }
    ]
  })
}
