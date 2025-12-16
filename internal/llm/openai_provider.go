package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Conceptual-Machines/grammar-school-go/gs"
	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/getsentry/sentry-go"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

const (
	// Role constants
	userRole       = "user"
	developerRole  = "developer"
	maxOutputTrunc = 200
	mcpCallType    = "mcp_call"
	// File permission for debug files
	debugFilePerms = 0644

	// Reasoning effort levels (GPT-5.2 supports: none, low, medium, high, xhigh)
	reasoningNone    = "none" // Default in GPT-5.2, lowest latency
	reasoningMinimal = "minimal"
	reasoningLow     = "low"
	reasoningMedium  = "medium"
	reasoningHigh    = "high"
	reasoningXHigh   = "xhigh" // New in GPT-5.2, maximum reasoning

	// Heartbeat interval for streaming (send every 10 seconds to keep connection alive during long operations)
	heartbeatIntervalSeconds = 10
	reasoningMin             = "min"
	reasoningMed             = "med"

	// Provider name
	providerNameOpenAI = "openai"

	// Logging limits
	maxArgsLogLength       = 100
	maxLogEventCountOpenAI = 5

	// String truncation limits for logging
	logTruncateShort      = 200   // Short truncation for field values
	logTruncateLong       = 500   // Long truncation for full field values
	logDumpLimit          = 5000  // Limit for dumping full structures
	logDumpMaxLimit       = 10000 // Maximum limit for large dumps
	maxPreviewChars       = 200
	maxErrorPreviewChars  = 500
	maxErrorResponseChars = 200
	maxPathPreviewLen     = 10
)

// OpenAIProvider implements the Provider interface using OpenAI's Responses API
type OpenAIProvider struct {
	client *openai.Client
	apiKey string // Store API key for raw HTTP requests when needed
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIProvider{
		client: &client,
		apiKey: apiKey,
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return providerNameOpenAI
}

// Generate implements non-streaming generation using OpenAI's Responses API
//
//nolint:gocyclo // Complex logic needed for handling CFG, JSON Schema, and standard requests
func (p *OpenAIProvider) Generate(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error) {
	startTime := time.Now()
	log.Printf("🎵 OPENAI GENERATION REQUEST STARTED (Model: %s)", request.Model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "openai.generate")
	defer transaction.Finish()

	transaction.SetTag("model", request.Model)
	transaction.SetTag("provider", "openai")
	transaction.SetTag("mcp_enabled", fmt.Sprintf("%t", request.MCPConfig != nil))

	// Build OpenAI-specific request parameters
	params := p.buildRequestParams(request)

	log.Printf("🚨 CRITICAL: About to call OpenAI API with params.Model='%s'", params.Model)

	// Call OpenAI API with Sentry span
	span := transaction.StartChild("openai.api_call")
	apiStartTime := time.Now()

	// Use raw HTTP request if we need CFG tools or verbosity
	// Marshal params to JSON, modify as needed, make raw HTTP request
	var resp *responses.Response
	var err error
	if request.CFGGrammar != nil || request.OutputSchema != nil {
		paramsJSON, _ := json.Marshal(params)
		var paramsMap map[string]any
		if json.Unmarshal(paramsJSON, &paramsMap) == nil {
			// Add verbosity to text if OutputSchema is provided
			if request.OutputSchema != nil {
				if text, ok := paramsMap["text"].(map[string]any); ok {
					text["verbosity"] = "low"
					log.Printf("✅ Added verbosity=low to text parameter")
				}
			}

			// Add CFG tool if configured
			if request.CFGGrammar != nil {
				// Use github.com/Conceptual-Machines/grammar-school-go utility to build OpenAI CFG tool payload
				cfgTool := gs.BuildOpenAICFGTool(gs.CFGConfig{
					ToolName:    request.CFGGrammar.ToolName,
					Description: request.CFGGrammar.Description,
					Grammar:     request.CFGGrammar.Grammar,
					Syntax:      request.CFGGrammar.Syntax,
				})

				// Set text format to plain text (not JSON schema) when using CFG
				paramsMap["text"] = gs.GetOpenAITextFormatForCFG()

				// Initialize tools array if not present
				var tools []any
				if paramsMap["tools"] == nil {
					tools = []any{}
				} else {
					var ok bool
					tools, ok = paramsMap["tools"].([]any)
					if !ok {
						// If tools is not a slice, try to convert from existing tools
						if existingTools, ok := paramsMap["tools"].([]responses.ToolUnionParam); ok {
							tools = make([]any, 0, len(existingTools))
							for _, t := range existingTools {
								toolJSON, _ := json.Marshal(t)
								var toolMap map[string]any
								if unmarshalErr := json.Unmarshal(toolJSON, &toolMap); unmarshalErr != nil {
									log.Printf("⚠️  Failed to unmarshal tool: %v", unmarshalErr)
									continue
								}
								tools = append(tools, toolMap)
							}
						} else {
							tools = []any{}
						}
					}
				}
				tools = append(tools, cfgTool)
				paramsMap["tools"] = tools
				paramsMap["parallel_tool_calls"] = false // CFG tools typically don't use parallel calls

				log.Printf("🔧 Added CFG tool: %s (syntax: %s)", request.CFGGrammar.ToolName, request.CFGGrammar.Syntax)
			}

			modifiedJSON, _ := json.Marshal(paramsMap)

			// Save full request payload to file
			log.Printf("🔍 DEBUG: request.CFGGrammar != nil = %v", request.CFGGrammar != nil)
			if request.CFGGrammar != nil {
				prettyJSON, _ := json.MarshalIndent(paramsMap, "", "  ")
				requestFile := "/tmp/openai_request_full.json"
				log.Printf("🔍 DEBUG: About to write request file: %s", requestFile)
				if writeErr := os.WriteFile(requestFile, prettyJSON, debugFilePerms); writeErr != nil {
					log.Printf("❌ FAILED to save request: %v", writeErr)
				} else {
					log.Printf("💾 Saved FULL request payload to %s (%d bytes)", requestFile, len(prettyJSON))
				}
			}

			log.Printf("📤 Making raw HTTP request (JSON size: %d bytes)", len(modifiedJSON))
			req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewReader(modifiedJSON))
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
			req.Header.Set("Content-Type", "application/json")

			httpResp, httpErr := http.DefaultClient.Do(req)
			if httpErr == nil {
				defer func() {
					if closeErr := httpResp.Body.Close(); closeErr != nil {
						log.Printf("⚠️  Failed to close response body: %v", closeErr)
					}
				}()
				body, _ := io.ReadAll(httpResp.Body)

				// Save full response payload to file
				if request.CFGGrammar != nil && httpResp.StatusCode == http.StatusOK {
					responseFile := "/tmp/openai_response_full.json"
					if writeErr := os.WriteFile(responseFile, body, debugFilePerms); writeErr != nil {
						log.Printf("❌ FAILED to save response: %v", writeErr)
					} else {
						log.Printf("💾 Saved FULL response payload to %s (%d bytes)", responseFile, len(body))
					}
				}

				if httpResp.StatusCode == http.StatusOK {
					resp = &responses.Response{}
					if json.Unmarshal(body, resp) != nil {
						err = fmt.Errorf("failed to parse response")
					} else {
						// Process response with CFG support
						processedResp, processErr := p.processResponseWithCFG(resp, startTime, transaction, request.OutputSchema, request.CFGGrammar)
						if processErr != nil {
							err = processErr
						} else {
							// Return the processed response
							return processedResp, nil
						}
					}
				} else {
					err = fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
				}
			} else {
				err = httpErr
			}
		}
	}

	// Fall back to SDK if raw request failed or no OutputSchema
	if resp == nil && err == nil {
		resp, err = p.client.Responses.New(ctx, params)
	}

	apiDuration := time.Since(apiStartTime)
	span.Finish()

	if err != nil {
		log.Printf("❌ OPENAI REQUEST FAILED after %v: %v", apiDuration, err)
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("openai request failed: %w", err)
	}

	log.Printf("⏱️  OPENAI API CALL COMPLETED in %v", apiDuration)

	// Process response
	result, err := p.processResponse(resp, startTime, transaction, request.OutputSchema)
	if err != nil {
		return nil, err
	}

	transaction.SetTag("success", "true")
	return result, nil
}

// GenerateStream implements streaming generation using OpenAI's Responses API
func (p *OpenAIProvider) GenerateStream(
	ctx context.Context, request *GenerationRequest, callback StreamCallback,
) (*GenerationResponse, error) {
	startTime := time.Now()
	log.Printf("🎵 OPENAI STREAMING GENERATION REQUEST STARTED (Model: %s)", request.Model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "openai.generate_stream")
	defer transaction.Finish()

	transaction.SetTag("model", request.Model)
	transaction.SetTag("provider", "openai")
	transaction.SetTag("streaming", "true")
	transaction.SetTag("mcp_enabled", fmt.Sprintf("%t", request.MCPConfig != nil))

	// Build OpenAI-specific request parameters
	// Note: CFG tools in streaming are not yet supported by the SDK
	// The LLM may still generate DSL as plain text, which we parse in parseActionsIncremental
	params := p.buildRequestParams(request)

	log.Printf("🚨 CRITICAL STREAMING: About to call OpenAI Streaming API with params.Model='%s'", params.Model)

	// Call OpenAI Streaming API
	stream := p.client.Responses.NewStreaming(ctx, params)

	// Process stream
	result, err := p.processStream(stream, callback, transaction, startTime)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, err
	}

	transaction.SetTag("success", "true")
	log.Printf("✅ STREAMING GENERATION COMPLETED in %v", time.Since(startTime))

	return result, nil
}

// buildRequestParams converts GenerationRequest to OpenAI-specific ResponseNewParams
func (p *OpenAIProvider) buildRequestParams(request *GenerationRequest) responses.ResponseNewParams {
	// Convert input_array to OpenAI messages format
	inputItems := responses.ResponseInputParam{}

	for _, item := range request.InputArray {
		role, hasRole := item["role"].(string)
		content, hasContent := item["content"].(string)

		if !hasRole || !hasContent {
			log.Printf("⚠️  Skipping invalid input item (missing role or content): %v", item)
			continue
		}

		// Convert role string to OpenAI enum
		var roleEnum responses.EasyInputMessageRole
		switch role {
		case developerRole:
			roleEnum = responses.EasyInputMessageRoleDeveloper
		case userRole:
			roleEnum = responses.EasyInputMessageRoleUser
		default:
			roleEnum = responses.EasyInputMessageRoleUser
		}

		inputItems = append(inputItems,
			responses.ResponseInputItemParamOfMessage(content, roleEnum),
		)
	}

	// Determine reasoning effort
	// GPT-5.2 supports: none (default), low, medium, high, xhigh
	var reasoningEffort shared.ReasoningEffort
	switch request.ReasoningMode {
	case reasoningNone:
		// "none" is the default in GPT-5.2 for lowest latency
		reasoningEffort = shared.ReasoningEffort("none")
	case reasoningMinimal, reasoningMin:
		// "minimal" maps to "low" for compatibility
		reasoningEffort = responses.ReasoningEffortLow
	case reasoningLow:
		reasoningEffort = responses.ReasoningEffortLow
	case reasoningMedium, reasoningMed:
		reasoningEffort = responses.ReasoningEffortMedium
	case reasoningHigh:
		reasoningEffort = responses.ReasoningEffortHigh
	case reasoningXHigh:
		// xhigh is new in GPT-5.2 for maximum reasoning
		reasoningEffort = shared.ReasoningEffort("xhigh")
	default:
		// Default to "medium" for music generation tasks (need reasoning for composition)
		reasoningEffort = responses.ReasoningEffortMedium
	}

	params := responses.ResponseNewParams{
		Model: request.Model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
		Instructions:      openai.String(request.SystemPrompt),
		ParallelToolCalls: openai.Bool(true),
		Reasoning: shared.ReasoningParam{
			Effort: reasoningEffort,
		},
	}

	// Add structured output schema if provided
	if request.OutputSchema != nil {
		params.Text = responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigParamOfJSONSchema(
				request.OutputSchema.Name,
				request.OutputSchema.Schema,
			),
		}
	}

	// Note: CFG tool support is handled in the raw HTTP request path (see Generate method)
	// This buildRequestParams method is used for SDK-based requests which don't support CFG yet

	// Add MCP tools if configured
	if request.MCPConfig != nil && request.MCPConfig.URL != "" {
		params.Tools = []responses.ToolUnionParam{
			{
				OfMcp: &responses.ToolMcpParam{
					ServerLabel: request.MCPConfig.Label,
					ServerURL:   request.MCPConfig.URL,
					RequireApproval: responses.ToolMcpRequireApprovalUnionParam{
						OfMcpToolApprovalFilter: &responses.ToolMcpRequireApprovalMcpToolApprovalFilterParam{
							Never: responses.ToolMcpRequireApprovalMcpToolApprovalFilterNeverParam{
								ToolNames: []string{}, // Empty = all tools never require approval
							},
						},
					},
				},
			},
		}
		log.Printf("🔗 MCP SERVER ENABLED: %s (label: %s)", request.MCPConfig.URL, request.MCPConfig.Label)
	}

	return params
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// extractCFGCodeFromArray extracts CFG code or input from an array of tool/output maps
func extractCFGCodeFromArray(items []any, arrayName string) string {
	log.Printf("🔍 Found '%s' array with %d items", arrayName, len(items))
	for j, item := range items {
		if itemMap, ok := item.(map[string]any); ok {
			log.Printf("🔍 %s[%d] keys: %v", arrayName, j, getMapKeys(itemMap))
			if code, ok := itemMap["code"].(string); ok && code != "" {
				log.Printf("🔧 Found CFG tool call code in %s[%d] (DSL): %s", arrayName, j, truncateString(code, maxPreviewChars))
				return code
			}
			if input, ok := itemMap["input"].(string); ok && input != "" {
				log.Printf("🔧 Found CFG tool call input in %s[%d] (DSL): %s", arrayName, j, truncateString(input, maxPreviewChars))
				return input
			}
		}
	}
	return ""
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// isDSLCode checks if a string looks like DSL code
// NOTE: We only support snake_case methods (new_clip, add_midi, delete_clip) - NOT camelCase
func (p *OpenAIProvider) isDSLCode(text string) bool {
	return strings.HasPrefix(text, "track(") ||
		strings.Contains(text, ".new_clip(") ||
		strings.Contains(text, ".add_midi(") ||
		strings.Contains(text, ".delete(") ||
		strings.Contains(text, ".delete_clip(") ||
		strings.Contains(text, ".filter(") ||
		strings.Contains(text, ".map(") ||
		strings.Contains(text, ".for_each(")
}

// validateCFGOutput validates that CFG output is DSL, not JSON
func (p *OpenAIProvider) validateCFGOutput(textOutput string) error {
	if textOutput == "" {
		return nil // Empty is handled elsewhere
	}

	// If it's DSL, it's valid
	if p.isDSLCode(textOutput) {
		return nil
	}

	// If it's JSON, that's invalid when CFG is configured
	if strings.HasPrefix(textOutput, "{") || strings.HasPrefix(textOutput, "[") {
		log.Printf("❌ CFG was configured but LLM generated JSON instead of using CFG tool")
		log.Printf("❌ JSON output (first %d chars): %s", maxPreviewChars, truncateString(textOutput, maxPreviewChars))
		return fmt.Errorf("CFG grammar was configured but LLM generated JSON in text output instead of using CFG tool. " +
			"LLM must use the CFG tool to generate DSL code")
	}

	// Otherwise it's invalid
	log.Printf("❌ CFG was configured but LLM output doesn't look like DSL")
	log.Printf("❌ Output (first %d chars): %s", maxPreviewChars, truncateString(textOutput, maxPreviewChars))
	return fmt.Errorf("CFG grammar was configured but LLM output doesn't look like DSL code. Expected format: track(id=0).delete() or similar")
}

// processResponse converts OpenAI Response to GenerationResponse
func (p *OpenAIProvider) processResponse(
	resp *responses.Response,
	startTime time.Time,
	transaction *sentry.Span,
	outputSchema *OutputSchema,
) (*GenerationResponse, error) {
	return p.processResponseWithCFG(resp, startTime, transaction, outputSchema, nil)
}

// processResponseWithCFG converts OpenAI Response to GenerationResponse, handling CFG tool calls
//
//nolint:gocyclo // Complex logic needed to check multiple possible locations for DSL code in response
func (p *OpenAIProvider) processResponseWithCFG(
	resp *responses.Response,
	startTime time.Time,
	transaction *sentry.Span,
	outputSchema *OutputSchema,
	cfgConfig *CFGConfig,
) (*GenerationResponse, error) {
	span := transaction.StartChild("process_response")
	defer span.Finish()

	// Check for CFG tool calls first
	if cfgConfig != nil {
		log.Printf("🔍 Searching for CFG tool call in %d output items", len(resp.Output))
		for i, outputItem := range resp.Output {
			// Try to extract tool call input (DSL code)
			// The structure depends on SDK version - we'll check multiple possibilities
			outputItemJSON, _ := json.Marshal(outputItem)
			var outputItemMap map[string]any
			if json.Unmarshal(outputItemJSON, &outputItemMap) == nil {
				log.Printf("🔍 Output item %d keys: %v", i, getMapKeys(outputItemMap))

				// IMMEDIATELY check code field - it's definitely there
				if codeVal, exists := outputItemMap["code"]; exists {
					log.Printf("🔍 'code' EXISTS! type=%T", codeVal)
					// Try string first
					if codeStr, ok := codeVal.(string); ok && codeStr != "" {
						log.Printf("🔧 Found CFG code as STRING: %s", truncateString(codeStr, maxPreviewChars))
						return &GenerationResponse{
							RawOutput: codeStr,
							Usage:     resp.Usage,
						}, nil
					}
					// Try converting to string directly
					if str, ok := codeVal.(string); ok && str != "" {
						log.Printf("🔧 Found CFG code (direct string): %s", truncateString(str, maxPreviewChars))
						return &GenerationResponse{
							RawOutput: str,
							Usage:     resp.Usage,
						}, nil
					}
					// Log what we actually got
					log.Printf("🔍 code value (raw): %+v", codeVal)
				} else {
					log.Printf("🔍 'code' field DOES NOT EXIST in map")
				}

				// DUMP FULL STRUCTURE FOR DEBUGGING - but check key fields first
				log.Printf("🔍 About to check all fields for output item %d (i==0: %v)", i, i == 0)
				if i == 0 {
					// Log all string fields that might contain DSL
					log.Printf("🔍 ========== CHECKING ALL STRING FIELDS IN OUTPUT ITEM FOR DSL CONTENT ==========")
					for key, val := range outputItemMap {
						if strVal, ok := val.(string); ok && strVal != "" {
							log.Printf("🔍 Field '%s' (string, %d chars): %s", key, len(strVal), truncateString(strVal, logTruncateLong))
							if p.isDSLCode(strVal) {
								log.Printf("🔧 ✅✅✅ FOUND DSL IN FIELD '%s': %s", key, truncateString(strVal, maxPreviewChars))
								return &GenerationResponse{
									RawOutput: strVal,
									Usage:     resp.Usage,
								}, nil
							}
						}
					}
					// Also check non-string fields that might be maps/arrays containing DSL
					log.Printf("🔍 ========== CHECKING NON-STRING FIELDS (arguments, tools, outputs) ==========")
					fullDump, _ := json.MarshalIndent(outputItemMap, "", "  ")
					dumpLen := len(fullDump)
					if dumpLen > logDumpMaxLimit {
						dumpLen = logDumpMaxLimit
					}
					log.Printf("🔍 FULL OUTPUT ITEM STRUCTURE (first %d chars):\n%s", dumpLen, string(fullDump[:dumpLen]))
				}

				// Check "code" field - try multiple type assertions
				if codeVal, exists := outputItemMap["code"]; exists {
					log.Printf("🔍 'code' field EXISTS: type=%T, value=%v", codeVal, codeVal)
					if codeStr, ok := codeVal.(string); ok && codeStr != "" {
						log.Printf("🔧 Found CFG tool call code (DSL): %s", truncateString(codeStr, maxPreviewChars))
						return &GenerationResponse{
							RawOutput: codeStr,
							Usage:     resp.Usage,
						}, nil
					}
					// Try as []byte
					if codeBytes, ok := codeVal.([]byte); ok && len(codeBytes) > 0 {
						codeStr := string(codeBytes)
						log.Printf("🔧 Found CFG tool call code (DSL from bytes): %s", truncateString(codeStr, maxPreviewChars))
						return &GenerationResponse{
							RawOutput: codeStr,
							Usage:     resp.Usage,
						}, nil
					}
				} else {
					log.Printf("🔍 'code' field DOES NOT EXIST")
				}

				// Check "arguments" field
				if argsVal, exists := outputItemMap["arguments"]; exists {
					log.Printf("🔍 'arguments' field EXISTS: type=%T", argsVal)
					// Log the full value for debugging
					argsJSON, _ := json.Marshal(argsVal)
					log.Printf("🔍 'arguments' field value (JSON): %s", truncateString(string(argsJSON), logTruncateLong))
					if argsStr, ok := argsVal.(string); ok && argsStr != "" {
						log.Printf("🔧 Found CFG tool call arguments (DSL): %s", truncateString(argsStr, maxPreviewChars))
						return &GenerationResponse{
							RawOutput: argsStr,
							Usage:     resp.Usage,
						}, nil
					}
					// Check if it's a map that might contain the DSL code
					if argsMap, ok := argsVal.(map[string]any); ok {
						log.Printf("🔍 'arguments' is a map with keys: %v", getMapKeys(argsMap))
						// Check common fields in arguments map
						for key, val := range argsMap {
							if strVal, ok := val.(string); ok && strVal != "" && len(strVal) > 10 {
								log.Printf("🔍 'arguments[%s]' = %s", key, truncateString(strVal, logTruncateShort))
								if p.isDSLCode(strVal) {
									log.Printf("🔧 Found DSL in arguments[%s]: %s", key, truncateString(strVal, maxPreviewChars))
									return &GenerationResponse{
										RawOutput: strVal,
										Usage:     resp.Usage,
									}, nil
								}
							}
						}
					}
				} else {
					log.Printf("🔍 'arguments' field DOES NOT EXIST")
				}

				// Check "result" field (might contain DSL code)
				if resultVal, exists := outputItemMap["result"]; exists {
					log.Printf("🔍 'result' field EXISTS: type=%T", resultVal)
					if resultStr, ok := resultVal.(string); ok && resultStr != "" {
						log.Printf("🔍 'result' field value (first 200 chars): %s", truncateString(resultStr, logTruncateShort))
						if p.isDSLCode(resultStr) {
							log.Printf("🔧 Found CFG tool call result (DSL): %s", truncateString(resultStr, maxPreviewChars))
							return &GenerationResponse{
								RawOutput: resultStr,
								Usage:     resp.Usage,
							}, nil
						}
					}
				}

				// Check "output" field (might contain DSL code)
				if outputVal, exists := outputItemMap["output"]; exists {
					log.Printf("🔍 'output' field EXISTS: type=%T", outputVal)
					if outputStr, ok := outputVal.(string); ok && outputStr != "" {
						log.Printf("🔍 'output' field value (first 200 chars): %s", truncateString(outputStr, logTruncateShort))
						if p.isDSLCode(outputStr) {
							log.Printf("🔧 Found CFG tool call output (DSL): %s", truncateString(outputStr, maxPreviewChars))
							return &GenerationResponse{
								RawOutput: outputStr,
								Usage:     resp.Usage,
							}, nil
						}
					}
				}

				// Check "content" field (might contain DSL code)
				if contentVal, exists := outputItemMap["content"]; exists {
					log.Printf("🔍 'content' field EXISTS: type=%T", contentVal)
					if contentStr, ok := contentVal.(string); ok && contentStr != "" {
						log.Printf("🔍 'content' field value (first 200 chars): %s", truncateString(contentStr, logTruncateShort))
						if p.isDSLCode(contentStr) {
							log.Printf("🔧 Found CFG tool call content (DSL): %s", truncateString(contentStr, maxPreviewChars))
							return &GenerationResponse{
								RawOutput: contentStr,
								Usage:     resp.Usage,
							}, nil
						}
					}
				}
				// Check "tools" array
				if tools, ok := outputItemMap["tools"].([]any); ok && len(tools) > 0 {
					if code := extractCFGCodeFromArray(tools, "tools"); code != "" {
						return &GenerationResponse{RawOutput: code, Usage: resp.Usage}, nil
					}
				}
				// Check "outputs" array
				if outputs, ok := outputItemMap["outputs"].([]any); ok && len(outputs) > 0 {
					if code := extractCFGCodeFromArray(outputs, "outputs"); code != "" {
						return &GenerationResponse{RawOutput: code, Usage: resp.Usage}, nil
					}
				}
				// Direct "input" field (tool call input)
				if input, ok := outputItemMap["input"].(string); ok && input != "" {
					log.Printf("🔧 Found CFG tool call input (DSL): %s", truncateString(input, maxPreviewChars))
					return &GenerationResponse{
						RawOutput: input,
						Usage:     resp.Usage,
					}, nil
				}

				// 2. Check for tool_calls array
				if toolCalls, ok := outputItemMap["tool_calls"].([]any); ok {
					for j, toolCall := range toolCalls {
						if toolCallMap, ok := toolCall.(map[string]any); ok {
							log.Printf("🔍 Tool call %d keys: %v", j, getMapKeys(toolCallMap))
							if input, ok := toolCallMap["input"].(string); ok && input != "" {
								log.Printf("🔧 Found CFG tool call input in tool_calls[%d] (DSL): %s", j, truncateString(input, maxPreviewChars))
								return &GenerationResponse{
									RawOutput: input,
									Usage:     resp.Usage,
								}, nil
							}
							// Also check for "function" -> "arguments" pattern
							if function, ok := toolCallMap["function"].(map[string]any); ok {
								if arguments, ok := function["arguments"].(string); ok && arguments != "" {
									log.Printf("🔧 Found CFG tool call arguments (DSL): %s", truncateString(arguments, maxPreviewChars))
									return &GenerationResponse{
										RawOutput: arguments,
										Usage:     resp.Usage,
									}, nil
								}
							}
						}
					}
				}

				// 3. Check for nested structure (output_item.tool_call.input)
				if toolCall, ok := outputItemMap["tool_call"].(map[string]any); ok {
					if input, ok := toolCall["input"].(string); ok && input != "" {
						log.Printf("🔧 Found CFG tool call input in tool_call (DSL): %s", truncateString(input, maxPreviewChars))
						return &GenerationResponse{
							RawOutput: input,
							Usage:     resp.Usage,
						}, nil
					}
				}
			}
		}
		log.Printf("⚠️  No CFG tool call found in response output items")

		// If CFG was expected but not found, log the full response structure for debugging
		if cfgConfig != nil && len(resp.Output) > 0 {
			log.Printf("🔍 CFG was configured but no tool call found. Dumping full response structure:")
			for i, outputItem := range resp.Output {
				outputItemJSON, _ := json.MarshalIndent(outputItem, "", "  ")
				dumpLen := len(outputItemJSON)
				if dumpLen > logDumpLimit {
					dumpLen = logDumpLimit
				}
				log.Printf("🔍 Output item %d (first %d chars):\n%s", i, dumpLen, string(outputItemJSON[:dumpLen]))
			}
		}
	}

	// Extract text output using SDK method
	textOutput := resp.OutputText()
	log.Printf("📥 OPENAI RESPONSE: output_length=%d, output_items=%d, tokens=%d",
		len(textOutput), len(resp.Output), resp.Usage.TotalTokens)

	if textOutput == "" {
		// If CFG was configured, this might be expected - the response might be in a different format
		if cfgConfig != nil {
			// Try one more time to extract from raw response structure
			if len(resp.Output) > 0 {
				log.Printf("⚠️  CFG configured but no text output. Attempting to extract from raw response structure...")
				// The response might have the DSL code in a different location
				// Let's check if there's any text-like content in the output items
				for _, outputItem := range resp.Output {
					outputItemJSON, _ := json.Marshal(outputItem)
					var outputItemMap map[string]any
					if json.Unmarshal(outputItemJSON, &outputItemMap) == nil {
						// Check for any string field that might contain DSL
						for key, val := range outputItemMap {
							if strVal, ok := val.(string); ok && len(strVal) > 10 {
								log.Printf("🔍 Found string field '%s' with %d chars: %s", key, len(strVal), truncateString(strVal, logTruncateShort))
								// If it looks like DSL code (contains track(), newClip(), etc.), use it
								if strings.Contains(strVal, "track(") || strings.Contains(strVal, "delete_track") || strings.Contains(strVal, "create_track") {
									log.Printf("🔧 Found DSL-like content in field '%s', using as output", key)
									return &GenerationResponse{
										RawOutput: strVal,
										Usage:     resp.Usage,
									}, nil
								}
							}
						}
					}
				}
			}
			return nil, fmt.Errorf("CFG grammar was configured but LLM did not use CFG tool to generate DSL code. " +
				"LLM must use the CFG tool to generate DSL code")
		}
		return nil, fmt.Errorf("openai response did not include any output text")
	}

	// CRITICAL: If CFG was configured, we MUST have DSL from tool call, not JSON in text output
	if cfgConfig != nil {
		// We already checked for CFG tool call above - if we got here, there's no tool call
		// and we have text output. This is an error - LLM must use CFG tool.
		// Validate that the text output is not JSON (which would indicate LLM ignored CFG)
		if err := p.validateCFGOutput(textOutput); err != nil {
			return nil, err
		}
		// If validation passed, textOutput contains DSL code (not JSON), which is valid
	}

	// Analyze MCP usage
	mcpUsed, mcpCalls, mcpTools := p.analyzeMCPUsage(resp)

	// Log MCP summary
	p.logMCPSummary(mcpUsed, mcpCalls, mcpTools)

	// Log usage stats
	p.logUsageStats(resp.Usage)

	// Parse JSON output based on schema type
	result := &GenerationResponse{
		Usage:    resp.Usage,
		MCPUsed:  mcpUsed,
		MCPCalls: mcpCalls,
		MCPTools: mcpTools,
	}

	// Check if this is MAGDA output (actions) or musical output (choices)
	if outputSchema != nil && outputSchema.Name == "MagdaActions" {
		// Store raw output for MAGDA - the service will parse it
		result.RawOutput = textOutput
		totalDuration := time.Since(startTime)
		log.Printf("✅ MAGDA GENERATION COMPLETED in %v (raw output stored)", totalDuration)
	} else {
		// Parse as musical output (default)
		var output models.MusicalOutput
		if err := json.Unmarshal([]byte(textOutput), &output); err != nil {
			log.Printf("❌ Failed to parse output JSON: %v", err)
			log.Printf("Raw output (first %d chars): %s", maxOutputTrunc, truncate(textOutput, maxOutputTrunc))
			return nil, fmt.Errorf("failed to parse model output: %w", err)
		}
		totalDuration := time.Since(startTime)
		log.Printf("✅ GENERATION COMPLETED in %v (choices: %d)", totalDuration, len(output.Choices))
		result.OutputParsed.Choices = output.Choices
	}

	return result, nil
}

// analyzeMCPUsage checks if MCP was used and returns usage details
func (p *OpenAIProvider) analyzeMCPUsage(resp *responses.Response) (bool, int, []string) {
	mcpUsed := false
	mcpCallCount := 0
	toolsUsed := make(map[string]bool)

	log.Printf("🔍 MCP USAGE ANALYSIS:")

	for _, outputItem := range resp.Output {
		if outputItem.Type == mcpCallType {
			mcpCall := outputItem.AsMcpCall()
			mcpUsed = true
			mcpCallCount++
			p.logMCPToolCall(mcpCall)
			toolsUsed[mcpCall.Name] = true

			// Add Sentry breadcrumb
			sentry.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "mcp",
				Message:  fmt.Sprintf("MCP tool called: %s", mcpCall.Name),
				Level:    sentry.LevelInfo,
				Data: map[string]interface{}{
					"tool_name":     mcpCall.Name,
					"server_label":  mcpCall.ServerLabel,
					"has_output":    mcpCall.Output != "",
					"output_length": len(mcpCall.Output),
					"has_error":     mcpCall.Error != "",
				},
			})
		}
	}

	uniqueTools := make([]string, 0, len(toolsUsed))
	for tool := range toolsUsed {
		uniqueTools = append(uniqueTools, tool)
	}

	if mcpCallCount == 0 {
		log.Printf("❌ MCP NOT USED: No MCP tool calls found in output")
	} else {
		log.Printf("📊 MCP TOOLS USED: %v", uniqueTools)
	}

	return mcpUsed, mcpCallCount, uniqueTools
}

// logMCPToolCall logs details of an MCP tool call
func (p *OpenAIProvider) logMCPToolCall(mcpCall responses.ResponseOutputItemMcpCall) {
	log.Printf("✅ MCP WAS USED: Tool call made")
	log.Printf("   🛠️  Tool Call: %s", mcpCall.Name)
	if mcpCall.Arguments != "" {
		argsStr := mcpCall.Arguments
		if len(argsStr) > maxArgsLogLength {
			argsStr = argsStr[:maxArgsLogLength] + "..."
		}
		log.Printf("     Arguments: %s", argsStr)
	}
	if mcpCall.Output != "" {
		output := mcpCall.Output
		if len(output) > maxOutputTrunc {
			output = output[:maxOutputTrunc] + "... (truncated)"
		}
		log.Printf("     Output: %s", output)
	}
	if mcpCall.Error != "" {
		log.Printf("     ⚠️  Error: %s", mcpCall.Error)
	}
}

// logUsageStats logs token usage statistics
func (p *OpenAIProvider) logUsageStats(usage responses.ResponseUsage) {
	reasoningTokens := int64(0)
	if usage.OutputTokensDetails.ReasoningTokens > 0 {
		reasoningTokens = usage.OutputTokensDetails.ReasoningTokens
	}
	log.Printf("📊 USAGE: input=%d, output=%d, reasoning=%d, total=%d",
		usage.InputTokens, usage.OutputTokens,
		reasoningTokens, usage.TotalTokens)
}

// logMCPSummary logs a summary of MCP usage
func (p *OpenAIProvider) logMCPSummary(mcpUsed bool, callCount int, tools []string) {
	if mcpUsed {
		log.Printf("🎯 MCP USAGE: %d calls to tools: %v", callCount, tools)
	} else {
		log.Printf("ℹ️  NO MCP USAGE in this generation")
	}
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// processStream processes the OpenAI streaming response
func (p *OpenAIProvider) processStream(
	stream *ssestream.Stream[responses.ResponseStreamEventUnion],
	callback StreamCallback,
	transaction *sentry.Span,
	startTime time.Time,
) (*GenerationResponse, error) {
	var finalResponse *models.MusicalOutput
	var mcpUsed bool
	var mcpCallCount int
	var mcpTools []string
	var usage any
	var accumulatedText string

	eventCount := 0

	// Start a background goroutine to send periodic heartbeats independent of stream events
	// This ensures heartbeats are sent even when stream.Next() blocks during long operations
	heartbeatDone := make(chan bool)
	go func() {
		ticker := time.NewTicker(heartbeatIntervalSeconds * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Send heartbeat
				elapsed := time.Since(startTime)
				_ = callback(StreamEvent{
					Type:    "heartbeat",
					Message: "Processing...",
					Data: map[string]any{
						"events_received": eventCount,
						"elapsed_seconds": int(elapsed.Seconds()),
						"note":            "Periodic heartbeat during stream processing",
					},
				})
			case <-heartbeatDone:
				return
			}
		}
	}()

	for stream.Next() {
		event := stream.Current()
		eventCount++

		// Also send heartbeat on event milestones (every 10 events)
		if eventCount%10 == 0 {
			if err := p.sendHeartbeat(eventCount, startTime, callback); err != nil {
				close(heartbeatDone)
				return nil, err
			}
		}

		// Handle the stream event
		if err := p.handleStreamEvent(
			event, eventCount, startTime, callback,
			&finalResponse, &mcpUsed, &mcpCallCount, &mcpTools, &usage, &accumulatedText,
		); err != nil {
			close(heartbeatDone)
			return nil, err
		}
	}

	// Stop heartbeat goroutine
	close(heartbeatDone)

	// Check for stream errors
	if streamErr := stream.Err(); streamErr != nil {
		log.Printf("❌ STREAMING ERROR: %v", streamErr)
		transaction.SetTag("error_type", "stream_error")
		_ = callback(StreamEvent{Type: "error", Message: fmt.Sprintf("Stream error: %v", streamErr)})
		return nil, fmt.Errorf("stream error: %w", streamErr)
	}

	if finalResponse == nil {
		log.Printf("❌ STREAM COMPLETE: finalResponse is nil - no output was parsed from stream")
		log.Printf("🔍 Stream ended with: eventCount=%d, accumulatedText length=%d", eventCount, len(accumulatedText))
		if accumulatedText != "" {
			log.Printf("⚠️  Accumulated text exists (%d chars) but was not parsed. Preview: %s",
				len(accumulatedText), truncate(accumulatedText, maxPreviewChars))
		}
		return nil, fmt.Errorf("no output received from stream")
	}

	log.Printf("✅ STREAM COMPLETE: finalResponse parsed successfully with %d choices", len(finalResponse.Choices))

	// Build result
	result := &GenerationResponse{
		Usage:    usage,
		MCPUsed:  mcpUsed,
		MCPCalls: mcpCallCount,
		MCPTools: mcpTools,
	}
	result.OutputParsed.Choices = finalResponse.Choices

	// Send completion event
	_ = callback(StreamEvent{
		Type:    "completed",
		Message: "Generation complete",
		Data: map[string]any{
			"choices_count": len(finalResponse.Choices),
			"mcp_used":      mcpUsed,
		},
	})

	totalDuration := time.Since(startTime)
	log.Printf("⏱️  STREAMING GENERATION TIME: %v (choices: %d)", totalDuration, len(finalResponse.Choices))

	return result, nil
}

// sendHeartbeat sends periodic heartbeat events
func (p *OpenAIProvider) sendHeartbeat(eventCount int, startTime time.Time, callback StreamCallback) error {
	elapsed := time.Since(startTime)
	if eventCount%10 == 0 || elapsed.Seconds() > 30 {
		return callback(StreamEvent{
			Type:    "heartbeat",
			Message: "Processing...",
			Data: map[string]any{
				"events_received": eventCount,
				"elapsed_seconds": int(elapsed.Seconds()),
			},
		})
	}
	return nil
}

// handleStreamEvent processes a single stream event
func (p *OpenAIProvider) handleStreamEvent(
	event responses.ResponseStreamEventUnion,
	eventCount int,
	startTime time.Time,
	callback StreamCallback,
	finalResponse **models.MusicalOutput,
	mcpUsed *bool,
	mcpCallCount *int,
	mcpTools *[]string,
	usage *any,
	accumulatedText *string,
) error {
	// Event logging removed to reduce verbosity

	wrappedData := map[string]any{
		"openai_event_type": event.Type,
		"event_count":       eventCount,
		"elapsed_ms":        time.Since(startTime).Milliseconds(),
	}

	switch event.Type {
	case "response.output_item.added":
		log.Printf("📝 output_item.added - starting output generation")
		return callback(StreamEvent{Type: "output_started", Message: "Generating output...", Data: wrappedData})

	case "response.output_text.delta":
		// Accumulate text deltas
		if deltaBytes, err := json.Marshal(event.Delta); err == nil {
			var deltaMap map[string]string
			if json.Unmarshal(deltaBytes, &deltaMap) == nil {
				if text, ok := deltaMap["OfString"]; ok {
					*accumulatedText += text
					// Send text delta in callback for incremental parsing
					return callback(StreamEvent{
						Type:    "output_text.delta",
						Message: "Text delta received",
						Data: map[string]interface{}{
							"text": text,
						},
					})
				}
			}
		}
		return nil

	case "response.output_item.done":
		return p.handleOutputItemDone(accumulatedText, finalResponse, callback, wrappedData)

	case "response.completed":
		log.Printf("✅ response.completed event")
		return p.handleResponseCompleted(event, wrappedData, callback, finalResponse, mcpUsed, mcpCallCount, mcpTools, usage)

	default:
		// Unknown event types - logging removed to reduce verbosity
	}

	return nil
}

// handleOutputItemDone processes the output_item.done event to reduce complexity
func (p *OpenAIProvider) handleOutputItemDone(
	accumulatedText *string,
	finalResponse **models.MusicalOutput,
	callback StreamCallback,
	wrappedData map[string]any,
) error {
	log.Printf("📦 output_item.done - accumulated text: %d chars", len(*accumulatedText))
	// Parse accumulated text when output item is complete and we have text content
	// Note: Some output items (like tool calls) have 0 chars, so we skip parsing those
	if *accumulatedText != "" && *finalResponse == nil {
		return p.parseAccumulatedText(accumulatedText, finalResponse, callback)
	}
	if *accumulatedText == "" && *finalResponse == nil {
		// This might be a tool call result - we'll wait for the actual text output or response.completed
		log.Printf("ℹ️  output_item.done with no text (likely tool call result), waiting for text output...")
	}
	return callback(StreamEvent{Type: "output_progress", Message: "Output item completed", Data: wrappedData})
}

// parseAccumulatedText parses the accumulated text into MusicalOutput
func (p *OpenAIProvider) parseAccumulatedText(
	accumulatedText *string,
	finalResponse **models.MusicalOutput,
	callback StreamCallback,
) error {
	var output models.MusicalOutput
	if parseErr := json.Unmarshal([]byte(*accumulatedText), &output); parseErr != nil {
		log.Printf("❌ Parse error in parseAccumulatedText: %v", parseErr)
		log.Printf("❌ Accumulated text (first %d chars): %s", maxErrorPreviewChars, truncate(*accumulatedText, maxErrorPreviewChars))
		sentry.CaptureException(parseErr)
		// Send error event but don't stop processing - let handleResponseCompleted try OutputText()
		_ = callback(StreamEvent{Type: "error", Message: fmt.Sprintf("Parse error: %v", parseErr)})
		// Don't return error - let the stream continue so handleResponseCompleted can try OutputText()
		return nil
	}
	*finalResponse = &output
	log.Printf("✅ Successfully parsed output: %d choices", len(output.Choices))
	// Only reset accumulated text after successful parsing
	*accumulatedText = ""
	return nil
}

// handleResponseCompleted handles the final response.completed event
func (p *OpenAIProvider) handleResponseCompleted(
	event responses.ResponseStreamEventUnion,
	wrappedData map[string]any,
	callback StreamCallback,
	finalResponse **models.MusicalOutput,
	mcpUsed *bool,
	mcpCallCount *int,
	mcpTools *[]string,
	usage *any,
) error {
	_ = callback(StreamEvent{Type: "analyzing", Message: "Analyzing response...", Data: wrappedData})

	resp := event.Response
	*mcpUsed, *mcpCallCount, *mcpTools = p.analyzeMCPUsage(&resp)
	*usage = resp.Usage

	log.Printf("🚨 OpenAI response Model: '%s'", resp.Model)
	log.Printf("📊 Token Usage: Total=%d, Input=%d, Output=%d",
		resp.Usage.TotalTokens, resp.Usage.InputTokens, resp.Usage.OutputTokens)

	// If we haven't parsed the output yet, try to get it from OutputText()
	// This handles cases where text output comes in a later output_item after tool calls
	if *finalResponse == nil {
		if err := p.tryParseOutputText(&resp, finalResponse, callback); err != nil {
			return err
		}
	} else {
		log.Printf("✅ finalResponse already set with %d choices", len((*finalResponse).Choices))
	}

	// Send MCP usage event if applicable
	if *mcpUsed {
		_ = callback(StreamEvent{
			Type:    "mcp_used",
			Message: fmt.Sprintf("MCP tools used: %v", *mcpTools),
			Data: map[string]any{
				"calls": *mcpCallCount,
				"tools": *mcpTools,
			},
		})
	}

	return nil
}

// buildParseErrorMessage builds a user-friendly error message for parse errors
func (p *OpenAIProvider) buildParseErrorMessage(outputText string, parseErr error) string {
	errorMsg := fmt.Sprintf("Parse error: %v", parseErr)
	if len(outputText) > 0 {
		trimmed := strings.TrimSpace(outputText)
		if strings.HasPrefix(trimmed, "<") || strings.HasPrefix(trimmed, "<!") {
			errorMsg = "Received HTML response instead of JSON. This may indicate a server error."
		} else if strings.HasPrefix(trimmed, "/") || !strings.HasPrefix(trimmed, "{") {
			errorMsg = fmt.Sprintf("Received invalid JSON response: %s", truncate(trimmed, maxErrorResponseChars))
		}
	}
	return errorMsg
}

// tryParseOutputText attempts to parse output from OutputText() method
func (p *OpenAIProvider) tryParseOutputText(
	resp *responses.Response,
	finalResponse **models.MusicalOutput,
	callback StreamCallback,
) error {
	log.Printf("⚠️  finalResponse is nil in handleResponseCompleted, trying OutputText()")
	outputText := resp.OutputText()
	log.Printf("🔍 OutputText() returned: length=%d", len(outputText))

	// Check if OutputText() returned something that's clearly not JSON (like a path or URL)
	outputText = p.validateOutputText(outputText)

	if outputText != "" {
		return p.parseOutputText(outputText, finalResponse, callback)
	}

	// OutputText() is empty or invalid - check output items
	log.Printf("⚠️  OutputText() is empty or invalid - checking if response has output items")
	log.Printf("🔍 Response has %d output items", len(resp.Output))
	for i, item := range resp.Output {
		log.Printf("   Output item #%d: Type=%v", i, item.Type)
	}

	// If we still don't have a response, send an error
	if *finalResponse == nil {
		errorMsg := "No valid output received from model. The response may have been empty or invalid."
		_ = callback(StreamEvent{Type: "error", Message: errorMsg})
		return fmt.Errorf("no output received from model")
	}

	return nil
}

// validateOutputText checks if outputText is valid JSON and returns empty string if not
func (p *OpenAIProvider) validateOutputText(outputText string) string {
	if outputText == "" {
		return ""
	}

	trimmed := strings.TrimSpace(outputText)
	// If it starts with '/' or doesn't start with '{' or '[', it's likely not JSON
	if strings.HasPrefix(trimmed, "/") || (!strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[")) {
		previewLen := len(trimmed)
		if previewLen > maxPathPreviewLen {
			previewLen = maxPathPreviewLen
		}
		log.Printf("⚠️  OutputText() returned non-JSON content (starts with '%s'), ignoring it", trimmed[:previewLen])
		return "" // Treat as empty so we can check output items
	}

	return outputText
}

// parseOutputText parses the outputText JSON into MusicalOutput
func (p *OpenAIProvider) parseOutputText(
	outputText string,
	finalResponse **models.MusicalOutput,
	callback StreamCallback,
) error {
	log.Printf("📝 Attempting to parse OutputText (first %d chars): %s", maxPreviewChars, truncate(outputText, maxPreviewChars))
	var output models.MusicalOutput
	if parseErr := json.Unmarshal([]byte(outputText), &output); parseErr != nil {
		log.Printf("❌ Failed to parse OutputText: %v", parseErr)
		preview := truncate(outputText, maxErrorPreviewChars)
		log.Printf("Raw OutputText (first %d chars): %s", maxErrorPreviewChars, preview)

		errorMsg := p.buildParseErrorMessage(outputText, parseErr)
		_ = callback(StreamEvent{Type: "error", Message: errorMsg})
		return fmt.Errorf("failed to parse output: %w", parseErr)
	}

	*finalResponse = &output
	log.Printf("✅ Parsed output from OutputText: %d choices", len(output.Choices))
	return nil
}
