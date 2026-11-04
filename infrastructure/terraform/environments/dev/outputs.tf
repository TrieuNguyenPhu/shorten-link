output "http_api_endpoint" { value = module.http_api.endpoint }
output "http_api_id" { value = module.http_api.id }
output "table_name" { value = module.table.name }
output "table_arn" { value = module.table.arn }
output "event_bus_name" { value = module.event_bus.name }
output "event_bus_arn" { value = module.event_bus.arn }
output "cognito_user_pool_id" { value = module.auth.user_pool_id }
output "cognito_client_id" { value = module.auth.client_id }
output "cognito_issuer" { value = module.auth.issuer }
output "cognito_login_domain" { value = module.auth.login_domain }
output "google_redirect_uri" { value = module.auth.google_redirect_uri }
output "lambda_roles" { value = { for key, fn in module.functions : key => fn.role_arn } }
output "lambda_aliases" { value = { for key, fn in module.functions : key => fn.alias_arn } }
output "resolve_published_version" { value = module.functions["resolve-link"].version }
