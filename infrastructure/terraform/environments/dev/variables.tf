variable "aws_region" {
  type    = string
  default = "ap-southeast-1"
}

variable "aws_account_id" {
  validation {
    condition     = can(regex("^[0-9]{12}$", var.aws_account_id))
    error_message = "aws_account_id must contain exactly 12 digits."
  }
  type        = string
  description = "Target account, used by provider allowed_account_ids and IAM resource scopes."
}

variable "name_prefix" {
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{2,35}$", var.name_prefix))
    error_message = "name_prefix must be a lowercase resource prefix, 3 to 36 characters."
  }
  type    = string
  default = "url-shortener-dev"
}

variable "runtime_boundary_arn" {
  type        = string
  default     = null
  description = "Optional explicit runtime boundary ARN; deterministic default matches bootstrap output."
  validation {
    condition     = var.runtime_boundary_arn == null || var.runtime_boundary_arn == "arn:aws:iam::${var.aws_account_id}:policy/url-shortener/bootstrap/${var.name_prefix}-runtime-boundary"
    error_message = "Provide the runtime boundary ARN from bootstrap."
  }
}

variable "table_name" {
  type    = string
  default = "url-shortener"
}

variable "event_bus_name" {
  type    = string
  default = "url-shortener-events"
}

variable "cognito_domain_prefix" {
  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$", var.cognito_domain_prefix))
    error_message = "Use a valid Cognito prefix domain."
  }
  type = string
}

variable "google_client_id" {
  validation {
    condition     = endswith(var.google_client_id, ".apps.googleusercontent.com")
    error_message = "Provide the Google OAuth Web client ID."
  }
  type = string
}

variable "google_client_secret" {
  validation {
    condition     = length(trimspace(var.google_client_secret)) > 0
    error_message = "Google client secret must not be empty."
  }
  type      = string
  sensitive = true
}

variable "callback_urls" {
  validation {
    condition     = alltrue([for u in var.callback_urls : can(regex("^(https://[^/?#@*[:space:]]+|http://(localhost|127[.]0[.]0[.]1)(:[0-9]+)?)/auth/callback/$", u))])
    error_message = "Callbacks must be exact HTTPS or local HTTP /auth/callback/ URLs."
  }
  type    = set(string)
  default = []
}

variable "logout_urls" {
  validation {
    condition     = alltrue([for u in var.logout_urls : can(regex("^(https://[^/?#@*[:space:]]+|http://(localhost|127[.]0[.]0[.]1)(:[0-9]+)?)/$", u))])
    error_message = "Logout URLs must be HTTPS or local HTTP root URLs."
  }
  type    = set(string)
  default = []
}

variable "cors_origins" {
  validation {
    condition     = alltrue([for u in var.cors_origins : can(regex("^(https://[^/?#@*]+|http://(localhost|127[.]0[.]0[.]1)(:[0-9]+)?)$", u))])
    error_message = "CORS origins must be exact HTTPS or local HTTP origins, without a wildcard or path."
  }
  type    = set(string)
  default = []
}

variable "lambda_package_directory" {
  type    = string
  default = "../../../../backend/dist"
}

variable "deletion_protection_enabled" {
  type    = bool
  default = true
}

variable "log_retention_days" {
  type    = number
  default = 365
}

variable "moderation_boundary_arn" {
  type        = string
  default     = null
  description = "Optional explicit moderation boundary ARN; deterministic default matches bootstrap output."
  validation {
    condition     = var.moderation_boundary_arn == null || var.moderation_boundary_arn == "arn:aws:iam::${var.aws_account_id}:policy/url-shortener/bootstrap/${var.name_prefix}-moderation-boundary"
    error_message = "Provide the dedicated moderation boundary ARN from bootstrap."
  }
}

variable "canary_boundary_arn" {
  type        = string
  default     = null
  description = "Optional explicit bootstrap ARN; deterministic default supports existing operator inputs."
  validation {
    condition     = var.canary_boundary_arn == null || var.canary_boundary_arn == "arn:aws:iam::${var.aws_account_id}:policy/url-shortener/bootstrap/${var.name_prefix}-canary-boundary"
    error_message = "Provide the dedicated canary boundary ARN from bootstrap."
  }
}
variable "alarm_email_addresses" {
  type    = set(string)
  default = []
}
