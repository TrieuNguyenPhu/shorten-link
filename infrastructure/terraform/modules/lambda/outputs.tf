output "function_name" { value = aws_lambda_function.handler.function_name }
output "arn" { value = aws_lambda_function.handler.arn }
output "version" { value = aws_lambda_function.handler.version }
output "role_arn" { value = aws_iam_role.execution.arn }
output "alias_arn" { value = one(concat(aws_lambda_alias.live[*].arn, aws_lambda_alias.current[*].arn)) }
output "alias_invoke_arn" { value = one(concat(aws_lambda_alias.live[*].invoke_arn, aws_lambda_alias.current[*].invoke_arn)) }
output "environment_variables" { value = var.environment_variables }
