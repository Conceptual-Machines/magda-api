# Release Management

## Overview

AIDEAS API uses semantic versioning with Git SHA tags for releases, integrated with Sentry for error tracking and Docker for deployment.

## Release Flow

```
Git Commit → CI Build → Docker Tag → Sentry Release → Deploy → Track Errors
```

### 1. **Build** (GitHub Actions)
- Extracts first 8 characters of Git SHA (e.g., `a1b2c3d4`)
- Builds Docker image with that tag
- Embeds version in binary via `-ldflags`

### 2. **Docker Tagging**
Each build creates two Docker tags:
- `:latest` - Always points to most recent build
- `:a1b2c3d4` - Specific version from Git SHA

**Image naming:**
```
<aws-account-id>.dkr.ecr.eu-west-2.amazonaws.com/aideas-api:a1b2c3d4
```

### 3. **Sentry Release**
Creates a Sentry release with:
- **Version**: `aideas-api@a1b2c3d4`
- **Environment**: `production`
- **Commits**: Auto-linked to GitHub commits

### 4. **Deployment**
- Pulls specific Docker image by tag (not `:latest`)
- Sets `RELEASE_VERSION` environment variable
- Application reports version to Sentry on startup

## Viewing Current Version

### In Logs
```bash
# SSH into EC2
ssh ubuntu@<ec2-ip>

# Check running containers
docker ps | grep aideas-api

# View application logs
docker logs aideas-api-prod | head -20
# Look for: ✅ Sentry initialized (environment: production, release: a1b2c3d4)
```

### In Sentry
1. Go to: https://sentry.io/organizations/[your-org]/projects/aideas-api/
2. Click "Releases" in sidebar
3. See all deployed versions with:
   - Deployment time
   - Commits included
   - Errors by version
   - Adoption % (how many users on each version)

### Via API
```bash
curl https://api.musicalaideas.com/health
# Future: Add version to health check response
```

## Release History

Sentry automatically tracks:
- **Deploy frequency** - How often you deploy
- **New errors** - Errors introduced in this release
- **Regressions** - Errors that came back
- **Suspect commits** - Which commit likely caused an error

## Rollback

To rollback to a previous version:

```bash
# SSH to EC2
ssh ubuntu@<ec2-ip>

# Find previous version
docker images | grep aideas-api

# Set the image tag
cd /opt/aideas-api
export API_IMAGE="<account-id>.dkr.ecr.eu-west-2.amazonaws.com/aideas-api:<old-version>"

# Deploy old version
sudo -E docker compose -f docker-compose.prod.yml up -d
```

Or trigger a new deployment from an old commit:
```bash
# Locally, create a new commit that reverts to old code
git revert <bad-commit-sha>
git push origin main

# CI/CD will automatically deploy
```

## GitHub Secrets Required

For Sentry releases integration:

1. **SENTRY_AUTH_TOKEN**
   - Go to: https://sentry.io/settings/account/api/auth-tokens/
   - Create new token with scopes: `project:releases`, `project:write`
   - Add to GitHub Secrets

2. **SENTRY_ORG**
   - Your Sentry organization slug (from URL)
   - Example: If URL is `https://sentry.io/organizations/my-company/`
   - Then `SENTRY_ORG=my-company`

## Benefits

### 🔍 **Error Tracking by Version**
See exactly which version introduced a bug

### 📊 **Deploy Tracking**
Monitor deployment frequency and success rate

### 🎯 **Suspect Commits**
Sentry suggests which commit caused an error

### 📈 **Adoption**
See what % of users are on each version

### 🔙 **Easy Rollback**
Quickly revert to a known-good version

## Example Release

```yaml
Release: aideas-api@a1b2c3d4
Environment: production
Deployed: 2025-10-03 22:30:15 UTC
Commits:
  - fix: resolve CloudWatch metrics integration
  - feat: add Sentry releases support
Errors: 0 new errors
Adoption: 100% of users
```

## Troubleshooting

### Release not showing in Sentry
- Check GitHub Actions logs for "Create Sentry release" step
- Verify `SENTRY_AUTH_TOKEN` and `SENTRY_ORG` secrets are set
- Ensure Sentry project name matches: `aideas-api`

### Version showing as "dev"
- Check Docker build logs for `RELEASE_VERSION` build arg
- Verify `-ldflags` in Dockerfile
- Ensure version is passed from CI to Docker build

### Wrong version deployed
- Check deployment logs for `Using image:` line
- Verify `API_IMAGE` environment variable
- Check running container: `docker inspect aideas-api-prod | grep Image`

## Future Enhancements

- [ ] Add `/version` endpoint to API
- [ ] Include version in health check response
- [ ] Auto-create GitHub releases
- [ ] Slack notifications for deployments
- [ ] Automatic rollback on error spike
