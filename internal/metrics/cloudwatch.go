package metrics

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const (
	namespace                = "AIDEAS/API"
	httpStatusServerError    = 500
	cloudwatchTimeoutSeconds = 5
)

// Client wraps CloudWatch client for custom metrics
type Client struct {
	client      *cloudwatch.Client
	enabled     bool
	environment string
}

// NewClient creates a new CloudWatch metrics client
func NewClient(ctx context.Context, environment string) (*Client, error) {
	// Only enable in production
	if environment != "production" {
		log.Printf("📊 CloudWatch Metrics: DISABLED (environment: %s)", environment)
		return &Client{
			enabled:     false,
			environment: environment,
		}, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to load AWS config for CloudWatch: %v", err)
		return &Client{enabled: false}, nil
	}

	client := cloudwatch.NewFromConfig(cfg)
	log.Printf("📊 CloudWatch Metrics: ✅ ENABLED (namespace: %s)", namespace)

	return &Client{
		client:      client,
		enabled:     true,
		environment: environment,
	}, nil
}

// RecordAPIRequest records an API request metric
func (m *Client) RecordAPIRequest(endpoint string, statusCode int, duration time.Duration) {
	if !m.enabled {
		return
	}

	go func() {
		ctx := context.Background()
		// Determine if success or error
		metricName := "APIRequests"
		if statusCode >= httpStatusServerError {
			metricName = "APIErrors"
		}

		dimensions := []types.Dimension{
			{
				Name:  aws.String("Endpoint"),
				Value: aws.String(endpoint),
			},
			{
				Name:  aws.String("Environment"),
				Value: aws.String(m.environment),
			},
		}

		// Record count
		if err := m.putMetric(ctx, metricName, 1, types.StandardUnitCount, dimensions); err != nil {
			log.Printf("Failed to record %s metric: %v", metricName, err)
		}

		// Record duration
		latencyMs := float64(duration.Milliseconds())
		if err := m.putMetric(ctx, "APILatency", latencyMs, types.StandardUnitMilliseconds, dimensions); err != nil {
			log.Printf("Failed to record APILatency metric: %v", err)
		}
	}()
}

// RecordTokenUsage records OpenAI token usage
func (m *Client) RecordTokenUsage(model string, totalTokens, inputTokens, outputTokens, reasoningTokens int) {
	if !m.enabled {
		return
	}

	go func() {
		ctx := context.Background()
		dimensions := []types.Dimension{
			{
				Name:  aws.String("Model"),
				Value: aws.String(model),
			},
			{
				Name:  aws.String("Environment"),
				Value: aws.String(m.environment),
			},
		}

		// Record total tokens
		totalFloat := float64(totalTokens)
		if err := m.putMetric(ctx, "OpenAITokens/Total", totalFloat, types.StandardUnitCount, dimensions); err != nil {
			log.Printf("Failed to record OpenAITokens/Total metric: %v", err)
		}

		// Record input tokens
		inputFloat := float64(inputTokens)
		if err := m.putMetric(ctx, "OpenAITokens/Input", inputFloat, types.StandardUnitCount, dimensions); err != nil {
			log.Printf("Failed to record OpenAITokens/Input metric: %v", err)
		}

		// Record output tokens
		outputFloat := float64(outputTokens)
		if err := m.putMetric(ctx, "OpenAITokens/Output", outputFloat, types.StandardUnitCount, dimensions); err != nil {
			log.Printf("Failed to record OpenAITokens/Output metric: %v", err)
		}

		// Record reasoning tokens (for GPT-5/o1 models)
		if reasoningTokens > 0 {
			reasoningFloat := float64(reasoningTokens)
			if err := m.putMetric(ctx, "OpenAITokens/Reasoning", reasoningFloat, types.StandardUnitCount, dimensions); err != nil {
				log.Printf("Failed to record OpenAITokens/Reasoning metric: %v", err)
			}
		}
	}()
}

// RecordMCPUsage records MCP server usage
func (m *Client) RecordMCPUsage(used bool, callCount int) {
	if !m.enabled {
		return
	}

	go func() {
		ctx := context.Background()
		dimensions := []types.Dimension{
			{
				Name:  aws.String("Environment"),
				Value: aws.String(m.environment),
			},
		}

		// Record if MCP was used
		usedValue := 0.0
		if used {
			usedValue = 1.0
		}
		if err := m.putMetric(ctx, "MCPUsage", usedValue, types.StandardUnitCount, dimensions); err != nil {
			log.Printf("Failed to record MCPUsage metric: %v", err)
		}

		// Record number of MCP calls
		if callCount > 0 {
			callsFloat := float64(callCount)
			if err := m.putMetric(ctx, "MCPCalls", callsFloat, types.StandardUnitCount, dimensions); err != nil {
				log.Printf("Failed to record MCPCalls metric: %v", err)
			}
		}
	}()
}

// RecordGenerationDuration records generation request duration
func (m *Client) RecordGenerationDuration(duration time.Duration, success bool) {
	if !m.enabled {
		return
	}

	go func() {
		ctx := context.Background()
		dimensions := []types.Dimension{
			{
				Name:  aws.String("Success"),
				Value: aws.String(boolToString(success)),
			},
			{
				Name:  aws.String("Environment"),
				Value: aws.String(m.environment),
			},
		}

		durationMs := float64(duration.Milliseconds())
		if err := m.putMetric(ctx, "GenerationDuration", durationMs, types.StandardUnitMilliseconds, dimensions); err != nil {
			log.Printf("Failed to record GenerationDuration metric: %v", err)
		}
	}()
}

// putMetric sends a metric to CloudWatch
func (m *Client) putMetric(
	_ context.Context,
	metricName string,
	value float64,
	unit types.StandardUnit,
	dimensions []types.Dimension,
) error {
	if !m.enabled || m.client == nil {
		return nil
	}

	// Create context with timeout for CloudWatch call
	timeout := time.Duration(cloudwatchTimeoutSeconds) * time.Second
	cwCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := m.client.PutMetricData(cwCtx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String(namespace),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String(metricName),
				Value:      aws.Float64(value),
				Unit:       unit,
				Timestamp:  aws.Time(time.Now()),
				Dimensions: dimensions,
			},
		},
	})

	return err
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
