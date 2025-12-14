package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMagdaSelectAndColorClips tests selecting clips by condition and coloring them
// This test verifies the specific use case: "select all clips shorter than one bar and color them blue"
func TestMagdaSelectAndColorClips(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	// Setup REAPER state with clips of varying lengths
	// One bar at 120 BPM ≈ 2.790698 seconds
	requestBody := MagdaChatRequest{
		Question: "select all clips shorter than one bar and color them blue",
		State: map[string]interface{}{
			"project": map[string]interface{}{
				"name":   "Test Project",
				"length": 20.0,
			},
			"tracks": []map[string]interface{}{
				{
					"index": 0,
					"name":  "Track 1",
					"clips": []map[string]interface{}{
						// Short clips (should be selected and colored)
						{"index": 0, "position": 0.0, "length": 1.395349, "selected": false},       // < 1 bar
						{"index": 1, "position": 2.790698, "length": 0.0, "selected": false},       // < 1 bar (empty clip)
						{"index": 2, "position": 4.186047, "length": 2.790698, "selected": false},  // = 1 bar (should NOT be selected)
						{"index": 3, "position": 6.976744, "length": 0.0, "selected": false},       // < 1 bar (empty clip)
						{"index": 4, "position": 8.372093, "length": 1.395349, "selected": false},  // < 1 bar
						{"index": 5, "position": 9.767442, "length": 4.186047, "selected": false},  // > 1 bar (should NOT be selected)
						{"index": 6, "position": 13.953488, "length": 0.0, "selected": false},      // < 1 bar (empty clip)
						{"index": 7, "position": 15.348837, "length": 1.395349, "selected": false}, // < 1 bar
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

	// Expected clips shorter than one bar (length < 2.790698)
	expectedShortClips := []struct {
		position float64
		length   float64
	}{
		{0.0, 1.395349},       // index 0
		{2.790698, 0.0},       // index 1
		{6.976744, 0.0},       // index 3
		{8.372093, 1.395349},  // index 4
		{13.953488, 0.0},      // index 6
		{15.348837, 1.395349}, // index 7
	}

	// Track selection and coloring actions
	selectionActions := 0
	colorActions := 0
	selectedPositions := make(map[float64]bool)
	coloredPositions := make(map[float64]bool)

	for _, actionInterface := range actions {
		action, ok := actionInterface.(map[string]interface{})
		require.True(t, ok)

		actionType, ok := action["action"].(string)
		require.True(t, ok)

		// Should use unified set_clip action, NOT set_clip_selected
		assert.NotEqual(t, "set_clip_selected", actionType, "Should NOT use deprecated set_clip_selected action")

		if actionType == "set_clip" {
			track, ok := action["track"].(float64)
			require.True(t, ok, "Should have track field")
			assert.Equal(t, float64(0), track, "All clips should be on track 0")

			position, hasPosition := action["position"].(float64)
			clipIndex, hasClipIndex := action["clip"].(float64)

			// Check for selection
			if selected, ok := action["selected"].(bool); ok && selected {
				selectionActions++
				if hasPosition {
					selectedPositions[position] = true
				} else if hasClipIndex {
					// If using clip index, we can't verify exact position but should count it
					selectedPositions[float64(int(clipIndex))] = true
				}
			}

			// Check for color
			if color, ok := action["color"].(string); ok {
				colorActions++
				// Color should be hex (converted from "blue" by parser)
				// Accept either hex or the color name (parser converts it)
				assert.True(t,
					color == "#0000ff" || color == "blue" || strings.HasPrefix(color, "#"),
					"Color should be hex (e.g., #0000ff) or 'blue', got: %s", color)

				if hasPosition {
					coloredPositions[position] = true
				} else if hasClipIndex {
					coloredPositions[float64(int(clipIndex))] = true
				}
			}
		}
	}

	// Verify we have both selection and coloring actions
	assert.Greater(t, selectionActions, 0, "Should have at least one set_clip(selected=true) action")
	assert.Greater(t, colorActions, 0, "Should have at least one set_clip(color=...) action")

	// Verify we have actions for all expected short clips
	// Note: The LLM might generate separate actions for selection and coloring,
	// or combined actions, so we check that we have at least the expected number
	expectedCount := len(expectedShortClips)
	assert.GreaterOrEqual(t, selectionActions, expectedCount,
		"Should have selection actions for all %d short clips, got %d", expectedCount, selectionActions)
	assert.GreaterOrEqual(t, colorActions, expectedCount,
		"Should have color actions for all %d short clips, got %d", expectedCount, colorActions)

	t.Logf("✅ Successfully selected %d clips and colored %d clips", selectionActions, colorActions)
	t.Logf("   Selected positions: %v", selectedPositions)
	t.Logf("   Colored positions: %v", coloredPositions)
}
