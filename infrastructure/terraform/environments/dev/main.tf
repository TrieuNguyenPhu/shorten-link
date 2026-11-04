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
module "auth" {
  source                      = "../../modules/cognito"
  name                        = var.name_prefix
  cognito_domain_prefix       = var.cognito_domain_prefix
  google_client_id            = var.google_client_id
  google_client_secret        = var.google_client_secret
  callback_urls               = var.callback_urls
  logout_urls                 = var.logout_urls
  deletion_protection_enabled = var.deletion_protection_enabled
}
locals {
  permissions = {
    create-link  = [{ actions = ["dynamodb:PutItem"], resources = [module.table.arn] }]
    resolve-link = [{ actions = ["dynamodb:GetItem"], resources = [module.table.arn] }, { actions = ["events:PutEvents"], resources = [module.event_bus.arn] }]
    list-links   = [{ actions = ["dynamodb:Query"], resources = [module.table.index_arn] }]
    delete-link  = [{ actions = ["dynamodb:DeleteItem"], resources = [module.table.arn] }]
  }
}
module "functions" {
  for_each             = local.permissions
  source               = "../../modules/lambda"
  function_name        = "${var.name_prefix}-${each.key}"
  filename             = abspath("${path.root}/${var.lambda_package_directory}/${each.key}.zip")
  role_path            = "/url-shortener/dev/"
  runtime_boundary_arn = var.runtime_boundary_arn
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
  cors_origins       = var.cors_origins
  log_retention_days = var.log_retention_days
  lambda_targets = { for key, fn in module.functions : key => {
    function_name = fn.function_name, alias_invoke_arn = fn.alias_invoke_arn
  } }
}
