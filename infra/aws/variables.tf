variable "aws_region" {
  type    = string
  default = "ap-southeast-2"
}

variable "deployment_name" {
  type    = string
  default = "npt-shortenlink-prod"
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,39}$", var.deployment_name))
    error_message = "Use 2-40 lowercase letters, numbers and hyphens."
  }
}

variable "domain_name" {
  type    = string
  default = "npt-shortenlink.dev"
  validation {
    condition     = can(regex("^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$", var.domain_name))
    error_message = "Use a lowercase DNS name."
  }
}

variable "certificate_arn" {
  type        = string
  description = "Issued ACM certificate in us-east-1 covering domain_name; never an AWS key."
  validation {
    condition     = can(regex("^arn:aws:acm:us-east-1:[0-9]{12}:certificate/[0-9a-fA-F-]+$", var.certificate_arn))
    error_message = "CloudFront requires an ACM certificate ARN from us-east-1."
  }
}
