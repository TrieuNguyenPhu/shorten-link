variable "name_prefix" { type = string }
variable "bucket_name" { type = string }
variable "domain_name" {
  type    = string
  default = null
  validation {
    condition     = var.domain_name == null || can(regex("^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?\\.[a-z]{2,}$", lower(var.domain_name)))
    error_message = "domain_name must be a public DNS name without a scheme or path."
  }
}
variable "certificate_arn" {
  type    = string
  default = null
  validation {
    condition     = var.certificate_arn == null || can(regex("^arn:[^:]+:acm:us-east-1:[0-9]{12}:certificate/[A-Za-z0-9-]+$", var.certificate_arn))
    error_message = "certificate_arn must be an ACM certificate ARN in us-east-1 for CloudFront."
  }
}
