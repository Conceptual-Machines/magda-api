#!/bin/bash

# AIDEAS API - Infrastructure Deployment Script
# Usage: ./deploy-infrastructure.sh [key-pair-name] [region]

set -e

KEY_PAIR=${1:-"magda-api"}
REGION=${2:-"eu-west-2"}

echo "🚀 Deploying AIDEAS API Infrastructure"
echo "Key Pair: $KEY_PAIR"
echo "Region: $REGION"

# Check if AWS CLI is configured
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    echo "❌ AWS CLI not configured. Please run 'aws configure' first"
    exit 1
fi

# Check if key pair exists
if ! aws ec2 describe-key-pairs --key-names "$KEY_PAIR" --region "$REGION" > /dev/null 2>&1; then
    echo "❌ Key pair '$KEY_PAIR' not found in region '$REGION'"
    echo "Please create it first: aws ec2 create-key-pair --key-name $KEY_PAIR --region $REGION"
    exit 1
fi

cd infra/terraform

# Initialize Terraform
echo "📦 Initializing Terraform..."
terraform init

# Create terraform.tfvars
cat > terraform.tfvars << EOF
region         = "$REGION"
instance_type  = "t3.nano"
ssh_key_name   = "$KEY_PAIR"
environment    = "production"
EOF

# Plan deployment
echo "📋 Planning deployment..."
terraform plan

# Ask for confirmation
echo ""
echo "⚠️  This will create AWS resources that will cost money!"
echo "💰 Estimated cost: ~$3/month for EC2 + minimal ECR storage"
read -p "Do you want to proceed? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Deployment cancelled"
    exit 1
fi

# Apply deployment
echo "🚀 Deploying infrastructure..."
terraform apply -auto-approve

# Get outputs
echo ""
echo "✅ Infrastructure deployed successfully!"
echo ""
echo "📋 Connection Information:"
echo "Public IP: $(terraform output -raw ec2_public_ip)"
echo "API URL: $(terraform output -raw api_url)"
echo "Health URL: $(terraform output -raw health_url)"
echo ""
echo "🔑 SSH Command:"
echo "$(terraform output -raw ssh_command)"
echo ""
echo "📝 Next Steps:"
echo "1. SSH into instance: $(terraform output -raw ssh_command)"
echo "2. Configure /opt/magda-api/.env with your values:"
echo "   - DATABASE_URL (PostgreSQL connection string)"
echo "   - JWT_SECRET (secure random string)"
echo "   - OPENAI_API_KEY (your OpenAI API key)"
echo "   - MCP_SERVER_URL (optional)"
echo "3. Deploy: cd /opt/magda-api && sudo ./deploy.sh"
echo ""
echo "🔄 For automated deployment:"
echo "- Add SSH_PRIVATE_KEY to GitHub secrets (contents of $KEY_PAIR.pem)"
echo "- Push to main branch → automatic deployment!"
echo ""
echo "💰 Remember to run 'terraform destroy' when you're done!"
