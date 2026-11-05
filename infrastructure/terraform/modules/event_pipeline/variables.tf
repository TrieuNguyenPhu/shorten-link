variable "name_prefix" {
  type = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{2,35}$", var.name_prefix))
    error_message = "Use a lowercase resource prefix, 3 to 36 characters."
  }
}
variable "event_bus_name" {
  type = string
  validation {
    condition     = can(regex("^[A-Za-z0-9._-]{1,256}$", var.event_bus_name)) && var.event_bus_name != "default"
    error_message = "Provide the existing custom bus name, not the default bus or an ARN."
  }
}
variable "event_bus_arn" {
  type = string
  validation {
    condition     = can(regex("^arn:[a-z0-9-]+:events:[a-z0-9-]+:[0-9]{12}:event-bus/[A-Za-z0-9._-]+$", var.event_bus_arn)) && endswith(var.event_bus_arn, ":event-bus/${var.event_bus_name}")
    error_message = "The existing custom bus ARN must match event_bus_name."
  }
}
variable "table_name" { type = string }
variable "table_arn" { type = string }
variable "runtime_boundary_arn" { type = string }
variable "aggregator_filename" { type = string }
variable "log_retention_days" {
  type    = number
  default = 14
}

variable "ws_endpoint" {
  type = string
  validation {
    condition     = try(startswith(var.ws_connections_arn, "arn:aws:execute-api:${split(":", var.table_arn)[3]}:${split(":", var.table_arn)[4]}:") && var.ws_endpoint == "https://${split("/", split(":", var.ws_connections_arn)[5])[0]}.execute-api.${split(":", var.table_arn)[3]}.amazonaws.com/prod", false) && can(regex("^https://[a-z0-9]+[.]execute-api[.][a-z0-9-]+[.]amazonaws[.]com/prod$", var.ws_endpoint))
    error_message = "Provide the HTTPS prod management endpoint together with its connections ARN."
  }
}
variable "ws_connections_arn" {
  type = string
  validation {
    condition     = can(regex("^arn:aws:execute-api:[a-z0-9-]+:[0-9]{12}:[a-z0-9]+/prod/POST/@connections/[*]$", var.ws_connections_arn))
    error_message = "ManageConnections must be limited to POST /prod/@connections on one API."
  }
}
