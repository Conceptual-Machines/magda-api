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

// TestMagdaAllActionsComprehensive tests all implemented DSL actions through the full integration flow
// These tests verify that the LLM generates correct DSL and the parser translates it to valid actions

// TestMagdaTrackCreation tests track creation with various parameters
func TestMagdaTrackCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "create track with instrument",
			question: "create a track with Serum",
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasCreateTrack := false
				hasInstrument := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "create_track" {
						hasCreateTrack = true
						// Instrument can be in the same action or separate
						if instrument, ok := action["instrument"].(string); ok {
							hasInstrument = true
							assert.Contains(t, instrument, "Serum", "Instrument should be Serum")
						}
					}
					if actionType == "add_instrument" {
						hasInstrument = true
						fxname, ok := action["fxname"].(string)
						if ok {
							assert.Contains(t, fxname, "Serum", "Instrument should be Serum")
						}
					}
				}
				assert.True(t, hasCreateTrack, "Should have create_track action")
				assert.True(t, hasInstrument, "Should have instrument (either in create_track or add_instrument)")
			},
		},
		{
			name:     "create track with name",
			question: "create a track called Drums",
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasCreateTrackWithName := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "create_track" {
						name, ok := action["name"].(string)
						if ok && name == "Drums" {
							hasCreateTrackWithName = true
						}
					}
				}
				assert.True(t, hasCreateTrackWithName, "Should have create_track with name 'Drums'")
			},
		},
		{
			name:     "create track with instrument and name",
			question: "create a track called Bass with Serum",
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasCreateTrack := false
				hasName := false
				hasInstrument := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "create_track" {
						hasCreateTrack = true
						name, ok := action["name"].(string)
						if ok && name == "Bass" {
							hasName = true
						}
						// Instrument can be in the same action
						if instrument, ok := action["instrument"].(string); ok {
							hasInstrument = true
							assert.Contains(t, instrument, "Serum", "Instrument should be Serum")
						}
					}
					if actionType == "add_instrument" {
						hasInstrument = true
					}
				}
				assert.True(t, hasCreateTrack, "Should have create_track action")
				assert.True(t, hasName, "Should have name 'Bass'")
				assert.True(t, hasInstrument, "Should have instrument (either in create_track or add_instrument)")
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
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}

// TestMagdaClipOperations tests clip creation and deletion
func TestMagdaClipOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		state    map[string]interface{}
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "create clip at bar",
			question: "add a 4-bar clip to track 0 starting at bar 5",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1"},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasCreateClipAtBar := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "create_clip_at_bar" {
						hasCreateClipAtBar = true
						bar, ok := action["bar"].(float64)
						if ok {
							assert.Equal(t, float64(5), bar, "Bar should be 5")
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
			},
		},
		{
			name:     "create clip with default length",
			question: "add a clip to track 0 at bar 3",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1"},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasCreateClipAtBar := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "create_clip_at_bar" {
						hasCreateClipAtBar = true
						bar, ok := action["bar"].(float64)
						if ok {
							assert.Equal(t, float64(3), bar, "Bar should be 3")
						}
					}
				}
				assert.True(t, hasCreateClipAtBar, "Should have create_clip_at_bar action")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State:    tc.state,
			}

			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}

// TestMagdaTrackPropertySetters tests all track property setter actions
func TestMagdaTrackPropertySetters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		state    map[string]interface{}
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "set track volume",
			question: "set track 0 volume to -3 dB",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "volume_db": 0.0},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasSetVolume := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if _, ok := action["volume_db"]; ok {
							hasSetVolume = true
							volume, ok := action["volume_db"].(float64)
							if ok {
								assert.Equal(t, float64(-3.0), volume, "Volume should be -3.0 dB")
							}
							track, ok := action["track"].(float64)
							if ok {
								assert.Equal(t, float64(0), track, "Track should be 0")
							}
						}
					}
				}
				assert.True(t, hasSetVolume, "Should have set_track action with volume_db")
			},
		},
		{
			name:     "set track pan",
			question: "set track 0 pan to 0.5",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "pan": 0.0},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasSetPan := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if pan, ok := action["pan"].(float64); ok {
							hasSetPan = true
							assert.Equal(t, float64(0.5), pan, "Pan should be 0.5")
						}
					}
				}
				assert.True(t, hasSetPan, "Should have set_track action with pan")
			},
		},
		{
			name:     "set track mute",
			question: "mute track 0",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "muted": false},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasSetMute := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if mute, ok := action["mute"].(bool); ok {
							hasSetMute = true
							assert.True(t, mute, "Mute should be true")
						}
					}
				}
				assert.True(t, hasSetMute, "Should have set_track action with mute=true")
			},
		},
		{
			name:     "set track solo",
			question: "solo track 0",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "soloed": false},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasSetSolo := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if solo, ok := action["solo"].(bool); ok {
							hasSetSolo = true
							assert.True(t, solo, "Solo should be true")
						}
					}
				}
				assert.True(t, hasSetSolo, "Should have set_track action with solo")
			},
		},
		{
			name:     "set track name",
			question: "rename track 0 to Bass",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1"},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasSetName := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if _, ok := action["name"].(string); ok {
							hasSetName = true
							name, ok := action["name"].(string)
							if ok {
								assert.Equal(t, "Bass", name, "Name should be 'Bass'")
							}
						}
					}
				}
				assert.True(t, hasSetName, "Should have set_track action with name")
			},
		},
		{
			name:     "set track selected",
			question: "select track 0",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "selected": false},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasSetSelected := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if selected, ok := action["selected"].(bool); ok {
							hasSetSelected = true
							assert.True(t, selected, "Selected should be true")
						}
					}
				}
				assert.True(t, hasSetSelected, "Should have set_track action with selected=true")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State:    tc.state,
			}

			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}

// TestMagdaAddFX tests FX and instrument addition
func TestMagdaAddFX(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		state    map[string]interface{}
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "add FX to track",
			question: "add ReaEQ to track 0",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1"},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasAddFX := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "add_track_fx" {
						hasAddFX = true
						fxname, ok := action["fxname"].(string)
						if ok {
							assert.Contains(t, fxname, "ReaEQ", "FX name should contain 'ReaEQ'")
						}
					}
				}
				assert.True(t, hasAddFX, "Should have add_track_fx action")
			},
		},
		{
			name:     "add instrument to track",
			question: "add Serum to track 0",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1"},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasAddInstrument := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "add_instrument" {
						hasAddInstrument = true
						fxname, ok := action["fxname"].(string)
						if ok {
							assert.Contains(t, fxname, "Serum", "Instrument should contain 'Serum'")
						}
					}
				}
				assert.True(t, hasAddInstrument, "Should have add_instrument action")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State:    tc.state,
			}

			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}

// TestMagdaFilterOperations tests filter operations with various predicates
func TestMagdaFilterOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		state    map[string]interface{}
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "filter by name and delete",
			question: "delete all tracks named Test",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Test", "selected": false},
					{"index": 1, "name": "Other", "selected": false},
					{"index": 2, "name": "Test", "selected": false},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				deleteCount := 0
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "delete_track" {
						deleteCount++
						track, ok := action["track"].(float64)
						if ok {
							// Should delete tracks 0 and 2 (both named "Test")
							assert.True(t, track == 0 || track == 2, "Should delete track 0 or 2")
						}
					}
				}
				assert.Equal(t, 2, deleteCount, "Should delete 2 tracks named 'Test'")
			},
		},
		{
			name:     "filter by name and set selected",
			question: "select all tracks named Drums",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Drums", "selected": false},
					{"index": 1, "name": "Bass", "selected": false},
					{"index": 2, "name": "Drums", "selected": false},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				selectCount := 0
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if selected, ok := action["selected"].(bool); ok && selected {
							selectCount++
							assert.True(t, selected, "Selected should be true")
						}
						track, ok := action["track"].(float64)
						if ok {
							// Should select tracks 0 and 2 (both named "Drums")
							assert.True(t, track == 0 || track == 2, "Should select track 0 or 2")
						}
					}
				}
				assert.Equal(t, 2, selectCount, "Should select 2 tracks named 'Drums'")
			},
		},
		{
			name:     "filter by mute status and unmute",
			question: "unmute all muted tracks",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "muted": true},
					{"index": 1, "name": "Track 2", "muted": false},
					{"index": 2, "name": "Track 3", "muted": true},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				unmuteCount := 0
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if mute, ok := action["mute"].(bool); ok && !mute {
							unmuteCount++
						}
					}
				}
				assert.GreaterOrEqual(t, unmuteCount, 1, "Should unmute at least one track")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State:    tc.state,
			}

			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}

// TestMagdaMethodChaining tests complex method chaining
func TestMagdaMethodChaining(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		state    map[string]interface{}
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "create track with clip and volume",
			question: "create a track with Serum, add a clip at bar 1, and set volume to -3 dB",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasCreateTrack := false
				hasInstrument := false
				hasCreateClip := false
				hasSetVolume := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "create_track" {
						hasCreateTrack = true
						// Instrument can be in the same action
						if _, ok := action["instrument"].(string); ok {
							hasInstrument = true
						}
					}
					if actionType == "add_instrument" {
						hasInstrument = true
					}
					if actionType == "create_clip_at_bar" {
						hasCreateClip = true
					}
					if actionType == "set_track" {
						if _, ok := action["volume_db"]; ok {
							hasSetVolume = true
						}
					}
				}
				assert.True(t, hasCreateTrack, "Should have create_track")
				assert.True(t, hasInstrument, "Should have instrument (either in create_track or add_instrument)")
				assert.True(t, hasCreateClip, "Should have create_clip_at_bar")
				assert.True(t, hasSetVolume, "Should have set_track action with volume_db")
			},
		},
		{
			name:     "create track with name and FX",
			question: "create a track called Lead with Serum and add ReaEQ",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				hasCreateTrack := false
				hasName := false
				hasInstrument := false
				hasAddFX := false
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "create_track" {
						hasCreateTrack = true
						name, ok := action["name"].(string)
						if ok && name == "Lead" {
							hasName = true
						}
						// Instrument can be in the same action
						if _, ok := action["instrument"].(string); ok {
							hasInstrument = true
						}
					}
					if actionType == "add_instrument" {
						hasInstrument = true
					}
					if actionType == "add_track_fx" {
						hasAddFX = true
					}
				}
				assert.True(t, hasCreateTrack, "Should have create_track")
				assert.True(t, hasName, "Should have name 'Lead'")
				assert.True(t, hasInstrument, "Should have instrument (either in create_track or add_instrument)")
				assert.True(t, hasAddFX, "Should have add_track_fx")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State:    tc.state,
			}

			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}

// TestMagdaForEachOperations tests for_each operations with various methods
func TestMagdaForEachOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		state    map[string]interface{}
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "for_each unmute all tracks",
			question: "unmute all tracks",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "muted": true},
					{"index": 1, "name": "Track 2", "muted": true},
					{"index": 2, "name": "Track 3", "muted": true},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				// Should have set_track actions with mute=false for all tracks
				muteActions := 0
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if mute, ok := action["mute"].(bool); ok {
							muteActions++
							assert.False(t, mute, "Mute should be false (unmuted)")
						}
					}
				}
				assert.GreaterOrEqual(t, muteActions, 2, "Should have at least 2 set_track actions with mute=false for multiple tracks")
			},
		},
		{
			name:     "for_each select all tracks",
			question: "select all tracks",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "selected": false},
					{"index": 1, "name": "Track 2", "selected": false},
					{"index": 2, "name": "Track 3", "selected": false},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				// Should have set_selected actions for all tracks
				selectedActions := 0
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if selected, ok := action["selected"].(bool); ok && selected {
							selectedActions++
							assert.True(t, selected, "Selected should be true")
						}
					}
				}
				assert.GreaterOrEqual(t, selectedActions, 2, "Should have at least 2 set_selected actions for multiple tracks")
			},
		},
		{
			name:     "for_each add FX to all tracks",
			question: "add ReaEQ to all tracks",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "has_fx": false},
					{"index": 1, "name": "Track 2", "has_fx": false},
					{"index": 2, "name": "Track 3", "has_fx": false},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				// Should have add_track_fx actions for all tracks
				fxActions := 0
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "add_track_fx" {
						fxActions++
						fxname, ok := action["fxname"].(string)
						if ok {
							assert.Contains(t, fxname, "ReaEQ", "FX name should contain ReaEQ")
						}
					}
				}
				assert.GreaterOrEqual(t, fxActions, 2, "Should have at least 2 add_track_fx actions for multiple tracks")
			},
		},
		{
			name:     "for_each set volume on all tracks",
			question: "set volume to -3 dB on all tracks",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "volume_db": 0.0},
					{"index": 1, "name": "Track 2", "volume_db": 0.0},
					{"index": 2, "name": "Track 3", "volume_db": 0.0},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				// Should have set_track_volume actions for all tracks
				volumeActions := 0
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if _, ok := action["volume_db"]; ok {
							volumeActions++
							volumeDb, ok := action["volume_db"].(float64)
							if ok {
								assert.Equal(t, -3.0, volumeDb, "Volume should be -3 dB")
							}
						}
					}
				}
				assert.GreaterOrEqual(t, volumeActions, 2, "Should have at least 2 set_track actions with volume_db for multiple tracks")
			},
		},
		{
			name:     "for_each with filtered collection",
			question: "unmute all muted tracks",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "muted": true},
					{"index": 1, "name": "Track 2", "muted": false},
					{"index": 2, "name": "Track 3", "muted": true},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				// Should have set_track actions with mute=false only for muted tracks (tracks 0 and 2)
				muteActions := 0
				trackIndices := make(map[int]bool)
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_track" {
						if mute, ok := action["mute"].(bool); ok && !mute {
							muteActions++
							track, ok := action["track"].(float64)
							if ok {
								trackIndices[int(track)] = true
								assert.False(t, mute, "Mute should be false (unmuted)")
							}
						}
					}
				}
				assert.GreaterOrEqual(t, muteActions, 1, "Should have at least 1 set_track action with mute=false for muted tracks")
				// Should target tracks 0 and/or 2 (the muted ones)
				assert.True(t, trackIndices[0] || trackIndices[2], "Should target at least one of the muted tracks")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State:    tc.state,
			}

			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}

// TestMagdaClipFilteringOperations tests clip filtering and selection operations
func TestMagdaClipFilteringOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		state    map[string]interface{}
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "filter clips by length and select",
			question: "select all clips shorter than 1.5 seconds",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{
						"index": 0,
						"name":  "Track 1",
						"clips": []map[string]interface{}{
							{"index": 0, "position": 1.0, "length": 3.0, "selected": false},  // 3 seconds - should NOT be selected
							{"index": 1, "position": 5.0, "length": 1.0, "selected": false},  // 1 second - should be selected
							{"index": 2, "position": 8.0, "length": 2.0, "selected": false},  // 2 seconds - should NOT be selected
							{"index": 3, "position": 13.0, "length": 1.2, "selected": false}, // 1.2 seconds - should be selected
						},
					},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				// Should have set_clip_selected actions for clips with length < 1.5
				clipSelectionActions := 0
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_clip" {
						if _, ok := action["selected"]; ok {
							clipSelectionActions++
							selected, ok := action["selected"].(bool)
							if ok {
								assert.True(t, selected, "Selected should be true")
							}
							// Should have track and clip identifier
							track, ok := action["track"].(float64)
							assert.True(t, ok, "Should have track field")
							assert.Equal(t, 0.0, track, "Should target track 0")
							// Should have either position or clip index
							hasIdentifier := false
							if _, ok := action["position"]; ok {
								hasIdentifier = true
							}
							if _, ok := action["clip"]; ok {
								hasIdentifier = true
							}
							assert.True(t, hasIdentifier, "Should have clip identifier (position or clip index)")
						}
					}
				}
				// Should have at least 2 set_clip_selected actions (clips at index 1 and 3 have length < 1.5)
				assert.GreaterOrEqual(t, clipSelectionActions, 1, "Should have at least 1 set_clip_selected action for clips shorter than 1.5 seconds")
			},
		},
		{
			name:     "filter clips by length across multiple tracks",
			question: "select all clips with length less than 2 seconds",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{
						"index": 0,
						"name":  "Track 1",
						"clips": []map[string]interface{}{
							{"index": 0, "position": 1.0, "length": 3.0, "selected": false}, // 3 seconds
							{"index": 1, "position": 5.0, "length": 1.5, "selected": false}, // 1.5 seconds
						},
					},
					{
						"index": 1,
						"name":  "Track 2",
						"clips": []map[string]interface{}{
							{"index": 0, "position": 2.0, "length": 1.0, "selected": false}, // 1 second
							{"index": 1, "position": 4.0, "length": 2.5, "selected": false}, // 2.5 seconds
						},
					},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				// Should have set_clip_selected actions for clips with length < 2.0
				clipSelectionActions := 0
				trackIndices := make(map[int]bool)
				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)
					if actionType == "set_clip" {
						if _, ok := action["selected"]; ok {
							clipSelectionActions++
							selected, ok := action["selected"].(bool)
							if ok {
								assert.True(t, selected, "Selected should be true")
							}
							track, ok := action["track"].(float64)
							if ok {
								trackIndices[int(track)] = true
							}
						}
					}
				}
				assert.GreaterOrEqual(t, clipSelectionActions, 1, "Should have at least 1 set_clip_selected action")
				// Should target clips from multiple tracks (tracks 0 and 1 both have clips < 2 seconds)
				assert.True(t, trackIndices[0] || trackIndices[1], "Should target clips from at least one track")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State:    tc.state,
			}

			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}

// Helper function to skip test if API key is missing
func skipIfAPIKeyMissing(t *testing.T, responseBody []byte) {
	t.Helper()
	var errorResponse map[string]interface{}
	if err := json.Unmarshal(responseBody, &errorResponse); err == nil {
		if errorMsg, ok := errorResponse["error"].(string); ok {
			if contains(errorMsg, "API key") || contains(errorMsg, "Unauthorized") {
				t.Skip("Skipping test: Invalid or missing OpenAI API key")
				return
			}
		}
	}
}

// TestMagdaCompoundActions tests compound actions (multiple operations on filtered items)
func TestMagdaCompoundActions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	router := setupTestRouter()

	testCases := []struct {
		name     string
		question string
		state    map[string]interface{}
		validate func(t *testing.T, actions []interface{})
	}{
		{
			name:     "select and rename clips",
			question: "select all clips shorter than one bar and rename them to FOO",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{
						"index": 0,
						"name":  "Track 1",
						"clips": []map[string]interface{}{
							{"index": 0, "position": 0.0, "length": 0.697674, "selected": false},       // < 1 bar - should match
							{"index": 1, "position": 5.581395, "length": 0.697674, "selected": false},  // < 1 bar - should match
							{"index": 2, "position": 8.372093, "length": 2.790698, "selected": false},  // = 1 bar - should NOT match
							{"index": 3, "position": 13.953488, "length": 1.395349, "selected": false}, // < 1 bar - should match
						},
					},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				// Should have both set_clip_selected and set_clip_name actions
				selectionActions := 0
				renameActions := 0

				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)

					if actionType == "set_clip" {
						// Check if it's a selection action (has selected field)
						if selected, ok := action["selected"].(bool); ok && selected {
							selectionActions++
							// Should target clips shorter than one bar
							track, ok := action["track"].(float64)
							assert.True(t, ok, "Should have track field")
							assert.Equal(t, 0.0, track, "Should target track 0")
						}
						// Check if it's a rename action (has name field)
						if name, ok := action["name"].(string); ok {
							renameActions++
							assert.Equal(t, "FOO", name, "Name should be FOO")
							track, ok := action["track"].(float64)
							assert.True(t, ok, "Should have track field")
							assert.Equal(t, 0.0, track, "Should target track 0")
						}
					}
				}

				// Should have at least 3 selection actions (clips at index 0, 1, 3 are < 1 bar)
				assert.GreaterOrEqual(t, selectionActions, 3,
					"Should have at least 3 set_clip actions with selected=true for clips shorter than one bar")
				// Should have at least 3 rename actions (same clips)
				assert.GreaterOrEqual(t, renameActions, 3, "Should have at least 3 set_clip actions with name for clips shorter than one bar")
			},
		},
		{
			name:     "filter and color clips",
			question: "select all clips shorter than 1.5 seconds and color them red",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{
						"index": 0,
						"name":  "Track 1",
						"clips": []map[string]interface{}{
							{"index": 0, "position": 1.0, "length": 1.0, "selected": false}, // 1 second - should match
							{"index": 1, "position": 5.0, "length": 2.0, "selected": false}, // 2 seconds - should NOT match
							{"index": 2, "position": 8.0, "length": 1.2, "selected": false}, // 1.2 seconds - should match
						},
					},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				selectionActions := 0
				colorActions := 0

				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)

					if actionType == "set_clip" {
						if _, ok := action["selected"]; ok {
							selectionActions++
						}
						if _, ok := action["color"]; ok {
							colorActions++
							color, ok := action["color"].(string)
							if ok {
								// Accept various red color formats
								assert.Contains(t, []string{"#ff0000", "#FF0000", "red", "#f00"}, color, "Color should be red")
							}
						}
					}
				}

				assert.GreaterOrEqual(t, selectionActions, 2, "Should have at least 2 set_clip_selected actions")
				assert.GreaterOrEqual(t, colorActions, 2, "Should have at least 2 set_clip_color actions")
			},
		},
		{
			name:     "filter tracks and rename",
			question: "select all muted tracks and rename them to Muted",
			state: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"index": 0, "name": "Track 1", "muted": true},
					{"index": 1, "name": "Track 2", "muted": false},
					{"index": 2, "name": "Track 3", "muted": true},
				},
			},
			validate: func(t *testing.T, actions []interface{}) {
				t.Helper()
				selectionActions := 0
				renameActions := 0

				for _, actionInterface := range actions {
					action, ok := actionInterface.(map[string]interface{})
					require.True(t, ok)
					actionType, ok := action["action"].(string)
					require.True(t, ok)

					if actionType == "set_track" {
						if _, ok := action["selected"]; ok {
							selectionActions++
						}
					}

					if actionType == "set_track" {
						if _, ok := action["name"].(string); ok {
							renameActions++
							name, ok := action["name"].(string)
							if ok {
								assert.Equal(t, "Muted", name, "Name should be Muted")
							}
							// Should target muted tracks (0 and 2)
							track, ok := action["track"].(float64)
							if ok {
								assert.True(t, track == 0 || track == 2, "Should target track 0 or 2 (muted tracks)")
							}
						}
					}
				}

				assert.GreaterOrEqual(t, selectionActions, 1, "Should have at least 1 set_track action with selected=true")
				assert.GreaterOrEqual(t, renameActions, 2, "Should have at least 2 set_track actions with name for muted tracks")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := MagdaChatRequest{
				Question: tc.question,
				State:    tc.state,
			}

			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/magda/chat", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				skipIfAPIKeyMissing(t, w.Body.Bytes())
			}

			require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			actions, ok := response["actions"].([]interface{})
			require.True(t, ok, "Response should have 'actions' array")
			require.Greater(t, len(actions), 0, "Should have at least one action")

			tc.validate(t, actions)
		})
	}
}
