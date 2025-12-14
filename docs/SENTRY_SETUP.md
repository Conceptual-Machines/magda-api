# Sentry Setup Guide

## Add Sentry DSN to GitHub Secrets

Your Sentry DSN needs to be added as a GitHub Secret so it's available during deployment.

### Steps:

1. **Go to your GitHub repository**:
   ```
   https://github.com/lucaromagnoli/aideas-api
   ```

2. **Navigate to Settings → Secrets and variables → Actions**

3. **Click "New repository secret"**

4. **Add the secret:**
   - **Name**: `SENTRY_DSN`
   - **Value**: `https://28f358c72c7f202f71c2b9c5591c73ea@o4510127655223296.ingest.de.sentry.io/4510127667085392`

5. **Click "Add secret"**

## Update Deployment Scripts

The `SENTRY_DSN` secret needs to be passed to your EC2 instance during deployment.

### For `deploy.yml` workflow:

Add to the SSH deployment step:
```yaml
- name: Deploy to EC2
  env:
    SENTRY_DSN: ${{ secrets.SENTRY_DSN }}
  run: |
    ssh -o StrictHostKeyChecking=no ubuntu@${{ secrets.EC2_HOST }} << 'EOF'
      export SENTRY_DSN="${{ secrets.SENTRY_DSN }}"
      cd /home/ubuntu/aideas-api
      # ... rest of deployment
    EOF
```

### For Terraform `user_data.sh`:

The secret will be injected via environment variable during Terraform apply.

## Verify Sentry is Working

### 1. Local Test
```bash
# Run the API locally
make dev

# You should see:
✅ Sentry initialized (environment: development)
```

### 2. Test Error Tracking

Send a test error:
```bash
curl http://localhost:8080/api/test-error
```

Check Sentry dashboard at: https://sentry.io/organizations/[your-org]/issues/

### 3. Production Deployment

After deployment, check EC2 logs:
```bash
ssh ubuntu@your-ec2-instance
docker logs aideas-api
# Look for: ✅ Sentry initialized (environment: production)
```

## Sentry Dashboard

Access your Sentry project:
- **URL**: https://sentry.io/organizations/[your-org]/projects/aideas-api/
- **Issues**: View all errors and exceptions
- **Performance**: Monitor API response times
- **Alerts**: Set up notifications for critical errors

## What Gets Tracked

With the current setup, Sentry automatically captures:

- ✅ **Panics/crashes** - Unhandled errors that crash the app
- ✅ **HTTP errors** - 4xx and 5xx responses
- ✅ **Stack traces** - Full context of where errors occur
- ✅ **Request context** - Headers, path, method, user ID
- ✅ **Performance** - Response times and slow endpoints
- ✅ **Breadcrumbs** - Trail of events leading to errors

## Privacy & Security

Sensitive data is automatically filtered:
- ❌ Authorization headers → `[REDACTED]`
- ❌ Cookie values → `[REDACTED]`
- ❌ API keys → `[REDACTED]`
- ✅ Request IDs, paths, methods → Visible
- ✅ User IDs (no PII) → Visible

## Free Tier Limits

- **5,000 errors/month** - Way more than needed for your usage
- **10,000 performance transactions/month** - Plenty for API monitoring
- **90-day retention** - Errors kept for 3 months

If you exceed limits, Sentry will just stop accepting new events until next month. No charges.

## Support

- **Sentry Docs**: https://docs.sentry.io/platforms/go/
- **Dashboard**: https://sentry.io
- **Issues**: Check GitHub issues or Sentry support
