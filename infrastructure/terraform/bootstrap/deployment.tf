# Operator-owned managed grants keep the deployment role below inline quotas.

resource "aws_iam_policy" "lambda_deployment" {
  name = "${var.name_prefix}-lambda-deploy"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Sid = "AppFunctions", Effect = "Allow", Action = ["lambda:CreateFunction", "lambda:GetFunction", "lambda:GetFunctionConfiguration", "lambda:GetFunctionCodeSigningConfig", "lambda:UpdateFunctionCode", "lambda:UpdateFunctionConfiguration", "lambda:DeleteFunction", "lambda:PublishVersion", "lambda:ListVersionsByFunction", "lambda:CreateAlias", "lambda:GetAlias", "lambda:UpdateAlias", "lambda:DeleteAlias", "lambda:ListAliases", "lambda:AddPermission", "lambda:RemovePermission", "lambda:GetPolicy", "lambda:TagResource", "lambda:UntagResource", "lambda:ListTags"], Resource = local.function_arns }
    ]
  })
}
resource "aws_iam_role_policy_attachment" "lambda" {
  role       = aws_iam_role.deployment.name
  policy_arn = aws_iam_policy.lambda_deployment.arn
}

resource "aws_iam_policy" "api_deployment" {
  name = "${var.name_prefix}-api-deploy"
  path = "/url-shortener/bootstrap/"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Sid = "CreateTaggedHttpApi", Effect = "Allow", Action = ["apigateway:POST"], Resource = ["${local.base_arn}:apigateway:${var.aws_region}::/apis"], Condition = { StringEquals = { "aws:RequestTag/Project" = "url-shortener", "aws:RequestTag/Environment" = "dev" } } },
      { Sid = "AppHttpApi", Effect = "Allow", Action = ["apigateway:GET", "apigateway:POST", "apigateway:PUT", "apigateway:PATCH", "apigateway:DELETE"], Resource = ["${local.base_arn}:apigateway:${var.aws_region}::/apis/*"], Condition = { StringEquals = { "aws:ResourceTag/Project" = "url-shortener", "aws:ResourceTag/Environment" = "dev" } } },
      { Sid = "AppHttpApiChildren", Effect = "Allow", Action = ["apigateway:GET", "apigateway:POST", "apigateway:PUT", "apigateway:PATCH", "apigateway:DELETE"], Resource = local.api_child_arns },
      { Sid = "AppApiTags", Effect = "Allow", Action = ["apigateway:GET", "apigateway:PUT", "apigateway:DELETE"], Resource = local.api_tag_arns },
      { Sid = "ProtectApiOwnershipTags", Effect = "Deny", Action = ["apigateway:PUT", "apigateway:DELETE"], Resource = local.api_tag_arns, Condition = { "ForAnyValue:StringEqualsIgnoreCase" = { "aws:TagKeys" = ["Project", "Environment"] } } }
    ]
  })
}
resource "aws_iam_role_policy_attachment" "api" {
  role       = aws_iam_role.deployment.name
  policy_arn = aws_iam_policy.api_deployment.arn
}
