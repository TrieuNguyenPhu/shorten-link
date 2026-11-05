variable "name_prefix" { type = string }
variable "bucket_name" { type = string }
variable "domain_name" {
  type        = string
  default     = ""
  description = "Optional lowercase custom domain for the frontend CloudFront distribution."
  validation {
    condition     = var.domain_name == "" || can(regex("^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$", var.domain_name))
    error_message = "Use a lowercase fully qualified domain name without a scheme or path."
  }
}
variable "certificate_arn" {
  type        = string
  default     = ""
  description = "Optional ACM certificate ARN in us-east-1 that covers domain_name."
  validation {
    condition     = var.certificate_arn == "" || can(regex("^arn:aws:acm:us-east-1:[0-9]{12}:certificate/[0-9a-f-]{36}$", var.certificate_arn))
    error_message = "Use an ACM certificate ARN from us-east-1."
  }
}
