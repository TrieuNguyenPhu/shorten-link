# Account-wide tracking is conservative: it also includes other account costs.
# No notification recipients are invented; this budget is visible in AWS Console.
resource "aws_budgets_budget" "dev" {
  name         = "${local.name}-monthly-guard"
  budget_type  = "COST"
  limit_amount = tostring(var.monthly_budget_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"
}
