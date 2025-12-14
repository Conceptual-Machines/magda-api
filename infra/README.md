# AIDEAS API - Infrastructure

This directory contains Terraform configuration for deploying the AIDEAS API to AWS.

## Architecture

- **EC2 Instance**: ARM64 t4g.nano instance (~$2.80/month - 20% cheaper than t3.nano)
- **Elastic IP**: Static IP address for consistent deployment
- **Security Group**: Open ports 22 (SSH) and 8080 (API)
- **IAM Role**: ECR read access for pulling Docker images
- **CloudWatch**: Optional log aggregation

## Prerequisites

1. **AWS CLI configured** with appropriate permissions
2. **SSH Key Pair** in the target region
3. **Terraform** installed locally
4. **Go application** already in ECR repository

## Quick Deploy (FULLY AUTOMATED)

**Option 1: GitHub Actions (Recommended)**

1. **Deploy Infrastructure**:
   - Go to: https://github.com/lucaromagnoli/aideas-api/actions/workflows/infrastructure.yml
   - Click "Run workflow" → Select region & SSH key → Run
   - Infrastructure automatically deploys with EC2, IP, security groups

2. **Configure Environment** (one-time setup):
   ```bash
   ssh -i ~/.ssh/aideas-key.pem ubuntu@YOUR_EC2_IP
   sudo nano /opt/aideas-api/.env
   # Update with your DATABASE_URL and OPENAI_API_KEY
   ```

3. **Deploy Application**:
   ```bash
   cd /opt/a-data-api
   sudo ./deploy.sh
   ```

4. **Enable Fully Automated Deployments**:
   ```bash
   cat ~/.ssh/aideas-key.pem
   # Copy to GitHub → Settings → Secrets → SSH_PRIVATE_KEY
   ```
   - Push to main → **FULLY AUTOMATED DEPLOYMENT**

**Option 2: Command Line**

```bash
# Deploy infrastructure
./deploy-infrastructure.sh aideas-key eu-west-2

# Configure environment variables on EC2
ssh -i ~/.ssh/aideas-key.pem ubuntu@YOUR_EC2_IP
sudo nano /opt/aideas-api/.env
# Edit with your DATABASE_URL, JWT_SECRET, OPENAI_API_KEY

# Deploy application
cd /opt/aideas-api
sudo ./deploy.sh
```

## Manual Setup (Alternative)

If you prefer manual Terraform management:

```bash
cd infra/terraform

# Initialize
terraform init

# Create variables
cat > terraform.tfvars << EOF
region        = "eu-west-2"
instance_type = "t4g.nano"
ssh_key_name  = "aideas-key"
environment   = "production"
EOF

# Plan and apply
terraform plan
terraform apply

# Get IP address
terraform output ec2_public_ip
```

## Automated Deployment

Once infrastructure is deployed:

1. **Add SSH private key to GitHub secrets**:
   ```bash
   # Copy your private key content
   cat ~/.ssh/aideas-key.pem

   # Add to: https://github.com/lucaromagnoli/aideas-api/settings/secrets/actions
   # Secret name: SSH_PRIVATE_KEY
   ```

2. **Push to main branch** → Automatic deployment via GitHub Actions!

## Environment Configuration

Required environment variables in `/opt/aideas-api/.env`:

```bash
# Database (PostgreSQL required)
DATABASE_URL=postgres://user:pass@your-rds-endpoint:5432/dbname?sslmode=require

# JWT signing secret (generate secure random string)
JWT_SECRET=your-jwt-secret-here

# OpenAI API key (required for generation)
OPENAI_API_KEY=sk-your-openai-api-key

# MCP server URL (optional, leave empty to disable)
MCP_SERVER_URL=https://mcp.musicalaideas.com
```

## Networking

The infrastructure creates:

- Security group with ports 22 (SSH) and 8080 (API) open
- Elastic IP for consistent addressing
- VPC network access for database connectivity

## Monitoring

CloudWatch agent is installed for log aggregation:

- Instance logs: `/aideas/api` log group
- Application logs: handled by Docker logging driver

## Cost Estimation

Monthly costs (~$16 total):

- **EC2 t4g.nano**: ~$3
- **Elastic IP**: ~$3.60 (when instance stopped)
- **ECR storage**: ~$0.10 (minimal, ~100MB images)
- **Data transfer**: ~$0 (minimal traffic)
- **RDS db.t4g.micro**: ~$12 (if using managed PostgreSQL)

## Cleanup

To remove all resources:

```bash
cd infra/terraform
terraform destroy
```

This will delete the EC2 instance, Elastic IP, security group, and IAM resources.
