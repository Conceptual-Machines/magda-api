#!/bin/bash
set -eo pipefail  # Remove 'u' flag to allow unset variables in .env

# Load environment variables from .env file
if [ -f "../../.env" ]; then
    set -a
    source ../../.env
    set +a
else
    echo "Error: .env file not found in project root"
    exit 1
fi

set -u  # Re-enable unset variable check after sourcing

# Verify required variables are set
if [ -z "${DATABASE_URL:-}" ]; then
    echo "Error: DATABASE_URL not set in .env file"
    exit 1
fi

# Get the latest image tag (or use 'latest')
IMAGE_TAG="${1:-086060940749.dkr.ecr.eu-west-2.amazonaws.com/aideas-api:latest}"

echo "🚀 Deploying AIDEAS API..."
echo "📦 Image: $IMAGE_TAG"
echo ""

# Run Ansible playbook
ansible-playbook -i inventory.yml tasks/deploy-app.yml -v \
    -e "image_tag=$IMAGE_TAG"

echo ""
echo "✅ Deployment complete!"
