variable "name" { type = string }
variable "aws_account_id" { type = string }
variable "lambda_targets" {
  type = map(object({ function_name = string, alias_invoke_arn = string }))
  validation {
    condition     = toset(keys(var.lambda_targets)) == toset(["ws-connect", "ws-disconnect"])
    error_message = "Provide exactly the two WebSocket handlers."
  }
}
