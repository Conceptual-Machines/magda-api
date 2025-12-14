#!/bin/bash

# AIDEAS API - EC2 Initialization Script
set -euo pipefail

# Update system
echo "📦 Updating system packages..."
apt update && apt upgrade -y

# Install Docker
echo "🐳 Installing Docker..."
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
rm get-docker.sh

# Add ubuntu user to docker group
usermod -aG docker ubuntu

# Install Docker Compose
echo "🐳 Installing Docker Compose..."
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Install required packages for AWS CLI
echo "📦 Installing required packages..."
sudo apt install -y unzip

# Install AWS CLI
echo "☁️ Installing AWS CLI..."
curl "https://awscli.amazonaws.com/awscli-exe-linux-$(uname -m).zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install
rm -rf aws awscliv2.zip

# Configure AWS CLI (automatic via IAM role)
echo "🔧 AWS CLI configured via IAM role"

# Create application directory
echo "📁 Setting up application directory..."
mkdir -p /opt/magda-api
chown ubuntu:ubuntu /opt/magda-api

# Copy docker-compose files from repository
echo "📋 Copying docker-compose files..."
# Note: These files will be copied by the deployment script from the repository

# Generate secure JWT secret
JWT_SECRET=$(openssl rand -base64 32)

# Create production .env file with secure defaults
cat > /opt/magda-api/.env << EOF
# AIDEAS API Environment Configuration
ENVIRONMENT=production
PORT=8080

# Database URL (using Terraform variables)
DATABASE_URL=postgres://aideas_admin:$${RDS_PASSWORD}@aideas-music-db.cdtjlpljy3yd.eu-west-2.rds.amazonaws.com:5432/aideas_api?sslmode=require

# JWT Secret (auto-generated secure secret)
JWT_SECRET=$${JWT_SECRET}

# OpenAI API Key (using environment variable)
OPENAI_API_KEY=$${OPENAI_API_KEY}

# MCP Server URL (using environment variable)
MCP_SERVER_URL=$${MCP_SERVER_URL:-https://mcp.musicalaideas.com/mcp}

# Frontend URL for email links
FRONTEND_URL=$${FRONTEND_URL:-https://beta.musicalaideas.com}

# AWS Region for SES
AWS_REGION=eu-west-2
EOF

# Create deployment script
cat > /opt/magda-api/deploy.sh << 'EOF'
#!/bin/bash

set -euo pipefail

echo "🚀 Deploying AIDEAS API..."

# Get AWS region and account ID
AWS_REGION=$${AWS_REGION:-eu-west-2}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Login to ECR
echo "🔐 Authenticating to Amazon ECR..."
aws ecr get-login-password --region $AWS_REGION | sudo docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com

# Pull latest image
echo "📦 Pulling latest image..."
sudo docker pull $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/magda-api:latest

# Copy docker-compose files from repository (if they exist)
if [ -f "/opt/magda-api/docker-compose.prod.yml" ]; then
    echo "📋 Using existing docker-compose.prod.yml"
else
    echo "⚠️ docker-compose.prod.yml not found, will be created by deployment workflow"
fi

# Restart containers
echo "🔄 Restarting containers..."
cd /opt/magda-api
sudo docker compose -f docker-compose.prod.yml down || true
sudo docker compose -f docker-compose.prod.yml up -d

# Wait for health check
echo "⏳ Waiting for service to be healthy..."
sleep 10

# Check status
if sudo docker ps | grep -q "magda-api"; then
    echo "✅ Deployment complete!"
    sudo docker ps
else
    echo "❌ Deployment failed. Check logs:"
    sudo docker compose -f docker-compose.prod.yml logs
    exit 1
fi
EOF

chmod +x /opt/magda-api/deploy.sh
chown ubuntu:ubuntu /opt/magda-api/deploy.sh

# Set up CloudWatch logs (optional)
echo "📝 Setting up CloudWatch agent..."
curl -sO https://s3.amazonaws.com/amazoncloudwatch-agent/ubuntu/amd64/latest/amazon-cloudwatch-agent.deb
dpkg -i amazon-cloudwatch-agent.deb || true
rm amazon-cloudwatch-agent.deb

# Configure CloudWatch agent
mkdir -p /opt/aws/amazon-cloudwatch-agent/etc

cat > /opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json << EOF
{
  "logs": {
    "logs_collected": {
      "files": {
        "collect_list": [
          {
            "file_path": "/var/log/cloud-init-output.log",
            "log_group_name": "/aideas/api",
            "log_stream_name": "{instance_id}-init"
          }
        ]
      }
    }
  }
}
EOF

systemctl enable amazon-cloudwatch-agent
systemctl start amazon-cloudwatch-agent

echo "✅ EC2 initialization complete!"
echo "📋 Next steps:"
echo "1. Configure /opt/magda-api/.env with your values"
echo "2. Run /opt/magda-api/deploy.sh to deploy"
echo "3. Check logs with: docker compose logs -f"

# Automatically deploy (if environment is configured)
echo "🚀 Attempting automatic deployment..."

# Wait for ECR to be available (sometimes takes a minute)
sleep 30

# Try to deploy
cd /opt/magda-api

# Only deploy if we have actual config (not placeholders)
if grep -q "sk-placeholder" .env || grep -q "rds-endpoint-placeholder" .env; then
    echo "⏳ Skipping deployment - configuration needed"
    echo "📝 Please update /opt/magda-api/.env with your values and run: ./deploy.sh"
else
    echo "🔄 Deploying automatically..."
    ./deploy.sh || echo "⚠️ Auto-deployment failed, manual deployment needed"
fi

# Signal completion
echo "magda-api-init-complete" > /opt/magda-api/init-complete.txt
curl -fsS --retry 3 https://hc-ping.com/YOUR_HEALTH_CHECK_URL || true
