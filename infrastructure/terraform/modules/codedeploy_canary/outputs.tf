output "application_name" { value = aws_codedeploy_app.resolve.name }
output "deployment_group_name" { value = aws_codedeploy_deployment_group.resolve.deployment_group_name }
output "role_arn" { value = aws_iam_role.service.arn }
