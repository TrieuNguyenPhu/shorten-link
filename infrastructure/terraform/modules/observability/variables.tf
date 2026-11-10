variable "name_prefix" { type = string }
variable "aws_region" { type = string }
variable "resolve_function_name" { type = string }
variable "queue_name" { type = string }
variable "dlq_name" { type = string }
variable "delivery_dlq_name" { type = string }
variable "alarm_email_addresses" {
  type    = set(string)
  default = []
  validation {
    condition     = alltrue([for email in var.alarm_email_addresses : can(regex("^[^@[:space:]]+@[^@[:space:]]+[.][^@[:space:]]+$", email))])
    error_message = "Provide explicit recipient email addresses; recipients must confirm SNS subscriptions."
  }
}
