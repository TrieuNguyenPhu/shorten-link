variable "function_name" {
  type = string
}

variable "filename" {
  type = string
}

variable "role_path" {
  type = string
}

variable "runtime_boundary_arn" {
  type = string
}

variable "environment_variables" {
  type = map(string)
}

variable "permissions" {
  type = list(object({ actions = list(string), resources = list(string) }))
}

variable "canary_managed" {
  type    = bool
  default = false
}

variable "log_retention_days" {
  type    = number
  default = 14
}
