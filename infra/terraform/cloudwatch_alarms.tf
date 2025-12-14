# SNS topic for CloudWatch alarms
resource "aws_sns_topic" "aideas_api_alerts" {
  name = "magda-api-alerts"

  tags = {
    Name        = "magda-api-alerts"
    Environment = "production"
  }
}

# SNS topic subscription (email)
resource "aws_sns_topic_subscription" "aideas_api_alerts_email" {
  topic_arn = aws_sns_topic.aideas_api_alerts.arn
  protocol  = "email"
  endpoint  = var.alert_email
}

# CloudWatch Log Metric Filter for errors
resource "aws_cloudwatch_log_metric_filter" "api_errors" {
  name           = "magda-api-errors"
  log_group_name = "/magda-api/app"
  pattern        = "[time, level=ERROR*, ...]"

  metric_transformation {
    name      = "APIErrorCount"
    namespace = "AIDEASApi"
    value     = "1"
  }
}

# CloudWatch Log Metric Filter for panics/crashes
resource "aws_cloudwatch_log_metric_filter" "api_panics" {
  name           = "magda-api-panics"
  log_group_name = "/magda-api/app"
  pattern        = "[time, level, msg=*panic*]"

  metric_transformation {
    name      = "APIPanicCount"
    namespace = "AIDEASApi"
    value     = "1"
  }
}

# CloudWatch Log Metric Filter for container restarts
resource "aws_cloudwatch_log_metric_filter" "container_restarts" {
  name           = "magda-api-restarts"
  log_group_name = "/magda-api/app"
  pattern        = "Starting server on port"

  metric_transformation {
    name      = "ContainerRestartCount"
    namespace = "AIDEASApi"
    value     = "1"
  }
}

# Alarm: High error rate
resource "aws_cloudwatch_metric_alarm" "high_error_rate" {
  alarm_name          = "magda-api-high-error-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "APIErrorCount"
  namespace           = "AIDEASApi"
  period              = "300" # 5 minutes
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "Triggers when API error rate exceeds 10 errors in 5 minutes"
  alarm_actions       = [aws_sns_topic.aideas_api_alerts.arn]
  treat_missing_data  = "notBreaching"

  tags = {
    Name        = "magda-api-high-error-rate"
    Environment = "production"
  }
}

# Alarm: Container crashes/panics
resource "aws_cloudwatch_metric_alarm" "container_panic" {
  alarm_name          = "magda-api-container-panic"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "APIPanicCount"
  namespace           = "AIDEASApi"
  period              = "60" # 1 minute
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "Triggers when API container panics/crashes"
  alarm_actions       = [aws_sns_topic.aideas_api_alerts.arn]
  treat_missing_data  = "notBreaching"

  tags = {
    Name        = "magda-api-container-panic"
    Environment = "production"
  }
}

# Alarm: Frequent container restarts
resource "aws_cloudwatch_metric_alarm" "frequent_restarts" {
  alarm_name          = "magda-api-frequent-restarts"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ContainerRestartCount"
  namespace           = "AIDEASApi"
  period              = "300" # 5 minutes
  statistic           = "Sum"
  threshold           = "3"
  alarm_description   = "Triggers when container restarts more than 3 times in 5 minutes"
  alarm_actions       = [aws_sns_topic.aideas_api_alerts.arn]
  treat_missing_data  = "notBreaching"

  tags = {
    Name        = "magda-api-frequent-restarts"
    Environment = "production"
  }
}

# Alarm: EC2 instance CPU utilization
resource "aws_cloudwatch_metric_alarm" "high_cpu" {
  alarm_name          = "magda-api-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "300" # 5 minutes
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "Triggers when CPU utilization exceeds 80% for 10 minutes"
  alarm_actions       = [aws_sns_topic.aideas_api_alerts.arn]

  dimensions = {
    InstanceId = aws_instance.aideas_api.id
  }

  tags = {
    Name        = "magda-api-high-cpu"
    Environment = "production"
  }
}

# Alarm: EC2 instance status check failed
resource "aws_cloudwatch_metric_alarm" "instance_status_check" {
  alarm_name          = "magda-api-instance-status-check-failed"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "StatusCheckFailed"
  namespace           = "AWS/EC2"
  period              = "60"
  statistic           = "Maximum"
  threshold           = "0"
  alarm_description   = "Triggers when EC2 instance status check fails"
  alarm_actions       = [aws_sns_topic.aideas_api_alerts.arn]

  dimensions = {
    InstanceId = aws_instance.aideas_api.id
  }

  tags = {
    Name        = "magda-api-instance-status-check-failed"
    Environment = "production"
  }
}

# Alarm: Disk space utilization
resource "aws_cloudwatch_metric_alarm" "high_disk_usage" {
  alarm_name          = "magda-api-high-disk-usage"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "disk_used_percent"
  namespace           = "CWAgent"
  period              = "300"
  statistic           = "Average"
  threshold           = "85"
  alarm_description   = "Triggers when disk usage exceeds 85%"
  alarm_actions       = [aws_sns_topic.aideas_api_alerts.arn]
  treat_missing_data  = "notBreaching"

  dimensions = {
    InstanceId = aws_instance.aideas_api.id
    path       = "/"
  }

  tags = {
    Name        = "magda-api-high-disk-usage"
    Environment = "production"
  }
}

# Output SNS topic ARN
output "sns_topic_arn" {
  description = "SNS topic ARN for CloudWatch alerts"
  value       = aws_sns_topic.aideas_api_alerts.arn
}
