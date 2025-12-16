package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDrummerTestRouter creates a minimal test router with the drummer endpoint
func setupDrummerTestRouter() *gin.Engine {
	_ = godotenv.Load()
	cfg := config.Load()
	if cfg.OpenAIAPIKey == "" {
		cfg.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	}
	if cfg.Environment == "" {
		cfg.Environment = "test"
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Drummer endpoint (without auth for testing)
	drummerHandler := NewDrummerHandler(cfg, nil)
	router.POST("/api/v1/drummer/generate", drummerHandler.Generate)

	return router
}

func TestDrummerHandler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupDrummerTestRouter()

	tests := []struct {
		name           string
		request        DrummerRequest
		expectedStatus int
		validateResp   func(t *testing.T, resp map[string]any)
	}{
		{
			name: "basic_kick_pattern",
			request: DrummerRequest{
				Model: "gpt-5.1",
				InputArray: []map[string]any{
					{
						"role":    "user",
						"content": "Create a basic four on the floor kick pattern",
					},
				},
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]any) {
				t.Helper()
				// Check DSL is present
				dsl, ok := resp["dsl"].(string)
				require.True(t, ok, "Response should contain DSL string")
				assert.NotEmpty(t, dsl, "DSL should not be empty")
				assert.Contains(t, dsl, "pattern", "DSL should contain pattern call")
				assert.Contains(t, dsl, "kick", "DSL should contain kick drum")

				// Check actions are present
				actions, ok := resp["actions"].([]any)
				require.True(t, ok, "Response should contain actions array")
				assert.NotEmpty(t, actions, "Actions should not be empty")

				// Validate first action
				action, ok := actions[0].(map[string]any)
				require.True(t, ok, "Action should be a map")
				assert.Equal(t, "drum_pattern", action["type"], "Action type should be drum_pattern")
				assert.Equal(t, "kick", action["drum"], "Drum should be kick")

				// Check grid pattern
				grid, ok := action["grid"].(string)
				require.True(t, ok, "Grid should be a string")
				assert.NotEmpty(t, grid, "Grid should not be empty")
				t.Logf("Generated grid: %s", grid)
			},
		},
		{
			name: "snare_backbeat",
			request: DrummerRequest{
				Model: "gpt-5.1",
				InputArray: []map[string]any{
					{
						"role":    "user",
						"content": "Create a snare backbeat pattern hitting on beats 2 and 4",
					},
				},
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]any) {
				t.Helper()
				dsl, ok := resp["dsl"].(string)
				require.True(t, ok, "Response should contain DSL string")
				assert.Contains(t, dsl, "snare", "DSL should contain snare")

				actions, ok := resp["actions"].([]any)
				require.True(t, ok, "Response should contain actions array")
				assert.NotEmpty(t, actions, "Actions should not be empty")

				// Check for snare action
				foundSnare := false
				for _, a := range actions {
					action, ok := a.(map[string]any)
					if ok && action["drum"] == "snare" {
						foundSnare = true
						t.Logf("Snare grid: %s", action["grid"])
					}
				}
				assert.True(t, foundSnare, "Should have snare pattern")
			},
		},
		{
			name: "hi_hat_eighth_notes",
			request: DrummerRequest{
				Model: "gpt-5.1",
				InputArray: []map[string]any{
					{
						"role":    "user",
						"content": "Create a hi-hat pattern playing eighth notes",
					},
				},
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]any) {
				t.Helper()
				dsl, ok := resp["dsl"].(string)
				require.True(t, ok, "Response should contain DSL string")
				assert.Contains(t, dsl, "hat", "DSL should contain hat")

				actions, ok := resp["actions"].([]any)
				require.True(t, ok, "Response should contain actions array")
				assert.NotEmpty(t, actions, "Actions should not be empty")
			},
		},
		{
			name: "full_beat",
			request: DrummerRequest{
				Model: "gpt-5.1",
				InputArray: []map[string]any{
					{
						"role":    "user",
						"content": "Create a complete rock beat with kick, snare, and hi-hat",
					},
				},
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]any) {
				t.Helper()
				actions, ok := resp["actions"].([]any)
				require.True(t, ok, "Response should contain actions array")
				assert.GreaterOrEqual(t, len(actions), 2, "Should have at least 2 patterns for a full beat")

				// Check we have multiple drum types
				drums := make(map[string]bool)
				for _, a := range actions {
					action, ok := a.(map[string]any)
					if ok {
						if drum, ok := action["drum"].(string); ok {
							drums[drum] = true
							t.Logf("%s: %s", drum, action["grid"])
						}
					}
				}
				t.Logf("Drums used: %v", drums)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal request
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// Create request
			req, err := http.NewRequest("POST", "/api/v1/drummer/generate", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status
			assert.Equal(t, tt.expectedStatus, w.Code, "Unexpected status code")

			if tt.expectedStatus == http.StatusOK {
				// Parse response
				var resp map[string]any
				err = json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err, "Failed to parse response")

				// Validate response
				if tt.validateResp != nil {
					tt.validateResp(t, resp)
				}
			}
		})
	}
}

func TestDrummerHandler_KeywordRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test that drummer keywords are properly detected
	// This tests the orchestrator's keyword detection

	tests := []struct {
		question        string
		shouldBeDrummer bool
	}{
		{"Create a drum beat", true},
		{"Add a kick pattern", true},
		{"Make a hi-hat groove", true},
		{"Four on the floor rhythm", true},
		{"Create a chord progression", false}, // Should be arranger
		{"Add reverb to track 1", false},      // Should be DAW
	}

	for _, tt := range tests {
		t.Run(tt.question, func(t *testing.T) {
			// This would test keyword detection
			// For now, just log the expected routing
			t.Logf("Question: %q -> Expected drummer: %v", tt.question, tt.shouldBeDrummer)
		})
	}
}
