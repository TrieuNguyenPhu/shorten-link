terraform {
  backend "s3" {
    key                  = "dev/terraform.tfstate"
    encrypt              = true
    use_lockfile         = true
    workspace_key_prefix = "dev/workspaces"
  }
}
module "table" {
  source                      = "../../modules/dynamodb"
  table_name                  = var.table_name
  deletion_protection_enabled = var.deletion_protection_enabled
}
module "event_bus" {
  source = "../../modules/event_bus"
  name   = var.event_bus_name
}
module "frontend_hosting" {
  source      = "../../modules/frontend_hosting"
  name_prefix = var.name_prefix
  bucket_name = "${var.name_prefix}-${var.aws_account_id}-frontend"
}

locals {
  runtime_boundary_arn    = coalesce(var.runtime_boundary_arn, "arn:aws:iam::${var.aws_account_id}:policy/url-shortener/bootstrap/${var.name_prefix}-runtime-boundary")
  moderation_boundary_arn = coalesce(var.moderation_boundary_arn, "arn:aws:iam::${var.aws_account_id}:policy/url-shortener/bootstrap/${var.name_prefix}-moderation-boundary")
  canary_boundary_arn     = coalesce(var.canary_boundary_arn, "arn:aws:iam::${var.aws_account_id}:policy/url-shortener/bootstrap/${var.name_prefix}-canary-boundary")
  frontend_callbacks      = setunion(var.callback_urls, ["${module.frontend_hosting.url}/auth/callback/"])
  frontend_logouts        = setunion(var.logout_urls, ["${module.frontend_hosting.url}/"])
  frontend_origins        = setunion(var.cors_origins, [module.frontend_hosting.url])
  permissions = {
    create-link  = [{ actions = ["dynamodb:PutItem"], resources = [module.table.arn] }]
    resolve-link = [{ actions = ["dynamodb:GetItem"], resources = [module.table.arn] }, { actions = ["events:PutEvents"], resources = [module.event_bus.arn] }]
    list-links   = [{ actions = ["dynamodb:Query"], resources = [module.table.index_arn] }]
    delete-link  = [{ actions = ["dynamodb:DeleteItem"], resources = [module.table.arn] }]
  }
  realtime_permissions = {
    ws-connect     = [{ actions = ["dynamodb:PutItem"], resources = [module.table.arn] }]
    ws-disconnect  = [{ actions = ["dynamodb:DeleteItem"], resources = [module.table.arn] }]
    moderate-check = []
  }
}

module "auth" {
  source                      = "../../modules/cognito"
  name                        = var.name_prefix
  cognito_domain_prefix       = var.cognito_domain_prefix
  google_client_id            = var.google_client_id
  google_client_secret        = var.google_client_secret
  callback_urls               = local.frontend_callbacks
  logout_urls                 = local.frontend_logouts
  deletion_protection_enabled = var.deletion_protection_enabled
}
module "functions" {
  for_each             = local.permissions
  source               = "../../modules/lambda"
  function_name        = "${var.name_prefix}-${each.key}"
  filename             = abspath("${path.root}/${var.lambda_package_directory}/${each.key}.zip")
  role_path            = "/url-shortener/dev/"
  runtime_boundary_arn = local.runtime_boundary_arn
  permissions          = each.value
  canary_managed       = each.key == "resolve-link"
  log_retention_days   = var.log_retention_days
  environment_variables = merge({ TABLE_NAME = module.table.name },
    each.key == "create-link" ? { PUBLIC_REDIRECT_BASE_URL = module.http_api.endpoint } : {},
  each.key == "resolve-link" ? { EVENT_BUS = module.event_bus.name } : {})
}
module "http_api" {
  source             = "../../modules/http_api"
  name               = var.name_prefix
  jwt_issuer         = module.auth.issuer
  jwt_audience       = module.auth.client_id
  cors_origins       = local.frontend_origins
  log_retention_days = var.log_retention_days
  lambda_targets = { for key, fn in module.functions : key => {
    function_name = fn.function_name, alias_invoke_arn = fn.alias_invoke_arn
  } }
}

module "event_pipeline" {
  source               = "../../modules/event_pipeline"
  name_prefix          = var.name_prefix
  event_bus_name       = module.event_bus.name
  event_bus_arn        = module.event_bus.arn
  table_name           = module.table.name
  table_arn            = module.table.arn
  runtime_boundary_arn = local.runtime_boundary_arn
  aggregator_filename  = abspath("${path.root}/${var.lambda_package_directory}/click-aggregator.zip")
  log_retention_days   = var.log_retention_days
  ws_endpoint          = module.websocket_api.management_endpoint
  ws_connections_arn   = module.websocket_api.connections_arn
}

module "realtime_functions" {
  for_each              = local.realtime_permissions
  source                = "../../modules/lambda"
  function_name         = "${var.name_prefix}-${each.key}"
  filename              = abspath("${path.root}/${var.lambda_package_directory}/${each.key}.zip")
  role_path             = "/url-shortener/dev/"
  runtime_boundary_arn  = local.runtime_boundary_arn
  permissions           = each.value
  environment_variables = each.key == "moderate-check" ? {} : { TABLE_NAME = module.table.name }
  log_retention_days    = var.log_retention_days
}
module "websocket_api" {
  source         = "../../modules/websocket_api"
  name           = var.name_prefix
  aws_account_id = var.aws_account_id
  lambda_targets = { for key, fn in module.realtime_functions : key => {
    function_name = fn.function_name, alias_invoke_arn = fn.alias_invoke_arn
  } if key != "moderate-check" }
}
module "moderation_sfn" {
  source                  = "../../modules/moderation_sfn"
  name_prefix             = var.name_prefix
  table_name              = module.table.name
  table_arn               = module.table.arn
  moderate_check_arn      = module.realtime_functions["moderate-check"].alias_arn
  moderation_boundary_arn = local.moderation_boundary_arn
}

module "observability" {
  source                = "../../modules/observability"
  name_prefix           = var.name_prefix
  aws_region            = var.aws_region
  resolve_function_name = module.functions["resolve-link"].function_name
  queue_name            = "${var.name_prefix}-ClickQueue"
  dlq_name              = "${var.name_prefix}-ClickDLQ"
  alarm_email_addresses = var.alarm_email_addresses
}
module "codedeploy_canary" {
  source       = "../../modules/codedeploy_canary"
  name_prefix  = var.name_prefix
  function_arn = module.functions["resolve-link"].arn
  boundary_arn = local.canary_boundary_arn
  alarm_names  = module.observability.canary_alarm_names
}
