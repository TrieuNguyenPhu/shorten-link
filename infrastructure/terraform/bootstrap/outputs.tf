output "state_bucket_name" { value = aws_s3_bucket.state.id }
output "state_key" { value = var.state_key }
output "deployment_role_arn" { value = aws_iam_role.deployment.arn }
output "runtime_boundary_arn" { value = aws_iam_policy.runtime_boundary.arn }
output "oidc_provider_arn" { value = aws_iam_openid_connect_provider.github.arn }
output "github_subject" { value = jsondecode(aws_iam_role.deployment.assume_role_policy).Statement[0].Condition.StringEquals["token.actions.githubusercontent.com:sub"] }
output "moderation_boundary_arn" { value = aws_iam_policy.moderation_boundary.arn }
output "canary_boundary_arn" { value = aws_iam_policy.canary_boundary.arn }
