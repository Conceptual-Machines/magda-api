package config

import "os"

type Config struct {
	Environment        string
	DatabaseURL        string
	JWTSecret          string
	OpenAIAPIKey       string
	GeminiAPIKey       string // Google Gemini API key
	MCPServerURL       string
	SentryDSN          string
	BaseURL            string // API base URL for OAuth callbacks
	FrontendURL        string // Frontend URL for redirects
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	Port               string
	AWSRegion          string // AWS region for SES
	EmailFrom          string // Sender email address
	LangfusePublicKey  string // Langfuse public key for client-side
	LangfuseSecretKey  string // Langfuse secret key for server-side
	LangfuseHost       string // Langfuse host URL (cloud or self-hosted)
	LangfuseEnabled    bool   // Feature flag for Langfuse
	AuthMode           string // Auth mode: "jwt" (default), "gateway" (trust X-User-* headers)
}

func Load() *Config {
	return &Config{
		Environment:        getEnv("ENVIRONMENT", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
		GeminiAPIKey:       getEnv("GEMINI_API_KEY", ""),
		MCPServerURL:       getEnv("MCP_SERVER_URL", ""),
		SentryDSN:          getEnv("SENTRY_DSN", ""),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		Port:               getEnv("PORT", "8080"),
		AWSRegion:          getEnv("AWS_REGION", "eu-west-2"),
		EmailFrom:          getEnv("EMAIL_FROM", "Musical AIDEAS <noreply@musicalaideas.com>"),
		LangfusePublicKey:  getEnv("LANGFUSE_PUBLIC_KEY", ""),
		LangfuseSecretKey:  getEnv("LANGFUSE_SECRET_KEY", ""),
		LangfuseHost:       getEnv("LANGFUSE_HOST", "https://cloud.langfuse.com"),
		LangfuseEnabled:    getEnv("LANGFUSE_ENABLED", "false") == "true",
		AuthMode:           getEnv("AUTH_MODE", "jwt"), // "jwt" or "gateway"
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	// Only use default if env var is not set at all
	if value != "" {
		return value
	}
	return defaultValue
}

// IsGatewayMode returns true if running behind the Express gateway
func (c *Config) IsGatewayMode() bool {
	return c.AuthMode == "gateway"
}
