output "alarm_topic_arn" { value = aws_sns_topic.alarms.arn }
output "canary_alarm_names" { value = [aws_cloudwatch_metric_alarm.resolve["errors"].alarm_name, aws_cloudwatch_metric_alarm.resolve["throttles"].alarm_name] }
output "dashboard_name" { value = aws_cloudwatch_dashboard.operations.dashboard_name }
