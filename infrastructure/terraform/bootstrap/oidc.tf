resource "aws_iam_openid_connect_provider" "github" {
  url            = "https://token.actions.githubusercontent.com"
  client_id_list = ["sts.amazonaws.com"]
  # GitHub TLS is verified using IAM's trusted CA store; no hardcoded thumbprint.
}
resource "aws_iam_role" "deployment" {
  name                 = "${var.name_prefix}-github-deploy"
  path                 = "/url-shortener/bootstrap/"
  max_session_duration = 3600
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.github.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = { StringEquals = {
        "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
        "token.actions.githubusercontent.com:sub" = var.github_environment == null ? "repo:${var.github_repository}:ref:refs/heads/main" : "repo:${var.github_repository}:environment:${var.github_environment}"
      } }
    }]
  })
}
