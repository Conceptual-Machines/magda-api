#!/bin/bash

set -euo pipefail

APP_DIR="/opt/magda-api"
DOMAIN="api.musicalaideas.com"
EMAIL="${SSL_EMAIL:-admin@musicalaideas.com}"

echo "🚀 Deploying AIDEAS API application..."

# Check if .env exists and is configured
if [ ! -f "$APP_DIR/.env" ]; then
    echo "❌ .env file not found at $APP_DIR/.env"
    echo "Please create it with DATABASE_URL, JWT_SECRET, OPENAI_API_KEY, MCP_SERVER_URL"
    exit 1
fi

# Load .env
set -a
source "$APP_DIR/.env"
set +a

# Ensure Docker Compose is installed
if ! command -v docker compose &> /dev/null; then
    echo "❌ Docker Compose not found. Please install Docker and Docker Compose."
    exit 1
fi

# Authenticate to Amazon ECR
echo "🔐 Authenticating to Amazon ECR..."
AWS_REGION=${AWS_REGION:-eu-west-2}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REGISTRY="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

aws ecr get-login-password --region $AWS_REGION | sudo docker login --username AWS --password-stdin $ECR_REGISTRY

# Pull latest images
echo "📦 Pulling latest images..."
cd "$APP_DIR"
sudo docker compose -f docker-compose.prod.yml pull

# Start services (nginx will handle Let's Encrypt challenge through Cloudflare proxy)
echo "🚀 Starting services..."
sudo docker compose -f docker-compose.prod.yml up -d --remove-orphans

# Install certbot if not present
if ! command -v certbot &> /dev/null; then
    echo "📦 Installing certbot..."
    sudo apt-get update
    sudo apt-get install -y certbot
fi

# Generate SSL certificates with Let's Encrypt
echo "🔐 Checking SSL certificates..."
sudo mkdir -p "$APP_DIR/ssl"

CERT_PATH="/etc/letsencrypt/live/magda-api/fullchain.pem"
GENERATE_CERT=false

# Check if certificate exists
if [ -f "$CERT_PATH" ]; then
    echo "📋 Certificate found, checking validity..."

    # Check if certificate expires in less than 30 days
    if sudo openssl x509 -checkend 2592000 -noout -in "$CERT_PATH" >/dev/null 2>&1; then
        echo "✅ Certificate is valid for at least 30 days, skipping generation"
        GENERATE_CERT=false
    else
        echo "⚠️  Certificate expires soon or is invalid, will regenerate"
        GENERATE_CERT=true
    fi
else
    echo "📋 No certificate found, will generate new one"
    GENERATE_CERT=true
fi

# Generate certificate if needed
if [ "$GENERATE_CERT" = true ]; then
    echo "📋 Requesting SSL certificate for domain: $DOMAIN"
    echo "🌐 Using HTTP-01 challenge (requires port 80 to be open)"

    # Stop nginx temporarily to free port 80
    echo "⏸️  Stopping nginx temporarily..."
    sudo docker compose -f "$APP_DIR/docker-compose.prod.yml" stop nginx 2>/dev/null || true

    sudo certbot certonly \
      --standalone \
      --preferred-challenges http \
      --email "$EMAIL" \
      --agree-tos \
      --non-interactive \
      -d "$DOMAIN" \
      --cert-name magda-api

    if [ $? -eq 0 ]; then
        echo "✅ SSL certificate generated successfully"
    else
        echo "❌ Failed to generate SSL certificate"
        exit 1
    fi
fi

# Copy certificates to nginx directory
echo "📋 Copying certificates to nginx directory..."
sudo cp "/etc/letsencrypt/live/magda-api/fullchain.pem" "$APP_DIR/ssl/certificate.pem"
sudo cp "/etc/letsencrypt/live/magda-api/privkey.pem" "$APP_DIR/ssl/private.key"
sudo chmod 644 "$APP_DIR/ssl/certificate.pem"
sudo chmod 600 "$APP_DIR/ssl/private.key"

echo "✅ SSL certificates ready"

# Restart all services with SSL certificates
echo "🔄 Restarting all services..."
sudo docker compose -f docker-compose.prod.yml up -d --remove-orphans

echo "✅ Deployment complete!"
echo "🌐 API: https://api.musicalaideas.com"
echo "🏥 Health: https://api.musicalaideas.com/health"
sudo docker ps
