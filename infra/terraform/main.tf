terraform {
  required_version = ">= 1.5.0"

  # Remote state backend for team collaboration and state locking
  backend "s3" {
    bucket         = "aideas-terraform-state"
    key            = "magda-api/terraform.tfstate"
    region         = "eu-west-2"
    dynamodb_table = "aideas-terraform-locks"
    encrypt        = true
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = ">= 4.0"
    }
  }
}

# Cloudflare provider configuration
provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

# Use default VPC instead of creating new one
data "aws_vpc" "default" {
  default = true
}

# Get available subnets
data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

# Get first available subnet
data "aws_subnet" "default" {
  id = data.aws_subnets.default.ids[0]
}

# Security group for magda-api
resource "aws_security_group" "aideas_api" {
  name   = "magda-api-sg"
  vpc_id = data.aws_vpc.default.id

  # SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "SSH access"
  }

  # HTTP access
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP"
  }

  # HTTPS access
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS"
  }

  # Outbound access
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = {
    Name = "magda-api-sg"
  }
}

# Allow API server to access RDS database
# Note: This rule already exists manually - not managed by Terraform
# resource "aws_security_group_rule" "api_to_rds" {
#   type                     = "ingress"
#   from_port                = 5432
#   to_port                  = 5432
#   protocol                 = "tcp"
#   security_group_id        = "sg-067490199295a88a9" # aideas-music-db RDS security group
#   source_security_group_id = aws_security_group.aideas_api.id
#   description              = "Allow API server to connect to database"
# }

# IAM role for EC2 instance
resource "aws_iam_role" "aideas_api_ec2" {
  name = "magda-api-ec2-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name = "magda-api-ec2-role"
  }
}

# Custom ECR policy for Docker login and image pulling
resource "aws_iam_policy" "aideas_api_ecr" {
  name        = "magda-api-ecr-policy"
  description = "Allow ECR login and image pulling for magda-api"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ecr:GetAuthorizationToken",
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage"
        ]
        Resource = "*"
      }
    ]
  })
}

# Attach custom ECR policy
resource "aws_iam_role_policy_attachment" "aideas_api_ecr" {
  role       = aws_iam_role.aideas_api_ec2.name
  policy_arn = aws_iam_policy.aideas_api_ecr.arn
}

# CloudWatch Logs policy for Docker logging
# Note: This policy already exists manually - not managed by Terraform
# resource "aws_iam_policy" "cloudwatch_logs" {
#   name        = "magda-api-cloudwatch-logs"
#   description = "Allow Docker containers to write logs to CloudWatch"
#
#   policy = jsonencode({
#     Version = "2012-10-17"
#     Statement = [
#       {
#         Effect = "Allow"
#         Action = [
#           "logs:CreateLogStream",
#           "logs:PutLogEvents",
#           "logs:DescribeLogStreams"
#         ]
#         Resource = [
#           "arn:aws:logs:${var.region}:${data.aws_caller_identity.current.account_id}:log-group:/magda-api/*:*"
#         ]
#       }
#     ]
#   })
# }

# resource "aws_iam_role_policy_attachment" "cloudwatch_logs" {
#   role       = aws_iam_role.aideas_api_ec2.name
#   policy_arn = aws_iam_policy.cloudwatch_logs.arn
# }

# Instance profile
resource "aws_iam_instance_profile" "aideas_api_ec2" {
  name = "magda-api-ec2-profile"
  role = aws_iam_role.aideas_api_ec2.name
}

# Ubuntu 22.04 AMD64 AMI
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
}

# User data script for EC2 initialization
locals {
  user_data = base64encode(templatefile("${path.module}/user_data.sh", {
    account_id     = data.aws_caller_identity.current.account_id
    region         = var.region
    rds_password   = var.rds_password
    openai_api_key = var.openai_api_key
    jwt_secret     = var.jwt_secret
    mcp_server_url = var.mcp_server_url
  }))
}

# Get current AWS account info
data "aws_caller_identity" "current" {}

# EC2 instance
resource "aws_instance" "aideas_api" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnet.default.id
  vpc_security_group_ids = [aws_security_group.aideas_api.id]
  key_name               = var.ssh_key_name
  iam_instance_profile   = aws_iam_instance_profile.aideas_api_ec2.name

  # ARM64 optimizations
  root_block_device {
    volume_type = "gp3"
    volume_size = 20
    encrypted   = true
  }

  user_data_base64 = local.user_data

  tags = {
    Name        = "magda-api-production"
    Environment = var.environment
  }
}

# Elastic IP
resource "aws_eip" "aideas_api" {
  domain = "vpc"
  tags = {
    Name = "magda-api-production-eip"
  }
}

resource "aws_eip_association" "aideas_api" {
  instance_id   = aws_instance.aideas_api.id
  allocation_id = aws_eip.aideas_api.id
}

# Cloudflare DNS Records
# Note: These records already exist manually - not managed by Terraform
# resource "cloudflare_dns_record" "api" {
#   count   = var.cloudflare_zone_id != "" && var.cloudflare_api_token != "" ? 1 : 0
#   zone_id = var.cloudflare_zone_id
#   name    = "api"
#   type    = "A"
#   ttl     = 1
#   content = aws_eip.aideas_api.public_ip
#   proxied = true
#   comment = "AIDEAS API endpoint"
#
#   lifecycle {
#     ignore_changes = [content]
#   }
# }
#
# resource "cloudflare_dns_record" "beta" {
#   count   = var.cloudflare_zone_id != "" && var.cloudflare_api_token != "" ? 1 : 0
#   zone_id = var.cloudflare_zone_id
#   name    = "beta"
#   type    = "A"
#   ttl     = 1
#   content = aws_eip.aideas_api.public_ip
#   proxied = true
#   comment = "AIDEAS Beta signup portal"
#
#   lifecycle {
#     ignore_changes = [content]
#   }
# }
#
# resource "cloudflare_dns_record" "root" {
#   count   = var.cloudflare_zone_id != "" && var.cloudflare_api_token != "" ? 1 : 0
#   zone_id = var.cloudflare_zone_id
#   name    = "@" # Root domain
#   type    = "A"
#   ttl     = 1
#   content = aws_eip.aideas_api.public_ip
#   proxied = true
#   comment = "AIDEAS main website (coming soon page)"
#
#   lifecycle {
#     ignore_changes = [content]
#   }
# }

# CloudWatch alarms
resource "aws_cloudwatch_metric_alarm" "aideas_api_cpu" {
  alarm_name          = "magda-api-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors ec2 cpu utilization"
  alarm_actions       = []

  dimensions = {
    InstanceId = aws_instance.aideas_api.id
  }
}

resource "aws_cloudwatch_metric_alarm" "aideas_api_status" {
  alarm_name          = "magda-api-status-check"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "StatusCheckFailed"
  namespace           = "AWS/EC2"
  period              = "300"
  statistic           = "Average"
  threshold           = "0"
  alarm_description   = "This metric monitors ec2 status check"
  alarm_actions       = []

  dimensions = {
    InstanceId = aws_instance.aideas_api.id
  }
}

# Outputs
output "ec2_public_ip" {
  value       = aws_eip.aideas_api.public_ip
  description = "Public IP address of the EC2 instance"
}

output "ec2_public_dns" {
  value       = aws_instance.aideas_api.public_dns
  description = "Public DNS of the EC2 instance"
}

output "api_url" {
  value       = var.cloudflare_zone_id != "" ? "https://api.musicalaideas.com" : "http://${aws_eip.aideas_api.public_ip}:8080"
  description = "AIDEAS API URL"
}

output "health_url" {
  value       = var.cloudflare_zone_id != "" ? "https://api.musicalaideas.com/health" : "http://${aws_eip.aideas_api.public_ip}:8080/health"
  description = "Health check URL"
}

output "api_domain" {
  value       = var.cloudflare_zone_id != "" ? "api.musicalaideas.com" : null
  description = "API domain name (if Cloudflare is configured)"
}

output "ssh_command" {
  value       = "ssh -i ~/.ssh/${var.ssh_key_name}.pem ubuntu@${aws_eip.aideas_api.public_ip}"
  description = "SSH command to connect to the instance"
}

output "deployment_instructions" {
  value = <<-EOT
    ✅ AIDEAS API Infrastructure Deployed!

    🚀 Connection Info:
    - IP: ${aws_eip.aideas_api.public_ip}
    - API: http://${aws_eip.aideas_api.public_ip}:8080
    - Health: http://${aws_eip.aideas_api.public_ip}:8080/health

    📋 Next Steps:
    1. SSH into instance: ssh -i ~/.ssh/${var.ssh_key_name}.pem ubuntu@${aws_eip.aideas_api.public_ip}
    2. Configure environment: /opt/magda-api/.env
    3. Manual deploy: cd /opt/magda-api && sudo docker compose up -d

    🔄 Automated deployment will work after GitHub secrets are configured:
    - AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
    - SSH_PRIVATE_KEY (contents of ${var.ssh_key_name}.pem)

    💰 Remember to run 'terraform destroy' when done!
  EOT
}
