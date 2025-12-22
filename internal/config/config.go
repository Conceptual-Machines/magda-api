package config

import "os"

// Config holds the application configuration
// Note: This is a stateless configuration - no database or auth secrets needed
// Auth, billing, and user management are handled by magda-cloud gateway
type Config struct {
	// Environment
	Environment string
	Port        string

	// LLM API Keys
	OpenAIAPIKey string // OpenAI API key for GPT models
	GeminiAPIKey string // Google Gemini API key

	// MCP Server (optional)
	MCPServerURL string

	// Observability
	SentryDSN         string // Sentry DSN for error tracking
	LangfusePublicKey string // Langfuse public key
	LangfuseSecretKey string // Langfuse secret key
	LangfuseHost      string // Langfuse host URL (cloud or self-hosted)
	LangfuseEnabled   bool   // Feature flag for Langfuse

	// Auth mode
	// - "none": No auth (self-hosted, local dev)
	// - "gateway": Trust X-User-* headers from magda-cloud
	AuthMode string
}

func Load() *Config {
	return &Config{
		Environment:       getEnv("ENVIRONMENT", "development"),
		Port:              getEnv("PORT", "8080"),
		OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
		GeminiAPIKey:      getEnv("GEMINI_API_KEY", ""),
		MCPServerURL:      getEnv("MCP_SERVER_URL", ""),
		SentryDSN:         getEnv("SENTRY_DSN", ""),
		LangfusePublicKey: getEnv("LANGFUSE_PUBLIC_KEY", ""),
		LangfuseSecretKey: getEnv("LANGFUSE_SECRET_KEY", ""),
		LangfuseHost:      getEnv("LANGFUSE_HOST", "https://cloud.langfuse.com"),
		LangfuseEnabled:   getEnv("LANGFUSE_ENABLED", "false") == "true",
		AuthMode:          getEnv("AUTH_MODE", "none"), // Default to no auth for self-hosted
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

// IsGatewayMode returns true if running behind the Express gateway
func (c *Config) IsGatewayMode() bool {
	return c.AuthMode == "gateway"
}
