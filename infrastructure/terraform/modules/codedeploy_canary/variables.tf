variable "name_prefix" { type = string }
variable "function_arn" { type = string }
variable "boundary_arn" { type = string }
variable "alarm_names" {
  type = list(string)
  validation {
    condition     = length(var.alarm_names) >= 2 && length(var.alarm_names) <= 10
    error_message = "Canary requires error and throttle alarms (at most ten)."
  }
}
