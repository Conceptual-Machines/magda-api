variable "region" {
  description = "AWS region"
  type        = string
  default     = "eu-west-2"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.nano" # AMD64 - faster builds, native Docker images
}

variable "ssh_key_name" {
  description = "Name of the SSH key pair for EC2 access"
  type        = string
  default     = "magda-api"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for musicalaideas.com"
  type        = string
  default     = ""
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token for DNS management"
  type        = string
  default     = ""
  sensitive   = true
}

variable "alert_email" {
  description = "Email address for CloudWatch alerts"
  type        = string
  default     = "romagnoli.luca@gmail.com"
}

variable "rds_password" {
  description = "Password for RDS database"
  type        = string
  sensitive   = true
}

variable "openai_api_key" {
  description = "OpenAI API key"
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "JWT secret for API authentication"
  type        = string
  sensitive   = true
}

variable "mcp_server_url" {
  description = "MCP Server URL"
  type        = string
  default     = "https://mcp.musicalaideas.com/mcp"
}
