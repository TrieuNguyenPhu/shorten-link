# Account-wide tracking is conservative: it also includes other account costs.
resource "aws_budgets_budget" "dev" {
  name         = "${local.name}-monthly-guard"
  budget_type  = "COST"
  limit_amount = tostring(var.monthly_budget_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  dynamic "notification" {
    for_each = length(var.budget_notification_emails) > 0 ? {
      actual     = { type = "ACTUAL", threshold = 80 }
      forecasted = { type = "FORECASTED", threshold = 100 }
    } : {}
    content {
      comparison_operator        = "GREATER_THAN"
      threshold                  = notification.value.threshold
      threshold_type             = "PERCENTAGE"
      notification_type          = notification.value.type
      subscriber_email_addresses = var.budget_notification_emails
    }
  }
}
