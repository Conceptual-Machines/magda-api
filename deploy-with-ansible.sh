#!/bin/bash

# Simple deployment script using Ansible
# This replaces the buggy Terraform + GitHub Actions approach

set -e

echo "🚀 Starting AIDEAS API deployment with Ansible..."

# Check if we're in the right directory
if [ ! -f "ansible/manage-existing.yml" ]; then
    echo "❌ Error: Run this script from the project root directory"
    exit 1
fi

# Install Ansible if not present
if ! command -v ansible-playbook &> /dev/null; then
    echo "📦 Installing Ansible..."
    pip install ansible boto3
fi

# Set up AWS credentials
if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo "❌ Error: AWS credentials not set"
    echo "Please set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY"
    exit 1
fi

# Run the Ansible playbook
echo "🎯 Running Ansible playbook..."
cd ansible
ansible-playbook -i inventory.yml manage-existing.yml -v

echo "✅ Deployment complete!"
