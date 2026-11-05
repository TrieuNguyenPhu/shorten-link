variable "name" {
  type = string
}

variable "jwt_issuer" {
  type = string
}

variable "jwt_audience" {
  type = string
}

variable "cors_origins" {
  validation {
    condition     = length(var.cors_origins) > 0 && alltrue([for u in var.cors_origins : can(regex("^(https://[^/?#@*]+|http://(localhost|127[.]0[.]0[.]1)(:[0-9]+)?)$", u))])
    error_message = "CORS origins must be exact HTTPS or local HTTP origins, without a wildcard or path."
  }
  type = set(string)
}

variable "lambda_targets" {
  type = map(object({ function_name = string, alias_invoke_arn = string }))
}

variable "log_retention_days" {
  type    = number
  default = 14
}

variable "throttling_burst_limit" {
  type    = number
  default = 200
}

variable "throttling_rate_limit" {
  type    = number
  default = 100
}
