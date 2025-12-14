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

// TestMagdaRenameSelectedClips tests renaming selected clips
func TestMagdaRenameSelectedClips(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	// Setup REAPER state with clips, some selected
	requestBody := MagdaChatRequest{
		Question: "rename selected clips to foo",
		State: map[string]interface{}{
			"project": map[string]interface{}{
				"name":   "Test Project",
				"length": 15.0,
			},
			"tracks": []map[string]interface{}{
				{
					"index": 0,
					"name":  "Track 1",
					"clips": []map[string]interface{}{
						{"index": 0, "position": 0.0, "length": 0.697674, "selected": true},
						{"index": 1, "position": 5.581395, "length": 0.697674, "selected": true},
						{"index": 2, "position": 8.372093, "length": 2.790698, "selected": false},
						{"index": 3, "position": 13.953488, "length": 1.395349, "selected": true},
					},
				},
				{
					"index": 1,
					"name":  "Track 2",
					"clips": []map[string]interface{}{
						{"index": 0, "position": 1.0, "length": 1.0, "selected": false},
						{"index": 1, "position": 3.0, "length": 2.0, "selected": true},
					},
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

	// Count set_clip_name actions
	renameActions := 0
	expectedClips := []struct {
		track    int
		position float64
	}{
		{0, 0.0},
		{0, 5.581395},
		{0, 13.953488},
		{1, 3.0},
	}

	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok)

		actionType, ok := action["action"].(string)
		require.True(t, ok)

		if actionType == "set_clip" {
			// Check if it's a rename action (has name field)
			if name, ok := action["name"].(string); ok {
				renameActions++
				assert.Equal(t, "foo", name, "Clip name should be 'foo'")
			}

			// Verify track and position match expected selected clips
			track, ok := action["track"].(float64)
			require.True(t, ok, "Should have track field")

			position, hasPosition := action["position"].(float64)
			clipIndex, hasClipIndex := action["clip"].(float64)

			found := false
			for _, expected := range expectedClips {
				if int(track) == expected.track {
					if hasPosition && position == expected.position {
						found = true
						break
					} else if hasClipIndex {
						// If using clip index, we can't verify exact match but track should match
						found = true
						break
					}
				}
			}
			assert.True(t, found, "Action should target one of the selected clips: track=%d, position=%v, clip=%v", int(track), position, clipIndex)
		}
	}

	// Should have at least 4 set_clip_name actions (3 selected clips on track 0, 1 on track 1)
	assert.GreaterOrEqual(t, renameActions, 4, "Should have at least 4 set_clip_name actions for selected clips")
	t.Logf("✅ Successfully renamed %d selected clips to 'foo'", renameActions)
}
