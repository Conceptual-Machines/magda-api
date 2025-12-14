# 🚀 AIDEAS API - AWS Deployment Guide

## Overview

Deployment is **fully automated** via GitHub Actions:
1. Push to `main` branch triggers deploy workflow
2. Tests run, Docker image builds and pushes to ECR
3. GitHub Actions SSHs into EC2 and pulls/restarts containers
4. Zero-downtime deployment (< 30 seconds)

## Prerequisites

1. **AWS EC2 Instance** (recommended: t4g.nano ARM64 or t3.nano AMD64)
   - Tagged: `Name=magda-api-production`
   - IAM role with ECR read access
   - SSH key pair for GitHub Actions access
2. **AWS ECR Repository**: `magda-api`
3. **PostgreSQL Database** (AWS RDS recommended)
4. **GitHub Secrets**:
   - `AWS_ACCESS_KEY_ID`
   - `AWS_SECRET_ACCESS_KEY`
   - `SSH_PRIVATE_KEY` (for EC2 access)

## Initial Setup

### 1. Create ECR Repository

```bash
aws ecr create-repository \
  --repository-name magda-api \
  --region eu-west-2
```

### 2. Launch EC2 Instance

```bash
aws ec2 run-instances \
  --image-id ami-0c55b159cbfafe1f0 \
  --instance-type t4g.nano \
  --key-name your-key-name \
  --security-group-ids sg-xxxxx \
  --subnet-id subnet-xxxxx \
  --iam-instance-profile Name=magda-api-ec2-profile \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=magda-api-production}]'
```

**Important:** Tag the instance with `Name=magda-api-production` so the deploy workflow can find it.

### 3. Configure EC2 IAM Role

The instance needs ECR read permissions:

```bash
# Create IAM role
aws iam create-role \
  --role-name magda-api-ec2-role \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }'

# Attach ECR policy
aws iam attach-role-policy \
  --role-name magda-api-ec2-role \
  --policy-arn arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly

# Create instance profile
aws iam create-instance-profile \
  --instance-profile-name magda-api-ec2-profile

# Add role to profile
aws iam add-role-to-instance-profile \
  --instance-profile-name magda-api-ec2-profile \
  --role-name magda-api-ec2-role
```

### 4. Configure Security Group

```bash
# Allow SSH from GitHub Actions (you may need to whitelist GitHub's IP ranges)
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxx \
  --protocol tcp \
  --port 22 \
  --cidr 0.0.0.0/0

# Allow API traffic
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxx \
  --protocol tcp \
  --port 8080 \
  --cidr 0.0.0.0/0
```

### 5. Set up EC2 Environment

SSH into your EC2 instance and run:

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker ubuntu

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Install AWS CLI
curl "https://awscli.amazonaws.com/awscli-exe-linux-$(uname -m).zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# Create application directory
sudo mkdir -p /opt/magda-api
sudo chown ubuntu:ubuntu /opt/magda-api

# Create environment file
cat > /opt/magda-api/.env << 'EOF'
ENVIRONMENT=production
DATABASE_URL=postgres://user:pass@your-rds-endpoint:5432/dbname?sslmode=disable
JWT_SECRET=your-jwt-secret-here
OPENAI_API_KEY=sk-your-openai-key
MCP_SERVER_URL=https://mcp.musicalaideas.com
EOF

# Create docker-compose.yml
cat > /opt/magda-api/docker-compose.yml << 'EOF'
version: '3.8'

services:
  magda-api:
    image: 086060940749.dkr.ecr.eu-west-2.amazonaws.com/magda-api:latest
    container_name: magda-api
    restart: unless-stopped
    ports:
      - "8080:8080"
    env_file:
      - .env
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
EOF
```

### 6. Configure GitHub Secrets

Go to your GitHub repository → Settings → Secrets and variables → Actions.

Add these secrets:
- `AWS_ACCESS_KEY_ID`: Your AWS access key (with ECR push permissions)
- `AWS_SECRET_ACCESS_KEY`: Your AWS secret key
- `SSH_PRIVATE_KEY`: Private key for SSH access to EC2 (contents of your `.pem` file)

### 7. Deploy!

Push to `main` branch:

```bash
git push origin main
```

The GitHub Actions workflow will:
1. Run tests
2. Build and push Docker image to ECR
3. SSH into EC2
4. Pull latest image and restart containers

## Manual Deployment

If you need to deploy manually (not recommended):

```bash
# SSH into EC2
ssh -i your-key.pem ubuntu@YOUR_EC2_IP

# Run deploy script
cd /opt/magda-api/app
./deploy.sh
```

## Monitoring

### Check Deployment Status

View workflow runs: https://github.com/lucaromagnoli/magda-api/actions

### Check Container Health

```bash
# SSH into EC2
ssh -i your-key.pem ubuntu@YOUR_EC2_IP

# View running containers
sudo docker ps

# View logs
sudo docker compose -f /opt/magda-api/docker-compose.yml logs -f

# Check health endpoint
curl http://localhost:8080/health
```

### CloudWatch Logs (Optional)

You can set up CloudWatch for centralized logging:

```bash
# Install CloudWatch agent on EC2
wget https://s3.amazonaws.com/amazoncloudwatch-agent/ubuntu/amd64/latest/amazon-cloudwatch-agent.deb
sudo dpkg -i amazon-cloudwatch-agent.deb
```

## Rollback

To rollback to a previous version:

1. Find the commit SHA of the working version
2. SSH into EC2:
   ```bash
   cd /opt/magda-api

   # Pull specific version
   sudo docker pull 086060940749.dkr.ecr.eu-west-2.amazonaws.com/magda-api:COMMIT_SHA

   # Tag it as latest
   sudo docker tag 086060940749.dkr.ecr.eu-west-2.amazonaws.com/magda-api:COMMIT_SHA \
                    086060940749.dkr.ecr.eu-west-2.amazonaws.com/magda-api:latest

   # Restart
   sudo docker compose up -d
   ```

## Cost Estimate

- **EC2 t4g.nano**: ~$3/month
- **ECR storage**: ~$0.10/GB/month (minimal for small images)
- **RDS db.t4g.micro**: ~$12/month (if using managed PostgreSQL)
- **Data transfer**: Minimal (< $1/month for low traffic)

**Total**: ~$16/month for a production setup
