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

variable "state_bucket_name" {
  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$", var.state_bucket_name))
    error_message = "Use a valid bucket name without wildcards or dots."
  }
  type        = string
  description = "Globally unique private state bucket name."
}

variable "state_key" {
  validation {
    condition     = startswith(var.state_key, "dev/") && endswith(var.state_key, ".tfstate") && !strcontains(var.state_key, "..") && !strcontains(var.state_key, "*") && !strcontains(var.state_key, "?")
    error_message = "State key must be an exact .tfstate object under dev/."
  }
  type    = string
  default = "dev/terraform.tfstate"
}

variable "github_repository" {
  validation {
    condition     = can(regex("^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$", var.github_repository))
    error_message = "Use an exact owner/repository without wildcards."
  }
  type    = string
  default = "TrieuNguyenPhu/shorten-link"
}

variable "github_environment" {
  validation {
    condition     = var.github_environment == null ? true : can(regex("^[A-Za-z0-9_-]+$", var.github_environment))
    error_message = "Use an exact environment name without wildcards or colons."
  }
  type        = string
  default     = "dev"
  nullable    = true
  description = "Exact protected environment name; null allows refs/heads/main instead."
}

variable "table_name" {
  type    = string
  default = "url-shortener"
}

variable "event_bus_name" {
  type    = string
  default = "url-shortener-events"
}

variable "http_api_ids" {
  type        = set(string)
  default     = []
  description = "Known HTTP API IDs for child-resource IAM scope. Empty permits bootstrap discovery within this region; pin the output API ID after first approved deployment."
  validation {
    condition     = alltrue([for id in var.http_api_ids : can(regex("^[a-z0-9]+$", id))])
    error_message = "Use actual HTTP API IDs, without wildcard characters."
  }
}

variable "cognito_user_pool_ids" {
  type        = set(string)
  default     = []
  description = "Exact user pool IDs provisioned by the operator; deployment cannot create or claim other pools."
  validation {
    condition     = alltrue([for id in var.cognito_user_pool_ids : startswith(id, "${var.aws_region}_") && can(regex("^[a-z0-9-]+_[A-Za-z0-9]+$", id))])
    error_message = "Use actual Cognito pool IDs from this region, without wildcards."
  }
}
