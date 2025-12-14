#!/bin/bash
set -eo pipefail

echo "🔐 Setting up SSL certificates for AIDEAS API..."
echo ""

# Run Ansible playbook
ansible-playbook -i inventory.yml tasks/setup-ssl.yml -v

echo ""
echo "✅ SSL setup complete!"
echo "Certificates will auto-renew daily at 3 AM"
