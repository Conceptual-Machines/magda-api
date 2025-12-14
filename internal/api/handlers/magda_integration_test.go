package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load environment variables from .env file if it exists
	// This allows tests to use the same environment configuration as the main application
	// Note: If using .envrc with direnv, run tests with: direnv exec . go test ...
	_ = godotenv.Load() // Ignore error - .env file is optional
}

// setupTestRouter creates a minimal test router with just the endpoints we need
func setupTestRouter() *gin.Engine {
	// Ensure .env is loaded (in case init() didn't run or .env was created after)
	_ = godotenv.Load() // Ignore error - .env file is optional

	// Load config from environment variables
	// The OpenAI API key should be set in .env or .envrc
	cfg := config.Load()

	// Debug: Check if API key is loaded
	if cfg.OpenAIAPIKey == "" {
		// Try to get it directly from environment
		cfg.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
		if cfg.OpenAIAPIKey == "" {
			// Last resort: try loading .env again with explicit path
			_ = godotenv.Load(".env")
			cfg.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
		}
	}

	// Override environment for testing if not already set
	if cfg.Environment == "" {
		cfg.Environment = "test"
	}

	// Setup minimal router for testing
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", HealthCheck)

	// MAGDA endpoint (without auth for testing)
	// Note: In production this would require JWT auth
	magdaHandler := NewMagdaHandler(cfg, nil)
	router.POST("/api/v1/magda/chat", magdaHandler.Chat)

	return router
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	router := setupTestRouter()

	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
}

// TestMagdaChatEndpoint tests the MAGDA chat endpoint
func TestMagdaChatEndpoint(t *testing.T) {
	router := setupTestRouter()

	// Create a test request
	requestBody := MagdaChatRequest{
		Question: "Create a new track called 'Drums'",
		State: map[string]interface{}{
			"project": map[string]interface{}{
				"name":   "Test Project",
				"length": 120.5,
			},
			"tracks": []map[string]interface{}{},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The endpoint should return either:
	// - 200 OK with actions (if OpenAI API key is valid)
	// - 500 Internal Server Error (if OpenAI API key is invalid or API call fails)
	// - 401 Unauthorized (if JWT auth is required)

	// For now, we just check that the endpoint exists and responds
	// In a real integration test, you'd mock the OpenAI provider or use a test API key
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError || w.Code == http.StatusUnauthorized,
		"Expected status 200, 500, or 401, got %d: %s", w.Code, w.Body.String())

	// If we got a 200, verify the response structure
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have actions field
		actions, ok := response["actions"]
		assert.True(t, ok, "Response should have 'actions' field")

		// Actions should be an array
		if ok {
			actionsArray, isArray := actions.([]interface{})
			assert.True(t, isArray, "Actions should be an array")

			// If there are actions, verify structure
			if len(actionsArray) > 0 {
				action, isMap := actionsArray[0].(map[string]interface{})
				assert.True(t, isMap, "Action should be a map")

				if isMap {
					actionType, hasAction := action["action"]
					assert.True(t, hasAction, "Action should have 'action' field")
					assert.NotEmpty(t, actionType, "Action type should not be empty")
				}
			}
		}
	}
}

// TestMagdaChatEndpointInvalidRequest tests error handling for invalid requests
func TestMagdaChatEndpointInvalidRequest(t *testing.T) {
	router := setupTestRouter()

	// Test with missing question field
	requestBody := map[string]interface{}{
		"state": map[string]interface{}{},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 Bad Request for invalid request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should have error message
	_, hasError := response["error"]
	assert.True(t, hasError, "Error response should have 'error' field")
}

// TestMagdaChatEndpointEmptyQuestion tests with empty question
func TestMagdaChatEndpointEmptyQuestion(t *testing.T) {
	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "",
		State:    map[string]interface{}{},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 Bad Request for empty question
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestMagdaChatEndpointWithState tests with REAPER state
func TestMagdaChatEndpointWithState(t *testing.T) {
	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "Add a clip to track 0 at bar 1",
		State: map[string]interface{}{
			"project": map[string]interface{}{
				"name":   "My Project",
				"length": 240.0,
			},
			"tracks": []map[string]interface{}{
				{
					"index": 0,
					"name":  "Track 1",
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should handle the request (may fail if OpenAI API key is invalid)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError || w.Code == http.StatusUnauthorized,
		"Expected status 200, 500, or 401, got %d: %s", w.Code, w.Body.String())
}

// TestMagdaCreateTrackWithSerum tests the end-to-end flow: create a track with Serum instrument
// This test requires a valid OpenAI API key and will make actual API calls
func TestMagdaCreateTrackWithSerum(t *testing.T) {
	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "Create a new track with Serum",
		State: map[string]interface{}{
			"project": map[string]interface{}{
				"name":   "Test Project",
				"length": 120.0,
			},
			"tracks": []map[string]interface{}{},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// If API key is invalid, skip this test
	if w.Code == http.StatusInternalServerError {
		var errorResponse map[string]interface{}
		if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &errorResponse); unmarshalErr == nil {
			if errorMsg, ok := errorResponse["error"].(string); ok {
				if contains(errorMsg, "API key") || contains(errorMsg, "Unauthorized") {
					t.Skip("Skipping test: Invalid or missing OpenAI API key")
					return
				}
			}
		}
	}

	require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
	actions, ok := response["actions"].([]interface{})
	require.True(t, ok, "Response should have 'actions' array")
	require.Greater(t, len(actions), 0, "Should have at least one action")

	// Verify we got a create_track action with instrument
	hasCreateTrack := false
	hasSerumInstrument := false

	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok, "Action should be a map")

		actionType, ok := action["action"].(string)
		require.True(t, ok, "Action should have 'action' field as string")

		const actionCreateTrack = "create_track"
		if actionType == actionCreateTrack {
			hasCreateTrack = true
			// Check if instrument is in create_track action
			if instrument, ok := action["instrument"].(string); ok {
				if strings.Contains(instrument, "Serum") {
					hasSerumInstrument = true
				}
			}
		}
		// Also check for add_instrument as fallback (old style)
		if actionType == "add_instrument" {
			hasSerumInstrument = true
			// Verify it's Serum
			fxname, ok := action["fxname"].(string)
			if ok {
				assert.Contains(t, fxname, "Serum", "Instrument should be Serum")
			}
		}
	}

	assert.True(t, hasCreateTrack, "Should have create_track action")
	assert.True(t, hasSerumInstrument, "Should have Serum instrument (either in create_track or add_instrument action)")
}

// TestMagdaCreateTrackWithName tests creating a track with a specific name
func TestMagdaCreateTrackWithName(t *testing.T) {
	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "Create a track called 'Drums'",
		State: map[string]interface{}{
			"project": map[string]interface{}{
				"name":   "Test Project",
				"length": 120.0,
			},
			"tracks": []map[string]interface{}{},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Skip if API key is invalid
	if w.Code == http.StatusInternalServerError {
		var errorResponse map[string]interface{}
		if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &errorResponse); unmarshalErr == nil {
			if errorMsg, ok := errorResponse["error"].(string); ok {
				if contains(errorMsg, "API key") || contains(errorMsg, "Unauthorized") {
					t.Skip("Skipping test: Invalid or missing OpenAI API key")
					return
				}
			}
		}
	}

	require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	actions, ok := response["actions"].([]interface{})
	require.True(t, ok, "Response should have 'actions' array")
	require.Greater(t, len(actions), 0, "Should have at least one action")

	// Verify we got a create_track action with name
	hasCreateTrackWithName := false

	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok, "Action should be a map")

		actionType, ok := action["action"].(string)
		require.True(t, ok, "Action should have 'action' field")

		const actionCreateTrack = "create_track"
		if actionType == actionCreateTrack {
			name, ok := action["name"].(string)
			if ok && name == "Drums" {
				hasCreateTrackWithName = true
			}
		}
	}

	assert.True(t, hasCreateTrackWithName, "Should have create_track action with name 'Drums'")
}

// TestMagdaCreateClipAtBar tests creating a clip at a specific bar
func TestMagdaCreateClipAtBar(t *testing.T) {
	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "Add a 4-bar clip to track 0 starting at bar 17",
		State: map[string]interface{}{
			"project": map[string]interface{}{
				"name":   "Test Project",
				"length": 240.0,
			},
			"tracks": []map[string]interface{}{
				{
					"index": 0,
					"name":  "Track 1",
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Skip if API key is invalid
	if w.Code == http.StatusInternalServerError {
		var errorResponse map[string]interface{}
		if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &errorResponse); unmarshalErr == nil {
			if errorMsg, ok := errorResponse["error"].(string); ok {
				if contains(errorMsg, "API key") || contains(errorMsg, "Unauthorized") {
					t.Skip("Skipping test: Invalid or missing OpenAI API key")
					return
				}
			}
		}
	}

	require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	actions, ok := response["actions"].([]interface{})
	require.True(t, ok, "Response should have 'actions' array")
	require.Greater(t, len(actions), 0, "Should have at least one action")

	// Verify we got a create_clip_at_bar action
	hasCreateClipAtBar := false

	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok, "Action should be a map")

		actionType, ok := action["action"].(string)
		require.True(t, ok, "Action should have 'action' field")

		if actionType == "create_clip_at_bar" {
			hasCreateClipAtBar = true
			// Verify bar and length_bars
			bar, ok := action["bar"].(float64) // JSON numbers are float64
			if ok {
				assert.Equal(t, float64(17), bar, "Bar should be 17")
			}
			lengthBars, ok := action["length_bars"].(float64)
			if ok {
				assert.Equal(t, float64(4), lengthBars, "Length should be 4 bars")
			}
			track, ok := action["track"].(float64)
			if ok {
				assert.Equal(t, float64(0), track, "Track should be 0")
			}
		}
	}

	assert.True(t, hasCreateClipAtBar, "Should have create_clip_at_bar action")
}

// TestMagdaRenameTracksWithFilter tests renaming tracks using functional methods
// This tests: "rename all tracks named Foo to Bar"
func TestMagdaRenameTracksWithFilter(t *testing.T) {
	router := setupTestRouter()

	// Setup REAPER state with multiple tracks, some named "Foo"
	requestBody := MagdaChatRequest{
		Question: "rename all tracks named Foo to Bar",
		State: map[string]interface{}{
			"project": map[string]interface{}{
				"name":   "Test Project",
				"length": 120.0,
			},
			"tracks": []map[string]interface{}{
				{
					"index":    0,
					"name":     "Drums",
					"selected": false,
				},
				{
					"index":    1,
					"name":     "Foo",
					"selected": false,
				},
				{
					"index":    2,
					"name":     "Bass",
					"selected": false,
				},
				{
					"index":    3,
					"name":     "Foo",
					"selected": false,
				},
				{
					"index":    4,
					"name":     "Guitar",
					"selected": false,
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// If API key is invalid, skip this test
	if w.Code == http.StatusInternalServerError {
		var errorResponse map[string]interface{}
		if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &errorResponse); unmarshalErr == nil {
			if errorMsg, ok := errorResponse["error"].(string); ok {
				if contains(errorMsg, "API key") || contains(errorMsg, "Unauthorized") {
					t.Skip("Skipping test: Invalid or missing OpenAI API key")
					return
				}
			}
		}
	}

	require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
	actions, ok := response["actions"].([]interface{})
	require.True(t, ok, "Response should have 'actions' array")
	require.Greater(t, len(actions), 0, "Should have at least one action")

	// Count how many set_track_name actions we got for tracks named "Foo" (indices 1 and 3)
	renameActions := make(map[int]bool) // Track which track indices were renamed

	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok, "Action should be a map")

		actionType, ok := action["action"].(string)
		require.True(t, ok, "Action should have 'action' field")

		if actionType == "set_track" {
			if _, ok := action["name"].(string); ok {
				// Verify this is renaming a "Foo" track to "Bar"
				name, ok := action["name"].(string)
				require.True(t, ok, "set_track should have 'name' field")
				assert.Equal(t, "Bar", name, "All renamed tracks should be named 'Bar'")

				// Get track index
				track, ok := action["track"].(float64) // JSON numbers are float64
				if ok {
					trackIndex := int(track)
					// Tracks at indices 1 and 3 should be renamed (they're named "Foo")
					if trackIndex == 1 || trackIndex == 3 {
						renameActions[trackIndex] = true
					} else {
						t.Logf("Warning: set_track action for unexpected track index: %d", trackIndex)
					}
				}
			}
		}
	}

	// Verify we renamed both "Foo" tracks (indices 1 and 3)
	assert.True(t, renameActions[1], "Track 1 (named 'Foo') should be renamed to 'Bar'")
	assert.True(t, renameActions[3], "Track 3 (named 'Foo') should be renamed to 'Bar'")
	assert.Equal(t, 2, len(renameActions), "Should have exactly 2 rename actions for the 2 tracks named 'Foo'")

	t.Logf("✅ Successfully renamed %d tracks named 'Foo' to 'Bar'", len(renameActions))
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
