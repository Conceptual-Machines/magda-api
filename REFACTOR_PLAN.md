# Plan: Make magda-api Stateless for Open Source

## Goal
Make `magda-api` a stateless, database-free service that's easy to self-host. All user management, auth, billing moves to `magda-cloud` (private).

---

## ✅ COMPLETED

This refactoring has been completed. Here's a summary:

### What Was Removed:
- `internal/models/user.go`, `invitation.go`, `roles.go` - User/auth models
- `internal/middleware/auth.go`, `admin.go` - JWT auth middleware
- `internal/database/` - Database connection/migrations
- `internal/services/email.go`, `credits.go` - Email and credits services
- `internal/api/handlers/auth.go`, `user.go`, `admin.go`, `invitation.go`, `oauth.go`, `bootstrap.go`, `dashboard.go` - Auth handlers
- `internal/api/middleware/auth.go` - API auth middleware
- `internal/web/` - All web UI handlers and templates

### What Was Updated:
- `internal/config/config.go` - Removed DB/OAuth config, kept LLM keys and auth mode
- `main.go` - Removed database initialization
- `internal/api/router.go` - Simplified with auth modes (none/gateway)
- All handlers - Removed `db *gorm.DB` parameter
- `Dockerfile` - Removed templ generation step

### Auth Modes:
- `AUTH_MODE=none` → No auth (self-hosted, local dev) - **Default**
- `AUTH_MODE=gateway` → Trust `X-User-*` headers from magda-cloud

---

## File Structure After Refactor

```
magda-api/
├── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── api/
│   │   ├── router.go
│   │   ├── handlers/
│   │   │   ├── jsfx.go        # JSFX agent
│   │   │   ├── magda.go       # DAW agent (chat, plugins, mix)
│   │   │   ├── drummer.go     # Drummer agent
│   │   │   ├── mix.go         # Mix analysis
│   │   │   ├── generation.go  # Arranger
│   │   │   ├── health.go      # Health check
│   │   │   ├── metrics.go     # Metrics
│   │   │   └── mcp.go         # MCP status
│   │   └── middleware/
│   │       ├── cors.go
│   │       ├── sentry.go
│   │       ├── gateway.go     # Trust X-User-* headers
│   │       └── noauth.go      # Pass-through for self-hosted
│   ├── llm/                   # LLM providers (OpenAI, Gemini)
│   ├── models/                # Request/response models
│   ├── observability/         # Langfuse integration
│   ├── prompt/                # Prompt builders
│   └── services/              # DSL parser, LLM params
├── pkg/embedded/              # Embedded resources
├── Dockerfile
├── docker-compose.yml
└── README.md
```

---

## Summary Table

| Component | magda-api (public) | magda-cloud (private) |
|-----------|-------------------|----------------------|
| Users/Auth | ❌ None | ✅ JWT, OAuth, API keys |
| Database | ❌ None | ✅ PostgreSQL (Prisma) |
| Billing | ❌ None | ✅ Stripe, credits |
| LLM Agents | ✅ JSFX, DAW, Drummer | ❌ Proxied |
| Rate Limit | ❌ None | ✅ Per-user |
| Deployment | ❌ Removed | ✅ Terraform, Ansible |

---

## Notes

- **Security**: Before making public, rotate any AWS keys that were in git history
- **magda-agents-go**: Plan to merge into `magda-api/pkg/agents/` later with DAW namespacing
- **Telemetry**: Sentry/Langfuse are optional - just set environment variables
