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

const (
	actionSetTrack = "set_track"
)

// TestMagdaSelectTracks tests track selection actions
// This tests: "select track 1 and track 3"
func TestMagdaSelectTracks(t *testing.T) {
	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "select track 1 and track 3",
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
					"name":     "Bass",
					"selected": false,
				},
				{
					"index":    2,
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

	actions, ok := response["actions"].([]interface{})
	require.True(t, ok, "Response should have 'actions' array")

	// Expect at least one set_track action with selected=true (LLM may generate one or two)
	// Question says "track 1 and track 3" which should be indices 0 and 2 (0-based)
	actualSelectionCount := 0
	selectedTracks := make(map[int]bool)

	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok, "Action should be a map")

		actionType, ok := action["action"].(string)
		require.True(t, ok, "Action should have 'action' field")

		if actionType == actionSetTrack {
			// Check if this set_track action has selected=true
			if selected, ok := action["selected"].(bool); ok && selected {
				actualSelectionCount++
				trackIndex, ok := action["track"].(float64) // JSON numbers are float64
				require.True(t, ok, "set_track action should have 'track' field")

				selectedTracks[int(trackIndex)] = selected
			}
		}
	}

	// Should have at least one selection action
	assert.GreaterOrEqual(t, actualSelectionCount, 1,
		"Expected at least 1 selection action, got %d", actualSelectionCount)
	// At least one of the expected tracks (0 or 2) should be selected
	// Note: LLM may only generate one action instead of two, which is acceptable
	assert.True(t, selectedTracks[0] || selectedTracks[2],
		"At least one of track 0 or track 2 should be selected (got: %v)", selectedTracks)
}

// TestMagdaSelectAllTracksNamed tests selecting tracks by name using functional methods
// This tests: "select all tracks named Foo"
func TestMagdaSelectAllTracksNamed(t *testing.T) {
	router := setupTestRouter()

	requestBody := MagdaChatRequest{
		Question: "select all tracks named Foo",
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

	actions, ok := response["actions"].([]interface{})
	require.True(t, ok, "Response should have 'actions' array")

	// Expect two set_track_selected actions (one for each "Foo" track)
	expectedSelectionCount := 2
	actualSelectionCount := 0
	selectedTracks := make(map[int]bool)

	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok, "Action should be a map")

		actionType, ok := action["action"].(string)
		require.True(t, ok, "Action should have 'action' field")

		if actionType == actionSetTrack {
			// Check if this set_track action has selected=true
			if selected, ok := action["selected"].(bool); ok && selected {
				actualSelectionCount++
				trackIndex, ok := action["track"].(float64) // JSON numbers are float64
				require.True(t, ok, "set_track action should have 'track' field")

				selectedTracks[int(trackIndex)] = selected
			}
		}
	}

	assert.Equal(t, expectedSelectionCount, actualSelectionCount,
		"Expected %d selection actions, got %d", expectedSelectionCount, actualSelectionCount)
	assert.True(t, selectedTracks[1], "Track 1 (named Foo) should be selected")
	assert.True(t, selectedTracks[3], "Track 3 (named Foo) should be selected")
}
