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
