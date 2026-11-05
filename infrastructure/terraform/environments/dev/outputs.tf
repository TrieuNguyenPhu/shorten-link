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
output "lambda_roles" { value = merge({ for key, fn in module.functions : key => fn.role_arn }, { click-aggregator = module.event_pipeline.aggregator_role_arn }, { for key, fn in module.realtime_functions : key => fn.role_arn }) }
output "lambda_aliases" { value = merge({ for key, fn in module.functions : key => fn.alias_arn }, { click-aggregator = module.event_pipeline.aggregator_alias_arn }, { for key, fn in module.realtime_functions : key => fn.alias_arn }) }
output "resolve_published_version" { value = module.functions["resolve-link"].version }
output "click_queue_arn" { value = module.event_pipeline.click_queue_arn }
output "click_queue_url" { value = module.event_pipeline.click_queue_url }
output "click_dlq_arn" { value = module.event_pipeline.click_dlq_arn }
output "click_dlq_url" { value = module.event_pipeline.click_dlq_url }
output "click_rule_arn" { value = module.event_pipeline.click_rule_arn }
output "click_event_source_mapping_id" { value = module.event_pipeline.event_source_mapping_id }

output "websocket_api_id" { value = module.websocket_api.id }
output "websocket_endpoint" { value = module.websocket_api.endpoint }
output "websocket_management_endpoint" { value = module.websocket_api.management_endpoint }
output "websocket_connections_arn" { value = module.websocket_api.connections_arn }
output "moderation_state_machine_arn" { value = module.moderation_sfn.arn }
output "moderation_role_arn" { value = module.moderation_sfn.role_arn }

output "frontend_bucket_name" { value = module.frontend_hosting.bucket_name }
output "cloudfront_distribution_id" { value = module.frontend_hosting.distribution_id }
output "cloudfront_distribution_domain_name" { value = module.frontend_hosting.distribution_domain_name }
output "frontend_url" { value = module.frontend_hosting.url }
output "frontend_resource_ids" { value = module.frontend_hosting.resource_ids }

output "alarm_topic_arn" { value = module.observability.alarm_topic_arn }
output "canary_alarm_names" { value = module.observability.canary_alarm_names }
output "dashboard_name" { value = module.observability.dashboard_name }
output "codedeploy_application_name" { value = module.codedeploy_canary.application_name }
output "codedeploy_deployment_group_name" { value = module.codedeploy_canary.deployment_group_name }
output "resolve_function_name" { value = module.functions["resolve-link"].function_name }
