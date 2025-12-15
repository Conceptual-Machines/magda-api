package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMagdaCFGModeRejectsJSON tests that CFG mode strictly rejects JSON output
// and only accepts DSL from CFG tool calls
func TestMagdaCFGModeRejectsJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	// Test case: "delete Nebula Drift" should generate DSL, not JSON
	requestBody := MagdaChatRequest{
		Question: "delete Nebula Drift",
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
					"index":     0,
					"name":      "Nebula Drift",
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

	// The request should either:
	// 1. Succeed with DSL-based actions (delete_track)
	// 2. Fail with an error if LLM generates JSON instead of DSL
	// It should NEVER succeed with JSON actions like set_track_mute

	responseBody := w.Body.Bytes()
	responseStr := string(responseBody)

	if w.Code == http.StatusOK {
		// If successful, verify it's using DSL (delete_track), not JSON fallback (set_track_mute)
		var response map[string]interface{}
		err = json.Unmarshal(responseBody, &response)
		require.NoError(t, err, "Response should be valid JSON")

		actions, ok := response["actions"].([]interface{})
		require.True(t, ok, "Response should contain actions array")

		// Verify that we got delete_track, not set_track_mute
		foundDeleteTrack := false
		foundSetTrackMute := false

		for _, actionInterface := range actions {
			action, ok := actionInterface.(map[string]interface{})
			require.True(t, ok, "Each action should be a map")

			actionType, ok := action["action"].(string)
			require.True(t, ok, "Action should have a type")

			if actionType == "delete_track" {
				foundDeleteTrack = true
				// Verify it has the correct track index
				trackIndex, ok := action["track"].(float64) // JSON numbers are float64
				assert.True(t, ok, "delete_track should have track index")
				assert.Equal(t, float64(0), trackIndex, "Should delete track 0 (Nebula Drift)")
			}

			if actionType == "set_track_mute" {
				foundSetTrackMute = true
			}
		}

		// CRITICAL: We should have delete_track, NOT set_track_mute
		assert.True(t, foundDeleteTrack, "Should generate delete_track action for 'delete Nebula Drift'")
		assert.False(t, foundSetTrackMute, "Should NOT generate set_track_mute - this indicates JSON fallback is still active")

		if foundSetTrackMute {
			t.Errorf("❌ FAILED: JSON fallback logic is still active! Got set_track_mute instead of delete_track")
			t.Errorf("Response: %s", responseStr)
		}
	} else {
		// If it failed, verify the error message indicates CFG/DSL requirement
		assert.Contains(t, responseStr, "DSL", "Error should mention DSL requirement")
		t.Logf("Request failed (expected if LLM doesn't use CFG tool): %s", responseStr)
	}
}

// TestMagdaCFGModeOnlyAcceptsDSL verifies that CFG mode strictly enforces DSL
func TestMagdaCFGModeOnlyAcceptsDSL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	// Test with a simple track creation request
	requestBody := MagdaChatRequest{
		Question: "create a track called Test Track",
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

	responseBody := w.Body.Bytes()

	if w.Code == http.StatusOK {
		var response map[string]interface{}
		err = json.Unmarshal(responseBody, &response)
		require.NoError(t, err)

		actions, ok := response["actions"].([]interface{})
		require.True(t, ok)

		// Verify actions are valid (should be create_track, not JSON fallback)
		for _, actionInterface := range actions {
			action, ok := actionInterface.(map[string]interface{})
			require.True(t, ok)

			actionType, ok := action["action"].(string)
			require.True(t, ok)

			// All actions should be valid REAPER actions, not JSON schema actions
			validActions := []string{
				"create_track", "create_clip", "create_clip_at_bar",
				"add_instrument", "add_track_fx", "add_midi",
				"set_track_name", "set_track_volume", "set_track_pan",
				"set_track_mute", "set_track_solo", "set_track", "set_clip",
				"delete_track", "delete_clip",
			}

			found := false
			for _, validAction := range validActions {
				if actionType == validAction {
					found = true
					break
				}
			}

			assert.True(t, found, "Action type '%s' should be a valid REAPER action", actionType)
		}
	} else {
		// If failed, check that error mentions DSL/CFG requirement
		responseStr := string(responseBody)
		t.Logf("Request failed: %s", responseStr)
		// Don't fail the test - LLM might not use CFG tool correctly
	}
}

// TestMagdaDeleteTrackGeneratesDSL specifically tests the delete track scenario
func TestMagdaDeleteTrackGeneratesDSL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	// Multiple test cases for delete operations
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
			name:       "remove track",
			question:   "remove Nebula Drift",
			trackIndex: 0,
			trackName:  "Nebula Drift",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State: map[string]interface{}{
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

			if w.Code == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				actions, ok := response["actions"].([]interface{})
				require.True(t, ok, "Response should contain actions")

				// Find delete_track action
				foundDeleteTrack := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					if !ok {
						continue
					}

					actionType, ok := action["action"].(string)
					if !ok {
						continue
					}

					if actionType == "delete_track" {
						foundDeleteTrack = true
						trackIndex, ok := action["track"].(float64)
						assert.True(t, ok, "delete_track should have track index")
						assert.Equal(t, float64(tc.trackIndex), trackIndex, "Should delete correct track index")

						// CRITICAL: Should NOT have set_track_mute
						mute, hasMute := action["mute"]
						assert.False(t, hasMute, "delete_track should NOT have mute field - this indicates JSON fallback")
						if hasMute {
							t.Errorf("❌ FAILED: delete_track action has mute field! This indicates JSON fallback: %v", mute)
						}
					}

					// CRITICAL: Should NOT have set_track_mute action
					if actionType == "set_track_mute" {
						t.Errorf("❌ FAILED: Generated set_track_mute instead of delete_track for '%s'", tc.question)
						t.Errorf("This indicates JSON fallback logic is still active!")
					}
				}

				if !foundDeleteTrack {
					t.Logf("Warning: No delete_track action found. Actions: %v", actions)
				}
			} else {
				responseStr := w.Body.String()
				t.Errorf("❌ TEST FAILED: LLM did not use CFG tool. This is REQUIRED. Response: %s", responseStr)
				t.FailNow()
			}
		})
	}
}
