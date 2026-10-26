variable "region" {
  type    = string
  default = "ap-southeast-2"
}
variable "monthly_budget_usd" {
  type    = number
  default = 4
  validation {
    condition     = var.monthly_budget_usd > 0
    error_message = "Monthly budget must be positive."
  }
}
variable "budget_notification_emails" {
  type        = list(string)
  default     = []
  description = "Optional email addresses to notify at 80% actual and 100% forecasted monthly spend."
  validation {
    condition     = alltrue([for email in var.budget_notification_emails : can(regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$", email))])
    error_message = "Each budget notification entry must be a valid email address."
  }
}
variable "project" {
  type    = string
  default = "shortly"
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{2,19}$", var.project))
    error_message = "Project must be 3–20 lowercase letters, digits or hyphens."
  }
}
variable "environment" {
  type    = string
  default = "dev"
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Use dev, staging or prod."
  }
}
variable "tags" {
  type    = map(string)
  default = {}
}
variable "web_origin" {
  type        = string
  default     = "https://npt-shortenlink.dev"
  description = "Allowed browser origin for direct API Gateway requests. Same-origin CloudFront requests do not need CORS; Cognito URLs follow site_url."
  validation {
    condition     = can(regex("^https://[a-zA-Z0-9.-]+(:[0-9]+)?$", var.web_origin))
    error_message = "Use exactly one HTTPS origin without a path or trailing slash."
  }
}
variable "site_domain" {
  type        = string
  default     = ""
  description = "Optional custom hostname. Empty uses the assigned CloudFront hostname. DNS changes are a separate deployment step."
  validation {
    condition     = (var.site_domain == "") == (var.acm_certificate_arn == null)
    error_message = "Set both site_domain and an existing us-east-1 ACM certificate ARN, or neither."
  }
}
variable "acm_certificate_arn" {
  type     = string
  default  = null
  nullable = true
  validation {
    condition     = var.acm_certificate_arn == null ? true : can(regex("^arn:aws:acm:us-east-1:[0-9]{12}:certificate/", var.acm_certificate_arn))
    error_message = "CloudFront needs an existing certificate in us-east-1."
  }
}
variable "throttle_rate" {
  type    = number
  default = 5
  validation {
    condition     = var.throttle_rate > 0
    error_message = "Throttle rate must be positive."
  }
}
variable "throttle_burst" {
  type    = number
  default = 10
  validation {
    condition     = var.throttle_burst >= 1 && floor(var.throttle_burst) == var.throttle_burst
    error_message = "Burst must be a positive integer."
  }
}
variable "log_retention_days" {
  type    = number
  default = 14
  validation {
    condition     = contains([1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365], var.log_retention_days)
    error_message = "Use a supported bounded CloudWatch retention period."
  }
}
locals { name = "${var.project}-${var.environment}" }
