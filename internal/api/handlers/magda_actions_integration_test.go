package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load .env file for tests - try multiple paths
	_ = godotenv.Load()                // Current directory
	_ = godotenv.Load(".env")          // Current directory explicit
	_ = godotenv.Load("../../.env")    // From handlers/ directory
	_ = godotenv.Load("../../../.env") // From internal/api/handlers/ directory
}

// TestMagdaDeleteTrack tests the delete_track action
func TestMagdaDeleteTrack(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name       string
		question   string
		trackIndex int
		trackName  string
	}{
		{
			name:       "delete by name",
			question:   "delete Nebula Drift",
			trackIndex: 0,
			trackName:  "Nebula Drift",
		},
		{
			name:       "delete track 0",
			question:   "delete track 0",
			trackIndex: 0,
			trackName:  "Nebula Drift",
		},
		{
			name:       "remove track by name",
			question:   "remove Nebula Drift",
			trackIndex: 0,
			trackName:  "Nebula Drift",
		},
		{
			name:       "delete track by index",
			question:   "delete the first track",
			trackIndex: 0,
			trackName:  "Nebula Drift",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State: map[string]interface{}{
					"project": map[string]interface{}{
						"name":   "Test Project",
						"length": 0.0,
					},
					"play_state": map[string]interface{}{
						"playing":   false,
						"paused":    false,
						"recording": false,
						"position":  0.0,
						"cursor":    0.0,
					},
					"time_selection": map[string]interface{}{
						"start": 0.0,
						"end":   0.0,
					},
					"tracks": []map[string]interface{}{
						{
							"index":     tc.trackIndex,
							"name":      tc.trackName,
							"folder":    false,
							"selected":  false,
							"has_fx":    false,
							"muted":     false,
							"soloed":    false,
							"rec_armed": true,
							"volume_db": 0.0,
							"pan":       0.0,
							"clips":     []interface{}{},
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

			if w.Code == http.StatusInternalServerError {
				var errorResponse map[string]interface{}
				if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &errorResponse); unmarshalErr == nil {
					if errorMsg, ok := errorResponse["error"].(string); ok {
						if contains(errorMsg, "API key") || contains(errorMsg, "Unauthorized") {
							t.Skip("Skipping test: Invalid or missing OpenAI API key")
							return
						}
						// LLM MUST use CFG tool - this is a failure
						if contains(errorMsg, "CFG grammar was configured but LLM did not use CFG tool") {
							t.Errorf("❌ TEST FAILED: LLM did not use CFG tool. This is REQUIRED. Error: %s", errorMsg)
							t.FailNow()
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
			require.True(t, ok, "Response should contain actions array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			// Find delete_track action
			foundDeleteTrack := false
			for _, actionInterface := range actions {
				action, ok := actionInterface.(map[string]interface{})
				require.True(t, ok, "Each action should be a map")

				actionType, ok := action["action"].(string)
				require.True(t, ok, "Action should have 'action' field")

				if actionType == "delete_track" {
					foundDeleteTrack = true
					trackIndex, ok := action["track"].(float64) // JSON numbers are float64
					assert.True(t, ok, "delete_track should have track index")
					assert.Equal(t, float64(tc.trackIndex), trackIndex, "Should delete correct track index")

					// Verify it doesn't have mute field (which would indicate JSON fallback)
					_, hasMute := action["mute"]
					assert.False(t, hasMute, "delete_track should NOT have mute field")
				}

				// CRITICAL: Should NOT have set_track_mute action
				if actionType == "set_track_mute" {
					t.Errorf("❌ FAILED: Generated set_track_mute instead of delete_track for '%s'", tc.question)
					t.Errorf("This indicates JSON fallback logic is still active!")
				}
			}

			assert.True(t, foundDeleteTrack, "Should generate delete_track action for '%s'", tc.question)
		})
	}
}

// TestMagdaCreateTrack tests the create_track action
func TestMagdaCreateTrack(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		expected map[string]interface{}
	}{
		{
			name:     "create track with name",
			question: "create a track called Drums",
			expected: map[string]interface{}{
				"action": "create_track",
				"name":   "Drums",
			},
		},
		{
			name:     "create track with instrument",
			question: "create a track with Serum",
			expected: map[string]interface{}{
				"action":     "create_track",
				"instrument": "Serum",
			},
		},
		{
			name:     "create track with name and instrument",
			question: "create a track called Bass with Serum",
			expected: map[string]interface{}{
				"action":     "create_track",
				"name":       "Bass",
				"instrument": "Serum",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State: map[string]interface{}{
					"project": map[string]interface{}{
						"name":   "Test Project",
						"length": 0.0,
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

			if w.Code == http.StatusInternalServerError {
				var errorResponse map[string]interface{}
				if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &errorResponse); unmarshalErr == nil {
					if errorMsg, ok := errorResponse["error"].(string); ok {
						if contains(errorMsg, "API key") || contains(errorMsg, "Unauthorized") {
							t.Skip("Skipping test: Invalid or missing OpenAI API key")
							return
						}
						// LLM MUST use CFG tool - this is a failure
						if contains(errorMsg, "CFG grammar was configured but LLM did not use CFG tool") {
							t.Errorf("❌ TEST FAILED: LLM did not use CFG tool. This is REQUIRED. Error: %s", errorMsg)
							t.FailNow()
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
			require.True(t, ok, "Response should contain actions array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			// Find create_track action
			foundCreateTrack := false
			for _, actionInterface := range actions {
				action, ok := actionInterface.(map[string]interface{})
				require.True(t, ok)

				actionType, ok := action["action"].(string)
				require.True(t, ok)

				if actionType == "create_track" {
					foundCreateTrack = true

					// Verify expected fields
					for key, expectedValue := range tc.expected {
						if key == "action" {
							continue
						}
						actualValue, ok := action[key]
						assert.True(t, ok, "create_track should have '%s' field", key)
						assert.Equal(t, expectedValue, actualValue, "create_track '%s' should match expected value", key)
					}
				}
			}

			assert.True(t, foundCreateTrack, "Should generate create_track action for '%s'", tc.question)
		})
	}
}

// TestMagdaCreateClip tests the create_clip and create_clip_at_bar actions
func TestMagdaCreateClip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name           string
		question       string
		expectedAction string
		expectedFields map[string]interface{}
	}{
		{
			name:           "create clip at bar",
			question:       "add a clip to track 0 at bar 4",
			expectedAction: "create_clip_at_bar",
			expectedFields: map[string]interface{}{
				"track": float64(0),
				"bar":   float64(4),
			},
		},
		{
			name:           "create clip with length",
			question:       "add a 4-bar clip to track 0 starting at bar 8",
			expectedAction: "create_clip_at_bar",
			expectedFields: map[string]interface{}{
				"track":       float64(0),
				"bar":         float64(8),
				"length_bars": float64(4),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State: map[string]interface{}{
					"project": map[string]interface{}{
						"name":   "Test Project",
						"length": 120.0,
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

			if w.Code == http.StatusInternalServerError {
				var errorResponse map[string]interface{}
				if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &errorResponse); unmarshalErr == nil {
					if errorMsg, ok := errorResponse["error"].(string); ok {
						if contains(errorMsg, "API key") || contains(errorMsg, "Unauthorized") {
							t.Skip("Skipping test: Invalid or missing OpenAI API key")
							return
						}
						// LLM MUST use CFG tool - this is a failure
						if contains(errorMsg, "CFG grammar was configured but LLM did not use CFG tool") {
							t.Errorf("❌ TEST FAILED: LLM did not use CFG tool. This is REQUIRED. Error: %s", errorMsg)
							t.FailNow()
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
			require.True(t, ok, "Response should contain actions array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			// Find the expected action
			foundAction := false
			for _, actionInterface := range actions {
				action, ok := actionInterface.(map[string]interface{})
				require.True(t, ok)

				actionType, ok := action["action"].(string)
				require.True(t, ok)

				if actionType == tc.expectedAction {
					foundAction = true

					// Verify expected fields
					for key, expectedValue := range tc.expectedFields {
						actualValue, ok := action[key]
						assert.True(t, ok, "%s should have '%s' field", tc.expectedAction, key)
						assert.Equal(t, expectedValue, actualValue, "%s '%s' should match expected value", tc.expectedAction, key)
					}
				}
			}

			assert.True(t, foundAction, "Should generate %s action for '%s'", tc.expectedAction, tc.question)
		})
	}
}

// TestMagdaSetTrackVolume tests the set_track_volume action
func TestMagdaSetTrackVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "set track 0 volume to -3 dB",
		State: map[string]interface{}{
			"tracks": []map[string]interface{}{
				{
					"index":     0,
					"name":      "Track 1",
					"volume_db": 0.0,
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
	require.True(t, ok, "Response should contain actions array")
	require.Greater(t, len(actions), 0, "Should have at least one action")

	foundSetVolume := false
	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok)

		actionType, ok := action["action"].(string)
		require.True(t, ok)

		if actionType == "set_track" {
			if _, ok := action["volume_db"]; ok {
				foundSetVolume = true
				track, ok := action["track"].(float64)
				assert.True(t, ok, "set_track should have track index")
				assert.Equal(t, float64(0), track, "Should set volume for track 0")

				volume, ok := action["volume_db"].(float64)
				assert.True(t, ok, "set_track should have volume_db")
				assert.Equal(t, float64(-3), volume, "Should set volume to -3 dB")
			}
		}
	}

	assert.True(t, foundSetVolume, "Should generate set_track_volume action")
}

// TestMagdaSetTrackMute tests the set_track_mute action
func TestMagdaSetTrackMute(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "mute track 0",
		State: map[string]interface{}{
			"tracks": []map[string]interface{}{
				{
					"index": 0,
					"name":  "Track 1",
					"muted": false,
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
	require.True(t, ok, "Response should contain actions array")
	require.Greater(t, len(actions), 0, "Should have at least one action")

	foundSetMute := false
	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok)

		actionType, ok := action["action"].(string)
		require.True(t, ok)

		if actionType == "set_track" {
			if mute, ok := action["mute"].(bool); ok && mute {
				foundSetMute = true
				track, ok := action["track"].(float64)
				assert.True(t, ok, "set_track should have track index")
				assert.Equal(t, float64(0), track, "Should mute track 0")
				assert.True(t, mute, "Should set mute to true")
			}
		}
	}

	assert.True(t, foundSetMute, "Should generate set_track action with mute=true")
}

// TestMagdaSetTrackName tests the set_track_name action
func TestMagdaSetTrackName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "rename track 0 to Bass",
		State: map[string]interface{}{
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
	require.True(t, ok, "Response should contain actions array")
	require.Greater(t, len(actions), 0, "Should have at least one action")

	foundSetName := false
	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok)

		actionType, ok := action["action"].(string)
		require.True(t, ok)

		if actionType == "set_track" {
			if _, ok := action["name"].(string); ok {
				foundSetName = true
				track, ok := action["track"].(float64)
				assert.True(t, ok, "set_track should have track index")
				assert.Equal(t, float64(0), track, "Should rename track 0")

				name, ok := action["name"].(string)
				assert.True(t, ok, "set_track should have name field")
				assert.Equal(t, "Bass", name, "Should set name to 'Bass'")
			}
		}
	}

	assert.True(t, foundSetName, "Should generate set_track_name action")
}
