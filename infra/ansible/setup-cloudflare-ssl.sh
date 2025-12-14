#!/bin/bash
set -eo pipefail

echo "🔐 Setting up Cloudflare Origin SSL certificates for AIDEAS API..."
echo ""

# Run Ansible playbook
ansible-playbook -i inventory.yml tasks/setup-cloudflare-ssl.yml -v

echo ""
echo "✅ SSL setup complete!"
