variable "name" {
  type = string
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
    condition     = length(var.callback_urls) > 0 && alltrue([for u in var.callback_urls : can(regex("^(https://[^/?#@*[:space:]]+|http://(localhost|127[.]0[.]0[.]1)(:[0-9]+)?)/auth/callback/$", u))])
    error_message = "Callbacks must be exact HTTPS or local HTTP /auth/callback/ URLs."
  }
  type = set(string)
}

variable "logout_urls" {
  validation {
    condition     = length(var.logout_urls) > 0 && alltrue([for u in var.logout_urls : can(regex("^(https://[^/?#@*[:space:]]+|http://(localhost|127[.]0[.]0[.]1)(:[0-9]+)?)/$", u))])
    error_message = "Logout URLs must be HTTPS or local HTTP root URLs."
  }
  type = set(string)
}

variable "deletion_protection_enabled" {
  type    = bool
  default = true
}
